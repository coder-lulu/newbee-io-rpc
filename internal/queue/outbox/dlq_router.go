package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/ent/dlqmessage"
	"github.com/zeromicro/go-zero/core/logx"
)

// DlqRouter 负责将失败的Outbox消息路由到DLQ
type DlqRouter struct {
	db      *ent.Client
	config  *DlqConfig
	metrics *DlqMetrics
}

// DlqConfig DLQ路由配置
type DlqConfig struct {
	// 是否启用DLQ
	Enabled bool

	// DLQ Topic前缀（实际topic为：prefix + original_topic）
	TopicPrefix string

	// 是否发布到Kafka DLQ Topic（如果为false，仅存储到数据库）
	PublishToKafka bool

	// 自动归档时间（天）- 超过此天数的resolved消息会被自动归档
	AutoArchiveDays int

	// 自动清理时间（天）- 超过此天数的archived消息会被自动清理
	AutoCleanupDays int
}

// DlqMetrics DLQ指标
type DlqMetrics struct {
	TotalRouted   int64 // 总路由数
	RouteFailed   int64 // 路由失败数
	TotalResolved int64 // 总解决数
	TotalArchived int64 // 总归档数
}

// NewDlqRouter 创建DLQ路由器
func NewDlqRouter(db *ent.Client, config *DlqConfig) *DlqRouter {
	if config == nil {
		config = DefaultDlqConfig()
	}
	return &DlqRouter{
		db:      db,
		config:  config,
		metrics: &DlqMetrics{},
	}
}

// DefaultDlqConfig 返回默认DLQ配置
func DefaultDlqConfig() *DlqConfig {
	return &DlqConfig{
		Enabled:         true,
		TopicPrefix:     "dlq.",
		PublishToKafka:  false, // 默认仅存储到数据库，不发布到Kafka
		AutoArchiveDays: 30,    // 30天后自动归档
		AutoCleanupDays: 90,    // 90天后自动清理
	}
}

// RouteToDLQ 将失败的Outbox消息路由到DLQ
// 此方法应在OutboxRelay的handleSendFailure中调用，当retry_count >= max_retries时
func (r *DlqRouter) RouteToDLQ(ctx context.Context, outboxMsg *ent.OutboxMessage, failureReason string) error {
	if !r.config.Enabled {
		logx.WithContext(ctx).Infow("DLQ disabled, skipping route",
			logx.Field("message_id", outboxMsg.ID),
			logx.Field("aggregate_type", outboxMsg.AggregateType),
			logx.Field("aggregate_id", outboxMsg.AggregateID))
		return nil
	}

	// 解析message_headers为map[string]string
	headersMap := make(map[string]string)
	if outboxMsg.MessageHeaders != nil {
		// Ent的JSON字段已经是map[string]string类型
		headersMap = outboxMsg.MessageHeaders
	}

	// 解析metadata为map[string]interface{}
	metadataMap := make(map[string]interface{})
	if outboxMsg.Metadata != nil {
		metadataMap = outboxMsg.Metadata
	}

	// 添加DLQ路由时间到metadata
	metadataMap["dlq_routed_at"] = time.Now().Format(time.RFC3339)
	metadataMap["original_message_id"] = outboxMsg.ID

	// 创建DLQ消息
	dlqMsg, err := r.db.DlqMessage.Create().
		SetTenantID(outboxMsg.TenantID).
		SetOriginalMessageID(outboxMsg.ID).
		SetAggregateType(outboxMsg.AggregateType).
		SetAggregateID(outboxMsg.AggregateID).
		SetTopic(r.buildDlqTopic(outboxMsg.Topic)).
		SetNillableMessageKey(stringPtrIfNotEmpty(outboxMsg.MessageKey)).
		SetMessageValue(outboxMsg.MessageValue).
		SetMessageHeaders(headersMap).
		SetNillableEventType(stringPtrIfNotEmpty(outboxMsg.EventType)).
		SetRetryCount(outboxMsg.RetryCount).
		SetFailureReason(failureReason).
		SetFailedAt(time.Now()).
		SetMetadata(metadataMap).
		SetStatus(dlqmessage.StatusPending).
		Save(ctx)

	if err != nil {
		logx.WithContext(ctx).Errorw("Failed to route message to DLQ",
			logx.Field("error", err),
			logx.Field("message_id", outboxMsg.ID),
			logx.Field("aggregate_type", outboxMsg.AggregateType))
		r.metrics.RouteFailed++
		return fmt.Errorf("failed to route message to DLQ: %w", err)
	}

	logx.WithContext(ctx).Infow("Message routed to DLQ",
		logx.Field("dlq_message_id", dlqMsg.ID),
		logx.Field("original_message_id", outboxMsg.ID),
		logx.Field("aggregate_type", outboxMsg.AggregateType),
		logx.Field("aggregate_id", outboxMsg.AggregateID),
		logx.Field("failure_reason", failureReason))

	r.metrics.TotalRouted++

	// 如果配置了发布到Kafka，则发布到DLQ Topic
	// TODO: 实现Kafka发布逻辑
	if r.config.PublishToKafka {
		if err := r.publishToDlqTopic(ctx, dlqMsg); err != nil {
			logx.WithContext(ctx).Errorw("Failed to publish DLQ message to Kafka",
				logx.Field("error", err),
				logx.Field("dlq_message_id", dlqMsg.ID))
			// 发布失败不影响DLQ存储，仅记录错误
		}
	}

	return nil
}

