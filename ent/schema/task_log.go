package schema

import (
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/mixins"
)

type TaskLog struct {
	ent.Schema
}

func (TaskLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.IDMixin{},
		mixins.TenantMixin{},
	}
}

func (TaskLog) Fields() []ent.Field {
	return []ent.Field{
		field.String("task_type").MaxLen(50).NotEmpty().Comment("任务类型: input, output"),
		field.Uint64("task_id").Comment("任务ID"),
		field.String("log_level").MaxLen(20).Default("info").Comment("日志级别"),
		field.Text("log_message").Comment("日志消息"),
		field.Text("log_detail").Optional().Comment("详细信息JSON"),
		field.Time("logged_at").Comment("记录时间"),
	}
}

func (TaskLog) Edges() []ent.Edge {
	return nil
}
// Annotations returns the annotations for TaskLog
func (TaskLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "io_task_logs"},
	}
}
