package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============ MockProvider Basic Tests ============

func TestMockProvider_GetMetadata(t *testing.T) {
	provider := NewMockProvider()
	metadata := provider.GetMetadata()

	assert.Equal(t, "mock", metadata.ID)
	assert.Equal(t, "Mock Provider", metadata.Name)
	assert.Equal(t, "mock", metadata.Type)
	assert.Contains(t, metadata.SupportedModes, "mock_mode")
}

func TestMockProvider_GetParameterSchema(t *testing.T) {
	provider := NewMockProvider()
	params := provider.GetParameterSchema()

	assert.NotEmpty(t, params)

	// 验证关键参数
	var foundRecordCount, foundFailOnPurpose, foundDelay bool
	for _, param := range params {
		switch param.Name {
		case "record_count":
			foundRecordCount = true
			assert.Equal(t, "integer", param.Type)
		case "fail_on_purpose":
			foundFailOnPurpose = true
			assert.Equal(t, "boolean", param.Type)
		case "delay_ms":
			foundDelay = true
			assert.Equal(t, "integer", param.Type)
		}
	}

	assert.True(t, foundRecordCount, "Should have record_count parameter")
	assert.True(t, foundFailOnPurpose, "Should have fail_on_purpose parameter")
	assert.True(t, foundDelay, "Should have delay_ms parameter")
}

func TestMockProvider_GetFieldSchema(t *testing.T) {
	provider := NewMockProvider()
	fields := provider.GetFieldSchema()

	assert.Len(t, fields, 3) // id, name, value

	fieldNames := make([]string, 0, len(fields))
	for _, field := range fields {
		fieldNames = append(fieldNames, field.Name)
	}

	assert.Contains(t, fieldNames, "id")
	assert.Contains(t, fieldNames, "name")
	assert.Contains(t, fieldNames, "value")
}

// ============ MockProvider Configuration Tests ============

func TestMockProvider_ValidateConfig_Success(t *testing.T) {
	provider := NewMockProvider()

	config := map[string]interface{}{
		"record_count": 10,
	}

	err := provider.ValidateConfig(config)
	assert.NoError(t, err)
	assert.Equal(t, 1, provider.ValidateCallCount)
}

func TestMockProvider_ValidateConfig_WithError(t *testing.T) {
	provider := NewMockProvider()
	provider.ValidateError = fmt.Errorf("mock validation error")

	config := map[string]interface{}{}

	err := provider.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock validation error")
}

func TestMockProvider_ValidateConfig_FailOnPurpose(t *testing.T) {
	provider := NewMockProvider()

	config := map[string]interface{}{
		"fail_on_purpose": true,
	}

	err := provider.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fail_on_purpose is true")
}

func TestMockProvider_TestConnection_Success(t *testing.T) {
	provider := NewMockProvider()

	config := map[string]interface{}{}

	result, err := provider.TestConnection(config)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "Mock connection test successful", result.Message)
	assert.Equal(t, 1, provider.TestConnectionCallCount)
}

func TestMockProvider_TestConnection_Failure(t *testing.T) {
	provider := NewMockProvider()
	provider.TestConnectionFail = true
	provider.TestConnectionMsg = "Custom failure message"

	config := map[string]interface{}{}

	result, err := provider.TestConnection(config)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "Custom failure message", result.Message)
}

func TestMockProvider_TestConnection_FailOnPurpose(t *testing.T) {
	provider := NewMockProvider()

	config := map[string]interface{}{
		"fail_on_purpose": true,
	}

	result, err := provider.TestConnection(config)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Message, "fail_on_purpose is true")
}

func TestMockProvider_TestConnection_WithDelay(t *testing.T) {
	provider := NewMockProvider()

	config := map[string]interface{}{
		"delay_ms": 100.0, // 100ms延迟
	}

	start := time.Now()
	result, err := provider.TestConnection(config)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.GreaterOrEqual(t, elapsed, 100*time.Millisecond, "Should have delayed at least 100ms")

	t.Logf("✅ Delay test: elapsed %v", elapsed)
}

