package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/mixins"
)

// CiLifecycleState CI生命周期状态
type CiLifecycleState struct {
	ent.Schema
}

func (CiLifecycleState) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.IDMixin{},         // id, created_at, updated_at
		mixins.TenantMixin{},     // tenant_id
		mixins.DepartmentMixin{}, // department_id
	}
}

func (CiLifecycleState) Fields() []ent.Field {
	return []ent.Field{
		// 状态ID（全局唯一标识符）
		field.String("state_id").
			Comment("状态ID").
			NotEmpty().
			MaxLen(100).
			Unique(),

		// CI信息
		field.Uint64("ci_id").
			Comment("CI实例ID").
			Optional(),

		field.Uint64("ci_type_id").
			Comment("CI类型ID").
			Optional(),

		// 状态信息
		field.String("state_name").
			Comment("状态名称").
			NotEmpty().
			MaxLen(100),

		field.String("state_type").
			Comment("状态类型: draft/submitted/validated/approved/executed/completed/cancelled/expired").
			NotEmpty().
			MaxLen(50),

		field.String("previous_state").
			Comment("上一个状态").
			Optional().
			MaxLen(50),

		// 时间信息
		field.Time("entered_at").
			Comment("进入该状态的时间").
			Immutable(),

		field.Time("exited_at").
			Comment("退出该状态的时间").
			Optional(),

		field.Time("expected_exit_at").
			Comment("预期退出时间（用于超时检测）").
			Optional(),

		field.Int("duration_seconds").
			Comment("在该状态停留的秒数").
			Default(0),

		// 触发信息
		field.String("trigger_type").
			Comment("触发类型: manual/auto/scheduled/event").
			NotEmpty().
			MaxLen(50),

		field.Uint64("triggered_by").
			Comment("触发人用户ID").
			Optional(),

		field.String("triggered_by_name").
			Comment("触发人用户名").
			Optional().
			MaxLen(100),

		// 状态数据（JSON）
		field.Text("state_data").
			Comment("状态相关数据JSON").
			Optional(),

		field.Text("metadata").
			Comment("元数据JSON").
			Optional(),

		// 超时和错误
		field.Bool("is_timeout").
			Comment("是否超时").
			Default(false),

		field.Bool("has_error").
			Comment("是否有错误").
			Default(false),

		field.String("error_message").
			Comment("错误信息").
			Optional().
			MaxLen(1000),

		field.String("error_code").
			Comment("错误代码").
			Optional().
			MaxLen(50),

		// 操作关联
		field.String("operation_id").
			Comment("关联的操作ID").
			Optional().
			MaxLen(100),

		field.String("change_record_id").
			Comment("关联的变更记录ID").
			Optional().
			MaxLen(100),

		// 状态标记
		field.Bool("is_current").
			Comment("是否为当前状态").
			Default(true),

		field.Bool("is_final").
			Comment("是否为最终状态").
			Default(false),

		field.Bool("can_retry").
			Comment("是否可以重试").
			Default(false),

		// 重试信息
		field.Int("retry_count").
			Comment("重试次数").
			Default(0),

		field.Int("max_retry_count").
			Comment("最大重试次数").
			Default(3),

		field.Time("last_retry_at").
			Comment("最后重试时间").
			Optional(),

		// 备注
		field.String("comment").
			Comment("备注说明").
			Optional().
			MaxLen(500),
	}
}

func (CiLifecycleState) Indexes() []ent.Index {
	return []ent.Index{
		// 按state_id查询（唯一索引）
		index.Fields("state_id").Unique(),

		// 按CI ID查询状态历史
		index.Fields("tenant_id", "ci_id", "entered_at"),

		// 按CI类型查询
		index.Fields("ci_type_id"),

		// 按状态类型查询
		index.Fields("state_type"),

		// 查询当前状态
		index.Fields("ci_id", "is_current"),

		// 按操作ID查询
		index.Fields("operation_id"),

		// 超时检测查询
		index.Fields("is_timeout", "expected_exit_at"),

		// 错误状态查询
		index.Fields("has_error"),

		// 复合索引：租户 + CI ID + 是否当前状态
		index.Fields("tenant_id", "ci_id", "is_current"),

		// 复合索引：租户 + 状态类型 + 时间
		index.Fields("tenant_id", "state_type", "entered_at"),
	}
}
