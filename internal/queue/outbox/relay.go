package outbox

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/ent/outboxmessage"
	"github.com/coder-lulu/newbee-io-rpc/internal/queue/producer"
	"github.com/zeromicro/go-zero/core/logx"
)

// OutboxRelay 负责定时扫描Outbox表并发送消息到Kafka
//
// 工作原理：
// 1. 定时扫描（例如每5秒）
// 2. 查询send_status='pending'的消息（批量，例如100条）
// 3. 逐条发送到Kafka
// 4. 发送成功 → 更新send_status='sent', sent_at=NOW()
// 5. 发送失败 → 增加retry_count，计算下次重试时间（指数退避）
// 6. 超过max_retries → 标记为failed
//
// 优势：
// - 异步发送：不阻塞业务操作
// - 失败重试：自动重试机制
// - 最终一致性：保证消息最终发送成功
type OutboxRelay struct {
	db              *ent.Client
	producer        producer.Producer
	secureProducer  *producer.SecureProducer
	dlqRouter       *DlqRouter     // DLQ路由器
	interval        time.Duration  // 扫描间隔
	batchSize       int            // 每次处理消息数量
	running         bool           // 是否正在运行
	mu              sync.RWMutex   // 读写锁
	stopCh          chan struct{}  // 停止信号
	wg              sync.WaitGroup // 等待所有任务完成
	enableMetrics   bool           // 是否启用指标收集
	metricsInterval time.Duration  // 指标收集间隔
}

// RelayConfig Relay配置
type RelayConfig struct {
	Interval        time.Duration // 扫描间隔（默认5秒）
	BatchSize       int           // 批量处理大小（默认100）
	EnableMetrics   bool          // 是否启用指标收集（默认false）
	MetricsInterval time.Duration // 指标收集间隔（默认1分钟）
	DlqConfig       *DlqConfig    // DLQ配置（可选，如果为nil则使用默认配置）
}

// DefaultRelayConfig 返回默认配置
func DefaultRelayConfig() *RelayConfig {
	return &RelayConfig{
		Interval:        5 * time.Second,
		BatchSize:       100,
		EnableMetrics:   false,
		MetricsInterval: 1 * time.Minute,
	}
}

// NewOutboxRelay 创建新的OutboxRelay
//
// 参数：
// - db: ent数据库客户端
// - producer: Kafka Producer
// - secureProducer: 安全的Kafka Producer（可选，用于多租户环境）
// - config: Relay配置
//
// 返回：
// - *OutboxRelay: Relay实例
func NewOutboxRelay(db *ent.Client, producer producer.Producer, secureProducer *producer.SecureProducer, config *RelayConfig) *OutboxRelay {
	if config == nil {
		config = DefaultRelayConfig()
	}

	// 创建DLQ路由器
	dlqRouter := NewDlqRouter(db, config.DlqConfig)

	return &OutboxRelay{
		db:              db,
		producer:        producer,
		secureProducer:  secureProducer,
		dlqRouter:       dlqRouter,
		interval:        config.Interval,
		batchSize:       config.BatchSize,
		enableMetrics:   config.EnableMetrics,
		metricsInterval: config.MetricsInterval,
		stopCh:          make(chan struct{}),
	}
}

// Start 启动Relay定时任务
//
// 这个方法会阻塞，应该在goroutine中调用
//
// 示例：
//   relay := NewOutboxRelay(db, producer, secureProducer, nil)
//   go relay.Start(context.Background())
//   // ... 业务逻辑 ...
//   relay.Stop() // 优雅关闭
func (r *OutboxRelay) Start(ctx context.Context) {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		logx.Info("OutboxRelay is already running")
		return
	}
	r.running = true
	r.mu.Unlock()

	logx.Infow("OutboxRelay started",
		logx.Field("interval", r.interval),
		logx.Field("batch_size", r.batchSize))

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	// 启动指标收集（如果启用）
	if r.enableMetrics {
		r.wg.Add(1)
		go r.collectMetrics(ctx)
	}

	// 立即执行一次
	r.processOnce(ctx)

	// 定时执行
	for {
		select {
		case <-ticker.C:
			r.processOnce(ctx)
		case <-r.stopCh:
			logx.Info("OutboxRelay stopping...")
			r.mu.Lock()
			r.running = false
			r.mu.Unlock()
			r.wg.Wait() // 等待所有任务完成
			logx.Info("OutboxRelay stopped")
			return
		case <-ctx.Done():
			logx.Info("OutboxRelay context canceled")
			r.Stop()
			return
		}
	}
}

