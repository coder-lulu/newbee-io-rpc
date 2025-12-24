package schema

import (
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/mixins"
)

type DiscoveryProviderSchema struct {
	ent.Schema
}

func (DiscoveryProviderSchema) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.IDMixin{},
		mixins.StatusMixin{},
	}
}

func (DiscoveryProviderSchema) Fields() []ent.Field {
	return []ent.Field{
		field.Uint64("tenant_id").
			Comment("Tenant ID | 租户 ID"),

		field.Uint64("department_id").
			Default(0).
			Comment("Department ID | 部门 ID (0=所有部门共享)"),

		field.String("provider_id").
			MaxLen(50).
			NotEmpty().
			Unique().
			Comment("Provider唯一标识"),
		
		field.String("provider_name").
			MaxLen(100).
			NotEmpty().
			Comment("Provider名称"),
		
		field.Enum("category").
			Values("api", "sdk", "file", "builtin", "agent").
			Comment("Provider类别"),
		
		field.JSON("parameter_schema", []interface{}{}).
			Optional().
			SchemaType(map[string]string{
				dialect.MySQL: "json",
			}).
			Comment("参数定义Schema(JSON)"),
		
		field.JSON("field_schema", []interface{}{}).
			Optional().
			SchemaType(map[string]string{
				dialect.MySQL: "json",
			}).
			Comment("字段定义Schema(JSON)"),
		
		field.String("description").
			MaxLen(500).
			Optional().
			Comment("描述"),
		
		field.String("version").
			MaxLen(20).
			Optional().
			Comment("版本号"),
		
		field.String("icon_url").
			MaxLen(255).
			Optional().
			Comment("图标URL"),
		
		field.Bool("is_builtin").
			Default(false).
			Comment("是否内置Provider"),
		
		field.Enum("execution_mode").
			Values("worker", "agent", "direct").
			Default("direct").
			Comment("执行模式: worker-后台任务, agent-Agent执行, direct-直接执行"),
		
		field.Bool("is_active").
			Default(true).
			Comment("是否启用"),
	}
}

func (DiscoveryProviderSchema) Edges() []ent.Edge {
	return nil
}

func (DiscoveryProviderSchema) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
	}
}

// Annotations returns the annotations for DiscoveryProviderSchema
func (DiscoveryProviderSchema) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "io_discovery_provider_schemas"},
	}
}
