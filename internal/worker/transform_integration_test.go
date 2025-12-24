package worker

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/coder-lulu/newbee-common/v2/orm/ent/hooks"
	"github.com/coder-lulu/newbee-io-rpc/ent/enttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"

	_ "github.com/mattn/go-sqlite3"
)

// TestWorker_Transform_Integration 测试Worker与Transform Engine的完整集成
func TestWorker_Transform_Integration(t *testing.T) {
	// 1. 创建测试数据库
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer db.Close()

	// 2. 注册Hook（仅租户隔离）
	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	// 3. 创建测试数据
	ctx := context.Background()
	tenantID := uint64(1)
	ctx = hooks.SetTenantIDToContext(ctx, tenantID)

	// 创建InputTask
	sourceConfig := map[string]interface{}{
		"provider_type": "mock",
	}
	sourceConfigJSON, _ := json.Marshal(sourceConfig)

	task, err := db.InputTask.Create().
		SetTaskName("Test Transform Task").
		SetTaskType("manual").
		SetInputSource("mock_provider").
		SetSourceConfig(string(sourceConfigJSON)).
		SetTaskStatus("pending").
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)
	t.Logf("Created task: ID=%d", task.ID)

	// 4. 创建FieldMapping规则
	// 规则1: name字段直接映射
	mapping1, err := db.FieldMapping.Create().
		SetMappingName("Name Mapping").
		SetMappingType("input").
		SetSourceField("name").
		SetTargetField("full_name").
		SetTransformType("direct").
		SetIsActive(true).
		SetAllowNull(true).
		SetPriority(10).
		SetSortOrder(1).
		SetInputTaskID(task.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)
	t.Logf("Created mapping1: ID=%d", mapping1.ID)

	// 规则2: age字段类型转换（string -> int）
	mapping2, err := db.FieldMapping.Create().
		SetMappingName("Age Convert").
		SetMappingType("input").
		SetSourceField("age").
		SetTargetField("user_age").
		SetTransformType("convert").
		SetTargetDataType("int").
		SetIsActive(true).
		SetAllowNull(true).
		SetPriority(10).
		SetSortOrder(2).
		SetInputTaskID(task.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)
	t.Logf("Created mapping2: ID=%d", mapping2.ID)

	// 规则3: status字段查找表转换
	statusLookup := map[string]string{
		"active":   "激活",
		"inactive": "未激活",
	}
	statusLookupJSON, _ := json.Marshal(statusLookup)

	mapping3, err := db.FieldMapping.Create().
		SetMappingName("Status Lookup").
		SetMappingType("input").
		SetSourceField("status").
		SetTargetField("status_text").
		SetTransformType("direct").
		SetLookupTable(string(statusLookupJSON)).
		SetIsActive(true).
		SetAllowNull(true).
		SetPriority(10).
		SetSortOrder(3).
		SetInputTaskID(task.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)
	t.Logf("Created mapping3: ID=%d", mapping3.ID)

	// 5. 模拟Provider返回的记录
	mockRecords := []map[string]interface{}{
		{"name": "John Doe", "age": "30", "status": "active"},
		{"name": "Jane Smith", "age": "25", "status": "inactive"},
	}

	// 6. 创建Executor并测试applyTransform
	logger := logx.WithContext(context.Background())
	executor := NewExecutor(nil, db, logger)

	execCtx := &ExecutionContext{
		Ctx:      ctx,
		TenantID: tenantID,
		UserID:   "test-user",
		TaskID:   task.ID,
		TaskType: TaskTypeInput,
		Timeout:  30 * time.Second,
		Logger:   logger,
		DB:       db,
	}

	// 7. 执行Transform
	transformedRecords, transformStats, err := executor.applyTransform(execCtx, mockRecords)
	require.NoError(t, err)

	// 8. 验证结果
	assert.Len(t, transformedRecords, 2, "Should return 2 transformed records")

	// 9. 验证第一条记录的Transform结果
	record1 := transformedRecords[0]
	assert.Equal(t, "John Doe", record1["full_name"], "Name should be mapped correctly")
	assert.Equal(t, int64(30), record1["user_age"], "Age should be converted to int")
	assert.Equal(t, "激活", record1["status_text"], "Status should be translated via lookup table")

	// 10. 验证第二条记录的Transform结果
	record2 := transformedRecords[1]
	assert.Equal(t, "Jane Smith", record2["full_name"])
	assert.Equal(t, int64(25), record2["user_age"])
	assert.Equal(t, "未激活", record2["status_text"])

	// 11. 验证Transform统计信息
	assert.Equal(t, 6, transformStats.TotalFields, "Should process 6 fields (3 mappings × 2 records)")
	assert.Equal(t, 6, transformStats.SuccessFields, "All fields should succeed")
	assert.Equal(t, 0, transformStats.FailedFields, "No fields should fail")
	assert.Equal(t, 0, transformStats.SkippedFields, "No fields should be skipped")

	t.Logf("✅ Transform statistics: total=%d, success=%d, failed=%d, skipped=%d",
		transformStats.TotalFields,
		transformStats.SuccessFields,
		transformStats.FailedFields,
		transformStats.SkippedFields)
}

