package schema

import (
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/mixins"
)

type WorkerMetrics struct {
	ent.Schema
}

func (WorkerMetrics) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.IDMixin{},
	}
}

func (WorkerMetrics) Fields() []ent.Field {
	return []ent.Field{
		field.String("worker_id").MaxLen(100).NotEmpty().Comment("Worker ID"),
		field.String("worker_name").MaxLen(100).NotEmpty().Comment("Worker名称"),
		field.String("worker_status").MaxLen(20).Default("idle").Comment("Worker状态: idle, busy, offline"),
		field.Int("current_tasks").Default(0).Comment("当前任务数"),
		field.Int("total_tasks").Default(0).Comment("总任务数"),
		field.Int("success_tasks").Default(0).Comment("成功任务数"),
		field.Int("failed_tasks").Default(0).Comment("失败任务数"),
		field.Float("cpu_usage").Default(0).Comment("CPU使用率"),
		field.Float("memory_usage").Default(0).Comment("内存使用率"),
		field.Time("last_heartbeat").Optional().Nillable().Comment("最后心跳时间"),
		field.Text("metadata").Optional().Comment("扩展元数据JSON"),
	}
}

func (WorkerMetrics) Edges() []ent.Edge {
	return nil
}
// Annotations returns the annotations for WorkerMetrics
func (WorkerMetrics) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "io_worker_metrics"},
	}
}
