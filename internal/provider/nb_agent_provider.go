package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type NBAgentProvider struct{}

func NewNBAgentProvider() *NBAgentProvider {
	return &NBAgentProvider{}
}

func (p *NBAgentProvider) GetMetadata() *ProviderMetadata {
	return &ProviderMetadata{
		ID:          "nb_agent",
		Name:        "NewBee Agent",
		Description: "Distributed discovery using NewBee Agent deployed on target machines",
		Version:     "1.0.0",
		Type:        "agent",
		IconURL:     "/static/icons/nb_agent.svg",
		SupportedModes: []string{
			"agent_scan",     // 通过Agent扫描本机
			"agent_network",  // 通过Agent扫描网络
			"agent_service",  // 通过Agent扫描服务
		},
		Tags: []string{"agent", "distributed", "real-time", "cross-platform"},
	}
}

func (p *NBAgentProvider) GetParameterSchema() []ParameterDefinition {
	return []ParameterDefinition{
		{
			Name:        "agent_host",
			DisplayName: "Agent主机地址",
			Type:        "string",
			Required:    true,
			Description: "NewBee Agent的主机地址或IP",
			Example:     "192.168.1.100",
			Validation: map[string]interface{}{
				"pattern": `^(\d{1,3}\.){3}\d{1,3}$|^[a-zA-Z0-9.-]+$`,
			},
		},
		{
			Name:        "agent_port",
			DisplayName: "Agent端口",
			Type:        "integer",
			Required:    false,
			Description: "NewBee Agent的服务端口",
			DefaultValue: 8080,
			Example:     "8080",
			Validation: map[string]interface{}{
				"min": 1,
				"max": 65535,
			},
		},
		{
			Name:        "api_key",
			DisplayName: "API密钥",
			Type:        "string",
			Required:    true,
			Description: "访问Agent API的认证密钥",
			Sensitive:   true,
			Example:     "nb_xxxxxxxxxxxxxxxx",
		},
		{
			Name:        "discovery_mode",
			DisplayName: "发现模式",
			Type:        "select",
			Required:    true,
			Description: "选择发现模式",
			DefaultValue: "agent_scan",
			Options: []SelectOption{
				{Value: "agent_scan", Label: "本机扫描", Description: "扫描Agent所在机器的资源"},
				{Value: "agent_network", Label: "网络扫描", Description: "通过Agent扫描网络中的设备"},
				{Value: "agent_service", Label: "服务扫描", Description: "扫描Agent机器上的服务"},
			},
		},
		{
			Name:        "scan_timeout",
			DisplayName: "扫描超时(秒)",
			Type:        "integer",
			Required:    false,
			Description: "单次扫描的超时时间",
			DefaultValue: 300,
			Example:     "300",
			Validation: map[string]interface{}{
				"min": 30,
				"max": 3600,
			},
		},
		{
			Name:        "network_range",
			DisplayName: "网络范围",
			Type:        "string",
			Required:    false,
			Description: "网络扫描时的IP范围 (仅在network模式下使用)",
			Example:     "192.168.1.0/24",
			Validation: map[string]interface{}{
				"pattern": `^(\d{1,3}\.){3}\d{1,3}\/\d{1,2}$`,
			},
		},
		{
			Name:        "include_services",
			DisplayName: "包含服务类型",
			Type:        "multiselect",
			Required:    false,
			Description: "选择要扫描的服务类型",
			Options: []SelectOption{
				{Value: "web", Label: "Web服务", Description: "HTTP/HTTPS服务"},
				{Value: "database", Label: "数据库", Description: "MySQL, PostgreSQL, Redis等"},
				{Value: "system", Label: "系统服务", Description: "SSH, FTP, DNS等"},
				{Value: "custom", Label: "自定义端口", Description: "指定端口范围"},
			},
		},
		{
			Name:        "custom_ports",
			DisplayName: "自定义端口",
			Type:        "string",
			Required:    false,
			Description: "自定义端口范围，如: 80,443,8080-8090",
			Example:     "80,443,8080-8090",
		},
	}
}

