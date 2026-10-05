package worker

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/coder-lulu/newbee-common/v2/orm/ent/hooks"
	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/ent/inputtask"
	"github.com/coder-lulu/newbee-io-rpc/internal/lock"
	"github.com/zeromicro/go-zero/core/logx"
)

// TaskWorker 任务拉取和处理Worker
type TaskWorker struct {
	db           *ent.Client
	config       *WorkerConfig
	metrics      *TaskWorkerMetrics
	dedupManager *TaskDedupManager // 🔒 任务去重管理器（防止多实例重复处理）
	running      bool
	mu           sync.RWMutex
	cancelCtx    context.CancelFunc
}

// WorkerConfig Worker配置
type WorkerConfig struct {
	// PullInterval 任务拉取间隔
	PullInterval time.Duration
	// BatchSize 每次拉取的任务数量
	BatchSize int
	// MaxConcurrent 最大并发处理任务数
	MaxConcurrent int
	// TaskTimeout 单个任务超时时间
	TaskTimeout time.Duration
	// StaleThreshold 任务被视为stale的时间阈值
	StaleThreshold time.Duration
	// StaleCheckInterval stale任务检查间隔
	StaleCheckInterval time.Duration
}

// DefaultWorkerConfig 返回默认Worker配置
func DefaultWorkerConfig() *WorkerConfig {
	return &WorkerConfig{
		PullInterval:       10 * time.Second, // 每10秒拉取一次
		BatchSize:          10,                // 每次拉取10个任务
		MaxConcurrent:      5,                 // 最大并发5个任务
		TaskTimeout:        5 * time.Minute,   // 单个任务5分钟超时
		StaleThreshold:     1 * time.Hour,     // 1小时未处理视为stale
		StaleCheckInterval: 5 * time.Minute,   // 每5分钟检查一次stale任务
	}
}

// TaskWorkerMetrics TaskWorker性能指标
type TaskWorkerMetrics struct {
	TotalPulled      int64     // 总拉取任务数
	TotalProcessed   int64     // 总处理任务数
	TotalSucceeded   int64     // 总成功任务数
	TotalFailed      int64     // 总失败任务数
	CurrentProcessing int64     // 当前正在处理的任务数
	LastPullTime     time.Time // 上次拉取时间
	LastCheckTime    time.Time // 上次检查stale任务时间
	mu               sync.RWMutex
}

// NewTaskWorker 创建TaskWorker
//
// 参数:
//   - db: 数据库客户端
//   - config: Worker配置
//   - lockManager: 分布式锁管理器（用于任务去重，可选）
func NewTaskWorker(db *ent.Client, config *WorkerConfig, lockManager *lock.DistributedLockManager) *TaskWorker {
	if config == nil {
		config = DefaultWorkerConfig()
	}

	// 验证并修正配置参数（但允许测试环境使用小值）
	if config.PullInterval < 10*time.Millisecond && config.PullInterval > 0 {
		logx.Infow("PullInterval very small, might be for testing",
			logx.Field("value", config.PullInterval))
	} else if config.PullInterval <= 0 {
		logx.Infow("PullInterval invalid, using 10s",
			logx.Field("original", config.PullInterval),
			logx.Field("corrected", 10*time.Second))
		config.PullInterval = 10 * time.Second
	}

	if config.MaxConcurrent < 1 {
		logx.Infow("MaxConcurrent too small, using 1",
			logx.Field("original", config.MaxConcurrent),
			logx.Field("corrected", 1))
		config.MaxConcurrent = 1
	}

	if config.BatchSize < 1 {
		logx.Infow("BatchSize too small, using 1",
			logx.Field("original", config.BatchSize),
			logx.Field("corrected", 1))
		config.BatchSize = 1
	}

	if config.TaskTimeout < 10*time.Millisecond && config.TaskTimeout > 0 {
		logx.Infow("TaskTimeout very small, might be for testing",
			logx.Field("value", config.TaskTimeout))
	} else if config.TaskTimeout <= 0 {
		logx.Infow("TaskTimeout invalid, using 5m",
			logx.Field("original", config.TaskTimeout),
			logx.Field("corrected", 5*time.Minute))
		config.TaskTimeout = 5 * time.Minute
	}

	if config.StaleThreshold < 10*time.Millisecond && config.StaleThreshold > 0 {
		logx.Infow("StaleThreshold very small, might be for testing",
			logx.Field("value", config.StaleThreshold))
	} else if config.StaleThreshold <= 0 {
		logx.Infow("StaleThreshold invalid, using 1h",
			logx.Field("original", config.StaleThreshold),
			logx.Field("corrected", time.Hour))
		config.StaleThreshold = time.Hour
	}

	if config.StaleCheckInterval < 10*time.Millisecond && config.StaleCheckInterval > 0 {
		logx.Infow("StaleCheckInterval very small, might be for testing",
			logx.Field("value", config.StaleCheckInterval))
	} else if config.StaleCheckInterval <= 0 {
		logx.Infow("StaleCheckInterval invalid, using 5m",
			logx.Field("original", config.StaleCheckInterval),
			logx.Field("corrected", 5*time.Minute))
		config.StaleCheckInterval = 5 * time.Minute
	}

	// 🔒 创建任务去重管理器（防止多实例重复处理）
	var dedupManager *TaskDedupManager
	if lockManager != nil {
		dedupManager = NewTaskDedupManager(lockManager)
		logx.Info("TaskWorker: Task deduplication enabled with distributed lock")
	} else {
		logx.Info("TaskWorker: No lockManager provided, task deduplication disabled")
	}

	return &TaskWorker{
		db:           db,
		config:       config,
		metrics:      &TaskWorkerMetrics{},
		dedupManager: dedupManager,
		running:      false,
	}
}

