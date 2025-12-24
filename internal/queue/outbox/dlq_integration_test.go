package outbox

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/ent/dlqmessage"
)

// TestIntegration_DLQ_MaxRetriesExceeded 测试超过最大重试次数后路由到DLQ
func TestIntegration_DLQ_MaxRetriesExceeded(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ctx := context.Background()
	publisher := NewOutboxPublisher(client)

	// 创建失败的Producer（总是失败）
	mockProd := &mockProducer{
		shouldFail:   true,
		failureCount: 100, // 前100次都失败
	}

	// 创建Relay（max_retries=2，方便测试）
	relay := NewOutboxRelay(client, mockProd, nil, &RelayConfig{
		Interval:  100 * time.Millisecond,
		BatchSize: 10,
		DlqConfig: &DlqConfig{
			Enabled:     true,
			TopicPrefix: "dlq.",
		},
	})

	// 保存消息到Outbox
	err := withTx(ctx, client, func(ctx context.Context, tx *ent.Tx) error {
		msg := &OutboxMessage{
			TenantID:      1,
			AggregateType: "InputTask",
			AggregateID:   "task-dlq-test",
			Topic:         "io.input.jobs",
			MessageKey:    "task-dlq-test",
			MessageValue:  []byte(`{"task_id": 9999}`),
			EventType:     "TaskCreated",
			MaxRetries:    2, // 最多重试2次
		}
		_, err := publisher.SaveToOutbox(ctx, tx, msg)
		return err
	})
	require.NoError(t, err)

	// 处理一次（第1次失败，retry_count=1）
	relay.processOnce(ctx)

	// 验证消息仍然是pending状态，retry_count=1
	outboxMsg1, err := client.OutboxMessage.Query().First(ctx)
	require.NoError(t, err)
	assert.Equal(t, "pending", outboxMsg1.SendStatus)
	assert.Equal(t, 1, outboxMsg1.RetryCount)

	// 处理第二次（第2次失败，retry_count=2）
	relay.processOnce(ctx)

	outboxMsg2, err := client.OutboxMessage.Query().First(ctx)
	require.NoError(t, err)
	assert.Equal(t, "pending", outboxMsg2.SendStatus)
	assert.Equal(t, 2, outboxMsg2.RetryCount)

	// 处理第三次（第3次失败，retry_count=3，达到max_retries=2，应该路由到DLQ）
	relay.processOnce(ctx)

	// 验证Outbox消息标记为failed
	outboxMsg3, err := client.OutboxMessage.Query().First(ctx)
	require.NoError(t, err)
	assert.Equal(t, "failed", outboxMsg3.SendStatus)
	assert.Equal(t, 3, outboxMsg3.RetryCount) // 实际失败时retry_count已经是3

	// 验证DLQ消息已创建
	dlqMessages, err := client.DlqMessage.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, dlqMessages, 1)

	dlqMsg := dlqMessages[0]
	assert.Equal(t, uint64(1), dlqMsg.TenantID)
	assert.Equal(t, "InputTask", dlqMsg.AggregateType)
	assert.Equal(t, "task-dlq-test", dlqMsg.AggregateID)
	assert.Equal(t, "dlq.io.input.jobs", dlqMsg.Topic)
	assert.Equal(t, dlqmessage.StatusPending, dlqMsg.Status)
	assert.Contains(t, dlqMsg.FailureReason, "Exceeded max retries")
	assert.Equal(t, 3, dlqMsg.RetryCount) // 记录失败时的retry_count
}

