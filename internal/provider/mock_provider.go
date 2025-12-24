package provider

import (
	"context"
	"fmt"
	"time"
)

// MockProvider 用于测试的Mock Provider
//
// 特性:
// - 可配置的响应数据
// - 可模拟错误场景
// - 可模拟延迟
// - 支持V2接口(原生Context支持)
type MockProvider struct {
	// Metadata
	ID          string
	Name        string
	Description string
	Type        string
	Version     string

	// Behavior Configuration
	ValidateError      error                    // ValidateConfig时返回的错误
	TestConnectionFail bool                     // TestConnection是否失败
	TestConnectionMsg  string                   // TestConnection消息
	DiscoverError      error                    // Discover时返回的错误
	DiscoverRecords    []map[string]interface{} // Discover返回的记录
	Delay              time.Duration            // 模拟延迟

	// Statistics (for testing)
	ValidateCallCount         int
	TestConnectionCallCount   int
	DiscoverCallCount         int
	DiscoverWithContextCalled bool
}

// NewMockProvider 创建默认Mock Provider
func NewMockProvider() *MockProvider {
	return &MockProvider{
		ID:          "mock",
		Name:        "Mock Provider",
		Description: "Mock provider for testing",
		Type:        "mock",
		Version:     "1.0.0",
	}
}

// NewMockProviderWithData 创建带初始数据的Mock Provider
func NewMockProviderWithData(records []map[string]interface{}) *MockProvider {
	provider := NewMockProvider()
	provider.DiscoverRecords = records
	return provider
}

// ============ IDiscoveryProvider Interface (V1) ============

func (m *MockProvider) GetMetadata() *ProviderMetadata {
	return &ProviderMetadata{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Version:     m.Version,
		Type:        m.Type,
		SupportedModes: []string{
			"mock_mode",
		},
		Tags: []string{"mock", "testing"},
	}
}

func (m *MockProvider) GetParameterSchema() []ParameterDefinition {
	return []ParameterDefinition{
		{
			Name:         "record_count",
			DisplayName:  "记录数量",
			Type:         "integer",
			Required:     false,
			Description:  "返回的记录数量",
			DefaultValue: 10,
		},
		{
			Name:         "fail_on_purpose",
			DisplayName:  "故意失败",
			Type:         "boolean",
			Required:     false,
			Description:  "设置为true时模拟失败",
			DefaultValue: false,
		},
		{
			Name:         "delay_ms",
			DisplayName:  "延迟(毫秒)",
			Type:         "integer",
			Required:     false,
			Description:  "模拟延迟时间(毫秒)",
			DefaultValue: 0,
		},
	}
}

func (m *MockProvider) GetFieldSchema() []FieldDefinition {
	return []FieldDefinition{
		{
			Name:        "id",
			DisplayName: "ID",
			Type:        "string",
			Description: "记录ID",
			Required:    true,
			Unique:      true,
		},
		{
			Name:        "name",
			DisplayName: "名称",
			Type:        "string",
			Description: "记录名称",
			Required:    true,
		},
		{
			Name:        "value",
			DisplayName: "值",
			Type:        "string",
			Description: "记录值",
			Required:    false,
		},
	}
}

func (m *MockProvider) ValidateConfig(config map[string]interface{}) error {
	m.ValidateCallCount++

	// 如果配置了ValidateError，返回该错误
	if m.ValidateError != nil {
		return m.ValidateError
	}

	// 检查fail_on_purpose配置
	if failOnPurpose, exists := config["fail_on_purpose"]; exists {
		if fail, ok := failOnPurpose.(bool); ok && fail {
			return fmt.Errorf("mock validation failure: fail_on_purpose is true")
		}
	}

	return nil
}

func (m *MockProvider) TestConnection(config map[string]interface{}) (*TestResult, error) {
	m.TestConnectionCallCount++

	// 应用延迟
	if delay, exists := config["delay_ms"]; exists {
		var delayMs int
		switch v := delay.(type) {
		case float64:
			delayMs = int(v)
		case int:
			delayMs = v
		case int64:
			delayMs = int(v)
		}
		if delayMs > 0 {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}
	} else if m.Delay > 0 {
		time.Sleep(m.Delay)
	}

	// 如果配置了TestConnectionFail，返回失败
	if m.TestConnectionFail {
		msg := m.TestConnectionMsg
		if msg == "" {
			msg = "Mock connection test failed"
		}
		return &TestResult{
			Success: false,
			Message: msg,
		}, nil
	}

	// 检查fail_on_purpose配置
	if failOnPurpose, exists := config["fail_on_purpose"]; exists {
		if fail, ok := failOnPurpose.(bool); ok && fail {
			return &TestResult{
				Success: false,
				Message: "Mock test connection failure: fail_on_purpose is true",
			}, nil
		}
	}

	return &TestResult{
		Success: true,
		Message: "Mock connection test successful",
		Details: map[string]interface{}{
			"mock": true,
			"time": time.Now().Format(time.RFC3339),
		},
	}, nil
}

