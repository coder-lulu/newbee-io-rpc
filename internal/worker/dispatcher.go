package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/ent/inputtask"
	"github.com/coder-lulu/newbee-io-rpc/ent/outputtask"
	"github.com/zeromicro/go-zero/core/logx"
)

// Dispatcher 任务分发器
// 职责:
// 1. 从数据库读取待执行任务(Pending状态)
// 2. 验证任务参数和租户隔离
// 3. 构建ExecutionContext并调用Executor
// 4. 支持定时轮询和手动触发两种模式
type Dispatcher struct {
	config         *DispatcherConfig
	db             *ent.Client
	executor       *Executor
	stateMachine   *StateMachine
	contextManager *ContextManager
	logger         logx.Logger

	// 运行时控制
	stopChan chan struct{}
	wg       sync.WaitGroup
	running  bool
	mu       sync.RWMutex
}

// NewDispatcher 创建新的任务分发器
func NewDispatcher(
	config *DispatcherConfig,
	db *ent.Client,
	executor *Executor,
	stateMachine *StateMachine,
	contextManager *ContextManager,
	logger logx.Logger,
) *Dispatcher {
	if config == nil {
		config = DefaultDispatcherConfig()
	}

	return &Dispatcher{
		config:         config,
		db:             db,
		executor:       executor,
		stateMachine:   stateMachine,
		contextManager: contextManager,
		logger:         logger,
		stopChan:       make(chan struct{}),
	}
}

// DispatchInputTask 分发单个输入任务(手动触发)
func (d *Dispatcher) DispatchInputTask(ctx context.Context, taskID uint64) error {
	d.logger.Infow("Dispatching input task",
		logx.Field("task_id", taskID))

	// 1. 读取任务
	task, err := d.db.InputTask.Query().
		Where(inputtask.IDEQ(taskID)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("failed to query input task %d: %w", taskID, err)
	}

	// 2. 验证任务状态
	if err := d.validateTaskStatus(TaskStatus(task.TaskStatus), TaskTypeInput, taskID); err != nil {
		return err
	}

	// 3. 验证任务配置
	if err := d.validateTaskConfig(task.InputSource); err != nil {
		d.logger.Errorw("Invalid task configuration",
			logx.Field("task_id", taskID),
			logx.Field("input_source", task.InputSource),
			logx.Field("error", err))
		return err
	}

	// 4. 构建执行上下文
	execCtx, err := d.contextManager.BuildExecutionContext(ctx, task, TaskTypeInput, d.db)
	if err != nil {
		return fmt.Errorf("failed to build execution context: %w", err)
	}

	// 5. 执行任务
	d.logger.Infow("Executing input task",
		logx.Field("task_id", taskID),
		logx.Field("tenant_id", execCtx.TenantID),
		logx.Field("provider_id", execCtx.ProviderID))

	return d.executor.ExecuteWithStateUpdate(execCtx)
}

// DispatchOutputTask 分发单个输出任务(手动触发)
func (d *Dispatcher) DispatchOutputTask(ctx context.Context, taskID uint64) error {
	d.logger.Infow("Dispatching output task",
		logx.Field("task_id", taskID))

	// 1. 读取任务
	task, err := d.db.OutputTask.Query().
		Where(outputtask.IDEQ(taskID)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("failed to query output task %d: %w", taskID, err)
	}

	// 2. 验证任务状态
	if err := d.validateTaskStatus(TaskStatus(task.TaskStatus), TaskTypeOutput, taskID); err != nil {
		return err
	}

	// 3. 验证任务配置
	if err := d.validateTaskConfig(task.OutputTarget); err != nil {
		d.logger.Errorw("Invalid task configuration",
			logx.Field("task_id", taskID),
			logx.Field("output_target", task.OutputTarget),
			logx.Field("error", err))
		return err
	}

	// 4. 构建执行上下文
	execCtx, err := d.contextManager.BuildExecutionContext(ctx, task, TaskTypeOutput, d.db)
	if err != nil {
		return fmt.Errorf("failed to build execution context: %w", err)
	}

	// 5. 执行任务
	d.logger.Infow("Executing output task",
		logx.Field("task_id", taskID),
		logx.Field("tenant_id", execCtx.TenantID),
		logx.Field("target_id", execCtx.ProviderID))

	return d.executor.ExecuteWithStateUpdate(execCtx)
}

