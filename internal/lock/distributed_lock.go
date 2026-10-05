package lock

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

// DistributedLock 分布式锁接口
type DistributedLock interface {
	// TryLock 尝试获取锁
	// ctx: 上下文
	// returns: 是否获取成功
	TryLock(ctx context.Context) (bool, error)

	// Lock 阻塞获取锁（带重试）
	// ctx: 上下文
	// returns: 错误
	Lock(ctx context.Context) error

	// Unlock 释放锁
	// returns: 错误
	Unlock() error

	// Refresh 刷新锁的过期时间
	// ctx: 上下文
	// returns: 是否刷新成功
	Refresh(ctx context.Context) (bool, error)
}

// RedisDistributedLock Redis分布式锁实现
type RedisDistributedLock struct {
	mutex   *redsync.Mutex
	lockKey string
	logger  logx.Logger
}

// DistributedLockManager 分布式锁管理器
type DistributedLockManager struct {
	rs     *redsync.Redsync
	logger logx.Logger
}

// NewDistributedLockManager 创建分布式锁管理器
func NewDistributedLockManager(redisClient redis.UniversalClient) *DistributedLockManager {
	// 创建redsync实例
	pool := goredis.NewPool(redisClient)
	rs := redsync.New(pool)

	return &DistributedLockManager{
		rs:     rs,
		logger: logx.WithContext(context.Background()),
	}
}

// NewLock 创建一个新的分布式锁
// lockKey: 锁的唯一标识
// expiry: 锁的过期时间（防止死锁）
// retries: 获取锁的重试次数
// retryDelay: 重试间隔
func (m *DistributedLockManager) NewLock(
	lockKey string,
	expiry time.Duration,
	retries int,
	retryDelay time.Duration,
) DistributedLock {
	// 使用redsync创建mutex
	mutex := m.rs.NewMutex(
		lockKey,
		redsync.WithExpiry(expiry),
		redsync.WithTries(retries),
		redsync.WithRetryDelay(retryDelay),
	)

	return &RedisDistributedLock{
		mutex:   mutex,
		lockKey: lockKey,
		logger:  m.logger,
	}
}

// TryLock 尝试获取锁（非阻塞）
func (l *RedisDistributedLock) TryLock(ctx context.Context) (bool, error) {
	err := l.mutex.TryLockContext(ctx)
	if err != nil {
		// 如果是因为锁已被占用，返回false而不是error
		// redsync在锁被占用时返回包含"lock already taken"的错误
		errMsg := err.Error()
		if strings.Contains(errMsg, "lock already taken") || errMsg == "failed to acquire lock" {
			l.logger.Debugw("Lock already taken",
				logx.Field("lock_key", l.lockKey))
			return false, nil
		}
		l.logger.Errorw("Failed to try lock",
			logx.Field("lock_key", l.lockKey),
			logx.Field("error", err))
		return false, fmt.Errorf("failed to try lock %s: %w", l.lockKey, err)
	}

	l.logger.Debugw("Lock acquired successfully",
		logx.Field("lock_key", l.lockKey))
	return true, nil
}

// Lock 阻塞获取锁（带重试）
func (l *RedisDistributedLock) Lock(ctx context.Context) error {
	err := l.mutex.LockContext(ctx)
	if err != nil {
		l.logger.Errorw("Failed to acquire lock",
			logx.Field("lock_key", l.lockKey),
			logx.Field("error", err))
		return fmt.Errorf("failed to acquire lock %s: %w", l.lockKey, err)
	}

	l.logger.Debugw("Lock acquired successfully (blocking)",
		logx.Field("lock_key", l.lockKey))
	return nil
}

// Unlock 释放锁
func (l *RedisDistributedLock) Unlock() error {
	ok, err := l.mutex.Unlock()
	if err != nil {
		l.logger.Errorw("Failed to unlock",
			logx.Field("lock_key", l.lockKey),
			logx.Field("error", err))
		return fmt.Errorf("failed to unlock %s: %w", l.lockKey, err)
	}

	if !ok {
		l.logger.Infow("Lock was not held or already released",
			logx.Field("lock_key", l.lockKey))
		return fmt.Errorf("lock %s was not held", l.lockKey)
	}

	l.logger.Debugw("Lock released successfully",
		logx.Field("lock_key", l.lockKey))
	return nil
}

