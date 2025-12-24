package types

import (
	"time"
)

// AssetType 资产类型定义
type AssetType struct {
	ID          string        `json:"id"`          // 类型唯一标识
	Name        string        `json:"name"`        // 类型名称
	DisplayName string        `json:"displayName"` // 显示名称
	Icon        string        `json:"icon,omitempty"` // 图标
	Category    string        `json:"category,omitempty"` // 分类
	Description string        `json:"description,omitempty"` // 描述
	Searchable  bool          `json:"searchable"`  // 是否可搜索
	Selectable  bool          `json:"selectable"`  // 是否可选择
	Fields      []*AssetField `json:"fields"`      // 字段定义列表
	CreatedAt   time.Time     `json:"createdAt"`   // 创建时间
	UpdatedAt   time.Time     `json:"updatedAt"`   // 更新时间
	
	// 业务相关元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AssetField 资产字段定义
type AssetField struct {
	ID            string         `json:"id"`            // 字段唯一标识
	Name          string         `json:"name"`          // 字段名称
	DisplayName   string         `json:"displayName"`   // 显示名称
	Type          FieldType      `json:"type"`          // 字段类型
	Required      bool           `json:"required"`      // 是否必填
	Searchable    bool           `json:"searchable"`    // 是否可搜索
	DisplayInList bool           `json:"displayInList"` // 是否在列表中显示
	DisplayInCard bool           `json:"displayInCard"` // 是否在卡片中显示
	Sortable      bool           `json:"sortable"`      // 是否可排序
	Width         int            `json:"width,omitempty"` // 显示宽度
	Order         int            `json:"order"`         // 显示顺序
	
	// 字段验证规则
	Validation *FieldValidation `json:"validation,omitempty"`
	
	// 枚举类型的选项
	Options []*FieldOption `json:"options,omitempty"`
	
	// 字段格式化配置
	Formatter *FieldFormatter `json:"formatter,omitempty"`
	
	// 业务相关元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// FieldType 字段类型枚举
type FieldType string

const (
	FieldTypeText     FieldType = "text"     // 文本
	FieldTypeNumber   FieldType = "number"   // 数字
	FieldTypeInteger  FieldType = "integer"  // 整数
	FieldTypeFloat    FieldType = "float"    // 浮点数
	FieldTypeBoolean  FieldType = "boolean"  // 布尔值
	FieldTypeDate     FieldType = "date"     // 日期
	FieldTypeDateTime FieldType = "datetime" // 日期时间
	FieldTypeTime     FieldType = "time"     // 时间
	FieldTypeEnum     FieldType = "enum"     // 枚举
	FieldTypeArray    FieldType = "array"    // 数组
	FieldTypeObject   FieldType = "object"   // 对象
	FieldTypeJSON     FieldType = "json"     // JSON
	FieldTypeURL      FieldType = "url"      // URL链接
	FieldTypeEmail    FieldType = "email"    // 邮箱
	FieldTypeIP       FieldType = "ip"       // IP地址
	FieldTypeMAC      FieldType = "mac"      // MAC地址
	FieldTypeUUID     FieldType = "uuid"     // UUID
)

// FieldValidation 字段验证规则
type FieldValidation struct {
	MinLength    *int     `json:"minLength,omitempty"`    // 最小长度
	MaxLength    *int     `json:"maxLength,omitempty"`    // 最大长度
	MinValue     *float64 `json:"minValue,omitempty"`     // 最小值
	MaxValue     *float64 `json:"maxValue,omitempty"`     // 最大值
	Pattern      string   `json:"pattern,omitempty"`      // 正则表达式
	CustomRules  []string `json:"customRules,omitempty"`  // 自定义验证规则
}

// FieldOption 字段选项（用于枚举类型）
type FieldOption struct {
	Value       interface{}            `json:"value"`       // 选项值
	Label       string                 `json:"label"`       // 选项标签
	Description string                 `json:"description,omitempty"` // 选项描述
	Color       string                 `json:"color,omitempty"` // 颜色
	Icon        string                 `json:"icon,omitempty"`  // 图标
	Disabled    bool                   `json:"disabled"`    // 是否禁用
	Style       map[string]interface{} `json:"style,omitempty"` // 样式配置
}

// FieldFormatter 字段格式化配置
type FieldFormatter struct {
	Type       string                 `json:"type"`       // 格式化类型
	Template   string                 `json:"template,omitempty"` // 格式化模板
	DateFormat string                 `json:"dateFormat,omitempty"` // 日期格式
	Precision  *int                   `json:"precision,omitempty"` // 数字精度
	Unit       string                 `json:"unit,omitempty"` // 单位
	Options    map[string]interface{} `json:"options,omitempty"` // 其他选项
}

// Asset 资产实例
type Asset struct {
	ID          string                 `json:"id"`          // 资产唯一标识
	TypeID      string                 `json:"typeId"`      // 资产类型ID
	TypeName    string                 `json:"typeName"`    // 资产类型名称
	DisplayName string                 `json:"displayName"` // 显示名称
	Status      AssetStatus            `json:"status"`      // 资产状态
	
	// 动态属性值
	Attributes map[string]*AssetAttributeValue `json:"attributes"` // 属性值映射
	
	// 系统字段
	CreatedAt   time.Time `json:"createdAt"`   // 创建时间
	UpdatedAt   time.Time `json:"updatedAt"`   // 更新时间
	CreatedBy   string    `json:"createdBy"`   // 创建者
	UpdatedBy   string    `json:"updatedBy"`   // 更新者
	
	// 业务字段
	TenantID     string   `json:"tenantId,omitempty"`     // 租户ID
	DepartmentID string   `json:"departmentId,omitempty"` // 部门ID
	OwnerID      string   `json:"ownerId,omitempty"`      // 所有者ID
	Tags         []string `json:"tags,omitempty"`         // 标签
	
	// 业务相关元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AssetStatus 资产状态枚举
type AssetStatus string

const (
	AssetStatusActive      AssetStatus = "active"      // 活跃
	AssetStatusInactive    AssetStatus = "inactive"    // 非活跃
	AssetStatusMaintenance AssetStatus = "maintenance" // 维护中
	AssetStatusRetired     AssetStatus = "retired"     // 已退役
	AssetStatusDeleted     AssetStatus = "deleted"     // 已删除
)

// AssetAttributeValue 资产属性值
type AssetAttributeValue struct {
	FieldID     string      `json:"fieldId"`     // 字段ID
	FieldName   string      `json:"fieldName"`   // 字段名称
	FieldType   FieldType   `json:"fieldType"`   // 字段类型
	Value       interface{} `json:"value"`       // 属性值
	DisplayValue string     `json:"displayValue"` // 显示值（格式化后）
	RawValue    interface{} `json:"rawValue,omitempty"` // 原始值
}

// AssetDetail 资产详情（包含关联信息）
type AssetDetail struct {
	*Asset
	
	// 关联的资产类型定义
	Type *AssetType `json:"type,omitempty"`
	
	// 关联关系（可选）
	Relations []*AssetRelation `json:"relations,omitempty"`
	
	// 变更历史（可选）
	ChangeHistory []*AssetChange `json:"changeHistory,omitempty"`
}

// AssetRelation 资产关系
type AssetRelation struct {
	ID             string    `json:"id"`             // 关系ID
	RelationType   string    `json:"relationType"`   // 关系类型
	SourceAssetID  string    `json:"sourceAssetId"`  // 源资产ID
	TargetAssetID  string    `json:"targetAssetId"`  // 目标资产ID
	Direction      string    `json:"direction"`      // 关系方向
	Properties     map[string]interface{} `json:"properties,omitempty"` // 关系属性
	CreatedAt      time.Time `json:"createdAt"`      // 创建时间
}

// AssetChange 资产变更记录
type AssetChange struct {
	ID          string                 `json:"id"`          // 变更ID
	AssetID     string                 `json:"assetId"`     // 资产ID
	ChangeType  string                 `json:"changeType"`  // 变更类型
	FieldName   string                 `json:"fieldName,omitempty"` // 字段名称
	OldValue    interface{}            `json:"oldValue,omitempty"`  // 旧值
	NewValue    interface{}            `json:"newValue,omitempty"`  // 新值
	Reason      string                 `json:"reason,omitempty"`    // 变更原因
	CreatedAt   time.Time              `json:"createdAt"`   // 变更时间
	CreatedBy   string                 `json:"createdBy"`   // 变更者
	Metadata    map[string]interface{} `json:"metadata,omitempty"`  // 元数据
}

// AssetSelectorConfig 资产选择器配置
type AssetSelectorConfig struct {
	ID          string `json:"id"`          // 配置ID
	Name        string `json:"name"`        // 配置名称
	Description string `json:"description"` // 配置描述
	
	// 显示配置
	DisplayMode     string   `json:"displayMode"`     // 显示模式: table, card, list
	PageSize        int      `json:"pageSize"`        // 每页大小
	EnableSearch    bool     `json:"enableSearch"`    // 是否启用搜索
	EnableFilter    bool     `json:"enableFilter"`    // 是否启用过滤
	MultiSelect     bool     `json:"multiSelect"`     // 是否支持多选
	ShowTypeFilter  bool     `json:"showTypeFilter"`  // 是否显示类型过滤
	DefaultTypeIDs  []string `json:"defaultTypeIds"`  // 默认类型ID列表
	
	// 字段配置
	DisplayFields   []string `json:"displayFields"`   // 显示字段列表
	SearchFields    []string `json:"searchFields"`    // 搜索字段列表
	SortFields      []*SortField `json:"sortFields"`   // 默认排序字段
	
	// 权限配置
	RequirePermission bool     `json:"requirePermission"` // 是否需要权限检查
	AllowedTypes      []string `json:"allowedTypes"`      // 允许的类型列表
	
	// UI配置
	Theme           string                 `json:"theme"`           // 主题
	CustomStyles    map[string]interface{} `json:"customStyles"`    // 自定义样式
	
	// 业务配置
	ValidationRules []string               `json:"validationRules"` // 验证规则
	Metadata        map[string]interface{} `json:"metadata"`        // 元数据
	
	CreatedAt time.Time `json:"createdAt"` // 创建时间
	UpdatedAt time.Time `json:"updatedAt"` // 更新时间
}