// ============ MockProvider Discovery Tests ============

func TestMockProvider_Discover_DefaultGeneration(t *testing.T) {
	provider := NewMockProvider()

	config := map[string]interface{}{
		"record_count": 5.0,
	}

	result, err := provider.Discover(config)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, int64(5), result.TotalRecords)
	assert.Len(t, result.Records, 5)

	// 验证生成的记录格式
	record1 := result.Records[0]
	assert.Equal(t, "mock-1", record1["id"])
	assert.Equal(t, "Mock Record 1", record1["name"])
	assert.Equal(t, "Value 1", record1["value"])
	assert.Equal(t, 1, record1["index"])

	t.Logf("✅ Default generation: %d records", len(result.Records))
}

func TestMockProvider_Discover_WithPresetData(t *testing.T) {
	provider := NewMockProvider()

	// 设置预设数据
	presetData := []map[string]interface{}{
		{"id": "custom-1", "name": "Custom Record 1"},
		{"id": "custom-2", "name": "Custom Record 2"},
		{"id": "custom-3", "name": "Custom Record 3"},
	}
	provider.SetMockData(presetData)

	config := map[string]interface{}{}

	result, err := provider.Discover(config)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, int64(3), result.TotalRecords)
	assert.Equal(t, presetData, result.Records)

	// 验证元数据标记
	assert.Equal(t, true, result.Metadata["preset_data"])

	t.Logf("✅ Preset data: %d records", len(result.Records))
}

func TestMockProvider_Discover_WithError(t *testing.T) {
	provider := NewMockProvider()
	provider.DiscoverError = fmt.Errorf("mock discovery error")

	config := map[string]interface{}{}

	result, err := provider.Discover(config)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "mock discovery error")
	assert.Equal(t, 1, provider.DiscoverCallCount)
}

func TestMockProvider_Discover_FailOnPurpose(t *testing.T) {
	provider := NewMockProvider()

	config := map[string]interface{}{
		"fail_on_purpose": true,
	}

	result, err := provider.Discover(config)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "fail_on_purpose is true")
}

func TestMockProvider_GetFieldMapping(t *testing.T) {
	provider := NewMockProvider()

	mapping, err := provider.GetFieldMapping("default")
	require.NoError(t, err)
	assert.NotNil(t, mapping)
	assert.Equal(t, "1.0", mapping.Version)

	defaultMappings, exists := mapping.Mappings["default"]
	assert.True(t, exists)
	assert.Len(t, defaultMappings, 3) // id, name, value
}

// ============ MockProvider V2 Interface Tests ============

func TestMockProvider_ValidateConfigWithContext(t *testing.T) {
	provider := NewMockProvider()

	config := map[string]interface{}{}
	ctx := context.Background()

	err := provider.ValidateConfigWithContext(ctx, config)
	assert.NoError(t, err)
	assert.Equal(t, 1, provider.ValidateCallCount)
}

func TestMockProvider_ValidateConfigWithContext_Cancelled(t *testing.T) {
	provider := NewMockProvider()

	config := map[string]interface{}{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	err := provider.ValidateConfigWithContext(ctx, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")
}

func TestMockProvider_TestConnectionWithContext(t *testing.T) {
	provider := NewMockProvider()

	config := map[string]interface{}{}
	ctx := context.Background()

	result, err := provider.TestConnectionWithContext(ctx, config)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, 1, provider.TestConnectionCallCount)
}

func TestMockProvider_TestConnectionWithContext_Timeout(t *testing.T) {
	provider := NewMockProvider()

	config := map[string]interface{}{
		"delay_ms": 1000.0, // 1秒延迟
	}

	// 设置200ms超时
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	result, err := provider.TestConnectionWithContext(ctx, config)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "cancelled")

	t.Log("✅ Timeout test: context cancellation worked")
}