// Stop 停止Relay
func (r *OutboxRelay) Stop() {
	r.mu.RLock()
	if !r.running {
		r.mu.RUnlock()
		return
	}
	r.mu.RUnlock()

	close(r.stopCh)
}

// IsRunning 检查是否正在运行
func (r *OutboxRelay) IsRunning() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.running
}

// processOnce 执行一次消息处理
func (r *OutboxRelay) processOnce(ctx context.Context) {
	startTime := time.Now()

	// 查询待发送消息
	messages, err := r.fetchPendingMessages(ctx)
	if err != nil {
		logx.Errorw("Failed to fetch pending messages", logx.Field("error", err))
		return
	}

	if len(messages) == 0 {
		return // 没有待发送消息
	}

	logx.Infow("Processing outbox messages",
		logx.Field("count", len(messages)),
		logx.Field("duration_ms", time.Since(startTime).Milliseconds()))

	// 逐条处理消息
	successCount := 0
	failedCount := 0
	for _, msg := range messages {
		if err := r.sendMessage(ctx, msg); err != nil {
			r.handleSendFailure(ctx, msg, err)
			failedCount++
		} else {
			r.markAsSent(ctx, msg)
			successCount++
		}
	}

	logx.Infow("Outbox messages processed",
		logx.Field("total", len(messages)),
		logx.Field("success", successCount),
		logx.Field("failed", failedCount),
		logx.Field("duration_ms", time.Since(startTime).Milliseconds()))
}

// fetchPendingMessages 查询待发送消息
func (r *OutboxRelay) fetchPendingMessages(ctx context.Context) ([]*ent.OutboxMessage, error) {
	now := time.Now()

	// 查询条件：
	// 1. send_status = 'pending'
	// 2. next_retry_at IS NULL OR next_retry_at <= NOW()
	// 排序：priority DESC, created_at ASC（高优先级优先，同优先级按创建时间）
	// 限制：batchSize
	messages, err := r.db.OutboxMessage.Query().
		Where(
			outboxmessage.SendStatusEQ("pending"),
			outboxmessage.Or(
				outboxmessage.NextRetryAtIsNil(),
				outboxmessage.NextRetryAtLTE(now),
			),
		).
		Order(ent.Desc(outboxmessage.FieldPriority), ent.Asc(outboxmessage.FieldCreatedAt)).
		Limit(r.batchSize).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to query pending messages: %w", err)
	}

	return messages, nil
}

// sendMessage 发送单条消息到Kafka
func (r *OutboxRelay) sendMessage(ctx context.Context, msg *ent.OutboxMessage) error {
	// 准备消息Headers
	var headers map[string]string
	if msg.MessageHeaders != nil {
		headers = msg.MessageHeaders
	} else {
		headers = make(map[string]string)
	}

	// 转换messageKey为[]byte
	var key []byte
	if msg.MessageKey != "" {
		key = []byte(msg.MessageKey)
	}

	// 发送到Kafka（使用Publish方法）
	err := r.producer.Publish(ctx, msg.Topic, key, msg.MessageValue, headers)
	if err != nil {
		return fmt.Errorf("failed to send kafka message: %w", err)
	}

	return nil
}

