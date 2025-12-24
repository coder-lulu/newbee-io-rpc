package outbox

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coder-lulu/newbee-io-rpc/ent/dlqmessage"
)

func TestNewDlqRouter(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	tests := []struct {
		name   string
		config *DlqConfig
		check  func(*testing.T, *DlqRouter)
	}{
		{
			name:   "with nil config uses defaults",
			config: nil,
			check: func(t *testing.T, router *DlqRouter) {
				assert.NotNil(t, router)
				assert.NotNil(t, router.config)
				assert.True(t, router.config.Enabled)
				assert.Equal(t, "dlq.", router.config.TopicPrefix)
				assert.False(t, router.config.PublishToKafka)
				assert.Equal(t, 30, router.config.AutoArchiveDays)
				assert.Equal(t, 90, router.config.AutoCleanupDays)
			},
		},
		{
			name: "with custom config",
			config: &DlqConfig{
				Enabled:         false,
				TopicPrefix:     "custom.dlq.",
				PublishToKafka:  true,
				AutoArchiveDays: 7,
				AutoCleanupDays: 14,
			},
			check: func(t *testing.T, router *DlqRouter) {
				assert.NotNil(t, router)
				assert.False(t, router.config.Enabled)
				assert.Equal(t, "custom.dlq.", router.config.TopicPrefix)
				assert.True(t, router.config.PublishToKafka)
				assert.Equal(t, 7, router.config.AutoArchiveDays)
				assert.Equal(t, 14, router.config.AutoCleanupDays)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewDlqRouter(client, tt.config)
			tt.check(t, router)
		})
	}
}

func TestRouteToDLQ(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	router := NewDlqRouter(client, nil)
	ctx := context.Background()

	// 创建一个Outbox消息
	outboxMsg, err := client.OutboxMessage.Create().
		SetTenantID(1).
		SetAggregateType("InputTask").
		SetAggregateID("task-123").
		SetTopic("io.input.jobs").
		SetMessageKey("task-123").
		SetMessageValue([]byte(`{"task_id": 123, "name": "test"}`)).
		SetMessageHeaders(map[string]string{
			"X-Tenant-ID": "1",
			"X-Event-ID":  "evt-123",
		}).
		SetEventType("TaskCreated").
		SetSendStatus("pending").
		SetRetryCount(3).
		SetMaxRetries(3).
		SetPriority(5).
		Save(ctx)
	require.NoError(t, err)

	// 路由到DLQ
	failureReason := "Exceeded max retries (3): kafka connection timeout"
	err = router.RouteToDLQ(ctx, outboxMsg, failureReason)
	require.NoError(t, err)

	// 验证DLQ消息
	dlqMessages, err := client.DlqMessage.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, dlqMessages, 1)

	dlqMsg := dlqMessages[0]
	assert.Equal(t, uint64(1), dlqMsg.TenantID)
	assert.Equal(t, outboxMsg.ID, dlqMsg.OriginalMessageID)
	assert.Equal(t, "InputTask", dlqMsg.AggregateType)
	assert.Equal(t, "task-123", dlqMsg.AggregateID)
	assert.Equal(t, "dlq.io.input.jobs", dlqMsg.Topic) // 带前缀
	assert.Equal(t, "task-123", dlqMsg.MessageKey)
	assert.Equal(t, outboxMsg.MessageValue, dlqMsg.MessageValue)
	assert.Equal(t, "TaskCreated", dlqMsg.EventType)
	assert.Equal(t, 3, dlqMsg.RetryCount)
	assert.Equal(t, failureReason, dlqMsg.FailureReason)
	assert.Equal(t, dlqmessage.StatusPending, dlqMsg.Status)
	assert.NotNil(t, dlqMsg.FailedAt)

	// 验证metadata包含DLQ路由信息
	assert.NotNil(t, dlqMsg.Metadata)
	assert.Contains(t, dlqMsg.Metadata, "dlq_routed_at")
	assert.Contains(t, dlqMsg.Metadata, "original_message_id")

	// 验证指标
	metrics := router.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalRouted)
	assert.Equal(t, int64(0), metrics.RouteFailed)
}