// RequeueToOutbox 将DLQ消息重新入队到Outbox
// 允许手动重试失败的消息
func (r *DlqRouter) RequeueToOutbox(ctx context.Context, dlqMessageID uint64) (*ent.OutboxMessage, error) {
	// 查询DLQ消息
	dlqMsg, err := r.db.DlqMessage.Get(ctx, dlqMessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get DLQ message: %w", err)
	}

	// 检查状态
	if dlqMsg.Status == dlqmessage.StatusResolved {
		return nil, fmt.Errorf("DLQ message already resolved, cannot requeue")
	}
	if dlqMsg.Status == dlqmessage.StatusArchived {
		return nil, fmt.Errorf("DLQ message archived, cannot requeue")
	}

	// 解析message_headers
	headersMap := make(map[string]string)
	if dlqMsg.MessageHeaders != nil {
		headersMap = dlqMsg.MessageHeaders
	}

	// 解析metadata
	metadataMap := make(map[string]interface{})
	if dlqMsg.Metadata != nil {
		metadataMap = dlqMsg.Metadata
	}

	// 添加重新入队信息到metadata
	metadataMap["requeued_from_dlq"] = true
	metadataMap["requeued_at"] = time.Now().Format(time.RFC3339)
	metadataMap["dlq_message_id"] = dlqMsg.ID

	// 创建新的Outbox消息
	outboxMsg, err := r.db.OutboxMessage.Create().
		SetTenantID(dlqMsg.TenantID).
		SetAggregateType(dlqMsg.AggregateType).
		SetAggregateID(dlqMsg.AggregateID).
		SetTopic(r.extractOriginalTopic(dlqMsg.Topic)).
		SetMessageKey(dlqMsg.MessageKey).
		SetMessageValue(dlqMsg.MessageValue).
		SetMessageHeaders(headersMap).
		SetEventType(dlqMsg.EventType).
		SetSendStatus("pending").
		SetRetryCount(0). // 重置重试次数
		SetMetadata(metadataMap).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to create requeued outbox message: %w", err)
	}

	// 更新DLQ消息状态为resolved
	_, err = r.db.DlqMessage.UpdateOneID(dlqMessageID).
		SetStatus(dlqmessage.StatusResolved).
		SetRequeuedAt(time.Now()).
		SetRequeuedMessageID(outboxMsg.ID).
		SetResolutionNotes("Manually requeued to Outbox").
		Save(ctx)

	if err != nil {
		logx.WithContext(ctx).Errorw("Failed to update DLQ message status",
			logx.Field("error", err),
			logx.Field("dlq_message_id", dlqMessageID))
		// 状态更新失败不影响重新入队，仅记录错误
	} else {
		r.metrics.TotalResolved++
	}

	logx.WithContext(ctx).Infow("DLQ message requeued to Outbox",
		logx.Field("dlq_message_id", dlqMessageID),
		logx.Field("new_outbox_message_id", outboxMsg.ID),
		logx.Field("aggregate_type", outboxMsg.AggregateType))

	return outboxMsg, nil
}

