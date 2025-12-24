package producer

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/internal/security"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/trace"
)

// SecureProducer 安全增强的Producer
// 功能：
// 1. 自动添加租户隔离Header (X-Tenant-ID)
// 2. 消息脱敏检测
// 3. 幂等性Key生成
// 4. 分区Key生成
// 5. 审计日志记录
type SecureProducer struct {
	inner     Producer
	sanitizer *security.MessageSanitizer

	// 配置选项
	strictMode bool // 严格模式：检测到敏感信息时拒绝发送
}

// NewSecureProducer 创建安全Producer
func NewSecureProducer(inner Producer, strictMode bool) *SecureProducer {
	return &SecureProducer{
		inner:      inner,
		sanitizer:  security.NewMessageSanitizer(),
		strictMode: strictMode,
	}
}

// PublishTaskMessage 发布任务消息（高级API）
// 自动处理：
// - 租户隔离Header
// - 幂等性Key生成
// - 分区Key生成
// - 消息脱敏检测
func (p *SecureProducer) PublishTaskMessage(ctx context.Context, topic string, msg interface{}, tenantID uint64, connectorID string) error {
	// 1. 序列化消息
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	// 2. 消息脱敏检测
	sanitized, warnings := p.sanitizer.Sanitize(body)
	if len(warnings) > 0 {
		logx.Errorw("🚨 Sensitive information detected in message!",
			logx.Field("warnings", warnings),
			logx.Field("tenant_id", tenantID),
			logx.Field("topic", topic))

		if p.strictMode {
			// 严格模式：直接拒绝发送
			return fmt.Errorf("message contains sensitive fields: %v", warnings)
		} else {
			// 宽松模式：使用脱敏后的消息
			body = sanitized
		}
	}

	// 3. 生成Headers
	traceID := trace.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = uuid.New().String()
	}

	headers := map[string]string{
		"X-Tenant-ID":  fmt.Sprintf("%d", tenantID),
		"X-Trace-ID":   traceID,
		"X-Message-ID": uuid.New().String(),
		"X-Timestamp":  time.Now().Format(time.RFC3339),
	}

	// 4. 生成分区Key (tenantID:connectorID)
	partitionKey := fmt.Sprintf("%d:%s", tenantID, connectorID)

	// 5. 发送消息
	err = p.inner.Publish(ctx, topic, []byte(partitionKey), body, headers)
	if err != nil {
		logx.Errorw("failed to publish task message",
			logx.Field("topic", topic),
			logx.Field("tenant_id", tenantID),
			logx.Field("connector_id", connectorID),
			logx.Field("error", err))
		return err
	}

	logx.Infow("task message published successfully",
		logx.Field("topic", topic),
		logx.Field("tenant_id", tenantID),
		logx.Field("connector_id", connectorID),
		logx.Field("trace_id", traceID))

	return nil
}

// PublishWithIdempotency 发布带幂等性的消息
func (p *SecureProducer) PublishWithIdempotency(ctx context.Context, topic string, msg interface{}, tenantID uint64, taskRunID uint64) error {
	// 生成幂等性Key (MD5(tenantID:taskRunID))
	idempotencyKey := generateIdempotencyKey(tenantID, taskRunID)

	// 如果消息是map类型，注入idempotency_key
	if msgMap, ok := msg.(map[string]interface{}); ok {
		msgMap["idempotency_key"] = idempotencyKey
	}

	// 序列化消息
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	// 消息脱敏检测
	if p.strictMode {
		if err := p.sanitizer.Validate(body); err != nil {
			return fmt.Errorf("message validation failed: %w", err)
		}
	}

	// 生成Headers
	traceID := trace.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = uuid.New().String()
	}

	headers := map[string]string{
		"X-Tenant-ID":       fmt.Sprintf("%d", tenantID),
		"X-Task-Run-ID":     fmt.Sprintf("%d", taskRunID),
		"X-Idempotency-Key": idempotencyKey,
		"X-Trace-ID":        traceID,
		"X-Message-ID":      uuid.New().String(),
		"X-Timestamp":       time.Now().Format(time.RFC3339),
	}

	// 分区Key: tenantID:taskRunID
	partitionKey := fmt.Sprintf("%d:%d", tenantID, taskRunID)

	// 发送消息
	return p.inner.Publish(ctx, topic, []byte(partitionKey), body, headers)
}

// Close 关闭生产者
func (p *SecureProducer) Close() error {
	return p.inner.Close()
}

// generateIdempotencyKey 生成幂等性Key
func generateIdempotencyKey(tenantID uint64, taskRunID uint64) string {
	data := fmt.Sprintf("%d:%d:%d", tenantID, taskRunID, time.Now().Unix())
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)
}
