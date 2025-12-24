package provider

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type AliyunECSProvider struct{}

func NewAliyunECSProvider() *AliyunECSProvider {
	return &AliyunECSProvider{}
}

func (p *AliyunECSProvider) GetMetadata() *ProviderMetadata {
	return &ProviderMetadata{
		ID:          "aliyun_ecs",
		Name:        "阿里云ECS",
		Description: "阿里云弹性计算服务实例发现",
		Version:     "1.0.0",
		Type:        "cloud",
		IconURL:     "/static/icons/aliyun.svg",
		SupportedModes: []string{
			"ecs_instances", // 发现ECS实例
			"all_regions",   // 所有区域
			"specific_region", // 指定区域
		},
		Tags: []string{"cloud", "aliyun", "ecs", "compute"},
	}
}

func (p *AliyunECSProvider) GetParameterSchema() []ParameterDefinition {
	return []ParameterDefinition{
		{
			Name:        "access_key_id",
			DisplayName: "AccessKey ID",
			Type:        "string",
			Required:    true,
			Description: "阿里云访问密钥ID",
			Sensitive:   false,
			Example:     "LTAI4G8G8G8G8G8G8G8G",
		},
		{
			Name:        "access_key_secret",
			DisplayName: "AccessKey Secret",
			Type:        "string",
			Required:    true,
			Description: "阿里云访问密钥Secret",
			Sensitive:   true,
			Example:     "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		},
		{
			Name:        "region_id",
			DisplayName: "地域ID",
			Type:        "select",
			Required:    false,
			Description: "指定地域ID，留空则扫描所有地域",
			DefaultValue: "",
			Options: []SelectOption{
				{Value: "", Label: "所有地域", Description: "扫描所有可用地域"},
				{Value: "cn-hangzhou", Label: "华东1(杭州)", Description: "cn-hangzhou"},
				{Value: "cn-shanghai", Label: "华东2(上海)", Description: "cn-shanghai"},
				{Value: "cn-beijing", Label: "华北2(北京)", Description: "cn-beijing"},
				{Value: "cn-shenzhen", Label: "华南1(深圳)", Description: "cn-shenzhen"},
				{Value: "cn-qingdao", Label: "华北1(青岛)", Description: "cn-qingdao"},
				{Value: "cn-zhangjiakou", Label: "华北3(张家口)", Description: "cn-zhangjiakou"},
				{Value: "cn-huhehaote", Label: "华北5(呼和浩特)", Description: "cn-huhehaote"},
				{Value: "cn-chengdu", Label: "西南1(成都)", Description: "cn-chengdu"},
				{Value: "cn-hongkong", Label: "香港", Description: "cn-hongkong"},
				{Value: "ap-southeast-1", Label: "新加坡", Description: "ap-southeast-1"},
				{Value: "ap-southeast-2", Label: "澳大利亚(悉尼)", Description: "ap-southeast-2"},
				{Value: "ap-southeast-3", Label: "马来西亚(吉隆坡)", Description: "ap-southeast-3"},
				{Value: "ap-southeast-5", Label: "印度尼西亚(雅加达)", Description: "ap-southeast-5"},
				{Value: "ap-northeast-1", Label: "日本(东京)", Description: "ap-northeast-1"},
				{Value: "ap-south-1", Label: "印度(孟买)", Description: "ap-south-1"},
				{Value: "us-east-1", Label: "美国(弗吉尼亚)", Description: "us-east-1"},
				{Value: "us-west-1", Label: "美国(硅谷)", Description: "us-west-1"},
				{Value: "eu-west-1", Label: "英国(伦敦)", Description: "eu-west-1"},
				{Value: "eu-central-1", Label: "德国(法兰克福)", Description: "eu-central-1"},
				{Value: "me-east-1", Label: "阿联酋(迪拜)", Description: "me-east-1"},
			},
		},
		{
			Name:        "instance_status",
			DisplayName: "实例状态过滤",
			Type:        "multiselect",
			Required:    false,
			Description: "选择要包含的实例状态",
			Options: []SelectOption{
				{Value: "Running", Label: "运行中", Description: "正在运行的实例"},
				{Value: "Stopped", Label: "已停止", Description: "已停止的实例"},
				{Value: "Starting", Label: "启动中", Description: "正在启动的实例"},
				{Value: "Stopping", Label: "停止中", Description: "正在停止的实例"},
			},
		},
		{
			Name:        "include_tags",
			DisplayName: "包含标签",
			Type:        "boolean",
			Required:    false,
			Description: "是否包含实例标签信息",
			DefaultValue: true,
		},
		{
			Name:        "include_security_groups",
			DisplayName: "包含安全组",
			Type:        "boolean",
			Required:    false,
			Description: "是否包含安全组信息",
			DefaultValue: true,
		},
		{
			Name:        "page_size",
			DisplayName: "分页大小",
			Type:        "integer",
			Required:    false,
			Description: "单次API调用返回的实例数量",
			DefaultValue: 100,
			Example:     "100",
			Validation: map[string]interface{}{
				"min": 10,
				"max": 100,
			},
		},
		{
			Name:        "max_instances",
			DisplayName: "最大实例数",
			Type:        "integer",
			Required:    false,
			Description: "限制发现的最大实例数量，0表示不限制",
			DefaultValue: 0,
			Example:     "1000",
			Validation: map[string]interface{}{
				"min": 0,
				"max": 10000,
			},
		},
	}
}

func (p *AliyunECSProvider) GetFieldSchema() []FieldDefinition {
	return []FieldDefinition{
		{
			Name:        "instance_id",
			DisplayName: "实例ID",
			Type:        "string",
			Description: "阿里云ECS实例ID",
			Required:    true,
			Unique:      true,
			Searchable:  true,
		},
		{
			Name:        "instance_name",
			DisplayName: "实例名称",
			Type:        "string",
			Description: "ECS实例的名称",
			Required:    false,
			Searchable:  true,
		},
		{
			Name:        "hostname",
			DisplayName: "主机名",
			Type:        "string",
			Description: "实例的主机名",
			Required:    false,
			Searchable:  true,
		},
		{
			Name:        "instance_type",
			DisplayName: "实例规格",
			Type:        "string",
			Description: "ECS实例规格",
			Required:    false,
			Searchable:  true,
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
			Name:        "os_name",
			DisplayName: "操作系统",
			Type:        "string",
			Description: "操作系统名称",
			Required:    false,
			Searchable:  true,
		},
		{
			Name:        "os_type",
			DisplayName: "系统类型",
			Type:        "string",
			Description: "操作系统类型(linux/windows)",
			Required:    false,
			Searchable:  true,
		},
		{
			Name:        "public_ip",
			DisplayName: "公网IP",
			Type:        "string",
			Description: "公网IP地址",
			Required:    false,
			Searchable:  true,
		},
		{
			Name:        "private_ip",
			DisplayName: "私网IP",
			Type:        "string",
			Description: "私网IP地址",
			Required:    false,
			Searchable:  true,
		},
		{
			Name:        "vpc_id",
			DisplayName: "VPC ID",
			Type:        "string",
			Description: "所属VPC ID",
			Required:    false,
			Searchable:  true,
		},
		{
			Name:        "vswitch_id",
			DisplayName: "交换机ID",
			Type:        "string",
			Description: "所属交换机ID",
			Required:    false,
		},
		{
			Name:        "security_groups",
			DisplayName: "安全组",
			Type:        "json",
			Description: "关联的安全组列表",
			Required:    false,
		},
		{
			Name:        "region_id",
			DisplayName: "地域ID",
			Type:        "string",
			Description: "所在地域",
			Required:    false,
			Searchable:  true,
		},
		{
			Name:        "zone_id",
			DisplayName: "可用区ID",
			Type:        "string",
			Description: "所在可用区",
			Required:    false,
			Searchable:  true,
		},
		{
			Name:        "status",
			DisplayName: "运行状态",
			Type:        "string",
			Description: "实例运行状态",
			Required:    false,
			Searchable:  true,
		},
		{
			Name:        "creation_time",
			DisplayName: "创建时间",
			Type:        "datetime",
			Description: "实例创建时间",
			Required:    false,
		},
		{
			Name:        "expired_time",
			DisplayName: "到期时间",
			Type:        "datetime",
			Description: "预付费实例到期时间",
			Required:    false,
		},
		{
			Name:        "instance_charge_type",
			DisplayName: "付费类型",
			Type:        "string",
			Description: "实例付费类型",
			Required:    false,
		},
		{
			Name:        "tags",
			DisplayName: "标签",
			Type:        "json",
			Description: "实例标签信息",
			Required:    false,
		},
		{
			Name:        "network_type",
			DisplayName: "网络类型",
			Type:        "string",
			Description: "网络类型(vpc/classic)",
			Required:    false,
		},
		{
			Name:        "internet_charge_type",
			DisplayName: "网络计费类型",
			Type:        "string",
			Description: "公网带宽计费类型",
			Required:    false,
		},
		{
			Name:        "internet_max_bandwidth_out",
			DisplayName: "公网出带宽",
			Type:        "integer",
			Description: "公网出方向带宽上限(Mbps)",
			Required:    false,
		},
	}
}

func (p *AliyunECSProvider) ValidateConfig(config map[string]interface{}) error {
	// 验证必需参数
	accessKeyID, ok := config["access_key_id"].(string)
	if !ok || accessKeyID == "" {
		return fmt.Errorf("access_key_id is required")
	}

	accessKeySecret, ok := config["access_key_secret"].(string)
	if !ok || accessKeySecret == "" {
		return fmt.Errorf("access_key_secret is required")
	}

	// 验证分页大小
	if pageSize, exists := config["page_size"]; exists {
		if pageSizeFloat, ok := pageSize.(float64); ok {
			if pageSizeFloat < 10 || pageSizeFloat > 100 {
				return fmt.Errorf("page_size must be between 10 and 100")
			}
		}
	}

	// 验证最大实例数
	if maxInstances, exists := config["max_instances"]; exists {
		if maxFloat, ok := maxInstances.(float64); ok {
			if maxFloat < 0 || maxFloat > 10000 {
				return fmt.Errorf("max_instances must be between 0 and 10000")
			}
		}
	}

	return nil
}

func (p *AliyunECSProvider) TestConnection(config map[string]interface{}) (*TestResult, error) {
	accessKeyID := config["access_key_id"].(string)
	accessKeySecret := config["access_key_secret"].(string)
	
	regionID := "cn-hangzhou" // 使用杭州作为测试区域
	if region, exists := config["region_id"]; exists {
		if regionStr, ok := region.(string); ok && regionStr != "" {
			regionID = regionStr
		}
	}

	// 调用DescribeRegions API进行连接测试
	params := map[string]string{
		"Action":           "DescribeRegions",
		"Version":          "2014-05-26",
		"AccessKeyId":      accessKeyID,
		"SignatureMethod":  "HMAC-SHA1",
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"SignatureVersion": "1.0",
		"SignatureNonce":   strconv.FormatInt(time.Now().UnixNano(), 10),
		"Format":           "JSON",
	}

	// 计算签名
	signature := p.calculateSignature(params, accessKeySecret, "GET")
	params["Signature"] = signature

	// 构建请求URL
	url := fmt.Sprintf("https://ecs.%s.aliyuncs.com/?%s", regionID, p.buildQueryString(params))

	// 发送请求
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("连接阿里云API失败: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	// 解析响应
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("解析API响应失败: %v", err),
		}, nil
	}

	// 检查是否有错误
	if code, exists := result["Code"]; exists {
		return &TestResult{
			Success: false,
			Message: fmt.Sprintf("API调用失败: %s - %s", code, result["Message"]),
			Details: map[string]interface{}{
				"error_code": code,
				"error_msg":  result["Message"],
			},
		}, nil
	}

	// 检查是否有Regions信息
	if regions, exists := result["Regions"]; exists {
		if regionList, ok := regions.(map[string]interface{}); ok {
			if regionArray, ok := regionList["Region"].([]interface{}); ok {
				return &TestResult{
					Success: true,
					Message: "连接阿里云API成功",
					Details: map[string]interface{}{
						"available_regions": len(regionArray),
						"test_region":       regionID,
						"api_version":       "2014-05-26",
					},
				}, nil
			}
		}
	}

	return &TestResult{
		Success: true,
		Message: "连接阿里云API成功",
		Details: map[string]interface{}{
			"test_region": regionID,
			"api_version": "2014-05-26",
		},
	}, nil
}

