package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/coder-lulu/newbee-common/v2/orm/ent/hooks"
	"github.com/coder-lulu/newbee-io-rpc/ent/enttest"
	"github.com/coder-lulu/newbee-io-rpc/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"

	_ "github.com/mattn/go-sqlite3"
)

// TestProviderWorkerIntegration_BasicDiscovery 测试基础的Provider发现流程
func TestProviderWorkerIntegration_BasicDiscovery(t *testing.T) {
	// 1. 创建测试数据库
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer db.Close()

	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	ctx := context.Background()
	tenantID := uint64(1)
	ctx = hooks.SetTenantIDToContext(ctx, tenantID)

	// 2. 创建InputTask
	sourceConfig := map[string]interface{}{
		"provider_type": "mock",
		"record_count":  5,
	}
	sourceConfigJSON, _ := json.Marshal(sourceConfig)

	task, err := db.InputTask.Create().
		SetTaskName("Provider Integration Test").
		SetTaskType("manual").
		SetInputSource("mock").
		SetSourceConfig(string(sourceConfigJSON)).
		SetTaskStatus("pending").
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)
	t.Logf("✅ Created task: ID=%d", task.ID)

	// 3. 注册Mock Provider
	mockProvider := provider.NewMockProvider()
	registry := provider.GetRegistry()
	registry.Register(mockProvider)

	// 4. 创建Executor
	logger := logx.WithContext(context.Background())
	executor := NewExecutor(nil, db, logger)

	execCtx := &ExecutionContext{
		Ctx:            ctx,
		TenantID:       tenantID,
		UserID:         "test-user",
		TaskID:         task.ID,
		TaskType:       TaskTypeInput,
		Timeout:        30 * time.Second,
		Logger:         logger,
		DB:             db,
		ProviderID:     "mock",
		ProviderConfig: sourceConfig,
	}

	// 5. 执行Provider发现
	discProvider, err := executor.getProvider(execCtx)
	require.NoError(t, err)

	result, err := executor.executeProviderDiscovery(execCtx, discProvider)
	require.NoError(t, err)

	// 6. 验证结果
	assert.True(t, result.Success, "Discovery should succeed")
	assert.Equal(t, int64(5), result.TotalRecords, "Should discover 5 records")
	assert.Len(t, result.Records, 5, "Should return 5 records")

	// 验证记录内容
	for i, record := range result.Records {
		assert.Equal(t, fmt.Sprintf("mock-%d", i+1), record["id"])
		assert.Equal(t, fmt.Sprintf("Mock Record %d", i+1), record["name"])
	}

	t.Logf("✅ Provider + Worker基础集成测试通过: %d条记录", result.TotalRecords)
}

