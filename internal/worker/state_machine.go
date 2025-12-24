package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/ent/inputtask"
	"github.com/coder-lulu/newbee-io-rpc/ent/outputtask"
	"github.com/zeromicro/go-zero/core/logx"
)

// StateMachine 任务状态机
type StateMachine struct {
	logger logx.Logger
	db     *ent.Client
}

// NewStateMachine 创建状态机
func NewStateMachine(db *ent.Client, logger logx.Logger) *StateMachine {
	return &StateMachine{
		logger: logger,
		db:     db,
	}
}

// ValidateTransition 验证状态转换是否合法
func (sm *StateMachine) ValidateTransition(from, to TaskStatus) error {
	// 定义合法的状态转换
	validTransitions := map[TaskStatus][]TaskStatus{
		TaskStatusPending: {
			TaskStatusRunning,
			TaskStatusCancelled,
		},
		TaskStatusRunning: {
			TaskStatusCompleted,
			TaskStatusFailed,
			TaskStatusCancelled,
		},
		TaskStatusCompleted: {}, // 终态，不能转换
		TaskStatusFailed: {
			TaskStatusPending, // 允许重试
		},
		TaskStatusCancelled: {}, // 终态，不能转换
	}

	allowedStates, exists := validTransitions[from]
	if !exists {
		return fmt.Errorf("unknown source state: %s", from)
	}

	for _, allowed := range allowedStates {
		if allowed == to {
			return nil // 合法转换
		}
	}

	return fmt.Errorf("invalid state transition: %s -> %s", from, to)
}

// UpdateTaskState 更新任务状态（通用方法）
func (sm *StateMachine) UpdateTaskState(
	ctx context.Context,
	taskID uint64,
	taskType TaskType,
	newStatus TaskStatus,
	result *TaskResult,
) error {
	sm.logger.Infow(
		"Updating task state",
		logx.Field("task_id", taskID),
		logx.Field("task_type", taskType),
		logx.Field("new_status", newStatus),
	)

	switch taskType {
	case TaskTypeInput:
		return sm.updateInputTaskState(ctx, taskID, newStatus, result)
	case TaskTypeOutput:
		return sm.updateOutputTaskState(ctx, taskID, newStatus, result)
	default:
		return fmt.Errorf("unknown task type: %s", taskType)
	}
}

// updateInputTaskState 更新输入任务状态
func (sm *StateMachine) updateInputTaskState(
	ctx context.Context,
	taskID uint64,
	newStatus TaskStatus,
	result *TaskResult,
) error {
	// 开启事务
	tx, err := sm.db.Tx(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if v := recover(); v != nil {
			tx.Rollback()
			panic(v)
		}
	}()

	// 读取当前任务
	task, err := tx.InputTask.Get(ctx, taskID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 验证状态转换
	currentStatus := TaskStatus(task.TaskStatus)
	if err := sm.ValidateTransition(currentStatus, newStatus); err != nil {
		tx.Rollback()
		return err
	}

	// 构建更新操作
	update := tx.InputTask.UpdateOneID(taskID).
		SetTaskStatus(string(newStatus))

	// 根据新状态更新相应字段
	now := time.Now()
	switch newStatus {
	case TaskStatusRunning:
		update = update.SetStartedAt(now)

	case TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled:
		update = update.SetCompletedAt(now)

		if result != nil {
			if result.TotalRecords > 0 {
				update = update.SetTotalRecords(result.TotalRecords)
			}
			if result.ProcessedRecords > 0 {
				update = update.SetProcessedRecords(result.ProcessedRecords)
			}
			if result.SuccessRecords > 0 {
				update = update.SetSuccessRecords(result.SuccessRecords)
			}
			if result.FailedRecords > 0 {
				update = update.SetFailedRecords(result.FailedRecords)
			}
			if result.ErrorMessage != "" {
				update = update.SetErrorMessage(result.ErrorMessage)
			}
		}
	}

	// 执行更新
	if err := update.Exec(ctx); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update task state: %w", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	sm.logger.Infow(
		"Task state updated successfully",
		logx.Field("task_id", taskID),
		logx.Field("old_status", currentStatus),
		logx.Field("new_status", newStatus),
	)

	return nil
}

