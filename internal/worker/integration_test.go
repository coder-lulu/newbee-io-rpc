package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/coder-lulu/newbee-common/v2/orm/ent/hooks"
	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/ent/enttest"
	"github.com/coder-lulu/newbee-io-rpc/ent/inputtask"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

// TestScheduledTaskIntegration 集成测试：验证定时任务在真实环境中的完整流程
func TestScheduledTaskIntegration(t *testing.T) {
	// 创建测试数据库
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1&_journal_mode=WAL", opts...)
	defer client.Close()

	// 注册租户Hook（模拟真实环境）
	client.Use(hooks.TenantMutationHook())
	client.Intercept(hooks.TenantQueryInterceptor())

	ctx := context.Background()

	t.Run("基础定时任务创建和执行", func(t *testing.T) {
		// 使用SystemContext创建测试任务（绕过租户检查）
		systemCtx := hooks.NewSystemContext(ctx)

		// 1. 创建一个30秒后到期的定时任务
		scheduledTime := time.Now().Add(30 * time.Second)
		task, err := client.InputTask.Create().
			SetTaskName("集成测试-基础定时任务").
			SetTaskType("scheduled").
			SetTaskStatus("pending").
			SetInputSource("test_integration").
			SetSourceConfig(`{"test": true}`).
			SetScheduledAt(scheduledTime).
			SetTenantID(1).
			Save(systemCtx)
		require.NoError(t, err)
		t.Logf("✅ 创建定时任务成功: ID=%d, scheduled_at=%s", task.ID, scheduledTime.Format(time.RFC3339))

		// 2. 验证初始状态
		assert.Equal(t, "pending", task.TaskStatus)
		assert.Equal(t, "scheduled", task.TaskType)
		assert.NotNil(t, task.ScheduledAt)

		// 3. 启动 TaskWorker
		workerConfig := &WorkerConfig{
			PullInterval:  5 * time.Second,  // 更短的间隔用于测试
			MaxConcurrent: 5,
			BatchSize:     10,
			TaskTimeout:   1 * time.Minute,
		}

		tw := NewTaskWorker(client, workerConfig)
		workerCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		// 启动worker（后台运行）
		go tw.Start(workerCtx)
		t.Log("✅ TaskWorker 已启动")

		// 4. 等待5秒，确认任务仍然是pending（未到期）
		time.Sleep(5 * time.Second)
		updatedTask, err := client.InputTask.Get(systemCtx, task.ID)
		require.NoError(t, err)
		assert.Equal(t, "pending", updatedTask.TaskStatus, "任务未到期前应保持pending状态")
		t.Log("✅ 验证通过: 任务未到期前保持pending状态")

		// 5. 等待任务到期并执行（等待70秒，确保scheduled ticker触发）
		t.Log("⏳ 等待任务到期执行（70秒，等待scheduled ticker触发）...")
		time.Sleep(70 * time.Second)

		// 6. 验证任务已被处理
		finalTask, err := client.InputTask.Get(systemCtx, task.ID)
		require.NoError(t, err)

		// 注意：在真实环境中，任务会执行实际的业务逻辑
		// 这里我们只验证任务状态变化（completed 或 failed）
		assert.Contains(t, []string{"completed", "processing", "failed"}, finalTask.TaskStatus,
			"任务应该已经被处理")

		if finalTask.TaskStatus == "completed" {
			t.Logf("✅ 任务执行成功: started_at=%v, completed_at=%v",
				finalTask.StartedAt, finalTask.CompletedAt)

			// 验证时间字段
			assert.NotNil(t, finalTask.StartedAt, "started_at应该被设置")
			assert.NotNil(t, finalTask.CompletedAt, "completed_at应该被设置")

			// 验证执行延迟（应该在合理范围内）
			if finalTask.StartedAt != nil {
				delay := finalTask.StartedAt.Sub(scheduledTime)
				t.Logf("📊 执行延迟: %.1f 秒", delay.Seconds())
				assert.Less(t, delay.Seconds(), 70.0, "执行延迟应该小于70秒（1分钟ticker + 容差）")
			}
		}
	})

	t.Run("验证任务类型隔离", func(t *testing.T) {
		// 使用SystemContext创建测试任务（绕过租户检查）
		systemCtx := hooks.NewSystemContext(ctx)

		// 1. 创建一个未来的定时任务（5分钟后）
		futureTask, err := client.InputTask.Create().
			SetTaskName("集成测试-未来定时任务").
			SetTaskType("scheduled").
			SetTaskStatus("pending").
			SetInputSource("test_integration").
			SetScheduledAt(time.Now().Add(5 * time.Minute)).
			SetTenantID(1).
			Save(systemCtx)
		require.NoError(t, err)
		t.Logf("✅ 创建未来定时任务: ID=%d", futureTask.ID)

		// 2. 创建一个手动任务
		manualTask, err := client.InputTask.Create().
			SetTaskName("集成测试-手动任务").
			SetTaskType("manual").
			SetTaskStatus("pending").
			SetInputSource("test_integration").
			SetTenantID(1).
			Save(systemCtx)
		require.NoError(t, err)
		t.Logf("✅ 创建手动任务: ID=%d", manualTask.ID)

		// 3. 启动 TaskWorker
		workerConfig := &WorkerConfig{
			PullInterval:  3 * time.Second,
			MaxConcurrent: 5,
			BatchSize:     10,
		}

		tw := NewTaskWorker(client, workerConfig)
		workerCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		go tw.Start(workerCtx)
		t.Log("✅ TaskWorker 已启动")

		// 4. 等待手动任务被处理（应该很快）
		time.Sleep(8 * time.Second)

		// 5. 验证手动任务已处理，定时任务未处理
		updatedManual, err := client.InputTask.Get(systemCtx, manualTask.ID)
		require.NoError(t, err)

		updatedFuture, err := client.InputTask.Get(systemCtx, futureTask.ID)
		require.NoError(t, err)

		// 手动任务应该已经被处理
		assert.Contains(t, []string{"completed", "processing", "failed"}, updatedManual.TaskStatus,
			"手动任务应该被processOnce处理")
		t.Logf("✅ 手动任务状态: %s", updatedManual.TaskStatus)

		// 未来的定时任务应该仍然是pending
		assert.Equal(t, "pending", updatedFuture.TaskStatus,
			"未到期的定时任务应该保持pending状态")
		t.Log("✅ 验证通过: 任务类型完全隔离")
	})

	t.Run("多租户定时任务测试", func(t *testing.T) {
		// 使用SystemContext创建测试任务（绕过租户检查）
		systemCtx := hooks.NewSystemContext(ctx)

		// 1. 为不同租户创建定时任务
		scheduledTime := time.Now().Add(20 * time.Second)

		tenant1Task, err := client.InputTask.Create().
			SetTaskName("集成测试-租户1").
			SetTaskType("scheduled").
			SetTaskStatus("pending").
			SetInputSource("test_integration").
			SetScheduledAt(scheduledTime).
			SetTenantID(1).
			Save(systemCtx)
		require.NoError(t, err)

		tenant2Task, err := client.InputTask.Create().
			SetTaskName("集成测试-租户2").
			SetTaskType("scheduled").
			SetTaskStatus("pending").
			SetInputSource("test_integration").
			SetScheduledAt(scheduledTime).
			SetTenantID(2).
			Save(systemCtx)
		require.NoError(t, err)

		t.Logf("✅ 创建多租户任务: 租户1=%d, 租户2=%d", tenant1Task.ID, tenant2Task.ID)

		// 2. 启动 TaskWorker
		workerConfig := &WorkerConfig{
			PullInterval:  5 * time.Second,
			MaxConcurrent: 5,
			BatchSize:     10,
		}

		tw := NewTaskWorker(client, workerConfig)
		workerCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		go tw.Start(workerCtx)
		t.Log("✅ TaskWorker 已启动")

		// 3. 等待任务到期并执行（等待70秒确保scheduled ticker触发）
		t.Log("⏳ 等待任务到期执行（70秒，等待scheduled ticker触发）...")
		time.Sleep(70 * time.Second)

		// 4. 验证两个租户的任务都被处理（使用已创建的systemCtx）
		updatedTenant1, err := client.InputTask.Query().
			Where(inputtask.IDEQ(tenant1Task.ID)).
			First(systemCtx) // 使用SystemContext查询
		require.NoError(t, err)

		updatedTenant2, err := client.InputTask.Query().
			Where(inputtask.IDEQ(tenant2Task.ID)).
			First(systemCtx)
		require.NoError(t, err)

		// 两个任务都应该被处理
		assert.Contains(t, []string{"completed", "processing", "failed"}, updatedTenant1.TaskStatus)
		assert.Contains(t, []string{"completed", "processing", "failed"}, updatedTenant2.TaskStatus)

		t.Logf("✅ 租户1任务状态: %s, 租户2任务状态: %s",
			updatedTenant1.TaskStatus, updatedTenant2.TaskStatus)
		t.Log("✅ 验证通过: 多租户任务都正确执行")
	})
}

