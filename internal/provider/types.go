package provider

// ProviderMetadata 提供者元数据
type ProviderMetadata struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Version        string   `json:"version"`
	Type           string   `json:"type"`
	Category       string   `json:"category"`  // 兼容字段
	IconURL        string   `json:"icon_url"`
	Icon           string   `json:"icon"`      // 兼容字段
	SupportedModes []string `json:"supported_modes"`
	Tags           []string `json:"tags"`
}

// SelectOption 下拉选项
type SelectOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// ParameterDefinition 参数定义
type ParameterDefinition struct {
	Name         string                 `json:"name"`
	DisplayName  string                 `json:"display_name"`
	Type         string                 `json:"type"`
	Required     bool                   `json:"required"`
	Description  string                 `json:"description"`
	DefaultValue interface{}            `json:"default_value"`
	Default      interface{}            `json:"default"`  // 兼容字段
	Example      string                 `json:"example"`
	Sensitive    bool                   `json:"sensitive"`
	Options      []SelectOption         `json:"options"`
	Validation   map[string]interface{} `json:"validation"`
}

// FieldDefinition 字段定义
type FieldDefinition struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Unique      bool   `json:"unique"`
	Searchable  bool   `json:"searchable"`
}

// TestResult 连接测试结果
type TestResult struct {
	// Success 是否成功
	Success bool `json:"success"`

	// Message 测试消息
	Message string `json:"message"`

	// Latency 测试延迟(毫秒)
	Latency int64 `json:"latency,omitempty"`

	// Details 详细信息(如: 版本号、连接池状态等)
	Details map[string]interface{} `json:"details,omitempty"`

	// Error 错误信息(失败时)
	Error string `json:"error,omitempty"`
}

// DiscoveryResult 发现结果
type DiscoveryResult struct {
	Success      bool                     `json:"success"`
	TotalRecords int64                    `json:"total_records"`
	Records      []map[string]interface{} `json:"records"`
	Metadata     map[string]interface{}   `json:"metadata"`
	ErrorMessage string                   `json:"error_message,omitempty"`
}

// FieldMapping 字段映射
type FieldMapping struct {
	SourceField     string                 `json:"source_field"`
	TargetField     string                 `json:"target_field"`
	Transform       string                 `json:"transform"`
	TransformParams map[string]interface{} `json:"transform_params"`
}

// TransformDefinition 转换定义
type TransformDefinition struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Parameters  []string `json:"parameters"`
}

// FieldMappingConfig 字段映射配置
type FieldMappingConfig struct {
	Version     string                             `json:"version"`
	Mappings    map[string][]FieldMapping          `json:"mappings"`
	Transforms  map[string]TransformDefinition     `json:"transforms"`
	Description string                             `json:"description"`
}

// IDiscoveryProvider 发现提供者接口
type IDiscoveryProvider interface {
	// GetMetadata 获取提供者元数据
	GetMetadata() *ProviderMetadata
	
	// GetParameterSchema 获取参数模式定义
	GetParameterSchema() []ParameterDefinition
	
	// GetFieldSchema 获取字段模式定义
	GetFieldSchema() []FieldDefinition
	
	// ValidateConfig 验证配置参数
	ValidateConfig(config map[string]interface{}) error
	
	// TestConnection 测试连接
	TestConnection(config map[string]interface{}) (*TestResult, error)
	
	// Discover 执行发现
	Discover(config map[string]interface{}) (*DiscoveryResult, error)
	
	// GetFieldMapping 获取字段映射配置
	GetFieldMapping(targetSchema string) (*FieldMappingConfig, error)
}
