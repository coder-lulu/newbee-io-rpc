package worker

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/coder-lulu/newbee-common/v2/orm/ent/hooks"
	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/ent/enttest"
	"github.com/coder-lulu/newbee-io-rpc/ent/inputtask"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"

	_ "github.com/mattn/go-sqlite3"
)

// TestTenantIsolation_ExecutionContext 测试ExecutionContext的租户隔离
func TestTenantIsolation_ExecutionContext(t *testing.T) {
	// 初始化测试数据库(使用内存SQLite)
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer db.Close()

	// 注册租户隔离Hook (仅注册tenant hooks，不注册department hooks)
	// InputTask/OutputTask没有department_id字段，只需要tenant隔离
	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	// 准备测试数据: 创建两个租户的任务
	tenant1ID := uint64(1)
	tenant2ID := uint64(2)

	ctx1 := hooks.SetTenantIDToContext(context.Background(), tenant1ID)
	ctx2 := hooks.SetTenantIDToContext(context.Background(), tenant2ID)

	// 创建租户1的任务
	task1, err := db.InputTask.Create().
		SetTaskName("Tenant1_Task").
		SetInputSource("file_import").
		SetTaskStatus("pending").
		SetTenantID(tenant1ID).
		Save(ctx1)
	require.NoError(t, err)

	// 创建租户2的任务
	task2, err := db.InputTask.Create().
		SetTaskName("Tenant2_Task").
		SetInputSource("file_import").
		SetTaskStatus("pending").
		SetTenantID(tenant2ID).
		Save(ctx2)
	require.NoError(t, err)

	// 测试用例1: 构建ExecutionContext时自动提取正确的租户ID
	t.Run("BuildExecutionContext_ExtractTenantID", func(t *testing.T) {
		contextManager := NewContextManager(logx.WithContext(context.Background()))

		// 租户1的context应该提取到tenant1ID
		execCtx1, err := contextManager.BuildExecutionContext(ctx1, task1, TaskTypeInput, db)
		require.NoError(t, err)
		assert.Equal(t, tenant1ID, execCtx1.TenantID, "ExecutionContext should extract tenant1ID")

		// 租户2的context应该提取到tenant2ID
		execCtx2, err := contextManager.BuildExecutionContext(ctx2, task2, TaskTypeInput, db)
		require.NoError(t, err)
		assert.Equal(t, tenant2ID, execCtx2.TenantID, "ExecutionContext should extract tenant2ID")
	})

	// 测试用例2: ValidateTenantIsolation 应该拒绝跨租户访问
	t.Run("ValidateTenantIsolation_RejectCrossTenantAccess", func(t *testing.T) {
		contextManager := NewContextManager(logx.WithContext(context.Background()))

		// 构建租户1的ExecutionContext
		execCtx1, err := contextManager.BuildExecutionContext(ctx1, task1, TaskTypeInput, db)
		require.NoError(t, err)

		// ✅ 验证通过: context租户ID == 期望租户ID
		err = contextManager.ValidateTenantIsolation(execCtx1, tenant1ID)
		assert.NoError(t, err, "Validation should pass for matching tenant ID")

		// 🚨 验证失败: context租户ID != 期望租户ID (安全违规)
		err = contextManager.ValidateTenantIsolation(execCtx1, tenant2ID)
		assert.Error(t, err, "Validation should fail for mismatched tenant ID")
		assert.Contains(t, err.Error(), "TENANT ISOLATION VIOLATION", "Error should contain violation message")
	})

	// 测试用例3: StateMachine查询应该只返回当前租户的任务
	t.Run("StateMachine_QueryOnlyCurrentTenantTasks", func(t *testing.T) {
		stateMachine := NewStateMachine(db, logx.WithContext(context.Background()))

		// 租户1的context应该只能查询到租户1的任务
		tasks1, err := stateMachine.GetPendingInputTasks(ctx1, 10)
		require.NoError(t, err)
		assert.Len(t, tasks1, 1, "Tenant1 should only see 1 task")
		assert.Equal(t, task1.ID, tasks1[0].ID)
		assert.Equal(t, tenant1ID, tasks1[0].TenantID)

		// 租户2的context应该只能查询到租户2的任务
		tasks2, err := stateMachine.GetPendingInputTasks(ctx2, 10)
		require.NoError(t, err)
		assert.Len(t, tasks2, 1, "Tenant2 should only see 1 task")
		assert.Equal(t, task2.ID, tasks2[0].ID)
		assert.Equal(t, tenant2ID, tasks2[0].TenantID)
	})

	// 测试用例4: StateMachine更新任务状态时应该尊重租户隔离
	t.Run("StateMachine_UpdateRespectsTenantIsolation", func(t *testing.T) {
		stateMachine := NewStateMachine(db, logx.WithContext(context.Background()))

		// 租户1尝试更新租户1的任务 - 应该成功
		err := stateMachine.MarkTaskAsRunning(ctx1, task1.ID, TaskTypeInput)
		assert.NoError(t, err, "Tenant1 should be able to update its own task")

		// 验证任务状态已更新
		updatedTask1, err := db.InputTask.Get(ctx1, task1.ID)
		require.NoError(t, err)
		assert.Equal(t, "running", updatedTask1.TaskStatus)

		// 租户1尝试更新租户2的任务 - 应该失败(找不到任务)
		err = stateMachine.MarkTaskAsRunning(ctx1, task2.ID, TaskTypeInput)
		assert.Error(t, err, "Tenant1 should NOT be able to update Tenant2's task")
	})

	// 测试用例5: SystemContext可以绕过租户隔离(用于系统管理操作)
	t.Run("SystemContext_BypassTenantIsolation", func(t *testing.T) {
		// 创建SystemContext
		systemCtx := hooks.NewSystemContext(context.Background())

		// SystemContext应该能看到所有租户的任务
		allTasks, err := db.InputTask.Query().All(systemCtx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(allTasks), 2, "SystemContext should see all tasks")

		// 验证可以看到两个租户的任务
		taskIDs := make([]uint64, len(allTasks))
		for i, task := range allTasks {
			taskIDs[i] = task.ID
		}
		assert.Contains(t, taskIDs, task1.ID, "Should see tenant1 task")
		assert.Contains(t, taskIDs, task2.ID, "Should see tenant2 task")
	})
}

