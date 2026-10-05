package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/ent/configauditlog"
	"github.com/coder-lulu/newbee-io-rpc/ent/configitem"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

// ConfigCenter 配置中心服务
//
// 职责:
// 1. 统一配置管理 (CRUD操作)
// 2. 三级缓存: 内存(1µs) → Redis(1ms) → MySQL(10ms)
// 3. 配置热重载 (Redis Pub/Sub)
// 4. 多租户隔离
// 5. 配置版本管理
type ConfigCenter struct {
	db          *ent.Client
	redis       redis.UniversalClient
	pubsub      *redis.PubSub
	memoryCache sync.Map // map[string]*ConfigValue - L1缓存
	subscribers sync.Map // map[string][]ConfigWatcher - 配置变更监听器
	logger      logx.Logger
}

// ConfigValue 配置值缓存结构
type ConfigValue struct {
	Value      string    // 配置值
	ValueType  string    // 值类型
	Version    int       // 版本号
	UpdatedAt  time.Time // 更新时间
	ExpireTime time.Time // 缓存过期时间
}

// ConfigWatcher 配置变更监听器
type ConfigWatcher func(key string, oldValue, newValue string)

// ConfigOptions 配置选项
type ConfigOptions struct {
	TenantID    uint64 // 租户ID
	ServiceName string // 服务名称
	Category    string // 配置分类
	CacheTTL    time.Duration
}

const (
	// Redis key前缀
	redisConfigPrefix  = "unified-io:config"
	redisPubSubChannel = "unified-io:config:update"

	// 默认缓存TTL
	defaultMemoryCacheTTL = 5 * time.Minute
	defaultRedisCacheTTL  = 30 * time.Minute
)

// NewConfigCenter 创建配置中心服务
func NewConfigCenter(db *ent.Client, rds redis.UniversalClient) *ConfigCenter {
	cc := &ConfigCenter{
		db:     db,
		redis:  rds,
		logger: logx.WithContext(context.Background()),
	}

	// 启动Redis订阅监听
	go cc.startConfigWatcher()

	return cc
}

// Get 获取配置值（三级缓存）
//
// 缓存策略:
// 1. 查询内存缓存 (1µs)
// 2. 未命中 → 查询Redis (1ms)
// 3. 未命中 → 查询MySQL (10ms)
// 4. 回写缓存
func (cc *ConfigCenter) Get(ctx context.Context, key string, opts *ConfigOptions) (string, error) {
	if opts == nil {
		opts = &ConfigOptions{}
	}

	// L1: 内存缓存
	if cached, ok := cc.getFromMemory(key, opts.TenantID); ok {
		cc.logger.Debugw("Config cache hit (memory)", logx.Field("key", key))
		return cached.Value, nil
	}

	// L2: Redis缓存
	if cached, err := cc.getFromRedis(ctx, key, opts.TenantID); err == nil {
		cc.logger.Debugw("Config cache hit (redis)", logx.Field("key", key))
		// 回写L1
		cc.setToMemory(key, opts.TenantID, cached)
		return cached.Value, nil
	}

	// L3: MySQL查询
	config, err := cc.db.ConfigItem.Query().
		Where(
			configitem.TenantIDEQ(opts.TenantID),
			configitem.ConfigKeyEQ(key),
			configitem.StatusEQ(1), // 只查询启用的配置
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", fmt.Errorf("config not found: %s", key)
		}
		return "", fmt.Errorf("failed to query config: %w", err)
	}

	// 回写缓存
	cached := &ConfigValue{
		Value:      config.ConfigValue,
		ValueType:  config.ValueType,
		Version:    config.Version,
		UpdatedAt:  config.UpdatedAt,
		ExpireTime: time.Now().Add(defaultMemoryCacheTTL),
	}
	cc.setToMemory(key, opts.TenantID, cached)
	cc.setToRedis(ctx, key, opts.TenantID, cached)

	cc.logger.Infow("Config loaded from database",
		logx.Field("key", key),
		logx.Field("tenant_id", opts.TenantID))

	return config.ConfigValue, nil
}