// Start 启动Worker
func (w *TaskWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("worker already running")
	}
	w.running = true
	w.mu.Unlock()

	// 创建可取消的上下文
	workerCtx, cancel := context.WithCancel(ctx)
	w.cancelCtx = cancel

	logx.Infow("TaskWorker starting",
		logx.Field("pull_interval", w.config.PullInterval),
		logx.Field("batch_size", w.config.BatchSize),
		logx.Field("max_concurrent", w.config.MaxConcurrent))

	// 启动任务拉取goroutine
	go w.pullLoop(workerCtx)

	// 启动stale任务检查goroutine
	go w.staleCheckLoop(workerCtx)

	logx.Info("TaskWorker started successfully")
	return nil
}

// Stop 停止Worker
func (w *TaskWorker) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return fmt.Errorf("worker not running")
	}

	logx.Info("TaskWorker stopping...")

	if w.cancelCtx != nil {
		w.cancelCtx()
	}

	w.running = false
	logx.Info("TaskWorker stopped")
	return nil
}

// IsRunning 检查Worker是否在运行
func (w *TaskWorker) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// GetMetrics 获取性能指标
func (w *TaskWorker) GetMetrics() *TaskWorkerMetrics {
	w.metrics.mu.RLock()
	defer w.metrics.mu.RUnlock()

	return &TaskWorkerMetrics{
		TotalPulled:      w.metrics.TotalPulled,
		TotalProcessed:   w.metrics.TotalProcessed,
		TotalSucceeded:   w.metrics.TotalSucceeded,
		TotalFailed:      w.metrics.TotalFailed,
		CurrentProcessing: w.metrics.CurrentProcessing,
		LastPullTime:     w.metrics.LastPullTime,
		LastCheckTime:    w.metrics.LastCheckTime,
	}
}

// pullLoop 任务拉取循环
func (w *TaskWorker) pullLoop(ctx context.Context) {
	ticker := time.NewTicker(w.config.PullInterval)
	defer ticker.Stop()

	// ⭐ 新增：定时任务检查ticker（每分钟检查一次）
	scheduledTicker := time.NewTicker(1 * time.Minute)
	defer scheduledTicker.Stop()

	// 立即执行一次
	w.processOnce(ctx)

	for {
		select {
		case <-ticker.C:
			w.processOnce(ctx)

		// ⭐ 新增：定时任务检查分支
		case <-scheduledTicker.C:
			w.processScheduledTasks(ctx)

		case <-ctx.Done():
			logx.Info("Pull loop stopped")
			return
		}
	}
}

