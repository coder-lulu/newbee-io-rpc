package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHProvider SSH 主机发现提供商
type SSHProvider struct{}

// NewSSHProvider 创建 SSH Provider 实例
func NewSSHProvider() *SSHProvider {
	return &SSHProvider{}
}

// GetMetadata 获取 Provider 元数据
func (p *SSHProvider) GetMetadata() *ProviderMetadata {
	return &ProviderMetadata{
		ID:          "ssh",
		Name:        "SSH 主机发现",
		Description: "通过 SSH 连接发现 Linux/Unix 主机信息",
		Version:     "1.0.0",
		Type:        "host",
		Category:    "infrastructure",
		IconURL:     "/static/icons/ssh.svg",
		SupportedModes: []string{
			"system_info",  // 系统信息
			"hardware",     // 硬件信息
			"network",      // 网络信息
			"full",         // 完整信息
		},
		Tags: []string{"ssh", "linux", "unix", "host", "server"},
	}
}

// GetParameterSchema 获取参数定义
func (p *SSHProvider) GetParameterSchema() []ParameterDefinition {
	return []ParameterDefinition{
		{
			Name:        "host",
			DisplayName: "主机地址",
			Type:        "string",
			Required:    true,
			Description: "SSH 主机地址或 IP",
			Example:     "192.168.1.10 或 server.example.com",
		},
		{
			Name:        "port",
			DisplayName: "SSH 端口",
			Type:        "integer",
			Required:    false,
			Description: "SSH 服务端口",
			DefaultValue: 22,
			Example:     "22",
			Validation: map[string]interface{}{
				"min": 1,
				"max": 65535,
			},
		},
		{
			Name:        "username",
			DisplayName: "用户名",
			Type:        "string",
			Required:    true,
			Description: "SSH 登录用户名",
			Example:     "root",
		},
		{
			Name:        "auth_method",
			DisplayName: "认证方式",
			Type:        "select",
			Required:    true,
			Description: "SSH 认证方式",
			DefaultValue: "password",
			Options: []SelectOption{
				{Value: "password", Label: "密码认证", Description: "使用密码进行 SSH 认证"},
				{Value: "key", Label: "密钥认证", Description: "使用私钥文件进行 SSH 认证"},
			},
		},
		{
			Name:        "password",
			DisplayName: "密码",
			Type:        "password",
			Required:    false,
			Description: "SSH 登录密码（auth_method=password 时必填）",
			Sensitive:   true,
		},
		{
			Name:        "private_key",
			DisplayName: "私钥内容",
			Type:        "textarea",
			Required:    false,
			Description: "SSH 私钥内容（auth_method=key 时必填）",
			Sensitive:   true,
			Example:     "-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----",
		},
		{
			Name:        "passphrase",
			DisplayName: "私钥密码",
			Type:        "password",
			Required:    false,
			Description: "私钥文件的加密密码（如果私钥已加密）",
			Sensitive:   true,
		},
		{
			Name:        "timeout",
			DisplayName: "连接超时(秒)",
			Type:        "integer",
			Required:    false,
			Description: "SSH 连接和命令执行超时时间",
			DefaultValue: 30,
			Example:     "30",
			Validation: map[string]interface{}{
				"min": 5,
				"max": 300,
			},
		},
		{
			Name:        "discover_mode",
			DisplayName: "发现模式",
			Type:        "select",
			Required:    false,
			Description: "选择要收集的信息类型",
			DefaultValue: "full",
			Options: []SelectOption{
				{Value: "system_info", Label: "系统信息", Description: "操作系统、主机名、内核版本"},
				{Value: "hardware", Label: "硬件信息", Description: "CPU、内存、磁盘"},
				{Value: "network", Label: "网络信息", Description: "IP地址、网卡信息"},
				{Value: "full", Label: "完整信息", Description: "收集所有可用信息"},
			},
		},
	}
}

