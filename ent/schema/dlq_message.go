package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/mixins"
)

// DlqMessage holds the schema definition for the DlqMessage entity.
// 死信队列消息表，用于存储Outbox发送失败且超过最大重试次数的消息
type DlqMessage struct {
	ent.Schema
}

// Mixin defines the mixins for the DlqMessage schema
func (DlqMessage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.IDMixin{},      // ID, CreatedAt, UpdatedAt
		mixins.TenantMixin{},  // TenantID, 租户隔离
	}
}

// Fields of the DlqMessage.
func (DlqMessage) Fields() []ent.Field {
	return []ent.Field{
		// 原始Outbox消息ID
		field.Uint64("original_message_id").
			Comment("原始Outbox消息ID"),

		// 聚合信息
		field.String("aggregate_type").
			MaxLen(100).
			NotEmpty().
			Comment("聚合类型（如：InputTask, DataTarget）"),

		field.String("aggregate_id").
			MaxLen(100).
			NotEmpty().
			Comment("聚合ID"),

		// Kafka消息信息
		field.String("topic").
			MaxLen(255).
			NotEmpty().
			Comment("Kafka Topic"),

		field.String("message_key").
			MaxLen(255).
			Optional().
			Comment("Kafka消息Key"),

		field.Bytes("message_value").
			Comment("Kafka消息内容（JSON）"),

		field.JSON("message_headers", map[string]string{}).
			Optional().
			SchemaType(map[string]string{
				dialect.MySQL:    "json",
				dialect.Postgres: "jsonb",
			}).
			Comment("Kafka消息Headers（JSON）"),

		// 事件信息
		field.String("event_type").
			MaxLen(100).
			Optional().
			Comment("事件类型"),

		// 失败信息
		field.Int("retry_count").
			Default(0).
			Comment("失败前的重试次数"),

		field.String("failure_reason").
			MaxLen(1000).
			Optional().
			Comment("失败原因"),

		field.Time("failed_at").
			Comment("失败时间"),

		// 元数据
		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			SchemaType(map[string]string{
				dialect.MySQL:    "json",
				dialect.Postgres: "jsonb",
			}).
			Comment("元数据（JSON）"),

		// 处理状态
		field.Enum("status").
			Values("pending", "processing", "resolved", "archived").
			Default("pending").
			Comment("处理状态：pending-待处理, processing-处理中, resolved-已解决, archived-已归档"),

		// 重新入队信息
		field.Time("requeued_at").
			Optional().
			Nillable().
			Comment("重新入队时间"),

		field.Uint64("requeued_message_id").
			Optional().
			Comment("重新入队后的新Outbox消息ID"),

		// 归档信息
		field.Time("archived_at").
			Optional().
			Nillable().
			Comment("归档时间"),

		field.String("resolution_notes").
			MaxLen(2000).
			Optional().
			Comment("处理说明"),
	}
}

// Edges of the DlqMessage.
func (DlqMessage) Edges() []ent.Edge {
	return nil
}

// Indexes of the DlqMessage.
func (DlqMessage) Indexes() []ent.Index {
	return []ent.Index{
		// 租户隔离索引
		index.Fields("tenant_id"),

		// 状态查询索引
		index.Fields("status"),

		// 聚合类型查询索引
		index.Fields("aggregate_type", "aggregate_id"),

		// Topic查询索引
		index.Fields("topic"),

		// 失败时间索引（用于统计和清理）
		index.Fields("failed_at"),

		// 组合索引：租户+状态
		index.Fields("tenant_id", "status"),
	}
}

// Annotations of the DlqMessage.
func (DlqMessage) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "io_dlq_messages"},
	}
}
