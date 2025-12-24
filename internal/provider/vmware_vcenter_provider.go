package provider

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type VMwareVCenterProvider struct{}

func NewVMwareVCenterProvider() *VMwareVCenterProvider {
	return &VMwareVCenterProvider{}
}

func (p *VMwareVCenterProvider) GetMetadata() *ProviderMetadata {
	return &ProviderMetadata{
		ID:          "vmware_vcenter",
		Name:        "VMware vCenter",
		Description: "VMware vSphere/vCenter虚拟机和主机发现",
		Version:     "1.0.0",
		Type:        "virtualization",
		IconURL:     "/static/icons/vmware.svg",
		SupportedModes: []string{
			"virtual_machines",
			"esxi_hosts",
			"all_resources",
		},
		Tags: []string{"vmware", "vsphere", "vcenter", "virtualization"},
	}
}

func (p *VMwareVCenterProvider) GetParameterSchema() []ParameterDefinition {
	return []ParameterDefinition{
		{
			Name:        "vcenter_host",
			DisplayName: "vCenter主机地址",
			Type:        "string",
			Required:    true,
			Description: "vCenter Server的主机地址或IP",
			Example:     "vcenter.company.com",
		},
		{
			Name:        "username",
			DisplayName: "用户名",
			Type:        "string",
			Required:    true,
			Description: "vCenter登录用户名",
			Example:     "administrator@vsphere.local",
		},
		{
			Name:        "password",
			DisplayName: "密码",
			Type:        "string",
			Required:    true,
			Description: "vCenter登录密码",
			Sensitive:   true,
		},
		{
			Name:        "discovery_mode",
			DisplayName: "发现模式",
			Type:        "select",
			Required:    true,
			Description: "选择发现资源类型",
			DefaultValue: "virtual_machines",
			Options: []SelectOption{
				{Value: "virtual_machines", Label: "虚拟机", Description: "发现vCenter中的虚拟机"},
				{Value: "esxi_hosts", Label: "ESXi主机", Description: "发现vCenter中的ESXi主机"},
				{Value: "all_resources", Label: "所有资源", Description: "发现虚拟机和ESXi主机"},
			},
		},
	}
}

func (p *VMwareVCenterProvider) GetFieldSchema() []FieldDefinition {
	return []FieldDefinition{
		{
			Name:        "object_type",
			DisplayName: "对象类型",
			Type:        "string",
			Description: "资源类型: VirtualMachine, HostSystem",
			Required:    true,
			Searchable:  true,
		},
		{
			Name:        "name",
			DisplayName: "名称",
			Type:        "string",
			Description: "资源名称",
			Required:    true,
			Searchable:  true,
		},
		{
			Name:        "moid",
			DisplayName: "托管对象ID",
			Type:        "string",
			Description: "VMware托管对象ID",
			Required:    true,
			Unique:      true,
		},
		{
			Name:        "power_state",
			DisplayName: "电源状态",
			Type:        "string",
			Description: "虚拟机电源状态",
			Required:    false,
			Searchable:  true,
		},
		{
			Name:        "ip_address",
			DisplayName: "IP地址",
			Type:        "string",
			Description: "主IP地址",
			Required:    false,
			Searchable:  true,
		},
	}
}

func (p *VMwareVCenterProvider) ValidateConfig(config map[string]interface{}) error {
	vcenterHost, ok := config["vcenter_host"].(string)
	if !ok || vcenterHost == "" {
		return fmt.Errorf("vcenter_host is required")
	}

	username, ok := config["username"].(string)
	if !ok || username == "" {
		return fmt.Errorf("username is required")
	}

	password, ok := config["password"].(string)
	if !ok || password == "" {
		return fmt.Errorf("password is required")
	}

	discoveryMode, ok := config["discovery_mode"].(string)
	if !ok || discoveryMode == "" {
		return fmt.Errorf("discovery_mode is required")
	}

	validModes := map[string]bool{
		"virtual_machines": true,
		"esxi_hosts":      true,
		"all_resources":   true,
	}
	if !validModes[discoveryMode] {
		return fmt.Errorf("invalid discovery_mode: %s", discoveryMode)
	}

	return nil
}