// staleCheckLoop stale任务检查循环
func (w *TaskWorker) staleCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(w.config.StaleCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.checkStaleTasks(ctx)
		case <-ctx.Done():
			logx.Info("Stale check loop stopped")
			return
		}
	}
}

// processOnce 执行一次任务拉取和分发
func (w *TaskWorker) processOnce(ctx context.Context) {
	// 检查当前并发数
	w.metrics.mu.RLock()
	currentProcessing := w.metrics.CurrentProcessing
	w.metrics.mu.RUnlock()

	if currentProcessing >= int64(w.config.MaxConcurrent) {
		logx.Debugf("Max concurrent tasks reached (%d), skipping pull", w.config.MaxConcurrent)
		return
	}

	// 计算可拉取的任务数
	availableSlots := int(int64(w.config.MaxConcurrent) - currentProcessing)
	pullSize := w.config.BatchSize
	if pullSize > availableSlots {
		pullSize = availableSlots
	}

	// 🔥 使用SystemContext查询所有租户的pending任务
	// Worker是后台进程，需要处理所有租户的任务，因此绕过租户隔离
	systemCtx := hooks.NewSystemContext(ctx)

	// ⭐ 查询pending任务（排除scheduled类型，scheduled任务由processScheduledTasks处理）
	tasks, err := w.db.InputTask.Query().
		Where(
			inputtask.TaskStatusEQ("pending"),
			inputtask.TaskTypeNEQ("scheduled"), // 排除scheduled类型
		).
		Order(ent.Asc(inputtask.FieldCreatedAt)).
		Limit(pullSize).
		All(systemCtx)

	if err != nil {
		logx.Errorw("Failed to query pending tasks",
			logx.Field("error", err))
		return
	}

	if len(tasks) == 0 {
		logx.Debug("No pending tasks found")
		return
	}

	// 更新指标
	w.metrics.mu.Lock()
	w.metrics.TotalPulled += int64(len(tasks))
	w.metrics.LastPullTime = time.Now()
	w.metrics.mu.Unlock()

	logx.Infow("Pulled pending tasks (manual/triggered)",
		logx.Field("count", len(tasks)),
		logx.Field("current_processing", currentProcessing))

	// 分发任务到goroutine处理
	for _, task := range tasks {
		w.incrementProcessing()
		go w.processTask(ctx, task)
	}
}

// processScheduledTasks 处理到期的定时任务
func (w *TaskWorker) processScheduledTasks(ctx context.Context) {
	// 检查当前并发数（复用现有逻辑）
	w.metrics.mu.RLock()
	currentProcessing := w.metrics.CurrentProcessing
	w.metrics.mu.RUnlock()

	if currentProcessing >= int64(w.config.MaxConcurrent) {
		logx.Debugf("Max concurrent tasks reached (%d), skipping scheduled pull", w.config.MaxConcurrent)
		return
	}

	// 计算可拉取的任务数
	availableSlots := int(int64(w.config.MaxConcurrent) - currentProcessing)
	pullSize := w.config.BatchSize
	if pullSize > availableSlots {
		pullSize = availableSlots
	}

	// 🔥 使用SystemContext查询到期的定时任务（绕过租户隔离，因为Worker是后台进程）
	systemCtx := hooks.NewSystemContext(ctx)

	// ⭐ 查询到期的scheduled任务
	tasks, err := w.db.InputTask.Query().
		Where(
			inputtask.TaskTypeEQ("scheduled"),           // 任务类型为scheduled
			inputtask.TaskStatusEQ("pending"),           // 状态为pending
			inputtask.ScheduledAtNotNil(),               // scheduled_at不为空
			inputtask.ScheduledAtLTE(time.Now()),        // 到期时间 <= 当前时间
		).
		Order(ent.Asc(inputtask.FieldScheduledAt)).      // 按计划时间排序
		Limit(pullSize).
		All(systemCtx)

	if err != nil {
		logx.Errorw("Failed to query scheduled tasks",
			logx.Field("error", err))
		return
	}

	if len(tasks) == 0 {
		logx.Debug("No scheduled tasks due")
		return
	}

	// 更新指标（复用现有metrics）
	w.metrics.mu.Lock()
	w.metrics.TotalPulled += int64(len(tasks))
	w.metrics.LastPullTime = time.Now()
	w.metrics.mu.Unlock()

	logx.Infow("Pulled scheduled tasks",
		logx.Field("count", len(tasks)),
		logx.Field("current_processing", currentProcessing))

	// ⭐ 分发任务（复用现有的processTask逻辑，无需修改）
	for _, task := range tasks {
		w.incrementProcessing()
		go w.processTask(ctx, task)  // 复用现有的任务处理流程
	}
}

