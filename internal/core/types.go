package core

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/types/cmdb"
	"github.com/gofrs/uuid/v5"
)

// OperationType 操作类型
type OperationType string

const (
	OperationCreate      OperationType = "create"
	OperationUpdate      OperationType = "update"
	OperationDelete      OperationType = "delete"
	OperationBatchCreate OperationType = "batch_create"
	OperationBatchUpdate OperationType = "batch_update"
	OperationBatchDelete OperationType = "batch_delete"
	OperationImport      OperationType = "import"
	OperationSync        OperationType = "sync"
)

// OperationStatus 操作状态
type OperationStatus string

const (
	StatusPending         OperationStatus = "pending"
	StatusValidating      OperationStatus = "validating"
	StatusProcessing      OperationStatus = "processing"
	StatusWaitingApproval OperationStatus = "waiting_approval"
	StatusApproved        OperationStatus = "approved"
	StatusRejected        OperationStatus = "rejected"
	StatusCompleted       OperationStatus = "completed"
	StatusFailed          OperationStatus = "failed"
)

// OperationSource 操作来源
type OperationSource string

const (
	SourceManual     OperationSource = "manual"
	SourceAPI        OperationSource = "api"
	SourceImport     OperationSource = "import"
	SourceSync       OperationSource = "sync"
	SourceAutomation OperationSource = "automation"
	SourceSystem     OperationSource = "system"
)

// ValidationLevel 校验级别
type ValidationLevel string

const (
	ValidationBasic  ValidationLevel = "basic"
	ValidationStrict ValidationLevel = "strict"
	ValidationFull   ValidationLevel = "full"
)

// PermissionLevel 权限级别
type PermissionLevel string

const (
	PermissionNone       PermissionLevel = "none"
	PermissionRead       PermissionLevel = "read"
	PermissionWrite      PermissionLevel = "write"
	PermissionAdmin      PermissionLevel = "admin"
	PermissionSuperAdmin PermissionLevel = "super_admin"
)

// LifecycleStage 生命周期阶段
type LifecycleStage string

const (
	StageDraft     LifecycleStage = "draft"
	StageSubmitted LifecycleStage = "submitted"
	StageValidated LifecycleStage = "validated"
	StageApproved  LifecycleStage = "approved"
	StageExecuted  LifecycleStage = "executed"
	StageCompleted LifecycleStage = "completed"
	StageCancelled LifecycleStage = "cancelled"
	StageExpired   LifecycleStage = "expired"
)

// CiOperationContext CI操作上下文
type CiOperationContext struct {
	// 操作基础信息
	OperationID   string          `json:"operation_id"`
	Type          OperationType   `json:"type"`
	Source        OperationSource `json:"source"`
	Reason        string          `json:"reason,omitempty"`
	TransactionID string          `json:"transaction_id,omitempty"`

	// 操作者信息
	OperatorID         uuid.UUID `json:"operator_id"`
	OperatorName       string    `json:"operator_name"`
	OperatorRole       string    `json:"operator_role,omitempty"`
	OperatorDepartment string    `json:"operator_department,omitempty"`

	// CI相关信息
	CiID       *uint64  `json:"ci_id,omitempty"`
	CiTypeID   uint64   `json:"ci_type_id"`
	BatchCiIDs []uint64 `json:"batch_ci_ids,omitempty"`

	// 数据内容
	DataBefore *cmdb.CisInfo   `json:"data_before,omitempty"`
	DataAfter  *cmdb.CisInfo   `json:"data_after,omitempty"`
	BatchData  []*cmdb.CisInfo `json:"batch_data,omitempty"`

	// 配置选项
	ValidationLevel     ValidationLevel `json:"validation_level"`
	RequireApproval     bool            `json:"require_approval"`
	SkipPermissionCheck bool            `json:"skip_permission_check"`
	SkipValidation      bool            `json:"skip_validation"`
	AsyncExecution      bool            `json:"async_execution"`

	// 上下文数据
	SourceDetail   string                 `json:"source_detail,omitempty"`
	RequestContext map[string]interface{} `json:"request_context,omitempty"`
	ClientIP       string                 `json:"client_ip,omitempty"`
	UserAgent      string                 `json:"user_agent,omitempty"`

	// 时间信息
	RequestTime time.Time     `json:"request_time"`
	Timeout     time.Duration `json:"timeout,omitempty"`

	// 扩展信息
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Tags     []string               `json:"tags,omitempty"`
}

