package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

// ============ Registry Tests ============

func TestProviderRegistry_GetRegistry(t *testing.T) {
	registry := GetRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.providers)

	// 验证单例模式
	registry2 := GetRegistry()
	assert.Equal(t, registry, registry2, "Registry should be singleton")
}

func TestProviderRegistry_BuiltinProviders(t *testing.T) {
	registry := GetRegistry()

	// 验证内置Providers已注册
	expectedProviders := []string{
		"file_import",
		"nb_agent",
		"aliyun_ecs",
		"vmware_vcenter",
	}

	for _, providerID := range expectedProviders {
		assert.True(t, registry.Exists(providerID), "Provider %s should be registered", providerID)

		provider, err := registry.Get(providerID)
		assert.NoError(t, err)
		assert.NotNil(t, provider)

		metadata := provider.GetMetadata()
		assert.Equal(t, providerID, metadata.ID)
		t.Logf("✅ Provider %s: %s", metadata.ID, metadata.Name)
	}
}

func TestProviderRegistry_List(t *testing.T) {
	registry := GetRegistry()
	providers := registry.List()

	assert.GreaterOrEqual(t, len(providers), 4, "Should have at least 4 built-in providers")

	for _, provider := range providers {
		metadata := provider.GetMetadata()
		assert.NotEmpty(t, metadata.ID)
		assert.NotEmpty(t, metadata.Name)
		assert.NotEmpty(t, metadata.Version)
	}
}

func TestProviderRegistry_GetNonExistent(t *testing.T) {
	registry := GetRegistry()

	provider, err := registry.Get("non_existent_provider")
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "provider not found")
}

func TestProviderRegistry_Exists(t *testing.T) {
	registry := GetRegistry()

	assert.True(t, registry.Exists("file_import"))
	assert.False(t, registry.Exists("non_existent"))
}

// ============ FileImportProvider Tests ============

func TestFileImportProvider_GetMetadata(t *testing.T) {
	provider := NewFileImportProvider()
	metadata := provider.GetMetadata()

	assert.Equal(t, "file_import", metadata.ID)
	assert.Equal(t, "文件导入", metadata.Name)
	assert.Equal(t, "file", metadata.Type)
	assert.Contains(t, metadata.SupportedModes, "excel")
	assert.Contains(t, metadata.SupportedModes, "csv")
	assert.Contains(t, metadata.SupportedModes, "json")
	assert.NotEmpty(t, metadata.Version)
}

func TestFileImportProvider_GetParameterSchema(t *testing.T) {
	provider := NewFileImportProvider()
	params := provider.GetParameterSchema()

	assert.NotEmpty(t, params)

	// 验证必需参数
	foundFilePath := false
	for _, param := range params {
		if param.Name == "file_path" {
			foundFilePath = true
			assert.True(t, param.Required, "file_path should be required")
			assert.Equal(t, "string", param.Type)
		}
	}
	assert.True(t, foundFilePath, "file_path parameter should exist")
}

func TestFileImportProvider_GetFieldSchema(t *testing.T) {
	provider := NewFileImportProvider()
	fields := provider.GetFieldSchema()

	assert.NotEmpty(t, fields)

	// 验证必需字段
	fieldNames := make([]string, 0, len(fields))
	for _, field := range fields {
		fieldNames = append(fieldNames, field.Name)
	}

	assert.Contains(t, fieldNames, "data")
	assert.Contains(t, fieldNames, "source_file")
}

func TestFileImportProvider_ValidateConfig_Success(t *testing.T) {
	provider := NewFileImportProvider()

	// 创建临时测试文件
	tmpFile, err := os.CreateTemp("", "test_*.xlsx")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	config := map[string]interface{}{
		"file_path": tmpFile.Name(),
	}

	err = provider.ValidateConfig(config)
	assert.NoError(t, err)
}

func TestFileImportProvider_ValidateConfig_MissingFilePath(t *testing.T) {
	provider := NewFileImportProvider()

	config := map[string]interface{}{}

	err := provider.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file_path is required")
}

func TestFileImportProvider_ValidateConfig_FileNotExists(t *testing.T) {
	provider := NewFileImportProvider()

	config := map[string]interface{}{
		"file_path": "/non/existent/file.xlsx",
	}

	err := provider.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file does not exist")
}

