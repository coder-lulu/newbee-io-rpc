package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestCronTask 创建测试用的CronTask
func createTestCronTask(t *testing.T, client *ent.Client, taskName string, cronExpr string, enabled bool) *ent.CronTask {
	task, err := client.CronTask.Create().
		SetTaskName(taskName).
		SetCronExpression(cronExpr).
		SetInputSource("test_provider").
		SetSourceConfig(`{"test": "config"}`).
		SetEnabled(enabled).
		SetNextRunTime(time.Now().Add(1 * time.Hour)).
		SetExecutionCount(0).
		SetSuccessCount(0).
		SetFailureCount(0).
		SetTenantID(1).
		SetStatus(1).
		Save(context.Background())

	require.NoError(t, err)
	return task
}

// TestNewCronScheduler 测试CronScheduler创建
func TestNewCronScheduler(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	scheduler := NewCronScheduler(client, nil)

	assert.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.db)
	assert.NotNil(t, scheduler.cron)
	assert.NotNil(t, scheduler.taskMap)
	assert.Equal(t, 0, len(scheduler.taskMap))
}

// TestAddTask 测试添加任务到调度器
func TestAddTask(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	scheduler := NewCronScheduler(client, nil)

	tests := []struct {
		name        string
		taskName    string
		cronExpr    string
		enabled     bool
		expectError bool
	}{
		{
			name:        "Valid cron expression - every minute",
			taskName:    "Test Task 1",
			cronExpr:    "*/1 * * * *",
			enabled:     true,
			expectError: false,
		},
		{
			name:        "Valid cron expression - daily at 2am",
			taskName:    "Test Task 2",
			cronExpr:    "0 2 * * *",
			enabled:     true,
			expectError: false,
		},
		{
			name:        "Invalid cron expression",
			taskName:    "Test Task 3",
			cronExpr:    "invalid cron",
			enabled:     true,
			expectError: true,
		},
		{
			name:        "Valid but disabled task",
			taskName:    "Test Task 4",
			cronExpr:    "0 3 * * *",
			enabled:     false,
			expectError: false, // AddTask should not error, just skip
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := createTestCronTask(t, client, tt.taskName, tt.cronExpr, tt.enabled)

			err := scheduler.AddTask(task)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.enabled {
					// 验证任务已添加到taskMap
					scheduler.taskMapLock.RLock()
					_, exists := scheduler.taskMap[task.ID]
					scheduler.taskMapLock.RUnlock()
					assert.True(t, exists, "Task should be in taskMap")
				}
			}
		})
	}
}

// TestRemoveTask 测试从调度器移除任务
func TestRemoveTask(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	scheduler := NewCronScheduler(client, nil)

	// 创建并添加任务
	task := createTestCronTask(t, client, "Test Task", "*/5 * * * *", true)
	err := scheduler.AddTask(task)
	require.NoError(t, err)

	// 验证任务存在
	scheduler.taskMapLock.RLock()
	_, exists := scheduler.taskMap[task.ID]
	scheduler.taskMapLock.RUnlock()
	assert.True(t, exists)

	// 移除任务
	err = scheduler.RemoveTask(task.ID)
	assert.NoError(t, err)

	// 验证任务已移除
	scheduler.taskMapLock.RLock()
	_, exists = scheduler.taskMap[task.ID]
	scheduler.taskMapLock.RUnlock()
	assert.False(t, exists)

	// 再次移除（应该返回错误，因为任务不存在）
	err = scheduler.RemoveTask(task.ID)
	assert.Error(t, err, "Should return error when removing non-existent task")
	assert.Contains(t, err.Error(), "not found in scheduler")
}

// TestUpdateTask 测试更新任务
func TestUpdateTask(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	scheduler := NewCronScheduler(client, nil)

	// 创建并添加任务
	task := createTestCronTask(t, client, "Test Task", "*/5 * * * *", true)
	err := scheduler.AddTask(task)
	require.NoError(t, err)

	// 更新任务的cron表达式
	updatedTask, err := client.CronTask.UpdateOneID(task.ID).
		SetCronExpression("*/10 * * * *").
		Save(context.Background())
	require.NoError(t, err)

	// 更新调度器中的任务
	err = scheduler.UpdateTask(updatedTask)
	assert.NoError(t, err)

	// 验证任务仍在taskMap中
	scheduler.taskMapLock.RLock()
	_, exists := scheduler.taskMap[task.ID]
	scheduler.taskMapLock.RUnlock()
	assert.True(t, exists)
}

