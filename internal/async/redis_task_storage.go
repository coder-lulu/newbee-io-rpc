package async

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

// TaskStats 任务统计信息
type TaskStats struct {
	TotalTasks      int `json:"total_tasks"`
	PendingTasks    int `json:"pending_tasks"`
	ProcessingTasks int `json:"processing_tasks"`
	CompletedTasks  int `json:"completed_tasks"`
	FailedTasks     int `json:"failed_tasks"`
	ActiveWorkers   int `json:"active_workers"`
}

// RedisTaskStorage Redis任务存储实现
type RedisTaskStorage struct {
	client    redis.UniversalClient
	logger    logx.Logger
	keyPrefix string
	ttl       time.Duration
}

// NewRedisTaskStorage 创建Redis任务存储
func NewRedisTaskStorage(client redis.UniversalClient) *RedisTaskStorage {
	return &RedisTaskStorage{
		client:    client,
		logger:    logx.WithContext(context.Background()),
		keyPrefix: "cmdb:task:",
		ttl:       7 * 24 * time.Hour, // 任务保留7天
	}
}

// SaveTask 保存任务到Redis
func (r *RedisTaskStorage) SaveTask(ctx context.Context, task *ImportTask) error {
	key := r.buildKey(task.ID)

	// 序列化任务数据
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("序列化任务失败: %v", err)
	}

	// 保存到Redis，设置TTL
	err = r.client.Set(ctx, key, data, r.ttl).Err()
	if err != nil {
		return fmt.Errorf("保存任务到Redis失败: %v", err)
	}

	// 添加到任务索引(用于查询和恢复)
	indexKey := r.buildIndexKey()
	err = r.client.SAdd(ctx, indexKey, task.ID).Err()
	if err != nil {
		r.logger.Errorf("添加任务索引失败: %v", err)
	}

	// 根据状态添加到不同的状态集合
	statusKey := r.buildStatusKey(task.Status)
	err = r.client.SAdd(ctx, statusKey, task.ID).Err()
	if err != nil {
		r.logger.Errorf("添加任务状态索引失败: %v", err)
	}

	r.logger.Infof("任务已保存到Redis: TaskID=%s, Status=%s", task.ID, task.Status)
	return nil
}

// GetTask 从Redis获取任务
func (r *RedisTaskStorage) GetTask(ctx context.Context, taskID string) (*ImportTask, error) {
	key := r.buildKey(taskID)

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("任务不存在: %s", taskID)
		}
		return nil, fmt.Errorf("从Redis获取任务失败: %v", err)
	}

	var task ImportTask
	err = json.Unmarshal([]byte(data), &task)
	if err != nil {
		return nil, fmt.Errorf("反序列化任务失败: %v", err)
	}

	return &task, nil
}

// UpdateTask 更新任务状态
func (r *RedisTaskStorage) UpdateTask(ctx context.Context, task *ImportTask) error {
	// 获取原任务状态用于索引更新
	oldTask, err := r.GetTask(ctx, task.ID)
	if err != nil {
		r.logger.Errorf("获取原任务状态失败: %v", err)
	}

	// 更新任务数据
	task.UpdateTime = time.Now()
	err = r.SaveTask(ctx, task)
	if err != nil {
		return err
	}

	// 更新状态索引
	if oldTask != nil && oldTask.Status != task.Status {
		// 从旧状态集合中移除
		oldStatusKey := r.buildStatusKey(oldTask.Status)
		r.client.SRem(ctx, oldStatusKey, task.ID)

		r.logger.Infof("任务状态已更新: TaskID=%s, %s -> %s", task.ID, oldTask.Status, task.Status)
	}

	return nil
}

// DeleteTask 删除任务
func (r *RedisTaskStorage) DeleteTask(ctx context.Context, taskID string) error {
	// 获取任务信息用于清理索引
	task, err := r.GetTask(ctx, taskID)
	if err != nil {
		r.logger.Errorf("获取任务信息失败: %v", err)
	}

	key := r.buildKey(taskID)

	// 删除任务数据
	err = r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("删除任务失败: %v", err)
	}

	// 清理索引
	indexKey := r.buildIndexKey()
	r.client.SRem(ctx, indexKey, taskID)

	if task != nil {
		statusKey := r.buildStatusKey(task.Status)
		r.client.SRem(ctx, statusKey, taskID)
	}

	r.logger.Infof("任务已删除: TaskID=%s", taskID)
	return nil
}

