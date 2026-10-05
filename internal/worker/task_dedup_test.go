package worker

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/internal/lock"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedis 创建测试用的Redis客户端
func setupTestRedis(t *testing.T) (redis.UniversalClient, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	err = client.Ping(context.Background()).Err()
	require.NoError(t, err)

	return client, mr
}

func TestComputeTaskFingerprint_CronTask(t *testing.T) {
	now := time.Now()
	cronTaskID := uint64(123)
	executionTime := now

	task := &ent.InputTask{
		ID:            1,
		TaskName:      "Test Cron Task",
		CronTaskID:    &cronTaskID,
		ExecutionTime: &executionTime,
		CreatedAt:     now,
	}

	fingerprint := ComputeTaskFingerprint(task)

	// 应该是16位的十六进制字符串
	assert.Len(t, fingerprint, 16)
	assert.Regexp(t, "^[0-9a-f]{16}$", fingerprint)

	// 相同的输入应该生成相同的指纹
	fingerprint2 := ComputeTaskFingerprint(task)
	assert.Equal(t, fingerprint, fingerprint2, "Same task should produce same fingerprint")
}

func TestComputeTaskFingerprint_NonCronTask(t *testing.T) {
	now := time.Now()
	sourceConfig := `{"host": "localhost"}`

	task := &ent.InputTask{
		ID:           1,
		TaskName:     "Test Manual Task",
		InputSource:  "file",
		SourceConfig: sourceConfig,
		CreatedAt:    now,
	}

	fingerprint := ComputeTaskFingerprint(task)

	assert.Len(t, fingerprint, 16)
	assert.Regexp(t, "^[0-9a-f]{16}$", fingerprint)

	// 相同的输入应该生成相同的指纹
	fingerprint2 := ComputeTaskFingerprint(task)
	assert.Equal(t, fingerprint, fingerprint2)
}

func TestComputeTaskFingerprint_DifferentTasks(t *testing.T) {
	now := time.Now()
	sourceConfig := `{"host": "localhost"}`

	task1 := &ent.InputTask{
		ID:           1,
		TaskName:     "Task 1",
		InputSource:  "file",
		SourceConfig: sourceConfig,
		CreatedAt:    now,
	}

	task2 := &ent.InputTask{
		ID:           2,
		TaskName:     "Task 2", // 不同的名称
		InputSource:  "file",
		SourceConfig: sourceConfig,
		CreatedAt:    now,
	}

	fingerprint1 := ComputeTaskFingerprint(task1)
	fingerprint2 := ComputeTaskFingerprint(task2)

	assert.NotEqual(t, fingerprint1, fingerprint2, "Different tasks should have different fingerprints")
}

func TestNewTaskDedupManager(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	lockManager := lock.NewDistributedLockManager(client)
	dedupManager := NewTaskDedupManager(lockManager)

	assert.NotNil(t, dedupManager)
	assert.NotNil(t, dedupManager.lockManager)
}

func TestTaskDedupManager_TryAcquireTaskLock_Success(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	lockManager := lock.NewDistributedLockManager(client)
	dedupManager := NewTaskDedupManager(lockManager)
	ctx := context.Background()

	taskID := uint64(123)

	// 第一次获取锁应该成功
	acquired, unlock, err := dedupManager.TryAcquireTaskLock(ctx, taskID)
	require.NoError(t, err)
	assert.True(t, acquired, "First lock acquisition should succeed")
	assert.NotNil(t, unlock, "Unlock function should be provided")

	// 释放锁
	unlock()
}

func TestTaskDedupManager_TryAcquireTaskLock_AlreadyHeld(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	lockManager := lock.NewDistributedLockManager(client)
	dedupManager := NewTaskDedupManager(lockManager)
	ctx := context.Background()

	taskID := uint64(456)

	// 第一次获取锁
	acquired1, unlock1, err := dedupManager.TryAcquireTaskLock(ctx, taskID)
	require.NoError(t, err)
	require.True(t, acquired1)
	defer unlock1()

	// 第二次尝试获取同一个任务的锁，应该失败
	acquired2, unlock2, err := dedupManager.TryAcquireTaskLock(ctx, taskID)
	require.NoError(t, err)
	assert.False(t, acquired2, "Second lock acquisition should fail")
	assert.Nil(t, unlock2, "Unlock function should be nil when lock not acquired")
}

