package async

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// ImportTask 导入任务定义
type ImportTask struct {
	ID         string        `json:"id"`
	Type       string        `json:"type"`
	Status     string        `json:"status"`
	Progress   *TaskProgress `json:"progress"`
	CreateTime time.Time     `json:"create_time"`
	UpdateTime time.Time     `json:"update_time"`
	Error      string        `json:"error,omitempty"`
}

// TaskProgress 任务进度
type TaskProgress struct {
	TotalItems     int     `json:"total_items"`
	ProcessedItems int     `json:"processed_items"`
	SuccessItems   int     `json:"success_items"`
	FailedItems    int     `json:"failed_items"`
	CurrentStep    string  `json:"current_step"`
	Percentage     float64 `json:"percentage"`
}

// 异步任务管理器 (已废弃，使用EnhancedAsyncTaskManager)
type AsyncTaskManager struct {
	taskQueue   chan *ImportTask
	taskStorage *RedisTaskStorage // 使用Redis持久化存储
	mutex       sync.RWMutex
	logger      logx.Logger
}

// SubmitBatchTask 提交批量任务 (兼容性方法)
func (m *AsyncTaskManager) SubmitBatchTask(ctx context.Context, operation string, ciIds []uint64, params *string) (*ImportTask, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	task := &ImportTask{
		ID:         generateTaskID(),
		Type:       "batch_operation",
		Status:     "pending",
		Progress:   &TaskProgress{TotalItems: len(ciIds)},
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}

	// 使用Redis存储
	err := m.taskStorage.SaveTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("保存任务到Redis失败: %v", err)
	}

	m.logger.Infof("批量任务已提交: TaskID=%s, Operation=%s, CIs=%d", task.ID, operation, len(ciIds))
	return task, nil
}

// GetTaskStatus 获取任务状态 (兼容性方法)
func (m *AsyncTaskManager) GetTaskStatus(ctx context.Context, taskID string) (*ImportTask, error) {
	// 从Redis获取任务
	task, err := m.taskStorage.GetTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("获取任务失败: %v", err)
	}
	return task, nil
}

// generateTaskID 生成任务ID
func generateTaskID() string {
	return fmt.Sprintf("task_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}
