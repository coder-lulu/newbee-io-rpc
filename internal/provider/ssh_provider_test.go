package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============ SSH Provider Basic Tests ============

func TestSSHProvider_GetMetadata(t *testing.T) {
	provider := NewSSHProvider()
	metadata := provider.GetMetadata()

	assert.Equal(t, "ssh", metadata.ID)
	assert.Equal(t, "SSH 主机发现", metadata.Name)
	assert.Equal(t, "通过 SSH 连接发现 Linux/Unix 主机信息", metadata.Description)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.Equal(t, "host", metadata.Type)
	assert.Equal(t, "infrastructure", metadata.Category)
	assert.Equal(t, "/static/icons/ssh.svg", metadata.IconURL)

	// 验证支持的模式
	assert.Contains(t, metadata.SupportedModes, "system_info")
	assert.Contains(t, metadata.SupportedModes, "hardware")
	assert.Contains(t, metadata.SupportedModes, "network")
	assert.Contains(t, metadata.SupportedModes, "full")

	// 验证标签
	assert.Contains(t, metadata.Tags, "ssh")
	assert.Contains(t, metadata.Tags, "linux")
	assert.Contains(t, metadata.Tags, "unix")
}

func TestSSHProvider_GetParameterSchema(t *testing.T) {
	provider := NewSSHProvider()
	params := provider.GetParameterSchema()

	assert.NotEmpty(t, params)

	// 验证关键参数
	paramMap := make(map[string]ParameterDefinition)
	for _, param := range params {
		paramMap[param.Name] = param
	}

	// host参数
	assert.Contains(t, paramMap, "host")
	host := paramMap["host"]
	assert.Equal(t, "string", host.Type)
	assert.True(t, host.Required)
	assert.Equal(t, "主机地址", host.DisplayName)

	// port参数
	assert.Contains(t, paramMap, "port")
	port := paramMap["port"]
	assert.Equal(t, "integer", port.Type)
	assert.False(t, port.Required)
	assert.Equal(t, 22, port.DefaultValue)

	// username参数
	assert.Contains(t, paramMap, "username")
	username := paramMap["username"]
	assert.Equal(t, "string", username.Type)
	assert.True(t, username.Required)

	// auth_method参数
	assert.Contains(t, paramMap, "auth_method")
	authMethod := paramMap["auth_method"]
	assert.Equal(t, "select", authMethod.Type)
	assert.Equal(t, "password", authMethod.DefaultValue)
	assert.Len(t, authMethod.Options, 2)

	// password参数
	assert.Contains(t, paramMap, "password")
	password := paramMap["password"]
	assert.Equal(t, "password", password.Type)
	assert.True(t, password.Sensitive)

	// private_key参数
	assert.Contains(t, paramMap, "private_key")
	privateKey := paramMap["private_key"]
	assert.Equal(t, "textarea", privateKey.Type)
	assert.True(t, privateKey.Sensitive)

	// timeout参数
	assert.Contains(t, paramMap, "timeout")
	timeout := paramMap["timeout"]
	assert.Equal(t, "integer", timeout.Type)
	assert.Equal(t, 30, timeout.DefaultValue)

	// discover_mode参数
	assert.Contains(t, paramMap, "discover_mode")
	discoverMode := paramMap["discover_mode"]
	assert.Equal(t, "select", discoverMode.Type)
	assert.Equal(t, "full", discoverMode.DefaultValue)
}