// processTask 处理单个任务
func (w *TaskWorker) processTask(ctx context.Context, task *ent.InputTask) {
	defer w.decrementProcessing()

	// 🔒 任务去重检查：尝试获取任务处理锁
	// 在多实例场景下，只有一个Worker能成功获取锁并处理该任务
	if w.dedupManager != nil {
		acquired, unlock, err := w.dedupManager.TryAcquireTaskLock(ctx, task.ID)
		if err != nil {
			logx.Errorw("Failed to acquire task lock",
				logx.Field("task_id", task.ID),
				logx.Field("error", err))
			// 锁获取失败，任务保持pending状态，下次可重试
			return
		}

		if !acquired {
			// 任务已被其他Worker抢占，跳过处理
			logx.Infow("Task already being processed by another worker, skipping",
				logx.Field("task_id", task.ID),
				logx.Field("task_name", task.TaskName))
			return
		}

		// 确保锁在函数返回时释放
		defer unlock()

		logx.Infow("Task lock acquired, proceeding with processing",
			logx.Field("task_id", task.ID))
	}

	// 添加panic恢复机制
	defer func() {
		if r := recover(); r != nil {
			logx.Errorw("Task processing panicked",
				logx.Field("task_id", task.ID),
				logx.Field("task_name", task.TaskName),
				logx.Field("panic", r),
				logx.Field("stack", string(debug.Stack())))

			// 🔥 使用SystemContext更新任务状态为failed
			// Worker是系统级进程，因此使用SystemContext绕过租户检查
			failCtx := hooks.NewSystemContext(context.Background())
			_ = task.Update().
				SetTaskStatus("failed").
				SetCompletedAt(time.Now()).
				SetErrorMessage(fmt.Sprintf("panic: %v", r)).
				Exec(failCtx)

			w.incrementFailed()
		}
	}()

	// 创建带超时的上下文
	taskCtx, cancel := context.WithTimeout(ctx, w.config.TaskTimeout)
	defer cancel()

	logx.Infow("Processing task started",
		logx.Field("task_id", task.ID),
		logx.Field("task_name", task.TaskName),
		logx.Field("task_type", task.TaskType),
		logx.Field("tenant_id", task.TenantID))

	startTime := time.Now()

	// 🔥 使用SystemContext标记任务为processing状态
	// Worker是系统级进程，因此使用SystemContext绕过租户检查
	systemTaskCtx := hooks.NewSystemContext(taskCtx)
	err := task.Update().
		SetTaskStatus("processing").
		SetStartedAt(startTime).
		Exec(systemTaskCtx)

	if err != nil {
		logx.Errorw("Failed to update task status to processing",
			logx.Field("task_id", task.ID),
			logx.Field("error", err))
		// 不增加失败计数，因为任务未真正执行
		// 任务会保持pending状态，下次可能成功获取
		return
	}

	// 执行任务处理逻辑
	err = w.executeTask(taskCtx, task)

	duration := time.Since(startTime)

	if err != nil {
		// 任务失败
		w.handleTaskFailure(systemTaskCtx, task, err, duration)
		return
	}

	// 任务成功
	w.handleTaskSuccess(systemTaskCtx, task, duration)
}

