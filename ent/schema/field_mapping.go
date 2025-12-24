package schema

import (
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/mixins"
)

type FieldMapping struct {
	ent.Schema
}

func (FieldMapping) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.IDMixin{},
		mixins.TenantMixin{},
		mixins.StatusMixin{},
	}
}

func (FieldMapping) Fields() []ent.Field {
	return []ent.Field{
		field.String("mapping_name").MaxLen(100).NotEmpty().Comment("映射名称"),
		field.String("description").MaxLen(500).Optional().Comment("描述"),
		field.String("mapping_type").MaxLen(50).Default("input").Comment("映射类型: input, output, transform, validation"),
		field.Bool("is_active").Default(true).Comment("是否启用"),
		
		field.String("source_field").MaxLen(100).NotEmpty().Comment("源字段名称"),
		field.String("source_field_path").MaxLen(200).Optional().Comment("源字段路径"),
		field.String("source_data_type").MaxLen(50).Optional().Comment("源数据类型"),
		field.String("source_format").MaxLen(100).Optional().Comment("源字段格式"),
		
		field.String("target_field").MaxLen(100).NotEmpty().Comment("目标字段名称"),
		field.String("target_field_path").MaxLen(200).Optional().Comment("目标字段路径"),
		field.String("target_data_type").MaxLen(50).Optional().Comment("目标数据类型"),
		field.String("target_format").MaxLen(100).Optional().Comment("目标字段格式"),
		
		field.String("transform_type").MaxLen(50).Default("direct").Comment("转换类型"),
		field.Text("transform_config").Optional().Comment("转换配置JSON"),
		
		field.String("default_value").MaxLen(500).Optional().Comment("默认值"),
		field.Bool("allow_null").Default(true).Comment("允许空值"),
		field.Bool("is_required").Default(false).Comment("是否必填"),
		field.Text("validation_rules").Optional().Comment("验证规则JSON"),
		field.String("validation_regex").MaxLen(500).Optional().Comment("验证正则"),
		
		field.Text("lookup_table").Optional().Comment("查找表JSON"),
		field.Bool("lookup_case_sensitive").Default(true).Comment("查找区分大小写"),
		field.Text("condition_rules").Optional().Comment("条件规则JSON"),
		
		field.Int("priority").Default(0).Comment("优先级"),
		field.Int("sort_order").Default(0).Comment("排序顺序"),
		
		field.Uint64("discovery_pool_id").Optional().Nillable().Comment("关联发现池ID"),
		field.Uint64("input_task_id").Optional().Nillable().Comment("关联输入任务ID"),
		field.Uint64("output_task_id").Optional().Nillable().Comment("关联输出任务ID"),
		
		field.Int64("usage_count").Default(0).Comment("使用次数"),
		field.Int64("success_count").Default(0).Comment("成功次数"),
		field.Int64("failed_count").Default(0).Comment("失败次数"),
		field.Time("last_used_at").Optional().Nillable().Comment("最后使用时间"),
		field.Text("last_error").Optional().Comment("最后错误信息"),
		field.Time("last_error_at").Optional().Nillable().Comment("最后错误时间"),
		
		field.Text("metadata").Optional().Comment("扩展元数据JSON"),
	}
}

func (FieldMapping) Edges() []ent.Edge {
	return nil
}
// Annotations returns the annotations for FieldMapping
func (FieldMapping) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "io_field_mappings"},
	}
}