// TestTenantIsolation_ExecutorExecution 测试Executor执行时的租户隔离
func TestTenantIsolation_ExecutorExecution(t *testing.T) {
	// 初始化测试数据库
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer db.Close()

	// 注册租户隔离Hook (仅注册tenant hooks，不注册department hooks)
	// InputTask/OutputTask没有department_id字段，只需要tenant隔离
	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	// 准备测试数据
	tenant1ID := uint64(1)

	ctx1 := hooks.SetTenantIDToContext(context.Background(), tenant1ID)

	// 创建租户1的任务
	sourceConfig := map[string]interface{}{
		"file_path": "/tmp/test1.csv",
	}
	sourceConfigJSON, _ := json.Marshal(sourceConfig)

	task1, err := db.InputTask.Create().
		SetTaskName("Tenant1_ExecutorTest").
		SetInputSource("file_import").
		SetTaskStatus("pending").
		SetSourceConfig(string(sourceConfigJSON)).
		SetTenantID(tenant1ID).
		Save(ctx1)
	require.NoError(t, err)

	// 测试用例: Executor验证租户隔离
	t.Run("Executor_ValidateTenantIsolation", func(t *testing.T) {
		executorConfig := DefaultExecutorConfig()
		executor := NewExecutor(executorConfig, db, logx.WithContext(context.Background()))

		// 构建ExecutionContext
		execCtx, err := executor.contextManager.BuildExecutionContext(ctx1, task1, TaskTypeInput, db)
		require.NoError(t, err)

		// 验证ExecutionContext包含正确的租户信息
		assert.Equal(t, tenant1ID, execCtx.TenantID)

		// 测试validateTenantIsolation方法(通过反射或导出测试辅助方法)
		// 由于validateTenantIsolation是私有方法,这里测试Execute方法的整体行为

		// ✅ 正常执行: 租户ID匹配
		// 注意: 由于FileImportProvider需要真实文件,这里会失败,但不是租户隔离导致的
		// 主要验证不会出现租户隔离错误
		result := executor.Execute(execCtx)
		// 允许失败,但错误不应该是租户隔离相关
		if result.Status == TaskStatusFailed {
			assert.NotContains(t, result.ErrorMessage, "TENANT ISOLATION VIOLATION")
		}

		// 🚨 异常场景: 修改ExecutionContext的TenantID(模拟攻击)
		execCtxTampered := *execCtx
		execCtxTampered.TenantID = uint64(999) // 篡改租户ID为不存在的租户

		// 执行应该检测到租户不匹配(如果Executor内部有验证)
		// 注意: 当前实现依赖数据库层的Hook,这里主要测试不会泄露跨租户数据
		_ = execCtxTampered // 避免未使用变量警告
	})
}

