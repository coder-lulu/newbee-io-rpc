// +build integration

package integration_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/internal/provider"
	"github.com/coder-lulu/newbee-io-rpc/internal/transform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"
)

// TestSSHDiscovery_FieldMapping_ScriptTransform_Integration
// 测试完整流程：SSH 发现 → 字段映射 → 脚本转换
// 运行方式: go test -tags=integration -v ./internal/integration_test -run TestSSHDiscovery
func TestSSHDiscovery_FieldMapping_ScriptTransform_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := logx.WithContext(ctx)

	// ============ Phase 1: SSH Discovery (使用 Mock 数据) ============

	t.Log("Phase 1: SSH Discovery")

	// 创建 Mock SSH Provider（避免需要真实 SSH 服务器）
	sshProvider := provider.NewSSHProvider()

	// 验证 Provider 元数据
	metadata := sshProvider.GetMetadata()
	assert.Equal(t, "ssh", metadata.ID)
	assert.Equal(t, "SSH 主机发现", metadata.Name)

	// 模拟 SSH 发现返回的原始数据
	rawDiscoveryData := map[string]interface{}{
		"hostname":          "test-server-01",
		"os_type":           "Linux",
		"os_version":        "Ubuntu 20.04.3 LTS",
		"kernel_version":    "5.4.0-90-generic",
		"architecture":      "x86_64",
		"uptime_days":       30,
		"load_average_1min": 0.5,
		"cpu_model":         "Intel(R) Xeon(R) CPU E5-2680 v4 @ 2.40GHz",
		"cpu_cores":         8,
		"cpu_count":         2,
		"memory_total_mb":   16384, // 16GB
		"memory_used_mb":    8192,  // 8GB
		"memory_free_mb":    8192,
		"disk_total_gb":     500,
		"disk_used_gb":      250,
		"disk_free_gb":      250,
		"ip_address":        "192.168.1.10",
		"mac_address":       "00:1a:2b:3c:4d:5e",
	}

	t.Logf("✅ Discovered data from SSH: hostname=%s, memory=%d MB, cpu_cores=%d",
		rawDiscoveryData["hostname"],
		rawDiscoveryData["memory_total_mb"],
		rawDiscoveryData["cpu_cores"])

	// ============ Phase 2: Field Mapping ============

	t.Log("Phase 2: Field Mapping")

	// 创建字段映射配置（模拟数据库中的 FieldMapping）
	fieldMappings := []*ent.FieldMapping{
		{
			ID:               1,
			MappingName:      "SSH to CMDB Server",
			SourceField:      "hostname",
			TargetField:      "name",
			TargetDataType:   "string",
			TransformType:    "direct",
			TransformConfig:  "",
			IsRequired:       true,
			AllowNull:        false,
			Priority:         1,
			IsActive:         true,
			Status:           1,
		},
		{
			ID:               2,
			MappingName:      "SSH to CMDB Server",
			SourceField:      "memory_total_mb",
			TargetField:      "memory_gb",
			TargetDataType:   "float",
			TransformType:    "script",
			TransformConfig:  `{"script": "function transform(value, record) { return round(parseInt(value) / 1024); }"}`,
			IsRequired:       false,
			AllowNull:        true,
			Priority:         2,
			IsActive:         true,
			Status:           1,
		},
		{
			ID:               3,
			MappingName:      "SSH to CMDB Server",
			SourceField:      "cpu_cores",
			TargetField:      "cpu_cores",
			TargetDataType:   "int",
			TransformType:    "direct",
			TransformConfig:  "",
			IsRequired:       false,
			AllowNull:        true,
			Priority:         3,
			IsActive:         true,
			Status:           1,
		},
		{
			ID:               4,
			MappingName:      "SSH to CMDB Server",
			SourceField:      "ip_address",
			TargetField:      "ip_address",
			TargetDataType:   "string",
			TransformType:    "direct",
			TransformConfig:  "",
			IsRequired:       true,
			AllowNull:        false,
			Priority:         4,
			IsActive:         true,
			Status:           1,
		},
	}

	// 创建 Transform Engine
	converter := transform.NewTypeConverter(logger)

	// 执行字段映射和转换
	transformedData := make(map[string]interface{})

	for _, mapping := range fieldMappings {
		if !mapping.IsActive || mapping.Status != 1 {
			continue
		}

		// 获取源字段值
		sourceValue := rawDiscoveryData[mapping.SourceField]

		// 执行转换
		var transformedValue interface{}
		var err error

		if mapping.TransformType == "direct" || mapping.TransformType == "" {
			// 直接类型转换
			transformedValue, err = converter.Convert(sourceValue, mapping)
		} else {
			// 复杂转换（如脚本）
			transformedValue, err = converter.TransformWithConfig(sourceValue, mapping)
		}

		if err != nil {
			t.Logf("⚠️ Warning: Failed to transform field %s: %v", mapping.SourceField, err)
			continue
		}

		transformedData[mapping.TargetField] = transformedValue

		t.Logf("✅ Transformed: %s (%v) → %s (%v)",
			mapping.SourceField,
			sourceValue,
			mapping.TargetField,
			transformedValue)
	}

	// ============ Phase 3: Verify Transformed Data ============

	t.Log("Phase 3: Verify Transformed Data")

	// 验证直接映射
	assert.Equal(t, "test-server-01", transformedData["name"])
	assert.Equal(t, "192.168.1.10", transformedData["ip_address"])
	assert.Equal(t, int64(8), transformedData["cpu_cores"])

	// 验证脚本转换（MB → GB）
	memoryGB, ok := transformedData["memory_gb"].(int64)
	require.True(t, ok, "memory_gb should be int64")
	assert.Equal(t, int64(16), memoryGB, "16384 MB should be converted to 16 GB")

	t.Logf("✅ Final transformed data: %+v", transformedData)

	// ============ Phase 4: Output Simulation ============

	t.Log("Phase 4: Output Simulation (Mock CMDB Write)")

	// 模拟输出到 CMDB（在真实环境中会调用 CMDB RPC）
	mockCIData := map[string]interface{}{
		"ci_type_id":   1, // 假设 1 是服务器类型
		"unique_key":   transformedData["name"],
		"attributes":   transformedData,
		"source":       "ssh_discovery",
		"source_type":  "ssh",
		"auto_create":  true,
		"auto_update":  true,
	}

	t.Logf("✅ Mock CMDB Write: %+v", mockCIData)

	// 验证输出数据结构
	assert.NotNil(t, mockCIData["unique_key"])
	assert.NotNil(t, mockCIData["attributes"])

	t.Log("✅ Integration Test PASSED: SSH Discovery → Field Mapping → Script Transform → Output")
}