func (p *AliyunECSProvider) Discover(config map[string]interface{}) (*DiscoveryResult, error) {
	accessKeyID := config["access_key_id"].(string)
	accessKeySecret := config["access_key_secret"].(string)
	
	regionID := ""
	if region, exists := config["region_id"]; exists {
		if regionStr, ok := region.(string); ok {
			regionID = regionStr
		}
	}

	pageSize := 50
	if ps, exists := config["page_size"]; exists {
		if psFloat, ok := ps.(float64); ok {
			pageSize = int(psFloat)
		}
	}

	maxInstances := 0
	if max, exists := config["max_instances"]; exists {
		if maxFloat, ok := max.(float64); ok {
			maxInstances = int(maxFloat)
		}
	}

	includeTagsRaw, _ := config["include_tags"]
	includeTags := true
	if includeTagsRaw != nil {
		if includeTagsBool, ok := includeTagsRaw.(bool); ok {
			includeTags = includeTagsBool
		}
	}

	includeSecurityGroupsRaw, _ := config["include_security_groups"]
	includeSecurityGroups := true
	if includeSecurityGroupsRaw != nil {
		if includeSecGroupsBool, ok := includeSecurityGroupsRaw.(bool); ok {
			includeSecurityGroups = includeSecGroupsBool
		}
	}

	// 状态过滤
	var statusFilter []string
	if statusFilterRaw, exists := config["instance_status"]; exists {
		if statusArray, ok := statusFilterRaw.([]interface{}); ok {
			for _, status := range statusArray {
				if statusStr, ok := status.(string); ok {
					statusFilter = append(statusFilter, statusStr)
				}
			}
		}
	}

	var allRecords []map[string]interface{}
	var regions []string

	// 确定要扫描的区域
	if regionID == "" {
		// 获取所有区域
		regionList, err := p.getRegions(accessKeyID, accessKeySecret)
		if err != nil {
			return nil, fmt.Errorf("获取区域列表失败: %v", err)
		}
		regions = regionList
	} else {
		regions = []string{regionID}
	}

	// 遍历每个区域
	for _, region := range regions {
		regionRecords, err := p.discoverRegion(accessKeyID, accessKeySecret, region, pageSize, maxInstances, statusFilter, includeTags, includeSecurityGroups)
		if err != nil {
			// 记录错误但继续处理其他区域
			fmt.Printf("区域 %s 发现失败: %v\n", region, err)
			continue
		}
		
		allRecords = append(allRecords, regionRecords...)

		// 检查是否达到最大实例数限制
		if maxInstances > 0 && len(allRecords) >= maxInstances {
			allRecords = allRecords[:maxInstances]
			break
		}
	}

	return &DiscoveryResult{
		Success:      true,
		TotalRecords: int64(len(allRecords)),
		Records:      allRecords,
		Metadata: map[string]interface{}{
			"scanned_regions":         regions,
			"include_tags":            includeTags,
			"include_security_groups": includeSecurityGroups,
			"status_filter":           statusFilter,
			"page_size":               pageSize,
			"max_instances":           maxInstances,
		},
	}, nil
}