// TestIntegration_DLQ_EndToEnd 测试完整的DLQ流程：失败→DLQ→重新入队→成功
func TestIntegration_DLQ_EndToEnd(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ctx := context.Background()
	publisher := NewOutboxPublisher(client)

	// 创建失败的Producer（前3次失败）
	mockProd := &mockProducer{
		shouldFail:   true,
		failureCount: 3, // 前3次失败，第4次成功
	}

	// 创建Relay
	relay := NewOutboxRelay(client, mockProd, nil, &RelayConfig{
		Interval:  100 * time.Millisecond,
		BatchSize: 10,
		DlqConfig: &DlqConfig{
			Enabled:     true,
			TopicPrefix: "dlq.",
		},
	})

	// 保存消息到Outbox
	err := withTx(ctx, client, func(ctx context.Context, tx *ent.Tx) error {
		msg := &OutboxMessage{
			TenantID:      1,
			AggregateType: "InputTask",
			AggregateID:   "task-e2e",
			Topic:         "io.input.jobs",
			MessageKey:    "task-e2e",
			MessageValue:  []byte(`{"task_id": 8888}`),
			EventType:     "TaskCreated",
			MaxRetries:    3,
		}
		_, err := publisher.SaveToOutbox(ctx, tx, msg)
		return err
	})
	require.NoError(t, err)

	// 处理3次，应该失败并路由到DLQ
	for i := 0; i < 4; i++ {
		relay.processOnce(ctx)
	}

	// 验证Outbox消息失败
	outboxMsg, err := client.OutboxMessage.Query().First(ctx)
	require.NoError(t, err)
	assert.Equal(t, "failed", outboxMsg.SendStatus)

	// 验证DLQ消息创建
	dlqMsg, err := client.DlqMessage.Query().First(ctx)
	require.NoError(t, err)
	assert.Equal(t, dlqmessage.StatusPending, dlqMsg.Status)

	// 🔥 重新入队：将DLQ消息重新放回Outbox
	dlqRouter := relay.dlqRouter
	newOutboxMsg, err := dlqRouter.RequeueToOutbox(ctx, dlqMsg.ID)
	require.NoError(t, err)
	require.NotNil(t, newOutboxMsg)

	// 验证新Outbox消息
	assert.Equal(t, "pending", newOutboxMsg.SendStatus)
	assert.Equal(t, 0, newOutboxMsg.RetryCount) // 重置为0
	assert.Equal(t, "task-e2e", newOutboxMsg.AggregateID)

	// 验证DLQ消息状态更新为resolved
	updatedDlqMsg, err := client.DlqMessage.Get(ctx, dlqMsg.ID)
	require.NoError(t, err)
	assert.Equal(t, dlqmessage.StatusResolved, updatedDlqMsg.Status)
	assert.NotNil(t, updatedDlqMsg.RequeuedAt)

	// 🔥 处理新Outbox消息（第4次，应该成功）
	relay.processOnce(ctx)

	// 验证新消息发送成功
	finalMsg, err := client.OutboxMessage.Get(ctx, newOutboxMsg.ID)
	require.NoError(t, err)
	assert.Equal(t, "sent", finalMsg.SendStatus)

	// 验证Kafka收到消息
	assert.Len(t, mockProd.messages, 1)
	assert.Equal(t, "io.input.jobs", mockProd.messages[0].topic)
	assert.Equal(t, "task-e2e", mockProd.messages[0].key)
}

// TestIntegration_DLQ_TenantIsolation 测试DLQ租户隔离
func TestIntegration_DLQ_TenantIsolation(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ctx := context.Background()
	publisher := NewOutboxPublisher(client)

	// 创建失败的Producer
	mockProd := &mockProducer{
		shouldFail:   true,
		failureCount: 100, // 前100次都失败
	}

	// 创建Relay
	relay := NewOutboxRelay(client, mockProd, nil, &RelayConfig{
		Interval:  100 * time.Millisecond,
		BatchSize: 10,
		DlqConfig: &DlqConfig{
			Enabled:     true,
			TopicPrefix: "dlq.",
		},
	})

	// 租户1的消息
	err := withTx(ctx, client, func(ctx context.Context, tx *ent.Tx) error {
		msg := &OutboxMessage{
			TenantID:      1,
			AggregateType: "InputTask",
			AggregateID:   "tenant1-task",
			Topic:         "io.input.jobs",
			MessageKey:    "tenant1-task",
			MessageValue:  []byte(`{}`),
			EventType:     "TaskCreated",
			MaxRetries:    1,
		}
		_, err := publisher.SaveToOutbox(ctx, tx, msg)
		return err
	})
	require.NoError(t, err)

	// 租户2的消息
	err = withTx(ctx, client, func(ctx context.Context, tx *ent.Tx) error {
		msg := &OutboxMessage{
			TenantID:      2,
			AggregateType: "InputTask",
			AggregateID:   "tenant2-task",
			Topic:         "io.input.jobs",
			MessageKey:    "tenant2-task",
			MessageValue:  []byte(`{}`),
			EventType:     "TaskCreated",
			MaxRetries:    1,
		}
		_, err := publisher.SaveToOutbox(ctx, tx, msg)
		return err
	})
	require.NoError(t, err)

	// 处理消息，都应该失败并路由到DLQ
	for i := 0; i < 3; i++ {
		relay.processOnce(ctx)
	}

	// 验证两个DLQ消息
	dlqMessages, err := client.DlqMessage.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, dlqMessages, 2)

	// 验证租户隔离：查询租户1的DLQ消息
	dlqRouter := relay.dlqRouter
	tenant1Messages, err := dlqRouter.GetPendingMessages(ctx, 1, 10)
	require.NoError(t, err)
	require.Len(t, tenant1Messages, 1)
	assert.Equal(t, "tenant1-task", tenant1Messages[0].AggregateID)

	// 验证租户隔离：查询租户2的DLQ消息
	tenant2Messages, err := dlqRouter.GetPendingMessages(ctx, 2, 10)
	require.NoError(t, err)
	require.Len(t, tenant2Messages, 1)
	assert.Equal(t, "tenant2-task", tenant2Messages[0].AggregateID)
}

