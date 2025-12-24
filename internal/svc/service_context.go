package svc

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	_ "github.com/coder-lulu/newbee-io-rpc/ent/runtime"
	"github.com/coder-lulu/newbee-io-rpc/internal/config"
	"github.com/coder-lulu/newbee-io-rpc/internal/worker"
	"github.com/coder-lulu/newbee-core/rpc/coreclient"
	"github.com/coder-lulu/newbee-ops-rpc/opsclient"

	"github.com/coder-lulu/newbee-common/v2/orm/ent/hooks"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config     config.Config
	DB         *ent.Client
	Redis      redis.UniversalClient
	CoreRpc    coreclient.Core // Core服务RPC客户端
	OpsRpc     opsclient.Ops   // Ops-Center服务RPC客户端
	TaskWorker *worker.TaskWorker // 任务处理Worker
}

func NewServiceContext(c config.Config) *ServiceContext {
	db := ent.NewClient(
		ent.Log(logx.Info), // logger
		ent.Driver(c.DatabaseConf.NewNoCacheDriver()),
		ent.Debug(), // debug mode
	)

	// 🎯 使用统一Hook系统 - 一键设置租户和部门Hook
	// 配置Unified-IO服务的租户过滤规则 - 添加特有的系统表
	hooks.AddExcludedTable("cmdb_asset_types")    // 资产类型表是系统级数据
	hooks.AddExcludedTable("cmdb_templates")      // 模板表是系统级数据
	hooks.AddExcludedTable("cmdb_attribute_definitions") // 属性定义表是系统级数据

	// 一键设置：初始化配置 + 注册所有hooks (租户Hook + 部门Hook)
	if err := hooks.QuickSetup(db); err != nil {
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
	if c.CoreRpc.Endpoints != nil && len(c.CoreRpc.Endpoints) > 0 {
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

	// 初始化TaskWorker
	var taskWorker *worker.TaskWorker
	if c.TaskWorker.Enabled {
		// 从配置创建WorkerConfig
		workerConfig := &worker.WorkerConfig{
			PullInterval:       c.TaskWorker.PullInterval,
			BatchSize:          c.TaskWorker.BatchSize,
			MaxConcurrent:      c.TaskWorker.MaxConcurrent,
			TaskTimeout:        c.TaskWorker.TaskTimeout,
			StaleThreshold:     c.TaskWorker.StaleThreshold,
			StaleCheckInterval: c.TaskWorker.StaleCheckInterval,
		}

		// 如果配置值为0，使用默认值
		if workerConfig.PullInterval == 0 {
			workerConfig.PullInterval = worker.DefaultWorkerConfig().PullInterval
		}
		if workerConfig.BatchSize == 0 {
			workerConfig.BatchSize = worker.DefaultWorkerConfig().BatchSize
		}
		if workerConfig.MaxConcurrent == 0 {
			workerConfig.MaxConcurrent = worker.DefaultWorkerConfig().MaxConcurrent
		}
		if workerConfig.TaskTimeout == 0 {
			workerConfig.TaskTimeout = worker.DefaultWorkerConfig().TaskTimeout
		}
		if workerConfig.StaleThreshold == 0 {
			workerConfig.StaleThreshold = worker.DefaultWorkerConfig().StaleThreshold
		}
		if workerConfig.StaleCheckInterval == 0 {
			workerConfig.StaleCheckInterval = worker.DefaultWorkerConfig().StaleCheckInterval
		}

		taskWorker = worker.NewTaskWorker(db, workerConfig)

		// 在独立goroutine中启动TaskWorker
		go func() {
			if err := taskWorker.Start(context.Background()); err != nil {
				logx.Errorw("Failed to start TaskWorker", logx.Field("error", err))
			}
		}()

		logx.Infow("TaskWorker initialized and started",
			logx.Field("pull_interval", workerConfig.PullInterval),
			logx.Field("batch_size", workerConfig.BatchSize),
			logx.Field("max_concurrent", workerConfig.MaxConcurrent))
	} else {
		logx.Info("TaskWorker is disabled in configuration")
	}

	return &ServiceContext{
		Config:     c,
		DB:         db,
		Redis:      c.RedisConf.MustNewUniversalRedis(),
		CoreRpc:    coreRpc,
		OpsRpc:     opsRpc,
		TaskWorker: taskWorker,
	}
}