func TestFileImportProvider_TestConnection_Success(t *testing.T) {
	provider := NewFileImportProvider()

	// 创建临时Excel文件
	tmpFile, err := os.CreateTemp("", "test_*.xlsx")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// 创建一个简单的Excel文件
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "Name")
	f.SetCellValue("Sheet1", "B1", "Age")
	f.SetCellValue("Sheet1", "A2", "Alice")
	f.SetCellValue("Sheet1", "B2", 25)
	err = f.SaveAs(tmpFile.Name())
	require.NoError(t, err)

	config := map[string]interface{}{
		"file_path": tmpFile.Name(),
	}

	result, err := provider.TestConnection(config)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "文件访问成功", result.Message)
	assert.NotNil(t, result.Details)
	assert.Equal(t, "excel", result.Details["file_type"])
}

func TestFileImportProvider_TestConnection_FileNotExists(t *testing.T) {
	provider := NewFileImportProvider()

	config := map[string]interface{}{
		"file_path": "/non/existent/file.xlsx",
	}

	result, err := provider.TestConnection(config)
	require.NoError(t, err) // TestConnection returns error in result, not as error
	assert.False(t, result.Success)
	assert.Contains(t, result.Message, "无法访问文件")
}

func TestFileImportProvider_Discover_Excel(t *testing.T) {
	provider := NewFileImportProvider()

	// 创建临时Excel文件
	tmpFile, err := os.CreateTemp("", "test_*.xlsx")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// 创建测试数据
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "Name")
	f.SetCellValue("Sheet1", "B1", "Age")
	f.SetCellValue("Sheet1", "C1", "City")
	f.SetCellValue("Sheet1", "A2", "Alice")
	f.SetCellValue("Sheet1", "B2", 25)
	f.SetCellValue("Sheet1", "C2", "Beijing")
	f.SetCellValue("Sheet1", "A3", "Bob")
	f.SetCellValue("Sheet1", "B3", 30)
	f.SetCellValue("Sheet1", "C3", "Shanghai")
	err = f.SaveAs(tmpFile.Name())
	require.NoError(t, err)

	config := map[string]interface{}{
		"file_path": tmpFile.Name(),
	}

	result, err := provider.Discover(config)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, int64(2), result.TotalRecords)
	assert.Len(t, result.Records, 2)

	// 验证第一条记录
	record1 := result.Records[0]
	assert.Equal(t, 2, record1["row_number"])
	assert.Equal(t, tmpFile.Name(), record1["source_file"])
	assert.Equal(t, "excel", record1["file_type"])

	data1, ok := record1["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Alice", data1["Name"])
	assert.Equal(t, "25", data1["Age"]) // Excel cells are read as strings
	assert.Equal(t, "Beijing", data1["City"])

	t.Logf("✅ Excel parsing: %d records discovered", len(result.Records))
}

func TestFileImportProvider_Discover_CSV(t *testing.T) {
	provider := NewFileImportProvider()

	// 创建临时CSV文件
	tmpFile, err := os.CreateTemp("", "test_*.csv")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	csvContent := `Name,Age,City
Alice,25,Beijing
Bob,30,Shanghai
Charlie,28,Guangzhou`

	err = os.WriteFile(tmpFile.Name(), []byte(csvContent), 0644)
	require.NoError(t, err)

	config := map[string]interface{}{
		"file_path": tmpFile.Name(),
	}

	result, err := provider.Discover(config)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, int64(3), result.TotalRecords)
	assert.Len(t, result.Records, 3)

	// 验证第一条记录
	record1 := result.Records[0]
	data1, ok := record1["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Alice", data1["Name"])
	assert.Equal(t, "25", data1["Age"])
	assert.Equal(t, "Beijing", data1["City"])

	t.Logf("✅ CSV parsing: %d records discovered", len(result.Records))
}