// markAsSent 标记消息为已发送
func (r *OutboxRelay) markAsSent(ctx context.Context, msg *ent.OutboxMessage) {
	now := time.Now()

	err := msg.Update().
		SetSendStatus("sent").
		SetSentAt(now).
		Exec(ctx)

	if err != nil {
		logx.Errorw("Failed to mark message as sent",
			logx.Field("outbox_id", msg.ID),
			logx.Field("tenant_id", msg.TenantID),
			logx.Field("topic", msg.Topic),
			logx.Field("error", err))
		return
	}

	logx.Infow("Outbox message sent successfully",
		logx.Field("outbox_id", msg.ID),
		logx.Field("tenant_id", msg.TenantID),
		logx.Field("topic", msg.Topic),
		logx.Field("event_type", msg.EventType),
		logx.Field("latency_ms", time.Since(msg.CreatedAt).Milliseconds()))
}

// handleSendFailure 处理发送失败
func (r *OutboxRelay) handleSendFailure(ctx context.Context, msg *ent.OutboxMessage, sendErr error) {
	retryCount := msg.RetryCount + 1

	logx.Errorw("Failed to send outbox message",
		logx.Field("outbox_id", msg.ID),
		logx.Field("tenant_id", msg.TenantID),
		logx.Field("topic", msg.Topic),
		logx.Field("retry_count", retryCount),
		logx.Field("max_retries", msg.MaxRetries),
		logx.Field("error", sendErr))

	// 检查是否超过最大重试次数
	if retryCount >= msg.MaxRetries {
		// 路由到DLQ
		failureReason := fmt.Sprintf("Exceeded max retries (%d): %v", msg.MaxRetries, sendErr)
		dlqErr := r.dlqRouter.RouteToDLQ(ctx, msg, failureReason)
		if dlqErr != nil {
			logx.Errorw("Failed to route message to DLQ",
				logx.Field("outbox_id", msg.ID),
				logx.Field("dlq_error", dlqErr),
				logx.Field("original_error", sendErr))
		}

		// 标记为失败
		err := msg.Update().
			SetSendStatus("failed").
			SetRetryCount(retryCount).
			SetLastError(sendErr.Error()).
			SetErrorMessage(failureReason).
			Exec(ctx)

		if err != nil {
			logx.Errorw("Failed to update message status to failed",
				logx.Field("outbox_id", msg.ID),
				logx.Field("error", err))
		} else {
			logx.Errorw("Outbox message marked as failed and routed to DLQ",
				logx.Field("outbox_id", msg.ID),
				logx.Field("tenant_id", msg.TenantID),
				logx.Field("topic", msg.Topic),
				logx.Field("retry_count", retryCount),
				logx.Field("dlq_routed", dlqErr == nil))
		}
		return
	}

	// 计算下次重试时间（指数退避）
	// 重试间隔 = 2^retryCount 秒
	// 例如：1次 → 2秒, 2次 → 4秒, 3次 → 8秒
	backoffSeconds := math.Pow(2, float64(retryCount))
	nextRetryAt := time.Now().Add(time.Duration(backoffSeconds) * time.Second)

	// 更新重试信息
	err := msg.Update().
		SetRetryCount(retryCount).
		SetLastError(sendErr.Error()).
		SetNextRetryAt(nextRetryAt).
		Exec(ctx)

	if err != nil {
		logx.Errorw("Failed to update message retry info",
			logx.Field("outbox_id", msg.ID),
			logx.Field("error", err))
	} else {
		logx.Infow("Scheduled outbox message retry",
			logx.Field("outbox_id", msg.ID),
			logx.Field("tenant_id", msg.TenantID),
			logx.Field("retry_count", retryCount),
			logx.Field("next_retry_at", nextRetryAt),
			logx.Field("backoff_seconds", backoffSeconds))
	}
}

