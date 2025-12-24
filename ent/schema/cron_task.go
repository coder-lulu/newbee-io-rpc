package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/mixins"
)

// CronTask holds the schema definition for the CronTask entity.
// Cron任务定义 - 周期性任务的元数据
type CronTask struct {
	ent.Schema
}

// Mixin of CronTask
func (CronTask) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.IDMixin{},       // ID和基础字段
		mixins.StatusMixin{},   // status字段
		mixins.TenantMixin{},   // 多租户支持 - tenant_id字段
	}
}

// Fields of the CronTask
func (CronTask) Fields() []ent.Field {
	return []ent.Field{
		// 任务名称
		field.String("task_name").
			Comment("任务名称"),

		// Cron表达式（标准5字段格式）
		field.String("cron_expression").
			Comment("Cron表达式, 例如: 0 3 * * * (每天凌晨3点), 0 */6 * * * (每6小时)"),

		// 数据源类型
		field.String("input_source").
			Comment("数据源类型, 例如: vmware_vcenter, aliyun_ecs等"),

		// 数据源配置（JSON格式）
		field.Text("source_config").
			Optional().
			Comment("数据源配置JSON字符串"),

		// 是否启用
		field.Bool("enabled").
			Default(true).
			Comment("是否启用此Cron任务"),

		// 下次执行时间
		field.Time("next_run_time").
			Optional().
			Comment("下次执行时间（由调度器自动更新）"),

		// 上次执行时间
		field.Time("last_run_time").
			Optional().
			Nillable().
			Comment("上次执行时间"),

		// 统计字段
		field.Int("execution_count").
			Default(0).
			Comment("总执行次数"),

		field.Int("success_count").
			Default(0).
			Comment("成功次数"),

		field.Int("failure_count").
			Default(0).
			Comment("失败次数"),

		// 任务描述
		field.Text("description").
			Optional().
			Comment("任务描述信息"),
	}
}

// Edges of the CronTask
func (CronTask) Edges() []ent.Edge {
	return []ent.Edge{
		// 一个CronTask可以有多个执行实例（InputTask）
		edge.To("executions", InputTask.Type),
	}
}

// Indexes of the CronTask
func (CronTask) Indexes() []ent.Index {
	return []ent.Index{
		// 租户 + 启用状态索引（快速查询启用的任务）
		index.Fields("tenant_id", "enabled"),

		// 下次执行时间索引（调度器快速查找即将到期的任务）
		index.Fields("next_run_time"),

		// 租户 + 任务名称唯一索引（同一租户内任务名称不能重复）
		index.Fields("tenant_id", "task_name").
			Unique(),
	}
}