func (p *NBAgentProvider) GetFieldSchema() []FieldDefinition {
	return []FieldDefinition{
		{
			Name:        "hostname",
			DisplayName: "主机名",
			Type:        "string",
			Description: "目标机器的主机名",
			Required:    true,
			Searchable:  true,
		},
		{
			Name:        "ip_address",
			DisplayName: "IP地址",
			Type:        "string",
			Description: "目标机器的IP地址",
			Required:    true,
			Unique:      true,
			Searchable:  true,
		},
		{
			Name:        "mac_address",
			DisplayName: "MAC地址",
			Type:        "string",
			Description: "网卡MAC地址",
			Required:    false,
			Unique:      true,
		},
		{
			Name:        "os_type",
			DisplayName: "操作系统类型",
			Type:        "string",
			Description: "操作系统类型",
			Required:    false,
			Searchable:  true,
		},
		{
			Name:        "os_version",
			DisplayName: "操作系统版本",
			Type:        "string",
			Description: "操作系统版本信息",
			Required:    false,
		},
		{
			Name:        "cpu_cores",
			DisplayName: "CPU核心数",
			Type:        "integer",
			Description: "CPU核心数量",
			Required:    false,
		},
		{
			Name:        "memory_mb",
			DisplayName: "内存(MB)",
			Type:        "integer",
			Description: "内存大小(MB)",
			Required:    false,
		},
		{
			Name:        "disk_gb",
			DisplayName: "磁盘(GB)",
			Type:        "integer",
			Description: "磁盘总容量(GB)",
			Required:    false,
		},
		{
			Name:        "services",
			DisplayName: "运行服务",
			Type:        "json",
			Description: "机器上运行的服务列表",
			Required:    false,
		},
		{
			Name:        "open_ports",
			DisplayName: "开放端口",
			Type:        "json",
			Description: "开放端口列表",
			Required:    false,
		},
		{
			Name:        "agent_version",
			DisplayName: "Agent版本",
			Type:        "string",
			Description: "NewBee Agent版本",
			Required:    false,
		},
		{
			Name:        "last_online",
			DisplayName: "最后在线时间",
			Type:        "datetime",
			Description: "Agent最后在线时间",
			Required:    false,
		},
		{
			Name:        "status",
			DisplayName: "状态",
			Type:        "string",
			Description: "设备状态: online, offline, unknown",
			Required:    false,
			Searchable:  true,
		},
	}
}

func (p *NBAgentProvider) ValidateConfig(config map[string]interface{}) error {
	// 验证必需参数
	agentHost, ok := config["agent_host"].(string)
	if !ok || agentHost == "" {
		return fmt.Errorf("agent_host is required")
	}

	apiKey, ok := config["api_key"].(string)
	if !ok || apiKey == "" {
		return fmt.Errorf("api_key is required")
	}

	discoveryMode, ok := config["discovery_mode"].(string)
	if !ok || discoveryMode == "" {
		return fmt.Errorf("discovery_mode is required")
	}

	// 验证发现模式
	validModes := map[string]bool{
		"agent_scan":    true,
		"agent_network": true,
		"agent_service": true,
	}
	if !validModes[discoveryMode] {
		return fmt.Errorf("invalid discovery_mode: %s", discoveryMode)
	}

	// 验证端口
	if port, exists := config["agent_port"]; exists {
		if portInt, ok := port.(float64); ok {
			if portInt < 1 || portInt > 65535 {
				return fmt.Errorf("agent_port must be between 1 and 65535")
			}
		}
	}

	// 网络模式下验证网络范围
	if discoveryMode == "agent_network" {
		if networkRange, exists := config["network_range"]; exists {
			if networkStr, ok := networkRange.(string); ok && networkStr != "" {
				if !strings.Contains(networkStr, "/") {
					return fmt.Errorf("network_range must be in CIDR format (e.g., 192.168.1.0/24)")
				}
			}
		}
	}

	return nil
}