// updateOutputTaskState 更新输出任务状态
func (sm *StateMachine) updateOutputTaskState(
	ctx context.Context,
	taskID uint64,
	newStatus TaskStatus,
	result *TaskResult,
) error {
	// 开启事务
	tx, err := sm.db.Tx(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if v := recover(); v != nil {
			tx.Rollback()
			panic(v)
		}
	}()

	// 读取当前任务
	task, err := tx.OutputTask.Get(ctx, taskID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 验证状态转换
	currentStatus := TaskStatus(task.TaskStatus)
	if err := sm.ValidateTransition(currentStatus, newStatus); err != nil {
		tx.Rollback()
		return err
	}

	// 构建更新操作
	update := tx.OutputTask.UpdateOneID(taskID).
		SetTaskStatus(string(newStatus))

	// 根据新状态更新相应字段
	now := time.Now()
	switch newStatus {
	case TaskStatusRunning:
		update = update.SetStartedAt(now)

	case TaskStatusCompleted, TaskStatusFailed, TaskStatusCancelled:
		update = update.SetCompletedAt(now)

		if result != nil {
			if result.TotalRecords > 0 {
				update = update.SetTotalRecords(result.TotalRecords)
			}
			if result.ProcessedRecords > 0 {
				update = update.SetProcessedRecords(result.ProcessedRecords)
			}
			if result.SuccessRecords > 0 {
				update = update.SetSuccessRecords(result.SuccessRecords)
			}
			if result.FailedRecords > 0 {
				update = update.SetFailedRecords(result.FailedRecords)
			}
			if result.ErrorMessage != "" {
				update = update.SetErrorMessage(result.ErrorMessage)
			}
		}
	}

	// 执行更新
	if err := update.Exec(ctx); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update task state: %w", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	sm.logger.Infow(
		"Task state updated successfully",
		logx.Field("task_id", taskID),
		logx.Field("old_status", currentStatus),
		logx.Field("new_status", newStatus),
	)

	return nil
}

// GetPendingInputTasks 获取待执行的输入任务
func (sm *StateMachine) GetPendingInputTasks(ctx context.Context, limit int) ([]*ent.InputTask, error) {
	tasks, err := sm.db.InputTask.Query().
		Where(inputtask.TaskStatusEQ(string(TaskStatusPending))).
		// 按调度时间升序（最早需要执行的优先）
		Order(ent.Asc(inputtask.FieldScheduledAt)).
		Limit(limit).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query pending input tasks: %w", err)
	}

	return tasks, nil
}

// GetPendingOutputTasks 获取待执行的输出任务
func (sm *StateMachine) GetPendingOutputTasks(ctx context.Context, limit int) ([]*ent.OutputTask, error) {
	tasks, err := sm.db.OutputTask.Query().
		Where(outputtask.TaskStatusEQ(string(TaskStatusPending))).
		Order(ent.Asc(outputtask.FieldScheduledAt)).
		Limit(limit).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query pending output tasks: %w", err)
	}

	return tasks, nil
}

// MarkTaskAsRunning 标记任务为运行中
func (sm *StateMachine) MarkTaskAsRunning(ctx context.Context, taskID uint64, taskType TaskType) error {
	return sm.UpdateTaskState(ctx, taskID, taskType, TaskStatusRunning, nil)
}

// MarkTaskAsCompleted 标记任务为已完成
func (sm *StateMachine) MarkTaskAsCompleted(ctx context.Context, taskID uint64, taskType TaskType, result *TaskResult) error {
	return sm.UpdateTaskState(ctx, taskID, taskType, TaskStatusCompleted, result)
}

// MarkTaskAsFailed 标记任务为失败
func (sm *StateMachine) MarkTaskAsFailed(ctx context.Context, taskID uint64, taskType TaskType, result *TaskResult) error {
	return sm.UpdateTaskState(ctx, taskID, taskType, TaskStatusFailed, result)
}