// CiOperationResult CI操作结果
type CiOperationResult struct {
	// 操作基础信息
	OperationID string          `json:"operation_id"`
	Status      OperationStatus `json:"status"`
	Message     string          `json:"message"`

	// 执行结果
	Success            bool                   `json:"success"`
	AffectedCiIDs      []uint64               `json:"affected_ci_ids,omitempty"`
	CreatedCiID        *uint64                `json:"created_ci_id,omitempty"`
	ExecutionTime      time.Duration          `json:"execution_time"`
	PerformanceMetrics map[string]interface{} `json:"performance_metrics,omitempty"`

	// 批量操作结果
	BatchTotal   int `json:"batch_total,omitempty"`
	BatchSuccess int `json:"batch_success,omitempty"`
	BatchFailed  int `json:"batch_failed,omitempty"`

	// 校验结果
	ValidationResult *ValidationResult `json:"validation_result,omitempty"`

	// 权限检查结果
	PermissionResult *PermissionResult `json:"permission_result,omitempty"`

	// 变更记录
	ChangeRecordID string `json:"change_record_id,omitempty"`

	// 生命周期状态
	LifecycleStage   LifecycleStage `json:"lifecycle_stage"`
	LifecycleStateID string         `json:"lifecycle_state_id,omitempty"`

	// 审批信息
	ApprovalFlowID  string `json:"approval_flow_id,omitempty"`
	RequireApproval bool   `json:"require_approval"`

	// 错误信息
	Error        error                  `json:"error,omitempty"`
	ErrorCode    string                 `json:"error_code,omitempty"`
	ErrorDetails map[string]interface{} `json:"error_details,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`

	// 扩展信息
	Data     interface{}            `json:"data,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ValidationResult 校验结果
type ValidationResult struct {
	Valid        bool                   `json:"valid"`
	Errors       []*ValidationError     `json:"errors,omitempty"`
	Warnings     []*ValidationWarning   `json:"warnings,omitempty"`
	CheckedRules []string               `json:"checked_rules,omitempty"`
	Performance  *ValidationPerformance `json:"performance,omitempty"`
}

// ValidationError 校验错误
type ValidationError struct {
	Field     string      `json:"field"`
	AttrID    uint64      `json:"attr_id,omitempty"`
	ErrorType string      `json:"error_type"`
	Message   string      `json:"message"`
	Code      string      `json:"code,omitempty"`
	Value     interface{} `json:"value,omitempty"`
}

// ValidationWarning 校验警告
type ValidationWarning struct {
	Field   string      `json:"field"`
	AttrID  uint64      `json:"attr_id,omitempty"`
	Type    string      `json:"type"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// ValidationPerformance 校验性能信息
type ValidationPerformance struct {
	TotalTime      time.Duration            `json:"total_time"`
	RuleCount      int                      `json:"rule_count"`
	AttributeCount int                      `json:"attribute_count"`
	Details        map[string]time.Duration `json:"details,omitempty"`
}

// PermissionResult 权限检查结果
type PermissionResult struct {
	Granted       bool                   `json:"granted"`
	Level         PermissionLevel        `json:"level"`
	Operations    []string               `json:"operations"`
	Restrictions  map[string]interface{} `json:"restrictions,omitempty"`
	CheckedRules  []string               `json:"checked_rules,omitempty"`
	DeniedReasons []string               `json:"denied_reasons,omitempty"`
	RequiresMFA   bool                   `json:"requires_mfa"`
	Performance   *PermissionPerformance `json:"performance,omitempty"`
}

// PermissionPerformance 权限检查性能信息
type PermissionPerformance struct {
	TotalTime   time.Duration `json:"total_time"`
	RuleCount   int           `json:"rule_count"`
	CacheHits   int           `json:"cache_hits"`
	CacheMisses int           `json:"cache_misses"`
}

// ChangeRecord 变更记录
type ChangeRecord struct {
	RecordID        string                 `json:"record_id"`
	OperationID     string                 `json:"operation_id"`
	CiID            uint64                 `json:"ci_id"`
	CiTypeID        uint64                 `json:"ci_type_id"`
	OperationType   OperationType          `json:"operation_type"`
	OperationTime   time.Time              `json:"operation_time"`
	OperatorID      uuid.UUID              `json:"operator_id"`
	OperatorName    string                 `json:"operator_name"`
	Source          OperationSource        `json:"source"`
	SourceDetail    string                 `json:"source_detail,omitempty"`
	DataBefore      map[string]interface{} `json:"data_before,omitempty"`
	DataAfter       map[string]interface{} `json:"data_after,omitempty"`
	ChangedFields   []string               `json:"changed_fields,omitempty"`
	ChangeSummary   map[string]interface{} `json:"change_summary,omitempty"`
	ChangeReason    string                 `json:"change_reason,omitempty"`
	Description     string                 `json:"description,omitempty"`
	IsMajorChange   bool                   `json:"is_major_change"`
	RequireApproval bool                   `json:"require_approval"`
	ApprovalStatus  string                 `json:"approval_status,omitempty"`
}

// LifecycleState 生命周期状态
type LifecycleState struct {
	StateID        string                 `json:"state_id"`
	StateName      string                 `json:"state_name"`
	StateType      string                 `json:"state_type"`
	CiID           uint64                 `json:"ci_id"`
	CiTypeID       uint64                 `json:"ci_type_id"`
	EnteredAt      time.Time              `json:"entered_at"`
	ExpectedExitAt *time.Time             `json:"expected_exit_at,omitempty"`
	TriggerType    string                 `json:"trigger_type"`
	TriggeredBy    uuid.UUID              `json:"triggered_by"`
	StateData      map[string]interface{} `json:"state_data,omitempty"`
	IsTimeout      bool                   `json:"is_timeout"`
	HasError       bool                   `json:"has_error"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
}

// ComponentInterface 组件接口定义
type ComponentInterface interface {
	Name() string
	Initialize(ctx context.Context) error
	Shutdown(ctx context.Context) error
	HealthCheck(ctx context.Context) error
}

// DataValidator 数据校验器接口
type DataValidator interface {
	ComponentInterface
	ValidateSchema(ctx context.Context, data *cmdb.CisInfo, typeID uint64) (*ValidationResult, error)
	ValidateBusinessRules(ctx context.Context, data *cmdb.CisInfo, typeID uint64) (*ValidationResult, error)
	ValidateUniqueness(ctx context.Context, data *cmdb.CisInfo, typeID uint64, excludeCiID *uint64) (*ValidationResult, error)
	ValidateComplete(ctx context.Context, operation *CiOperationContext) (*ValidationResult, error)
}

// PermissionChecker 权限检查器接口
type PermissionChecker interface {
	ComponentInterface
	CheckPermission(ctx context.Context, operation *CiOperationContext) (*PermissionResult, error)
	CheckFieldPermission(ctx context.Context, operatorID uuid.UUID, ciTypeID uint64, attrID uint64, operation string) (*PermissionResult, error)
	CheckBatchPermission(ctx context.Context, operation *CiOperationContext) (*PermissionResult, error)
	GetUserPermissions(ctx context.Context, userID uuid.UUID, ciTypeID uint64) (*PermissionResult, error)
}

// ChangeRecorder 变更记录器接口
type ChangeRecorder interface {
	ComponentInterface
	RecordCreate(ctx context.Context, operation *CiOperationContext, result *CiOperationResult) (*ChangeRecord, error)
	RecordUpdate(ctx context.Context, operation *CiOperationContext, result *CiOperationResult) (*ChangeRecord, error)
	RecordDelete(ctx context.Context, operation *CiOperationContext, result *CiOperationResult) (*ChangeRecord, error)
	RecordBatch(ctx context.Context, operation *CiOperationContext, result *CiOperationResult) ([]*ChangeRecord, error)
	QueryChangeHistory(ctx context.Context, ciID uint64, limit int, offset int) ([]*ChangeRecord, error)
}

// LifecycleManager 生命周期管理器接口
type LifecycleManager interface {
	ComponentInterface
	CreateState(ctx context.Context, operation *CiOperationContext) (*LifecycleState, error)
	UpdateState(ctx context.Context, stateID string, newStage LifecycleStage) (*LifecycleState, error)
	GetCurrentState(ctx context.Context, ciID uint64) (*LifecycleState, error)
	CheckStateTransition(ctx context.Context, currentStage, targetStage LifecycleStage) (bool, error)
	HandleTimeout(ctx context.Context, stateID string) error
	QueryStates(ctx context.Context, filters map[string]interface{}) ([]*LifecycleState, error)
}

// ApprovalManager 审批管理器接口
type ApprovalManager interface {
	ComponentInterface
	CheckRequireApproval(ctx context.Context, operation *CiOperationContext) (bool, string, error)
	SubmitForApproval(ctx context.Context, operation *CiOperationContext) (string, error)
	ProcessApproval(ctx context.Context, approvalID string, action string, comment string, approverID uuid.UUID) error
	GetApprovalStatus(ctx context.Context, approvalID string) (string, error)
	QueryPendingApprovals(ctx context.Context, approverID uuid.UUID) ([]map[string]interface{}, error)
}

// DataPersister 数据持久化器接口
type DataPersister interface {
	ComponentInterface
	Create(ctx context.Context, data *cmdb.CisInfo) (*cmdb.CisInfo, error)
	Update(ctx context.Context, ciID uint64, data *cmdb.CisInfo) (*cmdb.CisInfo, error)
	Delete(ctx context.Context, ciID uint64) error
	BatchCreate(ctx context.Context, dataList []*cmdb.CisInfo) ([]*cmdb.CisInfo, error)
	BatchUpdate(ctx context.Context, updates map[uint64]*cmdb.CisInfo) ([]*cmdb.CisInfo, error)
	BatchDelete(ctx context.Context, ciIDs []uint64) error
	Get(ctx context.Context, ciID uint64) (*cmdb.CisInfo, error)
}

// CiDataManager CI数据管理器接口
type CiDataManager interface {
	ComponentInterface

	// 基础CRUD操作
	CreateCi(ctx context.Context, operation *CiOperationContext) (*CiOperationResult, error)
	UpdateCi(ctx context.Context, operation *CiOperationContext) (*CiOperationResult, error)
	DeleteCi(ctx context.Context, operation *CiOperationContext) (*CiOperationResult, error)
	GetCi(ctx context.Context, ciID uint64, operatorID uuid.UUID) (*cmdb.CisInfo, error)

	// 批量操作
	BatchCreate(ctx context.Context, operation *CiOperationContext) (*CiOperationResult, error)
	BatchUpdate(ctx context.Context, operation *CiOperationContext) (*CiOperationResult, error)
	BatchDelete(ctx context.Context, operation *CiOperationContext) (*CiOperationResult, error)

	// 高级操作
	BulkImport(ctx context.Context, operation *CiOperationContext) (*CiOperationResult, error)
	SyncData(ctx context.Context, operation *CiOperationContext) (*CiOperationResult, error)

	// 操作查询和管理
	GetOperationStatus(ctx context.Context, operationID string) (*CiOperationResult, error)
	CancelOperation(ctx context.Context, operationID string, operatorID uuid.UUID) error
	RetryOperation(ctx context.Context, operationID string, operatorID uuid.UUID) (*CiOperationResult, error)

	// 审批流程
	SubmitForApproval(ctx context.Context, operationID string) error
	ProcessApproval(ctx context.Context, operationID string, action string, comment string, approverID uuid.UUID) error

	// 变更历史和审计
	GetChangeHistory(ctx context.Context, ciID uint64, limit int, offset int) ([]*ChangeRecord, error)
	GetOperationLog(ctx context.Context, filters map[string]interface{}) ([]*CiOperationContext, error)
}

// Configuration 配置结构
type Configuration struct {
	// 数据库配置
	Database struct {
		ConnectionTimeout time.Duration `json:"connection_timeout"`
		QueryTimeout      time.Duration `json:"query_timeout"`
		MaxConnections    int           `json:"max_connections"`
	} `json:"database"`

	// 校验配置
	Validation struct {
		DefaultLevel ValidationLevel `json:"default_level"`
		EnableCache  bool            `json:"enable_cache"`
		CacheTTL     time.Duration   `json:"cache_ttl"`
		Timeout      time.Duration   `json:"timeout"`
	} `json:"validation"`

	// 权限配置
	Permission struct {
		EnableCache       bool            `json:"enable_cache"`
		CacheTTL          time.Duration   `json:"cache_ttl"`
		DefaultLevel      PermissionLevel `json:"default_level"`
		RequireMFAForHigh bool            `json:"require_mfa_for_high"`
	} `json:"permission"`

	// 审批配置
	Approval struct {
		DefaultTimeout time.Duration `json:"default_timeout"`
		EnableNotify   bool          `json:"enable_notify"`
		AutoApprove    bool          `json:"auto_approve"`
	} `json:"approval"`

	// 生命周期配置
	Lifecycle struct {
		EnableTimeout   bool          `json:"enable_timeout"`
		DefaultTimeout  time.Duration `json:"default_timeout"`
		CleanupInterval time.Duration `json:"cleanup_interval"`
	} `json:"lifecycle"`

	// 性能配置
	Performance struct {
		BatchSize        int           `json:"batch_size"`
		MaxConcurrency   int           `json:"max_concurrency"`
		OperationTimeout time.Duration `json:"operation_timeout"`
		EnableMetrics    bool          `json:"enable_metrics"`
		MetricsInterval  time.Duration `json:"metrics_interval"`
	} `json:"performance"`
}

// DefaultConfiguration 返回默认配置
func DefaultConfiguration() *Configuration {
	return &Configuration{
		Database: struct {
			ConnectionTimeout time.Duration `json:"connection_timeout"`
			QueryTimeout      time.Duration `json:"query_timeout"`
			MaxConnections    int           `json:"max_connections"`
		}{
			ConnectionTimeout: 30 * time.Second,
			QueryTimeout:      60 * time.Second,
			MaxConnections:    100,
		},
		Validation: struct {
			DefaultLevel ValidationLevel `json:"default_level"`
			EnableCache  bool            `json:"enable_cache"`
			CacheTTL     time.Duration   `json:"cache_ttl"`
			Timeout      time.Duration   `json:"timeout"`
		}{
			DefaultLevel: ValidationBasic,
			EnableCache:  true,
			CacheTTL:     10 * time.Minute,
			Timeout:      30 * time.Second,
		},
		Permission: struct {
			EnableCache       bool            `json:"enable_cache"`
			CacheTTL          time.Duration   `json:"cache_ttl"`
			DefaultLevel      PermissionLevel `json:"default_level"`
			RequireMFAForHigh bool            `json:"require_mfa_for_high"`
		}{
			EnableCache:       true,
			CacheTTL:          5 * time.Minute,
			DefaultLevel:      PermissionRead,
			RequireMFAForHigh: true,
		},
		Approval: struct {
			DefaultTimeout time.Duration `json:"default_timeout"`
			EnableNotify   bool          `json:"enable_notify"`
			AutoApprove    bool          `json:"auto_approve"`
		}{
			DefaultTimeout: 24 * time.Hour,
			EnableNotify:   true,
			AutoApprove:    false,
		},
		Lifecycle: struct {
			EnableTimeout   bool          `json:"enable_timeout"`
			DefaultTimeout  time.Duration `json:"default_timeout"`
			CleanupInterval time.Duration `json:"cleanup_interval"`
		}{
			EnableTimeout:   true,
			DefaultTimeout:  1 * time.Hour,
			CleanupInterval: 6 * time.Hour,
		},
		Performance: struct {
			BatchSize        int           `json:"batch_size"`
			MaxConcurrency   int           `json:"max_concurrency"`
			OperationTimeout time.Duration `json:"operation_timeout"`
			EnableMetrics    bool          `json:"enable_metrics"`
			MetricsInterval  time.Duration `json:"metrics_interval"`
		}{
			BatchSize:        100,
			MaxConcurrency:   10,
			OperationTimeout: 5 * time.Minute,
			EnableMetrics:    true,
			MetricsInterval:  1 * time.Minute,
		},
	}
}
