package base

import (
	"context"
	"encoding/json"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/ent/discoveryproviderschema"
	"github.com/coder-lulu/newbee-common/v2/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

func convertJSONBytesToInterfaces(jsonBytes []byte) []interface{} {
	var result []interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil
	}
	return result
}

func (l *InitDatabaseLogic) insertDiscoveryProviderData(ctx context.Context, tenantID uint64) error {
	// department_id=0 表示租户级别数据，所有部门共享
	departmentID := uint64(0)

	existing, err := l.svcCtx.DB.DiscoveryProviderSchema.Query().
		Where(
			discoveryproviderschema.TenantIDEQ(tenantID),
			discoveryproviderschema.DepartmentIDEQ(departmentID),
		).
		Count(ctx)
	if err != nil {
		return errors.InternalWithCause("Discovery provider data initialization failed", err)
	}
	if existing > 0 {
		logx.Infow("Discovery Provider Schema已存在，跳过创建",
			logx.Field("tenant_id", tenantID),
			logx.Field("department_id", departmentID))
		return nil
	}

	var providers []*ent.DiscoveryProviderSchemaCreate

	nbAgentParams := []map[string]interface{}{
		{"name": "agent_endpoint", "label": "Agent地址", "type": "string", "required": true, "placeholder": "http://192.168.1.100:8888", "description": "NewBee Agent的HTTP服务地址"},
		{"name": "auth_token", "label": "认证Token", "type": "password", "required": true, "placeholder": "输入Agent认证Token", "description": "Agent API访问令牌"},
		{"name": "timeout", "label": "超时时间(秒)", "type": "number", "required": false, "defaultValue": 30, "description": "连接超时时间"},
	}
	nbAgentFields := []map[string]interface{}{
		{"name": "hostname", "label": "主机名", "dataType": "string", "required": true, "description": "服务器主机名", "example": "server-001"},
		{"name": "ip_address", "label": "IP地址", "dataType": "string", "required": true, "description": "主要IP地址", "example": "192.168.1.100"},
		{"name": "os_type", "label": "操作系统类型", "dataType": "string", "required": true, "description": "操作系统类型", "example": "Linux"},
		{"name": "os_version", "label": "系统版本", "dataType": "string", "required": true, "description": "操作系统版本", "example": "Ubuntu 20.04"},
		{"name": "cpu_model", "label": "CPU型号", "dataType": "string", "required": false, "description": "处理器型号", "example": "Intel Xeon E5-2680"},
		{"name": "cpu_cores", "label": "CPU核心数", "dataType": "integer", "required": false, "description": "CPU逻辑核心数", "example": "16"},
		{"name": "memory_total", "label": "总内存(GB)", "dataType": "float", "required": false, "description": "物理内存总量", "example": "32.0"},
		{"name": "disk_total", "label": "总磁盘(GB)", "dataType": "float", "required": false, "description": "磁盘总容量", "example": "500.0"},
		{"name": "serial_number", "label": "序列号", "dataType": "string", "required": false, "description": "硬件序列号", "example": "SN123456789"},
		{"name": "manufacturer", "label": "制造商", "dataType": "string", "required": false, "description": "服务器制造商", "example": "Dell"},
		{"name": "model", "label": "型号", "dataType": "string", "required": false, "description": "服务器型号", "example": "PowerEdge R730"},
	}
	nbAgentParamJSON, _ := json.Marshal(nbAgentParams)
	nbAgentFieldJSON, _ := json.Marshal(nbAgentFields)

	providers = append(providers, l.svcCtx.DB.DiscoveryProviderSchema.Create().
		SetTenantID(tenantID).
			SetDepartmentID(departmentID).
		SetProviderID("nb_agent").
		SetProviderName("NewBee Agent").
		SetCategory(discoveryproviderschema.CategoryAgent).
		SetParameterSchema(convertJSONBytesToInterfaces(nbAgentParamJSON)).
		SetFieldSchema(convertJSONBytesToInterfaces(nbAgentFieldJSON)).
		SetDescription("NewBee自研Agent自动发现，支持Linux/Windows服务器信息采集").
		SetVersion("1.0.0").
		SetIconURL("/assets/icons/nb-agent.svg").
		SetIsBuiltin(true).
		SetExecutionMode(discoveryproviderschema.ExecutionModeDirect).
		SetIsActive(true))

	vmwareParams := []map[string]interface{}{
		{"name": "vcenter_host", "label": "vCenter地址", "type": "string", "required": true, "placeholder": "vcenter.example.com"},
		{"name": "vcenter_port", "label": "端口", "type": "number", "required": false, "defaultValue": 443},
		{"name": "username", "label": "用户名", "type": "string", "required": true, "placeholder": "administrator@vsphere.local"},
		{"name": "password", "label": "密码", "type": "password", "required": true},
		{"name": "ignore_ssl", "label": "忽略SSL证书", "type": "boolean", "required": false, "defaultValue": false},
	}
	vmwareFields := []map[string]interface{}{
		{"name": "vm_name", "label": "虚拟机名称", "dataType": "string", "required": true},
		{"name": "vm_uuid", "label": "UUID", "dataType": "string", "required": true},
		{"name": "ip_address", "label": "IP地址", "dataType": "string", "required": true},
		{"name": "cpu_cores", "label": "CPU核心数", "dataType": "integer", "required": false},
		{"name": "memory_gb", "label": "内存(GB)", "dataType": "float", "required": false},
		{"name": "disk_gb", "label": "磁盘(GB)", "dataType": "float", "required": false},
		{"name": "power_state", "label": "电源状态", "dataType": "string", "required": false},
		{"name": "guest_os", "label": "操作系统", "dataType": "string", "required": false},
	}
	vmwareParamJSON, _ := json.Marshal(vmwareParams)
	vmwareFieldJSON, _ := json.Marshal(vmwareFields)

	providers = append(providers, l.svcCtx.DB.DiscoveryProviderSchema.Create().
		SetTenantID(tenantID).
			SetDepartmentID(departmentID).
		SetProviderID("vmware_vcenter").
		SetProviderName("VMware vCenter").
		SetCategory(discoveryproviderschema.CategorySdk).
		SetParameterSchema(convertJSONBytesToInterfaces(vmwareParamJSON)).
		SetFieldSchema(convertJSONBytesToInterfaces(vmwareFieldJSON)).
		SetDescription("通过VMware vCenter API发现虚拟机资源").
		SetVersion("1.0.0").
		SetIconURL("/assets/icons/vmware.svg").
		SetIsBuiltin(true).
		SetExecutionMode(discoveryproviderschema.ExecutionModeWorker).
		SetIsActive(true))

	linuxParams := []map[string]interface{}{
		{"name": "ssh_host", "label": "主机地址", "type": "string", "required": true, "placeholder": "192.168.1.100"},
		{"name": "ssh_port", "label": "SSH端口", "type": "number", "required": false, "defaultValue": 22},
		{"name": "username", "label": "用户名", "type": "string", "required": true, "defaultValue": "root"},
		{"name": "auth_method", "label": "认证方式", "type": "select", "required": true, "defaultValue": "password", "options": []string{"password", "key"}},
		{"name": "password", "label": "密码", "type": "password", "required": false},
		{"name": "private_key", "label": "私钥", "type": "textarea", "required": false},
	}
	linuxFields := []map[string]interface{}{
		{"name": "hostname", "label": "主机名", "dataType": "string", "required": true},
		{"name": "ip_address", "label": "IP地址", "dataType": "string", "required": true},
		{"name": "os_type", "label": "操作系统", "dataType": "string", "required": true},
		{"name": "os_version", "label": "系统版本", "dataType": "string", "required": true},
		{"name": "kernel_version", "label": "内核版本", "dataType": "string", "required": false},
		{"name": "cpu_model", "label": "CPU型号", "dataType": "string", "required": false},
		{"name": "cpu_cores", "label": "CPU核心数", "dataType": "integer", "required": false},
		{"name": "memory_total", "label": "总内存(GB)", "dataType": "float", "required": false},
	}
	linuxParamJSON, _ := json.Marshal(linuxParams)
	linuxFieldJSON, _ := json.Marshal(linuxFields)

	providers = append(providers, l.svcCtx.DB.DiscoveryProviderSchema.Create().
		SetTenantID(tenantID).
			SetDepartmentID(departmentID).
		SetProviderID("linux_ssh").
		SetProviderName("Linux SSH").
		SetCategory(discoveryproviderschema.CategorySdk).
		SetParameterSchema(convertJSONBytesToInterfaces(linuxParamJSON)).
		SetFieldSchema(convertJSONBytesToInterfaces(linuxFieldJSON)).
		SetDescription("通过SSH连接Linux服务器采集信息").
		SetVersion("1.0.0").
		SetIconURL("/assets/icons/linux.svg").
		SetIsBuiltin(true).
		SetExecutionMode(discoveryproviderschema.ExecutionModeWorker).
		SetIsActive(true))

	aliyunParams := []map[string]interface{}{
		{"name": "access_key_id", "label": "AccessKey ID", "type": "string", "required": true},
		{"name": "access_key_secret", "label": "AccessKey Secret", "type": "password", "required": true},
		{"name": "region_id", "label": "区域", "type": "select", "required": true, "defaultValue": "cn-hangzhou", "options": []string{"cn-hangzhou", "cn-shanghai", "cn-beijing", "cn-shenzhen"}},
	}
	aliyunFields := []map[string]interface{}{
		{"name": "instance_id", "label": "实例ID", "dataType": "string", "required": true},
		{"name": "instance_name", "label": "实例名称", "dataType": "string", "required": true},
		{"name": "ip_address", "label": "公网IP", "dataType": "string", "required": false},
		{"name": "private_ip", "label": "私网IP", "dataType": "string", "required": false},
		{"name": "cpu_cores", "label": "CPU核心数", "dataType": "integer", "required": false},
		{"name": "memory_gb", "label": "内存(GB)", "dataType": "float", "required": false},
		{"name": "instance_type", "label": "实例规格", "dataType": "string", "required": false},
		{"name": "os_type", "label": "操作系统", "dataType": "string", "required": false},
	}
	aliyunParamJSON, _ := json.Marshal(aliyunParams)
	aliyunFieldJSON, _ := json.Marshal(aliyunFields)

	providers = append(providers, l.svcCtx.DB.DiscoveryProviderSchema.Create().
		SetTenantID(tenantID).
			SetDepartmentID(departmentID).
		SetProviderID("aliyun_ecs").
		SetProviderName("阿里云ECS").
		SetCategory(discoveryproviderschema.CategoryAPI).
		SetParameterSchema(convertJSONBytesToInterfaces(aliyunParamJSON)).
		SetFieldSchema(convertJSONBytesToInterfaces(aliyunFieldJSON)).
		SetDescription("通过阿里云API自动发现ECS实例").
		SetVersion("1.0.0").
		SetIconURL("/assets/icons/aliyun.svg").
		SetIsBuiltin(true).
		SetExecutionMode(discoveryproviderschema.ExecutionModeWorker).
		SetIsActive(true))

	tencentParams := []map[string]interface{}{
		{"name": "secret_id", "label": "SecretId", "type": "string", "required": true},
		{"name": "secret_key", "label": "SecretKey", "type": "password", "required": true},
		{"name": "region", "label": "区域", "type": "select", "required": true, "defaultValue": "ap-guangzhou", "options": []string{"ap-guangzhou", "ap-shanghai", "ap-beijing"}},
	}
	tencentFields := []map[string]interface{}{
		{"name": "instance_id", "label": "实例ID", "dataType": "string", "required": true},
		{"name": "instance_name", "label": "实例名称", "dataType": "string", "required": true},
		{"name": "public_ip", "label": "公网IP", "dataType": "string", "required": false},
		{"name": "private_ip", "label": "私网IP", "dataType": "string", "required": false},
		{"name": "cpu_cores", "label": "CPU核心数", "dataType": "integer", "required": false},
		{"name": "memory_gb", "label": "内存(GB)", "dataType": "float", "required": false},
	}
	tencentParamJSON, _ := json.Marshal(tencentParams)
	tencentFieldJSON, _ := json.Marshal(tencentFields)

	providers = append(providers, l.svcCtx.DB.DiscoveryProviderSchema.Create().
		SetTenantID(tenantID).
			SetDepartmentID(departmentID).
		SetProviderID("tencent_cvm").
		SetProviderName("腾讯云CVM").
		SetCategory(discoveryproviderschema.CategoryAPI).
		SetParameterSchema(convertJSONBytesToInterfaces(tencentParamJSON)).
		SetFieldSchema(convertJSONBytesToInterfaces(tencentFieldJSON)).
		SetDescription("通过腾讯云API自动发现CVM实例").
		SetVersion("1.0.0").
		SetIconURL("/assets/icons/tencent.svg").
		SetIsBuiltin(true).
		SetExecutionMode(discoveryproviderschema.ExecutionModeWorker).
		SetIsActive(true))

	fileParams := []map[string]interface{}{
		{"name": "file_format", "label": "文件格式", "type": "select", "required": true, "defaultValue": "excel", "options": []string{"excel", "csv", "json"}},
		{"name": "file_path", "label": "文件路径", "type": "file", "required": true, "description": "上传Excel/CSV/JSON文件"},
		{"name": "mapping_config", "label": "字段映射", "type": "json", "required": true, "description": "文件列与CI属性的映射关系"},
	}
	fileFields := []map[string]interface{}{
		{"name": "dynamic_field_1", "label": "动态字段1", "dataType": "string", "required": false, "description": "根据文件内容动态识别"},
		{"name": "dynamic_field_2", "label": "动态字段2", "dataType": "string", "required": false, "description": "根据文件内容动态识别"},
	}
	fileParamJSON, _ := json.Marshal(fileParams)
	fileFieldJSON, _ := json.Marshal(fileFields)

	providers = append(providers, l.svcCtx.DB.DiscoveryProviderSchema.Create().
		SetTenantID(tenantID).
			SetDepartmentID(departmentID).
		SetProviderID("file_import").
		SetProviderName("文件导入").
		SetCategory(discoveryproviderschema.CategoryFile).
		SetParameterSchema(convertJSONBytesToInterfaces(fileParamJSON)).
		SetFieldSchema(convertJSONBytesToInterfaces(fileFieldJSON)).
		SetDescription("通过Excel/CSV/JSON文件批量导入资产信息").
		SetVersion("1.0.0").
		SetIconURL("/assets/icons/file.svg").
		SetIsBuiltin(true).
		SetExecutionMode(discoveryproviderschema.ExecutionModeDirect).
		SetIsActive(true))

	providers = append(providers,
		l.svcCtx.DB.DiscoveryProviderSchema.Create().
			SetTenantID(tenantID).
			SetDepartmentID(departmentID).
			SetProviderID("vmware_esxi").
			SetProviderName("VMware ESXi").
			SetCategory(discoveryproviderschema.CategorySdk).
			SetParameterSchema(convertJSONBytesToInterfaces(vmwareParamJSON)).
			SetFieldSchema(convertJSONBytesToInterfaces(vmwareFieldJSON)).
			SetDescription("直接连接ESXi主机发现虚拟机").
			SetVersion("1.0.0").
			SetIsBuiltin(true).
			SetExecutionMode(discoveryproviderschema.ExecutionModeWorker).
			SetIsActive(false),

		l.svcCtx.DB.DiscoveryProviderSchema.Create().
			SetTenantID(tenantID).
			SetDepartmentID(departmentID).
			SetProviderID("windows_wmi").
			SetProviderName("Windows WMI").
			SetCategory(discoveryproviderschema.CategorySdk).
			SetParameterSchema(convertJSONBytesToInterfaces(linuxParamJSON)).
			SetFieldSchema(convertJSONBytesToInterfaces(linuxFieldJSON)).
			SetDescription("通过WMI协议采集Windows服务器信息").
			SetVersion("1.0.0").
			SetIsBuiltin(true).
			SetExecutionMode(discoveryproviderschema.ExecutionModeWorker).
			SetIsActive(false),

		l.svcCtx.DB.DiscoveryProviderSchema.Create().
			SetTenantID(tenantID).
			SetDepartmentID(departmentID).
			SetProviderID("aws_ec2").
			SetProviderName("AWS EC2").
			SetCategory(discoveryproviderschema.CategoryAPI).
			SetParameterSchema(convertJSONBytesToInterfaces(aliyunParamJSON)).
			SetFieldSchema(convertJSONBytesToInterfaces(aliyunFieldJSON)).
			SetDescription("通过AWS API发现EC2实例").
			SetVersion("1.0.0").
			SetIsBuiltin(true).
			SetExecutionMode(discoveryproviderschema.ExecutionModeWorker).
			SetIsActive(false),

		l.svcCtx.DB.DiscoveryProviderSchema.Create().
			SetTenantID(tenantID).
			SetDepartmentID(departmentID).
			SetProviderID("huawei_ecs").
			SetProviderName("华为云ECS").
			SetCategory(discoveryproviderschema.CategoryAPI).
			SetParameterSchema(convertJSONBytesToInterfaces(aliyunParamJSON)).
			SetFieldSchema(convertJSONBytesToInterfaces(aliyunFieldJSON)).
			SetDescription("通过华为云API发现ECS实例").
			SetVersion("1.0.0").
			SetIsBuiltin(true).
			SetExecutionMode(discoveryproviderschema.ExecutionModeWorker).
			SetIsActive(false),

		l.svcCtx.DB.DiscoveryProviderSchema.Create().
			SetTenantID(tenantID).
			SetDepartmentID(departmentID).
			SetProviderID("azure_vm").
			SetProviderName("Azure虚拟机").
			SetCategory(discoveryproviderschema.CategoryAPI).
			SetParameterSchema(convertJSONBytesToInterfaces(aliyunParamJSON)).
			SetFieldSchema(convertJSONBytesToInterfaces(aliyunFieldJSON)).
			SetDescription("通过Azure API发现虚拟机").
			SetVersion("1.0.0").
			SetIsBuiltin(true).
			SetExecutionMode(discoveryproviderschema.ExecutionModeWorker).
			SetIsActive(false),

		l.svcCtx.DB.DiscoveryProviderSchema.Create().
			SetTenantID(tenantID).
			SetDepartmentID(departmentID).
			SetProviderID("openstack").
			SetProviderName("OpenStack").
			SetCategory(discoveryproviderschema.CategoryAPI).
			SetParameterSchema(convertJSONBytesToInterfaces(aliyunParamJSON)).
			SetFieldSchema(convertJSONBytesToInterfaces(aliyunFieldJSON)).
			SetDescription("通过OpenStack API发现虚拟机").
			SetVersion("1.0.0").
			SetIsBuiltin(true).
			SetExecutionMode(discoveryproviderschema.ExecutionModeWorker).
			SetIsActive(false),

		l.svcCtx.DB.DiscoveryProviderSchema.Create().
			SetTenantID(tenantID).
			SetDepartmentID(departmentID).
			SetProviderID("snmp_device").
			SetProviderName("SNMP网络设备").
			SetCategory(discoveryproviderschema.CategorySdk).
			SetParameterSchema(convertJSONBytesToInterfaces(linuxParamJSON)).
			SetFieldSchema(convertJSONBytesToInterfaces(linuxFieldJSON)).
			SetDescription("通过SNMP协议发现网络设备").
			SetVersion("1.0.0").
			SetIsBuiltin(true).
			SetExecutionMode(discoveryproviderschema.ExecutionModeWorker).
			SetIsActive(false),

		l.svcCtx.DB.DiscoveryProviderSchema.Create().
			SetTenantID(tenantID).
			SetDepartmentID(departmentID).
			SetProviderID("ssh_network").
			SetProviderName("SSH网络设备").
			SetCategory(discoveryproviderschema.CategorySdk).
			SetParameterSchema(convertJSONBytesToInterfaces(linuxParamJSON)).
			SetFieldSchema(convertJSONBytesToInterfaces(linuxFieldJSON)).
			SetDescription("通过SSH连接网络设备采集信息").
			SetVersion("1.0.0").
			SetIsBuiltin(true).
			SetExecutionMode(discoveryproviderschema.ExecutionModeWorker).
			SetIsActive(false),

		l.svcCtx.DB.DiscoveryProviderSchema.Create().
			SetTenantID(tenantID).
			SetDepartmentID(departmentID).
			SetProviderID("api_custom").
			SetProviderName("自定义API接口").
			SetCategory(discoveryproviderschema.CategoryAPI).
			SetParameterSchema(convertJSONBytesToInterfaces(fileParamJSON)).
			SetFieldSchema(convertJSONBytesToInterfaces(fileFieldJSON)).
			SetDescription("通过自定义API接口获取资产数据").
			SetVersion("1.0.0").
			SetIsBuiltin(true).
			SetExecutionMode(discoveryproviderschema.ExecutionModeDirect).
			SetIsActive(false),
	)

	err = l.svcCtx.DB.DiscoveryProviderSchema.CreateBulk(providers...).Exec(ctx)
	if err != nil {
		return errors.InternalWithCause("Discovery provider data initialization failed", err)
	}

	logx.Info("Discovery Provider Schema初始化成功，共创建15个内置Provider")
	return nil
}