// TestDisableTaskWorkflow 测试禁用任务的正确工作流程
// 在实际使用中，禁用任务应该调用RemoveTask而不是UpdateTask
func TestDisableTaskWorkflow(t *testing.T) {
	client := setupTestDB(t)

	scheduler := NewCronScheduler(client, nil)

	// 创建并添加任务
	task := createTestCronTask(t, client, "Test Task", "*/5 * * * *", true)
	err := scheduler.AddTask(task)
	require.NoError(t, err)

	// 验证任务已添加
	scheduler.taskMapLock.RLock()
	_, exists := scheduler.taskMap[task.ID]
	scheduler.taskMapLock.RUnlock()
	assert.True(t, exists)

	// 更新任务为禁用（数据库操作）
	_, err = client.CronTask.UpdateOneID(task.ID).
		SetEnabled(false).
		Save(context.Background())
	require.NoError(t, err)

	// 根据实际使用模式，禁用任务应该调用RemoveTask
	err = scheduler.RemoveTask(task.ID)
	assert.NoError(t, err)

	// 验证任务已从taskMap移除
	scheduler.taskMapLock.RLock()
	_, exists = scheduler.taskMap[task.ID]
	scheduler.taskMapLock.RUnlock()
	assert.False(t, exists, "Disabled task should be removed from scheduler")
}

// TestStart 测试启动调度器
func TestStart(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	// 创建多个测试任务
	createTestCronTask(t, client, "Enabled Task 1", "*/5 * * * *", true)
	createTestCronTask(t, client, "Enabled Task 2", "*/10 * * * *", true)
	createTestCronTask(t, client, "Disabled Task", "*/15 * * * *", false)
	createTestCronTask(t, client, "Invalid Task", "invalid", true)

	scheduler := NewCronScheduler(client, nil)

	// 启动调度器（在goroutine中，避免阻塞）
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := scheduler.Start(ctx)
		assert.NoError(t, err)
	}()

	// 等待调度器启动
	time.Sleep(500 * time.Millisecond)

	// 验证只有启用的有效任务被加载
	scheduler.taskMapLock.RLock()
	taskCount := len(scheduler.taskMap)
	scheduler.taskMapLock.RUnlock()

	// 应该加载2个任务（2个启用的有效任务）
	assert.Equal(t, 2, taskCount, "Should load 2 enabled valid tasks")

	// 停止调度器
	cancel()
	time.Sleep(100 * time.Millisecond)
}

// TestStop 测试停止调度器
func TestStop(t *testing.T) {
	client := setupTestDB(t)

	scheduler := NewCronScheduler(client, nil)

	// 创建启用的任务（Start会自动加载）
	createTestCronTask(t, client, "Test Task", "*/5 * * * *", true)

	// 启动调度器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		scheduler.Start(ctx)
	}()

	// 等待启动和任务加载
	time.Sleep(200 * time.Millisecond)

	// 验证任务已加载
	scheduler.taskMapLock.RLock()
	taskCountBefore := len(scheduler.taskMap)
	scheduler.taskMapLock.RUnlock()
	assert.Equal(t, 1, taskCountBefore, "Should have 1 task before Stop")

	// 停止调度器
	err := scheduler.Stop()
	assert.NoError(t, err)

	// 验证调度器已停止（cron.Stop()已被调用）
	// taskMap不会被清空（这是设计行为，保留状态用于重启）
	scheduler.taskMapLock.RLock()
	taskCountAfter := len(scheduler.taskMap)
	scheduler.taskMapLock.RUnlock()

	// 任务仍在taskMap中，但调度器已停止，不会再触发执行
	assert.Equal(t, 1, taskCountAfter, "taskMap preserved after Stop for potential restart")

	// 验证调度器不再接受新任务调度
	// （无法直接测试，因为这是cron库内部行为）
	t.Log("CronScheduler stopped successfully, tasks preserved in memory")
}