// GetFieldSchema 获取字段定义
func (p *SSHProvider) GetFieldSchema() []FieldDefinition {
	return []FieldDefinition{
		{Name: "hostname", DisplayName: "主机名", Type: "string", Required: true, Unique: true},
		{Name: "os_type", DisplayName: "操作系统类型", Type: "string", Required: false},
		{Name: "os_version", DisplayName: "操作系统版本", Type: "string", Required: false},
		{Name: "kernel_version", DisplayName: "内核版本", Type: "string", Required: false},
		{Name: "architecture", DisplayName: "系统架构", Type: "string", Required: false},
		{Name: "cpu_model", DisplayName: "CPU型号", Type: "string", Required: false},
		{Name: "cpu_cores", DisplayName: "CPU核心数", Type: "integer", Required: false},
		{Name: "cpu_count", DisplayName: "CPU数量", Type: "integer", Required: false},
		{Name: "memory_total_mb", DisplayName: "总内存(MB)", Type: "integer", Required: false},
		{Name: "memory_used_mb", DisplayName: "已用内存(MB)", Type: "integer", Required: false},
		{Name: "memory_free_mb", DisplayName: "空闲内存(MB)", Type: "integer", Required: false},
		{Name: "disk_total_gb", DisplayName: "总磁盘(GB)", Type: "integer", Required: false},
		{Name: "disk_used_gb", DisplayName: "已用磁盘(GB)", Type: "integer", Required: false},
		{Name: "disk_free_gb", DisplayName: "空闲磁盘(GB)", Type: "integer", Required: false},
		{Name: "ip_address", DisplayName: "IP地址", Type: "string", Required: false, Searchable: true},
		{Name: "mac_address", DisplayName: "MAC地址", Type: "string", Required: false},
		{Name: "uptime_days", DisplayName: "运行时间(天)", Type: "integer", Required: false},
		{Name: "load_average_1min", DisplayName: "负载(1分钟)", Type: "float", Required: false},
		{Name: "load_average_5min", DisplayName: "负载(5分钟)", Type: "float", Required: false},
		{Name: "load_average_15min", DisplayName: "负载(15分钟)", Type: "float", Required: false},
	}
}

// ValidateConfig 验证配置参数
func (p *SSHProvider) ValidateConfig(config map[string]interface{}) error {
	// 验证必填字段
	host, ok := config["host"].(string)
	if !ok || host == "" {
		return fmt.Errorf("host is required")
	}

	username, ok := config["username"].(string)
	if !ok || username == "" {
		return fmt.Errorf("username is required")
	}

	// 验证认证方式
	authMethod, _ := config["auth_method"].(string)
	if authMethod == "" {
		authMethod = "password"
	}

	switch authMethod {
	case "password":
		password, ok := config["password"].(string)
		if !ok || password == "" {
			return fmt.Errorf("password is required when auth_method=password")
		}
	case "key":
		privateKey, ok := config["private_key"].(string)
		if !ok || privateKey == "" {
			return fmt.Errorf("private_key is required when auth_method=key")
		}
	default:
		return fmt.Errorf("invalid auth_method: %s (must be 'password' or 'key')", authMethod)
	}

	return nil
}

// TestConnection 测试 SSH 连接
func (p *SSHProvider) TestConnection(config map[string]interface{}) (*TestResult, error) {
	startTime := time.Now()

	// 创建 SSH 客户端
	client, err := p.createSSHClient(config)
	if err != nil {
		return &TestResult{
			Success: false,
			Message: "SSH 连接失败",
			Error:   err.Error(),
			Latency: time.Since(startTime).Milliseconds(),
		}, nil
	}
	defer client.Close()

	// 执行简单的测试命令
	session, err := client.NewSession()
	if err != nil {
		return &TestResult{
			Success: false,
			Message: "创建 SSH 会话失败",
			Error:   err.Error(),
			Latency: time.Since(startTime).Milliseconds(),
		}, nil
	}
	defer session.Close()

	output, err := session.CombinedOutput("echo 'test'")
	if err != nil {
		return &TestResult{
			Success: false,
			Message: "执行测试命令失败",
			Error:   err.Error(),
			Latency: time.Since(startTime).Milliseconds(),
		}, nil
	}

	latency := time.Since(startTime).Milliseconds()

	return &TestResult{
		Success: true,
		Message: fmt.Sprintf("SSH 连接成功，延迟: %dms", latency),
		Latency: latency,
		Details: map[string]interface{}{
			"output": strings.TrimSpace(string(output)),
			"host":   config["host"],
		},
	}, nil
}

