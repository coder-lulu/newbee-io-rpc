package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/coder-lulu/newbee-common/v2/orm/ent/hooks"
	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/ent/crontask"
	"github.com/coder-lulu/newbee-io-rpc/internal/lock"
	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/logx"
)

// CronScheduler manages cron-based scheduled tasks
// CronScheduler 负责管理基于 Cron 表达式的周期性任务调度
type CronScheduler struct {
	db          *ent.Client
	cron        *cron.Cron
	taskMap     map[uint64]cron.EntryID // cronTaskID -> cron entryID
	taskMapLock sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	lockManager *lock.DistributedLockManager // 分布式锁管理器（防止多实例重复触发）
}

// NewCronScheduler creates a new CronScheduler instance
// 创建新的 CronScheduler 实例
func NewCronScheduler(db *ent.Client, lockManager *lock.DistributedLockManager) *CronScheduler {
	// 创建 cron 实例，使用标准5字段格式 (分 时 日 月 周)
	// 使用 UTC 时区，避免夏令时问题
	c := cron.New(
		cron.WithLocation(time.UTC),
		cron.WithChain(
			cron.SkipIfStillRunning(cron.DefaultLogger),
		),
	)

	ctx, cancel := context.WithCancel(context.Background())

	return &CronScheduler{
		db:          db,
		cron:        c,
		taskMap:     make(map[uint64]cron.EntryID),
		ctx:         ctx,
		cancel:      cancel,
		lockManager: lockManager,
	}
}

// Start starts the cron scheduler
// 启动 Cron 调度器
func (cs *CronScheduler) Start(ctx context.Context) error {
	logx.Info("Starting CronScheduler...")

	// 从数据库加载所有启用的 Cron 任务
	if err := cs.LoadTasks(ctx); err != nil {
		return fmt.Errorf("failed to load cron tasks: %w", err)
	}

	// 启动 cron 调度器
	cs.cron.Start()

	logx.Infow("CronScheduler started successfully",
		logx.Field("loaded_tasks", len(cs.taskMap)))

	return nil
}

// Stop stops the cron scheduler gracefully
// 优雅停止 Cron 调度器
func (cs *CronScheduler) Stop() error {
	logx.Info("Stopping CronScheduler...")

	// 取消上下文
	cs.cancel()

	// 停止 cron 调度器（等待所有正在运行的任务完成）
	stopCtx := cs.cron.Stop()
	<-stopCtx.Done()

	logx.Info("CronScheduler stopped successfully")
	return nil
}

// LoadTasks loads all enabled cron tasks from database
// 从数据库加载所有启用的 Cron 任务
func (cs *CronScheduler) LoadTasks(ctx context.Context) error {
	// 使用 SystemContext 绕过租户隔离，查询所有租户的 Cron 任务
	systemCtx := hooks.NewSystemContext(ctx)

	// 查询所有启用的 Cron 任务
	tasks, err := cs.db.CronTask.Query().
		Where(
			crontask.StatusEQ(1),      // status = 1 (normal)
			crontask.EnabledEQ(true),  // enabled = true
		).
		All(systemCtx)

	if err != nil {
		return fmt.Errorf("failed to query enabled cron tasks: %w", err)
	}

	logx.Infow("Loading cron tasks from database",
		logx.Field("count", len(tasks)))

	// 为每个任务添加到调度器
	for _, task := range tasks {
		if err := cs.AddTask(task); err != nil {
			logx.Errorw("Failed to add cron task to scheduler",
				logx.Field("task_id", task.ID),
				logx.Field("task_name", task.TaskName),
				logx.Field("error", err))
			// 继续加载其他任务，不因单个任务失败而中断
			continue
		}
	}

	return nil
}

