package idempotency

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

// Guard 幂等性守护者
type Guard interface {
	// CheckAndMark 检查并标记消息是否已处理
	// 返回 (true, nil) 表示首次处理
	// 返回 (false, nil) 表示重复消息
	// 返回 (false, err) 表示检查失败
	CheckAndMark(ctx context.Context, tenantID uint64, taskRunID uint64) (bool, error)

	// Remove 移除幂等性标记（用于测试）
	Remove(ctx context.Context, tenantID uint64, taskRunID uint64) error
}

// Config 幂等性配置
type Config struct {
	WindowTime time.Duration // 幂等窗口时间
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		WindowTime: 1 * time.Hour, // 默认1小时
	}
}

// redisGuard 基于Redis的幂等性守护者
type redisGuard struct {
	redis      redis.UniversalClient
	windowTime time.Duration
}

// NewGuard 创建幂等性守护者
func NewGuard(rds redis.UniversalClient, config *Config) Guard {
	if config == nil {
		config = DefaultConfig()
	}

	return &redisGuard{
		redis:      rds,
		windowTime: config.WindowTime,
	}
}

// CheckAndMark 检查并标记消息是否已处理
func (g *redisGuard) CheckAndMark(ctx context.Context, tenantID uint64, taskRunID uint64) (bool, error) {
	key := fmt.Sprintf("idempotency:%d:%d", tenantID, taskRunID)

	// 使用 SETNX + EXPIRE 原子性检查并标记
	ok, err := g.redis.SetNX(ctx, key, time.Now().Unix(), g.windowTime).Result()
	if err != nil {
		logx.Errorw("redis setnx failed",
			logx.Field("tenant_id", tenantID),
			logx.Field("task_run_id", taskRunID),
			logx.Field("error", err))
		return false, fmt.Errorf("redis setnx: %w", err)
	}

	if !ok {
		// 已处理过
		logx.Infow("duplicate message detected (Redis)",
			logx.Field("tenant_id", tenantID),
			logx.Field("task_run_id", taskRunID))
		return false, nil
	}

	// 首次处理
	logx.Infow("new message marked",
		logx.Field("tenant_id", tenantID),
		logx.Field("task_run_id", taskRunID),
		logx.Field("window_time", g.windowTime))

	return true, nil
}

// Remove 移除幂等性标记
func (g *redisGuard) Remove(ctx context.Context, tenantID uint64, taskRunID uint64) error {
	key := fmt.Sprintf("idempotency:%d:%d", tenantID, taskRunID)
	return g.redis.Del(ctx, key).Err()
}

// CompositeGuard 组合守护者（Redis + DB）
type CompositeGuard struct {
	redisGuard Guard
	dbChecker  DBChecker
}

// DBChecker 数据库幂等性检查器接口
type DBChecker interface {
	// CheckAndInsert 检查并插入幂等性记录
	// 返回 (true, nil) 表示首次处理
	// 返回 (false, nil) 表示重复消息（数据库唯一约束冲突）
	// 返回 (false, err) 表示检查失败
	CheckAndInsert(ctx context.Context, tenantID uint64, taskRunID uint64, idempotencyKey string) (bool, error)
}

// NewCompositeGuard 创建组合守护者
func NewCompositeGuard(redisGuard Guard, dbChecker DBChecker) *CompositeGuard {
	return &CompositeGuard{
		redisGuard: redisGuard,
		dbChecker:  dbChecker,
	}
}

// CheckAndMark 两层检查：先Redis，再DB
func (g *CompositeGuard) CheckAndMark(ctx context.Context, tenantID uint64, taskRunID uint64, idempotencyKey string) (bool, error) {
	// 第一层：Redis快速去重
	isNew, err := g.redisGuard.CheckAndMark(ctx, tenantID, taskRunID)
	if err != nil {
		// Redis失败，继续使用DB检查（降级）
		logx.Errorw("redis check failed, fallback to DB",
			logx.Field("tenant_id", tenantID),
			logx.Field("task_run_id", taskRunID),
			logx.Field("error", err))
	} else if !isNew {
		// Redis检测到重复
		return false, nil
	}

	// 第二层：DB持久化去重
	if g.dbChecker != nil {
		isNew, err := g.dbChecker.CheckAndInsert(ctx, tenantID, taskRunID, idempotencyKey)
		if err != nil {
			return false, fmt.Errorf("db check failed: %w", err)
		}
		if !isNew {
			logx.Infow("duplicate message detected (DB)",
				logx.Field("tenant_id", tenantID),
				logx.Field("task_run_id", taskRunID))
			return false, nil
		}
	}

	return true, nil
}