func TestMockProvider_DiscoverWithContext(t *testing.T) {
	provider := NewMockProvider()

	config := map[string]interface{}{
		"record_count": 3.0,
	}
	ctx := context.Background()

	result, err := provider.DiscoverWithContext(ctx, config)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, int64(3), result.TotalRecords)
	assert.True(t, provider.DiscoverWithContextCalled)
}

func TestMockProvider_DiscoverWithContext_Cancelled(t *testing.T) {
	provider := NewMockProvider()

	config := map[string]interface{}{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	result, err := provider.DiscoverWithContext(ctx, config)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "context cancelled")
}

func TestMockProvider_DiscoverWithContext_Timeout(t *testing.T) {
	provider := NewMockProvider()

	config := map[string]interface{}{
		"delay_ms": 1000.0, // 1秒延迟
	}

	// 设置200ms超时
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	result, err := provider.DiscoverWithContext(ctx, config)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "cancelled")

	t.Log("✅ Discovery timeout test: context cancellation worked")
}

// ============ MockProvider Helper Methods Tests ============

func TestMockProvider_SetMockData(t *testing.T) {
	provider := NewMockProvider()

	data := []map[string]interface{}{
		{"test": "data1"},
		{"test": "data2"},
	}

	provider.SetMockData(data)

	config := map[string]interface{}{}
	result, err := provider.Discover(config)
	require.NoError(t, err)
	assert.Equal(t, data, result.Records)
}

func TestMockProvider_SetFailure(t *testing.T) {
	provider := NewMockProvider()

	provider.SetFailure(
		fmt.Errorf("validate error"),
		true,
		fmt.Errorf("discover error"),
	)

	// 验证ValidateError
	err := provider.ValidateConfig(map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validate error")

	// 验证TestConnectionFail
	result, err := provider.TestConnection(map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result.Success)

	// 验证DiscoverError
	_, err = provider.Discover(map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "discover error")
}

func TestMockProvider_SetDelay(t *testing.T) {
	provider := NewMockProvider()
	provider.SetDelay(100 * time.Millisecond)

	start := time.Now()
	_, _ = provider.TestConnection(map[string]interface{}{})
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed, 100*time.Millisecond)
}

func TestMockProvider_ResetCounters(t *testing.T) {
	provider := NewMockProvider()

	// 调用几次方法
	_ = provider.ValidateConfig(map[string]interface{}{})
	_, _ = provider.TestConnection(map[string]interface{}{})
	_, _ = provider.Discover(map[string]interface{}{})

	assert.Greater(t, provider.ValidateCallCount, 0)
	assert.Greater(t, provider.TestConnectionCallCount, 0)
	assert.Greater(t, provider.DiscoverCallCount, 0)

	// 重置计数器
	provider.ResetCounters()

	assert.Equal(t, 0, provider.ValidateCallCount)
	assert.Equal(t, 0, provider.TestConnectionCallCount)
	assert.Equal(t, 0, provider.DiscoverCallCount)
	assert.False(t, provider.DiscoverWithContextCalled)
}

func TestMockProvider_GetCallStats(t *testing.T) {
	provider := NewMockProvider()

	// 调用几次方法
	_ = provider.ValidateConfig(map[string]interface{}{})
	_ = provider.ValidateConfig(map[string]interface{}{})
	_, _ = provider.TestConnection(map[string]interface{}{})
	ctx := context.Background()
	_, _ = provider.DiscoverWithContext(ctx, map[string]interface{}{})

	stats := provider.GetCallStats()

	assert.Equal(t, 2, stats["validate_calls"])
	assert.Equal(t, 1, stats["test_connection_calls"])
	assert.Equal(t, 1, stats["discover_calls"])
	assert.Equal(t, true, stats["discover_with_context"])

	t.Logf("✅ Call stats: %+v", stats)
}

// ============ MockProvider Factory Tests ============

func TestNewMockProvider(t *testing.T) {
	provider := NewMockProvider()

	assert.Equal(t, "mock", provider.ID)
	assert.Equal(t, "Mock Provider", provider.Name)
	assert.Equal(t, "1.0.0", provider.Version)
	assert.Nil(t, provider.DiscoverRecords)
	assert.Zero(t, provider.ValidateCallCount)
}