// Discover 执行主机发现
func (p *SSHProvider) Discover(config map[string]interface{}) (*DiscoveryResult, error) {
	// 创建 SSH 客户端
	client, err := p.createSSHClient(config)
	if err != nil {
		return &DiscoveryResult{
			Success:      false,
			TotalRecords: 0,
			ErrorMessage: fmt.Sprintf("SSH connection failed: %v", err),
		}, nil
	}
	defer client.Close()

	// 获取发现模式
	discoverMode, _ := config["discover_mode"].(string)
	if discoverMode == "" {
		discoverMode = "full"
	}

	// 收集主机信息
	hostInfo := make(map[string]interface{})

	// 收集系统信息
	if discoverMode == "system_info" || discoverMode == "full" {
		if err := p.collectSystemInfo(client, hostInfo); err != nil {
			return &DiscoveryResult{
				Success:      false,
				TotalRecords: 0,
				ErrorMessage: fmt.Sprintf("Failed to collect system info: %v", err),
			}, nil
		}
	}

	// 收集硬件信息
	if discoverMode == "hardware" || discoverMode == "full" {
		if err := p.collectHardwareInfo(client, hostInfo); err != nil {
			return &DiscoveryResult{
				Success:      false,
				TotalRecords: 0,
				ErrorMessage: fmt.Sprintf("Failed to collect hardware info: %v", err),
			}, nil
		}
	}

	// 收集网络信息
	if discoverMode == "network" || discoverMode == "full" {
		if err := p.collectNetworkInfo(client, hostInfo); err != nil {
			return &DiscoveryResult{
				Success:      false,
				TotalRecords: 0,
				ErrorMessage: fmt.Sprintf("Failed to collect network info: %v", err),
			}, nil
		}
	}

	return &DiscoveryResult{
		Success:      true,
		TotalRecords: 1,
		Records:      []map[string]interface{}{hostInfo},
		Metadata: map[string]interface{}{
			"discover_mode": discoverMode,
			"host":          config["host"],
			"collected_at":  time.Now().Format(time.RFC3339),
		},
	}, nil
}

// GetFieldMapping 获取字段映射配置
func (p *SSHProvider) GetFieldMapping(targetSchema string) (*FieldMappingConfig, error) {
	// 默认映射配置
	mappings := map[string][]FieldMapping{
		"cmdb_server": {
			{SourceField: "hostname", TargetField: "name", Transform: "direct"},
			{SourceField: "os_type", TargetField: "os_type", Transform: "direct"},
			{SourceField: "os_version", TargetField: "os_version", Transform: "direct"},
			{SourceField: "cpu_cores", TargetField: "cpu_cores", Transform: "int"},
			{SourceField: "memory_total_mb", TargetField: "memory_mb", Transform: "int"},
			{SourceField: "ip_address", TargetField: "ip_address", Transform: "direct"},
		},
	}

	return &FieldMappingConfig{
		Version:     "1.0.0",
		Mappings:    mappings,
		Description: "SSH Provider 默认字段映射",
	}, nil
}

// ========== 私有方法 ==========