func TestRouteToDLQ_Disabled(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	// 创建禁用DLQ的路由器
	router := NewDlqRouter(client, &DlqConfig{
		Enabled: false,
	})
	ctx := context.Background()

	// 创建Outbox消息
	outboxMsg, err := client.OutboxMessage.Create().
		SetTenantID(1).
		SetAggregateType("InputTask").
		SetAggregateID("task-456").
		SetTopic("io.input.jobs").
		SetMessageValue([]byte(`{}`)).
		SetEventType("TaskCreated").
		SetSendStatus("pending").
		SetRetryCount(3).
		SetMaxRetries(3).
		Save(ctx)
	require.NoError(t, err)

	// 尝试路由到DLQ（应该跳过）
	err = router.RouteToDLQ(ctx, outboxMsg, "test failure")
	require.NoError(t, err)

	// 验证没有创建DLQ消息
	count, err := client.DlqMessage.Query().Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// 指标应该为0
	metrics := router.GetMetrics()
	assert.Equal(t, int64(0), metrics.TotalRouted)
}

func TestRequeueToOutbox(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	router := NewDlqRouter(client, nil)
	ctx := context.Background()

	// 创建DLQ消息
	dlqMsg, err := client.DlqMessage.Create().
		SetTenantID(1).
		SetOriginalMessageID(999).
		SetAggregateType("InputTask").
		SetAggregateID("task-requeue").
		SetTopic("dlq.io.input.jobs").
		SetMessageKey("task-requeue").
		SetMessageValue([]byte(`{"task_id": 999, "name": "requeue test"}`)).
		SetMessageHeaders(map[string]string{
			"X-Tenant-ID": "1",
		}).
		SetEventType("TaskCreated").
		SetRetryCount(3).
		SetFailureReason("Original failure").
		SetFailedAt(time.Now()).
		SetStatus(dlqmessage.StatusPending).
		Save(ctx)
	require.NoError(t, err)

	// 重新入队
	outboxMsg, err := router.RequeueToOutbox(ctx, dlqMsg.ID)
	require.NoError(t, err)
	require.NotNil(t, outboxMsg)

	// 验证新的Outbox消息
	assert.Equal(t, uint64(1), outboxMsg.TenantID)
	assert.Equal(t, "InputTask", outboxMsg.AggregateType)
	assert.Equal(t, "task-requeue", outboxMsg.AggregateID)
	assert.Equal(t, "io.input.jobs", outboxMsg.Topic) // 去掉了dlq.前缀
	assert.Equal(t, "task-requeue", outboxMsg.MessageKey)
	assert.Equal(t, "pending", outboxMsg.SendStatus)
	assert.Equal(t, 0, outboxMsg.RetryCount) // 重置为0

	// 验证metadata包含重新入队信息
	assert.NotNil(t, outboxMsg.Metadata)
	assert.Equal(t, true, outboxMsg.Metadata["requeued_from_dlq"])
	assert.Contains(t, outboxMsg.Metadata, "requeued_at")
	assert.Equal(t, uint64(dlqMsg.ID), outboxMsg.Metadata["dlq_message_id"])

	// 验证DLQ消息状态更新
	updatedDlqMsg, err := client.DlqMessage.Get(ctx, dlqMsg.ID)
	require.NoError(t, err)
	assert.Equal(t, dlqmessage.StatusResolved, updatedDlqMsg.Status)
	assert.NotNil(t, updatedDlqMsg.RequeuedAt)
	assert.Equal(t, outboxMsg.ID, updatedDlqMsg.RequeuedMessageID)
	assert.Equal(t, "Manually requeued to Outbox", updatedDlqMsg.ResolutionNotes)

	// 验证指标
	metrics := router.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalResolved)
}