// getRegions 获取所有可用区域
func (p *AliyunECSProvider) getRegions(accessKeyID, accessKeySecret string) ([]string, error) {
	params := map[string]string{
		"Action":           "DescribeRegions",
		"Version":          "2014-05-26",
		"AccessKeyId":      accessKeyID,
		"SignatureMethod":  "HMAC-SHA1",
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"SignatureVersion": "1.0",
		"SignatureNonce":   strconv.FormatInt(time.Now().UnixNano(), 10),
		"Format":           "JSON",
	}

	signature := p.calculateSignature(params, accessKeySecret, "GET")
	params["Signature"] = signature

	url := fmt.Sprintf("https://ecs.cn-hangzhou.aliyuncs.com/?%s", p.buildQueryString(params))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if code, exists := result["Code"]; exists {
		return nil, fmt.Errorf("API错误: %s - %s", code, result["Message"])
	}

	var regions []string
	if regionsData, exists := result["Regions"]; exists {
		if regionList, ok := regionsData.(map[string]interface{}); ok {
			if regionArray, ok := regionList["Region"].([]interface{}); ok {
				for _, regionItem := range regionArray {
					if region, ok := regionItem.(map[string]interface{}); ok {
						if regionId, ok := region["RegionId"].(string); ok {
							regions = append(regions, regionId)
						}
					}
				}
			}
		}
	}

	return regions, nil
}

