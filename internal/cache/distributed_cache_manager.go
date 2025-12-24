package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// DistributedCacheManager 分布式缓存管理器
// 基于Redis实现的分布式缓存
type DistributedCacheManager struct {
	redis  *redis.Redis
	config *DistributedCacheConfig
}

// DistributedCacheConfig 分布式缓存配置
type DistributedCacheConfig struct {
	DefaultTTL          time.Duration `json:"defaultTTL"`          // 默认TTL
	KeyPrefix           string        `json:"keyPrefix"`           // 键前缀
	CompressEnabled     bool          `json:"compressEnabled"`     // 是否启用压缩
	SerializationFormat string        `json:"serializationFormat"` // 序列化格式
	BatchSize           int           `json:"batchSize"`           // 批量操作大小
	RetryCount          int           `json:"retryCount"`          // 重试次数
	RetryDelay          time.Duration `json:"retryDelay"`          // 重试延迟
}

// CacheValue 缓存值包装
type CacheValue struct {
	Data      interface{} `json:"data"`
	CreatedAt time.Time   `json:"createdAt"`
	TTL       int64       `json:"ttl"` // 秒
	Version   string      `json:"version"`
}

// NewDistributedCacheManager 创建分布式缓存管理器
func NewDistributedCacheManager(redis *redis.Redis, config *DistributedCacheConfig) *DistributedCacheManager {
	if config == nil {
		config = getDefaultDistributedCacheConfig()
	}

	return &DistributedCacheManager{
		redis:  redis,
		config: config,
	}
}

// Get 获取缓存项
func (m *DistributedCacheManager) Get(ctx context.Context, key string) (interface{}, error) {
	cacheKey := m.buildKey(key)

	// 从Redis获取数据
	data, err := m.redis.GetCtx(ctx, cacheKey)
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("缓存项不存在: %s", key)
		}
		return nil, fmt.Errorf("获取缓存失败: %w", err)
	}

	// 反序列化数据
	var cacheValue CacheValue
	if err := json.Unmarshal([]byte(data), &cacheValue); err != nil {
		logx.Errorf("反序列化缓存数据失败 [%s]: %v", cacheKey, err)
		return nil, fmt.Errorf("反序列化缓存数据失败: %w", err)
	}

	return cacheValue.Data, nil
}

// Set 设置缓存项
func (m *DistributedCacheManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if ttl == 0 {
		ttl = m.config.DefaultTTL
	}

	cacheKey := m.buildKey(key)

	// 构建缓存值
	cacheValue := CacheValue{
		Data:      value,
		CreatedAt: time.Now(),
		TTL:       int64(ttl.Seconds()),
		Version:   "1.0",
	}

	// 序列化数据
	data, err := json.Marshal(cacheValue)
	if err != nil {
		logx.Errorf("序列化缓存数据失败 [%s]: %v", cacheKey, err)
		return fmt.Errorf("序列化缓存数据失败: %w", err)
	}

	// 设置到Redis
	err = m.redis.SetexCtx(ctx, cacheKey, string(data), int(ttl.Seconds()))
	if err != nil {
		return fmt.Errorf("设置缓存失败: %w", err)
	}

	return nil
}

// Delete 删除缓存项
func (m *DistributedCacheManager) Delete(ctx context.Context, key string) error {
	cacheKey := m.buildKey(key)

	_, err := m.redis.DelCtx(ctx, cacheKey)
	if err != nil {
		return fmt.Errorf("删除缓存失败: %w", err)
	}

	return nil
}

// InvalidatePattern 删除匹配模式的缓存项
func (m *DistributedCacheManager) InvalidatePattern(ctx context.Context, pattern string) error {
	cachePattern := m.buildKey(pattern)

	// 使用SCAN命令查找匹配的键
	keys, err := m.scanKeys(ctx, cachePattern)
	if err != nil {
		return fmt.Errorf("扫描缓存键失败: %w", err)
	}

	if len(keys) == 0 {
		return nil // 没有匹配的键
	}

	// 批量删除
	if err := m.batchDelete(ctx, keys); err != nil {
		return fmt.Errorf("批量删除缓存失败: %w", err)
	}

	return nil
}

// Exists 检查缓存项是否存在
func (m *DistributedCacheManager) Exists(ctx context.Context, key string) (bool, error) {
	cacheKey := m.buildKey(key)

	exists, err := m.redis.ExistsCtx(ctx, cacheKey)
	if err != nil {
		return false, fmt.Errorf("检查缓存存在性失败: %w", err)
	}

	return exists, nil
}

// Expire 设置缓存项过期时间
func (m *DistributedCacheManager) Expire(ctx context.Context, key string, ttl time.Duration) error {
	cacheKey := m.buildKey(key)

	err := m.redis.ExpireCtx(ctx, cacheKey, int(ttl.Seconds()))
	if err != nil {
		return fmt.Errorf("设置缓存过期时间失败: %w", err)
	}

	return nil
}