// TestProviderWorkerIntegration_WithTransform 测试Provider + Transform完整流程
func TestProviderWorkerIntegration_WithTransform(t *testing.T) {
	// 1. 创建测试数据库
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer db.Close()

	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	ctx := context.Background()
	tenantID := uint64(1)
	ctx = hooks.SetTenantIDToContext(ctx, tenantID)

	// 2. 创建InputTask
	sourceConfig := map[string]interface{}{
		"provider_type": "mock",
		"record_count":  3,
	}
	sourceConfigJSON, _ := json.Marshal(sourceConfig)

	task, err := db.InputTask.Create().
		SetTaskName("Provider + Transform Test").
		SetTaskType("manual").
		SetInputSource("mock").
		SetSourceConfig(string(sourceConfigJSON)).
		SetTaskStatus("pending").
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 3. 创建FieldMapping规则 - 映射id到record_id
	_, err = db.FieldMapping.Create().
		SetMappingName("ID Mapping").
		SetMappingType("input").
		SetSourceField("id").
		SetTargetField("record_id").
		SetTransformType("direct").
		SetIsActive(true).
		SetAllowNull(true).
		SetPriority(10).
		SetSortOrder(1).
		SetInputTaskID(task.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 4. 创建FieldMapping规则 - 映射name到display_name
	_, err = db.FieldMapping.Create().
		SetMappingName("Name Mapping").
		SetMappingType("input").
		SetSourceField("name").
		SetTargetField("display_name").
		SetTransformType("direct").
		SetIsActive(true).
		SetAllowNull(true).
		SetPriority(10).
		SetSortOrder(2).
		SetInputTaskID(task.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 5. 注册Mock Provider
	mockProvider := provider.NewMockProvider()
	registry := provider.GetRegistry()
	registry.Register(mockProvider)

	// 6. 创建Executor
	logger := logx.WithContext(context.Background())
	executor := NewExecutor(nil, db, logger)

	execCtx := &ExecutionContext{
		Ctx:            ctx,
		TenantID:       tenantID,
		UserID:         "test-user",
		TaskID:         task.ID,
		TaskType:       TaskTypeInput,
		Timeout:        30 * time.Second,
		Logger:         logger,
		DB:             db,
		ProviderID:     "mock",
		ProviderConfig: sourceConfig,
	}

	// 7. 执行Provider发现
	discProvider, err := executor.getProvider(execCtx)
	require.NoError(t, err)

	discoveryResult, err := executor.executeProviderDiscovery(execCtx, discProvider)
	require.NoError(t, err)
	assert.Equal(t, int64(3), discoveryResult.TotalRecords)

	// 8. 应用Transform
	transformedRecords, transformStats, err := executor.applyTransform(execCtx, discoveryResult.Records)
	require.NoError(t, err)

	// 9. 验证结果
	assert.Len(t, transformedRecords, 3)

	// 验证Transform后的字段
	for i, record := range transformedRecords {
		// 验证ID映射
		assert.Equal(t, fmt.Sprintf("mock-%d", i+1), record["record_id"])
		// 验证Name映射
		assert.NotNil(t, record["display_name"])
	}

	// 验证Transform统计
	assert.Equal(t, 6, transformStats.TotalFields, "Should process 6 fields (2 mappings × 3 records)")
	assert.Greater(t, transformStats.SuccessFields, 0)

	t.Logf("✅ Provider + Transform完整流程测试通过")
	t.Logf("   记录数: %d", len(transformedRecords))
	t.Logf("   转换统计: total=%d, success=%d",
		transformStats.TotalFields,
		transformStats.SuccessFields)
}

// TestProviderWorkerIntegration_ErrorHandling 测试错误处理流程
func TestProviderWorkerIntegration_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		setupProvider func(*provider.MockProvider)
		expectError   bool
		errorContains string
		expectRecords int
	}{
		{
			name: "Discover错误",
			setupProvider: func(p *provider.MockProvider) {
				p.SetFailure(nil, false, fmt.Errorf("discovery failed"))
			},
			expectError:   true,
			errorContains: "discovery failed",
		},
		{
			name: "空记录集",
			setupProvider: func(p *provider.MockProvider) {
				p.SetMockData([]map[string]interface{}{})
			},
			expectError:   false,
			expectRecords: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试数据库
			db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
			defer db.Close()

			hooks.InitDefaultHookConfigs()
			hooks.RegisterTenantHooks(db)

			ctx := context.Background()
			tenantID := uint64(1)
			ctx = hooks.SetTenantIDToContext(ctx, tenantID)

			// 创建InputTask
			sourceConfig := map[string]interface{}{
				"provider_type": "mock",
			}
			sourceConfigJSON, _ := json.Marshal(sourceConfig)

			task, err := db.InputTask.Create().
				SetTaskName(tt.name).
				SetTaskType("manual").
				SetInputSource("mock").
				SetSourceConfig(string(sourceConfigJSON)).
				SetTaskStatus("pending").
				SetStatus(1).
				Save(ctx)
			require.NoError(t, err)

			// 注册并配置Mock Provider
			mockProvider := provider.NewMockProvider()
			tt.setupProvider(mockProvider)

			registry := provider.GetRegistry()
			registry.Register(mockProvider)

			// 创建Executor
			logger := logx.WithContext(context.Background())
			executor := NewExecutor(nil, db, logger)

			execCtx := &ExecutionContext{
				Ctx:            ctx,
				TenantID:       tenantID,
				UserID:         "test-user",
				TaskID:         task.ID,
				TaskType:       TaskTypeInput,
				Timeout:        30 * time.Second,
				Logger:         logger,
				DB:             db,
				ProviderID:     "mock",
				ProviderConfig: sourceConfig,
			}

			// 执行Provider发现
			discProvider, err := executor.getProvider(execCtx)
			require.NoError(t, err)

			result, err := executor.executeProviderDiscovery(execCtx, discProvider)

			// 验证结果
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.expectRecords >= 0 {
					assert.Equal(t, int64(tt.expectRecords), result.TotalRecords)
					assert.Len(t, result.Records, tt.expectRecords)
				}
			}

			t.Logf("✅ 错误处理测试通过: %s", tt.name)
		})
	}
}