// GetPendingMessages 获取待处理的DLQ消息
func (r *DlqRouter) GetPendingMessages(ctx context.Context, tenantID uint64, limit int) ([]*ent.DlqMessage, error) {
	return r.db.DlqMessage.Query().
		Where(
			dlqmessage.TenantIDEQ(tenantID),
			dlqmessage.StatusEQ(dlqmessage.StatusPending),
		).
		Order(ent.Asc(dlqmessage.FieldFailedAt)).
		Limit(limit).
		All(ctx)
}

// ArchiveResolvedMessages 归档已解决的消息
func (r *DlqRouter) ArchiveResolvedMessages(ctx context.Context, olderThanDays int) (int, error) {
	cutoffTime := time.Now().AddDate(0, 0, -olderThanDays)

	count, err := r.db.DlqMessage.Update().
		Where(
			dlqmessage.StatusEQ(dlqmessage.StatusResolved),
			dlqmessage.RequeuedAtLT(cutoffTime),
		).
		SetStatus(dlqmessage.StatusArchived).
		SetArchivedAt(time.Now()).
		Save(ctx)

	if err != nil {
		return 0, fmt.Errorf("failed to archive resolved messages: %w", err)
	}

	r.metrics.TotalArchived += int64(count)
	logx.WithContext(ctx).Infow("Archived resolved DLQ messages",
		logx.Field("count", count),
		logx.Field("older_than_days", olderThanDays))

	return count, nil
}

// DeleteArchivedMessages 删除已归档的消息
func (r *DlqRouter) DeleteArchivedMessages(ctx context.Context, olderThanDays int) (int, error) {
	cutoffTime := time.Now().AddDate(0, 0, -olderThanDays)

	count, err := r.db.DlqMessage.Delete().
		Where(
			dlqmessage.StatusEQ(dlqmessage.StatusArchived),
			dlqmessage.ArchivedAtLT(cutoffTime),
		).
		Exec(ctx)

	if err != nil {
		return 0, fmt.Errorf("failed to delete archived messages: %w", err)
	}

	logx.WithContext(ctx).Infow("Deleted archived DLQ messages",
		logx.Field("count", count),
		logx.Field("older_than_days", olderThanDays))

	return count, nil
}

// GetMetrics 获取DLQ指标
func (r *DlqRouter) GetMetrics() *DlqMetrics {
	return r.metrics
}

// buildDlqTopic 构建DLQ Topic名称
// 例如：io.input.jobs -> dlq.io.input.jobs
func (r *DlqRouter) buildDlqTopic(originalTopic string) string {
	return r.config.TopicPrefix + originalTopic
}

// extractOriginalTopic 提取原始Topic名称
// 例如：dlq.io.input.jobs -> io.input.jobs
func (r *DlqRouter) extractOriginalTopic(dlqTopic string) string {
	prefix := r.config.TopicPrefix
	if len(dlqTopic) > len(prefix) && dlqTopic[:len(prefix)] == prefix {
		return dlqTopic[len(prefix):]
	}
	return dlqTopic
}

// publishToDlqTopic 发布消息到Kafka DLQ Topic
// TODO: 实现Kafka发布逻辑
func (r *DlqRouter) publishToDlqTopic(ctx context.Context, dlqMsg *ent.DlqMessage) error {
	// 构建DLQ消息payload
	payload := map[string]interface{}{
		"dlq_message_id":      dlqMsg.ID,
		"original_message_id": dlqMsg.OriginalMessageID,
		"aggregate_type":      dlqMsg.AggregateType,
		"aggregate_id":        dlqMsg.AggregateID,
		"tenant_id":           dlqMsg.TenantID,
		"failure_reason":      dlqMsg.FailureReason,
		"retry_count":         dlqMsg.RetryCount,
		"failed_at":           dlqMsg.FailedAt.Format(time.RFC3339),
		"original_payload":    string(dlqMsg.MessageValue),
		"metadata":            dlqMsg.Metadata,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ payload: %w", err)
	}

	logx.WithContext(ctx).Infow("DLQ message ready to publish to Kafka",
		logx.Field("topic", dlqMsg.Topic),
		logx.Field("dlq_message_id", dlqMsg.ID),
		logx.Field("payload_size", len(payloadBytes)))

	// TODO: 实际发布到Kafka
	// 可以集成KafkaProducer，类似OutboxRelay
	return nil
}

// stringPtrIfNotEmpty 返回字符串指针（如果非空）
func stringPtrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