// Refresh 刷新锁的过期时间
func (l *RedisDistributedLock) Refresh(ctx context.Context) (bool, error) {
	ok, err := l.mutex.ExtendContext(ctx)
	if err != nil {
		l.logger.Errorw("Failed to refresh lock",
			logx.Field("lock_key", l.lockKey),
			logx.Field("error", err))
		return false, fmt.Errorf("failed to refresh lock %s: %w", l.lockKey, err)
	}

	if !ok {
		l.logger.Infow("Failed to refresh lock (might have expired)",
			logx.Field("lock_key", l.lockKey))
	}

	return ok, nil
}

// LockGuard 使用defer自动释放锁的帮助函数
// 示例用法:
//
//	lock := manager.NewLock("my-lock", 10*time.Second, 3, time.Second)
//	defer lock.LockGuard(ctx)()
//	// ... 业务逻辑 ...
func (l *RedisDistributedLock) LockGuard(ctx context.Context) func() {
	// 尝试获取锁
	acquired, err := l.TryLock(ctx)
	if err != nil || !acquired {
		l.logger.Infow("LockGuard: Failed to acquire lock",
			logx.Field("lock_key", l.lockKey),
			logx.Field("acquired", acquired),
			logx.Field("error", err))
		// 返回空函数，防止panic
		return func() {}
	}

	// 返回unlock函数
	return func() {
		if err := l.Unlock(); err != nil {
			l.logger.Errorw("LockGuard: Failed to release lock",
				logx.Field("lock_key", l.lockKey),
				logx.Field("error", err))
		}
	}
}

// ExecuteWithLock 在分布式锁保护下执行函数
// 示例用法:
//
//	err := manager.ExecuteWithLock(ctx, "my-lock", 10*time.Second, func() error {
//	    // ... 业务逻辑 ...
//	    return nil
//	})
func (m *DistributedLockManager) ExecuteWithLock(
	ctx context.Context,
	lockKey string,
	expiry time.Duration,
	fn func() error,
) error {
	// 创建锁（默认重试3次，每次间隔1秒）
	lock := m.NewLock(lockKey, expiry, 3, time.Second)

	// 尝试获取锁
	acquired, err := lock.TryLock(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !acquired {
		return fmt.Errorf("lock %s is already held by another process", lockKey)
	}

	// 确保锁被释放
	defer func() {
		if err := lock.Unlock(); err != nil {
			m.logger.Errorw("ExecuteWithLock: Failed to release lock",
				logx.Field("lock_key", lockKey),
				logx.Field("error", err))
		}
	}()

	// 执行业务逻辑
	return fn()
}

// ExecuteWithLockOrSkip 在分布式锁保护下执行函数，如果获取锁失败则跳过
// 返回: (executed bool, error)
// executed: 是否执行了业务逻辑
func (m *DistributedLockManager) ExecuteWithLockOrSkip(
	ctx context.Context,
	lockKey string,
	expiry time.Duration,
	fn func() error,
) (bool, error) {
	// 创建锁
	lock := m.NewLock(lockKey, expiry, 1, 0) // 只尝试一次，不重试

	// 尝试获取锁
	acquired, err := lock.TryLock(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to try lock: %w", err)
	}

	if !acquired {
		// 锁已被占用，跳过执行
		m.logger.Infow("Lock is held by another instance, skipping execution",
			logx.Field("lock_key", lockKey))
		return false, nil
	}

	// 确保锁被释放
	defer func() {
		if err := lock.Unlock(); err != nil {
			m.logger.Errorw("ExecuteWithLockOrSkip: Failed to release lock",
				logx.Field("lock_key", lockKey),
				logx.Field("error", err))
		}
	}()

	// 执行业务逻辑
	if err := fn(); err != nil {
		return true, err
	}

	return true, nil
}
