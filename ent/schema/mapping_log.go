package schema

import (
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/mixins"
)

type MappingLog struct {
	ent.Schema
}

func (MappingLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.IDMixin{},
		mixins.TenantMixin{},
	}
}

func (MappingLog) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("field_mapping_id").Comment("字段映射ID"),
		field.String("source_value").MaxLen(1000).Optional().Comment("源值"),
		field.String("target_value").MaxLen(1000).Optional().Comment("目标值"),
		field.String("transform_status").MaxLen(20).Default("success").Comment("转换状态: success, failed, skipped"),
		field.Text("error_message").Optional().Comment("错误信息"),
		field.Time("logged_at").Comment("记录时间"),
	}
}

func (MappingLog) Edges() []ent.Edge {
	return nil
}
// Annotations returns the annotations for MappingLog
func (MappingLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "io_mapping_logs"},
	}
}
