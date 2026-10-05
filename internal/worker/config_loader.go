package worker

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/internal/service"
	"github.com/zeromicro/go-zero/core/logx"
)

// LoadWorkerConfigFromCenter 从ConfigCenter加载Worker配置
//
// 参数:
//   - ctx: 上下文
//   - configCenter: 配置中心服务
//   - tenantID: 租户ID
//
// 返回:
//   - *WorkerConfig: Worker配置
//
// 配置加载顺序:
// 1. 尝试从ConfigCenter读取
// 2. 如果ConfigCenter不可用或配置不存在，使用默认值
// 3. 记录配置来源日志
func LoadWorkerConfigFromCenter(ctx context.Context, configCenter *service.ConfigCenter, tenantID uint64) *WorkerConfig {
	opts := &service.ConfigOptions{
		TenantID:    tenantID,
		ServiceName: "unified-io",
		Category:    "service",
	}

	config := DefaultWorkerConfig()

	// 如果ConfigCenter未初始化，直接返回默认配置
	if configCenter == nil {
		logx.Info("ConfigCenter not available, using default worker config")
		return config
	}

	// 加载 PullInterval
	if val, err := configCenter.Get(ctx, "worker.pull_interval", opts); err == nil {
		if duration, err := time.ParseDuration(val); err == nil {
			config.PullInterval = duration
			logx.Infow("Loaded worker.pull_interval from ConfigCenter",
				logx.Field("value", duration))
		} else {
			logx.Errorw("Failed to parse worker.pull_interval",
				logx.Field("value", val),
				logx.Field("error", err))
		}
	} else {
		logx.Infow("worker.pull_interval not found in ConfigCenter, using default",
			logx.Field("default", config.PullInterval))
	}

	// 加载 BatchSize
	if val, err := configCenter.GetInt(ctx, "worker.batch_size", opts); err == nil {
		config.BatchSize = int(val)
		logx.Infow("Loaded worker.batch_size from ConfigCenter",
			logx.Field("value", val))
	} else {
		logx.Infow("worker.batch_size not found in ConfigCenter, using default",
			logx.Field("default", config.BatchSize))
	}

	// 加载 MaxConcurrent
	if val, err := configCenter.GetInt(ctx, "worker.max_concurrent", opts); err == nil {
		config.MaxConcurrent = int(val)
		logx.Infow("Loaded worker.max_concurrent from ConfigCenter",
			logx.Field("value", val))
	} else {
		logx.Infow("worker.max_concurrent not found in ConfigCenter, using default",
			logx.Field("default", config.MaxConcurrent))
	}

	// 加载 TaskTimeout
	if val, err := configCenter.Get(ctx, "worker.task_timeout", opts); err == nil {
		if duration, err := time.ParseDuration(val); err == nil {
			config.TaskTimeout = duration
			logx.Infow("Loaded worker.task_timeout from ConfigCenter",
				logx.Field("value", duration))
		} else {
			logx.Errorw("Failed to parse worker.task_timeout",
				logx.Field("value", val),
				logx.Field("error", err))
		}
	} else {
		logx.Infow("worker.task_timeout not found in ConfigCenter, using default",
			logx.Field("default", config.TaskTimeout))
	}

	// 加载 StaleThreshold
	if val, err := configCenter.Get(ctx, "worker.stale_threshold", opts); err == nil {
		if duration, err := time.ParseDuration(val); err == nil {
			config.StaleThreshold = duration
			logx.Infow("Loaded worker.stale_threshold from ConfigCenter",
				logx.Field("value", duration))
		} else {
			logx.Errorw("Failed to parse worker.stale_threshold",
				logx.Field("value", val),
				logx.Field("error", err))
		}
	} else {
		logx.Infow("worker.stale_threshold not found in ConfigCenter, using default",
			logx.Field("default", config.StaleThreshold))
	}

	// 加载 StaleCheckInterval
	if val, err := configCenter.Get(ctx, "worker.stale_check_interval", opts); err == nil {
		if duration, err := time.ParseDuration(val); err == nil {
			config.StaleCheckInterval = duration
			logx.Infow("Loaded worker.stale_check_interval from ConfigCenter",
				logx.Field("value", duration))
		} else {
			logx.Errorw("Failed to parse worker.stale_check_interval",
				logx.Field("value", val),
				logx.Field("error", err))
		}
	} else {
		logx.Infow("worker.stale_check_interval not found in ConfigCenter, using default",
			logx.Field("default", config.StaleCheckInterval))
	}

	return config
}