func TestRequeueToOutbox_AlreadyResolved(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	router := NewDlqRouter(client, nil)
	ctx := context.Background()

	// 创建已resolved的DLQ消息
	dlqMsg, err := client.DlqMessage.Create().
		SetTenantID(1).
		SetOriginalMessageID(888).
		SetAggregateType("InputTask").
		SetAggregateID("task-resolved").
		SetTopic("dlq.io.input.jobs").
		SetMessageValue([]byte(`{}`)).
		SetEventType("TaskCreated").
		SetRetryCount(3).
		SetFailureReason("Test").
		SetFailedAt(time.Now()).
		SetStatus(dlqmessage.StatusResolved). // 已resolved
		SetRequeuedAt(time.Now()).
		SetRequeuedMessageID(100).
		Save(ctx)
	require.NoError(t, err)

	// 尝试重新入队（应该失败）
	_, err = router.RequeueToOutbox(ctx, dlqMsg.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already resolved")
}

func TestRequeueToOutbox_Archived(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	router := NewDlqRouter(client, nil)
	ctx := context.Background()

	// 创建已archived的DLQ消息
	dlqMsg, err := client.DlqMessage.Create().
		SetTenantID(1).
		SetOriginalMessageID(777).
		SetAggregateType("InputTask").
		SetAggregateID("task-archived").
		SetTopic("dlq.io.input.jobs").
		SetMessageValue([]byte(`{}`)).
		SetEventType("TaskCreated").
		SetRetryCount(3).
		SetFailureReason("Test").
		SetFailedAt(time.Now()).
		SetStatus(dlqmessage.StatusArchived). // 已archived
		SetArchivedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	// 尝试重新入队（应该失败）
	_, err = router.RequeueToOutbox(ctx, dlqMsg.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "archived")
}

func TestGetPendingMessages(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	router := NewDlqRouter(client, nil)
	ctx := context.Background()

	// 创建多个DLQ消息（不同状态）
	now := time.Now()

	// Pending消息1（最早）
	_, err := client.DlqMessage.Create().
		SetTenantID(1).
		SetOriginalMessageID(1).
		SetAggregateType("InputTask").
		SetAggregateID("task-1").
		SetTopic("dlq.io.input.jobs").
		SetMessageValue([]byte(`{}`)).
		SetEventType("TaskCreated").
		SetRetryCount(3).
		SetFailureReason("Test").
		SetFailedAt(now.Add(-3 * time.Hour)).
		SetStatus(dlqmessage.StatusPending).
		Save(ctx)
	require.NoError(t, err)

	// Pending消息2
	_, err = client.DlqMessage.Create().
		SetTenantID(1).
		SetOriginalMessageID(2).
		SetAggregateType("InputTask").
		SetAggregateID("task-2").
		SetTopic("dlq.io.input.jobs").
		SetMessageValue([]byte(`{}`)).
		SetEventType("TaskCreated").
		SetRetryCount(3).
		SetFailureReason("Test").
		SetFailedAt(now.Add(-1 * time.Hour)).
		SetStatus(dlqmessage.StatusPending).
		Save(ctx)
	require.NoError(t, err)

	// Resolved消息（不应该返回）
	_, err = client.DlqMessage.Create().
		SetTenantID(1).
		SetOriginalMessageID(3).
		SetAggregateType("InputTask").
		SetAggregateID("task-3").
		SetTopic("dlq.io.input.jobs").
		SetMessageValue([]byte(`{}`)).
		SetEventType("TaskCreated").
		SetRetryCount(3).
		SetFailureReason("Test").
		SetFailedAt(now).
		SetStatus(dlqmessage.StatusResolved).
		Save(ctx)
	require.NoError(t, err)

	// 查询待处理消息
	messages, err := router.GetPendingMessages(ctx, 1, 10)
	require.NoError(t, err)
	assert.Len(t, messages, 2)

	// 验证按failed_at升序排列（最早的在前）
	assert.Equal(t, "task-1", messages[0].AggregateID)
	assert.Equal(t, "task-2", messages[1].AggregateID)
}

func TestGetPendingMessages_TenantIsolation(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	router := NewDlqRouter(client, nil)
	ctx := context.Background()

	// 租户1的消息
	_, err := client.DlqMessage.Create().
		SetTenantID(1).
		SetOriginalMessageID(1).
		SetAggregateType("InputTask").
		SetAggregateID("tenant1-task").
		SetTopic("dlq.io.input.jobs").
		SetMessageValue([]byte(`{}`)).
		SetEventType("TaskCreated").
		SetRetryCount(3).
		SetFailureReason("Test").
		SetFailedAt(time.Now()).
		SetStatus(dlqmessage.StatusPending).
		Save(ctx)
	require.NoError(t, err)

	// 租户2的消息
	_, err = client.DlqMessage.Create().
		SetTenantID(2).
		SetOriginalMessageID(2).
		SetAggregateType("InputTask").
		SetAggregateID("tenant2-task").
		SetTopic("dlq.io.input.jobs").
		SetMessageValue([]byte(`{}`)).
		SetEventType("TaskCreated").
		SetRetryCount(3).
		SetFailureReason("Test").
		SetFailedAt(time.Now()).
		SetStatus(dlqmessage.StatusPending).
		Save(ctx)
	require.NoError(t, err)

	// 查询租户1的消息
	messages, err := router.GetPendingMessages(ctx, 1, 10)
	require.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, "tenant1-task", messages[0].AggregateID)
	assert.Equal(t, uint64(1), messages[0].TenantID)

	// 查询租户2的消息
	messages, err = router.GetPendingMessages(ctx, 2, 10)
	require.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, "tenant2-task", messages[0].AggregateID)
	assert.Equal(t, uint64(2), messages[0].TenantID)
}