// discoverRegion 发现指定区域的ECS实例
func (p *AliyunECSProvider) discoverRegion(accessKeyID, accessKeySecret, regionID string, pageSize, maxInstances int, statusFilter []string, includeTags, includeSecurityGroups bool) ([]map[string]interface{}, error) {
	var allInstances []map[string]interface{}
	pageNumber := 1

	for {
		params := map[string]string{
			"Action":           "DescribeInstances",
			"Version":          "2014-05-26",
			"RegionId":         regionID,
			"PageSize":         strconv.Itoa(pageSize),
			"PageNumber":       strconv.Itoa(pageNumber),
			"AccessKeyId":      accessKeyID,
			"SignatureMethod":  "HMAC-SHA1",
			"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
			"SignatureVersion": "1.0",
			"SignatureNonce":   strconv.FormatInt(time.Now().UnixNano(), 10),
			"Format":           "JSON",
		}

		// 添加状态过滤
		if len(statusFilter) > 0 {
			for i, status := range statusFilter {
				params[fmt.Sprintf("Status.%d", i+1)] = status
			}
		}

		signature := p.calculateSignature(params, accessKeySecret, "GET")
		params["Signature"] = signature

		url := fmt.Sprintf("https://ecs.%s.aliyuncs.com/?%s", regionID, p.buildQueryString(params))

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			return nil, err
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		if code, exists := result["Code"]; exists {
			return nil, fmt.Errorf("API错误: %s - %s", code, result["Message"])
		}

		// 解析实例数据
		if instancesData, exists := result["Instances"]; exists {
			if instanceList, ok := instancesData.(map[string]interface{}); ok {
				if instanceArray, ok := instanceList["Instance"].([]interface{}); ok {
					for _, instanceItem := range instanceArray {
						if instance, ok := instanceItem.(map[string]interface{}); ok {
							record := p.normalizeECSInstance(instance, regionID, includeTags, includeSecurityGroups)
							allInstances = append(allInstances, record)

							// 检查是否达到最大实例数限制
							if maxInstances > 0 && len(allInstances) >= maxInstances {
								return allInstances, nil
							}
						}
					}
				}
			}
		}

		// 检查是否还有更多页面
		totalCount := 0
		if tc, exists := result["TotalCount"]; exists {
			if tcFloat, ok := tc.(float64); ok {
				totalCount = int(tcFloat)
			}
		}

		if pageNumber*pageSize >= totalCount {
			break
		}

		pageNumber++
	}

	return allInstances, nil
}