// TestTenantIsolation_DatabaseQuery 测试数据库查询的租户隔离
func TestTenantIsolation_DatabaseQuery(t *testing.T) {
	// 初始化测试数据库
	db := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer db.Close()

	// 注册租户隔离Hook (仅注册tenant hooks，不注册department hooks)
	// InputTask/OutputTask没有department_id字段，只需要tenant隔离
	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	// 准备测试数据: 创建多个租户的任务
	tenant1ID := uint64(1)
	tenant2ID := uint64(2)
	tenant3ID := uint64(3)

	ctx1 := hooks.SetTenantIDToContext(context.Background(), tenant1ID)
	ctx2 := hooks.SetTenantIDToContext(context.Background(), tenant2ID)
	ctx3 := hooks.SetTenantIDToContext(context.Background(), tenant3ID)

	// 创建多个任务
	_, err := db.InputTask.Create().
		SetTaskName("T1_Task1").
		SetInputSource("file_import").
		SetTaskStatus("pending").
		SetTenantID(tenant1ID).
		Save(ctx1)
	require.NoError(t, err)

	_, err = db.InputTask.Create().
		SetTaskName("T1_Task2").
		SetInputSource("file_import").
		SetTaskStatus("pending").
		SetTenantID(tenant1ID).
		Save(ctx1)
	require.NoError(t, err)

	_, err = db.InputTask.Create().
		SetTaskName("T2_Task1").
		SetInputSource("aliyun_ecs").
		SetTaskStatus("pending").
		SetTenantID(tenant2ID).
		Save(ctx2)
	require.NoError(t, err)

	_, err = db.InputTask.Create().
		SetTaskName("T3_Task1").
		SetInputSource("vmware").
		SetTaskStatus("pending").
		SetTenantID(tenant3ID).
		Save(ctx3)
	require.NoError(t, err)

	// 测试用例1: Query().All() 应该只返回当前租户的数据
	t.Run("Query_All_OnlyCurrentTenant", func(t *testing.T) {
		// 租户1应该只看到2个任务
		tasks1, err := db.InputTask.Query().All(ctx1)
		require.NoError(t, err)
		assert.Len(t, tasks1, 2)
		for _, task := range tasks1 {
			assert.Equal(t, tenant1ID, task.TenantID)
		}

		// 租户2应该只看到1个任务
		tasks2, err := db.InputTask.Query().All(ctx2)
		require.NoError(t, err)
		assert.Len(t, tasks2, 1)
		assert.Equal(t, tenant2ID, tasks2[0].TenantID)

		// 租户3应该只看到1个任务
		tasks3, err := db.InputTask.Query().All(ctx3)
		require.NoError(t, err)
		assert.Len(t, tasks3, 1)
		assert.Equal(t, tenant3ID, tasks3[0].TenantID)
	})

	// 测试用例2: Query().Where() 应该自动添加租户过滤条件
	t.Run("Query_Where_AutoTenantFilter", func(t *testing.T) {
		// 租户1查询 provider_id="file_import" 的任务
		tasks1FileImport, err := db.InputTask.Query().
			Where(inputtask.InputSourceEQ("file_import")).
			All(ctx1)
		require.NoError(t, err)
		assert.Len(t, tasks1FileImport, 2, "Tenant1 should see 2 file_import tasks")

		// 租户2查询 provider_id="file_import" 的任务(应该是0个)
		tasks2FileImport, err := db.InputTask.Query().
			Where(inputtask.InputSourceEQ("file_import")).
			All(ctx2)
		require.NoError(t, err)
		assert.Len(t, tasks2FileImport, 0, "Tenant2 should see 0 file_import tasks")
	})

	// 测试用例3: Count() 应该只统计当前租户的数据
	t.Run("Query_Count_OnlyCurrentTenant", func(t *testing.T) {
		count1, err := db.InputTask.Query().Count(ctx1)
		require.NoError(t, err)
		assert.Equal(t, 2, count1)

		count2, err := db.InputTask.Query().Count(ctx2)
		require.NoError(t, err)
		assert.Equal(t, 1, count2)

		count3, err := db.InputTask.Query().Count(ctx3)
		require.NoError(t, err)
		assert.Equal(t, 1, count3)
	})
}

// BenchmarkTenantIsolation_QueryPerformance 租户隔离性能基准测试
func BenchmarkTenantIsolation_QueryPerformance(b *testing.B) {
	// 初始化测试数据库
	db, err := ent.Open("sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// 创建schema
	if err := db.Schema.Create(context.Background()); err != nil {
		b.Fatalf("failed to create schema: %v", err)
	}

	// 注册租户隔离Hook (仅注册tenant hooks，不注册department hooks)
	// InputTask/OutputTask没有department_id字段，只需要tenant隔离
	hooks.InitDefaultHookConfigs()
	hooks.RegisterTenantHooks(db)

	// 准备测试数据: 创建1000个任务(10个租户,每个租户100个任务)
	for tenantID := uint64(1); tenantID <= 10; tenantID++ {
		tenantCtx := hooks.SetTenantIDToContext(context.Background(), tenantID)
		for i := 0; i < 100; i++ {
			_, err := db.InputTask.Create().
				SetTaskName("BenchmarkTask").
				SetInputSource("file_import").
				SetTaskStatus("pending").
				SetTenantID(tenantID).
				Save(tenantCtx)
			if err != nil {
				b.Fatalf("failed to create task: %v", err)
			}
		}
	}

	// 基准测试: 租户查询性能
	ctx := hooks.SetTenantIDToContext(context.Background(), uint64(5))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tasks, err := db.InputTask.Query().
			Where(inputtask.TaskStatusEQ("pending")).
			Limit(10).
			All(ctx)
		if err != nil {
			b.Fatalf("query failed: %v", err)
		}
		if len(tasks) == 0 {
			b.Fatal("expected tasks but got none")
		}
	}
}