// GetInt 获取整数配置
func (cc *ConfigCenter) GetInt(ctx context.Context, key string, opts *ConfigOptions) (int64, error) {
	val, err := cc.Get(ctx, key, opts)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

// GetBool 获取布尔配置
func (cc *ConfigCenter) GetBool(ctx context.Context, key string, opts *ConfigOptions) (bool, error) {
	val, err := cc.Get(ctx, key, opts)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(val)
}

// GetFloat 获取浮点配置
func (cc *ConfigCenter) GetFloat(ctx context.Context, key string, opts *ConfigOptions) (float64, error) {
	val, err := cc.Get(ctx, key, opts)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(val, 64)
}

// GetJSON 获取JSON配置并反序列化
func (cc *ConfigCenter) GetJSON(ctx context.Context, key string, opts *ConfigOptions, target interface{}) error {
	val, err := cc.Get(ctx, key, opts)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), target)
}

// Set 设置配置值
//
// 操作:
// 1. 更新MySQL（版本号+1）
// 2. 清除缓存（L1 + L2）
// 3. 发布变更通知（Redis Pub/Sub）
// 4. 记录审计日志（异步）
func (cc *ConfigCenter) Set(ctx context.Context, key, value string, opts *ConfigOptions) error {
	if opts == nil {
		opts = &ConfigOptions{}
	}

	// 提取审计上下文
	auditCtx := cc.extractAuditContext(ctx)

	// 查询现有配置
	existing, err := cc.db.ConfigItem.Query().
		Where(
			configitem.TenantIDEQ(opts.TenantID),
			configitem.ConfigKeyEQ(key),
		).
		First(ctx)

	var newVersion int
	var oldValue string
	var changeType string
	var oldVersion int

	if err != nil {
		if !ent.IsNotFound(err) {
			return fmt.Errorf("failed to query config: %w", err)
		}

		// 配置不存在，创建新配置
		_, err = cc.db.ConfigItem.Create().
			SetTenantID(opts.TenantID).
			SetConfigKey(key).
			SetConfigValue(value).
			SetValueType(opts.inferValueType(value)).
			SetCategory(opts.Category).
			SetServiceName(opts.ServiceName).
			SetVersion(1).
			SetStatus(1).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to create config: %w", err)
		}
		newVersion = 1
		changeType = "create"
		oldVersion = 0
		oldValue = ""
	} else {
		// 配置存在，更新
		oldValue = existing.ConfigValue
		oldVersion = existing.Version

		_, err = existing.Update().
			SetConfigValue(value).
			SetVersion(existing.Version + 1).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to update config: %w", err)
		}
		newVersion = existing.Version + 1
		changeType = "update"
	}

	// 清除缓存
	cc.invalidateCache(ctx, key, opts.TenantID)

	// 发布变更通知
	cc.publishUpdate(ctx, key, opts.TenantID, newVersion)

	// 📝 记录审计日志（异步，不阻塞）
	cc.recordAuditLog(ctx, key, oldValue, value, changeType, oldVersion, newVersion, opts, auditCtx)

	cc.logger.Infow("Config updated",
		logx.Field("key", key),
		logx.Field("tenant_id", opts.TenantID),
		logx.Field("version", newVersion),
		logx.Field("change_type", changeType))

	return nil
}