// GetTasksByStatus 根据状态获取任务列表
func (r *RedisTaskStorage) GetTasksByStatus(ctx context.Context, status string) ([]*ImportTask, error) {
	statusKey := r.buildStatusKey(status)

	taskIDs, err := r.client.SMembers(ctx, statusKey).Result()
	if err != nil {
		return nil, fmt.Errorf("获取状态任务列表失败: %v", err)
	}

	var tasks []*ImportTask
	for _, taskID := range taskIDs {
		task, err := r.GetTask(ctx, taskID)
		if err != nil {
			r.logger.Errorf("获取任务失败: TaskID=%s, Error=%v", taskID, err)
			continue
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetAllTasks 获取所有任务
func (r *RedisTaskStorage) GetAllTasks(ctx context.Context) ([]*ImportTask, error) {
	indexKey := r.buildIndexKey()

	taskIDs, err := r.client.SMembers(ctx, indexKey).Result()
	if err != nil {
		return nil, fmt.Errorf("获取任务索引失败: %v", err)
	}

	var tasks []*ImportTask
	for _, taskID := range taskIDs {
		task, err := r.GetTask(ctx, taskID)
		if err != nil {
			r.logger.Errorf("获取任务失败: TaskID=%s, Error=%v", taskID, err)
			continue
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// RecoverPendingTasks 恢复待处理任务
func (r *RedisTaskStorage) RecoverPendingTasks(ctx context.Context) ([]*ImportTask, error) {
	r.logger.Info("开始恢复待处理任务...")

	// 获取所有pending和processing状态的任务
	pendingTasks, err := r.GetTasksByStatus(ctx, "pending")
	if err != nil {
		return nil, fmt.Errorf("获取pending任务失败: %v", err)
	}

	processingTasks, err := r.GetTasksByStatus(ctx, "processing")
	if err != nil {
		return nil, fmt.Errorf("获取processing任务失败: %v", err)
	}

	// 将processing任务重置为pending状态 (系统重启后需要重新处理)
	for _, task := range processingTasks {
		task.Status = "pending"
		task.UpdateTime = time.Now()
		err = r.UpdateTask(ctx, task)
		if err != nil {
			r.logger.Errorf("重置任务状态失败: TaskID=%s, Error=%v", task.ID, err)
			continue
		}
		r.logger.Infof("任务状态已重置: TaskID=%s, processing -> pending", task.ID)
	}

	// 合并所有需要恢复的任务
	allRecoverTasks := append(pendingTasks, processingTasks...)

	r.logger.Infof("任务恢复完成，待处理任务数: %d", len(allRecoverTasks))
	return allRecoverTasks, nil
}

// GetTaskStats 获取任务统计信息
func (r *RedisTaskStorage) GetTaskStats(ctx context.Context) (*TaskStats, error) {
	stats := &TaskStats{}

	// 统计各状态任务数量
	statuses := []string{"pending", "processing", "completed", "failed", "cancelled"}

	for _, status := range statuses {
		statusKey := r.buildStatusKey(status)
		count, err := r.client.SCard(ctx, statusKey).Result()
		if err != nil {
			r.logger.Errorf("统计状态任务失败: Status=%s, Error=%v", status, err)
			continue
		}

		switch status {
		case "pending":
			stats.PendingTasks = int(count)
		case "processing":
			stats.ProcessingTasks = int(count)
		case "completed":
			stats.CompletedTasks = int(count)
		case "failed":
			stats.FailedTasks = int(count)
		}
	}

	stats.TotalTasks = stats.PendingTasks + stats.ProcessingTasks + stats.CompletedTasks + stats.FailedTasks
	return stats, nil
}

// CleanupExpiredTasks 清理过期任务
func (r *RedisTaskStorage) CleanupExpiredTasks(ctx context.Context) error {
	r.logger.Info("开始清理过期任务...")

	// 获取已完成和失败的任务
	completedTasks, _ := r.GetTasksByStatus(ctx, "completed")
	failedTasks, _ := r.GetTasksByStatus(ctx, "failed")

	allFinishedTasks := append(completedTasks, failedTasks...)

	// 清理7天前的任务
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	deletedCount := 0

	for _, task := range allFinishedTasks {
		if task.UpdateTime.Before(cutoff) {
			err := r.DeleteTask(ctx, task.ID)
			if err != nil {
				r.logger.Errorf("删除过期任务失败: TaskID=%s, Error=%v", task.ID, err)
				continue
			}
			deletedCount++
		}
	}

	r.logger.Infof("过期任务清理完成，删除任务数: %d", deletedCount)
	return nil
}

// 构建Redis key
func (r *RedisTaskStorage) buildKey(taskID string) string {
	return r.keyPrefix + taskID
}

func (r *RedisTaskStorage) buildIndexKey() string {
	return r.keyPrefix + "index"
}

func (r *RedisTaskStorage) buildStatusKey(status string) string {
	return r.keyPrefix + "status:" + status
}