// TestScheduledTaskValidation 集成测试：验证参数验证逻辑
func TestScheduledTaskValidation(t *testing.T) {
	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1", opts...)
	defer client.Close()

	ctx := context.Background()

	t.Run("scheduled任务可以创建", func(t *testing.T) {
		// 这个测试验证schema层面允许创建scheduled任务
		futureTime := time.Now().Add(1 * time.Hour)
		task, err := client.InputTask.Create().
			SetTaskName("验证测试-正常定时任务").
			SetTaskType("scheduled").
			SetTaskStatus("pending").
			SetInputSource("test").
			SetScheduledAt(futureTime).
			SetTenantID(1).
			Save(ctx)

		require.NoError(t, err, "应该能成功创建scheduled任务")
		assert.Equal(t, "scheduled", task.TaskType)
		assert.NotNil(t, task.ScheduledAt)
		t.Logf("✅ 成功创建scheduled任务: ID=%d", task.ID)
	})

	t.Run("manual任务可以不提供scheduled_at", func(t *testing.T) {
		task, err := client.InputTask.Create().
			SetTaskName("验证测试-手动任务").
			SetTaskType("manual").
			SetTaskStatus("pending").
			SetInputSource("test").
			SetTenantID(1).
			Save(ctx)

		require.NoError(t, err, "手动任务不需要scheduled_at")
		assert.Equal(t, "manual", task.TaskType)
		assert.Nil(t, task.ScheduledAt)
		t.Logf("✅ 成功创建manual任务（无scheduled_at）: ID=%d", task.ID)
	})
}