func TestFileImportProvider_Discover_JSON_Array(t *testing.T) {
	provider := NewFileImportProvider()

	// 创建临时JSON文件
	tmpFile, err := os.CreateTemp("", "test_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	jsonData := []map[string]interface{}{
		{"name": "Alice", "age": 25, "city": "Beijing"},
		{"name": "Bob", "age": 30, "city": "Shanghai"},
	}

	jsonBytes, err := json.Marshal(jsonData)
	require.NoError(t, err)

	err = os.WriteFile(tmpFile.Name(), jsonBytes, 0644)
	require.NoError(t, err)

	config := map[string]interface{}{
		"file_path": tmpFile.Name(),
	}

	result, err := provider.Discover(config)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, int64(2), result.TotalRecords)
	assert.Len(t, result.Records, 2)

	t.Logf("✅ JSON array parsing: %d records discovered", len(result.Records))
}

func TestFileImportProvider_Discover_JSON_Object(t *testing.T) {
	provider := NewFileImportProvider()

	// 创建临时JSON文件
	tmpFile, err := os.CreateTemp("", "test_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	jsonData := map[string]interface{}{
		"name": "Alice",
		"age":  25,
		"city": "Beijing",
	}

	jsonBytes, err := json.Marshal(jsonData)
	require.NoError(t, err)

	err = os.WriteFile(tmpFile.Name(), jsonBytes, 0644)
	require.NoError(t, err)

	config := map[string]interface{}{
		"file_path": tmpFile.Name(),
	}

	result, err := provider.Discover(config)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, int64(1), result.TotalRecords)
	assert.Len(t, result.Records, 1)

	// 验证数据
	record := result.Records[0]
	data, ok := record["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Alice", data["name"])
	assert.Equal(t, float64(25), data["age"])

	t.Logf("✅ JSON object parsing: 1 record discovered")
}

func TestFileImportProvider_GetFieldMapping(t *testing.T) {
	provider := NewFileImportProvider()

	mapping, err := provider.GetFieldMapping("default")
	require.NoError(t, err)
	assert.NotNil(t, mapping)
	assert.Equal(t, "1.0", mapping.Version)
	assert.NotEmpty(t, mapping.Mappings)
	assert.NotEmpty(t, mapping.Transforms)

	// 验证默认映射
	defaultMappings, exists := mapping.Mappings["default"]
	assert.True(t, exists)
	assert.NotEmpty(t, defaultMappings)

	t.Logf("✅ Field mapping config: %d mappings", len(defaultMappings))
}

// ============ V2 Adapter Tests ============

func TestProviderV2Adapter_WrapV1Provider(t *testing.T) {
	v1Provider := NewFileImportProvider()
	v2Provider := NewProviderV2Adapter(v1Provider)

	assert.NotNil(t, v2Provider)

	// 验证元数据方法转发
	metadata := v2Provider.GetMetadata()
	assert.Equal(t, "file_import", metadata.ID)
}

func TestProviderV2Adapter_ValidateConfigWithContext(t *testing.T) {
	v1Provider := NewFileImportProvider()
	v2Provider := NewProviderV2Adapter(v1Provider)

	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "test_*.xlsx")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	config := map[string]interface{}{
		"file_path": tmpFile.Name(),
	}

	ctx := context.Background()
	err = v2Provider.ValidateConfigWithContext(ctx, config)
	assert.NoError(t, err)
}

func TestProviderV2Adapter_ValidateConfigWithContext_Cancelled(t *testing.T) {
	v1Provider := NewFileImportProvider()
	v2Provider := NewProviderV2Adapter(v1Provider)

	config := map[string]interface{}{
		"file_path": "/tmp/test.xlsx",
	}

	// 创建已取消的context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	err := v2Provider.ValidateConfigWithContext(ctx, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")
}

func TestProviderV2Adapter_TestConnectionWithContext(t *testing.T) {
	v1Provider := NewFileImportProvider()
	v2Provider := NewProviderV2Adapter(v1Provider)

	// 创建临时Excel文件
	tmpFile, err := os.CreateTemp("", "test_*.xlsx")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "Test")
	err = f.SaveAs(tmpFile.Name())
	require.NoError(t, err)

	config := map[string]interface{}{
		"file_path": tmpFile.Name(),
	}

	ctx := context.Background()
	result, err := v2Provider.TestConnectionWithContext(ctx, config)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestProviderV2Adapter_TestConnectionWithContext_Timeout(t *testing.T) {
	// 这个测试很难模拟，因为FileImportProvider是同步的
	// 在实际的网络Provider中才能真正测试超时
	t.Skip("Timeout test requires async provider")
}