// AddTask adds a cron task to the scheduler
// 添加 Cron 任务到调度器
func (cs *CronScheduler) AddTask(task *ent.CronTask) error {
	cs.taskMapLock.Lock()
	defer cs.taskMapLock.Unlock()

	// 检查任务是否已存在
	if _, exists := cs.taskMap[task.ID]; exists {
		return fmt.Errorf("cron task %d already exists in scheduler", task.ID)
	}

	// 解析 Cron 表达式
	entryID, err := cs.cron.AddFunc(task.CronExpression, func() {
		cs.executeTask(task.ID)
	})

	if err != nil {
		return fmt.Errorf("failed to parse cron expression '%s': %w", task.CronExpression, err)
	}

	// 保存 entryID 到映射
	cs.taskMap[task.ID] = entryID

	// 更新 next_run_time 到数据库
	entry := cs.cron.Entry(entryID)
	nextRunTime := entry.Next

	systemCtx := hooks.NewSystemContext(cs.ctx)
	_, err = cs.db.CronTask.UpdateOneID(task.ID).
		SetNextRunTime(nextRunTime).
		Save(systemCtx)

	if err != nil {
		logx.Errorw("Failed to update next_run_time",
			logx.Field("task_id", task.ID),
			logx.Field("error", err))
		// 不影响任务添加，只记录日志
	}

	logx.Infow("Added cron task to scheduler",
		logx.Field("task_id", task.ID),
		logx.Field("task_name", task.TaskName),
		logx.Field("cron_expression", task.CronExpression),
		logx.Field("next_run", nextRunTime.Format(time.RFC3339)))

	return nil
}

// RemoveTask removes a cron task from the scheduler
// 从调度器移除 Cron 任务
func (cs *CronScheduler) RemoveTask(taskID uint64) error {
	cs.taskMapLock.Lock()
	defer cs.taskMapLock.Unlock()

	// 查找 entryID
	entryID, exists := cs.taskMap[taskID]
	if !exists {
		return fmt.Errorf("cron task %d not found in scheduler", taskID)
	}

	// 从 cron 调度器移除
	cs.cron.Remove(entryID)

	// 从映射中删除
	delete(cs.taskMap, taskID)

	logx.Infow("Removed cron task from scheduler",
		logx.Field("task_id", taskID))

	return nil
}

// UpdateTask updates a cron task in the scheduler
// 更新调度器中的 Cron 任务（先删除后添加）
func (cs *CronScheduler) UpdateTask(task *ent.CronTask) error {
	// 先移除旧任务（如果存在）
	if err := cs.RemoveTask(task.ID); err != nil {
		// 如果任务不存在，忽略错误
		logx.Infow("Task not found in scheduler, will add as new",
			logx.Field("task_id", task.ID))
	}

	// 添加新任务
	return cs.AddTask(task)
}

// executeTask executes a cron task by creating an InputTask instance
// 执行 Cron 任务（创建一个 InputTask 实例）
//
// 🔒 多实例保护：使用分布式锁确保同一时刻只有一个实例能创建InputTask
func (cs *CronScheduler) executeTask(cronTaskID uint64) {
	logx.Infow("Cron task triggered",
		logx.Field("cron_task_id", cronTaskID))

	// 🔒 使用分布式锁防止多实例重复创建InputTask
	// 锁的key格式：cron:task:{cronTaskID}:exec
	// 锁的过期时间：30秒（足够完成一次InputTask创建）
	// 策略：如果获取锁失败，跳过本次执行（由获取到锁的实例执行）
	lockKey := fmt.Sprintf("cron:task:%d:exec", cronTaskID)

	if cs.lockManager == nil {
		logx.Infow("LockManager not initialized, executing without distributed lock protection",
			logx.Field("cron_task_id", cronTaskID))
		cs.executeTaskWithoutLock(cronTaskID)
		return
	}

	// 尝试获取锁并执行
	executed, err := cs.lockManager.ExecuteWithLockOrSkip(
		cs.ctx,
		lockKey,
		30*time.Second, // 锁的过期时间
		func() error {
			cs.executeTaskWithoutLock(cronTaskID)
			return nil
		},
	)

	if err != nil {
		logx.Errorw("Failed to execute cron task with lock",
			logx.Field("cron_task_id", cronTaskID),
			logx.Field("error", err))
		return
	}

	if !executed {
		logx.Infow("Cron task execution skipped (lock held by another instance)",
			logx.Field("cron_task_id", cronTaskID),
			logx.Field("lock_key", lockKey))
	}
}