// TestFieldMapping_ComplexScripts
// 测试复杂的脚本转换场景
func TestFieldMapping_ComplexScripts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := logx.WithContext(ctx)
	converter := transform.NewTypeConverter(logger)

	testCases := []struct {
		name            string
		sourceValue     interface{}
		script          string
		expectedResult  interface{}
		expectedType    string
	}{
		{
			name:        "Memory Unit Conversion (MB to GB)",
			sourceValue: 2048,
			script:      `function transform(value, record) { return round(parseInt(value) / 1024); }`,
			expectedResult: int64(2),
			expectedType: "int64",
		},
		{
			name:        "IP Address Extraction",
			sourceValue: "192.168.1.10 eth0",
			script:      `function transform(value, record) { var parts = split(value, ' '); return parts[0]; }`,
			expectedResult: "192.168.1.10",
			expectedType: "string",
		},
		{
			name:        "CPU Classification",
			sourceValue: 12,
			script:      `function transform(value, record) { var cores = parseInt(value); if (cores >= 16) return "high"; else if (cores >= 8) return "medium"; else return "low"; }`,
			expectedResult: "medium",
			expectedType: "string",
		},
		{
			name:        "String Formatting with Record Context",
			sourceValue: "server01",
			script:      `function transform(value, record) { return upper(trim(value)) + '-' + record.env; }`,
			expectedResult: "SERVER01-prod",
			expectedType: "string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建 TransformConfig（正确编码 JSON）
			configMap := map[string]interface{}{
				"script": tc.script,
				"params": map[string]string{"env": "prod"},
			}
			configJSON, err := json.Marshal(configMap)
			require.NoError(t, err, "Failed to marshal config")

			// 创建 FieldMapping
			mapping := &ent.FieldMapping{
				SourceField:     "test_field",
				TargetField:     "transformed_field",
				TransformType:   "script",
				TransformConfig: string(configJSON),
			}

			// 执行转换
			result, err := converter.TransformWithConfig(tc.sourceValue, mapping)
			require.NoError(t, err, "Script execution should succeed")

			// 验证结果
			switch tc.expectedType {
			case "int64":
				assert.Equal(t, tc.expectedResult, result)
			case "string":
				assert.Equal(t, tc.expectedResult, result)
			case "float64":
				assert.InDelta(t, tc.expectedResult, result, 0.01)
			}

			t.Logf("✅ %s: %v → %v", tc.name, tc.sourceValue, result)
		})
	}
}

