package provider

import (
	"context"
	"fmt"
)

// IDiscoveryProviderV2 新版Provider接口 - 支持context传递和超时控制
//
// 与V1接口的核心区别:
// 1. 所有方法都接收 context.Context 参数,支持:
//    - 超时控制(context.WithTimeout)
//    - 取消信号(context.WithCancel)
//    - 租户隔离信息传递(context.Value)
//    - 分布式追踪(OpenTelemetry)
//
// 2. 更严格的错误处理:
//    - 返回结构化错误类型
//    - 区分临时错误和永久错误
//
// 3. 向后兼容:
//    - 保留V1接口的元数据方法
//    - 通过ProviderV2Adapter包装V1实现
type IDiscoveryProviderV2 interface {
	// ============ 元数据方法 (与V1兼容) ============

	// GetMetadata 获取Provider元数据
	GetMetadata() *ProviderMetadata

	// GetParameterSchema 获取参数Schema定义
	GetParameterSchema() []ParameterDefinition

	// GetFieldSchema 获取字段Schema定义
	GetFieldSchema() []FieldDefinition

	// GetFieldMapping 获取字段映射配置
	GetFieldMapping(targetSchema string) (*FieldMappingConfig, error)

	// ============ V2新增方法 (支持context) ============

	// ValidateConfigWithContext 验证配置参数(带context)
	//
	// 用途:
	// - 在任务创建时验证配置完整性
	// - 在任务执行前验证配置有效性
	//
	// 参数:
	// - ctx: 携带超时控制和租户信息
	// - config: 用户提供的Provider配置
	//
	// 返回:
	// - error: 配置无效时返回详细错误信息
	ValidateConfigWithContext(ctx context.Context, config map[string]interface{}) error

	// TestConnectionWithContext 测试连接(带context)
	//
	// 用途:
	// - 验证凭据有效性
	// - 检查网络连通性
	// - 验证权限配置
	//
	// 参数:
	// - ctx: 携带超时控制(建议5-10秒)
	// - config: 连接配置(含凭据)
	//
	// 返回:
	// - TestResult: 测试结果详情
	// - error: 测试执行错误
	TestConnectionWithContext(ctx context.Context, config map[string]interface{}) (*TestResult, error)

	// DiscoverWithContext 执行数据发现(带context)
	//
	// 用途:
	// - 从数据源采集数据
	// - 返回标准化的发现结果
	//
	// 参数:
	// - ctx: 携带超时控制(长时间操作,建议30-300秒)和租户隔离信息
	// - config: 采集配置
	//
	// 返回:
	// - DiscoveryResult: 采集结果
	// - error: 采集错误(区分临时/永久错误)
	DiscoverWithContext(ctx context.Context, config map[string]interface{}) (*DiscoveryResult, error)
}

// ProviderV2Adapter 将V1 Provider适配为V2接口
//
// 用途:
// - 无需修改现有Provider实现代码
// - 自动添加context支持
// - 提供默认超时控制
//
// 示例:
//   v1Provider := &FileImportProvider{}
//   v2Provider := NewProviderV2Adapter(v1Provider)
//   result, err := v2Provider.DiscoverWithContext(ctx, config)
type ProviderV2Adapter struct {
	v1Provider IDiscoveryProvider
}

// NewProviderV2Adapter 创建V2适配器
func NewProviderV2Adapter(v1Provider IDiscoveryProvider) IDiscoveryProviderV2 {
	return &ProviderV2Adapter{
		v1Provider: v1Provider,
	}
}

// ============ 元数据方法 (直接转发到V1) ============

func (a *ProviderV2Adapter) GetMetadata() *ProviderMetadata {
	return a.v1Provider.GetMetadata()
}

func (a *ProviderV2Adapter) GetParameterSchema() []ParameterDefinition {
	return a.v1Provider.GetParameterSchema()
}

func (a *ProviderV2Adapter) GetFieldSchema() []FieldDefinition {
	return a.v1Provider.GetFieldSchema()
}

func (a *ProviderV2Adapter) GetFieldMapping(targetSchema string) (*FieldMappingConfig, error) {
	return a.v1Provider.GetFieldMapping(targetSchema)
}

// ============ V2方法 (包装V1方法,添加context支持) ============

func (a *ProviderV2Adapter) ValidateConfigWithContext(ctx context.Context, config map[string]interface{}) error {
	// 检查context是否已取消
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before validation: %w", err)
	}

	// 调用V1方法(V1方法不支持取消,但至少能检测到初始状态)
	return a.v1Provider.ValidateConfig(config)
}

func (a *ProviderV2Adapter) TestConnectionWithContext(ctx context.Context, config map[string]interface{}) (*TestResult, error) {
	// 检查context是否已取消
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled before test connection: %w", err)
	}

	// 使用goroutine执行V1方法,监听context取消
	resultChan := make(chan *TestResult, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := a.v1Provider.TestConnection(config)
		if err != nil {
			errChan <- err
		} else {
			resultChan <- result
		}
	}()

	// 等待结果或context取消
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("test connection cancelled: %w", ctx.Err())
	case err := <-errChan:
		return nil, err
	case result := <-resultChan:
		return result, nil
	}
}

func (a *ProviderV2Adapter) DiscoverWithContext(ctx context.Context, config map[string]interface{}) (*DiscoveryResult, error) {
	// 检查context是否已取消
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled before discovery: %w", err)
	}

	// 使用goroutine执行V1方法,监听context取消
	resultChan := make(chan *DiscoveryResult, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := a.v1Provider.Discover(config)
		if err != nil {
			errChan <- err
		} else {
			resultChan <- result
		}
	}()

	// 等待结果或context取消
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("discovery cancelled: %w", ctx.Err())
	case err := <-errChan:
		return nil, err
	case result := <-resultChan:
		return result, nil
	}
}

// ============ 辅助函数 ============

// IsProviderV2 检查Provider是否原生支持V2接口
func IsProviderV2(provider interface{}) bool {
	_, ok := provider.(IDiscoveryProviderV2)
	return ok
}

// ToProviderV2 将Provider转换为V2接口
//
// 逻辑:
// - 如果已经是V2接口,直接返回
// - 如果是V1接口,使用适配器包装
// - 否则返回错误
func ToProviderV2(provider interface{}) (IDiscoveryProviderV2, error) {
	// 检查是否已经是V2
	if v2, ok := provider.(IDiscoveryProviderV2); ok {
		return v2, nil
	}

	// 检查是否是V1
	if v1, ok := provider.(IDiscoveryProvider); ok {
		return NewProviderV2Adapter(v1), nil
	}

	return nil, fmt.Errorf("provider does not implement IDiscoveryProvider or IDiscoveryProviderV2")
}
