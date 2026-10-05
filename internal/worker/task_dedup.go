package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/internal/lock"
	"github.com/zeromicro/go-zero/core/logx"
)

// TaskDedupManager 任务去重管理器
type TaskDedupManager struct {
	lockManager *lock.DistributedLockManager
}

// NewTaskDedupManager 创建任务去重管理器
func NewTaskDedupManager(lockManager *lock.DistributedLockManager) *TaskDedupManager {
	return &TaskDedupManager{
		lockManager: lockManager,
	}
}

// ComputeTaskFingerprint 计算任务指纹（用于检测重复任务）
//
// 指纹生成规则：
// - 对于 cron 任务：hash(cron_task_id + execution_time)
// - 对于其他任务：hash(task_name + input_source + source_config + created_at)
func ComputeTaskFingerprint(task *ent.InputTask) string {
	h := sha256.New()

	// 如果是 cron 任务，使用 cron_task_id + execution_time
	if task.CronTaskID != nil && task.ExecutionTime != nil {
		h.Write([]byte(fmt.Sprintf("cron:%d:%d", *task.CronTaskID, task.ExecutionTime.Unix())))
	} else {
		// 其他任务使用 task_name + input_source + source_config + created_at
		h.Write([]byte(task.TaskName))
		h.Write([]byte(task.InputSource))
		if task.SourceConfig != "" {
			h.Write([]byte(task.SourceConfig))
		}
		h.Write([]byte(task.CreatedAt.Format(time.RFC3339)))
	}

	return hex.EncodeToString(h.Sum(nil))[:16] // 取前16位
}

// TryAcquireTaskLock 尝试获取任务处理锁
//
// 返回值:
//   - acquired: 是否成功获取锁
//   - unlock: 解锁函数（如果获取成功）
//   - err: 错误信息
//
// 使用示例:
//   acquired, unlock, err := dedup.TryAcquireTaskLock(ctx, taskID)
//   if !acquired {
//       // 任务已被其他Worker处理，跳过
//       return nil
//   }
//   defer unlock()
//   // 执行任务...
func (tdm *TaskDedupManager) TryAcquireTaskLock(
	ctx context.Context,
	taskID uint64,
) (acquired bool, unlock func(), err error) {
	if tdm.lockManager == nil {
		// 如果没有配置lockManager，返回获取成功（允许单实例测试）
		logx.Infow("TaskDedupManager: lockManager not configured, skipping lock",
			logx.Field("task_id", taskID))
		return true, func() {}, nil
	}

	// 锁的key格式：task:process:{taskID}:lock
	lockKey := fmt.Sprintf("task:process:%d:lock", taskID)

	// 创建锁（单次尝试，不重试）
	taskLock := tdm.lockManager.NewLock(
		lockKey,
		5*time.Minute, // 锁过期时间：5分钟（与TaskTimeout一致）
		1,             // 只尝试1次
		0,             // 不重试
	)

	// 尝试获取锁
	acquired, err = taskLock.TryLock(ctx)
	if err != nil {
		return false, nil, fmt.Errorf("failed to try lock: %w", err)
	}

	if !acquired {
		// 锁已被其他Worker持有
		logx.Infow("Task lock held by another worker, skipping",
			logx.Field("task_id", taskID),
			logx.Field("lock_key", lockKey))
		return false, nil, nil
	}

	// 成功获取锁，返回解锁函数
	unlockFunc := func() {
		if err := taskLock.Unlock(); err != nil {
			logx.Errorw("Failed to release task lock",
				logx.Field("task_id", taskID),
				logx.Field("lock_key", lockKey),
				logx.Field("error", err))
		}
	}

	return true, unlockFunc, nil
}

// ExecuteWithTaskLock 在任务锁保护下执行函数
//
// 这是TryAcquireTaskLock的便捷封装，自动处理锁的获取和释放
//
// 返回值:
//   - executed: 是否执行了函数（false表示被其他Worker抢占）
//   - err: 执行错误
func (tdm *TaskDedupManager) ExecuteWithTaskLock(
	ctx context.Context,
	taskID uint64,
	fn func() error,
) (executed bool, err error) {
	// 尝试获取锁
	acquired, unlock, err := tdm.TryAcquireTaskLock(ctx, taskID)
	if err != nil {
		return false, fmt.Errorf("failed to acquire task lock: %w", err)
	}

	if !acquired {
		// 任务已被其他Worker处理
		return false, nil
	}

	// 确保锁被释放
	defer unlock()

	// 执行业务逻辑
	if err := fn(); err != nil {
		return true, err
	}

	return true, nil
}
