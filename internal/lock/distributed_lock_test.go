package lock

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedis 创建测试用的Redis客户端（基于miniredis）
func setupTestRedis(t *testing.T) (redis.UniversalClient, *miniredis.Miniredis) {
	// 创建内存Redis服务器
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 测试连接
	err = client.Ping(context.Background()).Err()
	require.NoError(t, err)

	return client, mr
}

func TestNewDistributedLockManager(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	// 创建锁管理器
	manager := NewDistributedLockManager(client)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.rs)
	assert.NotNil(t, manager.logger)
}

func TestDistributedLock_TryLock_Success(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	manager := NewDistributedLockManager(client)
	ctx := context.Background()

	// 创建锁
	lock := manager.NewLock("test:lock:1", 10*time.Second, 1, 0)

	// 第一次尝试获取锁，应该成功
	acquired, err := lock.TryLock(ctx)
	require.NoError(t, err)
	assert.True(t, acquired, "First TryLock should succeed")

	// 释放锁
	err = lock.Unlock()
	require.NoError(t, err)
}

func TestDistributedLock_TryLock_AlreadyHeld(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	manager := NewDistributedLockManager(client)
	ctx := context.Background()

	lockKey := "test:lock:2"

	// 第一个锁
	lock1 := manager.NewLock(lockKey, 10*time.Second, 1, 0)
	acquired1, err := lock1.TryLock(ctx)
	require.NoError(t, err)
	assert.True(t, acquired1, "First lock should succeed")

	// 第二个锁（同一个key）
	lock2 := manager.NewLock(lockKey, 10*time.Second, 1, 0)
	acquired2, err := lock2.TryLock(ctx)
	require.NoError(t, err)
	assert.False(t, acquired2, "Second TryLock should fail (lock held)")

	// 释放第一个锁
	err = lock1.Unlock()
	require.NoError(t, err)

	// 现在第二个锁应该可以获取
	lock3 := manager.NewLock(lockKey, 10*time.Second, 1, 0)
	acquired3, err := lock3.TryLock(ctx)
	require.NoError(t, err)
	assert.True(t, acquired3, "Third TryLock should succeed after unlock")

	// 清理
	err = lock3.Unlock()
	require.NoError(t, err)
}

func TestDistributedLock_Unlock(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	manager := NewDistributedLockManager(client)
	ctx := context.Background()

	lock := manager.NewLock("test:lock:3", 10*time.Second, 1, 0)

	// 获取锁
	acquired, err := lock.TryLock(ctx)
	require.NoError(t, err)
	require.True(t, acquired)

	// 释放锁
	err = lock.Unlock()
	require.NoError(t, err)

	// 再次释放应该失败（锁已经不存在）
	err = lock.Unlock()
	assert.Error(t, err, "Unlocking already released lock should fail")
}

func TestDistributedLock_Expiry(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping expiry test in short mode")
	}

	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	manager := NewDistributedLockManager(client)
	ctx := context.Background()

	lockKey := "test:lock:expiry"

	// 创建一个1秒过期的锁
	lock1 := manager.NewLock(lockKey, 1*time.Second, 1, 0)
	acquired1, err := lock1.TryLock(ctx)
	require.NoError(t, err)
	require.True(t, acquired1)

	// 快进时间（miniredis支持）
	mr.FastForward(2 * time.Second)

	// 现在锁应该已经过期，另一个锁可以获取
	lock2 := manager.NewLock(lockKey, 10*time.Second, 1, 0)
	acquired2, err := lock2.TryLock(ctx)
	require.NoError(t, err)
	assert.True(t, acquired2, "Lock should be acquirable after expiry")

	// 清理
	_ = lock2.Unlock()
}

func TestDistributedLockManager_ExecuteWithLockOrSkip_Success(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	manager := NewDistributedLockManager(client)
	ctx := context.Background()

	executed := false
	lockKey := "test:lock:execute:1"

	// 执行函数，应该成功获取锁并执行
	result, err := manager.ExecuteWithLockOrSkip(
		ctx,
		lockKey,
		10*time.Second,
		func() error {
			executed = true
			return nil
		},
	)

	require.NoError(t, err)
	assert.True(t, result, "Should have executed")
	assert.True(t, executed, "Function should have been called")
}

func TestDistributedLockManager_ExecuteWithLockOrSkip_Skip(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	manager := NewDistributedLockManager(client)
	ctx := context.Background()

	lockKey := "test:lock:execute:2"

	// 先获取锁
	lock := manager.NewLock(lockKey, 10*time.Second, 1, 0)
	acquired, err := lock.TryLock(ctx)
	require.NoError(t, err)
	require.True(t, acquired)
	defer lock.Unlock()

	// 现在尝试执行，应该跳过
	executed := false
	result, err := manager.ExecuteWithLockOrSkip(
		ctx,
		lockKey,
		10*time.Second,
		func() error {
			executed = true
			return nil
		},
	)

	require.NoError(t, err)
	assert.False(t, result, "Should have skipped")
	assert.False(t, executed, "Function should not have been called")
}

func TestDistributedLockManager_ExecuteWithLockOrSkip_FunctionError(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	manager := NewDistributedLockManager(client)
	ctx := context.Background()

	lockKey := "test:lock:execute:3"

	// 执行一个返回错误的函数
	executed := false
	expectedErr := assert.AnError
	result, err := manager.ExecuteWithLockOrSkip(
		ctx,
		lockKey,
		10*time.Second,
		func() error {
			executed = true
			return expectedErr
		},
	)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.True(t, result, "Should have been executed")
	assert.True(t, executed, "Function should have been called despite error")
}

func TestDistributedLock_Refresh(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	manager := NewDistributedLockManager(client)
	ctx := context.Background()

	lock := manager.NewLock("test:lock:refresh", 2*time.Second, 1, 0)

	// 获取锁
	acquired, err := lock.TryLock(ctx)
	require.NoError(t, err)
	require.True(t, acquired)
	defer lock.Unlock()

	// 刷新锁
	refreshed, err := lock.Refresh(ctx)
	require.NoError(t, err)
	assert.True(t, refreshed, "Refresh should succeed for held lock")
}

func TestDistributedLock_ConcurrentAccess(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	manager := NewDistributedLockManager(client)
	ctx := context.Background()

	lockKey := "test:lock:concurrent"
	var successCount, skipCount int32
	done := make(chan bool, 10)

	// 10个goroutine同时尝试获取锁
	for i := 0; i < 10; i++ {
		go func() {
			executed, err := manager.ExecuteWithLockOrSkip(
				ctx,
				lockKey,
				10*time.Second,
				func() error {
					time.Sleep(50 * time.Millisecond) // 模拟工作，确保其他goroutine都启动
					return nil
				},
			)
			if err == nil {
				if executed {
					atomic.AddInt32(&successCount, 1)
				} else {
					atomic.AddInt32(&skipCount, 1)
				}
			}
			done <- true
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证：总数应该是10，至少有一些被跳过（证明锁工作了）
	total := atomic.LoadInt32(&successCount) + atomic.LoadInt32(&skipCount)
	assert.Equal(t, int32(10), total, "All goroutines should complete")
	assert.Greater(t, atomic.LoadInt32(&skipCount), int32(0), "Some goroutines should skip (lock is working)")
	// 由于时间竞争，可能有1-2个goroutine成功执行（第一个完成后，第二个立即获取锁）
	assert.LessOrEqual(t, atomic.LoadInt32(&successCount), int32(2), "At most 2 goroutines should execute")
}