// executeTaskWithoutLock 实际执行任务的内部方法（不包含锁逻辑）
func (cs *CronScheduler) executeTaskWithoutLock(cronTaskID uint64) {
	// 使用 SystemContext 绕过租户隔离
	systemCtx := hooks.NewSystemContext(cs.ctx)

	// 查询 CronTask 详情
	cronTask, err := cs.db.CronTask.Get(systemCtx, cronTaskID)
	if err != nil {
		logx.Errorw("Failed to get cron task",
			logx.Field("cron_task_id", cronTaskID),
			logx.Field("error", err))
		return
	}

	// 创建 InputTask 实例
	executionTime := time.Now()
	inputTask, err := cs.db.InputTask.Create().
		SetTaskName(fmt.Sprintf("%s - %s", cronTask.TaskName, executionTime.Format("2006-01-02 15:04:05"))).
		SetTaskType("cron").
		SetInputSource(cronTask.InputSource).
		SetSourceConfig(cronTask.SourceConfig).
		SetTaskStatus("pending").
		SetCronTaskID(cronTask.ID).
		SetExecutionTime(executionTime).
		SetTenantID(cronTask.TenantID).
		SetStatus(1).
		Save(systemCtx)

	if err != nil {
		logx.Errorw("Failed to create InputTask for cron task",
			logx.Field("cron_task_id", cronTaskID),
			logx.Field("error", err))

		// 更新失败计数
		cs.updateTaskStats(cronTaskID, false)
		return
	}

	logx.Infow("Created InputTask for cron task",
		logx.Field("cron_task_id", cronTaskID),
		logx.Field("input_task_id", inputTask.ID),
		logx.Field("execution_time", executionTime.Format(time.RFC3339)))

	// 更新 CronTask 统计信息
	cs.updateTaskStats(cronTaskID, true)

	// 更新 last_run_time 和 next_run_time
	cs.updateRunTimes(cronTaskID)
}

// updateTaskStats updates execution statistics for a cron task
// 更新 Cron 任务的执行统计
func (cs *CronScheduler) updateTaskStats(cronTaskID uint64, success bool) {
	systemCtx := hooks.NewSystemContext(cs.ctx)

	cronTask, err := cs.db.CronTask.Get(systemCtx, cronTaskID)
	if err != nil {
		logx.Errorw("Failed to get cron task for stats update",
			logx.Field("cron_task_id", cronTaskID),
			logx.Field("error", err))
		return
	}

	update := cs.db.CronTask.UpdateOneID(cronTaskID).
		SetExecutionCount(cronTask.ExecutionCount + 1)

	if success {
		update = update.SetSuccessCount(cronTask.SuccessCount + 1)
	} else {
		update = update.SetFailureCount(cronTask.FailureCount + 1)
	}

	_, err = update.Save(systemCtx)
	if err != nil {
		logx.Errorw("Failed to update cron task stats",
			logx.Field("cron_task_id", cronTaskID),
			logx.Field("error", err))
	}
}

// updateRunTimes updates last_run_time and next_run_time for a cron task
// 更新 Cron 任务的 last_run_time 和 next_run_time
func (cs *CronScheduler) updateRunTimes(cronTaskID uint64) {
	cs.taskMapLock.RLock()
	entryID, exists := cs.taskMap[cronTaskID]
	cs.taskMapLock.RUnlock()

	if !exists {
		logx.Infow("Cron task not found in taskMap",
			logx.Field("cron_task_id", cronTaskID))
		return
	}

	entry := cs.cron.Entry(entryID)
	nextRunTime := entry.Next
	lastRunTime := time.Now()

	systemCtx := hooks.NewSystemContext(cs.ctx)
	_, err := cs.db.CronTask.UpdateOneID(cronTaskID).
		SetLastRunTime(lastRunTime).
		SetNextRunTime(nextRunTime).
		Save(systemCtx)

	if err != nil {
		logx.Errorw("Failed to update run times",
			logx.Field("cron_task_id", cronTaskID),
			logx.Field("error", err))
	}
}

// GetTaskCount returns the number of tasks currently in the scheduler
// 获取调度器中的任务数量
func (cs *CronScheduler) GetTaskCount() int {
	cs.taskMapLock.RLock()
	defer cs.taskMapLock.RUnlock()
	return len(cs.taskMap)
}

// IsTaskScheduled checks if a task is currently scheduled
// 检查任务是否已在调度器中
func (cs *CronScheduler) IsTaskScheduled(taskID uint64) bool {
	cs.taskMapLock.RLock()
	defer cs.taskMapLock.RUnlock()
	_, exists := cs.taskMap[taskID]
	return exists
}