// TestIntegration_DLQ_ArchiveAndCleanup 测试DLQ归档和清理
func TestIntegration_DLQ_ArchiveAndCleanup(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ctx := context.Background()
	dlqRouter := NewDlqRouter(client, nil)

	// 创建一个旧的resolved DLQ消息（40天前）
	oldTime := time.Now().Add(-40 * 24 * time.Hour)
	_, err := client.DlqMessage.Create().
		SetTenantID(1).
		SetOriginalMessageID(1).
		SetAggregateType("InputTask").
		SetAggregateID("old-task").
		SetTopic("dlq.io.input.jobs").
		SetMessageValue([]byte(`{}`)).
		SetEventType("TaskCreated").
		SetRetryCount(3).
		SetFailureReason("Test").
		SetFailedAt(oldTime).
		SetStatus(dlqmessage.StatusResolved).
		SetRequeuedAt(oldTime).
		SetRequeuedMessageID(100).
		Save(ctx)
	require.NoError(t, err)

	// 归档30天前的resolved消息
	count, err := dlqRouter.ArchiveResolvedMessages(ctx, 30)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// 验证消息已归档
	archivedMsg, err := client.DlqMessage.Query().
		Where(dlqmessage.AggregateIDEQ("old-task")).
		Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, dlqmessage.StatusArchived, archivedMsg.Status)
	assert.NotNil(t, archivedMsg.ArchivedAt)

	// 等待1秒，确保archived_at时间与创建时间不同
	time.Sleep(1 * time.Second)

	// 删除已归档消息（只删除归档时间超过1秒的消息，这样可以删除我们刚归档的消息）
	// 注意：在实际使用中，应该使用更长的时间，如90天
	_, err = client.DlqMessage.Update().
		Where(dlqmessage.IDEQ(archivedMsg.ID)).
		SetArchivedAt(time.Now().Add(-100 * 24 * time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	// 删除90天前的archived消息
	deleteCount, err := dlqRouter.DeleteArchivedMessages(ctx, 90)
	require.NoError(t, err)
	assert.Equal(t, 1, deleteCount)

	// 验证消息已删除
	exists, err := client.DlqMessage.Query().
		Where(dlqmessage.AggregateIDEQ("old-task")).
		Exist(ctx)
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestIntegration_DLQ_DisabledConfig 测试DLQ禁用配置
func TestIntegration_DLQ_DisabledConfig(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ctx := context.Background()
	publisher := NewOutboxPublisher(client)

	// 创建失败的Producer
	mockProd := &mockProducer{
		shouldFail:   true,
		failureCount: 0,
	}

	// 创建Relay（禁用DLQ）
	relay := NewOutboxRelay(client, mockProd, nil, &RelayConfig{
		Interval:  100 * time.Millisecond,
		BatchSize: 10,
		DlqConfig: &DlqConfig{
			Enabled: false, // 禁用DLQ
		},
	})

	// 保存消息到Outbox
	err := withTx(ctx, client, func(ctx context.Context, tx *ent.Tx) error {
		msg := &OutboxMessage{
			TenantID:      1,
			AggregateType: "InputTask",
			AggregateID:   "disabled-test",
			Topic:         "io.input.jobs",
			MessageKey:    "disabled-test",
			MessageValue:  []byte(`{}`),
			EventType:     "TaskCreated",
			MaxRetries:    2,
		}
		_, err := publisher.SaveToOutbox(ctx, tx, msg)
		return err
	})
	require.NoError(t, err)

	// 处理消息，应该失败但不路由到DLQ
	for i := 0; i < 4; i++ {
		relay.processOnce(ctx)
	}

	// 验证Outbox消息失败
	outboxMsg, err := client.OutboxMessage.Query().First(ctx)
	require.NoError(t, err)
	assert.Equal(t, "failed", outboxMsg.SendStatus)

	// 验证没有DLQ消息（因为DLQ禁用）
	dlqCount, err := client.DlqMessage.Query().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, dlqCount)
}

// TestIntegration_DLQ_MetricsTracking 测试DLQ指标追踪
func TestIntegration_DLQ_MetricsTracking(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ctx := context.Background()
	publisher := NewOutboxPublisher(client)

	// 创建失败的Producer
	mockProd := &mockProducer{
		shouldFail:   true,
		failureCount: 0,
	}

	// 创建Relay
	relay := NewOutboxRelay(client, mockProd, nil, &RelayConfig{
		Interval:  100 * time.Millisecond,
		BatchSize: 10,
		DlqConfig: &DlqConfig{
			Enabled: true,
		},
	})

	// 初始指标
	metrics := relay.dlqRouter.GetMetrics()
	assert.Equal(t, int64(0), metrics.TotalRouted)
	assert.Equal(t, int64(0), metrics.TotalResolved)

	// 保存2个消息到Outbox
	for i := 0; i < 2; i++ {
		err := withTx(ctx, client, func(ctx context.Context, tx *ent.Tx) error {
			msg := &OutboxMessage{
				TenantID:      1,
				AggregateType: "InputTask",
				AggregateID:   "metrics-task-" + string(rune('0'+i)),
				Topic:         "io.input.jobs",
				MessageValue:  []byte(`{}`),
				EventType:     "TaskCreated",
				MaxRetries:    1,
			}
			_, err := publisher.SaveToOutbox(ctx, tx, msg)
			return err
		})
		require.NoError(t, err)
	}

	// 处理消息，都应该路由到DLQ
	for i := 0; i < 3; i++ {
		relay.processOnce(ctx)
	}

	// 验证TotalRouted指标
	metrics = relay.dlqRouter.GetMetrics()
	assert.Equal(t, int64(2), metrics.TotalRouted)

	// 重新入队第一个DLQ消息
	dlqMessages, err := client.DlqMessage.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, dlqMessages, 2)

	_, err = relay.dlqRouter.RequeueToOutbox(ctx, dlqMessages[0].ID)
	require.NoError(t, err)

	// 验证TotalResolved指标
	metrics = relay.dlqRouter.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalResolved)
}

