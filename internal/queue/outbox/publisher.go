package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/zeromicro/go-zero/core/logx"
)

// OutboxPublisher 负责在事务中保存消息到Outbox表
//
// 工作原理：
// 1. 业务逻辑在事务中操作
// 2. 使用Publisher在同一事务中保存消息到outbox_messages表
// 3. 事务提交后，消息和业务数据都持久化
// 4. OutboxRelay定时任务负责发送消息
//
// 优势：
// - 事务一致性：消息和业务操作要么都成功，要么都失败
// - 消息不丢失：即使Kafka不可用，消息也已保存在数据库中
// - 解耦：业务逻辑不直接依赖Kafka
type OutboxPublisher struct {
	db *ent.Client
}

// NewOutboxPublisher 创建新的OutboxPublisher
func NewOutboxPublisher(db *ent.Client) *OutboxPublisher {
	return &OutboxPublisher{
		db: db,
	}
}

// OutboxMessage 表示要保存的Outbox消息
type OutboxMessage struct {
	TenantID      uint64                 // 租户ID
	AggregateType string                 // 聚合类型（例如：InputTask, OutputTask）
	AggregateID   string                 // 聚合ID（业务实体ID）
	Topic         string                 // Kafka Topic
	MessageKey    string                 // 消息Key（用于分区）
	MessageValue  []byte                 // 消息体（JSON字节）
	MessageHeaders map[string]string     // 消息Headers
	EventType     string                 // 事件类型（例如：TaskCreated, TaskCompleted）
	Priority      int                    // 优先级（1-10，默认5）
	MaxRetries    int                    // 最大重试次数（默认3）
	Metadata      map[string]interface{} // 扩展元数据
}

// SaveToOutbox 在事务中保存消息到Outbox表
//
// 参数：
// - ctx: 上下文
// - tx: ent事务对象（必须在事务中调用）
// - msg: 要保存的消息
//
// 返回：
// - *ent.OutboxMessage: 保存后的消息实体
// - error: 错误信息
//
// 示例：
//   err := entx.WithTx(ctx, db, func(tx *ent.Tx) error {
//       // 业务操作
//       task, err := tx.InputTask.Create().SetTaskName("test").Save(ctx)
//       if err != nil {
//           return err
//       }
//
//       // 保存消息到Outbox
//       msg := &OutboxMessage{
//           TenantID:      tenantID,
//           AggregateType: "InputTask",
//           AggregateID:   fmt.Sprintf("%d", task.ID),
//           Topic:         "io.input.jobs",
//           MessageKey:    fmt.Sprintf("%d:%d", tenantID, task.ID),
//           MessageValue:  jsonBytes,
//           EventType:     "TaskCreated",
//       }
//       _, err = publisher.SaveToOutbox(ctx, tx, msg)
//       return err
//   })
func (p *OutboxPublisher) SaveToOutbox(ctx context.Context, tx *ent.Tx, msg *OutboxMessage) (*ent.OutboxMessage, error) {
	// 参数验证
	if msg.TenantID == 0 {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if msg.AggregateType == "" {
		return nil, fmt.Errorf("aggregate_type is required")
	}
	if msg.AggregateID == "" {
		return nil, fmt.Errorf("aggregate_id is required")
	}
	if msg.Topic == "" {
		return nil, fmt.Errorf("topic is required")
	}
	if len(msg.MessageValue) == 0 {
		return nil, fmt.Errorf("message_value is required")
	}
	if msg.EventType == "" {
		return nil, fmt.Errorf("event_type is required")
	}

	// 设置默认值
	if msg.Priority == 0 {
		msg.Priority = 5 // 默认优先级
	}
	if msg.MaxRetries == 0 {
		msg.MaxRetries = 3 // 默认最大重试3次
	}

	// 创建Outbox消息
	builder := tx.OutboxMessage.Create().
		SetTenantID(msg.TenantID).
		SetAggregateType(msg.AggregateType).
		SetAggregateID(msg.AggregateID).
		SetTopic(msg.Topic).
		SetMessageValue(msg.MessageValue).
		SetEventType(msg.EventType).
		SetSendStatus("pending"). // 初始状态为pending
		SetRetryCount(0).          // 初始重试次数为0
		SetMaxRetries(msg.MaxRetries).
		SetPriority(msg.Priority)

	// 可选字段
	if msg.MessageKey != "" {
		builder.SetMessageKey(msg.MessageKey)
	}
	if msg.MessageHeaders != nil {
		builder.SetMessageHeaders(msg.MessageHeaders)
	}
	if msg.Metadata != nil {
		builder.SetMetadata(msg.Metadata)
	}

	// 保存到数据库
	outboxMsg, err := builder.Save(ctx)
	if err != nil {
		logx.Errorw("Failed to save outbox message",
			logx.Field("tenant_id", msg.TenantID),
			logx.Field("aggregate_type", msg.AggregateType),
			logx.Field("aggregate_id", msg.AggregateID),
			logx.Field("topic", msg.Topic),
			logx.Field("event_type", msg.EventType),
			logx.Field("error", err))
		return nil, fmt.Errorf("failed to save outbox message: %w", err)
	}

	logx.Infow("Outbox message saved",
		logx.Field("outbox_id", outboxMsg.ID),
		logx.Field("tenant_id", msg.TenantID),
		logx.Field("aggregate_type", msg.AggregateType),
		logx.Field("aggregate_id", msg.AggregateID),
		logx.Field("topic", msg.Topic),
		logx.Field("event_type", msg.EventType))

	return outboxMsg, nil
}

// SaveTaskCreatedMessage 保存任务创建事件消息（便捷方法）
//
// 参数：
// - ctx: 上下文
// - tx: ent事务对象
// - task: 任务实体（InputTask或OutputTask）
// - tenantID: 租户ID
//
// 返回：
// - error: 错误信息
func (p *OutboxPublisher) SaveTaskCreatedMessage(ctx context.Context, tx *ent.Tx, taskID uint64, taskName string, tenantID uint64) error {
	// 构造消息体
	payload := map[string]interface{}{
		"task_id":    taskID,
		"task_name":  taskName,
		"tenant_id":  tenantID,
		"event_type": "TaskCreated",
		"timestamp":  time.Now().Unix(),
	}
	messageValue, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal task message: %w", err)
	}

	// 保存到Outbox
	msg := &OutboxMessage{
		TenantID:      tenantID,
		AggregateType: "InputTask",
		AggregateID:   fmt.Sprintf("%d", taskID),
		Topic:         "io.input.jobs",
		MessageKey:    fmt.Sprintf("%d:%d", tenantID, taskID),
		MessageValue:  messageValue,
		MessageHeaders: map[string]string{
			"X-Tenant-ID":      fmt.Sprintf("%d", tenantID),
			"X-Event-Type":     "TaskCreated",
			"X-Correlation-ID": fmt.Sprintf("task-%d-%d", tenantID, taskID),
		},
		EventType: "TaskCreated",
		Priority:  5,
	}

	_, err = p.SaveToOutbox(ctx, tx, msg)
	return err
}