func (p *NBAgentProvider) TestConnection(config map[string]interface{}) (*TestResult, error) {
	agentHost := config["agent_host"].(string)
	apiKey := config["api_key"].(string)
	
	port := 8080 // 默认端口
	if portVal, exists := config["agent_port"]; exists {
		if portFloat, ok := portVal.(float64); ok {
			port = int(portFloat)
		}
	}

	// 构建请求URL
	url := fmt.Sprintf("http://%s:%d/api/v1/health", agentHost, port)
	
	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("创建请求失败: %v", err),
		}, nil
	}

	// 添加认证头
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("连接Agent失败: %v", err),
			Details: map[string]interface{}{
				"url":   url,
				"error": err.Error(),
			},
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("Agent返回错误状态: %d", resp.StatusCode),
			Details: map[string]interface{}{
				"status_code": resp.StatusCode,
				"url":         url,
			},
		}, nil
	}

	// 解析健康检查响应
	var healthResp struct {
		Status  string `json:"status"`
		Version string `json:"version"`
		Time    string `json:"time"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("解析Agent响应失败: %v", err),
		}, nil
	}

	return &TestResult{
		Success: true,
		Message: "连接Agent成功",
		Details: map[string]interface{}{
			"agent_version": healthResp.Version,
			"agent_status":  healthResp.Status,
			"response_time": healthResp.Time,
			"url":           url,
		},
	}, nil
}

func (p *NBAgentProvider) Discover(config map[string]interface{}) (*DiscoveryResult, error) {
	agentHost := config["agent_host"].(string)
	apiKey := config["api_key"].(string)
	discoveryMode := config["discovery_mode"].(string)
	
	port := 8080
	if portVal, exists := config["agent_port"]; exists {
		if portFloat, ok := portVal.(float64); ok {
			port = int(portFloat)
		}
	}

	timeout := 300
	if timeoutVal, exists := config["scan_timeout"]; exists {
		if timeoutFloat, ok := timeoutVal.(float64); ok {
			timeout = int(timeoutFloat)
		}
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	var url string
	var requestBody map[string]interface{}

	switch discoveryMode {
	case "agent_scan":
		url = fmt.Sprintf("http://%s:%d/api/v1/discovery/scan", agentHost, port)
		requestBody = map[string]interface{}{
			"scan_type": "local",
		}
	case "agent_network":
		url = fmt.Sprintf("http://%s:%d/api/v1/discovery/network", agentHost, port)
		requestBody = map[string]interface{}{
			"scan_type": "network",
		}
		if networkRange, exists := config["network_range"]; exists {
			requestBody["network_range"] = networkRange
		}
	case "agent_service":
		url = fmt.Sprintf("http://%s:%d/api/v1/discovery/services", agentHost, port)
		requestBody = map[string]interface{}{
			"scan_type": "services",
		}
		if includeServices, exists := config["include_services"]; exists {
			requestBody["service_types"] = includeServices
		}
		if customPorts, exists := config["custom_ports"]; exists {
			requestBody["custom_ports"] = customPorts
		}
	default:
		return nil, fmt.Errorf("unsupported discovery mode: %s", discoveryMode)
	}

	// 序列化请求体
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %v", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", url, strings.NewReader(string(requestBodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 添加认证头
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求Agent失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Agent返回错误状态: %d", resp.StatusCode)
	}

	// 解析响应
	var agentResp struct {
		Success bool                     `json:"success"`
		Message string                   `json:"message"`
		Data    []map[string]interface{} `json:"data"`
		Total   int                      `json:"total"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&agentResp); err != nil {
		return nil, fmt.Errorf("解析Agent响应失败: %v", err)
	}

	if !agentResp.Success {
		return nil, fmt.Errorf("Agent发现失败: %s", agentResp.Message)
	}

	// 转换为标准格式
	records := make([]map[string]interface{}, 0, len(agentResp.Data))
	for _, item := range agentResp.Data {
		record := p.normalizeRecord(item, discoveryMode)
		if record != nil {
			records = append(records, record)
		}
	}

	return &DiscoveryResult{
		Success:      true,
		TotalRecords: int64(len(records)),
		Records:      records,
		Metadata: map[string]interface{}{
			"discovery_mode": discoveryMode,
			"agent_host":     agentHost,
			"agent_port":     port,
			"scan_timeout":   timeout,
			"agent_total":    agentResp.Total,
		},
	}, nil
}

// normalizeRecord 将Agent返回的数据标准化为统一格式
func (p *NBAgentProvider) normalizeRecord(item map[string]interface{}, mode string) map[string]interface{} {
	record := make(map[string]interface{})

	// 通用字段映射
	if hostname, exists := item["hostname"]; exists {
		record["hostname"] = hostname
	}
	if ip, exists := item["ip"]; exists {
		record["ip_address"] = ip
	} else if ipAddr, exists := item["ip_address"]; exists {
		record["ip_address"] = ipAddr
	}
	if mac, exists := item["mac"]; exists {
		record["mac_address"] = mac
	} else if macAddr, exists := item["mac_address"]; exists {
		record["mac_address"] = macAddr
	}

	// 系统信息
	if osType, exists := item["os_type"]; exists {
		record["os_type"] = osType
	} else if os, exists := item["os"]; exists {
		record["os_type"] = os
	}
	if osVersion, exists := item["os_version"]; exists {
		record["os_version"] = osVersion
	}

	// 硬件信息
	if cpu, exists := item["cpu_cores"]; exists {
		if cpuInt, ok := cpu.(float64); ok {
			record["cpu_cores"] = int(cpuInt)
		}
	}
	if memory, exists := item["memory_mb"]; exists {
		if memInt, ok := memory.(float64); ok {
			record["memory_mb"] = int(memInt)
		}
	} else if memoryGB, exists := item["memory_gb"]; exists {
		if memFloat, ok := memoryGB.(float64); ok {
			record["memory_mb"] = int(memFloat * 1024)
		}
	}
	if disk, exists := item["disk_gb"]; exists {
		if diskInt, ok := disk.(float64); ok {
			record["disk_gb"] = int(diskInt)
		}
	}

	// 服务和端口信息
	if services, exists := item["services"]; exists {
		record["services"] = services
	}
	if ports, exists := item["open_ports"]; exists {
		record["open_ports"] = ports
	} else if openPorts, exists := item["ports"]; exists {
		record["open_ports"] = openPorts
	}

	// Agent特定信息
	if version, exists := item["agent_version"]; exists {
		record["agent_version"] = version
	}
	if lastOnline, exists := item["last_online"]; exists {
		record["last_online"] = lastOnline
	}
	if status, exists := item["status"]; exists {
		record["status"] = status
	} else {
		record["status"] = "online" // Agent能返回数据说明在线
	}

	// 模式特定的处理
	switch mode {
	case "agent_network":
		// 网络扫描可能只有基本信息
		if record["hostname"] == nil {
			if ip, exists := record["ip_address"]; exists {
				record["hostname"] = fmt.Sprintf("host_%s", strings.ReplaceAll(ip.(string), ".", "_"))
			}
		}
	case "agent_service":
		// 服务扫描重点关注服务信息
		if record["services"] == nil {
			record["services"] = []interface{}{}
		}
		if record["open_ports"] == nil {
			record["open_ports"] = []interface{}{}
		}
	}

	return record
}