// executeTask 执行任务的实际业务逻辑
func (w *TaskWorker) executeTask(ctx context.Context, task *ent.InputTask) error {
	// TODO: 实现实际的任务处理逻辑
	// 这里根据task.TaskType分发到不同的处理器

	logx.Infow("Executing task business logic",
		logx.Field("task_id", task.ID),
		logx.Field("task_type", task.TaskType))

	// 模拟任务处理
	select {
	case <-time.After(100 * time.Millisecond):
		// 任务处理成功
		return nil
	case <-ctx.Done():
		// 任务超时或取消
		return ctx.Err()
	}
}

// handleTaskSuccess 处理任务成功
func (w *TaskWorker) handleTaskSuccess(ctx context.Context, task *ent.InputTask, duration time.Duration) {
	err := task.Update().
		SetTaskStatus("completed").
		SetCompletedAt(time.Now()).
		Exec(ctx)

	if err != nil {
		logx.Errorw("Failed to update task status to completed",
			logx.Field("task_id", task.ID),
			logx.Field("error", err))
		w.incrementFailed()
		return
	}

	w.incrementSucceeded()

	logx.Infow("Task completed successfully",
		logx.Field("task_id", task.ID),
		logx.Field("duration", duration))
}

// handleTaskFailure 处理任务失败
func (w *TaskWorker) handleTaskFailure(ctx context.Context, task *ent.InputTask, taskErr error, duration time.Duration) {
	errorMsg := taskErr.Error()

	err := task.Update().
		SetTaskStatus("failed").
		SetCompletedAt(time.Now()).
		SetErrorMessage(errorMsg).
		Exec(ctx)

	if err != nil {
		logx.Errorw("Failed to update task status to failed",
			logx.Field("task_id", task.ID),
			logx.Field("error", err))
	}

	w.incrementFailed()

	logx.Errorw("Task failed",
		logx.Field("task_id", task.ID),
		logx.Field("duration", duration),
		logx.Field("error", taskErr))
}

// checkStaleTasks 检查长时间未处理的任务
func (w *TaskWorker) checkStaleTasks(ctx context.Context) {
	threshold := time.Now().Add(-w.config.StaleThreshold)

	// 🔥 使用SystemContext查询所有租户的stale任务
	// Worker是后台进程，需要检查所有租户的任务，因此绕过租户隔离
	systemCtx := hooks.NewSystemContext(ctx)

	staleCount, err := w.db.InputTask.Query().
		Where(
			inputtask.TaskStatusEQ("pending"),
			inputtask.CreatedAtLT(threshold),
		).
		Count(systemCtx)

	if err != nil {
		logx.Errorw("Failed to query stale tasks",
			logx.Field("error", err))
		return
	}

	// 更新检查时间
	w.metrics.mu.Lock()
	w.metrics.LastCheckTime = time.Now()
	w.metrics.mu.Unlock()

	if staleCount > 0 {
		logx.Errorw("Found stale tasks",
			logx.Field("count", staleCount),
			logx.Field("threshold", w.config.StaleThreshold))

		// TODO: 发送告警到监控系统
		// 可以集成Prometheus、钉钉、邮件等告警渠道
	} else {
		logx.Debugf("No stale tasks found")
	}
}

// incrementProcessing 增加正在处理的任务计数
func (w *TaskWorker) incrementProcessing() {
	w.metrics.mu.Lock()
	defer w.metrics.mu.Unlock()
	w.metrics.CurrentProcessing++
}

// decrementProcessing 减少正在处理的任务计数
func (w *TaskWorker) decrementProcessing() {
	w.metrics.mu.Lock()
	defer w.metrics.mu.Unlock()
	w.metrics.CurrentProcessing--
	w.metrics.TotalProcessed++
}

// incrementSucceeded 增加成功任务计数
func (w *TaskWorker) incrementSucceeded() {
	w.metrics.mu.Lock()
	defer w.metrics.mu.Unlock()
	w.metrics.TotalSucceeded++
}

// incrementFailed 增加失败任务计数
func (w *TaskWorker) incrementFailed() {
	w.metrics.mu.Lock()
	defer w.metrics.mu.Unlock()
	w.metrics.TotalFailed++
}