func TestTaskDedupManager_TryAcquireTaskLock_AfterRelease(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	lockManager := lock.NewDistributedLockManager(client)
	dedupManager := NewTaskDedupManager(lockManager)
	ctx := context.Background()

	taskID := uint64(789)

	// 第一次获取锁
	acquired1, unlock1, err := dedupManager.TryAcquireTaskLock(ctx, taskID)
	require.NoError(t, err)
	require.True(t, acquired1)

	// 释放锁
	unlock1()

	// 现在应该可以再次获取锁
	acquired2, unlock2, err := dedupManager.TryAcquireTaskLock(ctx, taskID)
	require.NoError(t, err)
	assert.True(t, acquired2, "Lock should be acquirable after release")
	defer unlock2()
}

func TestTaskDedupManager_TryAcquireTaskLock_NilLockManager(t *testing.T) {
	// 创建没有lockManager的dedupManager
	dedupManager := &TaskDedupManager{
		lockManager: nil,
	}
	ctx := context.Background()

	taskID := uint64(999)

	// 应该返回成功（允许单实例运行）
	acquired, unlock, err := dedupManager.TryAcquireTaskLock(ctx, taskID)
	require.NoError(t, err)
	assert.True(t, acquired, "Should succeed when lockManager is nil")
	assert.NotNil(t, unlock, "Should provide a no-op unlock function")

	// 调用unlock不应该panic
	unlock()
}

func TestTaskDedupManager_ExecuteWithTaskLock_Success(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	lockManager := lock.NewDistributedLockManager(client)
	dedupManager := NewTaskDedupManager(lockManager)
	ctx := context.Background()

	taskID := uint64(111)
	executed := false

	// 执行函数
	result, err := dedupManager.ExecuteWithTaskLock(
		ctx,
		taskID,
		func() error {
			executed = true
			return nil
		},
	)

	require.NoError(t, err)
	assert.True(t, result, "Should have executed")
	assert.True(t, executed, "Function should have been called")
}

func TestTaskDedupManager_ExecuteWithTaskLock_Skip(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	lockManager := lock.NewDistributedLockManager(client)
	dedupManager := NewTaskDedupManager(lockManager)
	ctx := context.Background()

	taskID := uint64(222)

	// 先获取锁
	acquired, unlock, err := dedupManager.TryAcquireTaskLock(ctx, taskID)
	require.NoError(t, err)
	require.True(t, acquired)
	defer unlock()

	// 现在尝试执行，应该跳过
	executed := false
	result, err := dedupManager.ExecuteWithTaskLock(
		ctx,
		taskID,
		func() error {
			executed = true
			return nil
		},
	)

	require.NoError(t, err)
	assert.False(t, result, "Should have skipped")
	assert.False(t, executed, "Function should not have been called")
}

func TestTaskDedupManager_ExecuteWithTaskLock_FunctionError(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	lockManager := lock.NewDistributedLockManager(client)
	dedupManager := NewTaskDedupManager(lockManager)
	ctx := context.Background()

	taskID := uint64(333)
	executed := false
	expectedErr := assert.AnError

	// 执行一个返回错误的函数
	result, err := dedupManager.ExecuteWithTaskLock(
		ctx,
		taskID,
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

func TestTaskDedupManager_ConcurrentLockAttempts(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	lockManager := lock.NewDistributedLockManager(client)
	dedupManager := NewTaskDedupManager(lockManager)
	ctx := context.Background()

	taskID := uint64(444)
	successCount := 0
	skipCount := 0
	done := make(chan bool, 10)

	// 10个goroutine同时尝试获取同一个任务的锁
	for i := 0; i < 10; i++ {
		go func() {
			executed, err := dedupManager.ExecuteWithTaskLock(
				ctx,
				taskID,
				func() error {
					time.Sleep(10 * time.Millisecond) // 模拟任务处理
					return nil
				},
			)
			if err == nil {
				if executed {
					successCount++
				} else {
					skipCount++
				}
			}
			done <- true
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 只有一个应该成功执行
	assert.Equal(t, 1, successCount, "Only one goroutine should execute")
	assert.Equal(t, 9, skipCount, "Nine goroutines should skip")
}

func TestTaskDedupManager_DifferentTasks(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()
	defer client.Close()

	lockManager := lock.NewDistributedLockManager(client)
	dedupManager := NewTaskDedupManager(lockManager)
	ctx := context.Background()

	taskID1 := uint64(555)
	taskID2 := uint64(666)

	// 获取第一个任务的锁
	acquired1, unlock1, err := dedupManager.TryAcquireTaskLock(ctx, taskID1)
	require.NoError(t, err)
	require.True(t, acquired1)
	defer unlock1()

	// 应该能够获取第二个任务的锁（不同的taskID）
	acquired2, unlock2, err := dedupManager.TryAcquireTaskLock(ctx, taskID2)
	require.NoError(t, err)
	assert.True(t, acquired2, "Different tasks should have independent locks")
	defer unlock2()
}
