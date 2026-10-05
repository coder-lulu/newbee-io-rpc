package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/mixins"
)

// CiChangeHistory CI变更历史记录
type CiChangeHistory struct {
	ent.Schema
}

// Mixin 定义Schema的Mixin
func (CiChangeHistory) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.IDMixin{},         // 提供id, created_at, updated_at
		mixins.TenantMixin{},     // 提供tenant_id
		mixins.DepartmentMixin{}, // 提供department_id
	}
}

// Fields 定义Schema的字段
func (CiChangeHistory) Fields() []ent.Field {
	return []ent.Field{
		// 操作ID（全局唯一标识符）
		field.String("operation_id").
			Comment("操作ID").
			NotEmpty().
			MaxLen(100),

		// CI信息
		field.Uint64("ci_id").
			Comment("CI实例ID").
			Optional(),

		field.Uint64("ci_type_id").
			Comment("CI类型ID").
			Optional(),

		field.String("ci_type").
			Comment("CI类型名称").
			Optional().
			MaxLen(100),

		// 操作信息
		field.String("operation_type").
			Comment("操作类型: create/update/delete/import/export/approve/reject").
			NotEmpty().
			MaxLen(50),

		field.String("operation_name").
			Comment("操作名称描述").
			Optional().
			MaxLen(200),

		// 操作人信息
		field.Uint64("operator_id").
			Comment("操作人用户ID").
			Optional(),

		field.String("operator_name").
			Comment("操作人用户名").
			Optional().
			MaxLen(100),

		// 变更详情
		field.Text("changed_fields").
			Comment("变更字段JSON: [{field, old_value, new_value}]").
			Optional(),

		field.Text("old_values").
			Comment("变更前完整数据JSON").
			Optional(),

		field.Text("new_values").
			Comment("变更后完整数据JSON").
			Optional(),

		field.String("change_reason").
			Comment("变更原因").
			Optional().
			MaxLen(500),

		// 数据来源
		field.String("source").
			Comment("数据来源: manual/discovery/import/api/script").
			Optional().
			MaxLen(50),

		field.String("source_detail").
			Comment("来源详情").
			Optional().
			MaxLen(500),

		field.String("source_task_id").
			Comment("来源任务ID（如果来自任务）").
			Optional().
			MaxLen(100),

		// IP和设备信息
		field.String("ip_address").
			Comment("操作IP地址").
			Optional().
			MaxLen(50),

		field.String("user_agent").
			Comment("用户代理").
			Optional().
			MaxLen(500),

		// 审批信息（如果需要审批）
		field.Bool("needs_approval").
			Comment("是否需要审批").
			Default(false),

		field.Bool("is_approved").
			Comment("是否已审批").
			Optional(),

		field.Uint64("approved_by").
			Comment("审批人用户ID").
			Optional(),

		field.String("approved_by_name").
			Comment("审批人用户名").
			Optional().
			MaxLen(100),

		field.Time("approved_at").
			Comment("审批时间").
			Optional(),

		field.String("approval_comment").
			Comment("审批意见").
			Optional().
			MaxLen(500),

		// 状态信息
		field.String("status").
			Comment("记录状态: pending/success/failed/rollback").
			Default("success").
			MaxLen(20),

		field.String("error_message").
			Comment("错误信息（如果失败）").
			Optional().
			MaxLen(1000),

		// 回滚信息
		field.Bool("is_rollback").
			Comment("是否为回滚操作").
			Default(false),

		field.Uint64("rollback_from_id").
			Comment("回滚来源记录ID").
			Optional(),

		field.Bool("can_rollback").
			Comment("是否可回滚").
			Default(true),

		// 统计信息
		field.Int("affected_count").
			Comment("影响的记录数（批量操作）").
			Default(1),

		field.Int("duration_ms").
			Comment("操作耗时（毫秒）").
			Default(0),

		// 元数据
		field.Text("metadata").
			Comment("额外元数据JSON").
			Optional(),
	}
}

// Indexes 定义Schema的索引
func (CiChangeHistory) Indexes() []ent.Index {
	return []ent.Index{
		// 按租户和CI ID查询历史
		index.Fields("tenant_id", "ci_id"),

		// 按CI类型查询
		index.Fields("ci_type_id"),

		// 按操作类型查询
		index.Fields("operation_type"),

		// 按操作人查询
		index.Fields("operator_id"),

		// 按时间查询（最常用）
		index.Fields("created_at"),

		// 按状态查询
		index.Fields("status"),

		// 复合索引：租户 + CI ID + 时间（最常用查询）
		index.Fields("tenant_id", "ci_id", "created_at"),

		// 复合索引：租户 + 操作类型 + 时间
		index.Fields("tenant_id", "operation_type", "created_at"),

		// 复合索引：操作ID查询（追溯完整操作链）
		index.Fields("operation_id"),

		// 审批查询
		index.Fields("needs_approval", "is_approved"),

		// 来源任务查询
		index.Fields("source_task_id"),
	}
}