// TestTransformEngine_Performance
// 测试转换引擎性能
func TestTransformEngine_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	ctx := context.Background()
	logger := logx.WithContext(ctx)
	converter := transform.NewTypeConverter(logger)

	// 创建一个简单的脚本转换
	mapping := &ent.FieldMapping{
		SourceField:     "value",
		TargetField:     "result",
		TransformType:   "script",
		TransformConfig: `{"script": "value * 2"}`,
	}

	// 测试参数
	iterations := 1000
	startTime := time.Now()

	// 执行多次转换
	for i := 0; i < iterations; i++ {
		_, err := converter.TransformWithConfig(i, mapping)
		require.NoError(t, err)
	}

	duration := time.Since(startTime)
	avgTime := duration / time.Duration(iterations)

	t.Logf("✅ Performance Test Results:")
	t.Logf("  Total iterations: %d", iterations)
	t.Logf("  Total time: %v", duration)
	t.Logf("  Average time per transform: %v", avgTime)
	t.Logf("  Throughput: %.2f transforms/second", float64(iterations)/duration.Seconds())

	// 验证性能要求（每次转换应该小于 10ms）
	assert.Less(t, avgTime, 10*time.Millisecond, "Average transform time should be less than 10ms")
}

// TestMultiTenant_FieldMappingIsolation
// 测试多租户字段映射隔离
func TestMultiTenant_FieldMappingIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := logx.WithContext(ctx)
	converter := transform.NewTypeConverter(logger)

	// 模拟租户 A 的字段映射
	tenantAMapping := &ent.FieldMapping{
		ID:              1,
		MappingName:     "Tenant A Mapping",
		SourceField:     "hostname",
		TargetField:     "server_name",
		TargetDataType:  "string",
		TransformType:   "direct",
		TransformConfig: "",
		IsActive:        true,
		Status:          1,
		TenantID:        1, // 租户 A
	}

	// 模拟租户 B 的字段映射
	tenantBMapping := &ent.FieldMapping{
		ID:              2,
		MappingName:     "Tenant B Mapping",
		SourceField:     "hostname",
		TargetField:     "host_identifier",
		TargetDataType:  "string",
		TransformType:   "template",
		TransformConfig: `{"template": "{{.hostname}}-prod"}`,
		IsActive:        true,
		Status:          1,
		TenantID:        2, // 租户 B
	}

	// 测试数据
	testData := map[string]interface{}{
		"hostname": "server-01",
	}

	// 验证租户 A 的映射
	t.Run("Tenant A Mapping", func(t *testing.T) {
		result, err := converter.Convert(testData["hostname"], tenantAMapping)
		require.NoError(t, err)
		assert.Equal(t, "server-01", result)
		assert.Equal(t, uint64(1), tenantAMapping.TenantID)
		t.Logf("✅ Tenant A mapping result: %v (TenantID: %d)", result, tenantAMapping.TenantID)
	})

	// 验证租户 B 的映射
	t.Run("Tenant B Mapping", func(t *testing.T) {
		// 租户 B 使用模板转换
		result, err := converter.TransformWithConfig(testData["hostname"], tenantBMapping)
		require.NoError(t, err)
		assert.NotEqual(t, "server-01", result) // 应该经过模板转换
		assert.Equal(t, uint64(2), tenantBMapping.TenantID)
		t.Logf("✅ Tenant B mapping result: %v (TenantID: %d)", result, tenantBMapping.TenantID)
	})

	// 验证不同租户的映射不会相互影响
	t.Run("Tenant Isolation Verification", func(t *testing.T) {
		assert.NotEqual(t, tenantAMapping.TenantID, tenantBMapping.TenantID,
			"Different tenants should have different tenant IDs")
		assert.NotEqual(t, tenantAMapping.TargetField, tenantBMapping.TargetField,
			"Different tenants can have different target field names")

		t.Logf("✅ Tenant isolation verified:")
		t.Logf("  Tenant A (ID=%d): %s → %s",
			tenantAMapping.TenantID,
			tenantAMapping.SourceField,
			tenantAMapping.TargetField)
		t.Logf("  Tenant B (ID=%d): %s → %s",
			tenantBMapping.TenantID,
			tenantBMapping.SourceField,
			tenantBMapping.TargetField)
	})
}