// createSSHClient 创建 SSH 客户端连接
func (p *SSHProvider) createSSHClient(config map[string]interface{}) (*ssh.Client, error) {
	host, _ := config["host"].(string)
	username, _ := config["username"].(string)

	// 获取端口
	port := 22
	if p, ok := config["port"].(float64); ok {
		port = int(p)
	} else if p, ok := config["port"].(int); ok {
		port = p
	}

	// 获取超时时间
	timeout := 30
	if t, ok := config["timeout"].(float64); ok {
		timeout = int(t)
	} else if t, ok := config["timeout"].(int); ok {
		timeout = t
	}

	// 构建认证方法
	var authMethods []ssh.AuthMethod
	authMethod, _ := config["auth_method"].(string)
	if authMethod == "" {
		authMethod = "password"
	}

	switch authMethod {
	case "password":
		password, _ := config["password"].(string)
		authMethods = append(authMethods, ssh.Password(password))

	case "key":
		privateKeyStr, _ := config["private_key"].(string)
		passphrase, _ := config["passphrase"].(string)

		var signer ssh.Signer
		var err error
		if passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(privateKeyStr), []byte(passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(privateKeyStr))
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// SSH 客户端配置
	sshConfig := &ssh.ClientConfig{
		User:            username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应该验证 host key
		Timeout:         time.Duration(timeout) * time.Second,
	}

	// 建立连接
	address := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	return client, nil
}

// executeCommand 执行 SSH 命令
func (p *SSHProvider) executeCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return "", fmt.Errorf("command failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// collectSystemInfo 收集系统信息
func (p *SSHProvider) collectSystemInfo(client *ssh.Client, hostInfo map[string]interface{}) error {
	// 获取主机名
	hostname, err := p.executeCommand(client, "hostname")
	if err == nil {
		hostInfo["hostname"] = hostname
	}

	// 获取操作系统信息 (uname -a)
	unameOutput, err := p.executeCommand(client, "uname -a")
	if err == nil {
		hostInfo["uname_output"] = unameOutput
		// 解析 uname 输出
		parts := strings.Fields(unameOutput)
		if len(parts) >= 3 {
			hostInfo["os_type"] = parts[0]      // Linux
			hostInfo["kernel_version"] = parts[2] // 5.4.0-42-generic
		}
	}

	// 获取系统架构
	arch, err := p.executeCommand(client, "uname -m")
	if err == nil {
		hostInfo["architecture"] = arch // x86_64
	}

	// 获取操作系统发行版信息
	osRelease, err := p.executeCommand(client, "cat /etc/os-release 2>/dev/null || cat /etc/redhat-release 2>/dev/null")
	if err == nil && osRelease != "" {
		hostInfo["os_release"] = osRelease
		// 尝试提取版本号
		if strings.Contains(osRelease, "VERSION=") {
			re := regexp.MustCompile(`VERSION="([^"]+)"`)
			if matches := re.FindStringSubmatch(osRelease); len(matches) > 1 {
				hostInfo["os_version"] = matches[1]
			}
		}
	}

	// 获取系统运行时间
	uptime, err := p.executeCommand(client, "cat /proc/uptime")
	if err == nil {
		parts := strings.Fields(uptime)
		if len(parts) > 0 {
			if uptimeSeconds, err := strconv.ParseFloat(parts[0], 64); err == nil {
				hostInfo["uptime_seconds"] = int(uptimeSeconds)
				hostInfo["uptime_days"] = int(uptimeSeconds / 86400)
			}
		}
	}

	// 获取系统负载
	loadavg, err := p.executeCommand(client, "cat /proc/loadavg")
	if err == nil {
		parts := strings.Fields(loadavg)
		if len(parts) >= 3 {
			if load1, err := strconv.ParseFloat(parts[0], 64); err == nil {
				hostInfo["load_average_1min"] = load1
			}
			if load5, err := strconv.ParseFloat(parts[1], 64); err == nil {
				hostInfo["load_average_5min"] = load5
			}
			if load15, err := strconv.ParseFloat(parts[2], 64); err == nil {
				hostInfo["load_average_15min"] = load15
			}
		}
	}

	return nil
}

// collectHardwareInfo 收集硬件信息
func (p *SSHProvider) collectHardwareInfo(client *ssh.Client, hostInfo map[string]interface{}) error {
	// 获取 CPU 信息
	cpuInfo, err := p.executeCommand(client, "cat /proc/cpuinfo")
	if err == nil {
		// 统计 CPU 核心数
		coreCount := strings.Count(cpuInfo, "processor")
		hostInfo["cpu_cores"] = coreCount
		hostInfo["cpu_count"] = coreCount

		// 提取 CPU 型号
		re := regexp.MustCompile(`model name\s*:\s*(.+)`)
		if matches := re.FindStringSubmatch(cpuInfo); len(matches) > 1 {
			hostInfo["cpu_model"] = strings.TrimSpace(matches[1])
		}
	}

	// 获取内存信息 (单位: MB)
	memInfo, err := p.executeCommand(client, "free -m")
	if err == nil {
		lines := strings.Split(memInfo, "\n")
		if len(lines) >= 2 {
			// 第二行是 Mem 信息
			fields := strings.Fields(lines[1])
			if len(fields) >= 4 {
				if total, err := strconv.Atoi(fields[1]); err == nil {
					hostInfo["memory_total_mb"] = total
				}
				if used, err := strconv.Atoi(fields[2]); err == nil {
					hostInfo["memory_used_mb"] = used
				}
				if free, err := strconv.Atoi(fields[3]); err == nil {
					hostInfo["memory_free_mb"] = free
				}
			}
		}
	}

	// 获取磁盘信息 (单位: GB)
	diskInfo, err := p.executeCommand(client, "df -BG / | tail -1")
	if err == nil {
		fields := strings.Fields(diskInfo)
		if len(fields) >= 4 {
			// 去除 'G' 后缀
			totalStr := strings.TrimSuffix(fields[1], "G")
			usedStr := strings.TrimSuffix(fields[2], "G")
			freeStr := strings.TrimSuffix(fields[3], "G")

			if total, err := strconv.Atoi(totalStr); err == nil {
				hostInfo["disk_total_gb"] = total
			}
			if used, err := strconv.Atoi(usedStr); err == nil {
				hostInfo["disk_used_gb"] = used
			}
			if free, err := strconv.Atoi(freeStr); err == nil {
				hostInfo["disk_free_gb"] = free
			}
		}
	}

	return nil
}

// collectNetworkInfo 收集网络信息
func (p *SSHProvider) collectNetworkInfo(client *ssh.Client, hostInfo map[string]interface{}) error {
	// 获取 IP 地址 (排除 loopback)
	ipAddr, err := p.executeCommand(client, "ip -4 addr show | grep inet | grep -v 127.0.0.1 | head -1 | awk '{print $2}' | cut -d/ -f1")
	if err == nil && ipAddr != "" {
		hostInfo["ip_address"] = ipAddr
	}

	// 如果上面的命令失败，尝试 ifconfig
	if ipAddr == "" {
		ipAddr, err = p.executeCommand(client, "ifconfig | grep 'inet ' | grep -v 127.0.0.1 | head -1 | awk '{print $2}'")
		if err == nil && ipAddr != "" {
			hostInfo["ip_address"] = ipAddr
		}
	}

	// 获取 MAC 地址
	macAddr, err := p.executeCommand(client, "ip link show | grep ether | head -1 | awk '{print $2}'")
	if err == nil && macAddr != "" {
		hostInfo["mac_address"] = macAddr
	}

	// 如果上面的命令失败，尝试 ifconfig
	if macAddr == "" {
		macAddr, err = p.executeCommand(client, "ifconfig | grep ether | head -1 | awk '{print $2}'")
		if err == nil && macAddr != "" {
			hostInfo["mac_address"] = macAddr
		}
	}

	return nil
}