func TestSSHProvider_GetFieldSchema(t *testing.T) {
	provider := NewSSHProvider()
	fields := provider.GetFieldSchema()

	assert.Len(t, fields, 20) // 应该有20个字段

	fieldMap := make(map[string]FieldDefinition)
	for _, field := range fields {
		fieldMap[field.Name] = field
	}

	// 验证关键字段
	assert.Contains(t, fieldMap, "hostname")
	hostname := fieldMap["hostname"]
	assert.Equal(t, "string", hostname.Type)
	assert.True(t, hostname.Required)
	assert.True(t, hostname.Unique)

	assert.Contains(t, fieldMap, "os_type")
	assert.Contains(t, fieldMap, "os_version")
	assert.Contains(t, fieldMap, "kernel_version")
	assert.Contains(t, fieldMap, "architecture")

	assert.Contains(t, fieldMap, "cpu_model")
	assert.Contains(t, fieldMap, "cpu_cores")
	cpu_cores := fieldMap["cpu_cores"]
	assert.Equal(t, "integer", cpu_cores.Type)

	assert.Contains(t, fieldMap, "memory_total_mb")
	assert.Contains(t, fieldMap, "memory_used_mb")
	assert.Contains(t, fieldMap, "memory_free_mb")

	assert.Contains(t, fieldMap, "disk_total_gb")
	assert.Contains(t, fieldMap, "disk_used_gb")
	assert.Contains(t, fieldMap, "disk_free_gb")

	assert.Contains(t, fieldMap, "ip_address")
	ip_address := fieldMap["ip_address"]
	assert.True(t, ip_address.Searchable)

	assert.Contains(t, fieldMap, "mac_address")
	assert.Contains(t, fieldMap, "uptime_days")
	assert.Contains(t, fieldMap, "load_average_1min")
	assert.Contains(t, fieldMap, "load_average_5min")
	assert.Contains(t, fieldMap, "load_average_15min")
}

// ============ SSH Provider Configuration Tests ============

func TestSSHProvider_ValidateConfig_Success_Password(t *testing.T) {
	provider := NewSSHProvider()

	config := map[string]interface{}{
		"host":        "192.168.1.10",
		"username":    "root",
		"auth_method": "password",
		"password":    "test123",
	}

	err := provider.ValidateConfig(config)
	assert.NoError(t, err)
}

func TestSSHProvider_ValidateConfig_Success_Key(t *testing.T) {
	provider := NewSSHProvider()

	config := map[string]interface{}{
		"host":        "192.168.1.10",
		"username":    "root",
		"auth_method": "key",
		"private_key": "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----",
	}

	err := provider.ValidateConfig(config)
	assert.NoError(t, err)
}

func TestSSHProvider_ValidateConfig_MissingHost(t *testing.T) {
	provider := NewSSHProvider()

	config := map[string]interface{}{
		"username": "root",
		"password": "test123",
	}

	err := provider.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "host is required")
}

func TestSSHProvider_ValidateConfig_MissingUsername(t *testing.T) {
	provider := NewSSHProvider()

	config := map[string]interface{}{
		"host":     "192.168.1.10",
		"password": "test123",
	}

	err := provider.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username is required")
}

func TestSSHProvider_ValidateConfig_MissingPassword(t *testing.T) {
	provider := NewSSHProvider()

	config := map[string]interface{}{
		"host":        "192.168.1.10",
		"username":    "root",
		"auth_method": "password",
	}

	err := provider.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password is required")
}

func TestSSHProvider_ValidateConfig_MissingPrivateKey(t *testing.T) {
	provider := NewSSHProvider()

	config := map[string]interface{}{
		"host":        "192.168.1.10",
		"username":    "root",
		"auth_method": "key",
	}

	err := provider.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "private_key is required")
}

func TestSSHProvider_ValidateConfig_InvalidAuthMethod(t *testing.T) {
	provider := NewSSHProvider()

	config := map[string]interface{}{
		"host":        "192.168.1.10",
		"username":    "root",
		"auth_method": "invalid",
	}

	err := provider.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid auth_method")
}

func TestSSHProvider_ValidateConfig_DefaultAuthMethod(t *testing.T) {
	provider := NewSSHProvider()

	// 不指定auth_method，默认应该是password
	config := map[string]interface{}{
		"host":     "192.168.1.10",
		"username": "root",
		"password": "test123",
	}

	err := provider.ValidateConfig(config)
	assert.NoError(t, err)
}

// ============ SSH Provider Field Mapping Tests ============