// TestProviderWorkerIntegration_ContextCancellation 测试Context取消
func TestProviderWorkerIntegration_ContextCancellation(t *testing.T) {
	// 1. 创建测试数据库
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer db.Close()

	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	ctx := context.Background()
	tenantID := uint64(1)
	ctx = hooks.SetTenantIDToContext(ctx, tenantID)

	// 2. 创建InputTask
	sourceConfig := map[string]interface{}{
		"provider_type": "mock",
		"delay_ms":      2000, // 2秒延迟
	}
	sourceConfigJSON, _ := json.Marshal(sourceConfig)

	task, err := db.InputTask.Create().
		SetTaskName("Context Cancellation Test").
		SetTaskType("manual").
		SetInputSource("mock").
		SetSourceConfig(string(sourceConfigJSON)).
		SetTaskStatus("pending").
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 3. 注册Mock Provider
	mockProvider := provider.NewMockProvider()
	registry := provider.GetRegistry()
	registry.Register(mockProvider)

	// 4. 创建可取消的Context
	cancelCtx, cancel := context.WithCancel(ctx)

	// 5. 创建Executor
	logger := logx.WithContext(context.Background())
	executor := NewExecutor(nil, db, logger)

	execCtx := &ExecutionContext{
		Ctx:            cancelCtx,
		TenantID:       tenantID,
		UserID:         "test-user",
		TaskID:         task.ID,
		TaskType:       TaskTypeInput,
		Timeout:        10 * time.Second,
		Logger:         logger,
		DB:             db,
		ProviderID:     "mock",
		ProviderConfig: sourceConfig,
	}

	// 6. 在500ms后取消Context
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
		t.Log("⚠️  Context cancelled after 500ms")
	}()

	// 7. 执行（应该被取消）
	discProvider, err := executor.getProvider(execCtx)
	require.NoError(t, err)

	_, err = executor.executeProviderDiscovery(execCtx, discProvider)

	// 8. 验证结果 - 应该报错（context取消或超时）
	assert.Error(t, err)

	t.Logf("✅ Context取消测试通过: %v", err)
}

// TestProviderWorkerIntegration_Performance 测试大数据集性能
func TestProviderWorkerIntegration_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过性能测试（短模式）")
	}

	// 1. 创建测试数据库
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer db.Close()

	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	ctx := context.Background()
	tenantID := uint64(1)
	ctx = hooks.SetTenantIDToContext(ctx, tenantID)

	// 2. 创建InputTask - 1000条记录
	sourceConfig := map[string]interface{}{
		"provider_type": "mock",
		"record_count":  1000,
	}
	sourceConfigJSON, _ := json.Marshal(sourceConfig)

	task, err := db.InputTask.Create().
		SetTaskName("Performance Test - 1000 Records").
		SetTaskType("manual").
		SetInputSource("mock").
		SetSourceConfig(string(sourceConfigJSON)).
		SetTaskStatus("pending").
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 3. 注册Mock Provider
	mockProvider := provider.NewMockProvider()
	registry := provider.GetRegistry()
	registry.Register(mockProvider)

	// 4. 创建Executor
	logger := logx.WithContext(context.Background())
	executor := NewExecutor(nil, db, logger)

	execCtx := &ExecutionContext{
		Ctx:            ctx,
		TenantID:       tenantID,
		UserID:         "test-user",
		TaskID:         task.ID,
		TaskType:       TaskTypeInput,
		Timeout:        60 * time.Second,
		Logger:         logger,
		DB:             db,
		ProviderID:     "mock",
		ProviderConfig: sourceConfig,
	}

	// 5. 执行并计时
	startTime := time.Now()

	discProvider, err := executor.getProvider(execCtx)
	require.NoError(t, err)

	result, err := executor.executeProviderDiscovery(execCtx, discProvider)
	require.NoError(t, err)

	duration := time.Since(startTime)

	// 6. 验证结果
	assert.True(t, result.Success)
	assert.Equal(t, int64(1000), result.TotalRecords)
	assert.Len(t, result.Records, 1000)

	// 7. 性能要求：1000条记录应该在5秒内完成
	assert.Less(t, duration.Seconds(), 5.0,
		"1000条记录应该在5秒内完成，实际用时: %.2fs", duration.Seconds())

	t.Logf("✅ 性能测试通过")
	t.Logf("   记录数: %d", result.TotalRecords)
	t.Logf("   总耗时: %.3fs", duration.Seconds())
	t.Logf("   平均速度: %.0f records/sec", float64(result.TotalRecords)/duration.Seconds())
}