func TestProviderV2Adapter_DiscoverWithContext(t *testing.T) {
	v1Provider := NewFileImportProvider()
	v2Provider := NewProviderV2Adapter(v1Provider)

	// 创建临时CSV文件
	tmpFile, err := os.CreateTemp("", "test_*.csv")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	csvContent := `Name,Age
Alice,25
Bob,30`

	err = os.WriteFile(tmpFile.Name(), []byte(csvContent), 0644)
	require.NoError(t, err)

	config := map[string]interface{}{
		"file_path": tmpFile.Name(),
	}

	ctx := context.Background()
	result, err := v2Provider.DiscoverWithContext(ctx, config)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, int64(2), result.TotalRecords)

	t.Logf("✅ V2 Adapter DiscoverWithContext: %d records", result.TotalRecords)
}

func TestProviderV2Adapter_DiscoverWithContext_Cancelled(t *testing.T) {
	v1Provider := NewFileImportProvider()
	v2Provider := NewProviderV2Adapter(v1Provider)

	// 创建临时CSV文件
	tmpFile, err := os.CreateTemp("", "test_*.csv")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	csvContent := `Name,Age
Alice,25`

	err = os.WriteFile(tmpFile.Name(), []byte(csvContent), 0644)
	require.NoError(t, err)

	config := map[string]interface{}{
		"file_path": tmpFile.Name(),
	}

	// 创建已取消的context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	// 由于FileImportProvider是同步的，goroutine可能已经完成
	// 但我们仍然应该检查context是否被正确处理
	result, err := v2Provider.DiscoverWithContext(ctx, config)

	// 可能是立即返回context错误，也可能goroutine太快已经完成
	if err != nil {
		assert.Contains(t, err.Error(), "cancelled")
	} else {
		// 如果没有错误，说明goroutine在context取消前就完成了
		assert.True(t, result.Success)
	}
}

// ============ ToProviderV2 Helper Tests ============

func TestToProviderV2_V1Provider(t *testing.T) {
	v1Provider := NewFileImportProvider()

	v2Provider, err := ToProviderV2(v1Provider)
	require.NoError(t, err)
	assert.NotNil(t, v2Provider)

	// 验证是V2适配器
	_, ok := v2Provider.(*ProviderV2Adapter)
	assert.True(t, ok, "Should be wrapped in V2 adapter")
}

func TestToProviderV2_InvalidProvider(t *testing.T) {
	invalidProvider := "not a provider"

	v2Provider, err := ToProviderV2(invalidProvider)
	assert.Error(t, err)
	assert.Nil(t, v2Provider)
	assert.Contains(t, err.Error(), "does not implement")
}

func TestIsProviderV2_V1Provider(t *testing.T) {
	v1Provider := NewFileImportProvider()
	assert.False(t, IsProviderV2(v1Provider))
}

func TestIsProviderV2_V2Adapter(t *testing.T) {
	v1Provider := NewFileImportProvider()
	v2Provider := NewProviderV2Adapter(v1Provider)
	assert.True(t, IsProviderV2(v2Provider))
}

// ============ NBAgentProvider Basic Tests ============

func TestNBAgentProvider_GetMetadata(t *testing.T) {
	provider := NewNBAgentProvider()
	metadata := provider.GetMetadata()

	assert.Equal(t, "nb_agent", metadata.ID)
	assert.Equal(t, "NewBee Agent", metadata.Name)
	assert.Equal(t, "agent", metadata.Type)
	assert.Contains(t, metadata.SupportedModes, "agent_scan")
	assert.Contains(t, metadata.SupportedModes, "agent_network")
	assert.Contains(t, metadata.SupportedModes, "agent_service")
}

func TestNBAgentProvider_ValidateConfig_Success(t *testing.T) {
	provider := NewNBAgentProvider()

	config := map[string]interface{}{
		"agent_host":     "192.168.1.100",
		"api_key":        "nb_test_key_12345",
		"discovery_mode": "agent_scan",
	}

	err := provider.ValidateConfig(config)
	assert.NoError(t, err)
}

