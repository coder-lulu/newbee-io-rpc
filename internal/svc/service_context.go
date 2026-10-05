package svc

import (
	"context"

	"github.com/coder-lulu/newbee-cmdb-rpc/cmdbclient"
	"github.com/coder-lulu/newbee-core/rpc/coreclient"
	"github.com/coder-lulu/newbee-io-rpc/ent"
	_ "github.com/coder-lulu/newbee-io-rpc/ent/runtime"
	"github.com/coder-lulu/newbee-io-rpc/internal/config"
	"github.com/coder-lulu/newbee-io-rpc/internal/lock"
	"github.com/coder-lulu/newbee-io-rpc/internal/monitoring"
	"github.com/coder-lulu/newbee-io-rpc/internal/service"
	"github.com/coder-lulu/newbee-io-rpc/internal/worker"
	"github.com/coder-lulu/newbee-ops-rpc/opsclient"

	"github.com/coder-lulu/newbee-common/v2/orm/ent/hooks"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config           config.Config
	DB               *ent.Client
	Redis            redis.UniversalClient
	CoreRpc          coreclient.Core              // Core服务RPC客户端
	OpsRpc           opsclient.Ops                // Ops-Center服务RPC客户端
	CmdbRpc          cmdbclient.Cmdb              // CMDB服务RPC客户端
	TaskWorker       *worker.TaskWorker           // 任务处理Worker
	CronScheduler    *worker.CronScheduler        // Cron调度器
	ConfigCenter     *service.ConfigCenter        // 配置中心服务
	OutputProcessor  *worker.OutputProcessor      // 输出处理器
	PrometheusServer *monitoring.PrometheusServer // Prometheus监控服务器
	MetricsCollector *monitoring.MetricsCollector // 指标收集器
}

