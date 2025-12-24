package idempotency

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedis 设置测试用Redis
func setupTestRedis(t *testing.T) (redis.UniversalClient, func()) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cleanup := func() {
		client.Close()
		mr.Close()
	}

	return client, cleanup
}

// TestDefaultConfig 测试默认配置
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, 1*time.Hour, config.WindowTime)
}

// TestNewGuard 测试创建Guard
func TestNewGuard(t *testing.T) {
	rds, cleanup := setupTestRedis(t)
	defer cleanup()

	tests := []struct {
		name   string
		config *Config
	}{
		{
			name:   "with config",
			config: &Config{WindowTime: 30 * time.Minute},
		},
		{
			name:   "with nil config (uses default)",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guard := NewGuard(rds, tt.config)
			assert.NotNil(t, guard)
		})
	}
}

// TestGuard_CheckAndMark 测试检查并标记
func TestGuard_CheckAndMark(t *testing.T) {
	rds, cleanup := setupTestRedis(t)
	defer cleanup()

	config := &Config{WindowTime: 1 * time.Hour}
	guard := NewGuard(rds, config)

	ctx := context.Background()
	tenantID := uint64(1)
	taskRunID := uint64(100)

	// 第一次检查：应该是新消息
	isNew, err := guard.CheckAndMark(ctx, tenantID, taskRunID)
	require.NoError(t, err)
	assert.True(t, isNew, "first check should return true")

	// 第二次检查：应该是重复消息
	isNew, err = guard.CheckAndMark(ctx, tenantID, taskRunID)
	require.NoError(t, err)
	assert.False(t, isNew, "second check should return false (duplicate)")
}

// TestGuard_DifferentTenants 测试不同租户的消息
func TestGuard_DifferentTenants(t *testing.T) {
	rds, cleanup := setupTestRedis(t)
	defer cleanup()

	guard := NewGuard(rds, DefaultConfig())
	ctx := context.Background()

	tenant1 := uint64(1)
	tenant2 := uint64(2)
	taskRunID := uint64(100)

	// 租户1的消息
	isNew, err := guard.CheckAndMark(ctx, tenant1, taskRunID)
	require.NoError(t, err)
	assert.True(t, isNew)

	// 租户2的消息（相同taskRunID，但不同租户）
	isNew, err = guard.CheckAndMark(ctx, tenant2, taskRunID)
	require.NoError(t, err)
	assert.True(t, isNew, "different tenant should be treated as new message")

	// 租户1再次检查
	isNew, err = guard.CheckAndMark(ctx, tenant1, taskRunID)
	require.NoError(t, err)
	assert.False(t, isNew, "tenant1 duplicate check")
}

// TestGuard_DifferentTaskRuns 测试不同任务的消息
func TestGuard_DifferentTaskRuns(t *testing.T) {
	rds, cleanup := setupTestRedis(t)
	defer cleanup()

	guard := NewGuard(rds, DefaultConfig())
	ctx := context.Background()

	tenantID := uint64(1)
	taskRun1 := uint64(100)
	taskRun2 := uint64(101)

	// 任务1
	isNew, err := guard.CheckAndMark(ctx, tenantID, taskRun1)
	require.NoError(t, err)
	assert.True(t, isNew)

	// 任务2
	isNew, err = guard.CheckAndMark(ctx, tenantID, taskRun2)
	require.NoError(t, err)
	assert.True(t, isNew, "different task run should be treated as new message")

	// 任务1再次检查
	isNew, err = guard.CheckAndMark(ctx, tenantID, taskRun1)
	require.NoError(t, err)
	assert.False(t, isNew, "task run 1 duplicate check")
}

// TestGuard_Remove 测试移除幂等性标记
func TestGuard_Remove(t *testing.T) {
	rds, cleanup := setupTestRedis(t)
	defer cleanup()

	guard := NewGuard(rds, DefaultConfig())
	ctx := context.Background()

	tenantID := uint64(1)
	taskRunID := uint64(100)

	// 标记消息
	isNew, err := guard.CheckAndMark(ctx, tenantID, taskRunID)
	require.NoError(t, err)
	assert.True(t, isNew)

	// 验证重复检查
	isNew, err = guard.CheckAndMark(ctx, tenantID, taskRunID)
	require.NoError(t, err)
	assert.False(t, isNew)

	// 移除标记
	err = guard.Remove(ctx, tenantID, taskRunID)
	require.NoError(t, err)

	// 再次检查：应该是新消息
	isNew, err = guard.CheckAndMark(ctx, tenantID, taskRunID)
	require.NoError(t, err)
	assert.True(t, isNew, "after removal, should be treated as new message")
}

// TestGuard_WindowExpiration 测试窗口过期
func TestGuard_WindowExpiration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping window expiration test in short mode")
	}

	rds, cleanup := setupTestRedis(t)
	defer cleanup()

	// 设置短窗口时间用于测试
	config := &Config{WindowTime: 100 * time.Millisecond}
	guard := NewGuard(rds, config)

	ctx := context.Background()
	tenantID := uint64(1)
	taskRunID := uint64(100)

	// 标记消息
	isNew, err := guard.CheckAndMark(ctx, tenantID, taskRunID)
	require.NoError(t, err)
	assert.True(t, isNew)

	// 等待窗口过期
	time.Sleep(150 * time.Millisecond)

	// 再次检查：应该是新消息（因为窗口已过期）
	isNew, err = guard.CheckAndMark(ctx, tenantID, taskRunID)
	require.NoError(t, err)
	assert.True(t, isNew, "after window expiration, should be treated as new message")
}
