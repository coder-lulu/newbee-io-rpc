package worker

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/zeromicro/go-zero/core/logx"
)

// TaskType 任务类型枚举
type TaskType string

const (
	TaskTypeInput  TaskType = "input"  // 输入任务（数据采集）
	TaskTypeOutput TaskType = "output" // 输出任务（数据分发）
)

// TaskStatus 任务状态枚举
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // 待执行
	TaskStatusRunning   TaskStatus = "running"   // 执行中
	TaskStatusCompleted TaskStatus = "completed" // 已完成
	TaskStatusFailed    TaskStatus = "failed"    // 失败
	TaskStatusCancelled TaskStatus = "cancelled" // 已取消
)

// ExecutionContext 任务执行上下文
type ExecutionContext struct {
	// 上下文控制
	Ctx context.Context

	// 租户信息
	TenantID uint64
	UserID   string

	// 任务信息
	TaskID       uint64
	TaskType     TaskType
	TaskConfig   map[string]interface{}

	// Provider/Target信息
	// 对于InputTask: InputSource + SourceConfig
	// 对于OutputTask: OutputTarget + TargetConfig
	ProviderID   string // InputSource 或 OutputTarget
	ProviderConfig map[string]interface{} // SourceConfig 或 TargetConfig (JSON)

	// 执行控制
	Timeout      time.Duration
	MaxRetries   int
	RetryCount   int

	// 日志
	Logger       logx.Logger

	// 数据库客户端
	DB           *ent.Client
}

// TaskResult 任务执行结果
type TaskResult struct {
	// 执行状态
	Status        TaskStatus
	ErrorMessage  string

	// 统计信息
	TotalRecords    int64
	ProcessedRecords int64
	SuccessRecords   int64
	FailedRecords    int64

	// 执行时间
	StartedAt    time.Time
	CompletedAt  time.Time
	Duration     time.Duration

	// 结果数据（可选）
	Data         []map[string]interface{}
	Metadata     map[string]interface{}
}

// DispatcherConfig 分发器配置
type DispatcherConfig struct {
	// 批量处理配置
	BatchSize       int           // 批量获取任务数
	PollInterval    time.Duration // 轮询间隔
	MaxConcurrency  int           // 最大并发任务数

	// 超时配置
	DefaultTimeout  time.Duration // 默认任务超时时间
	MaxTimeout      time.Duration // 最大任务超时时间

	// 重试配置
	MaxRetries      int           // 最大重试次数
	RetryDelay      time.Duration // 重试延迟

	// 租户隔离
	EnableTenantIsolation bool    // 启用租户隔离验证
}

// DefaultDispatcherConfig 返回默认的分发器配置
func DefaultDispatcherConfig() *DispatcherConfig {
	return &DispatcherConfig{
		BatchSize:             10,
		PollInterval:          5 * time.Second,
		MaxConcurrency:        5,
		DefaultTimeout:        5 * time.Minute,
		MaxTimeout:            30 * time.Minute,
		MaxRetries:            3,
		RetryDelay:            10 * time.Second,
		EnableTenantIsolation: true,
	}
}

// ExecutorConfig 执行器配置
type ExecutorConfig struct {
	// 超时配置
	DefaultTimeout time.Duration

	// Provider配置
	ProviderTimeout time.Duration

	// 错误处理
	PanicRecovery   bool          // 是否启用panic恢复

	// 性能配置
	EnableMetrics   bool          // 启用性能指标收集
}

// DefaultExecutorConfig 返回默认的执行器配置
func DefaultExecutorConfig() *ExecutorConfig {
	return &ExecutorConfig{
		DefaultTimeout:  5 * time.Minute,
		ProviderTimeout: 3 * time.Minute,
		PanicRecovery:   true,
		EnableMetrics:   true,
	}
}

// WorkerMetrics Worker性能指标
type WorkerMetrics struct {
	// 任务统计
	TasksDispatched   int64
	TasksCompleted    int64
	TasksFailed       int64
	TasksCancelled    int64

	// 性能指标
	AvgExecutionTime  time.Duration
	MaxExecutionTime  time.Duration
	MinExecutionTime  time.Duration

	// 吞吐量
	ThroughputPerMin  float64

	// 最后更新时间
	LastUpdated       time.Time
}