func TestNBAgentProvider_ValidateConfig_MissingRequired(t *testing.T) {
	provider := NewNBAgentProvider()

	tests := []struct {
		name        string
		config      map[string]interface{}
		expectedErr string
	}{
		{
			name:        "Missing agent_host",
			config:      map[string]interface{}{"api_key": "test", "discovery_mode": "agent_scan"},
			expectedErr: "agent_host is required",
		},
		{
			name:        "Missing api_key",
			config:      map[string]interface{}{"agent_host": "192.168.1.1", "discovery_mode": "agent_scan"},
			expectedErr: "api_key is required",
		},
		{
			name:        "Missing discovery_mode",
			config:      map[string]interface{}{"agent_host": "192.168.1.1", "api_key": "test"},
			expectedErr: "discovery_mode is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.ValidateConfig(tt.config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// ============ AliyunECSProvider Basic Tests ============

func TestAliyunECSProvider_GetMetadata(t *testing.T) {
	provider := NewAliyunECSProvider()
	metadata := provider.GetMetadata()

	assert.Equal(t, "aliyun_ecs", metadata.ID)
	assert.Equal(t, "阿里云ECS", metadata.Name)
	assert.Equal(t, "cloud", metadata.Type)
	assert.Contains(t, metadata.SupportedModes, "ecs_instances")
}

func TestAliyunECSProvider_ValidateConfig_Success(t *testing.T) {
	provider := NewAliyunECSProvider()

	config := map[string]interface{}{
		"access_key_id":     "LTAI4G8G8G8G8G8G8G8G",
		"access_key_secret": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
	}

	err := provider.ValidateConfig(config)
	assert.NoError(t, err)
}

func TestAliyunECSProvider_ValidateConfig_MissingRequired(t *testing.T) {
	provider := NewAliyunECSProvider()

	tests := []struct {
		name        string
		config      map[string]interface{}
		expectedErr string
	}{
		{
			name:        "Missing access_key_id",
			config:      map[string]interface{}{"access_key_secret": "secret"},
			expectedErr: "access_key_id is required",
		},
		{
			name:        "Missing access_key_secret",
			config:      map[string]interface{}{"access_key_id": "id"},
			expectedErr: "access_key_secret is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.ValidateConfig(tt.config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// ============ VMwareVCenterProvider Basic Tests ============

func TestVMwareVCenterProvider_GetMetadata(t *testing.T) {
	provider := NewVMwareVCenterProvider()
	metadata := provider.GetMetadata()

	assert.Equal(t, "vmware_vcenter", metadata.ID)
	assert.Equal(t, "VMware vCenter", metadata.Name)
	assert.Equal(t, "virtualization", metadata.Type)
	assert.Contains(t, metadata.SupportedModes, "virtual_machines")
	assert.Contains(t, metadata.SupportedModes, "esxi_hosts")
}

func TestVMwareVCenterProvider_ValidateConfig_Success(t *testing.T) {
	provider := NewVMwareVCenterProvider()

	config := map[string]interface{}{
		"vcenter_host":   "vcenter.example.com",
		"username":       "administrator@vsphere.local",
		"password":       "password123",
		"discovery_mode": "virtual_machines",
	}

	err := provider.ValidateConfig(config)
	assert.NoError(t, err)
}

func TestVMwareVCenterProvider_ValidateConfig_MissingRequired(t *testing.T) {
	provider := NewVMwareVCenterProvider()

	tests := []struct {
		name        string
		config      map[string]interface{}
		expectedErr string
	}{
		{
			name:        "Missing vcenter_host",
			config:      map[string]interface{}{"username": "user", "password": "pass", "discovery_mode": "virtual_machines"},
			expectedErr: "vcenter_host is required",
		},
		{
			name:        "Missing username",
			config:      map[string]interface{}{"vcenter_host": "host", "password": "pass", "discovery_mode": "virtual_machines"},
			expectedErr: "username is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.ValidateConfig(tt.config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// ============ Performance Benchmarks ============

func BenchmarkFileImportProvider_Discover_CSV(b *testing.B) {
	provider := NewFileImportProvider()

	// 创建测试CSV文件
	tmpFile, err := os.CreateTemp("", "bench_*.csv")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// 生成100行测试数据
	csvContent := "Name,Age,City\n"
	for i := 0; i < 100; i++ {
		csvContent += fmt.Sprintf("User%d,%d,City%d\n", i, 20+i%50, i%10)
	}

	if err := os.WriteFile(tmpFile.Name(), []byte(csvContent), 0644); err != nil {
		b.Fatal(err)
	}

	config := map[string]interface{}{
		"file_path": tmpFile.Name(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.Discover(config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProviderV2Adapter_DiscoverWithContext(b *testing.B) {
	v1Provider := NewFileImportProvider()
	v2Provider := NewProviderV2Adapter(v1Provider)

	// 创建测试CSV文件
	tmpFile, err := os.CreateTemp("", "bench_*.csv")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	csvContent := "Name,Age\nAlice,25\nBob,30\n"
	if err := os.WriteFile(tmpFile.Name(), []byte(csvContent), 0644); err != nil {
		b.Fatal(err)
	}

	config := map[string]interface{}{
		"file_path": tmpFile.Name(),
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := v2Provider.DiscoverWithContext(ctx, config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ============ Integration Scenarios ============

func TestProviderRegistry_CompleteWorkflow(t *testing.T) {
	// 模拟完整工作流: 获取Registry -> 列出Providers -> 选择Provider -> 发现数据

	// 1. 获取Registry
	registry := GetRegistry()
	assert.NotNil(t, registry)

	// 2. 列出所有Providers
	providers := registry.List()
	assert.GreaterOrEqual(t, len(providers), 4)

	// 3. 选择FileImportProvider
	fileProvider, err := registry.Get("file_import")
	require.NoError(t, err)

	// 4. 获取参数Schema
	paramSchema := fileProvider.GetParameterSchema()
	assert.NotEmpty(t, paramSchema)

	// 5. 准备配置
	tmpFile, err := os.CreateTemp("", "test_*.csv")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	csvContent := "Name,Value\nTest,123\n"
	err = os.WriteFile(tmpFile.Name(), []byte(csvContent), 0644)
	require.NoError(t, err)

	config := map[string]interface{}{
		"file_path": tmpFile.Name(),
	}

	// 6. 验证配置
	err = fileProvider.ValidateConfig(config)
	require.NoError(t, err)

	// 7. 测试连接
	testResult, err := fileProvider.TestConnection(config)
	require.NoError(t, err)
	assert.True(t, testResult.Success)

	// 8. 执行发现
	discoverResult, err := fileProvider.Discover(config)
	require.NoError(t, err)
	assert.True(t, discoverResult.Success)
	assert.Greater(t, discoverResult.TotalRecords, int64(0))

	t.Logf("✅ Complete workflow test passed: discovered %d records", discoverResult.TotalRecords)
}

func TestProviderV2_WorkflowWithContext(t *testing.T) {
	// 测试带Context的完整工作流

	// 1. 获取Provider并转换为V2
	v1Provider := NewFileImportProvider()
	v2Provider, err := ToProviderV2(v1Provider)
	require.NoError(t, err)

	// 2. 准备配置和Context
	tmpFile, err := os.CreateTemp("", "test_*.csv")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	csvContent := "ID,Name\n1,Alice\n2,Bob\n"
	err = os.WriteFile(tmpFile.Name(), []byte(csvContent), 0644)
	require.NoError(t, err)

	config := map[string]interface{}{
		"file_path": tmpFile.Name(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 3. 验证配置(带Context)
	err = v2Provider.ValidateConfigWithContext(ctx, config)
	require.NoError(t, err)

	// 4. 测试连接(带Context)
	testResult, err := v2Provider.TestConnectionWithContext(ctx, config)
	require.NoError(t, err)
	assert.True(t, testResult.Success)

	// 5. 执行发现(带Context)
	discoverResult, err := v2Provider.DiscoverWithContext(ctx, config)
	require.NoError(t, err)
	assert.True(t, discoverResult.Success)
	assert.Equal(t, int64(2), discoverResult.TotalRecords)

	t.Logf("✅ V2 workflow with context: discovered %d records", discoverResult.TotalRecords)
}
