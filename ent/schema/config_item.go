package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/mixins"
)

// ConfigItem 配置项 - 统一配置中心
//
// 设计目标:
// 1. 统一管理所有服务配置（替代分散在代码中的硬编码配置）
// 2. 支持配置热重载（通过Redis Pub/Sub）
// 3. 支持配置版本管理和回滚
// 4. 支持多租户隔离
// 5. 三级缓存：内存(1µs) → Redis(1ms) → MySQL(10ms)
//
// 配置分类:
// - system: 系统级配置（跨租户共享）
// - tenant: 租户级配置
// - service: 服务级配置
//
// 配置值类型:
// - string: 字符串
// - int: 整数
// - float: 浮点数
// - bool: 布尔值
// - json: JSON对象/数组
//
// 示例配置:
// - worker.pull_interval = "5s"
// - worker.batch_size = "10"
// - worker.max_concurrent = "5"
// - cron.default_timeout = "30m"
type ConfigItem struct {
	ent.Schema
}

// Mixin 混入字段
func (ConfigItem) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.IDMixin{},         // ID, 雪花ID
		mixins.StatusMixin{},     // status (1=启用, 2=禁用)
		mixins.TenantMixin{},     // tenant_id
		mixins.DepartmentMixin{}, // department_id (可选)
	}
}

// Fields 字段定义
func (ConfigItem) Fields() []ent.Field {
	return []ent.Field{
		// 配置键（唯一标识）
		field.String("config_key").
			Comment("配置键，格式：service.module.key，例如：worker.pull_interval").
			NotEmpty().
			MaxLen(255),

		// 配置值
		field.String("config_value").
			Comment("配置值，根据value_type解析").
			NotEmpty().
			MaxLen(4096), // 支持较长的JSON配置

		// 值类型
		field.String("value_type").
			Comment("值类型：string/int/float/bool/json").
			Default("string").
			MaxLen(20),

		// 配置分类
		field.String("category").
			Comment("配置分类：system/tenant/service").
			Default("service").
			MaxLen(50),

		// 服务名称
		field.String("service_name").
			Comment("所属服务：unified-io/core/cmdb等").
			Optional().
			MaxLen(100),

		// 配置描述
		field.String("description").
			Comment("配置说明").
			Optional().
			MaxLen(500),

		// 默认值
		field.String("default_value").
			Comment("默认值（用于回滚）").
			Optional().
			MaxLen(4096),

		// 配置版本
		field.Int("version").
			Comment("配置版本号，每次更新递增").
			Default(1).
			Positive(),

		// 最后修改人
		field.Uint64("updated_by").
			Comment("最后修改人ID").
			Optional(),

		// 是否只读
		field.Bool("is_readonly").
			Comment("是否只读（系统配置不可修改）").
			Default(false),

		// 是否敏感
		field.Bool("is_sensitive").
			Comment("是否敏感配置（加密存储）").
			Default(false),

		// 生效范围
		field.String("scope").
			Comment("生效范围：global/tenant/service").
			Default("service").
			MaxLen(50),

		// 配置组
		field.String("config_group").
			Comment("配置分组，用于批量管理").
			Optional().
			MaxLen(100),

		// 标签（JSON数组）
		field.String("tags").
			Comment("标签，JSON数组，例如：[\"performance\", \"critical\"]").
			Optional().
			MaxLen(500),
	}
}

// Edges 关联关系
func (ConfigItem) Edges() []ent.Edge {
	return []ent.Edge{
		// 可以添加与用户、部门的关联
	}
}

// Indexes 索引定义
func (ConfigItem) Indexes() []ent.Index {
	return []ent.Index{
		// 唯一索引：租户+配置键
		index.Fields("tenant_id", "config_key").
			Unique(),

		// 查询索引：分类+服务
		index.Fields("category", "service_name"),

		// 查询索引：配置组
		index.Fields("config_group"),

		// 查询索引：状态
		index.Fields("status"),
	}
}

// Annotations 表注解
func (ConfigItem) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "config_items",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_unicode_ci",
		},
		schema.Comment("配置项表 - 统一配置中心"),
	}
}