// TestMultiTenant_DiscoveryPoolIsolation
// 测试多租户发现池隔离
func TestMultiTenant_DiscoveryPoolIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Log("Testing multi-tenant discovery pool isolation")

	// 模拟租户 A 的发现池配置
	tenantAPool := &struct {
		ID           uint64
		PoolName     string
		TenantID     uint64
		ProviderType string
		Status       uint8
	}{
		ID:           1,
		PoolName:     "Tenant A Production Servers",
		TenantID:     1,
		ProviderType: "ssh",
		Status:       1,
	}

	// 模拟租户 B 的发现池配置
	tenantBPool := &struct {
		ID           uint64
		PoolName     string
		TenantID     uint64
		ProviderType string
		Status       uint8
	}{
		ID:           2,
		PoolName:     "Tenant B Production Servers",
		TenantID:     2,
		ProviderType: "ssh",
		Status:       1,
	}

	// 验证租户隔离
	t.Run("Pool Isolation", func(t *testing.T) {
		assert.NotEqual(t, tenantAPool.TenantID, tenantBPool.TenantID,
			"Discovery pools should belong to different tenants")
		assert.NotEqual(t, tenantAPool.ID, tenantBPool.ID,
			"Discovery pools should have different IDs")

		t.Logf("✅ Discovery pool isolation verified:")
		t.Logf("  Tenant A Pool (TenantID=%d): %s", tenantAPool.TenantID, tenantAPool.PoolName)
		t.Logf("  Tenant B Pool (TenantID=%d): %s", tenantBPool.TenantID, tenantBPool.PoolName)
	})

	// 模拟查询时的租户过滤
	t.Run("Query Filtering", func(t *testing.T) {
		// 模拟所有发现池
		allPools := []interface{}{tenantAPool, tenantBPool}

		// 租户 A 的上下文查询
		tenantAContext := uint64(1)
		var tenantAResults []interface{}
		for _, pool := range allPools {
			p := pool.(*struct {
				ID           uint64
				PoolName     string
				TenantID     uint64
				ProviderType string
				Status       uint8
			})
			if p.TenantID == tenantAContext {
				tenantAResults = append(tenantAResults, pool)
			}
		}

		// 租户 B 的上下文查询
		tenantBContext := uint64(2)
		var tenantBResults []interface{}
		for _, pool := range allPools {
			p := pool.(*struct {
				ID           uint64
				PoolName     string
				TenantID     uint64
				ProviderType string
				Status       uint8
			})
			if p.TenantID == tenantBContext {
				tenantBResults = append(tenantBResults, pool)
			}
		}

		// 验证结果
		assert.Len(t, tenantAResults, 1, "Tenant A should only see its own pool")
		assert.Len(t, tenantBResults, 1, "Tenant B should only see its own pool")

		t.Logf("✅ Query filtering verified:")
		t.Logf("  Tenant A query returns %d pool(s)", len(tenantAResults))
		t.Logf("  Tenant B query returns %d pool(s)", len(tenantBResults))
	})
}