func NewServiceContext(c config.Config) *ServiceContext {
	db := ent.NewClient(
		ent.Log(logx.Info), // logger
		ent.Driver(c.DatabaseConf.NewNoCacheDriver()),
		ent.Debug(), // debug mode
	)

	if err := setupIOHooks(db); err != nil {
		logx.Errorw("Failed to setup unified hooks", logx.Field("error", err.Error()))
		panic("统一Hook初始化失败: " + err.Error())
	}
	logx.Infow("✅ Unified-IO service: Unified hooks initialized successfully")

	// RPC服务层不注册数据权限拦截器
	// 数据权限控制应该在API层通过中间件处理，RPC层作为数据访问层不承担权限职责
	// 这样可以保持清晰的层次分离，避免跨服务的上下文传递问题

	// Unified IO service does not need relation types initialization

	// 初始化Core RPC客户端 - 参考Core服务的实现模式
	var coreRpc coreclient.Core
	if len(c.CoreRpc.Endpoints) > 0 || c.CoreRpc.Target != "" || c.CoreRpc.Etcd.Key != "" {
		// 创建RPC客户端，使用SystemContext拦截器支持系统级操作
		rpcClient, err := zrpc.NewClient(c.CoreRpc, zrpc.WithUnaryClientInterceptor(hooks.SystemContextClientInterceptor()))
		if err != nil {
			logx.Errorf("Failed to create Core RPC client: %v", err)
		} else {
			coreRpc = coreclient.NewCore(rpcClient)
		}
	}

	// 初始化Ops RPC客户端
	var opsRpc opsclient.Ops
	if c.OpsRpc.Endpoints != nil && len(c.OpsRpc.Endpoints) > 0 {
		logx.Infow("Initializing OpsRpc client...",
			logx.Field("endpoints", c.OpsRpc.Endpoints))

		// 创建RPC客户端，使用SystemContext拦截器支持系统级操作
		rpcClient, err := zrpc.NewClient(c.OpsRpc, zrpc.WithUnaryClientInterceptor(hooks.SystemContextClientInterceptor()))
		if err != nil {
			logx.Errorw("Failed to create Ops RPC client", logx.Field("error", err))
		} else {
			opsRpc = opsclient.NewOps(rpcClient)
			logx.Infow("✅ OpsRpc client initialized successfully")
		}
	} else {
		logx.Info("OpsRpc endpoints not configured, skipping OpsRpc client initialization")
	}

	// 初始化CMDB RPC客户端
	var cmdbRpc cmdbclient.Cmdb
	if c.CmdbRpc.Endpoints != nil && len(c.CmdbRpc.Endpoints) > 0 {
		logx.Infow("Initializing CmdbRpc client...",
			logx.Field("endpoints", c.CmdbRpc.Endpoints))

		// 创建RPC客户端，使用SystemContext拦截器支持系统级操作
		rpcClient, err := zrpc.NewClient(c.CmdbRpc, zrpc.WithUnaryClientInterceptor(hooks.SystemContextClientInterceptor()))
		if err != nil {
			logx.Errorw("Failed to create CMDB RPC client", logx.Field("error", err))
		} else {
			cmdbRpc = cmdbclient.NewCmdb(rpcClient)
			logx.Infow("✅ CmdbRpc client initialized successfully")
		}
	} else {
		logx.Info("CmdbRpc endpoints not configured, skipping CmdbRpc client initialization")
	}

	// 初始化OutputProcessor（输出处理器）
	var outputProcessor *worker.OutputProcessor
	if cmdbRpc != nil {
		outputProcessor = worker.NewOutputProcessor(db, cmdbRpc, logx.WithContext(context.Background()))
		logx.Infow("✅ OutputProcessor initialized with CMDB integration")
	} else {
		logx.Info("OutputProcessor initialization skipped (CmdbRpc not available)")
	}

	// 🎯 优先初始化ConfigCenter (统一配置管理)
	rds := c.RedisConf.MustNewUniversalRedis()
	configCenter := service.NewConfigCenter(db, rds)
	logx.Infow("✅ ConfigCenter initialized with three-level caching",
		logx.Field("memory_cache", true),
		logx.Field("redis_cache", true),
		logx.Field("hot_reload", true))

	// 初始化TaskWorker（带任务去重支持）
	var taskWorker *worker.TaskWorker
	if c.TaskWorker.Enabled {
		// 🎯 从ConfigCenter加载Worker配置（优先级：ConfigCenter > yaml配置文件 > 默认值）
		var workerConfig *worker.WorkerConfig

		// 尝试从ConfigCenter加载，如果失败则fallback到yaml配置
		workerConfig = worker.LoadWorkerConfigFromCenter(context.Background(), configCenter, 1)

		// yaml配置覆盖（如果yaml有显式配置且非零值）
		if c.TaskWorker.PullInterval > 0 {
			workerConfig.PullInterval = c.TaskWorker.PullInterval
			logx.Infow("Using PullInterval from yaml config",
				logx.Field("value", c.TaskWorker.PullInterval))
		}
		if c.TaskWorker.BatchSize > 0 {
			workerConfig.BatchSize = c.TaskWorker.BatchSize
			logx.Infow("Using BatchSize from yaml config",
				logx.Field("value", c.TaskWorker.BatchSize))
		}
		if c.TaskWorker.MaxConcurrent > 0 {
			workerConfig.MaxConcurrent = c.TaskWorker.MaxConcurrent
			logx.Infow("Using MaxConcurrent from yaml config",
				logx.Field("value", c.TaskWorker.MaxConcurrent))
		}
		if c.TaskWorker.TaskTimeout > 0 {
			workerConfig.TaskTimeout = c.TaskWorker.TaskTimeout
			logx.Infow("Using TaskTimeout from yaml config",
				logx.Field("value", c.TaskWorker.TaskTimeout))
		}
		if c.TaskWorker.StaleThreshold > 0 {
			workerConfig.StaleThreshold = c.TaskWorker.StaleThreshold
			logx.Infow("Using StaleThreshold from yaml config",
				logx.Field("value", c.TaskWorker.StaleThreshold))
		}
		if c.TaskWorker.StaleCheckInterval > 0 {
			workerConfig.StaleCheckInterval = c.TaskWorker.StaleCheckInterval
			logx.Infow("Using StaleCheckInterval from yaml config",
				logx.Field("value", c.TaskWorker.StaleCheckInterval))
		}

		// 🔒 创建分布式锁管理器（与CronScheduler共享）
		lockManager := lock.NewDistributedLockManager(rds)

		// 创建TaskWorker，传入分布式锁管理器（启用任务去重）
		taskWorker = worker.NewTaskWorker(db, workerConfig, lockManager)

		// 🔥 注册配置热重载监听器（无需重启）
		worker.WatchWorkerConfigChanges(configCenter, taskWorker, 1)
		logx.Infow("✅ Worker config hot-reload monitoring started",
			logx.Field("monitored_configs", 6),
			logx.Field("restart_required", false))

		// 在独立goroutine中启动TaskWorker
		go func() {
			if err := taskWorker.Start(context.Background()); err != nil {
				logx.Errorw("Failed to start TaskWorker", logx.Field("error", err))
			}
		}()

		logx.Infow("TaskWorker initialized with task deduplication support",
			logx.Field("pull_interval", workerConfig.PullInterval),
			logx.Field("batch_size", workerConfig.BatchSize),
			logx.Field("max_concurrent", workerConfig.MaxConcurrent),
			logx.Field("deduplication_enabled", true))
	} else {
		logx.Info("TaskWorker is disabled in configuration")
	}

	// 初始化CronScheduler (带分布式锁支持)
	var cronScheduler *worker.CronScheduler
	if c.TaskWorker.Enabled { // 与TaskWorker共享enabled配置
		// 🔒 使用相同的分布式锁管理器（防止多实例重复触发）
		lockManager := lock.NewDistributedLockManager(rds)

		// 创建CronScheduler，传入分布式锁管理器
		cronScheduler = worker.NewCronScheduler(db, lockManager)

		// 在独立goroutine中启动CronScheduler
		go func() {
			if err := cronScheduler.Start(context.Background()); err != nil {
				logx.Errorw("Failed to start CronScheduler", logx.Field("error", err))
			}
		}()

		logx.Infow("CronScheduler initialized with distributed lock support",
			logx.Field("enabled", true),
			logx.Field("lock_protection", true))
	} else {
		logx.Info("CronScheduler is disabled (TaskWorker disabled)")
	}

	// 初始化Prometheus监控服务器
	metricsCollector := monitoring.NewMetricsCollector()
	prometheusServer := monitoring.NewPrometheusServer(
		c.Prometheus.Host,
		c.Prometheus.Port,
		c.Prometheus.Path,
		metricsCollector,
	)

	// 启动Prometheus服务器
	if err := prometheusServer.Start(); err != nil {
		logx.Errorw("Failed to start Prometheus server", logx.Field("error", err))
	} else {
		logx.Infow("✅ Prometheus metrics server started",
			logx.Field("address", c.Prometheus.Host+":"+string(rune(c.Prometheus.Port))),
			logx.Field("metrics_path", c.Prometheus.Path))
	}

	// 初始化组件健康状态
	metricsCollector.SetComponentHealth("database", true)
	metricsCollector.SetComponentHealth("redis", true)
	if coreRpc != nil {
		metricsCollector.SetComponentHealth("core_rpc", true)
	}
	if cmdbRpc != nil {
		metricsCollector.SetComponentHealth("cmdb_rpc", true)
	}
	if opsRpc != nil {
		metricsCollector.SetComponentHealth("ops_rpc", true)
	}

	return &ServiceContext{
		Config:           c,
		DB:               db,
		Redis:            c.RedisConf.MustNewUniversalRedis(),
		CoreRpc:          coreRpc,
		OpsRpc:           opsRpc,
		CmdbRpc:          cmdbRpc,
		TaskWorker:       taskWorker,
		CronScheduler:    cronScheduler,
		ConfigCenter:     configCenter,
		OutputProcessor:  outputProcessor,
		PrometheusServer: prometheusServer,
		MetricsCollector: metricsCollector,
	}
}

// setupIOHooks declares the one global metrics entity that has no TenantMixin.
// Tenant-bearing IO entities continue to use strict tenant hooks.
func setupIOHooks(db *ent.Client) error {
	hooks.AddExcludedTable("cmdb_asset_types")
	hooks.AddExcludedTable("cmdb_templates")
	hooks.AddExcludedTable("cmdb_attribute_definitions")
	hooks.AddExcludedTable("io_worker_metrics")
	if err := hooks.QuickSetup(db); err != nil {
		return err
	}
	tenantConfig := hooks.GlobalHookManager.GetConfig(hooks.FieldTypeTenant)
	for _, entity := range tenantConfig.ExcludedEntities {
		if entity == "WorkerMetrics" {
			return nil
		}
	}
	tenantConfig.ExcludedEntities = append(tenantConfig.ExcludedEntities, "WorkerMetrics")
	return nil
}