// Delete 删除配置（软删除，设置status=2）
func (cc *ConfigCenter) Delete(ctx context.Context, key string, tenantID uint64) error {
	// 提取审计上下文
	auditCtx := cc.extractAuditContext(ctx)

	// 查询现有配置（用于审计日志）
	existing, err := cc.db.ConfigItem.Query().
		Where(
			configitem.TenantIDEQ(tenantID),
			configitem.ConfigKeyEQ(key),
		).
		First(ctx)

	var oldValue string
	var oldVersion int
	var opts *ConfigOptions

	if err == nil {
		oldValue = existing.ConfigValue
		oldVersion = existing.Version
		opts = &ConfigOptions{
			TenantID:    tenantID,
			ServiceName: existing.ServiceName,
			Category:    existing.Category,
		}
	} else {
		// 配置不存在，仍然继续删除操作（可能是重复删除）
		opts = &ConfigOptions{TenantID: tenantID}
	}

	// 执行软删除
	_, err = cc.db.ConfigItem.Update().
		Where(
			configitem.TenantIDEQ(tenantID),
			configitem.ConfigKeyEQ(key),
		).
		SetStatus(2). // 禁用
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}

	// 清除缓存
	cc.invalidateCache(ctx, key, tenantID)

	// 📝 记录审计日志（异步，不阻塞）
	cc.recordAuditLog(ctx, key, oldValue, "", "delete", oldVersion, 0, opts, auditCtx)

	// 发布删除通知
	cc.publishUpdate(ctx, key, tenantID, 0)

	return nil
}

// Watch 监听配置变更
func (cc *ConfigCenter) Watch(key string, watcher ConfigWatcher) {
	if val, ok := cc.subscribers.Load(key); ok {
		watchers := val.([]ConfigWatcher)
		watchers = append(watchers, watcher)
		cc.subscribers.Store(key, watchers)
	} else {
		cc.subscribers.Store(key, []ConfigWatcher{watcher})
	}
}

// ============== 私有方法 ==============

// getFromMemory 从内存缓存获取
func (cc *ConfigCenter) getFromMemory(key string, tenantID uint64) (*ConfigValue, bool) {
	cacheKey := cc.buildCacheKey(key, tenantID)
	if val, ok := cc.memoryCache.Load(cacheKey); ok {
		cached := val.(*ConfigValue)
		// 检查是否过期
		if time.Now().Before(cached.ExpireTime) {
			return cached, true
		}
		// 过期删除
		cc.memoryCache.Delete(cacheKey)
	}
	return nil, false
}

// setToMemory 写入内存缓存
func (cc *ConfigCenter) setToMemory(key string, tenantID uint64, value *ConfigValue) {
	cacheKey := cc.buildCacheKey(key, tenantID)
	cc.memoryCache.Store(cacheKey, value)
}

// getFromRedis 从Redis获取
func (cc *ConfigCenter) getFromRedis(ctx context.Context, key string, tenantID uint64) (*ConfigValue, error) {
	redisKey := cc.buildRedisKey(key, tenantID)
	data, err := cc.redis.Get(ctx, redisKey).Result()
	if err != nil {
		return nil, err
	}

	var cached ConfigValue
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		return nil, err
	}

	return &cached, nil
}

// setToRedis 写入Redis
func (cc *ConfigCenter) setToRedis(ctx context.Context, key string, tenantID uint64, value *ConfigValue) {
	redisKey := cc.buildRedisKey(key, tenantID)
	data, _ := json.Marshal(value)
	cc.redis.Set(ctx, redisKey, data, defaultRedisCacheTTL)
}

// invalidateCache 清除缓存
func (cc *ConfigCenter) invalidateCache(ctx context.Context, key string, tenantID uint64) {
	// 清除L1
	cacheKey := cc.buildCacheKey(key, tenantID)
	cc.memoryCache.Delete(cacheKey)

	// 清除L2
	redisKey := cc.buildRedisKey(key, tenantID)
	cc.redis.Del(ctx, redisKey)
}

// publishUpdate 发布配置变更通知
func (cc *ConfigCenter) publishUpdate(ctx context.Context, key string, tenantID uint64, version int) {
	msg := map[string]interface{}{
		"key":       key,
		"tenant_id": tenantID,
		"version":   version,
		"timestamp": time.Now().Unix(),
	}
	data, _ := json.Marshal(msg)
	cc.redis.Publish(ctx, redisPubSubChannel, data)
}

