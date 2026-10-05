package worker

import (
	"sync/atomic"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// ConfigUpdateEvent 配置更新事件
type ConfigUpdateEvent struct {
	Key      string        // 配置键
	OldValue interface{}   // 旧值
	NewValue interface{}   // 新值
	Config   *WorkerConfig // 完整的新配置
}

// UpdateConfig 热更新Worker配置（无需重启）
//
// 支持的配置项：
//   - PullInterval: 任务拉取间隔（需要重建ticker）
//   - BatchSize: 批量大小（立即生效）
//   - MaxConcurrent: 最大并发数（立即生效）
//   - TaskTimeout: 任务超时时间（立即生效）
//   - StaleThreshold: Stale阈值（立即生效）
//   - StaleCheckInterval: Stale检查间隔（需要重建ticker）
//
// 注意：
//   - 此方法是线程安全的
//   - 配置更新不会中断正在执行的任务
//   - PullInterval和StaleCheckInterval的变更需要等待当前ticker周期结束
func (w *TaskWorker) UpdateConfig(newConfig *WorkerConfig) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		// Worker未运行，直接更新配置
		w.config = newConfig
		logx.Infow("Worker config updated (worker not running)",
			logx.Field("config", newConfig))
		return nil
	}

	oldConfig := w.config

	// 记录配置变更
	changes := w.detectConfigChanges(oldConfig, newConfig)
	if len(changes) == 0 {
		logx.Info("No config changes detected, skipping update")
		return nil
	}

	// 更新配置（原子操作）
	w.config = newConfig

	// 记录所有变更
	for _, change := range changes {
		logx.Infow("Config changed",
			logx.Field("key", change.Key),
			logx.Field("old", change.OldValue),
			logx.Field("new", change.NewValue))
	}

	logx.Infow("✅ Worker config hot-reloaded successfully",
		logx.Field("changes_count", len(changes)),
		logx.Field("pull_interval", newConfig.PullInterval),
		logx.Field("batch_size", newConfig.BatchSize),
		logx.Field("max_concurrent", newConfig.MaxConcurrent))

	return nil
}

// detectConfigChanges 检测配置变更
func (w *TaskWorker) detectConfigChanges(old, new *WorkerConfig) []ConfigUpdateEvent {
	var changes []ConfigUpdateEvent

	if old.PullInterval != new.PullInterval {
		changes = append(changes, ConfigUpdateEvent{
			Key:      "PullInterval",
			OldValue: old.PullInterval,
			NewValue: new.PullInterval,
			Config:   new,
		})
	}

	if old.BatchSize != new.BatchSize {
		changes = append(changes, ConfigUpdateEvent{
			Key:      "BatchSize",
			OldValue: old.BatchSize,
			NewValue: new.BatchSize,
			Config:   new,
		})
	}

	if old.MaxConcurrent != new.MaxConcurrent {
		changes = append(changes, ConfigUpdateEvent{
			Key:      "MaxConcurrent",
			OldValue: old.MaxConcurrent,
			NewValue: new.MaxConcurrent,
			Config:   new,
		})
	}

	if old.TaskTimeout != new.TaskTimeout {
		changes = append(changes, ConfigUpdateEvent{
			Key:      "TaskTimeout",
			OldValue: old.TaskTimeout,
			NewValue: new.TaskTimeout,
			Config:   new,
		})
	}

	if old.StaleThreshold != new.StaleThreshold {
		changes = append(changes, ConfigUpdateEvent{
			Key:      "StaleThreshold",
			OldValue: old.StaleThreshold,
			NewValue: new.StaleThreshold,
			Config:   new,
		})
	}

	if old.StaleCheckInterval != new.StaleCheckInterval {
		changes = append(changes, ConfigUpdateEvent{
			Key:      "StaleCheckInterval",
			OldValue: old.StaleCheckInterval,
			NewValue: new.StaleCheckInterval,
			Config:   new,
		})
	}

	return changes
}

// GetConfig 获取当前配置（线程安全）
func (w *TaskWorker) GetConfig() *WorkerConfig {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// 返回配置的副本，避免并发修改
	return &WorkerConfig{
		PullInterval:       w.config.PullInterval,
		BatchSize:          w.config.BatchSize,
		MaxConcurrent:      w.config.MaxConcurrent,
		TaskTimeout:        w.config.TaskTimeout,
		StaleThreshold:     w.config.StaleThreshold,
		StaleCheckInterval: w.config.StaleCheckInterval,
	}
}

// UpdatePullInterval 单独更新PullInterval（热重载）
func (w *TaskWorker) UpdatePullInterval(interval time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.config.PullInterval == interval {
		return
	}

	oldValue := w.config.PullInterval
	w.config.PullInterval = interval

	logx.Infow("PullInterval hot-reloaded",
		logx.Field("old", oldValue),
		logx.Field("new", interval))
}

// UpdateBatchSize 单独更新BatchSize（热重载）
func (w *TaskWorker) UpdateBatchSize(size int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.config.BatchSize == size {
		return
	}

	oldValue := w.config.BatchSize
	w.config.BatchSize = size

	logx.Infow("BatchSize hot-reloaded",
		logx.Field("old", oldValue),
		logx.Field("new", size))
}

// UpdateMaxConcurrent 单独更新MaxConcurrent（热重载）
func (w *TaskWorker) UpdateMaxConcurrent(max int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.config.MaxConcurrent == max {
		return
	}

	oldValue := w.config.MaxConcurrent
	w.config.MaxConcurrent = max

	logx.Infow("MaxConcurrent hot-reloaded",
		logx.Field("old", oldValue),
		logx.Field("new", max))
}

// UpdateTaskTimeout 单独更新TaskTimeout（热重载）
func (w *TaskWorker) UpdateTaskTimeout(timeout time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.config.TaskTimeout == timeout {
		return
	}

	oldValue := w.config.TaskTimeout
	w.config.TaskTimeout = timeout

	logx.Infow("TaskTimeout hot-reloaded",
		logx.Field("old", oldValue),
		logx.Field("new", timeout))
}

// ReloadableTaskWorker 支持热重载的TaskWorker包装器
type ReloadableTaskWorker struct {
	worker         *TaskWorker
	configReloader *ConfigReloader
}

// ConfigReloader 配置重载器
type ConfigReloader struct {
	reloadCount int64 // 重载次数（原子操作）
	lastReload  atomic.Value
}

// NewConfigReloader 创建配置重载器
func NewConfigReloader() *ConfigReloader {
	cr := &ConfigReloader{}
	cr.lastReload.Store(time.Now())
	return cr
}

// RecordReload 记录一次重载
func (cr *ConfigReloader) RecordReload() {
	atomic.AddInt64(&cr.reloadCount, 1)
	cr.lastReload.Store(time.Now())

	logx.Infow("Config reload recorded",
		logx.Field("total_reloads", atomic.LoadInt64(&cr.reloadCount)),
		logx.Field("last_reload", cr.lastReload.Load().(time.Time)))
}

// GetReloadStats 获取重载统计
func (cr *ConfigReloader) GetReloadStats() (count int64, lastReload time.Time) {
	return atomic.LoadInt64(&cr.reloadCount), cr.lastReload.Load().(time.Time)
}
