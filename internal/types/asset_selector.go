package types

import (
	"context"
	"time"
)

// AssetService 通用资产服务接口
// 定义了所有资产选择器需要的核心功能
type AssetService interface {
	// GetAssetTypes 获取资产类型列表
	GetAssetTypes(ctx context.Context, filter *AssetTypeFilter) ([]*AssetType, error)

	// GetAssets 获取资产列表（支持复杂查询）
	GetAssets(ctx context.Context, query *AssetQuery) (*AssetListResponse, error)

	// GetAssetDetail 获取资产详情
	GetAssetDetail(ctx context.Context, id string) (*AssetDetail, error)

	// GetSearchSuggestions 获取搜索建议（自动完成）
	GetSearchSuggestions(ctx context.Context, input string, searchContext *SearchContext) ([]*Suggestion, error)

	// ValidateAssetSelection 验证资产选择（业务规则验证）
	ValidateAssetSelection(ctx context.Context, assets []*Asset) (*ValidationResult, error)
}

// PermissionService 权限控制接口
// 提供统一的权限检查和数据过滤功能
type PermissionService interface {
	// CheckAssetTypeAccess 检查用户对特定资产类型的访问权限
	CheckAssetTypeAccess(ctx context.Context, userID string, assetTypeID string) (bool, error)

	// FilterVisibleAssets 过滤用户可见的资产列表
	FilterVisibleAssets(ctx context.Context, userID string, assets []*Asset) ([]*Asset, error)

	// GetUserAssetScope 获取用户的资产访问范围
	GetUserAssetScope(ctx context.Context, userID string) (*AssetScope, error)
}

// CacheService 缓存服务接口
// 提供多层缓存管理功能
type CacheService interface {
	// Get 获取缓存数据
	Get(ctx context.Context, key string) (interface{}, error)

	// Set 设置缓存数据
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete 删除缓存数据
	Delete(ctx context.Context, key string) error

	// InvalidatePattern 批量删除匹配模式的缓存
	InvalidatePattern(ctx context.Context, pattern string) error
}

// AssetTypeFilter 资产类型过滤条件
type AssetTypeFilter struct {
	IDs        []string `json:"ids,omitempty"`        // 类型ID列表
	Category   string   `json:"category,omitempty"`   // 类型分类
	Searchable *bool    `json:"searchable,omitempty"` // 是否可搜索
	Selectable *bool    `json:"selectable,omitempty"` // 是否可选择
	Keywords   string   `json:"keywords,omitempty"`   // 关键词搜索
}

// AssetQuery 资产查询条件
type AssetQuery struct {
	// 基础分页参数
	Page     int `json:"page" validate:"min=1"`
	PageSize int `json:"pageSize" validate:"min=1,max=1000"`

	// 类型过滤
	TypeIDs []string `json:"typeIds,omitempty"`

	// 文本搜索
	Search       string   `json:"search,omitempty"`       // 全文搜索关键词
	SearchFields []string `json:"searchFields,omitempty"` // 指定搜索字段

	// 属性过滤
	AttributeFilters []*AttributeFilter `json:"attributeFilters,omitempty"`

	// 复合过滤条件
	FilterGroups []*FilterGroup `json:"filterGroups,omitempty"`

	// 排序
	SortFields []*SortField `json:"sortFields,omitempty"`

	// 返回字段控制
	IncludeFields []string `json:"includeFields,omitempty"` // 包含的字段
	ExcludeFields []string `json:"excludeFields,omitempty"` // 排除的字段
	WithDetails   bool     `json:"withDetails,omitempty"`   // 是否包含详细信息

	// 权限上下文
	UserID   string `json:"userId,omitempty"`
	TenantID string `json:"tenantId,omitempty"`
}

// AttributeFilter 属性过滤条件
type AttributeFilter struct {
	FieldID   string      `json:"fieldId"`             // 字段ID
	FieldName string      `json:"fieldName,omitempty"` // 字段名称
	Operator  string      `json:"operator"`            // 操作符: eq, ne, gt, lt, like, in, etc.
	Value     interface{} `json:"value,omitempty"`     // 过滤值
	Values    []interface{} `json:"values,omitempty"`   // 多个值（用于in操作符）
}

// FilterGroup 复合过滤条件组
type FilterGroup struct {
	Logic   string             `json:"logic"`   // 逻辑操作符: and, or
	Filters []*AttributeFilter `json:"filters"` // 属性过滤条件
	Groups  []*FilterGroup     `json:"groups"`  // 嵌套过滤组
}

// SortField 排序字段
type SortField struct {
	FieldID   string `json:"fieldId"`             // 字段ID
	FieldName string `json:"fieldName,omitempty"` // 字段名称
	Direction string `json:"direction"`           // 排序方向: asc, desc
	Priority  int    `json:"priority,omitempty"`  // 排序优先级
}

// SearchContext 搜索上下文
type SearchContext struct {
	UserID     string   `json:"userId,omitempty"`
	TypeIDs    []string `json:"typeIds,omitempty"`
	MaxResults int      `json:"maxResults,omitempty"`
}

// AssetListResponse 资产列表响应
type AssetListResponse struct {
	Total  int64    `json:"total"`  // 总数量
	Assets []*Asset `json:"assets"` // 资产列表
	// 聚合统计信息（可选）
	Aggregations map[string]interface{} `json:"aggregations,omitempty"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid   bool                     `json:"valid"`   // 是否有效
	Errors  []*ValidationError       `json:"errors"`  // 错误列表
	Warnings []*ValidationWarning    `json:"warnings"` // 警告列表
}

// ValidationError 验证错误
type ValidationError struct {
	Code     string `json:"code"`     // 错误代码
	Message  string `json:"message"`  // 错误消息
	Field    string `json:"field,omitempty"` // 相关字段
	AssetID  string `json:"assetId,omitempty"` // 相关资产ID
}

// ValidationWarning 验证警告
type ValidationWarning struct {
	Code    string `json:"code"`    // 警告代码
	Message string `json:"message"` // 警告消息
	AssetID string `json:"assetId,omitempty"` // 相关资产ID
}

// AssetScope 资产访问范围
type AssetScope struct {
	AllowedTypes []string `json:"allowedTypes"` // 允许访问的类型
	// 可以根据具体权限模型扩展
	DepartmentIDs []string               `json:"departmentIds,omitempty"`
	CustomRules   map[string]interface{} `json:"customRules,omitempty"`
}

// Suggestion 搜索建议
type Suggestion struct {
	Value       string `json:"value"`       // 建议值
	Label       string `json:"label"`       // 显示标签
	Type        string `json:"type"`        // 建议类型
	Description string `json:"description,omitempty"` // 描述
}