func (p *VMwareVCenterProvider) TestConnection(config map[string]interface{}) (*TestResult, error) {
	vcenterHost := config["vcenter_host"].(string)
	username := config["username"].(string)
	password := config["password"].(string)
	
	port := 443

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	sessionURL := fmt.Sprintf("https://%s:%d/rest/com/vmware/cis/session", vcenterHost, port)
	
	req, err := http.NewRequest("POST", sessionURL, nil)
	if err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("创建会话请求失败: %v", err),
		}, nil
	}

	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("连接vCenter失败: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("vCenter返回错误状态: %d", resp.StatusCode),
		}, nil
	}

	var sessionResp struct {
		Value string `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("解析会话响应失败: %v", err),
		}, nil
	}

	return &TestResult{
		Success: true,
		Message: "连接vCenter成功",
		Details: map[string]interface{}{
			"host":       vcenterHost,
			"port":       port,
			"session_id": sessionResp.Value[:8] + "...",
		},
	}, nil
}

func (p *VMwareVCenterProvider) Discover(config map[string]interface{}) (*DiscoveryResult, error) {
	vcenterHost := config["vcenter_host"].(string)
	username := config["username"].(string)
	password := config["password"].(string)
	discoveryMode := config["discovery_mode"].(string)
	
	port := 443

	vcenterClient := &VCenterClient{
		Host:      vcenterHost,
		Port:      port,
		Username:  username,
		Password:  password,
		IgnoreSSL: true,
		Timeout:   30,
	}

	if err := vcenterClient.Connect(); err != nil {
		return nil, fmt.Errorf("连接vCenter失败: %v", err)
	}
	defer vcenterClient.Disconnect()

	var allRecords []map[string]interface{}

	switch discoveryMode {
	case "virtual_machines":
		vms, err := vcenterClient.GetVirtualMachines()
		if err != nil {
			return nil, fmt.Errorf("获取虚拟机失败: %v", err)
		}
		for _, vm := range vms {
			record := p.normalizeVirtualMachine(vm)
			allRecords = append(allRecords, record)
		}
	case "esxi_hosts":
		hosts, err := vcenterClient.GetESXiHosts()
		if err != nil {
			return nil, fmt.Errorf("获取ESXi主机失败: %v", err)
		}
		for _, host := range hosts {
			record := p.normalizeESXiHost(host)
			allRecords = append(allRecords, record)
		}
	case "all_resources":
		vms, err := vcenterClient.GetVirtualMachines()
		if err != nil {
			return nil, fmt.Errorf("获取虚拟机失败: %v", err)
		}
		for _, vm := range vms {
			record := p.normalizeVirtualMachine(vm)
			allRecords = append(allRecords, record)
		}
		
		hosts, err := vcenterClient.GetESXiHosts()
		if err != nil {
			return nil, fmt.Errorf("获取ESXi主机失败: %v", err)
		}
		for _, host := range hosts {
			record := p.normalizeESXiHost(host)
			allRecords = append(allRecords, record)
		}
	default:
		return nil, fmt.Errorf("unsupported discovery mode: %s", discoveryMode)
	}

	return &DiscoveryResult{
		Success:      true,
		TotalRecords: int64(len(allRecords)),
		Records:      allRecords,
		Metadata: map[string]interface{}{
			"vcenter_host":   vcenterHost,
			"discovery_mode": discoveryMode,
		},
	}, nil
}

type VCenterClient struct {
	Host      string
	Port      int
	Username  string
	Password  string
	IgnoreSSL bool
	Timeout   int
	
	client    *http.Client
	sessionID string
}

func (c *VCenterClient) Connect() error {
	c.client = &http.Client{
		Timeout: time.Duration(c.Timeout) * time.Second,
	}

	if c.IgnoreSSL {
		c.client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	sessionURL := fmt.Sprintf("https://%s:%d/rest/com/vmware/cis/session", c.Host, c.Port)
	
	req, err := http.NewRequest("POST", sessionURL, nil)
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed: status %d", resp.StatusCode)
	}

	var sessionResp struct {
		Value string `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		return err
	}

	c.sessionID = sessionResp.Value
	return nil
}