func TestArchiveResolvedMessages(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	router := NewDlqRouter(client, nil)
	ctx := context.Background()

	now := time.Now()

	// 创建旧的resolved消息（35天前）
	_, err := client.DlqMessage.Create().
		SetTenantID(1).
		SetOriginalMessageID(1).
		SetAggregateType("InputTask").
		SetAggregateID("old-resolved").
		SetTopic("dlq.io.input.jobs").
		SetMessageValue([]byte(`{}`)).
		SetEventType("TaskCreated").
		SetRetryCount(3).
		SetFailureReason("Test").
		SetFailedAt(now.Add(-40 * 24 * time.Hour)).
		SetStatus(dlqmessage.StatusResolved).
		SetRequeuedAt(now.Add(-35 * 24 * time.Hour)). // 35天前重新入队
		SetRequeuedMessageID(100).
		Save(ctx)
	require.NoError(t, err)

	// 创建新的resolved消息（15天前）
	_, err = client.DlqMessage.Create().
		SetTenantID(1).
		SetOriginalMessageID(2).
		SetAggregateType("InputTask").
		SetAggregateID("new-resolved").
		SetTopic("dlq.io.input.jobs").
		SetMessageValue([]byte(`{}`)).
		SetEventType("TaskCreated").
		SetRetryCount(3).
		SetFailureReason("Test").
		SetFailedAt(now.Add(-20 * 24 * time.Hour)).
		SetStatus(dlqmessage.StatusResolved).
		SetRequeuedAt(now.Add(-15 * 24 * time.Hour)). // 15天前重新入队
		SetRequeuedMessageID(101).
		Save(ctx)
	require.NoError(t, err)

	// 创建pending消息（不应该被归档）
	_, err = client.DlqMessage.Create().
		SetTenantID(1).
		SetOriginalMessageID(3).
		SetAggregateType("InputTask").
		SetAggregateID("pending-msg").
		SetTopic("dlq.io.input.jobs").
		SetMessageValue([]byte(`{}`)).
		SetEventType("TaskCreated").
		SetRetryCount(3).
		SetFailureReason("Test").
		SetFailedAt(now.Add(-40 * 24 * time.Hour)).
		SetStatus(dlqmessage.StatusPending).
		Save(ctx)
	require.NoError(t, err)

	// 归档30天前的resolved消息
	count, err := router.ArchiveResolvedMessages(ctx, 30)
	require.NoError(t, err)
	assert.Equal(t, 1, count) // 只有old-resolved被归档

	// 验证old-resolved已归档
	oldMsg, err := client.DlqMessage.Query().
		Where(dlqmessage.AggregateIDEQ("old-resolved")).
		Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, dlqmessage.StatusArchived, oldMsg.Status)
	assert.NotNil(t, oldMsg.ArchivedAt)

	// 验证new-resolved仍然是resolved
	newMsg, err := client.DlqMessage.Query().
		Where(dlqmessage.AggregateIDEQ("new-resolved")).
		Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, dlqmessage.StatusResolved, newMsg.Status)

	// 验证pending消息未被影响
	pendingMsg, err := client.DlqMessage.Query().
		Where(dlqmessage.AggregateIDEQ("pending-msg")).
		Only(ctx)
	require.NoError(t, err)
	assert.Equal(t, dlqmessage.StatusPending, pendingMsg.Status)

	// 验证指标
	metrics := router.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalArchived)
}

