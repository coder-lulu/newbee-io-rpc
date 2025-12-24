package config

import (
	"time"

	"github.com/coder-lulu/newbee-common/v2/config"

	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	DatabaseConf config.DatabaseConf
	RedisConf    config.RedisConf
	CoreRpc      zrpc.RpcClientConf `json:",optional"` // Core服务RPC配置
	OpsRpc       zrpc.RpcClientConf `json:",optional"` // Ops-Center服务RPC配置
	TaskWorker   TaskWorkerConf     `json:",optional"` // TaskWorker配置
}

// TaskWorkerConf TaskWorker配置
type TaskWorkerConf struct {
	Enabled            bool          `json:",default=true"`            // 是否启用TaskWorker
	PullInterval       time.Duration `json:",default=10s"`             // 任务拉取间隔
	BatchSize          int           `json:",default=10"`              // 每次拉取任务数量
	MaxConcurrent      int           `json:",default=5"`               // 最大并发处理任务数
	TaskTimeout        time.Duration `json:",default=5m"`              // 单个任务超时时间
	StaleThreshold     time.Duration `json:",default=1h"`              // 任务stale阈值
	StaleCheckInterval time.Duration `json:",default=5m"`              // stale任务检查间隔
}