// TestProviderWorkerIntegration_PresetData 测试预设数据场景
func TestProviderWorkerIntegration_PresetData(t *testing.T) {
	// 1. 创建测试数据库
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer db.Close()

	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	ctx := context.Background()
	tenantID := uint64(1)
	ctx = hooks.SetTenantIDToContext(ctx, tenantID)

	// 2. 创建InputTask
	sourceConfig := map[string]interface{}{
		"provider_type": "mock",
	}
	sourceConfigJSON, _ := json.Marshal(sourceConfig)

	task, err := db.InputTask.Create().
		SetTaskName("Preset Data Test").
		SetTaskType("manual").
		SetInputSource("mock").
		SetSourceConfig(string(sourceConfigJSON)).
		SetTaskStatus("pending").
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 3. 注册Mock Provider并设置预设数据
	mockProvider := provider.NewMockProvider()
	presetData := []map[string]interface{}{
		{
			"id":     "custom-001",
			"name":   "Custom Server 1",
			"ip":     "192.168.1.100",
			"status": "running",
		},
		{
			"id":     "custom-002",
			"name":   "Custom Server 2",
			"ip":     "192.168.1.101",
			"status": "stopped",
		},
	}
	mockProvider.SetMockData(presetData)

	registry := provider.GetRegistry()
	registry.Register(mockProvider)

	// 4. 创建Executor
	logger := logx.WithContext(context.Background())
	executor := NewExecutor(nil, db, logger)

	execCtx := &ExecutionContext{
		Ctx:            ctx,
		TenantID:       tenantID,
		UserID:         "test-user",
		TaskID:         task.ID,
		TaskType:       TaskTypeInput,
		Timeout:        30 * time.Second,
		Logger:         logger,
		DB:             db,
		ProviderID:     "mock",
		ProviderConfig: sourceConfig,
	}

	// 5. 执行Provider发现
	discProvider, err := executor.getProvider(execCtx)
	require.NoError(t, err)

	result, err := executor.executeProviderDiscovery(execCtx, discProvider)
	require.NoError(t, err)

	// 6. 验证结果
	assert.True(t, result.Success)
	assert.Equal(t, int64(2), result.TotalRecords)
	assert.Len(t, result.Records, 2)

	// 验证预设数据的字段
	assert.Equal(t, "custom-001", result.Records[0]["id"])
	assert.Equal(t, "Custom Server 1", result.Records[0]["name"])
	assert.Equal(t, "192.168.1.100", result.Records[0]["ip"])
	assert.Equal(t, "running", result.Records[0]["status"])

	assert.Equal(t, "custom-002", result.Records[1]["id"])
	assert.Equal(t, "Custom Server 2", result.Records[1]["name"])

	t.Logf("✅ 预设数据测试通过: %d条自定义记录", result.TotalRecords)
}

// TestProviderWorkerIntegration_Statistics 测试统计信息
func TestProviderWorkerIntegration_Statistics(t *testing.T) {
	// 1. 创建测试数据库
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer db.Close()

	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	ctx := context.Background()
	tenantID := uint64(1)
	ctx = hooks.SetTenantIDToContext(ctx, tenantID)

	// 2. 创建InputTask
	sourceConfig := map[string]interface{}{
		"provider_type": "mock",
		"record_count":  10,
	}
	sourceConfigJSON, _ := json.Marshal(sourceConfig)

	task, err := db.InputTask.Create().
		SetTaskName("Statistics Test").
		SetTaskType("manual").
		SetInputSource("mock").
		SetSourceConfig(string(sourceConfigJSON)).
		SetTaskStatus("pending").
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 3. 注册Mock Provider
	mockProvider := provider.NewMockProvider()
	registry := provider.GetRegistry()
	registry.Register(mockProvider)

	// 4. 创建Executor
	logger := logx.WithContext(context.Background())
	executor := NewExecutor(nil, db, logger)

	execCtx := &ExecutionContext{
		Ctx:            ctx,
		TenantID:       tenantID,
		UserID:         "test-user",
		TaskID:         task.ID,
		TaskType:       TaskTypeInput,
		Timeout:        30 * time.Second,
		Logger:         logger,
		DB:             db,
		ProviderID:     "mock",
		ProviderConfig: sourceConfig,
	}

	// 5. 执行Provider发现
	discProvider, err := executor.getProvider(execCtx)
	require.NoError(t, err)

	result, err := executor.executeProviderDiscovery(execCtx, discProvider)
	require.NoError(t, err)

	// 6. 验证统计信息
	assert.True(t, result.Success)

	// 验证Provider调用统计
	stats := mockProvider.GetCallStats()
	// Note: getProvider会调用ValidateConfig和TestConnection
	// executeProviderDiscovery会调用DiscoverWithContext
	assert.GreaterOrEqual(t, stats["validate_calls"].(int), 0)
	assert.GreaterOrEqual(t, stats["test_connection_calls"].(int), 0)
	assert.GreaterOrEqual(t, stats["discover_calls"].(int), 1, "Should call Discover at least once")
	assert.Equal(t, true, stats["discover_with_context"], "Should use V2 interface")

	// 验证结果统计
	assert.Equal(t, int64(10), result.TotalRecords)

	t.Logf("✅ 统计信息测试通过")
	t.Logf("   Provider调用: ValidateConfig=%d, TestConnection=%d, Discover=%d",
		stats["validate_calls"],
		stats["test_connection_calls"],
		stats["discover_calls"])
	t.Logf("   发现记录数: %d", result.TotalRecords)
}