func (m *MockProvider) Discover(config map[string]interface{}) (*DiscoveryResult, error) {
	m.DiscoverCallCount++

	// 应用延迟
	if delay, exists := config["delay_ms"]; exists {
		var delayMs int
		switch v := delay.(type) {
		case float64:
			delayMs = int(v)
		case int:
			delayMs = v
		case int64:
			delayMs = int(v)
		}
		if delayMs > 0 {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}
	} else if m.Delay > 0 {
		time.Sleep(m.Delay)
	}

	// 如果配置了DiscoverError，返回该错误
	if m.DiscoverError != nil {
		return nil, m.DiscoverError
	}

	// 检查fail_on_purpose配置
	if failOnPurpose, exists := config["fail_on_purpose"]; exists {
		if fail, ok := failOnPurpose.(bool); ok && fail {
			return nil, fmt.Errorf("mock discovery failure: fail_on_purpose is true")
		}
	}

	// 如果预设了DiscoverRecords，返回它们
	if m.DiscoverRecords != nil {
		return &DiscoveryResult{
			Success:      true,
			TotalRecords: int64(len(m.DiscoverRecords)),
			Records:      m.DiscoverRecords,
			Metadata: map[string]interface{}{
				"provider":    "mock",
				"preset_data": true,
			},
		}, nil
	}

	// 根据record_count配置生成记录
	recordCount := 10
	if count, exists := config["record_count"]; exists {
		switch v := count.(type) {
		case float64:
			recordCount = int(v)
		case int:
			recordCount = v
		case int64:
			recordCount = int(v)
		}
	}

	records := make([]map[string]interface{}, recordCount)
	for i := 0; i < recordCount; i++ {
		records[i] = map[string]interface{}{
			"id":    fmt.Sprintf("mock-%d", i+1),
			"name":  fmt.Sprintf("Mock Record %d", i+1),
			"value": fmt.Sprintf("Value %d", i+1),
			"index": i + 1,
		}
	}

	return &DiscoveryResult{
		Success:      true,
		TotalRecords: int64(recordCount),
		Records:      records,
		Metadata: map[string]interface{}{
			"provider": "mock",
			"generated": true,
		},
	}, nil
}

func (m *MockProvider) GetFieldMapping(targetSchema string) (*FieldMappingConfig, error) {
	mappings := make(map[string][]FieldMapping)

	defaultMappings := []FieldMapping{
		{
			SourceField: "id",
			TargetField: "record_id",
			Transform:   "direct",
		},
		{
			SourceField: "name",
			TargetField: "record_name",
			Transform:   "direct",
		},
		{
			SourceField: "value",
			TargetField: "record_value",
			Transform:   "direct",
		},
	}

	mappings["default"] = defaultMappings

	return &FieldMappingConfig{
		Version:  "1.0",
		Mappings: mappings,
		Transforms: map[string]TransformDefinition{
			"direct": {
				Name:        "direct",
				Description: "Direct mapping without transformation",
				Parameters:  []string{},
			},
		},
		Description: "Mock provider field mapping configuration",
	}, nil
}

// ============ IDiscoveryProviderV2 Interface ============

func (m *MockProvider) ValidateConfigWithContext(ctx context.Context, config map[string]interface{}) error {
	// 检查context是否已取消
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before validation: %w", err)
	}

	return m.ValidateConfig(config)
}

func (m *MockProvider) TestConnectionWithContext(ctx context.Context, config map[string]interface{}) (*TestResult, error) {
	// 检查context是否已取消
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled before test connection: %w", err)
	}

	// 应用延迟时检查context
	delay := m.Delay
	if delayVal, exists := config["delay_ms"]; exists {
		var delayMs int
		switch v := delayVal.(type) {
		case float64:
			delayMs = int(v)
		case int:
			delayMs = v
		case int64:
			delayMs = int(v)
		}
		if delayMs > 0 {
			delay = time.Duration(delayMs) * time.Millisecond
		}
	}

	if delay > 0 {
		select {
		case <-time.After(delay):
			// 延迟结束
		case <-ctx.Done():
			// Context被取消
			return nil, fmt.Errorf("test connection cancelled: %w", ctx.Err())
		}
	}

	return m.TestConnection(config)
}

func (m *MockProvider) DiscoverWithContext(ctx context.Context, config map[string]interface{}) (*DiscoveryResult, error) {
	m.DiscoverWithContextCalled = true

	// 检查context是否已取消
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled before discovery: %w", err)
	}

	// 应用延迟时检查context
	delay := m.Delay
	if delayVal, exists := config["delay_ms"]; exists {
		var delayMs int
		switch v := delayVal.(type) {
		case float64:
			delayMs = int(v)
		case int:
			delayMs = v
		case int64:
			delayMs = int(v)
		}
		if delayMs > 0 {
			delay = time.Duration(delayMs) * time.Millisecond
		}
	}

	if delay > 0 {
		select {
		case <-time.After(delay):
			// 延迟结束
		case <-ctx.Done():
			// Context被取消
			return nil, fmt.Errorf("discovery cancelled: %w", ctx.Err())
		}
	}

	return m.Discover(config)
}

// ============ Helper Methods for Testing ============

// SetMockData 设置Mock返回的数据
func (m *MockProvider) SetMockData(records []map[string]interface{}) {
	m.DiscoverRecords = records
}

// SetFailure 设置失败场景
func (m *MockProvider) SetFailure(validateErr error, testConnFail bool, discoverErr error) {
	m.ValidateError = validateErr
	m.TestConnectionFail = testConnFail
	m.DiscoverError = discoverErr
}

// SetDelay 设置延迟
func (m *MockProvider) SetDelay(delay time.Duration) {
	m.Delay = delay
}

// ResetCounters 重置调用计数器
func (m *MockProvider) ResetCounters() {
	m.ValidateCallCount = 0
	m.TestConnectionCallCount = 0
	m.DiscoverCallCount = 0
	m.DiscoverWithContextCalled = false
}

// GetCallStats 获取调用统计
func (m *MockProvider) GetCallStats() map[string]interface{} {
	return map[string]interface{}{
		"validate_calls":           m.ValidateCallCount,
		"test_connection_calls":    m.TestConnectionCallCount,
		"discover_calls":           m.DiscoverCallCount,
		"discover_with_context":    m.DiscoverWithContextCalled,
	}
}
