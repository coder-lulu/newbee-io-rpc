package schema

import (
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/mixins"
)

type DiscoveryPool struct {
	ent.Schema
}

func (DiscoveryPool) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.IDMixin{},
		mixins.TenantMixin{},
		mixins.StatusMixin{},
	}
}

func (DiscoveryPool) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(100).
			NotEmpty().
			Comment("发现池名称"),
		
		field.String("description").
			MaxLen(500).
			Optional().
			Comment("发现池描述"),
		
		field.String("discovery_type").
			MaxLen(50).
			NotEmpty().
			Comment("发现类型: file, api, sdk, builtin"),
		
		field.String("pool_status").
			MaxLen(20).
			Default("inactive").
			Comment("池状态: active, inactive, error, maintain"),
		
		field.Text("discovery_config").
			Optional().
			Comment("发现配置JSON"),
		
		field.String("schedule").
			MaxLen(100).
			Optional().
			Comment("调度表达式"),
		
		field.Int("batch_size").
			Default(100).
			Comment("批处理大小"),
		
		field.Int("concurrent_limit").
			Default(5).
			Comment("并发限制"),
		
		field.Int("max_retry").
			Default(3).
			Comment("最大重试次数"),
		
		field.Int("retry_interval").
			Default(60).
			Comment("重试间隔(秒)"),
		
		field.Text("field_mapping").
			Optional().
			Comment("字段映射配置JSON"),
		
		field.Int64("total_runs").
			Default(0).
			Comment("总执行次数"),
		
		field.Int64("success_runs").
			Default(0).
			Comment("成功执行次数"),
		
		field.Int64("failed_runs").
			Default(0).
			Comment("失败执行次数"),
		
		field.Time("last_run_at").
			Optional().
			Nillable().
			Comment("最后执行时间"),
		
		field.Time("last_success_at").
			Optional().
			Nillable().
			Comment("最后成功时间"),
		
		field.Text("last_error").
			Optional().
			Comment("最后错误信息"),
		
		field.String("approval_status").
			MaxLen(20).
			Default("pending").
			Comment("审核状态: pending, approved, rejected"),
		
		field.Uint64("approved_by").
			Optional().
			Nillable().
			Comment("审核人ID"),
		
		field.Time("approved_at").
			Optional().
			Nillable().
			Comment("审核时间"),
		
		field.Text("rejection_reason").
			Optional().
			Comment("拒绝原因"),
		
		field.Text("metadata").
			Optional().
			Comment("扩展元数据JSON"),
	}
}

func (DiscoveryPool) Edges() []ent.Edge {
	return nil
}
// Annotations returns the annotations for DiscoveryPool
func (DiscoveryPool) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "io_discovery_pools"},
	}
}