func TestDeleteArchivedMessages(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	router := NewDlqRouter(client, nil)
	ctx := context.Background()

	now := time.Now()

	// 创建旧的archived消息（100天前归档）
	_, err := client.DlqMessage.Create().
		SetTenantID(1).
		SetOriginalMessageID(1).
		SetAggregateType("InputTask").
		SetAggregateID("old-archived").
		SetTopic("dlq.io.input.jobs").
		SetMessageValue([]byte(`{}`)).
		SetEventType("TaskCreated").
		SetRetryCount(3).
		SetFailureReason("Test").
		SetFailedAt(now.Add(-120 * 24 * time.Hour)).
		SetStatus(dlqmessage.StatusArchived).
		SetArchivedAt(now.Add(-100 * 24 * time.Hour)). // 100天前归档
		Save(ctx)
	require.NoError(t, err)

	// 创建新的archived消息（60天前归档）
	_, err = client.DlqMessage.Create().
		SetTenantID(1).
		SetOriginalMessageID(2).
		SetAggregateType("InputTask").
		SetAggregateID("new-archived").
		SetTopic("dlq.io.input.jobs").
		SetMessageValue([]byte(`{}`)).
		SetEventType("TaskCreated").
		SetRetryCount(3).
		SetFailureReason("Test").
		SetFailedAt(now.Add(-80 * 24 * time.Hour)).
		SetStatus(dlqmessage.StatusArchived).
		SetArchivedAt(now.Add(-60 * 24 * time.Hour)). // 60天前归档
		Save(ctx)
	require.NoError(t, err)

	// 创建resolved消息（不应该被删除）
	_, err = client.DlqMessage.Create().
		SetTenantID(1).
		SetOriginalMessageID(3).
		SetAggregateType("InputTask").
		SetAggregateID("resolved-msg").
		SetTopic("dlq.io.input.jobs").
		SetMessageValue([]byte(`{}`)).
		SetEventType("TaskCreated").
		SetRetryCount(3).
		SetFailureReason("Test").
		SetFailedAt(now.Add(-120 * 24 * time.Hour)).
		SetStatus(dlqmessage.StatusResolved).
		Save(ctx)
	require.NoError(t, err)

	// 删除90天前的archived消息
	count, err := router.DeleteArchivedMessages(ctx, 90)
	require.NoError(t, err)
	assert.Equal(t, 1, count) // 只有old-archived被删除

	// 验证old-archived已删除
	exists, err := client.DlqMessage.Query().
		Where(dlqmessage.AggregateIDEQ("old-archived")).
		Exist(ctx)
	require.NoError(t, err)
	assert.False(t, exists)

	// 验证new-archived仍然存在
	exists, err = client.DlqMessage.Query().
		Where(dlqmessage.AggregateIDEQ("new-archived")).
		Exist(ctx)
	require.NoError(t, err)
	assert.True(t, exists)

	// 验证resolved消息未被删除
	exists, err = client.DlqMessage.Query().
		Where(dlqmessage.AggregateIDEQ("resolved-msg")).
		Exist(ctx)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestDlqMetrics(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	router := NewDlqRouter(client, nil)
	ctx := context.Background()

	// 初始指标应该为0
	metrics := router.GetMetrics()
	assert.Equal(t, int64(0), metrics.TotalRouted)
	assert.Equal(t, int64(0), metrics.RouteFailed)
	assert.Equal(t, int64(0), metrics.TotalResolved)
	assert.Equal(t, int64(0), metrics.TotalArchived)

	// 创建Outbox消息并路由到DLQ
	outboxMsg, err := client.OutboxMessage.Create().
		SetTenantID(1).
		SetAggregateType("InputTask").
		SetAggregateID("metrics-test").
		SetTopic("io.input.jobs").
		SetMessageValue([]byte(`{}`)).
		SetEventType("TaskCreated").
		SetSendStatus("pending").
		SetRetryCount(3).
		SetMaxRetries(3).
		Save(ctx)
	require.NoError(t, err)

	err = router.RouteToDLQ(ctx, outboxMsg, "test failure")
	require.NoError(t, err)

	// 验证TotalRouted增加
	metrics = router.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalRouted)

	// 获取DLQ消息ID
	dlqMsg, err := client.DlqMessage.Query().First(ctx)
	require.NoError(t, err)

	// 重新入队
	_, err = router.RequeueToOutbox(ctx, dlqMsg.ID)
	require.NoError(t, err)

	// 验证TotalResolved增加
	metrics = router.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalResolved)
}