// DispatchPendingTasks 分发所有待执行任务(批量处理)
func (d *Dispatcher) DispatchPendingTasks(ctx context.Context) error {
	d.logger.Infow("Dispatching pending tasks",
		logx.Field("batch_size", d.config.BatchSize))

	// 1. 获取待执行的输入任务
	inputTasks, err := d.stateMachine.GetPendingInputTasks(ctx, d.config.BatchSize)
	if err != nil {
		d.logger.Errorw("Failed to query pending input tasks", logx.Field("error", err))
		return fmt.Errorf("failed to query pending input tasks: %w", err)
	}

	// 2. 获取待执行的输出任务
	outputTasks, err := d.stateMachine.GetPendingOutputTasks(ctx, d.config.BatchSize)
	if err != nil {
		d.logger.Errorw("Failed to query pending output tasks", logx.Field("error", err))
		return fmt.Errorf("failed to query pending output tasks: %w", err)
	}

	totalTasks := len(inputTasks) + len(outputTasks)
	if totalTasks == 0 {
		d.logger.Infow("No pending tasks to dispatch")
		return nil
	}

	d.logger.Infow("Found pending tasks",
		logx.Field("input_tasks", len(inputTasks)),
		logx.Field("output_tasks", len(outputTasks)),
		logx.Field("total", totalTasks))

	// 3. 分发输入任务
	for _, task := range inputTasks {
		if err := d.DispatchInputTask(ctx, task.ID); err != nil {
			d.logger.Errorw("Failed to dispatch input task",
				logx.Field("task_id", task.ID),
				logx.Field("error", err))
			// 继续处理其他任务,不中断
			continue
		}
	}

	// 4. 分发输出任务
	for _, task := range outputTasks {
		if err := d.DispatchOutputTask(ctx, task.ID); err != nil {
			d.logger.Errorw("Failed to dispatch output task",
				logx.Field("task_id", task.ID),
				logx.Field("error", err))
			// 继续处理其他任务,不中断
			continue
		}
	}

	d.logger.Infow("Finished dispatching pending tasks",
		logx.Field("processed", totalTasks))

	return nil
}

// Start 启动定时轮询模式
// 每隔 PollInterval 自动扫描并执行待执行任务
// 使用 ctx 控制生命周期: ctx.Done() 时停止轮询
func (d *Dispatcher) Start(ctx context.Context) error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return fmt.Errorf("dispatcher already running")
	}
	d.running = true
	d.mu.Unlock()

	d.logger.Infow("Starting dispatcher",
		logx.Field("polling_interval", d.config.PollInterval),
		logx.Field("batch_size", d.config.BatchSize))

	d.wg.Add(1)
	go d.runPollingLoop(ctx)

	return nil
}

// Stop 停止轮询
func (d *Dispatcher) Stop() error {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return fmt.Errorf("dispatcher not running")
	}
	d.running = false
	d.mu.Unlock()

	d.logger.Infow("Stopping dispatcher...")
	close(d.stopChan)
	d.wg.Wait()
	d.logger.Infow("Dispatcher stopped")

	return nil
}

// IsRunning 检查是否正在运行
func (d *Dispatcher) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

// runPollingLoop 轮询循环(内部方法)
func (d *Dispatcher) runPollingLoop(ctx context.Context) {
	defer d.wg.Done()

	ticker := time.NewTicker(d.config.PollInterval)
	defer ticker.Stop()

	d.logger.Infow("Polling loop started")

	for {
		select {
		case <-ctx.Done():
			d.logger.Infow("Context cancelled, stopping polling loop")
			return

		case <-d.stopChan:
			d.logger.Infow("Stop signal received, stopping polling loop")
			return

		case <-ticker.C:
			d.logger.Infow("Polling tick - checking for pending tasks")

			// 使用新的context执行任务,避免ctx.Done()影响当前批次
			execCtx, cancel := context.WithTimeout(context.Background(), d.config.PollInterval)
			if err := d.DispatchPendingTasks(execCtx); err != nil {
				d.logger.Errorw("Failed to dispatch pending tasks",
					logx.Field("error", err))
			}
			cancel()
		}
	}
}

// validateTaskStatus 验证任务状态
func (d *Dispatcher) validateTaskStatus(status TaskStatus, taskType TaskType, taskID uint64) error {
	if status != TaskStatusPending {
		return fmt.Errorf(
			"task %d (type=%s) has invalid status for dispatch: %s (expected: %s)",
			taskID, taskType, status, TaskStatusPending,
		)
	}
	return nil
}

// validateTaskConfig 验证任务配置
func (d *Dispatcher) validateTaskConfig(sourceOrTarget string) error {
	if sourceOrTarget == "" {
		return fmt.Errorf("input_source or output_target is required")
	}

	// TODO: 后续可以添加更复杂的配置验证
	// 例如: 验证Provider/Target是否存在, 验证config参数完整性

	return nil
}