func (p *NBAgentProvider) GetFieldMapping(targetSchema string) (*FieldMappingConfig, error) {
	mappings := make(map[string][]FieldMapping)

	// 通用映射规则
	commonMappings := []FieldMapping{
		{
			SourceField: "hostname",
			TargetField: "device_name",
			Transform:   "direct",
		},
		{
			SourceField: "ip_address",
			TargetField: "ip",
			Transform:   "direct",
		},
		{
			SourceField: "mac_address",
			TargetField: "mac",
			Transform:   "direct",
		},
		{
			SourceField: "os_type",
			TargetField: "os_family",
			Transform:   "direct",
		},
		{
			SourceField: "os_version",
			TargetField: "os_version",
			Transform:   "direct",
		},
		{
			SourceField: "status",
			TargetField: "status",
			Transform:   "direct",
		},
	}

	// 根据目标schema提供特定映射
	switch targetSchema {
	case "cmdb_device":
		mappings["cmdb_device"] = append(commonMappings, []FieldMapping{
			{
				SourceField: "cpu_cores",
				TargetField: "cpu_cores",
				Transform:   "direct",
			},
			{
				SourceField: "memory_mb",
				TargetField: "memory_mb",
				Transform:   "direct",
			},
			{
				SourceField: "disk_gb",
				TargetField: "storage_gb",
				Transform:   "direct",
			},
		}...)
	case "network_device":
		mappings["network_device"] = []FieldMapping{
			{
				SourceField: "hostname",
				TargetField: "device_name",
				Transform:   "direct",
			},
			{
				SourceField: "ip_address",
				TargetField: "management_ip",
				Transform:   "direct",
			},
			{
				SourceField: "mac_address",
				TargetField: "mac_address",
				Transform:   "direct",
			},
			{
				SourceField: "status",
				TargetField: "status",
				Transform:   "direct",
			},
		}
	case "service_instance":
		mappings["service_instance"] = []FieldMapping{
			{
				SourceField: "hostname",
				TargetField: "host_name",
				Transform:   "direct",
			},
			{
				SourceField: "ip_address",
				TargetField: "host_ip",
				Transform:   "direct",
			},
			{
				SourceField: "services",
				TargetField: "services",
				Transform:   "json_extract",
				TransformParams: map[string]interface{}{
					"extract_field": "name",
				},
			},
			{
				SourceField: "open_ports",
				TargetField: "ports",
				Transform:   "json_to_string",
			},
		}
	default:
		mappings["default"] = commonMappings
	}

	return &FieldMappingConfig{
		Version:     "1.0",
		Mappings:    mappings,
		Transforms:  p.getAvailableTransforms(),
		Description: "NewBee Agent发现结果字段映射配置",
	}, nil
}

func (p *NBAgentProvider) getAvailableTransforms() map[string]TransformDefinition {
	return map[string]TransformDefinition{
		"direct": {
			Name:        "direct",
			Description: "直接映射，不做任何转换",
			Parameters:  []string{},
		},
		"json_extract": {
			Name:        "json_extract",
			Description: "从JSON数组中提取指定字段",
			Parameters:  []string{"extract_field"},
		},
		"json_to_string": {
			Name:        "json_to_string",
			Description: "将JSON转换为字符串",
			Parameters:  []string{},
		},
		"upper": {
			Name:        "upper",
			Description: "转换为大写",
			Parameters:  []string{},
		},
		"lower": {
			Name:        "lower",
			Description: "转换为小写",
			Parameters:  []string{},
		},
	}
}