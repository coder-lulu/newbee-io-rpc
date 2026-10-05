package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/mixins"
)

// ConfigAuditLog 配置变更审计日志
type ConfigAuditLog struct {
	ent.Schema
}

// Mixin 定义Schema的Mixin
func (ConfigAuditLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.IDMixin{},         // 提供id, created_at, updated_at
		mixins.TenantMixin{},     // 提供tenant_id
		mixins.DepartmentMixin{}, // 提供department_id
	}
}

// Fields 定义Schema的字段
func (ConfigAuditLog) Fields() []ent.Field {
	return []ent.Field{
		// 基础信息
		field.String("config_key").
			Comment("配置键").
			NotEmpty().
			MaxLen(255),

		field.String("old_value").
			Comment("旧值").
			Optional().
			MaxLen(4096),

		field.String("new_value").
			Comment("新值").
			Optional().
			MaxLen(4096),

		field.String("change_type").
			Comment("变更类型: create/update/delete").
			Default("update").
			MaxLen(20),

		// 变更人信息
		field.Uint64("changed_by").
			Comment("变更人用户ID").
			Optional(),

		field.String("changed_by_name").
			Comment("变更人用户名").
			Optional().
			MaxLen(100),

		// 服务信息
		field.String("service_name").
			Comment("所属服务名称").
			Optional().
			MaxLen(100),

		field.String("category").
			Comment("配置分类").
			Optional().
			MaxLen(50),

		field.String("config_group").
			Comment("配置组").
			Optional().
			MaxLen(100),

		// 变更详情
		field.String("change_reason").
			Comment("变更原因").
			Optional().
			MaxLen(500),

		field.String("ip_address").
			Comment("操作IP地址").
			Optional().
			MaxLen(50),

		field.String("user_agent").
			Comment("用户代理").
			Optional().
			MaxLen(500),

		// 版本信息
		field.Int("old_version").
			Comment("旧版本号").
			Optional().
			Default(0),

		field.Int("new_version").
			Comment("新版本号").
			Optional().
			Default(1),

		// 状态信息
		field.Bool("is_rollback").
			Comment("是否为回滚操作").
			Default(false),

		field.String("rollback_from_log_id").
			Comment("回滚来源日志ID").
			Optional().
			MaxLen(100),
	}
}

// Indexes 定义Schema的索引
func (ConfigAuditLog) Indexes() []ent.Index {
	return []ent.Index{
		// 按租户和配置键查询历史
		index.Fields("tenant_id", "config_key"),

		// 按时间查询
		index.Fields("created_at"),

		// 按变更人查询
		index.Fields("changed_by"),

		// 按变更类型查询
		index.Fields("change_type"),

		// 按服务查询
		index.Fields("service_name", "category"),

		// 复合索引：租户 + 配置键 + 时间（最常用查询）
		index.Fields("tenant_id", "config_key", "created_at"),
	}
}
