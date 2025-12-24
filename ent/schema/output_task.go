package schema

import (
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/mixins"
)

type OutputTask struct {
	ent.Schema
}

func (OutputTask) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.IDMixin{},
		mixins.TenantMixin{},
		mixins.StatusMixin{},
	}
}

func (OutputTask) Fields() []ent.Field {
	return []ent.Field{
		field.String("task_name").MaxLen(100).NotEmpty().Comment("任务名称"),
		field.String("task_type").MaxLen(50).Default("manual").Comment("任务类型"),
		field.String("output_target").MaxLen(50).NotEmpty().Comment("输出目标类型"),
		field.Text("target_config").Optional().Comment("目标配置JSON"),
		field.String("task_status").MaxLen(20).Default("pending").Comment("任务状态"),
		field.Uint64("data_target_id").Optional().Nillable().Comment("关联数据目标ID"),
		field.Time("scheduled_at").Optional().Nillable().Comment("计划执行时间"),
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

func (OutputTask) Edges() []ent.Edge {
	return nil
}
// Annotations returns the annotations for OutputTask
func (OutputTask) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "io_output_tasks"},
	}
}