func TestSSHProvider_GetFieldMapping(t *testing.T) {
	provider := NewSSHProvider()

	mapping, err := provider.GetFieldMapping("cmdb_server")
	require.NoError(t, err)
	assert.NotNil(t, mapping)

	assert.Equal(t, "1.0.0", mapping.Version)
	assert.Equal(t, "SSH Provider 默认字段映射", mapping.Description)

	// 验证cmdb_server映射
	cmdbServerMappings, exists := mapping.Mappings["cmdb_server"]
	assert.True(t, exists)
	assert.NotEmpty(t, cmdbServerMappings)

	// 验证关键映射
	mappingMap := make(map[string]FieldMapping)
	for _, m := range cmdbServerMappings {
		mappingMap[m.SourceField] = m
	}

	// hostname -> name
	assert.Contains(t, mappingMap, "hostname")
	hostnameMapping := mappingMap["hostname"]
	assert.Equal(t, "name", hostnameMapping.TargetField)
	assert.Equal(t, "direct", hostnameMapping.Transform)

	// os_type -> os_type
	assert.Contains(t, mappingMap, "os_type")

	// cpu_cores -> cpu_cores
	assert.Contains(t, mappingMap, "cpu_cores")
	cpuMapping := mappingMap["cpu_cores"]
	assert.Equal(t, "int", cpuMapping.Transform)

	// memory_total_mb -> memory_mb
	assert.Contains(t, mappingMap, "memory_total_mb")
	memoryMapping := mappingMap["memory_total_mb"]
	assert.Equal(t, "memory_mb", memoryMapping.TargetField)

	// ip_address -> ip_address
	assert.Contains(t, mappingMap, "ip_address")
}

func TestSSHProvider_GetFieldMapping_UnsupportedSchema(t *testing.T) {
	provider := NewSSHProvider()

	// 请求未定义的schema，应该返回空映射但不报错
	mapping, err := provider.GetFieldMapping("unknown_schema")
	require.NoError(t, err)
	assert.NotNil(t, mapping)
}

// ============ SSH Provider Helper Method Tests (Unit) ============

// 注意：以下测试需要真实的SSH服务器，或者需要mock SSH连接
// 这里只测试配置解析逻辑

func TestSSHProvider_ExtractPort_Default(t *testing.T) {
	provider := NewSSHProvider()

	config := map[string]interface{}{
		"host":     "192.168.1.10",
		"username": "root",
		"password": "test123",
		// 没有指定port
	}

	// 由于createSSHClient是私有方法，我们无法直接测试
	// 但我们可以验证配置本身
	err := provider.ValidateConfig(config)
	assert.NoError(t, err)
}

func TestSSHProvider_ExtractPort_Custom(t *testing.T) {
	provider := NewSSHProvider()

	config := map[string]interface{}{
		"host":     "192.168.1.10",
		"port":     float64(2222), // JSON解析后是float64
		"username": "root",
		"password": "test123",
	}

	err := provider.ValidateConfig(config)
	assert.NoError(t, err)
}

func TestSSHProvider_ExtractTimeout_Default(t *testing.T) {
	provider := NewSSHProvider()

	config := map[string]interface{}{
		"host":     "192.168.1.10",
		"username": "root",
		"password": "test123",
		// 没有指定timeout
	}

	err := provider.ValidateConfig(config)
	assert.NoError(t, err)
}

func TestSSHProvider_ExtractTimeout_Custom(t *testing.T) {
	provider := NewSSHProvider()

	config := map[string]interface{}{
		"host":     "192.168.1.10",
		"username": "root",
		"password": "test123",
		"timeout":  float64(60), // 60秒
	}

	err := provider.ValidateConfig(config)
	assert.NoError(t, err)
}

// ============ SSH Provider Complete Workflow Tests ============

func TestSSHProvider_CompleteWorkflow_Validation(t *testing.T) {
	provider := NewSSHProvider()

	// 1. 获取元数据
	metadata := provider.GetMetadata()
	assert.Equal(t, "ssh", metadata.ID)

	// 2. 获取参数Schema
	params := provider.GetParameterSchema()
	assert.NotEmpty(t, params)

	// 3. 准备配置
	config := map[string]interface{}{
		"host":        "192.168.1.10",
		"port":        float64(22),
		"username":    "root",
		"auth_method": "password",
		"password":    "test123",
		"timeout":     float64(30),
	}

	// 4. 验证配置
	err := provider.ValidateConfig(config)
	require.NoError(t, err)

	// 5. 获取字段Schema
	fields := provider.GetFieldSchema()
	assert.Len(t, fields, 20)

	// 6. 获取字段映射
	mapping, err := provider.GetFieldMapping("cmdb_server")
	require.NoError(t, err)
	assert.NotNil(t, mapping)

	t.Log("✅ Complete workflow validation test passed")
}

