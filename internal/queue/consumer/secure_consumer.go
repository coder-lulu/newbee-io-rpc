package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coder-lulu/newbee-io-rpc/internal/idempotency"
	"github.com/coder-lulu/newbee-io-rpc/internal/queue/types"
	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-zero/core/logx"
)

// TaskMessageHandler 任务消息处理器接口
type TaskMessageHandler interface {
	// HandleTask 处理任务消息
	// 返回nil表示处理成功
	HandleTask(ctx context.Context, msg *types.TaskMessage) error
}

// SecureConsumer 安全增强的Consumer
// 功能：
// 1. 租户隔离验证（Header + Body双重验证）
// 2. 幂等性检查（Redis + DB）
// 3. 审计日志记录
// 4. 错误处理和告警
type SecureConsumer struct {
	allowedTenantID uint64
	groupID         string
	handler         TaskMessageHandler
	idempotencyGuard idempotency.Guard
}

// NewSecureConsumer 创建安全Consumer
func NewSecureConsumer(
	allowedTenantID uint64,
	groupID string,
	handler TaskMessageHandler,
	idempotencyGuard idempotency.Guard,
) *SecureConsumer {
	return &SecureConsumer{
		allowedTenantID:  allowedTenantID,
		groupID:          groupID,
		handler:          handler,
		idempotencyGuard: idempotencyGuard,
	}
}

