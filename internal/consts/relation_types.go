package consts

// 标准化关系类型常量
// 基于veops CMDB和ITIL标准定义的核心关系类型
const (
	// RelationContain 包含关系 - 表示层级包含，如部门包含产品线
	RelationContain = "contain"
	
	// RelationDeploy 部署关系 - 表示软件部署，如应用部署在虚拟机
	RelationDeploy = "deploy"
	
	// RelationInstall 安装关系 - 表示软件安装，如MySQL安装在物理机
	RelationInstall = "install"
	
	// RelationConnect 连接关系 - 表示物理或逻辑连接，如交换机连接路由器
	RelationConnect = "connect"
	
	// RelationDependsOn 依赖关系 - 表示服务依赖，如应用依赖数据库
	RelationDependsOn = "depends_on"
	
	// RelationHostedOn 宿主关系 - 表示运行环境，如虚拟机运行在物理机
	RelationHostedOn = "hosted_on"
	
	// RelationCompose 组成关系 - 表示集群组成，如集群由多个节点组成
	RelationCompose = "compose"
	
	// RelationManage 管理关系 - 表示管理控制，如负载均衡管理服务器
	RelationManage = "manage"
)

// 关系类型分类常量
const (
	// CategoryPhysical 物理关系 - 表示物理层面的连接和依赖
	CategoryPhysical = "physical"
	
	// CategoryLogical 逻辑关系 - 表示逻辑层面的关联和依赖
	CategoryLogical = "logical"
	
	// CategoryBusiness 业务关系 - 表示业务层面的组织和管理
	CategoryBusiness = "business"
)

// 关系方向常量
const (
	// DirectionUnidirectional 单向关系 - 关系只能从源到目标
	DirectionUnidirectional = "unidirectional"
	
	// DirectionBidirectional 双向关系 - 关系可以双向建立
	DirectionBidirectional = "bidirectional"
)

// StandardRelationType 标准关系类型定义
type StandardRelationType struct {
	Code        string // 关系代码
	Name        string // 关系名称
	Category    string // 分类
	Direction   string // 方向
	Description string // 描述
}

// StandardRelationTypes 预设的标准关系类型列表
var StandardRelationTypes = []StandardRelationType{
	{
		Code:        RelationContain,
		Name:        "包含",
		Category:    CategoryBusiness,
		Direction:   DirectionUnidirectional,
		Description: "表示层级包含关系，如部门包含产品线、组织包含团队",
	},
	{
		Code:        RelationDeploy,
		Name:        "部署",
		Category:    CategoryLogical,
		Direction:   DirectionUnidirectional,
		Description: "表示软件部署关系，如应用部署在虚拟机、服务部署在容器",
	},
	{
		Code:        RelationInstall,
		Name:        "安装",
		Category:    CategoryLogical,
		Direction:   DirectionUnidirectional,
		Description: "表示软件安装关系，如MySQL安装在物理机、驱动安装在服务器",
	},
	{
		Code:        RelationConnect,
		Name:        "连接",
		Category:    CategoryPhysical,
		Direction:   DirectionBidirectional,
		Description: "表示物理或逻辑连接关系，如交换机连接路由器、服务间网络连接",
	},
	{
		Code:        RelationDependsOn,
		Name:        "依赖",
		Category:    CategoryLogical,
		Direction:   DirectionUnidirectional,
		Description: "表示服务依赖关系，如应用依赖数据库、服务依赖中间件",
	},
	{
		Code:        RelationHostedOn,
		Name:        "运行于",
		Category:    CategoryPhysical,
		Direction:   DirectionUnidirectional,
		Description: "表示运行环境关系，如虚拟机运行在物理机、容器运行在节点",
	},
	{
		Code:        RelationCompose,
		Name:        "组成",
		Category:    CategoryLogical,
		Direction:   DirectionUnidirectional,
		Description: "表示集群组成关系，如集群由多个节点组成、负载均衡组包含多个后端",
	},
	{
		Code:        RelationManage,
		Name:        "管理",
		Category:    CategoryBusiness,
		Direction:   DirectionUnidirectional,
		Description: "表示管理控制关系，如负载均衡管理服务器、监控系统管理主机",
	},
}

// GetStandardRelationTypeByCode 根据代码获取标准关系类型
func GetStandardRelationTypeByCode(code string) *StandardRelationType {
	for _, rt := range StandardRelationTypes {
		if rt.Code == code {
			return &rt
		}
	}
	return nil
}

// IsStandardRelationType 检查是否为标准关系类型
func IsStandardRelationType(code string) bool {
	return GetStandardRelationTypeByCode(code) != nil
}

// GetRelationTypesByCategory 根据分类获取关系类型列表
func GetRelationTypesByCategory(category string) []StandardRelationType {
	var result []StandardRelationType
	for _, rt := range StandardRelationTypes {
		if rt.Category == category {
			result = append(result, rt)
		}
	}
	return result
}