// normalizeECSInstance 标准化ECS实例数据
func (p *AliyunECSProvider) normalizeECSInstance(instance map[string]interface{}, regionID string, includeTags, includeSecurityGroups bool) map[string]interface{} {
	record := make(map[string]interface{})

	// 基本信息
	if instanceId, exists := instance["InstanceId"]; exists {
		record["instance_id"] = instanceId
	}
	if instanceName, exists := instance["InstanceName"]; exists {
		record["instance_name"] = instanceName
	}
	if hostname, exists := instance["HostName"]; exists {
		record["hostname"] = hostname
	}
	if instanceType, exists := instance["InstanceType"]; exists {
		record["instance_type"] = instanceType
	}

	// CPU和内存信息
	if cpu, exists := instance["Cpu"]; exists {
		if cpuFloat, ok := cpu.(float64); ok {
			record["cpu_cores"] = int(cpuFloat)
		}
	}
	if memory, exists := instance["Memory"]; exists {
		if memFloat, ok := memory.(float64); ok {
			record["memory_mb"] = int(memFloat * 1024) // 转换为MB
		}
	}

	// 操作系统信息
	if osName, exists := instance["OSName"]; exists {
		record["os_name"] = osName
	}
	if osType, exists := instance["OSType"]; exists {
		record["os_type"] = strings.ToLower(osType.(string))
	}

	// 网络信息
	if publicIpAddress, exists := instance["PublicIpAddress"]; exists {
		if ipList, ok := publicIpAddress.(map[string]interface{}); ok {
			if ipArray, ok := ipList["IpAddress"].([]interface{}); ok && len(ipArray) > 0 {
				record["public_ip"] = ipArray[0]
			}
		}
	}

	if privateIpAddress, exists := instance["InnerIpAddress"]; exists {
		if ipList, ok := privateIpAddress.(map[string]interface{}); ok {
			if ipArray, ok := ipList["IpAddress"].([]interface{}); ok && len(ipArray) > 0 {
				record["private_ip"] = ipArray[0]
			}
		}
	}

	if vpcAttributes, exists := instance["VpcAttributes"]; exists {
		if vpc, ok := vpcAttributes.(map[string]interface{}); ok {
			if vpcId, exists := vpc["VpcId"]; exists {
				record["vpc_id"] = vpcId
			}
			if vswitchId, exists := vpc["VSwitchId"]; exists {
				record["vswitch_id"] = vswitchId
			}
			if privateIp, exists := vpc["PrivateIpAddress"]; exists {
				if ipList, ok := privateIp.(map[string]interface{}); ok {
					if ipArray, ok := ipList["IpAddress"].([]interface{}); ok && len(ipArray) > 0 {
						record["private_ip"] = ipArray[0]
					}
				}
			}
		}
	}

	// 安全组信息
	if includeSecurityGroups {
		if securityGroups, exists := instance["SecurityGroupIds"]; exists {
			if sgList, ok := securityGroups.(map[string]interface{}); ok {
				if sgArray, ok := sgList["SecurityGroupId"].([]interface{}); ok {
					record["security_groups"] = sgArray
				}
			}
		}
	}

	// 地域和可用区
	record["region_id"] = regionID
	if zoneId, exists := instance["ZoneId"]; exists {
		record["zone_id"] = zoneId
	}

	// 状态信息
	if status, exists := instance["Status"]; exists {
		record["status"] = status
	}

	// 时间信息
	if creationTime, exists := instance["CreationTime"]; exists {
		record["creation_time"] = creationTime
	}
	if expiredTime, exists := instance["ExpiredTime"]; exists {
		record["expired_time"] = expiredTime
	}

	// 计费信息
	if chargeType, exists := instance["InstanceChargeType"]; exists {
		record["instance_charge_type"] = chargeType
	}

	// 网络类型
	if networkType, exists := instance["InstanceNetworkType"]; exists {
		record["network_type"] = networkType
	}
	if internetChargeType, exists := instance["InternetChargeType"]; exists {
		record["internet_charge_type"] = internetChargeType
	}
	if internetMaxBandwidthOut, exists := instance["InternetMaxBandwidthOut"]; exists {
		if bandwidthFloat, ok := internetMaxBandwidthOut.(float64); ok {
			record["internet_max_bandwidth_out"] = int(bandwidthFloat)
		}
	}

	// 标签信息
	if includeTags {
		if tags, exists := instance["Tags"]; exists {
			if tagList, ok := tags.(map[string]interface{}); ok {
				if tagArray, ok := tagList["Tag"].([]interface{}); ok {
					tagMap := make(map[string]interface{})
					for _, tagItem := range tagArray {
						if tag, ok := tagItem.(map[string]interface{}); ok {
							if key, keyExists := tag["TagKey"]; keyExists {
								if value, valueExists := tag["TagValue"]; valueExists {
									tagMap[key.(string)] = value
								}
							}
						}
					}
					record["tags"] = tagMap
				}
			}
		}
	}

	return record
}

