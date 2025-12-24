package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/internal/queue/types"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTaskHandler 用于测试的Mock Task Handler
type mockTaskHandler struct {
	messages []*types.TaskMessage
	err      error
}

func (m *mockTaskHandler) HandleTask(ctx context.Context, msg *types.TaskMessage) error {
	m.messages = append(m.messages, msg)
	return m.err
}

// mockIdempotencyGuard 用于测试的Mock幂等性守护者
type mockIdempotencyGuard struct {
	checkedMessages map[string]bool
	shouldFail      bool
}

func newMockIdempotencyGuard() *mockIdempotencyGuard {
	return &mockIdempotencyGuard{
		checkedMessages: make(map[string]bool),
	}
}

func (m *mockIdempotencyGuard) CheckAndMark(ctx context.Context, tenantID uint64, taskRunID uint64) (bool, error) {
	if m.shouldFail {
		return false, assert.AnError
	}

	key := keyForMessage(tenantID, taskRunID)
	if m.checkedMessages[key] {
		return false, nil // 重复消息
	}

	m.checkedMessages[key] = true
	return true, nil // 首次处理
}

func (m *mockIdempotencyGuard) Remove(ctx context.Context, tenantID uint64, taskRunID uint64) error {
	key := keyForMessage(tenantID, taskRunID)
	delete(m.checkedMessages, key)
	return nil
}

func keyForMessage(tenantID uint64, taskRunID uint64) string {
	return fmt.Sprintf("%d:%d", tenantID, taskRunID)
}

// TestSecureConsumer_TenantValidation 测试租户验证
func TestSecureConsumer_TenantValidation(t *testing.T) {
	handler := &mockTaskHandler{}
	idempotencyGuard := newMockIdempotencyGuard()
	secureConsumer := NewSecureConsumer(1, "test-group", handler, idempotencyGuard)

	tests := []struct {
		name          string
		headers       []kafka.Header
		taskMsg       *types.TaskMessage
		expectedError bool
	}{
		{
			name: "valid tenant",
			headers: []kafka.Header{
				{Key: "X-Tenant-ID", Value: []byte("1")},
			},
			taskMsg: &types.TaskMessage{
				TenantID:  1,
				TaskRunID: 100,
				TaskType:  "test",
			},
			expectedError: false,
		},
		{
			name: "header tenant mismatch",
			headers: []kafka.Header{
				{Key: "X-Tenant-ID", Value: []byte("2")},
			},
			taskMsg: &types.TaskMessage{
				TenantID:  1,
				TaskRunID: 101,
				TaskType:  "test",
			},
			expectedError: true,
		},
		{
			name: "missing tenant header",
			headers: []kafka.Header{
				{Key: "X-Trace-ID", Value: []byte("trace-123")},
			},
			taskMsg: &types.TaskMessage{
				TenantID:  1,
				TaskRunID: 102,
				TaskType:  "test",
			},
			expectedError: true,
		},
		{
			name: "body tenant mismatch",
			headers: []kafka.Header{
				{Key: "X-Tenant-ID", Value: []byte("1")},
			},
			taskMsg: &types.TaskMessage{
				TenantID:  2,
				TaskRunID: 103,
				TaskType:  "test",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.taskMsg)
			msg := kafka.Message{
				Headers: tt.headers,
				Value:   body,
			}

			err := secureConsumer.Handle(context.Background(), &msg)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSecureConsumer_IdempotencyCheck 测试幂等性检查
func TestSecureConsumer_IdempotencyCheck(t *testing.T) {
	handler := &mockTaskHandler{}
	idempotencyGuard := newMockIdempotencyGuard()
	secureConsumer := NewSecureConsumer(1, "test-group", handler, idempotencyGuard)

	taskMsg := &types.TaskMessage{
		TenantID:  1,
		TaskRunID: 200,
		TaskType:  "test",
	}

	body, _ := json.Marshal(taskMsg)
	msg := kafka.Message{
		Headers: []kafka.Header{
			{Key: "X-Tenant-ID", Value: []byte("1")},
		},
		Value: body,
	}

	ctx := context.Background()

	// 第一次处理
	err := secureConsumer.Handle(ctx, &msg)
	require.NoError(t, err)
	assert.Len(t, handler.messages, 1)

	// 第二次处理（重复消息）
	err = secureConsumer.Handle(ctx, &msg)
	require.NoError(t, err)
	assert.Len(t, handler.messages, 1) // 应该还是1，因为被幂等性检查拦截了
}

// TestSecureConsumer_MessageProcessing 测试消息处理
func TestSecureConsumer_MessageProcessing(t *testing.T) {
	handler := &mockTaskHandler{}
	idempotencyGuard := newMockIdempotencyGuard()
	secureConsumer := NewSecureConsumer(1, "test-group", handler, idempotencyGuard)

	taskMsg := &types.TaskMessage{
		TenantID:  1,
		TaskRunID: 300,
		TaskType:  "sync_users",
		ProviderID: "ldap",
		UserID:    100,
		CreatedAt: time.Now(),
	}

	body, _ := json.Marshal(taskMsg)
	msg := kafka.Message{
		Headers: []kafka.Header{
			{Key: "X-Tenant-ID", Value: []byte("1")},
			{Key: "X-Trace-ID", Value: []byte("trace-123")},
		},
		Value:     body,
		Topic:     "io.input.jobs",
		Partition: 0,
		Offset:    100,
	}

	err := secureConsumer.Handle(context.Background(), &msg)
	require.NoError(t, err)

	require.Len(t, handler.messages, 1)
	processedMsg := handler.messages[0]

	assert.Equal(t, taskMsg.TenantID, processedMsg.TenantID)
	assert.Equal(t, taskMsg.TaskRunID, processedMsg.TaskRunID)
	assert.Equal(t, taskMsg.TaskType, processedMsg.TaskType)
	assert.Equal(t, taskMsg.ProviderID, processedMsg.ProviderID)
}

// TestSecureConsumer_HandlerError 测试Handler错误处理
func TestSecureConsumer_HandlerError(t *testing.T) {
	handler := &mockTaskHandler{
		err: assert.AnError,
	}
	idempotencyGuard := newMockIdempotencyGuard()
	secureConsumer := NewSecureConsumer(1, "test-group", handler, idempotencyGuard)

	taskMsg := &types.TaskMessage{
		TenantID:  1,
		TaskRunID: 400,
		TaskType:  "test",
	}

	body, _ := json.Marshal(taskMsg)
	msg := kafka.Message{
		Headers: []kafka.Header{
			{Key: "X-Tenant-ID", Value: []byte("1")},
		},
		Value: body,
	}

	err := secureConsumer.Handle(context.Background(), &msg)
	assert.Error(t, err)
	assert.Len(t, handler.messages, 1)
}