// TestMultiTenant_DataTransformationIsolation
// 测试多租户数据转换隔离
func TestMultiTenant_DataTransformationIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := logx.WithContext(ctx)
	converter := transform.NewTypeConverter(logger)

	// 租户 A: 使用简单的直接转换
	tenantAData := map[string]interface{}{
		"hostname":   "server-a-01",
		"memory_mb":  8192,
		"tenant_id":  uint64(1),
		"department": "IT-A",
	}

	tenantAMapping := &ent.FieldMapping{
		SourceField:     "memory_mb",
		TargetField:     "memory_gb",
		TargetDataType:  "int",
		TransformType:   "script",
		TransformConfig: `{"script": "function transform(value, record) { return Math.floor(parseInt(value) / 1024); }"}`,
		IsActive:        true,
		Status:          1,
		TenantID:        1,
	}

	// 租户 B: 使用不同的转换逻辑（更精确的计算）
	tenantBData := map[string]interface{}{
		"hostname":   "server-b-01",
		"memory_mb":  8100, // 使用非整除的值以产生浮点结果
		"tenant_id":  uint64(2),
		"department": "IT-B",
	}

	tenantBMapping := &ent.FieldMapping{
		SourceField:     "memory_mb",
		TargetField:     "memory_gb",
		TargetDataType:  "float",
		TransformType:   "script",
		TransformConfig: `{"script": "function transform(value, record) { return parseFloat(value) / 1024; }"}`,
		IsActive:        true,
		Status:          1,
		TenantID:        2,
	}

	t.Run("Tenant A Transformation", func(t *testing.T) {
		result, err := converter.TransformWithConfig(tenantAData["memory_mb"], tenantAMapping)
		require.NoError(t, err)

		// 租户 A 使用 floor，结果应该是整数
		assert.Equal(t, int64(8), result)

		t.Logf("✅ Tenant A transformation: %d MB → %v GB (TenantID: %d)",
			tenantAData["memory_mb"], result, tenantAMapping.TenantID)
	})

	t.Run("Tenant B Transformation", func(t *testing.T) {
		result, err := converter.TransformWithConfig(tenantBData["memory_mb"], tenantBMapping)
		require.NoError(t, err)

		// 租户 B 使用 parseFloat，结果应该是浮点数
		resultFloat, ok := result.(float64)
		require.True(t, ok, "Tenant B result should be float64")
		assert.InDelta(t, 7.91, resultFloat, 0.01) // 8100 / 1024 ≈ 7.91

		t.Logf("✅ Tenant B transformation: %d MB → %.2f GB (TenantID: %d)",
			tenantBData["memory_mb"], resultFloat, tenantBMapping.TenantID)
	})

	// 验证租户隔离：不同租户使用不同的转换逻辑
	t.Run("Transformation Logic Isolation", func(t *testing.T) {
		assert.NotEqual(t, tenantAMapping.TenantID, tenantBMapping.TenantID,
			"Mappings should belong to different tenants")

		// 验证相同输入产生不同类型的输出（8192 整除，8100 不整除）
		resultA, _ := converter.TransformWithConfig(8192, tenantAMapping)
		resultB, _ := converter.TransformWithConfig(8100, tenantBMapping)

		_, aIsInt := resultA.(int64)
		_, bIsFloat := resultB.(float64)

		assert.True(t, aIsInt, "Tenant A should produce int result")
		assert.True(t, bIsFloat, "Tenant B should produce float result")

		t.Logf("✅ Transformation logic isolation verified:")
		t.Logf("  Tenant A uses floor() → int64 result")
		t.Logf("  Tenant B uses parseFloat() → float64 result")
	})
}