// TestWorker_Transform_WithValidation 测试带验证的Transform
func TestWorker_Transform_WithValidation(t *testing.T) {
	// 1. 创建测试数据库
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer db.Close()

	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	ctx := context.Background()
	tenantID := uint64(1)
	ctx = hooks.SetTenantIDToContext(ctx, tenantID)

	// 2. 创建InputTask
	sourceConfig := map[string]interface{}{"provider_type": "mock"}
	sourceConfigJSON, _ := json.Marshal(sourceConfig)

	task, err := db.InputTask.Create().
		SetTaskName("Validation Test Task").
		SetTaskType("manual").
		SetInputSource("mock_provider").
		SetSourceConfig(string(sourceConfigJSON)).
		SetTaskStatus("pending").
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 3. 创建带验证规则的FieldMapping
	validationRules := `[
		{"type": "required", "message": "Email is required"},
		{"type": "email", "message": "Invalid email format"}
	]`

	_, err = db.FieldMapping.Create().
		SetMappingName("Email with Validation").
		SetMappingType("input").
		SetSourceField("email").
		SetTargetField("user_email").
		SetTransformType("direct").
		SetIsRequired(true).
		SetAllowNull(false).
		SetValidationRules(validationRules).
		SetIsActive(true).
		SetPriority(10).
		SetInputTaskID(task.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 4. 准备测试记录
	mockRecords := []map[string]interface{}{
		{"email": "john@example.com"},      // 有效邮箱
		{"email": "invalid-email"},         // 无效邮箱
		{"name": "No Email"},               // 缺少email字段
	}

	// 5. 执行Transform
	logger := logx.WithContext(context.Background())
	executor := NewExecutor(nil, db, logger)

	execCtx := &ExecutionContext{
		Ctx:      ctx,
		TenantID: tenantID,
		UserID:   "test-user",
		TaskID:   task.ID,
		TaskType: TaskTypeInput,
		Timeout:  30 * time.Second,
		Logger:   logger,
		DB:       db,
	}

	transformedRecords, transformStats, err := executor.applyTransform(execCtx, mockRecords)
	require.NoError(t, err)

	// 6. 验证结果
	// 由于采用宽松模式，失败的记录会被跳过
	assert.Len(t, transformedRecords, 1, "Only 1 record should pass validation")

	// 验证通过的记录
	record := transformedRecords[0]
	assert.Equal(t, "john@example.com", record["user_email"])

	// 验证统计信息
	assert.Equal(t, 2, transformStats.FailedFields, "2 records should fail validation")

	t.Logf("✅ Validation test: %d success, %d failed out of %d total",
		len(transformedRecords), transformStats.FailedFields, len(mockRecords))
}

// TestWorker_Transform_NoMappings 测试没有映射规则时的行为
func TestWorker_Transform_NoMappings(t *testing.T) {
	// 1. 创建测试数据库
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer db.Close()

	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	ctx := context.Background()
	tenantID := uint64(1)
	ctx = hooks.SetTenantIDToContext(ctx, tenantID)

	// 2. 创建InputTask（没有关联任何FieldMapping）
	sourceConfig := map[string]interface{}{"provider_type": "mock"}
	sourceConfigJSON, _ := json.Marshal(sourceConfig)

	task, err := db.InputTask.Create().
		SetTaskName("No Mappings Task").
		SetTaskType("manual").
		SetInputSource("mock_provider").
		SetSourceConfig(string(sourceConfigJSON)).
		SetTaskStatus("pending").
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 3. 准备测试记录
	mockRecords := []map[string]interface{}{
		{"name": "John", "age": 30},
	}

	// 4. 执行Transform
	logger := logx.WithContext(context.Background())
	executor := NewExecutor(nil, db, logger)

	execCtx := &ExecutionContext{
		Ctx:      ctx,
		TenantID: tenantID,
		UserID:   "test-user",
		TaskID:   task.ID,
		TaskType: TaskTypeInput,
		Timeout:  30 * time.Second,
		Logger:   logger,
		DB:       db,
	}

	transformedRecords, transformStats, err := executor.applyTransform(execCtx, mockRecords)
	require.NoError(t, err)

	// 5. 验证：没有映射规则时应返回原始数据
	assert.Len(t, transformedRecords, 1)

	// 数据应该保持原样
	record := transformedRecords[0]
	assert.Equal(t, "John", record["name"])
	assert.Equal(t, 30, record["age"])

	// 统计信息应该为空
	assert.Equal(t, 0, transformStats.TotalFields)

	t.Log("✅ No mappings test passed: original data returned unchanged")
}

// TestWorker_Transform_NestedPath 测试嵌套路径映射
func TestWorker_Transform_NestedPath(t *testing.T) {
	// 1. 创建测试数据库
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer db.Close()

	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	ctx := context.Background()
	tenantID := uint64(1)
	ctx = hooks.SetTenantIDToContext(ctx, tenantID)

	// 2. 创建InputTask
	task, err := db.InputTask.Create().
		SetTaskName("Nested Path Test").
		SetTaskType("manual").
		SetInputSource("mock").
		SetSourceConfig("{}").
		SetTaskStatus("pending").
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 3. 创建嵌套路径映射
	_, err = db.FieldMapping.Create().
		SetMappingName("Nested Name Mapping").
		SetMappingType("input").
		SetSourceField("user.profile.name").
		SetSourceFieldPath("user.profile.name").
		SetTargetField("name").
		SetTransformType("direct").
		SetIsActive(true).
		SetAllowNull(true).
		SetPriority(10).
		SetInputTaskID(task.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 4. 准备嵌套数据
	mockRecords := []map[string]interface{}{
		{
			"user": map[string]interface{}{
				"profile": map[string]interface{}{
					"name": "Alice",
					"age":  28,
				},
			},
		},
	}

	// 5. 执行Transform
	logger := logx.WithContext(context.Background())
	executor := NewExecutor(nil, db, logger)

	execCtx := &ExecutionContext{
		Ctx:      ctx,
		TenantID: tenantID,
		UserID:   "test-user",
		TaskID:   task.ID,
		TaskType: TaskTypeInput,
		Timeout:  30 * time.Second,
		Logger:   logger,
		DB:       db,
	}

	transformedRecords, _, err := executor.applyTransform(execCtx, mockRecords)
	require.NoError(t, err)

	// 6. 验证嵌套路径提取成功
	assert.Len(t, transformedRecords, 1)
	record := transformedRecords[0]
	assert.Equal(t, "Alice", record["name"], "Should extract nested value correctly")

	t.Log("✅ Nested path test passed")
}