// WatchWorkerConfigChanges 监听Worker配置变更并动态更新
//
// 参数:
//   - configCenter: 配置中心服务
//   - worker: TaskWorker实例
//   - tenantID: 租户ID
//
// 特性:
//   - ✅ 实时热重载：配置变更无需重启服务
//   - ✅ 原子更新：不会中断正在执行的任务
//   - ✅ 类型安全：自动类型转换和验证
//   - ✅ 错误处理：解析失败时保持旧配置
func WatchWorkerConfigChanges(configCenter *service.ConfigCenter, worker *TaskWorker, tenantID uint64) {
	if configCenter == nil || worker == nil {
		return
	}

	opts := &service.ConfigOptions{
		TenantID:    tenantID,
		ServiceName: "unified-io",
		Category:    "service",
	}

	// 监听 PullInterval
	configCenter.Watch("worker.pull_interval", func(key, oldValue, newValue string) {
		if newValue == "" {
			return
		}

		duration, err := time.ParseDuration(newValue)
		if err != nil {
			logx.Errorw("Failed to parse worker.pull_interval",
				logx.Field("value", newValue),
				logx.Field("error", err))
			return
		}

		worker.UpdatePullInterval(duration)
		logx.Infow("🔥 Worker config hot-reloaded",
			logx.Field("key", "pull_interval"),
			logx.Field("old", oldValue),
			logx.Field("new", duration))
	})

	// 监听 BatchSize
	configCenter.Watch("worker.batch_size", func(key, oldValue, newValue string) {
		if newValue == "" {
			return
		}

		size, err := configCenter.GetInt(context.Background(), "worker.batch_size", opts)
		if err != nil {
			logx.Errorw("Failed to get worker.batch_size",
				logx.Field("error", err))
			return
		}

		worker.UpdateBatchSize(int(size))
		logx.Infow("🔥 Worker config hot-reloaded",
			logx.Field("key", "batch_size"),
			logx.Field("old", oldValue),
			logx.Field("new", size))
	})

	// 监听 MaxConcurrent
	configCenter.Watch("worker.max_concurrent", func(key, oldValue, newValue string) {
		if newValue == "" {
			return
		}

		max, err := configCenter.GetInt(context.Background(), "worker.max_concurrent", opts)
		if err != nil {
			logx.Errorw("Failed to get worker.max_concurrent",
				logx.Field("error", err))
			return
		}

		worker.UpdateMaxConcurrent(int(max))
		logx.Infow("🔥 Worker config hot-reloaded",
			logx.Field("key", "max_concurrent"),
			logx.Field("old", oldValue),
			logx.Field("new", max))
	})

	// 监听 TaskTimeout
	configCenter.Watch("worker.task_timeout", func(key, oldValue, newValue string) {
		if newValue == "" {
			return
		}

		duration, err := time.ParseDuration(newValue)
		if err != nil {
			logx.Errorw("Failed to parse worker.task_timeout",
				logx.Field("value", newValue),
				logx.Field("error", err))
			return
		}

		worker.UpdateTaskTimeout(duration)
		logx.Infow("🔥 Worker config hot-reloaded",
			logx.Field("key", "task_timeout"),
			logx.Field("old", oldValue),
			logx.Field("new", duration))
	})

	// 监听 StaleThreshold
	configCenter.Watch("worker.stale_threshold", func(key, oldValue, newValue string) {
		if newValue == "" {
			return
		}

		duration, err := time.ParseDuration(newValue)
		if err != nil {
			logx.Errorw("Failed to parse worker.stale_threshold",
				logx.Field("value", newValue),
				logx.Field("error", err))
			return
		}

		// StaleThreshold可以直接更新，下次检查时生效
		currentConfig := worker.GetConfig()
		currentConfig.StaleThreshold = duration
		worker.UpdateConfig(currentConfig)

		logx.Infow("🔥 Worker config hot-reloaded",
			logx.Field("key", "stale_threshold"),
			logx.Field("old", oldValue),
			logx.Field("new", duration))
	})

	// 监听 StaleCheckInterval
	configCenter.Watch("worker.stale_check_interval", func(key, oldValue, newValue string) {
		if newValue == "" {
			return
		}

		duration, err := time.ParseDuration(newValue)
		if err != nil {
			logx.Errorw("Failed to parse worker.stale_check_interval",
				logx.Field("value", newValue),
				logx.Field("error", err))
			return
		}

		// StaleCheckInterval可以直接更新，下次ticker触发时生效
		currentConfig := worker.GetConfig()
		currentConfig.StaleCheckInterval = duration
		worker.UpdateConfig(currentConfig)

		logx.Infow("🔥 Worker config hot-reloaded",
			logx.Field("key", "stale_check_interval"),
			logx.Field("old", oldValue),
			logx.Field("new", duration))
	})

	logx.Infow("✅ Worker config hot-reload watchers registered",
		logx.Field("monitored_configs", 6),
		logx.Field("restart_required", false))
}