// TTL 获取缓存项剩余过期时间
func (m *DistributedCacheManager) TTL(ctx context.Context, key string) (time.Duration, error) {
	cacheKey := m.buildKey(key)

	ttl, err := m.redis.TtlCtx(ctx, cacheKey)
	if err != nil {
		return 0, fmt.Errorf("获取缓存TTL失败: %w", err)
	}

	if ttl == -2 {
		return 0, fmt.Errorf("缓存项不存在: %s", key)
	}

	if ttl == -1 {
		return 0, nil // 永不过期
	}

	return time.Duration(ttl) * time.Second, nil
}

// SetMultiple 批量设置缓存项
func (m *DistributedCacheManager) SetMultiple(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	err := m.redis.Pipelined(func(pipe redis.Pipeliner) error {
		for key, value := range items {
			cacheKey := m.buildKey(key)

			// 构建缓存值
			cacheValue := CacheValue{
				Data:      value,
				CreatedAt: time.Now(),
				TTL:       int64(ttl.Seconds()),
				Version:   "1.0",
			}

			// 序列化数据
			data, err := json.Marshal(cacheValue)
			if err != nil {
				logx.Errorf("序列化缓存数据失败 [%s]: %v", key, err)
				continue
			}

			pipe.SetEx(context.Background(), cacheKey, string(data), ttl)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("批量设置缓存失败: %w", err)
	}

	return nil
}

// GetMultiple 批量获取缓存项
func (m *DistributedCacheManager) GetMultiple(ctx context.Context, keys []string) (map[string]interface{}, error) {
	if len(keys) == 0 {
		return make(map[string]interface{}), nil
	}

	// 构建缓存键
	cacheKeys := make([]string, len(keys))
	for i, key := range keys {
		cacheKeys[i] = m.buildKey(key)
	}

	// 批量获取
	values, err := m.redis.MgetCtx(ctx, cacheKeys...)
	if err != nil {
		return nil, fmt.Errorf("批量获取缓存失败: %w", err)
	}

	// 处理结果
	result := make(map[string]interface{})
	for i, value := range values {
		if value == "" {
			continue // 键不存在
		}

		// 反序列化数据
		var cacheValue CacheValue
		if err := json.Unmarshal([]byte(value), &cacheValue); err != nil {
			logx.Errorf("反序列化缓存数据失败 [%s]: %v", keys[i], err)
			continue
		}

		result[keys[i]] = cacheValue.Data
	}

	return result, nil
}

// GetStats 获取缓存统计信息
func (m *DistributedCacheManager) GetStats(ctx context.Context) (*DistributedCacheStats, error) {
	// redis.Redis 没有 InfoCtx 方法，这里使用 Pipelined 来执行 INFO 命令，或者直接忽略 Info
	// 由于 go-zero redis 封装没有直接暴露 Info 命令，我们这里简化实现，只返回 KeyCount

	stats := &DistributedCacheStats{
		Info: "info command not supported directly by go-zero redis wrapper",
	}

	// 统计键数量
	pattern := m.buildKey("*")
	keys, err := m.scanKeys(ctx, pattern)
	if err != nil {
		logx.Errorf("扫描缓存键失败: %v", err)
	} else {
		stats.KeyCount = len(keys)
	}

	return stats, nil
}

// 私有方法

// buildKey 构建完整的缓存键
func (m *DistributedCacheManager) buildKey(key string) string {
	if m.config.KeyPrefix == "" {
		return key
	}
	return m.config.KeyPrefix + ":" + key
}

// scanKeys 扫描匹配的键
func (m *DistributedCacheManager) scanKeys(ctx context.Context, pattern string) ([]string, error) {
	var allKeys []string
	cursor := uint64(0)

	for {
		keys, nextCursor, err := m.redis.ScanCtx(ctx, cursor, pattern, 100)
		if err != nil {
			return nil, err
		}

		allKeys = append(allKeys, keys...)

		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}

	return allKeys, nil
}

// batchDelete 批量删除键
func (m *DistributedCacheManager) batchDelete(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	batchSize := m.config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[i:end]
		_, err := m.redis.DelCtx(ctx, batch...)
		if err != nil {
			logx.Errorf("批量删除缓存失败 [batch %d-%d]: %v", i, end-1, err)
			return err
		}
	}

	return nil
}

// DistributedCacheStats 分布式缓存统计信息
type DistributedCacheStats struct {
	KeyCount int    `json:"keyCount"`
	Info     string `json:"info"`
}

// 默认配置
func getDefaultDistributedCacheConfig() *DistributedCacheConfig {
	return &DistributedCacheConfig{
		DefaultTTL:          15 * time.Minute,
		KeyPrefix:           "asset_selector",
		CompressEnabled:     false,
		SerializationFormat: "json",
		BatchSize:           100,
		RetryCount:          3,
		RetryDelay:          100 * time.Millisecond,
	}
}
