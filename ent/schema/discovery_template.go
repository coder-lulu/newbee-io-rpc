package schema

import (
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/mixins"
)

type DiscoveryTemplate struct {
	ent.Schema
}

func (DiscoveryTemplate) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.IDMixin{},
		mixins.TenantMixin{},
		mixins.StatusMixin{},
	}
}

func (DiscoveryTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.String("template_name").
			MaxLen(100).
			NotEmpty().
			Comment("模板名称"),
		
		field.String("template_code").
			MaxLen(50).
			NotEmpty().
			Unique().
			Comment("模板编码"),
		
		field.String("description").
			MaxLen(500).
			Optional().
			Comment("描述"),
		
		field.String("version").
			MaxLen(20).
			Default("1.0.0").
			Comment("版本号"),
		
		field.String("template_type").
			MaxLen(50).
			Default("standard").
			Comment("模板类型"),
		
		field.Text("discovery_config").
			Optional().
			Comment("发现配置JSON"),
		
		field.Text("field_mapping_templates").
			Optional().
			Comment("字段映射模板JSON"),
		
		field.Text("validation_rules").
			Optional().
			Comment("验证规则JSON"),
		
		field.Bool("is_public").
			Default(false).
			Comment("是否公开"),
		
		field.Bool("is_system").
			Default(false).
			Comment("是否系统模板"),
		
		field.Int64("usage_count").
			Default(0).
			Comment("使用次数"),
		
		field.String("tags").
			MaxLen(500).
			Optional().
			Comment("标签,逗号分隔"),
		
		field.Text("metadata").
			Optional().
			Comment("扩展元数据JSON"),
	}
}

func (DiscoveryTemplate) Edges() []ent.Edge {
	return nil
}
// Annotations returns the annotations for DiscoveryTemplate
func (DiscoveryTemplate) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "io_discovery_templates"},
	}
}
