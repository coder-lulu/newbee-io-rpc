package schema

import (
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/mixins"
)

type InputTask struct {
	ent.Schema
}

func (InputTask) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.IDMixin{},
		mixins.TenantMixin{},
		mixins.StatusMixin{},
	}
}

func (InputTask) Fields() []ent.Field {
	return []ent.Field{
		field.String("task_name").MaxLen(100).NotEmpty().Comment("任务名称"),
		field.String("task_type").MaxLen(50).Default("manual").Comment("任务类型: manual, scheduled, triggered"),
		field.String("input_source").MaxLen(50).NotEmpty().Comment("输入源: file, api, database, mq"),
		field.Text("source_config").Optional().Comment("源配置JSON"),
		field.String("task_status").MaxLen(20).Default("pending").Comment("任务状态"),
		field.Uint64("discovery_pool_id").Optional().Nillable().Comment("关联发现池ID"),
		field.Time("scheduled_at").Optional().Nillable().Comment("计划执行时间"),
		// ⭐ Phase 2: Cron 任务关联字段
		field.Uint64("cron_task_id").Optional().Nillable().Comment("关联的CronTask ID（如果是cron任务）"),
		field.Time("execution_time").Optional().Nillable().Comment("本次计划执行时间（用于cron任务）"),
		field.Time("started_at").Optional().Nillable().Comment("开始时间"),
		field.Time("completed_at").Optional().Nillable().Comment("完成时间"),
		field.Int64("total_records").Default(0).Comment("总记录数"),
		field.Int64("processed_records").Default(0).Comment("已处理记录数"),
		field.Int64("success_records").Default(0).Comment("成功记录数"),
		field.Int64("failed_records").Default(0).Comment("失败记录数"),
		field.Text("error_message").Optional().Comment("错误信息"),
		field.Text("metadata").Optional().Comment("扩展元数据JSON"),
	}
}

func (InputTask) Edges() []ent.Edge {
	return []ent.Edge{
		// ⭐ Phase 2: 关联到 CronTask (多个 InputTask 实例属于一个 CronTask)
		edge.From("cron_task", CronTask.Type).
			Ref("executions").
			Field("cron_task_id").
			Unique(),
	}
}
// Annotations returns the annotations for InputTask
func (InputTask) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "io_input_tasks"},
	}
}
