package transform

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/zeromicro/go-zero/core/logx"
)

// TransformType 转换类型
type TransformType string

const (
	TransformTypeDirect    TransformType = "direct"     // 直接映射
	TransformTypeConvert   TransformType = "convert"    // 类型转换
	TransformTypeLookup    TransformType = "lookup"     // 查找表
	TransformTypeTemplate  TransformType = "template"   // 模板
	TransformTypeScript    TransformType = "script"     // 脚本
	TransformTypeConcat    TransformType = "concat"     // 拼接
	TransformTypeSplit     TransformType = "split"      // 拆分
	TransformTypeCalculate TransformType = "calculate"  // 计算
	TransformTypeCondition TransformType = "condition"  // 条件
)

// MappingStatus 映射状态
type MappingStatus string

const (
	MappingStatusSuccess MappingStatus = "success" // 成功
	MappingStatusFailed  MappingStatus = "failed"  // 失败
	MappingStatusSkipped MappingStatus = "skipped" // 跳过
)

// TransformContext Transform执行上下文
type TransformContext struct {
	Ctx      context.Context
	TenantID uint64
	UserID   string
	TaskID   uint64
	TaskType string // "input" or "output"
	Logger   logx.Logger
	DB       *ent.Client

	// 输入数据
	SourceData map[string]interface{} // 原始数据

	// 映射规则
	Mappings []*ent.FieldMapping

	// 输出数据
	TargetData map[string]interface{} // 转换后数据

	// 执行统计
	Stats *TransformStats
}

// TransformStats Transform执行统计
type TransformStats struct {
	TotalFields    int           // 总字段数
	MappedFields   int           // 已映射字段数
	SuccessFields  int           // 成功字段数
	FailedFields   int           // 失败字段数
	SkippedFields  int           // 跳过字段数
	StartTime      time.Time     // 开始时间
	EndTime        time.Time     // 结束时间
	Duration       time.Duration // 执行时长
	ErrorMessages  []string      // 错误信息列表
}

// TransformResult Transform执行结果
type TransformResult struct {
	Success      bool                   // 是否成功
	TargetData   map[string]interface{} // 转换后数据
	Stats        *TransformStats        // 执行统计
	Logs         []*MappingLogEntry     // 映射日志
	ErrorMessage string                 // 错误信息
}

// MappingLogEntry 映射日志条目
type MappingLogEntry struct {
	FieldMappingID uint64
	MappingName    string
	SourceField    string
	TargetField    string
	SourceValue    interface{}
	TargetValue    interface{}
	Status         MappingStatus
	ErrorMessage   string
	LoggedAt       time.Time
}

// ValidationRule 验证规则
type ValidationRule struct {
	Type     string      // 规则类型: required, regex, range, enum, custom
	Params   interface{} // 规则参数
	Message  string      // 错误消息
}

// LookupTable 查找表
type LookupTable map[string]string

// ConditionRule 条件规则
type ConditionRule struct {
	Condition string      // 条件表达式
	TrueValue interface{} // 条件为真时的值
	FalseValue interface{} // 条件为假时的值
}

// TransformConfig 转换配置
type TransformConfig struct {
	Type       TransformType          // 转换类型
	Template   string                 // 模板（用于template类型）
	Script     string                 // 脚本（用于script类型）
	Separator  string                 // 分隔符（用于concat/split类型）
	Expression string                 // 表达式（用于calculate类型）
	Params     map[string]interface{} // 其他参数
}

// EngineConfig Transform Engine配置
type EngineConfig struct {
	// 验证配置
	EnableValidation   bool // 启用验证
	StrictMode         bool // 严格模式（验证失败时中止）

	// 日志配置
	EnableMappingLog   bool // 启用映射日志
	LogSuccessMapping  bool // 记录成功的映射
	LogFailedMapping   bool // 记录失败的映射

	// 性能配置
	MaxConcurrentMappings int // 最大并发映射数
	MappingTimeout        time.Duration // 映射超时时间

	// 容错配置
	ContinueOnError    bool // 出错时继续
	MaxErrors          int  // 最大错误数

	// 缓存配置
	EnableLookupCache  bool // 启用查找表缓存
	LookupCacheTTL     time.Duration // 查找表缓存TTL
}

// DefaultEngineConfig 默认Engine配置
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		EnableValidation:      true,
		StrictMode:            false,
		EnableMappingLog:      true,
		LogSuccessMapping:     false, // 默认只记录失败
		LogFailedMapping:      true,
		MaxConcurrentMappings: 10,
		MappingTimeout:        30 * time.Second,
		ContinueOnError:       true,
		MaxErrors:             100,
		EnableLookupCache:     true,
		LookupCacheTTL:        5 * time.Minute,
	}
}