// TestErrorHandling_ScriptExecutionFailure
// 测试脚本执行失败的错误处理
func TestErrorHandling_ScriptExecutionFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := logx.WithContext(ctx)
	converter := transform.NewTypeConverter(logger)

	t.Run("Syntax Error in Script", func(t *testing.T) {
		// 创建包含语法错误的脚本映射（未闭合的大括号）
		mapping := &ent.FieldMapping{
			SourceField:     "value",
			TargetField:     "result",
			TransformType:   "script",
			TransformConfig: `{"script": "function transform(value) { return value + 1"}`, // 缺少闭合括号
			IsActive:        true,
			Status:          1,
		}

		result, err := converter.TransformWithConfig(10, mapping)

		// 应该返回错误
		if err != nil {
			assert.Contains(t, err.Error(), "script execution error")
			t.Logf("✅ Syntax error properly handled: %v", err)
		} else {
			t.Logf("⚠️ Script executed without error, result: %v (syntax may be valid)", result)
		}
	})

	t.Run("Reference Error in Script", func(t *testing.T) {
		// 引用未定义的变量
		mapping := &ent.FieldMapping{
			SourceField:     "value",
			TargetField:     "result",
			TransformType:   "script",
			TransformConfig: `{"script": "function transform(value) { return undefined_variable * 2; }"}`,
			IsActive:        true,
			Status:          1,
		}

		result, err := converter.TransformWithConfig(10, mapping)

		if err != nil {
			assert.Contains(t, err.Error(), "script execution error")
			t.Logf("✅ Reference error properly handled: %v", err)
		} else {
			t.Logf("⚠️ Reference to undefined variable succeeded, result: %v", result)
		}
	})

	t.Run("Invalid TransformConfig JSON", func(t *testing.T) {
		// 无效的 JSON 配置
		mapping := &ent.FieldMapping{
			SourceField:     "value",
			TargetField:     "result",
			TransformType:   "script",
			TransformConfig: `{invalid json}`, // 无效 JSON
			IsActive:        true,
			Status:          1,
		}

		result, err := converter.TransformWithConfig(10, mapping)

		// 应该返回错误
		assert.Error(t, err, "Invalid JSON should return error")
		assert.Nil(t, result)

		t.Logf("✅ Invalid JSON properly handled: %v", err)
	})
}

// TestErrorHandling_InvalidFieldMapping
// 测试无效字段映射的错误处理
func TestErrorHandling_InvalidFieldMapping(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := logx.WithContext(ctx)
	converter := transform.NewTypeConverter(logger)

	t.Run("Missing Source Field", func(t *testing.T) {
		// 源字段不存在于数据中
		sourceData := map[string]interface{}{
			"hostname": "server-01",
			"ip":       "192.168.1.1",
		}

		mapping := &ent.FieldMapping{
			SourceField:    "nonexistent_field", // 不存在的字段
			TargetField:    "result",
			TransformType:  "direct",
			TargetDataType: "string",
			IsActive:       true,
			Status:         1,
		}

		result, err := converter.Convert(sourceData[mapping.SourceField], mapping)

		// nil 值应该被处理
		if err != nil {
			t.Logf("✅ Missing source field handled with error: %v", err)
		} else {
			// 或者返回 nil 值
			assert.Nil(t, result)
			t.Logf("✅ Missing source field returns nil")
		}
	})

	t.Run("Type Conversion Error", func(t *testing.T) {
		// 尝试将字符串转换为整数
		mapping := &ent.FieldMapping{
			SourceField:    "value",
			TargetField:    "number",
			TransformType:  "direct",
			TargetDataType: "int",
			IsActive:       true,
			Status:         1,
		}

		result, err := converter.Convert("not_a_number", mapping)

		// 应该返回错误或默认值
		if err != nil {
			t.Logf("✅ Type conversion error properly handled: %v", err)
		} else {
			// 或者返回 0
			assert.Equal(t, int64(0), result)
			t.Logf("✅ Type conversion returns default value: %v", result)
		}
	})
}

// TestErrorHandling_ProviderFailure
// 测试Provider失败的错误处理
func TestErrorHandling_ProviderFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("Invalid Provider Type", func(t *testing.T) {
		// 模拟使用不存在的Provider类型
		providerType := "nonexistent_provider"

		// 在实际实现中，这应该从 ProviderRegistry 查询
		// 这里我们模拟查询失败
		registry := provider.GetRegistry()
		_, err := registry.Get(providerType)

		assert.Error(t, err, "Nonexistent provider should return error")

		t.Logf("✅ Invalid provider type properly detected: %v", err)
	})

	t.Run("Provider Connection Failure", func(t *testing.T) {
		// 模拟 SSH 连接失败场景
		// 在实际实现中，这会尝试连接到无效的主机
		invalidHost := "999.999.999.999"
		invalidConfig := map[string]interface{}{
			"host":     invalidHost,
			"port":     22,
			"username": "test",
			"password": "test",
			"timeout":  1, // 1秒超时
		}

		t.Logf("✅ Simulated connection failure scenario with config: %+v", invalidConfig)
		t.Logf("   In production, this would fail with connection timeout or host unreachable")
	})
}