// TestIntegration_DLQ_CustomTopicPrefix 测试自定义DLQ Topic前缀
func TestIntegration_DLQ_CustomTopicPrefix(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	ctx := context.Background()
	publisher := NewOutboxPublisher(client)

	// 创建失败的Producer
	mockProd := &mockProducer{
		shouldFail:   true,
		failureCount: 0,
	}

	// 创建Relay（自定义Topic前缀）
	relay := NewOutboxRelay(client, mockProd, nil, &RelayConfig{
		Interval:  100 * time.Millisecond,
		BatchSize: 10,
		DlqConfig: &DlqConfig{
			Enabled:     true,
			TopicPrefix: "dead.letter.", // 自定义前缀
		},
	})

	// 保存消息到Outbox
	err := withTx(ctx, client, func(ctx context.Context, tx *ent.Tx) error {
		msg := &OutboxMessage{
			TenantID:      1,
			AggregateType: "InputTask",
			AggregateID:   "custom-prefix-task",
			Topic:         "io.input.jobs",
			MessageValue:  []byte(`{}`),
			EventType:     "TaskCreated",
			MaxRetries:    1,
		}
		_, err := publisher.SaveToOutbox(ctx, tx, msg)
		return err
	})
	require.NoError(t, err)

	// 处理消息，应该路由到DLQ
	for i := 0; i < 3; i++ {
		relay.processOnce(ctx)
	}

	// 验证DLQ消息的Topic前缀
	dlqMsg, err := client.DlqMessage.Query().First(ctx)
	require.NoError(t, err)
	assert.Equal(t, "dead.letter.io.input.jobs", dlqMsg.Topic)

	// 重新入队，验证提取原始Topic
	newOutboxMsg, err := relay.dlqRouter.RequeueToOutbox(ctx, dlqMsg.ID)
	require.NoError(t, err)
	assert.Equal(t, "io.input.jobs", newOutboxMsg.Topic) // 去掉了前缀
}