// CleanupSentMessages 清理已发送的旧消息
//
// 参数：
// - ctx: 上下文
// - retentionDays: 保留天数（例如7天）
//
// 返回：
// - int: 删除的消息数量
// - error: 错误信息
//
// 建议定期调用（例如每天一次）
func (r *OutboxRelay) CleanupSentMessages(ctx context.Context, retentionDays int) (int, error) {
	cutoffTime := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)

	deletedCount, err := r.db.OutboxMessage.Delete().
		Where(
			outboxmessage.SendStatusEQ("sent"),
			outboxmessage.CreatedAtLT(cutoffTime),
		).
		Exec(ctx)

	if err != nil {
		return 0, fmt.Errorf("failed to cleanup sent messages: %w", err)
	}

	logx.Infow("Cleaned up sent outbox messages",
		logx.Field("deleted_count", deletedCount),
		logx.Field("retention_days", retentionDays),
		logx.Field("cutoff_time", cutoffTime))

	return deletedCount, nil
}

// GetMetrics 获取Relay指标
func (r *OutboxRelay) GetMetrics(ctx context.Context) (*RelayMetrics, error) {
	metrics := &RelayMetrics{}

	// 统计各状态消息数量
	pendingCount, err := r.db.OutboxMessage.Query().
		Where(outboxmessage.SendStatusEQ("pending")).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	metrics.PendingCount = pendingCount

	sentCount, err := r.db.OutboxMessage.Query().
		Where(outboxmessage.SendStatusEQ("sent")).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	metrics.SentCount = sentCount

	failedCount, err := r.db.OutboxMessage.Query().
		Where(outboxmessage.SendStatusEQ("failed")).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	metrics.FailedCount = failedCount

	// 查询最老的pending消息
	oldestPending, err := r.db.OutboxMessage.Query().
		Where(outboxmessage.SendStatusEQ("pending")).
		Order(ent.Asc(outboxmessage.FieldCreatedAt)).
		First(ctx)
	if err == nil && oldestPending != nil {
		age := time.Since(oldestPending.CreatedAt)
		metrics.OldestPendingAgeSeconds = int(age.Seconds())
	}

	// 计算平均发送延迟（最近1小时）
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	recentSent, err := r.db.OutboxMessage.Query().
		Where(
			outboxmessage.SendStatusEQ("sent"),
			outboxmessage.SentAtGTE(oneHourAgo),
		).
		All(ctx)
	if err == nil && len(recentSent) > 0 {
		totalLatency := int64(0)
		for _, msg := range recentSent {
			if msg.SentAt != nil {
				latency := msg.SentAt.Sub(msg.CreatedAt)
				totalLatency += latency.Milliseconds()
			}
		}
		metrics.AvgSendLatencyMs = int(totalLatency / int64(len(recentSent)))
	}

	return metrics, nil
}

// RelayMetrics Relay指标
type RelayMetrics struct {
	PendingCount            int `json:"pending_count"`              // 待发送消息数量
	SentCount               int `json:"sent_count"`                 // 已发送消息数量
	FailedCount             int `json:"failed_count"`               // 失败消息数量
	OldestPendingAgeSeconds int `json:"oldest_pending_age_seconds"` // 最老待发送消息年龄（秒）
	AvgSendLatencyMs        int `json:"avg_send_latency_ms"`        // 平均发送延迟（毫秒）
}

// collectMetrics 定期收集并打印指标
func (r *OutboxRelay) collectMetrics(ctx context.Context) {
	defer r.wg.Done()

	ticker := time.NewTicker(r.metricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics, err := r.GetMetrics(ctx)
			if err != nil {
				logx.Errorw("Failed to collect metrics", logx.Field("error", err))
				continue
			}

			logx.Infow("OutboxRelay metrics",
				logx.Field("pending_count", metrics.PendingCount),
				logx.Field("sent_count", metrics.SentCount),
				logx.Field("failed_count", metrics.FailedCount),
				logx.Field("oldest_pending_age_seconds", metrics.OldestPendingAgeSeconds),
				logx.Field("avg_send_latency_ms", metrics.AvgSendLatencyMs))

		case <-r.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}
