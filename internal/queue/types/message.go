package types

import "time"

// TaskMessage 任务消息体
type TaskMessage struct {
	TenantID        uint64                 `json:"tenant_id"`
	TaskRunID       uint64                 `json:"task_run_id"`
	TaskType        string                 `json:"task_type"`

	// 凭证引用（不传递实际密码）
	CredentialRefID string                 `json:"credential_ref_id,omitempty"`

	// Provider配置
	ProviderID      string                 `json:"provider_id"`
	ProviderConfig  map[string]interface{} `json:"provider_config,omitempty"`

	// 映射配置
	MappingRules    []FieldMapping         `json:"mapping_rules,omitempty"`

	// 元数据
	UserID          uint64                 `json:"user_id"`
	IdempotencyKey  string                 `json:"idempotency_key"`
	CreatedAt       time.Time              `json:"created_at"`

	// 扩展参数
	Params          map[string]interface{} `json:"params,omitempty"`
}

// FieldMapping 字段映射规则
type FieldMapping struct {
	SourceField string `json:"source_field"`
	TargetField string `json:"target_field"`
	Transform   string `json:"transform,omitempty"`
}

// LargeMessageRef 大消息引用（用于>900KB的消息）
type LargeMessageRef struct {
	StorageRef string    `json:"storage_ref"`
	Size       int       `json:"size"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// MessageHeaders Kafka消息头
type MessageHeaders struct {
	TenantID  string `json:"X-Tenant-ID"`
	TraceID   string `json:"X-Trace-ID"`
	MessageID string `json:"X-Message-ID"`
	Timestamp string `json:"X-Timestamp"`
}