func TestNewMockProviderWithData(t *testing.T) {
	data := []map[string]interface{}{
		{"id": "1", "name": "Record 1"},
		{"id": "2", "name": "Record 2"},
	}

	provider := NewMockProviderWithData(data)

	assert.Equal(t, "mock", provider.ID)
	assert.Equal(t, data, provider.DiscoverRecords)

	// 验证Discover返回预设数据
	result, err := provider.Discover(map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, data, result.Records)
}

// ============ MockProvider Integration Scenarios ============

func TestMockProvider_CompleteWorkflow(t *testing.T) {
	provider := NewMockProvider()

	// 1. 获取元数据
	metadata := provider.GetMetadata()
	assert.Equal(t, "mock", metadata.ID)

	// 2. 准备配置
	config := map[string]interface{}{
		"record_count": 5.0,
	}

	// 3. 验证配置
	err := provider.ValidateConfig(config)
	require.NoError(t, err)

	// 4. 测试连接
	testResult, err := provider.TestConnection(config)
	require.NoError(t, err)
	assert.True(t, testResult.Success)

	// 5. 执行发现
	discoverResult, err := provider.Discover(config)
	require.NoError(t, err)
	assert.True(t, discoverResult.Success)
	assert.Equal(t, int64(5), discoverResult.TotalRecords)

	t.Log("✅ Complete workflow test passed")
}

func TestMockProvider_ErrorScenarioWorkflow(t *testing.T) {
	provider := NewMockProvider()

	// 设置失败场景
	provider.SetFailure(
		fmt.Errorf("config error"),
		true,
		fmt.Errorf("discovery error"),
	)

	// 1. 验证配置失败
	err := provider.ValidateConfig(map[string]interface{}{})
	assert.Error(t, err)

	// 2. 测试连接失败
	testResult, err := provider.TestConnection(map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, testResult.Success)

	// 3. 发现失败
	_, err = provider.Discover(map[string]interface{}{})
	assert.Error(t, err)

	t.Log("✅ Error scenario workflow test passed")
}

func TestMockProvider_WithV2Adapter(t *testing.T) {
	// 测试Mock Provider作为V1 Provider可以被V2 Adapter包装

	v1Provider := NewMockProvider()
	v2Provider, err := ToProviderV2(v1Provider)
	require.NoError(t, err)
	assert.NotNil(t, v2Provider)

	// 实际上Mock Provider原生支持V2，所以应该直接返回
	mockProviderV2, ok := v2Provider.(*MockProvider)
	assert.True(t, ok, "Mock Provider natively implements V2 interface")
	assert.Equal(t, v1Provider, mockProviderV2)
}

func TestMockProvider_V2WorkflowWithContext(t *testing.T) {
	provider := NewMockProvider()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config := map[string]interface{}{
		"record_count": 3.0,
	}

	// 1. 验证配置(带Context)
	err := provider.ValidateConfigWithContext(ctx, config)
	require.NoError(t, err)

	// 2. 测试连接(带Context)
	testResult, err := provider.TestConnectionWithContext(ctx, config)
	require.NoError(t, err)
	assert.True(t, testResult.Success)

	// 3. 执行发现(带Context)
	discoverResult, err := provider.DiscoverWithContext(ctx, config)
	require.NoError(t, err)
	assert.True(t, discoverResult.Success)
	assert.Equal(t, int64(3), discoverResult.TotalRecords)

	t.Log("✅ V2 workflow with context test passed")
}

// ============ Performance Benchmarks ============

func BenchmarkMockProvider_Discover(b *testing.B) {
	provider := NewMockProvider()
	config := map[string]interface{}{
		"record_count": 100.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.Discover(config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMockProvider_DiscoverWithContext(b *testing.B) {
	provider := NewMockProvider()
	config := map[string]interface{}{
		"record_count": 100.0,
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.DiscoverWithContext(ctx, config)
		if err != nil {
			b.Fatal(err)
		}
	}
}