// Handle 实现MessageHandler接口
func (c *SecureConsumer) Handle(ctx context.Context, msg *kafka.Message) error {
	// 1. 从Header提取租户ID
	tenantIDFromHeader, err := extractTenantIDFromHeaders(msg.Headers)
	if err != nil {
		logx.Errorw("missing tenant ID in headers",
			logx.Field("partition", msg.Partition),
			logx.Field("offset", msg.Offset))
		return err
	}

	// 2. 验证租户ID是否匹配
	if tenantIDFromHeader != c.allowedTenantID {
		logx.Errorw("🚨 Tenant isolation violation!",
			logx.Field("consumer_group", c.groupID),
			logx.Field("expected_tenant", c.allowedTenantID),
			logx.Field("actual_tenant", tenantIDFromHeader),
			logx.Field("partition", msg.Partition),
			logx.Field("offset", msg.Offset))

		// 触发安全告警（在生产环境应发送到告警系统）
		// TODO: 集成AlertManager

		return fmt.Errorf("unauthorized: tenant mismatch (expected %d, got %d)",
			c.allowedTenantID, tenantIDFromHeader)
	}

	// 3. 解析消息体
	var taskMsg types.TaskMessage
	if err := json.Unmarshal(msg.Value, &taskMsg); err != nil {
		logx.Errorw("failed to unmarshal message",
			logx.Field("partition", msg.Partition),
			logx.Field("offset", msg.Offset),
			logx.Field("error", err))
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	// 4. 双重验证：消息体中的租户ID
	if taskMsg.TenantID != tenantIDFromHeader {
		logx.Errorw("header and body tenant mismatch",
			logx.Field("header_tenant", tenantIDFromHeader),
			logx.Field("body_tenant", taskMsg.TenantID),
			logx.Field("partition", msg.Partition),
			logx.Field("offset", msg.Offset))
		return fmt.Errorf("tenant mismatch: header=%d body=%d",
			tenantIDFromHeader, taskMsg.TenantID)
	}

	// 5. 幂等性检查
	if c.idempotencyGuard != nil {
		isNew, err := c.idempotencyGuard.CheckAndMark(ctx, taskMsg.TenantID, taskMsg.TaskRunID)
		if err != nil {
			// 幂等性检查失败，重试
			logx.Errorw("idempotency check failed",
				logx.Field("tenant_id", taskMsg.TenantID),
				logx.Field("task_run_id", taskMsg.TaskRunID),
				logx.Field("error", err))
			return err
		}

		if !isNew {
			// 重复消息，跳过处理
			logx.Infow("skipping duplicate message",
				logx.Field("tenant_id", taskMsg.TenantID),
				logx.Field("task_run_id", taskMsg.TaskRunID),
				logx.Field("partition", msg.Partition),
				logx.Field("offset", msg.Offset))
			return nil
		}
	}

	// 6. 执行业务逻辑
	logx.Infow("processing task message",
		logx.Field("tenant_id", taskMsg.TenantID),
		logx.Field("task_run_id", taskMsg.TaskRunID),
		logx.Field("task_type", taskMsg.TaskType),
		logx.Field("partition", msg.Partition),
		logx.Field("offset", msg.Offset))

	if err := c.handler.HandleTask(ctx, &taskMsg); err != nil {
		logx.Errorw("task handler failed",
			logx.Field("tenant_id", taskMsg.TenantID),
			logx.Field("task_run_id", taskMsg.TaskRunID),
			logx.Field("task_type", taskMsg.TaskType),
			logx.Field("error", err))
		return err
	}

	// 7. 记录审计日志
	logx.Infow("task message processed successfully",
		logx.Field("tenant_id", taskMsg.TenantID),
		logx.Field("task_run_id", taskMsg.TaskRunID),
		logx.Field("task_type", taskMsg.TaskType),
		logx.Field("partition", msg.Partition),
		logx.Field("offset", msg.Offset))

	return nil
}

// extractTenantIDFromHeaders 从Kafka Header提取租户ID
func extractTenantIDFromHeaders(headers []kafka.Header) (uint64, error) {
	for _, h := range headers {
		if h.Key == "X-Tenant-ID" {
			var tenantID uint64
			_, err := fmt.Sscanf(string(h.Value), "%d", &tenantID)
			if err != nil {
				return 0, fmt.Errorf("invalid tenant ID format: %s", string(h.Value))
			}
			return tenantID, nil
		}
	}
	return 0, fmt.Errorf("X-Tenant-ID header not found")
}

// BatchSecureConsumer 批量安全Consumer
type BatchSecureConsumer struct {
	allowedTenantID uint64
	groupID         string
	batchHandler    BatchTaskMessageHandler
	idempotencyGuard idempotency.Guard
}

// BatchTaskMessageHandler 批量任务消息处理器接口
type BatchTaskMessageHandler interface {
	// HandleBatch 批量处理任务消息
	HandleBatch(ctx context.Context, messages []*types.TaskMessage) error
}

// NewBatchSecureConsumer 创建批量安全Consumer
func NewBatchSecureConsumer(
	allowedTenantID uint64,
	groupID string,
	batchHandler BatchTaskMessageHandler,
	idempotencyGuard idempotency.Guard,
) *BatchSecureConsumer {
	return &BatchSecureConsumer{
		allowedTenantID:  allowedTenantID,
		groupID:          groupID,
		batchHandler:     batchHandler,
		idempotencyGuard: idempotencyGuard,
	}
}

// HandleBatch 批量处理消息
func (c *BatchSecureConsumer) HandleBatch(ctx context.Context, kafkaMessages []kafka.Message) error {
	validMessages := make([]*types.TaskMessage, 0, len(kafkaMessages))

	// 1. 逐个验证和解析消息
	for i, msg := range kafkaMessages {
		// 租户验证
		tenantIDFromHeader, err := extractTenantIDFromHeaders(msg.Headers)
		if err != nil {
			logx.Errorw("invalid message in batch",
				logx.Field("index", i),
				logx.Field("error", err))
			continue
		}

		if tenantIDFromHeader != c.allowedTenantID {
			logx.Errorw("tenant mismatch in batch",
				logx.Field("index", i),
				logx.Field("expected_tenant", c.allowedTenantID),
				logx.Field("actual_tenant", tenantIDFromHeader))
			continue
		}

		// 解析消息
		var taskMsg types.TaskMessage
		if err := json.Unmarshal(msg.Value, &taskMsg); err != nil {
			logx.Errorw("unmarshal failed in batch",
				logx.Field("index", i),
				logx.Field("error", err))
			continue
		}

		// 幂等性检查
		if c.idempotencyGuard != nil {
			isNew, err := c.idempotencyGuard.CheckAndMark(ctx, taskMsg.TenantID, taskMsg.TaskRunID)
			if err != nil || !isNew {
				if err != nil {
					logx.Errorw("idempotency check failed in batch",
						logx.Field("index", i),
						logx.Field("error", err))
				} else {
					logx.Infow("duplicate message in batch",
						logx.Field("index", i),
						logx.Field("task_run_id", taskMsg.TaskRunID))
				}
				continue
			}
		}

		validMessages = append(validMessages, &taskMsg)
	}

	// 2. 批量处理有效消息
	if len(validMessages) == 0 {
		logx.Infow("no valid messages in batch",
			logx.Field("total", len(kafkaMessages)))
		return nil
	}

	logx.Infow("processing batch",
		logx.Field("total", len(kafkaMessages)),
		logx.Field("valid", len(validMessages)))

	return c.batchHandler.HandleBatch(ctx, validMessages)
}
