package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/coder-lulu/newbee-common/v2/orm/ent/hooks"
	"github.com/coder-lulu/newbee-io-rpc/ent/enttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"

	_ "github.com/mattn/go-sqlite3"
)

// TestE2E_InputTask_Full 测试InputTask完整流程
// 流程: HTTP Provider → Transform → 获取结果数据
func TestE2E_InputTask_Full(t *testing.T) {
	// 1. 创建测试数据库
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer db.Close()

	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	ctx := context.Background()
	tenantID := uint64(1)
	ctx = hooks.SetTenantIDToContext(ctx, tenantID)

	// 2. 创建InputTask
	providerConfig := map[string]interface{}{
		"provider_type": "mock",
		"mock_data": []map[string]interface{}{
			{"user_id": "1001", "user_name": "Alice", "age": "28", "status": "active"},
			{"user_id": "1002", "user_name": "Bob", "age": "35", "status": "inactive"},
		},
	}
	providerConfigJSON, _ := json.Marshal(providerConfig)

	task, err := db.InputTask.Create().
		SetTaskName("E2E Test - User Data Import").
		SetTaskType("manual").
		SetInputSource("mock_http_api").
		SetSourceConfig(string(providerConfigJSON)).
		SetTaskStatus("pending").
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)
	t.Logf("✅ Created InputTask: ID=%d", task.ID)

	// 3. 创建FieldMapping规则
	// 规则1: user_id → id (直接映射)
	mapping1, err := db.FieldMapping.Create().
		SetMappingName("User ID Mapping").
		SetMappingType("input").
		SetSourceField("user_id").
		SetTargetField("id").
		SetTransformType("direct").
		SetIsActive(true).
		SetAllowNull(false).
		SetIsRequired(true).
		SetPriority(10).
		SetSortOrder(1).
		SetInputTaskID(task.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 规则2: user_name → name (直接映射)
	mapping2, err := db.FieldMapping.Create().
		SetMappingName("User Name Mapping").
		SetMappingType("input").
		SetSourceField("user_name").
		SetTargetField("name").
		SetTransformType("direct").
		SetIsActive(true).
		SetAllowNull(false).
		SetIsRequired(true).
		SetPriority(10).
		SetSortOrder(2).
		SetInputTaskID(task.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 规则3: age → age_int (类型转换 string → int)
	mapping3, err := db.FieldMapping.Create().
		SetMappingName("Age Convert").
		SetMappingType("input").
		SetSourceField("age").
		SetTargetField("age_int").
		SetTransformType("convert").
		SetTargetDataType("int").
		SetIsActive(true).
		SetAllowNull(false).
		SetPriority(10).
		SetSortOrder(3).
		SetInputTaskID(task.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 规则4: status → status_text (查找表转换)
	statusLookup := map[string]string{
		"active":   "激活",
		"inactive": "未激活",
	}
	statusLookupJSON, _ := json.Marshal(statusLookup)
	mapping4, err := db.FieldMapping.Create().
		SetMappingName("Status Lookup").
		SetMappingType("input").
		SetSourceField("status").
		SetTargetField("status_text").
		SetTransformType("direct").
		SetLookupTable(string(statusLookupJSON)).
		SetIsActive(true).
		SetAllowNull(true).
		SetPriority(10).
		SetSortOrder(4).
		SetInputTaskID(task.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	t.Logf("✅ Created 4 FieldMappings: [%d, %d, %d, %d]",
		mapping1.ID, mapping2.ID, mapping3.ID, mapping4.ID)

	// 4. 准备模拟数据（模拟Provider返回）
	mockRecords := providerConfig["mock_data"].([]map[string]interface{})

	// 5. 创建Executor并执行applyTransform
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

	// 6. 执行Transform
	transformedRecords, transformStats, err := executor.applyTransform(execCtx, mockRecords)
	require.NoError(t, err)

	// 7. 验证结果
	assert.Len(t, transformedRecords, 2, "Should have 2 records")

	// 验证第一条记录
	record1 := transformedRecords[0]
	assert.Equal(t, "1001", record1["id"])
	assert.Equal(t, "Alice", record1["name"])
	assert.Equal(t, int64(28), record1["age_int"])
	assert.Equal(t, "激活", record1["status_text"])

	// 验证第二条记录
	record2 := transformedRecords[1]
	assert.Equal(t, "1002", record2["id"])
	assert.Equal(t, "Bob", record2["name"])
	assert.Equal(t, int64(35), record2["age_int"])
	assert.Equal(t, "未激活", record2["status_text"])

	// 验证统计信息
	assert.Equal(t, 8, transformStats.TotalFields) // 4 mappings × 2 records
	assert.Equal(t, 8, transformStats.SuccessFields)
	assert.Equal(t, 0, transformStats.FailedFields)

	t.Logf("✅ InputTask E2E Test Passed!")
	t.Logf("   📊 Transform Stats: total=%d, success=%d, failed=%d",
		transformStats.TotalFields,
		transformStats.SuccessFields,
		transformStats.FailedFields)
	t.Logf("   📦 Sample Data: %+v", record1)
}

// TestE2E_OutputTask_Full 测试OutputTask完整流程
// 流程: 准备数据 → Transform → 模拟写入Target
func TestE2E_OutputTask_Full(t *testing.T) {
	// 1. 创建测试数据库
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer db.Close()

	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	ctx := context.Background()
	tenantID := uint64(1)
	ctx = hooks.SetTenantIDToContext(ctx, tenantID)

	// 2. 创建DataTarget
	targetConfig := map[string]interface{}{
		"target_type": "mock_database",
		"table_name":  "users",
	}
	targetConfigJSON, _ := json.Marshal(targetConfig)

	dataTarget, err := db.DataTarget.Create().
		SetTargetName("Mock Database Target").
		SetTargetCode("mock_db_001").
		SetTargetType("database").
		SetTargetSystem("PostgreSQL").
		SetConnectionConfig(string(targetConfigJSON)).
		SetIsActive(true).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)
	t.Logf("✅ Created DataTarget: ID=%d", dataTarget.ID)

	// 3. 创建OutputTask
	outputTaskConfig := map[string]interface{}{
		"target_id": dataTarget.ID,
		"batch_size": 100,
	}
	outputTaskConfigJSON, _ := json.Marshal(outputTaskConfig)

	outputTask, err := db.OutputTask.Create().
		SetTaskName("E2E Test - User Data Export").
		SetTaskType("manual").
		SetOutputTarget("database").
		SetTargetConfig(string(outputTaskConfigJSON)).
		SetTaskStatus("pending").
		SetDataTargetID(dataTarget.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)
	t.Logf("✅ Created OutputTask: ID=%d", outputTask.ID)

	// 4. 创建FieldMapping规则（OutputTask端的二次转换）
	// 规则1: id → user_id (字段重命名)
	mapping1, err := db.FieldMapping.Create().
		SetMappingName("ID to User ID").
		SetMappingType("output").
		SetSourceField("id").
		SetTargetField("user_id").
		SetTransformType("direct").
		SetIsActive(true).
		SetAllowNull(false).
		SetPriority(10).
		SetSortOrder(1).
		SetOutputTaskID(outputTask.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 规则2: name → user_name
	mapping2, err := db.FieldMapping.Create().
		SetMappingName("Name to User Name").
		SetMappingType("output").
		SetSourceField("name").
		SetTargetField("user_name").
		SetTransformType("direct").
		SetIsActive(true).
		SetAllowNull(false).
		SetPriority(10).
		SetSortOrder(2).
		SetOutputTaskID(outputTask.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 规则3: age_int → age_str (类型转换 int → string)
	mapping3, err := db.FieldMapping.Create().
		SetMappingName("Age to String").
		SetMappingType("output").
		SetSourceField("age_int").
		SetTargetField("age_str").
		SetTransformType("convert").
		SetTargetDataType("string").
		SetIsActive(true).
		SetAllowNull(false).
		SetPriority(10).
		SetSortOrder(3).
		SetOutputTaskID(outputTask.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	t.Logf("✅ Created 3 FieldMappings for OutputTask: [%d, %d, %d]",
		mapping1.ID, mapping2.ID, mapping3.ID)

	// 5. 准备输入数据（模拟从InputTask获取的已转换数据）
	inputRecords := []map[string]interface{}{
		{"id": "1001", "name": "Alice", "age_int": int64(28), "status_text": "激活"},
		{"id": "1002", "name": "Bob", "age_int": int64(35), "status_text": "未激活"},
	}

	// 6. 创建Executor并执行applyTransform
	logger := logx.WithContext(context.Background())
	executor := NewExecutor(nil, db, logger)

	execCtx := &ExecutionContext{
		Ctx:      ctx,
		TenantID: tenantID,
		UserID:   "test-user",
		TaskID:   outputTask.ID,
		TaskType: TaskTypeOutput,
		Timeout:  30 * time.Second,
		Logger:   logger,
		DB:       db,
	}

	// 7. 执行Transform
	transformedRecords, transformStats, err := executor.applyTransform(execCtx, inputRecords)
	require.NoError(t, err)

	// 8. 验证结果
	assert.Len(t, transformedRecords, 2, "Should have 2 records")

	// 验证第一条记录
	record1 := transformedRecords[0]
	assert.Equal(t, "1001", record1["user_id"])
	assert.Equal(t, "Alice", record1["user_name"])
	assert.Equal(t, "28", record1["age_str"]) // int → string

	// 验证第二条记录
	record2 := transformedRecords[1]
	assert.Equal(t, "1002", record2["user_id"])
	assert.Equal(t, "Bob", record2["user_name"])
	assert.Equal(t, "35", record2["age_str"])

	// 验证统计信息
	assert.Equal(t, 6, transformStats.TotalFields) // 3 mappings × 2 records
	assert.Equal(t, 6, transformStats.SuccessFields)
	assert.Equal(t, 0, transformStats.FailedFields)

	t.Logf("✅ OutputTask E2E Test Passed!")
	t.Logf("   📊 Transform Stats: total=%d, success=%d, failed=%d",
		transformStats.TotalFields,
		transformStats.SuccessFields,
		transformStats.FailedFields)
	t.Logf("   📦 Sample Data: %+v", record1)
}

// TestE2E_CompleteFlow 测试完整的端到端流程
// 流程: InputTask → OutputTask 串联执行
func TestE2E_CompleteFlow(t *testing.T) {
	// 1. 创建测试数据库
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer db.Close()

	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	ctx := context.Background()
	tenantID := uint64(1)
	ctx = hooks.SetTenantIDToContext(ctx, tenantID)

	logger := logx.WithContext(context.Background())
	executor := NewExecutor(nil, db, logger)

	// 🎯 阶段1: InputTask - 数据导入
	t.Log("📥 Phase 1: InputTask - Data Import")

	// 创建InputTask
	inputProviderConfig := map[string]interface{}{
		"provider_type": "mock",
	}
	inputProviderConfigJSON, _ := json.Marshal(inputProviderConfig)

	inputTask, err := db.InputTask.Create().
		SetTaskName("Complete E2E - Input").
		SetTaskType("manual").
		SetInputSource("mock_http_api").
		SetSourceConfig(string(inputProviderConfigJSON)).
		SetTaskStatus("pending").
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 创建InputTask的FieldMapping
	_, err = db.FieldMapping.Create().
		SetMappingName("Input: ID Mapping").
		SetMappingType("input").
		SetSourceField("raw_id").
		SetTargetField("id").
		SetTransformType("direct").
		SetIsActive(true).
		SetAllowNull(false).
		SetPriority(10).
		SetSortOrder(1).
		SetInputTaskID(inputTask.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	_, err = db.FieldMapping.Create().
		SetMappingName("Input: Name Mapping").
		SetMappingType("input").
		SetSourceField("raw_name").
		SetTargetField("name").
		SetTransformType("direct").
		SetIsActive(true).
		SetAllowNull(false).
		SetPriority(10).
		SetSortOrder(2).
		SetInputTaskID(inputTask.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 模拟InputTask执行
	inputMockRecords := []map[string]interface{}{
		{"raw_id": "A001", "raw_name": "Product A"},
		{"raw_id": "A002", "raw_name": "Product B"},
		{"raw_id": "A003", "raw_name": "Product C"},
	}

	inputExecCtx := &ExecutionContext{
		Ctx:      ctx,
		TenantID: tenantID,
		UserID:   "test-user",
		TaskID:   inputTask.ID,
		TaskType: TaskTypeInput,
		Timeout:  30 * time.Second,
		Logger:   logger,
		DB:       db,
	}

	inputResult, inputStats, err := executor.applyTransform(inputExecCtx, inputMockRecords)
	require.NoError(t, err)
	require.Len(t, inputResult, 3)

	t.Logf("   ✅ InputTask completed: %d records transformed", len(inputResult))
	t.Logf("   📊 Input Stats: success=%d, failed=%d", inputStats.SuccessFields, inputStats.FailedFields)

	// 🎯 阶段2: OutputTask - 数据导出
	t.Log("📤 Phase 2: OutputTask - Data Export")

	// 创建DataTarget
	dataTarget, err := db.DataTarget.Create().
		SetTargetName("Complete E2E Target").
		SetTargetCode("e2e_target_001").
		SetTargetType("database").
		SetTargetSystem("PostgreSQL").
		SetConnectionConfig("{}").
		SetIsActive(true).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 创建OutputTask
	outputTask, err := db.OutputTask.Create().
		SetTaskName("Complete E2E - Output").
		SetTaskType("manual").
		SetOutputTarget("database").
		SetTargetConfig("{}").
		SetTaskStatus("pending").
		SetDataTargetID(dataTarget.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 创建OutputTask的FieldMapping（二次转换）
	_, err = db.FieldMapping.Create().
		SetMappingName("Output: ID Prefix").
		SetMappingType("output").
		SetSourceField("id").
		SetTargetField("product_id").
		SetTransformType("direct").
		SetIsActive(true).
		SetAllowNull(false).
		SetPriority(10).
		SetSortOrder(1).
		SetOutputTaskID(outputTask.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	_, err = db.FieldMapping.Create().
		SetMappingName("Output: Name Upper").
		SetMappingType("output").
		SetSourceField("name").
		SetTargetField("product_name").
		SetTransformType("direct").
		SetIsActive(true).
		SetAllowNull(false).
		SetPriority(10).
		SetSortOrder(2).
		SetOutputTaskID(outputTask.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 使用InputTask的输出作为OutputTask的输入
	outputExecCtx := &ExecutionContext{
		Ctx:      ctx,
		TenantID: tenantID,
		UserID:   "test-user",
		TaskID:   outputTask.ID,
		TaskType: TaskTypeOutput,
		Timeout:  30 * time.Second,
		Logger:   logger,
		DB:       db,
	}

	outputResult, outputStats, err := executor.applyTransform(outputExecCtx, inputResult)
	require.NoError(t, err)
	require.Len(t, outputResult, 3)

	t.Logf("   ✅ OutputTask completed: %d records transformed", len(outputResult))
	t.Logf("   📊 Output Stats: success=%d, failed=%d", outputStats.SuccessFields, outputStats.FailedFields)

	// 🎯 验证完整流程
	t.Log("✅ Complete E2E Flow Verification")

	// 验证数据转换正确性
	assert.Equal(t, "A001", outputResult[0]["product_id"])
	assert.Equal(t, "Product A", outputResult[0]["product_name"])

	assert.Equal(t, "A002", outputResult[1]["product_id"])
	assert.Equal(t, "Product B", outputResult[1]["product_name"])

	assert.Equal(t, "A003", outputResult[2]["product_id"])
	assert.Equal(t, "Product C", outputResult[2]["product_name"])

	t.Logf("   ✅ Data flow verified: InputTask → OutputTask")
	t.Logf("   📊 Total pipeline: %d input fields → %d output fields",
		inputStats.SuccessFields, outputStats.SuccessFields)
}

// TestE2E_MultiTenant_Isolation 测试多租户隔离
func TestE2E_MultiTenant_Isolation(t *testing.T) {
	// 1. 创建测试数据库
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer db.Close()

	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	logger := logx.WithContext(context.Background())
	executor := NewExecutor(nil, db, logger)

	// 🎯 租户A的任务
	t.Log("👤 Tenant A: Creating Task")

	ctxA := context.Background()
	tenantA := uint64(1)
	ctxA = hooks.SetTenantIDToContext(ctxA, tenantA)

	taskA, err := db.InputTask.Create().
		SetTaskName("Tenant A Task").
		SetTaskType("manual").
		SetInputSource("mock").
		SetSourceConfig("{}").
		SetTaskStatus("pending").
		SetStatus(1).
		Save(ctxA)
	require.NoError(t, err)

	_, err = db.FieldMapping.Create().
		SetMappingName("Tenant A Mapping").
		SetMappingType("input").
		SetSourceField("field_a").
		SetTargetField("target_a").
		SetTransformType("direct").
		SetIsActive(true).
		SetAllowNull(true).
		SetPriority(10).
		SetInputTaskID(taskA.ID).
		SetStatus(1).
		Save(ctxA)
	require.NoError(t, err)

	// 🎯 租户B的任务
	t.Log("👤 Tenant B: Creating Task")

	ctxB := context.Background()
	tenantB := uint64(2)
	ctxB = hooks.SetTenantIDToContext(ctxB, tenantB)

	taskB, err := db.InputTask.Create().
		SetTaskName("Tenant B Task").
		SetTaskType("manual").
		SetInputSource("mock").
		SetSourceConfig("{}").
		SetTaskStatus("pending").
		SetStatus(1).
		Save(ctxB)
	require.NoError(t, err)

	_, err = db.FieldMapping.Create().
		SetMappingName("Tenant B Mapping").
		SetMappingType("input").
		SetSourceField("field_b").
		SetTargetField("target_b").
		SetTransformType("direct").
		SetIsActive(true).
		SetAllowNull(true).
		SetPriority(10).
		SetInputTaskID(taskB.ID).
		SetStatus(1).
		Save(ctxB)
	require.NoError(t, err)

	// 🎯 验证租户隔离
	t.Log("🔒 Verifying Tenant Isolation")

	// 租户A执行：应该只看到租户A的FieldMapping
	mockDataA := []map[string]interface{}{
		{"field_a": "value_a", "field_b": "should_not_map"},
	}

	execCtxA := &ExecutionContext{
		Ctx:      ctxA,
		TenantID: tenantA,
		UserID:   "user-a",
		TaskID:   taskA.ID,
		TaskType: TaskTypeInput,
		Timeout:  30 * time.Second,
		Logger:   logger,
		DB:       db,
	}

	resultA, statsA, err := executor.applyTransform(execCtxA, mockDataA)
	require.NoError(t, err)

	// 验证：租户A只映射了field_a → target_a
	assert.Len(t, resultA, 1)
	assert.Equal(t, "value_a", resultA[0]["target_a"])
	assert.NotContains(t, resultA[0], "target_b") // 租户B的映射不应该被应用

	t.Logf("   ✅ Tenant A: %d mappings applied, %d fields transformed",
		statsA.TotalFields, statsA.SuccessFields)

	// 租户B执行：应该只看到租户B的FieldMapping
	mockDataB := []map[string]interface{}{
		{"field_a": "should_not_map", "field_b": "value_b"},
	}

	execCtxB := &ExecutionContext{
		Ctx:      ctxB,
		TenantID: tenantB,
		UserID:   "user-b",
		TaskID:   taskB.ID,
		TaskType: TaskTypeInput,
		Timeout:  30 * time.Second,
		Logger:   logger,
		DB:       db,
	}

	resultB, statsB, err := executor.applyTransform(execCtxB, mockDataB)
	require.NoError(t, err)

	// 验证：租户B只映射了field_b → target_b
	assert.Len(t, resultB, 1)
	assert.Equal(t, "value_b", resultB[0]["target_b"])
	assert.NotContains(t, resultB[0], "target_a") // 租户A的映射不应该被应用

	t.Logf("   ✅ Tenant B: %d mappings applied, %d fields transformed",
		statsB.TotalFields, statsB.SuccessFields)

	t.Log("✅ Multi-Tenant Isolation Verified!")
}

// TestE2E_ErrorRecovery 测试错误恢复
func TestE2E_ErrorRecovery(t *testing.T) {
	// 1. 创建测试数据库
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer db.Close()

	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	ctx := context.Background()
	tenantID := uint64(1)
	ctx = hooks.SetTenantIDToContext(ctx, tenantID)

	logger := logx.WithContext(context.Background())
	executor := NewExecutor(nil, db, logger)

	// 2. 创建InputTask
	task, err := db.InputTask.Create().
		SetTaskName("Error Recovery Test").
		SetTaskType("manual").
		SetInputSource("mock").
		SetSourceConfig("{}").
		SetTaskStatus("pending").
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 3. 创建会导致错误的FieldMapping
	// 必填字段验证
	validationRules := `[
		{"type": "required", "message": "Field is required"},
		{"type": "regex", "params": "^[A-Z]{2,5}$", "message": "Must be 2-5 uppercase letters"}
	]`

	_, err = db.FieldMapping.Create().
		SetMappingName("Strict Validation").
		SetMappingType("input").
		SetSourceField("code").
		SetTargetField("valid_code").
		SetTransformType("direct").
		SetValidationRules(validationRules).
		SetIsRequired(true).
		SetAllowNull(false).
		SetIsActive(true).
		SetPriority(10).
		SetInputTaskID(task.ID).
		SetStatus(1).
		Save(ctx)
	require.NoError(t, err)

	// 4. 准备混合数据（部分成功，部分失败）
	mockRecords := []map[string]interface{}{
		{"code": "ABC"},        // ✅ 有效
		{"code": "XYZ"},        // ✅ 有效
		{"code": "invalid"},    // ❌ 不符合regex
		{"other": "no_code"},   // ❌ 缺少必填字段
		{"code": "VALID"},      // ✅ 有效
	}

	// 5. 执行Transform
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

	// 6. 验证错误恢复
	// 应该有3条成功记录（ABC, XYZ, VALID）
	// 2条失败记录被跳过（invalid, no_code）
	assert.Len(t, transformedRecords, 3, "Should have 3 valid records")
	assert.Equal(t, 2, transformStats.FailedFields, "Should have 2 failed fields")
	assert.Equal(t, 3, transformStats.SuccessFields, "Should have 3 successful fields")

	// 验证成功记录的内容
	assert.Equal(t, "ABC", transformedRecords[0]["valid_code"])
	assert.Equal(t, "XYZ", transformedRecords[1]["valid_code"])
	assert.Equal(t, "VALID", transformedRecords[2]["valid_code"])

	t.Logf("✅ Error Recovery Test Passed!")
	t.Logf("   📊 Stats: total=%d, success=%d, failed=%d",
		transformStats.TotalFields,
		transformStats.SuccessFields,
		transformStats.FailedFields)
	t.Logf("   🛡️ System continued processing despite errors (lenient mode)")
}

// BenchmarkE2E_Transform_Performance 性能基准测试
func BenchmarkE2E_Transform_Performance(b *testing.B) {
	// 1. 创建测试数据库
	db := enttest.Open(b, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer db.Close()

	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	ctx := context.Background()
	tenantID := uint64(1)
	ctx = hooks.SetTenantIDToContext(ctx, tenantID)

	logger := logx.WithContext(context.Background())
	executor := NewExecutor(nil, db, logger)

	// 2. 创建InputTask
	task, _ := db.InputTask.Create().
		SetTaskName("Performance Benchmark").
		SetTaskType("manual").
		SetInputSource("mock").
		SetSourceConfig("{}").
		SetTaskStatus("pending").
		SetStatus(1).
		Save(ctx)

	// 3. 创建10个FieldMapping规则
	for i := 0; i < 10; i++ {
		_, _ = db.FieldMapping.Create().
			SetMappingName(fmt.Sprintf("Mapping %d", i+1)).
			SetMappingType("input").
			SetSourceField(fmt.Sprintf("field_%d", i)).
			SetTargetField(fmt.Sprintf("target_%d", i)).
			SetTransformType("direct").
			SetIsActive(true).
			SetAllowNull(true).
			SetPriority(10).
			SetSortOrder(i).
			SetInputTaskID(task.ID).
			SetStatus(1).
			Save(ctx)
	}

	// 4. 准备100条记录
	mockRecords := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		record := make(map[string]interface{})
		for j := 0; j < 10; j++ {
			record[fmt.Sprintf("field_%d", j)] = fmt.Sprintf("value_%d_%d", i, j)
		}
		mockRecords[i] = record
	}

	execCtx := &ExecutionContext{
		Ctx:      ctx,
		TenantID: tenantID,
		UserID:   "bench-user",
		TaskID:   task.ID,
		TaskType: TaskTypeInput,
		Timeout:  30 * time.Second,
		Logger:   logger,
		DB:       db,
	}

	// 5. 运行基准测试
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = executor.applyTransform(execCtx, mockRecords)
	}
}