func TestBuildDlqTopic(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	tests := []struct {
		name           string
		topicPrefix    string
		originalTopic  string
		expectedDlqTopic string
	}{
		{
			name:           "default prefix",
			topicPrefix:    "dlq.",
			originalTopic:  "io.input.jobs",
			expectedDlqTopic: "dlq.io.input.jobs",
		},
		{
			name:           "custom prefix",
			topicPrefix:    "dead.letter.",
			originalTopic:  "io.tasks",
			expectedDlqTopic: "dead.letter.io.tasks",
		},
		{
			name:           "empty prefix",
			topicPrefix:    "",
			originalTopic:  "io.events",
			expectedDlqTopic: "io.events",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewDlqRouter(client, &DlqConfig{
				TopicPrefix: tt.topicPrefix,
			})

			dlqTopic := router.buildDlqTopic(tt.originalTopic)
			assert.Equal(t, tt.expectedDlqTopic, dlqTopic)
		})
	}
}

func TestExtractOriginalTopic(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	tests := []struct {
		name           string
		topicPrefix    string
		dlqTopic       string
		expectedOriginal string
	}{
		{
			name:           "default prefix",
			topicPrefix:    "dlq.",
			dlqTopic:       "dlq.io.input.jobs",
			expectedOriginal: "io.input.jobs",
		},
		{
			name:           "custom prefix",
			topicPrefix:    "dead.letter.",
			dlqTopic:       "dead.letter.io.tasks",
			expectedOriginal: "io.tasks",
		},
		{
			name:           "no prefix match",
			topicPrefix:    "dlq.",
			dlqTopic:       "io.events",
			expectedOriginal: "io.events",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewDlqRouter(client, &DlqConfig{
				TopicPrefix: tt.topicPrefix,
			})

			originalTopic := router.extractOriginalTopic(tt.dlqTopic)
			assert.Equal(t, tt.expectedOriginal, originalTopic)
		})
	}
}

// 测试metadata和headers的序列化
func TestRouteToDLQ_MetadataAndHeaders(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	router := NewDlqRouter(client, nil)
	ctx := context.Background()

	// 创建带复杂metadata和headers的Outbox消息
	metadata := map[string]interface{}{
		"user_id":        "user-123",
		"correlation_id": "corr-456",
		"source":         "api",
		"version":        1.0,
	}

	headers := map[string]string{
		"X-Tenant-ID":       "1",
		"X-Correlation-ID":  "corr-456",
		"X-Source-Service":  "unified-io",
	}

	outboxMsg, err := client.OutboxMessage.Create().
		SetTenantID(1).
		SetAggregateType("InputTask").
		SetAggregateID("task-metadata").
		SetTopic("io.input.jobs").
		SetMessageValue([]byte(`{"test": "data"}`)).
		SetMessageHeaders(headers).
		SetEventType("TaskCreated").
		SetMetadata(metadata).
		SetSendStatus("pending").
		SetRetryCount(3).
		SetMaxRetries(3).
		Save(ctx)
	require.NoError(t, err)

	// 路由到DLQ
	err = router.RouteToDLQ(ctx, outboxMsg, "test failure")
	require.NoError(t, err)

	// 验证DLQ消息
	dlqMsg, err := client.DlqMessage.Query().First(ctx)
	require.NoError(t, err)

	// 验证headers完整复制
	assert.NotNil(t, dlqMsg.MessageHeaders)
	assert.Equal(t, "1", dlqMsg.MessageHeaders["X-Tenant-ID"])
	assert.Equal(t, "corr-456", dlqMsg.MessageHeaders["X-Correlation-ID"])
	assert.Equal(t, "unified-io", dlqMsg.MessageHeaders["X-Source-Service"])

	// 验证metadata包含原始数据+DLQ路由信息
	assert.NotNil(t, dlqMsg.Metadata)
	assert.Equal(t, "user-123", dlqMsg.Metadata["user_id"])
	assert.Equal(t, "corr-456", dlqMsg.Metadata["correlation_id"])
	assert.Equal(t, "api", dlqMsg.Metadata["source"])
	assert.Contains(t, dlqMsg.Metadata, "dlq_routed_at")
	assert.Equal(t, float64(outboxMsg.ID), dlqMsg.Metadata["original_message_id"])
}