// TestScheduledTaskPerformance 性能测试：批量定时任务处理
func TestScheduledTaskPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过性能测试（使用 -short 标志）")
	}

	opts := []enttest.Option{
		enttest.WithOptions(ent.Log(t.Log)),
	}
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1&_journal_mode=WAL", opts...)
	defer client.Close()

	ctx := context.Background()

	t.Run("批量定时任务处理", func(t *testing.T) {
		// 使用SystemContext创建测试任务（绕过租户检查）
		systemCtx := hooks.NewSystemContext(ctx)

		// 1. 创建30个同时到期的定时任务
		scheduledTime := time.Now().Add(20 * time.Second)
		taskCount := 30

		t.Logf("📝 创建 %d 个定时任务...", taskCount)
		createdTasks := make([]*ent.InputTask, 0, taskCount)

		for i := 0; i < taskCount; i++ {
			task, err := client.InputTask.Create().
				SetTaskName(fmt.Sprintf("性能测试-%d", i+1)).
				SetTaskType("scheduled").
				SetTaskStatus("pending").
				SetInputSource("test_performance").
				SetScheduledAt(scheduledTime).
				SetTenantID(1).
				Save(systemCtx)
			require.NoError(t, err)
			createdTasks = append(createdTasks, task)
		}
		t.Logf("✅ 成功创建 %d 个定时任务", taskCount)

		// 2. 启动 TaskWorker（配置较小的并发数以观察排队行为）
		workerConfig := &WorkerConfig{
			PullInterval:  5 * time.Second,
			MaxConcurrent: 3,  // 限制并发数
			BatchSize:     10, // 每次拉取10个
		}

		tw := NewTaskWorker(client, workerConfig)
		workerCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		startTime := time.Now()
		go tw.Start(workerCtx)
		t.Log("✅ TaskWorker 已启动 (MaxConcurrent=3, BatchSize=10)")

		// 3. 等待所有任务完成（最多等待3分钟）
		t.Log("⏳ 等待所有任务执行完成...")
		deadline := time.Now().Add(3 * time.Minute)

		completedCount := 0
		for time.Now().Before(deadline) {
			time.Sleep(10 * time.Second)

			// 统计已完成的任务
			systemCtx := hooks.NewSystemContext(ctx)
			count, err := client.InputTask.Query().
				Where(
					inputtask.TaskNameHasPrefix("性能测试-"),
					inputtask.TaskStatusIn("completed", "failed"),
				).
				Count(systemCtx)
			require.NoError(t, err)
			completedCount = count

			t.Logf("📊 进度: %d/%d 任务已完成", completedCount, taskCount)

			if completedCount == taskCount {
				break
			}
		}

		totalDuration := time.Since(startTime)
		t.Logf("⏱️  总耗时: %.1f 秒", totalDuration.Seconds())

		// 4. 统计结果（使用已创建的systemCtx）
		completedTasks, err := client.InputTask.Query().
			Where(
				inputtask.TaskNameHasPrefix("性能测试-"),
				inputtask.TaskStatusEQ("completed"),
			).
			All(systemCtx)
		require.NoError(t, err)

		failedTasks, err := client.InputTask.Query().
			Where(
				inputtask.TaskNameHasPrefix("性能测试-"),
				inputtask.TaskStatusEQ("failed"),
			).
			Count(systemCtx)
		require.NoError(t, err)

		t.Logf("\n📈 性能测试结果:")
		t.Logf("   总任务数: %d", taskCount)
		t.Logf("   成功完成: %d", len(completedTasks))
		t.Logf("   执行失败: %d", failedTasks)
		t.Logf("   总耗时: %.1f 秒", totalDuration.Seconds())

		// 计算执行延迟统计
		if len(completedTasks) > 0 {
			var totalDelay time.Duration
			var minDelay, maxDelay time.Duration
			minDelay = 999 * time.Hour // 初始化为很大的值

			for _, task := range completedTasks {
				if task.StartedAt != nil && task.ScheduledAt != nil {
					delay := task.StartedAt.Sub(*task.ScheduledAt)
					totalDelay += delay

					if delay < minDelay {
						minDelay = delay
					}
					if delay > maxDelay {
						maxDelay = delay
					}
				}
			}

			avgDelay := totalDelay / time.Duration(len(completedTasks))
			t.Logf("   平均延迟: %.1f 秒", avgDelay.Seconds())
			t.Logf("   最小延迟: %.1f 秒", minDelay.Seconds())
			t.Logf("   最大延迟: %.1f 秒", maxDelay.Seconds())
		}

		// 验证大部分任务都成功完成
		successRate := float64(len(completedTasks)) / float64(taskCount) * 100
		t.Logf("   成功率: %.1f%%\n", successRate)

		assert.GreaterOrEqual(t, successRate, 80.0, "至少80%的任务应该成功完成")
	})
}