// SaveTaskCompletedMessage 保存任务完成事件消息（便捷方法）
func (p *OutboxPublisher) SaveTaskCompletedMessage(ctx context.Context, tx *ent.Tx, taskID uint64, taskName string, tenantID uint64, status string) error {
	// 构造消息体
	payload := map[string]interface{}{
		"task_id":    taskID,
		"task_name":  taskName,
		"tenant_id":  tenantID,
		"status":     status,
		"event_type": "TaskCompleted",
		"timestamp":  time.Now().Unix(),
	}
	messageValue, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal task message: %w", err)
	}

	// 保存到Outbox
	msg := &OutboxMessage{
		TenantID:      tenantID,
		AggregateType: "InputTask",
		AggregateID:   fmt.Sprintf("%d", taskID),
		Topic:         "io.input.jobs",
		MessageKey:    fmt.Sprintf("%d:%d", tenantID, taskID),
		MessageValue:  messageValue,
		MessageHeaders: map[string]string{
			"X-Tenant-ID":      fmt.Sprintf("%d", tenantID),
			"X-Event-Type":     "TaskCompleted",
			"X-Correlation-ID": fmt.Sprintf("task-%d-%d", tenantID, taskID),
		},
		EventType: "TaskCompleted",
		Priority:  5,
	}

	_, err = p.SaveToOutbox(ctx, tx, msg)
	return err
}

// SaveCustomMessage 保存自定义消息（通用方法）
//
// 参数：
// - ctx: 上下文
// - tx: ent事务对象
// - tenantID: 租户ID
// - topic: Kafka Topic
// - eventType: 事件类型
// - payload: 消息体（将被JSON序列化）
// - options: 可选参数（aggregateType, aggregateID, priority等）
//
// 返回：
// - error: 错误信息
func (p *OutboxPublisher) SaveCustomMessage(ctx context.Context, tx *ent.Tx, tenantID uint64, topic string, eventType string, payload interface{}, options ...func(*OutboxMessage)) error {
	// 序列化消息体
	messageValue, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal custom message: %w", err)
	}

	// 创建基础消息
	msg := &OutboxMessage{
		TenantID:      tenantID,
		AggregateType: "Custom",
		AggregateID:   fmt.Sprintf("%d-%d", tenantID, time.Now().UnixNano()),
		Topic:         topic,
		MessageKey:    fmt.Sprintf("%d:%d", tenantID, time.Now().UnixNano()),
		MessageValue:  messageValue,
		MessageHeaders: map[string]string{
			"X-Tenant-ID":  fmt.Sprintf("%d", tenantID),
			"X-Event-Type": eventType,
		},
		EventType: eventType,
		Priority:  5,
	}

	// 应用可选参数
	for _, opt := range options {
		opt(msg)
	}

	// 保存到Outbox
	_, err = p.SaveToOutbox(ctx, tx, msg)
	return err
}

// 可选参数函数

// WithAggregateType 设置聚合类型
func WithAggregateType(aggregateType string) func(*OutboxMessage) {
	return func(msg *OutboxMessage) {
		msg.AggregateType = aggregateType
	}
}

// WithAggregateID 设置聚合ID
func WithAggregateID(aggregateID string) func(*OutboxMessage) {
	return func(msg *OutboxMessage) {
		msg.AggregateID = aggregateID
	}
}

// WithMessageKey 设置消息Key
func WithMessageKey(messageKey string) func(*OutboxMessage) {
	return func(msg *OutboxMessage) {
		msg.MessageKey = messageKey
	}
}

// WithPriority 设置优先级
func WithPriority(priority int) func(*OutboxMessage) {
	return func(msg *OutboxMessage) {
		msg.Priority = priority
	}
}

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(maxRetries int) func(*OutboxMessage) {
	return func(msg *OutboxMessage) {
		msg.MaxRetries = maxRetries
	}
}

// WithMetadata 设置元数据
func WithMetadata(metadata map[string]interface{}) func(*OutboxMessage) {
	return func(msg *OutboxMessage) {
		msg.Metadata = metadata
	}
}

// WithHeaders 设置消息Headers
func WithHeaders(headers map[string]string) func(*OutboxMessage) {
	return func(msg *OutboxMessage) {
		if msg.MessageHeaders == nil {
			msg.MessageHeaders = make(map[string]string)
		}
		for k, v := range headers {
			msg.MessageHeaders[k] = v
		}
	}
}