// 测试重新入队时metadata的传递
func TestRequeueToOutbox_MetadataPreservation(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	router := NewDlqRouter(client, nil)
	ctx := context.Background()

	// 创建带metadata的DLQ消息
	originalMetadata := map[string]interface{}{
		"user_id":      "user-999",
		"request_id":   "req-888",
		"dlq_routed_at": "2025-10-22T10:00:00Z",
	}

	dlqMsg, err := client.DlqMessage.Create().
		SetTenantID(1).
		SetOriginalMessageID(777).
		SetAggregateType("InputTask").
		SetAggregateID("task-preserve").
		SetTopic("dlq.io.input.jobs").
		SetMessageKey("preserve-test").
		SetMessageValue([]byte(`{"data": "preserved"}`)).
		SetEventType("TaskCreated").
		SetMetadata(originalMetadata).
		SetRetryCount(3).
		SetFailureReason("Original failure").
		SetFailedAt(time.Now()).
		SetStatus(dlqmessage.StatusPending).
		Save(ctx)
	require.NoError(t, err)

	// 重新入队
	outboxMsg, err := router.RequeueToOutbox(ctx, dlqMsg.ID)
	require.NoError(t, err)

	// 验证新Outbox消息的metadata
	assert.NotNil(t, outboxMsg.Metadata)

	// 原始metadata应该保留
	assert.Equal(t, "user-999", outboxMsg.Metadata["user_id"])
	assert.Equal(t, "req-888", outboxMsg.Metadata["request_id"])

	// DLQ重新入队信息应该添加
	assert.Equal(t, true, outboxMsg.Metadata["requeued_from_dlq"])
	assert.Contains(t, outboxMsg.Metadata, "requeued_at")
	assert.Equal(t, uint64(dlqMsg.ID), outboxMsg.Metadata["dlq_message_id"])
}

func TestRequeueToOutbox_NonExistentMessage(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	router := NewDlqRouter(client, nil)
	ctx := context.Background()

	// 尝试重新入队不存在的DLQ消息
	_, err := router.RequeueToOutbox(ctx, 99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get DLQ message")
}

// 测试limit参数
func TestGetPendingMessages_Limit(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	router := NewDlqRouter(client, nil)
	ctx := context.Background()

	// 创建10个pending消息
	for i := 1; i <= 10; i++ {
		_, err := client.DlqMessage.Create().
			SetTenantID(1).
			SetOriginalMessageID(uint64(i)).
			SetAggregateType("InputTask").
			SetAggregateID(fmt.Sprintf("task-%d", i)).
			SetTopic("dlq.io.input.jobs").
			SetMessageValue([]byte(`{}`)).
			SetEventType("TaskCreated").
			SetRetryCount(3).
			SetFailureReason("Test").
			SetFailedAt(time.Now().Add(time.Duration(-i) * time.Hour)).
			SetStatus(dlqmessage.StatusPending).
			Save(ctx)
		require.NoError(t, err)
	}

	// 测试limit=5
	messages, err := router.GetPendingMessages(ctx, 1, 5)
	require.NoError(t, err)
	assert.Len(t, messages, 5)

	// 测试limit=20（实际只有10个）
	messages, err = router.GetPendingMessages(ctx, 1, 20)
	require.NoError(t, err)
	assert.Len(t, messages, 10)
}