// calculateSignature 计算阿里云API签名
func (p *AliyunECSProvider) calculateSignature(params map[string]string, accessKeySecret, method string) string {
	// 删除Signature参数
	delete(params, "Signature")

	// 排序参数
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 构建查询字符串
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(params[k])))
	}
	queryString := strings.Join(parts, "&")

	// 构建签名字符串
	stringToSign := fmt.Sprintf("%s&%s&%s", method, url.QueryEscape("/"), url.QueryEscape(queryString))

	// 计算HMAC-SHA1签名
	h := hmac.New(sha1.New, []byte(accessKeySecret+"&"))
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return signature
}

// buildQueryString 构建查询字符串
func (p *AliyunECSProvider) buildQueryString(params map[string]string) string {
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(params[k])))
	}

	return strings.Join(parts, "&")
}

func (p *AliyunECSProvider) GetFieldMapping(targetSchema string) (*FieldMappingConfig, error) {
	mappings := make(map[string][]FieldMapping)

	// 通用映射规则
	commonMappings := []FieldMapping{
		{
			SourceField: "instance_name",
			TargetField: "device_name",
			Transform:   "direct",
		},
		{
			SourceField: "hostname",
			TargetField: "hostname",
			Transform:   "direct",
		},
		{
			SourceField: "public_ip",
			TargetField: "public_ip",
			Transform:   "direct",
		},
		{
			SourceField: "private_ip",
			TargetField: "private_ip",
			Transform:   "direct",
		},
		{
			SourceField: "os_type",
			TargetField: "os_family",
			Transform:   "direct",
		},
		{
			SourceField: "os_name",
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
				SourceField: "instance_id",
				TargetField: "device_id",
				Transform:   "direct",
			},
			{
				SourceField: "instance_type",
				TargetField: "device_model",
				Transform:   "direct",
			},
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
				SourceField: "region_id",
				TargetField: "location",
				Transform:   "direct",
			},
		}...)
	case "cloud_instance":
		mappings["cloud_instance"] = []FieldMapping{
			{
				SourceField: "instance_id",
				TargetField: "instance_id",
				Transform:   "direct",
			},
			{
				SourceField: "instance_name",
				TargetField: "instance_name",
				Transform:   "direct",
			},
			{
				SourceField: "instance_type",
				TargetField: "instance_type",
				Transform:   "direct",
			},
			{
				SourceField: "region_id",
				TargetField: "region",
				Transform:   "direct",
			},
			{
				SourceField: "zone_id",
				TargetField: "availability_zone",
				Transform:   "direct",
			},
			{
				SourceField: "vpc_id",
				TargetField: "vpc_id",
				Transform:   "direct",
			},
			{
				SourceField: "instance_charge_type",
				TargetField: "billing_mode",
				Transform:   "direct",
			},
		}
	default:
		mappings["default"] = commonMappings
	}

	return &FieldMappingConfig{
		Version:     "1.0",
		Mappings:    mappings,
		Transforms:  p.getAvailableTransforms(),
		Description: "阿里云ECS实例发现结果字段映射配置",
	}, nil
}

func (p *AliyunECSProvider) getAvailableTransforms() map[string]TransformDefinition {
	return map[string]TransformDefinition{
		"direct": {
			Name:        "direct",
			Description: "直接映射，不做任何转换",
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
		"prefix": {
			Name:        "prefix",
			Description: "添加前缀",
			Parameters:  []string{"prefix"},
		},
		"suffix": {
			Name:        "suffix",
			Description: "添加后缀",
			Parameters:  []string{"suffix"},
		},
	}
}