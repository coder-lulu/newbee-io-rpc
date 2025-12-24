package schema

import (
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/mixins"
)

type DataTarget struct {
	ent.Schema
}

func (DataTarget) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.IDMixin{},
		mixins.TenantMixin{},
		mixins.StatusMixin{},
	}
}

func (DataTarget) Fields() []ent.Field {
	return []ent.Field{
		field.String("target_name").MaxLen(100).NotEmpty().Comment("目标名称"),
		field.String("target_code").MaxLen(50).NotEmpty().Unique().Comment("目标编码"),
		field.String("description").MaxLen(500).Optional().Comment("描述"),
		field.String("target_type").MaxLen(50).NotEmpty().Comment("目标类型: database, api, file, mq, cache"),
		field.String("target_system").MaxLen(100).Optional().Comment("目标系统名称"),
		field.Text("connection_config").Optional().Comment("连接配置JSON"),
		field.Text("auth_config").Optional().Comment("认证配置JSON"),
		field.Bool("is_active").Default(true).Comment("是否启用"),
		field.Text("metadata").Optional().Comment("扩展元数据JSON"),
	}
}

func (DataTarget) Edges() []ent.Edge {
	return nil
}
// Annotations returns the annotations for DataTarget
func (DataTarget) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "io_data_targets"},
	}
}