func (c *VCenterClient) Disconnect() {
	if c.sessionID != "" {
		sessionURL := fmt.Sprintf("https://%s:%d/rest/com/vmware/cis/session", c.Host, c.Port)
		req, _ := http.NewRequest("DELETE", sessionURL, nil)
		req.Header.Set("vmware-api-session-id", c.sessionID)
		c.client.Do(req)
	}
}

func (c *VCenterClient) apiCall(method, path string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://%s:%d%s", c.Host, c.Port, path)
	
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("vmware-api-session-id", c.sessionID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API call failed: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *VCenterClient) GetVirtualMachines() ([]map[string]interface{}, error) {
	result, err := c.apiCall("GET", "/rest/vcenter/vm")
	if err != nil {
		return nil, err
	}

	vms, ok := result["value"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected API response format")
	}

	var vmList []map[string]interface{}
	for _, vmData := range vms {
		if vm, ok := vmData.(map[string]interface{}); ok {
			vmList = append(vmList, vm)
		}
	}

	return vmList, nil
}

func (c *VCenterClient) GetESXiHosts() ([]map[string]interface{}, error) {
	result, err := c.apiCall("GET", "/rest/vcenter/host")
	if err != nil {
		return nil, err
	}

	hosts, ok := result["value"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected API response format")
	}

	var hostList []map[string]interface{}
	for _, hostData := range hosts {
		if host, ok := hostData.(map[string]interface{}); ok {
			hostList = append(hostList, host)
		}
	}

	return hostList, nil
}

func (p *VMwareVCenterProvider) normalizeVirtualMachine(vm map[string]interface{}) map[string]interface{} {
	record := make(map[string]interface{})

	record["object_type"] = "VirtualMachine"

	if name, exists := vm["name"]; exists {
		record["name"] = name
	}
	if vmID, exists := vm["vm"]; exists {
		record["moid"] = vmID
	}
	if powerState, exists := vm["power_state"]; exists {
		record["power_state"] = powerState
	}

	record["ip_address"] = ""

	return record
}

func (p *VMwareVCenterProvider) normalizeESXiHost(host map[string]interface{}) map[string]interface{} {
	record := make(map[string]interface{})

	record["object_type"] = "HostSystem"

	if name, exists := host["name"]; exists {
		record["name"] = name
	}
	if hostID, exists := host["host"]; exists {
		record["moid"] = hostID
	}
	if connectionState, exists := host["connection_state"]; exists {
		record["power_state"] = connectionState
	}

	record["ip_address"] = ""

	return record
}

func (p *VMwareVCenterProvider) GetFieldMapping(targetSchema string) (*FieldMappingConfig, error) {
	mappings := make(map[string][]FieldMapping)

	commonMappings := []FieldMapping{
		{
			SourceField: "name",
			TargetField: "device_name",
			Transform:   "direct",
		},
		{
			SourceField: "moid",
			TargetField: "device_id",
			Transform:   "direct",
		},
		{
			SourceField: "ip_address",
			TargetField: "ip",
			Transform:   "direct",
		},
		{
			SourceField: "power_state",
			TargetField: "status",
			Transform:   "direct",
		},
	}

	mappings["default"] = commonMappings

	return &FieldMappingConfig{
		Version:     "1.0",
		Mappings:    mappings,
		Transforms: map[string]TransformDefinition{
			"direct": {
				Name:        "direct",
				Description: "直接映射，不做任何转换",
				Parameters:  []string{},
			},
		},
		Description: "VMware vCenter发现结果字段映射配置",
	}, nil
}