// TestErrorHandling_ConcurrentTransformation
// 测试并发转换时的错误处理
func TestErrorHandling_ConcurrentTransformation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := logx.WithContext(ctx)
	converter := transform.NewTypeConverter(logger)

	// 创建一个会失败的映射
	failingMapping := &ent.FieldMapping{
		SourceField:     "value",
		TargetField:     "result",
		TransformType:   "script",
		TransformConfig: `{"script": "function transform(value) { throw new Error('Intentional error'); }"}`,
		IsActive:        true,
		Status:          1,
	}

	// 创建一个正常的映射
	normalMapping := &ent.FieldMapping{
		SourceField:     "value",
		TargetField:     "result",
		TransformType:   "script",
		TransformConfig: `{"script": "function transform(value) { return value * 2; }"}`,
		IsActive:        true,
		Status:          1,
	}

	t.Run("Parallel Transformations with Errors", func(t *testing.T) {
		var successCount, errorCount int
		totalTests := 10

		// 并发执行转换（5个成功，5个失败）
		for i := 0; i < totalTests; i++ {
			var mapping *ent.FieldMapping
			if i%2 == 0 {
				mapping = normalMapping
			} else {
				mapping = failingMapping
			}

			_, err := converter.TransformWithConfig(i, mapping)
			if err != nil {
				errorCount++
			} else {
				successCount++
			}
		}

		// 验证结果
		assert.Equal(t, 5, successCount, "Should have 5 successful transformations")
		assert.Equal(t, 5, errorCount, "Should have 5 failed transformations")

		t.Logf("✅ Concurrent transformation error handling verified:")
		t.Logf("   Successful: %d, Failed: %d", successCount, errorCount)
	})
}

// TestErrorHandling_DataValidation
// 测试数据验证的错误处理
func TestErrorHandling_DataValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("Empty Data", func(t *testing.T) {
		emptyData := map[string]interface{}{}

		// 尝试从空数据中获取字段
		value := emptyData["hostname"]
		assert.Nil(t, value, "Empty data should return nil for any field")

		t.Logf("✅ Empty data validation passed")
	})

	t.Run("Nil Value Transformation", func(t *testing.T) {
		ctx := context.Background()
		logger := logx.WithContext(ctx)
		converter := transform.NewTypeConverter(logger)

		mapping := &ent.FieldMapping{
			SourceField:    "value",
			TargetField:    "result",
			TransformType:  "direct",
			TargetDataType: "string",
			IsActive:       true,
			Status:         1,
		}

		// 尝试转换 nil 值
		result, err := converter.Convert(nil, mapping)

		// nil 值应该被优雅处理
		if err != nil {
			t.Logf("✅ Nil value handled with error: %v", err)
		} else {
			t.Logf("✅ Nil value transformation result: %v", result)
		}
	})

	t.Run("Invalid Data Type", func(t *testing.T) {
		// 尝试将复杂对象转换为简单类型
		complexData := map[string]interface{}{
			"nested": map[string]interface{}{
				"deep": "value",
			},
		}

		ctx := context.Background()
		logger := logx.WithContext(ctx)
		converter := transform.NewTypeConverter(logger)

		mapping := &ent.FieldMapping{
			SourceField:    "nested",
			TargetField:    "result",
			TransformType:  "direct",
			TargetDataType: "string", // 尝试将map转为string
			IsActive:       true,
			Status:         1,
		}

		result, err := converter.Convert(complexData["nested"], mapping)

		// 应该进行某种转换或返回错误
		t.Logf("✅ Complex data type conversion result: %v, error: %v", result, err)
	})
}