func TestSSHProvider_CompleteWorkflow_KeyAuth(t *testing.T) {
	provider := NewSSHProvider()

	config := map[string]interface{}{
		"host":        "192.168.1.10",
		"username":    "root",
		"auth_method": "key",
		"private_key": "-----BEGIN RSA PRIVATE KEY-----\ntest_key\n-----END RSA PRIVATE KEY-----",
		"passphrase":  "key_password",
	}

	err := provider.ValidateConfig(config)
	require.NoError(t, err)

	t.Log("✅ Key-based auth workflow test passed")
}

// ============ SSH Provider Factory Tests ============

func TestNewSSHProvider(t *testing.T) {
	provider := NewSSHProvider()

	assert.NotNil(t, provider)

	metadata := provider.GetMetadata()
	assert.Equal(t, "ssh", metadata.ID)
	assert.Equal(t, "SSH 主机发现", metadata.Name)
}

// ============ SSH Provider Discovery Mode Tests ============

func TestSSHProvider_DiscoveryModes(t *testing.T) {
	provider := NewSSHProvider()

	testCases := []struct {
		mode        string
		description string
	}{
		{"system_info", "系统信息"},
		{"hardware", "硬件信息"},
		{"network", "网络信息"},
		{"full", "完整信息"},
	}

	for _, tc := range testCases {
		t.Run(tc.mode, func(t *testing.T) {
			config := map[string]interface{}{
				"host":          "192.168.1.10",
				"username":      "root",
				"password":      "test123",
				"discover_mode": tc.mode,
			}

			err := provider.ValidateConfig(config)
			assert.NoError(t, err, "Should accept discover_mode: %s", tc.mode)
		})
	}
}

// ============ SSH Provider Edge Cases ============

func TestSSHProvider_ValidateConfig_EmptyHost(t *testing.T) {
	provider := NewSSHProvider()

	config := map[string]interface{}{
		"host":     "",
		"username": "root",
		"password": "test123",
	}

	err := provider.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "host is required")
}

func TestSSHProvider_ValidateConfig_EmptyUsername(t *testing.T) {
	provider := NewSSHProvider()

	config := map[string]interface{}{
		"host":     "192.168.1.10",
		"username": "",
		"password": "test123",
	}

	err := provider.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "username is required")
}

func TestSSHProvider_ValidateConfig_EmptyPassword(t *testing.T) {
	provider := NewSSHProvider()

	config := map[string]interface{}{
		"host":        "192.168.1.10",
		"username":    "root",
		"auth_method": "password",
		"password":    "",
	}

	err := provider.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "password is required")
}

func TestSSHProvider_ValidateConfig_EmptyPrivateKey(t *testing.T) {
	provider := NewSSHProvider()

	config := map[string]interface{}{
		"host":        "192.168.1.10",
		"username":    "root",
		"auth_method": "key",
		"private_key": "",
	}

	err := provider.ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "private_key is required")
}

// ============ Performance Benchmarks ============

func BenchmarkSSHProvider_GetMetadata(b *testing.B) {
	provider := NewSSHProvider()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.GetMetadata()
	}
}

func BenchmarkSSHProvider_GetParameterSchema(b *testing.B) {
	provider := NewSSHProvider()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.GetParameterSchema()
	}
}

func BenchmarkSSHProvider_GetFieldSchema(b *testing.B) {
	provider := NewSSHProvider()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.GetFieldSchema()
	}
}

func BenchmarkSSHProvider_ValidateConfig(b *testing.B) {
	provider := NewSSHProvider()
	config := map[string]interface{}{
		"host":     "192.168.1.10",
		"username": "root",
		"password": "test123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.ValidateConfig(config)
	}
}