// startConfigWatcher 启动配置变更监听
func (cc *ConfigCenter) startConfigWatcher() {
	cc.pubsub = cc.redis.Subscribe(context.Background(), redisPubSubChannel)
	defer cc.pubsub.Close()

	ch := cc.pubsub.Channel()
	for msg := range ch {
		var update map[string]interface{}
		if err := json.Unmarshal([]byte(msg.Payload), &update); err != nil {
			cc.logger.Errorw("Failed to parse config update message",
				logx.Field("error", err))
			continue
		}

		key := update["key"].(string)
		tenantID := uint64(update["tenant_id"].(float64))

		// 清除本地缓存
		cacheKey := cc.buildCacheKey(key, tenantID)
		cc.memoryCache.Delete(cacheKey)

		// 触发监听器
		if val, ok := cc.subscribers.Load(key); ok {
			watchers := val.([]ConfigWatcher)
			for _, watcher := range watchers {
				go watcher(key, "", "") // 异步通知
			}
		}

		cc.logger.Infow("Config update received",
			logx.Field("key", key),
			logx.Field("tenant_id", tenantID))
	}
}

// buildCacheKey 构建内存缓存key
func (cc *ConfigCenter) buildCacheKey(key string, tenantID uint64) string {
	return fmt.Sprintf("%d:%s", tenantID, key)
}

// buildRedisKey 构建Redis key
func (cc *ConfigCenter) buildRedisKey(key string, tenantID uint64) string {
	return fmt.Sprintf("%s:tenant:%d:%s", redisConfigPrefix, tenantID, key)
}

// inferValueType 推断配置值类型
func (opts *ConfigOptions) inferValueType(value string) string {
	// 尝试解析为int
	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return "int"
	}
	// 尝试解析为float
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return "float"
	}
	// 尝试解析为bool
	if _, err := strconv.ParseBool(value); err == nil {
		return "bool"
	}
	// 尝试解析为JSON
	var js interface{}
	if err := json.Unmarshal([]byte(value), &js); err == nil {
		return "json"
	}
	return "string"
}

// GetWithDefault 获取配置，如果不存在则返回默认值
func (cc *ConfigCenter) GetWithDefault(ctx context.Context, key string, defaultValue string, opts *ConfigOptions) string {
	val, err := cc.Get(ctx, key, opts)
	if err != nil {
		return defaultValue
	}
	return val
}

// ================================================================================
// 审计日志功能
// ================================================================================

// AuditContext 审计上下文（从context或opts提取）
type AuditContext struct {
	UserID      uint64 // 变更人用户ID
	Username    string // 变更人用户名
	IPAddress   string // 操作IP地址
	UserAgent   string // 用户代理
	Reason      string // 变更原因
	IsRollback  bool   // 是否为回滚操作
	RollbackRef string // 回滚来源日志ID
}