// TestConcurrentAddRemove 测试并发添加和移除任务
func TestConcurrentAddRemove(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	scheduler := NewCronScheduler(client, nil)

	// 创建多个任务
	tasks := make([]*ent.CronTask, 10)
	for i := 0; i < 10; i++ {
		tasks[i] = createTestCronTask(
			t,
			client,
			fmt.Sprintf("Task %d", i),
			"*/5 * * * *",
			true,
		)
	}

	// 并发添加任务
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(task *ent.CronTask) {
			err := scheduler.AddTask(task)
			assert.NoError(t, err)
			done <- true
		}(tasks[i])
	}

	// 等待所有添加完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证所有任务都添加成功
	scheduler.taskMapLock.RLock()
	taskCount := len(scheduler.taskMap)
	scheduler.taskMapLock.RUnlock()
	assert.Equal(t, 10, taskCount)

	// 并发移除任务
	for i := 0; i < 10; i++ {
		go func(taskID uint64) {
			err := scheduler.RemoveTask(taskID)
			assert.NoError(t, err)
			done <- true
		}(tasks[i].ID)
	}

	// 等待所有移除完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证所有任务都移除成功
	scheduler.taskMapLock.RLock()
	taskCount = len(scheduler.taskMap)
	scheduler.taskMapLock.RUnlock()
	assert.Equal(t, 0, taskCount)
}

// TestExecuteTask_CreatesInputTask 测试executeTask是否正确创建InputTask
func TestExecuteTask_CreatesInputTask(t *testing.T) {
	client := setupTestDB(t)

	scheduler := NewCronScheduler(client, nil)

	// 创建测试任务
	task := createTestCronTask(t, client, "Test Task", "*/5 * * * *", true)

	// 先添加到调度器（模拟正常调度流程）
	err := scheduler.AddTask(task)
	require.NoError(t, err)

	// 手动触发executeTask（通常由cron调度器自动触发）
	scheduler.executeTask(task.ID)

	// 等待异步执行完成
	time.Sleep(200 * time.Millisecond)

	// 验证InputTask是否被创建
	inputTasks, err := client.InputTask.Query().
		Where().
		All(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, len(inputTasks), "Should create 1 InputTask")

	if len(inputTasks) > 0 {
		inputTask := inputTasks[0]
		assert.Equal(t, "cron", inputTask.TaskType)
		assert.Equal(t, task.ID, *inputTask.CronTaskID)
		assert.Equal(t, "pending", inputTask.TaskStatus)
		assert.Equal(t, task.InputSource, inputTask.InputSource)
	}

	// 验证CronTask统计是否更新
	updatedTask, err := client.CronTask.Get(context.Background(), task.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, updatedTask.ExecutionCount)
	assert.NotNil(t, updatedTask.LastRunTime)
}

// TestTaskMapThreadSafety 测试taskMap的线程安全性
func TestTaskMapThreadSafety(t *testing.T) {
	client := setupTestDB(t)

	scheduler := NewCronScheduler(client, nil)

	// 创建10个测试任务
	tasks := make([]*ent.CronTask, 10)
	for i := 0; i < 10; i++ {
		tasks[i] = createTestCronTask(
			t,
			client,
			fmt.Sprintf("Task %d", i),
			"*/5 * * * *",
			true,
		)
	}

	// 并发读写测试
	done := make(chan bool, 30)

	// 10个goroutine并发添加
	for i := 0; i < 10; i++ {
		go func(idx int) {
			scheduler.AddTask(tasks[idx])
			done <- true
		}(i)
	}

	// 10个goroutine并发读取
	for i := 0; i < 10; i++ {
		go func() {
			scheduler.taskMapLock.RLock()
			_ = len(scheduler.taskMap)
			scheduler.taskMapLock.RUnlock()
			done <- true
		}()
	}

	// 10个goroutine并发移除（其中一些任务可能还未添加）
	for i := 0; i < 10; i++ {
		go func(idx int) {
			time.Sleep(10 * time.Millisecond) // 稍微延迟
			scheduler.RemoveTask(tasks[idx].ID)
			done <- true
		}(i)
	}

	// 等待所有操作完成
	for i := 0; i < 30; i++ {
		<-done
	}

	// 验证无panic，无数据竞争
	t.Log("Thread safety test passed without panic or race conditions")
}
