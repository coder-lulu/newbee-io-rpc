package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/coder-lulu/newbee-common/v2/orm/ent/mixins"
)

// OutboxMessage holds the schema definition for the OutboxMessage entity.
// Outbox模式用于确保消息发送的事务一致性和可靠性。
//
// 工作原理：
// 1. 业务操作和消息保存在同一数据库事务中完成
// 2. 后台Relay任务定时扫描pending状态的消息
// 3. 发送成功后更新状态为sent
// 4. 失败后增加重试次数，超过阈值后标记为failed
//
// 优势：
// - 消息不会丢失（事务保证）
// - 最终一致性保证
// - 失败可重试
// - 租户隔离
type OutboxMessage struct {
	ent.Schema
}

// Mixin returns the mixins for OutboxMessage
func (OutboxMessage) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.IDMixin{},      // ID, CreatedAt, UpdatedAt
		mixins.TenantMixin{},  // TenantID, 租户隔离
		mixins.StatusMixin{},  // Status, 状态管理
	}
}

// Fields returns the fields for OutboxMessage
func (OutboxMessage) Fields() []ent.Field {
	return []ent.Field{
		// 聚合信息 - 用于关联业务实体
		field.String("aggregate_type").
			MaxLen(100).
			NotEmpty().
			Comment("聚合类型 | Aggregate type (e.g., InputTask, OutputTask, FieldMapping)"),

		field.String("aggregate_id").
			MaxLen(100).
			NotEmpty().
			Comment("聚合ID | Aggregate ID (business entity ID)"),

		// Kafka消息信息
		field.String("topic").
			MaxLen(200).
			NotEmpty().
			Comment("Kafka Topic | Kafka topic name"),

		field.String("message_key").
			MaxLen(500).
			Optional().
			Comment("消息Key | Message key for partitioning"),

		field.Bytes("message_value").
			Comment("消息体 | Message payload (JSON bytes)"),

		field.JSON("message_headers", map[string]string{}).
			Optional().
			Comment("消息Header | Message headers (e.g., X-Tenant-ID, X-Correlation-ID)"),

		// 事件类型
		field.String("event_type").
			MaxLen(100).
			NotEmpty().
			Comment("事件类型 | Event type (e.g., TaskCreated, TaskCompleted, MappingUpdated)"),

		// 发送状态
		field.String("send_status").
			MaxLen(20).
			Default("pending").
			Comment("发送状态 | Send status: pending, sent, failed"),

		// 重试机制
		field.Int("retry_count").
			Default(0).
			Comment("重试次数 | Retry count"),

		field.Int("max_retries").
			Default(3).
			Comment("最大重试次数 | Maximum retry attempts"),

		// 时间信息
		field.Time("sent_at").
			Optional().
			Nillable().
			Comment("发送时间 | Sent timestamp"),

		field.Time("next_retry_at").
			Optional().
			Nillable().
			Comment("下次重试时间 | Next retry timestamp (for exponential backoff)"),

		// 错误信息
		field.Text("error_message").
			Optional().
			Comment("错误信息 | Error message if send failed"),

		field.Text("last_error").
			Optional().
			Comment("最后一次错误 | Last error details"),

		// 元数据
		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			Comment("扩展元数据 | Extended metadata (correlation_id, user_id, etc.)"),

		// 优先级（可选，用于优先级队列）
		field.Int("priority").
			Default(5).
			Comment("优先级 | Priority (1-10, higher = more urgent)"),
	}
}

// Edges returns the edges for OutboxMessage
func (OutboxMessage) Edges() []ent.Edge {
	return nil
}

// Indexes returns the indexes for OutboxMessage
func (OutboxMessage) Indexes() []ent.Index {
	return []ent.Index{
		// 租户 + 发送状态索引（最重要，用于查询待发送消息）
		index.Fields("tenant_id", "send_status"),

		// 租户 + 聚合类型 + 聚合ID索引（用于查询某个业务实体的消息）
		index.Fields("tenant_id", "aggregate_type", "aggregate_id"),

		// 下次重试时间索引（用于Relay定时任务）
		index.Fields("send_status", "next_retry_at"),

		// 创建时间索引（用于清理旧数据）
		index.Fields("created_at"),

		// 主题索引（用于监控和统计）
		index.Fields("topic", "send_status"),
	}
}
// Annotations returns the annotations for OutboxMessage
func (OutboxMessage) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "io_outbox_messages"},
	}
}