// recordAuditLog 记录配置变更审计日志
//
// 参数:
//   - ctx: 上下文
//   - key: 配置键
//   - oldValue: 旧值（create时为空）
//   - newValue: 新值（delete时为空）
//   - changeType: 变更类型（create/update/delete）
//   - opts: 配置选项
//   - auditCtx: 审计上下文
//
// 特性:
//   - 异步记录，不阻塞主流程
//   - 记录失败只打印日志，不影响配置变更
func (cc *ConfigCenter) recordAuditLog(
	ctx context.Context,
	key string,
	oldValue string,
	newValue string,
	changeType string,
	oldVersion int,
	newVersion int,
	opts *ConfigOptions,
	auditCtx *AuditContext,
) {
	// 异步记录审计日志
	go func() {
		// 创建审计日志记录
		builder := cc.db.ConfigAuditLog.Create().
			SetTenantID(opts.TenantID).
			SetConfigKey(key).
			SetChangeType(changeType).
			SetOldVersion(oldVersion).
			SetNewVersion(newVersion)

		// 设置值
		if oldValue != "" {
			builder.SetOldValue(oldValue)
		}
		if newValue != "" {
			builder.SetNewValue(newValue)
		}

		// 设置服务信息
		if opts.ServiceName != "" {
			builder.SetServiceName(opts.ServiceName)
		}
		if opts.Category != "" {
			builder.SetCategory(opts.Category)
		}

		// 设置审计上下文
		if auditCtx != nil {
			if auditCtx.UserID > 0 {
				builder.SetChangedBy(auditCtx.UserID)
			}
			if auditCtx.Username != "" {
				builder.SetChangedByName(auditCtx.Username)
			}
			if auditCtx.IPAddress != "" {
				builder.SetIPAddress(auditCtx.IPAddress)
			}
			if auditCtx.UserAgent != "" {
				builder.SetUserAgent(auditCtx.UserAgent)
			}
			if auditCtx.Reason != "" {
				builder.SetChangeReason(auditCtx.Reason)
			}
			if auditCtx.IsRollback {
				builder.SetIsRollback(true)
				if auditCtx.RollbackRef != "" {
					builder.SetRollbackFromLogID(auditCtx.RollbackRef)
				}
			}
		}

		// 保存审计日志
		auditContext, cancel := newAuditContext(ctx)
		defer cancel()
		if _, err := builder.Save(auditContext); err != nil {
			cc.logger.Errorw("Failed to record config audit log",
				logx.Field("key", key),
				logx.Field("change_type", changeType),
				logx.Field("error", err))
		} else {
			cc.logger.Infow("📝 Config audit log recorded",
				logx.Field("key", key),
				logx.Field("change_type", changeType),
				logx.Field("old_version", oldVersion),
				logx.Field("new_version", newVersion))
		}
	}()
}

// extractAuditContext 从context中提取审计信息
func (cc *ConfigCenter) extractAuditContext(ctx context.Context) *AuditContext {
	auditCtx := &AuditContext{}

	// 尝试从context提取用户信息（如果有的话）
	if userID, ok := ctx.Value("userId").(uint64); ok {
		auditCtx.UserID = userID
	}
	if username, ok := ctx.Value("username").(string); ok {
		auditCtx.Username = username
	}
	if ip, ok := ctx.Value("clientIp").(string); ok {
		auditCtx.IPAddress = ip
	}
	if ua, ok := ctx.Value("userAgent").(string); ok {
		auditCtx.UserAgent = ua
	}

	return auditCtx
}

// QueryAuditLogs 查询配置变更审计日志
//
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID
//   - configKey: 配置键（可选，为空则查询所有）
//   - limit: 返回条数（默认100，最大1000）
//
// 返回:
//   - 审计日志列表（按时间倒序）
func (cc *ConfigCenter) QueryAuditLogs(ctx context.Context, tenantID uint64, configKey string, limit int) ([]*ent.ConfigAuditLog, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	query := cc.db.ConfigAuditLog.Query().
		Where(configauditlog.TenantIDEQ(tenantID)).
		Order(ent.Desc(configauditlog.FieldCreatedAt)).
		Limit(limit)

	// 如果指定了配置键，过滤
	if configKey != "" {
		query = query.Where(configauditlog.ConfigKeyEQ(configKey))
	}

	logs, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}

	return logs, nil
}

// GetConfigHistory 获取指定配置的变更历史
//
// 返回该配置的所有变更记录，按时间正序排列
func (cc *ConfigCenter) GetConfigHistory(ctx context.Context, tenantID uint64, configKey string) ([]*ent.ConfigAuditLog, error) {
	logs, err := cc.db.ConfigAuditLog.Query().
		Where(
			configauditlog.TenantIDEQ(tenantID),
			configauditlog.ConfigKeyEQ(configKey),
		).
		Order(ent.Asc(configauditlog.FieldCreatedAt)).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get config history: %w", err)
	}

	return logs, nil
}

// newAuditContext lets the audit write finish after the caller disconnects while retaining tenant identity.
func newAuditContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
}
