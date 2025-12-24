package outbox

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/ent/enttest"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *ent.Client {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	return client
}

func TestNewOutboxPublisher(t *testing.T) {
	tests := []struct {
		name string
		db   *ent.Client
	}{
		{
			name: "nil database client",
			db:   nil,
		},
		{
			name: "valid database client",
			db:   &ent.Client{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publisher := NewOutboxPublisher(tt.db)
			assert.NotNil(t, publisher)
			assert.Equal(t, tt.db, publisher.db)
		})
	}
}

func TestSaveToOutbox(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	publisher := NewOutboxPublisher(client)
	require.NotNil(t, publisher)

	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func(t *testing.T) (*ent.Tx, *OutboxMessage)
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message with all fields",
			setup: func(t *testing.T) (*ent.Tx, *OutboxMessage) {
				tx, err := client.Tx(ctx)
				require.NoError(t, err)

				msg := &OutboxMessage{
					TenantID:      1,
					AggregateType: "InputTask",
					AggregateID:   "task-123",
					Topic:         "io.tasks",
					MessageKey:    "task-123",
					MessageValue:  []byte(`{"id":"task-123","name":"test"}`),
					MessageHeaders: map[string]string{
						"X-Tenant-ID": "1",
						"X-Event-ID":  "evt-123",
					},
					EventType: "TaskCreated",
					Priority:  8,
					MaxRetries: 5,
					Metadata: map[string]interface{}{
						"user_id": "user-456",
						"source":  "api",
					},
				}
				return tx, msg
			},
			wantErr: false,
		},
		{
			name: "missing tenant_id",
			setup: func(t *testing.T) (*ent.Tx, *OutboxMessage) {
				tx, err := client.Tx(ctx)
				require.NoError(t, err)

				msg := &OutboxMessage{
					AggregateType: "InputTask",
					Topic:         "io.tasks",
					EventType:     "TaskCreated",
				}
				return tx, msg
			},
			wantErr: true,
			errMsg:  "tenant_id is required",
		},
		{
			name: "missing aggregate_type",
			setup: func(t *testing.T) (*ent.Tx, *OutboxMessage) {
				tx, err := client.Tx(ctx)
				require.NoError(t, err)

				msg := &OutboxMessage{
					TenantID:  1,
					Topic:     "io.tasks",
					EventType: "TaskCreated",
				}
				return tx, msg
			},
			wantErr: true,
			errMsg:  "aggregate_type is required",
		},
		{
			name: "missing topic",
			setup: func(t *testing.T) (*ent.Tx, *OutboxMessage) {
				tx, err := client.Tx(ctx)
				require.NoError(t, err)

				msg := &OutboxMessage{
					TenantID:      1,
					AggregateType: "InputTask",
					AggregateID:   "task-123",
					EventType:     "TaskCreated",
					MessageValue:  []byte(`{}`),
				}
				return tx, msg
			},
			wantErr: true,
			errMsg:  "topic is required",
		},
		{
			name: "default priority and max_retries",
			setup: func(t *testing.T) (*ent.Tx, *OutboxMessage) {
				tx, err := client.Tx(ctx)
				require.NoError(t, err)

				msg := &OutboxMessage{
					TenantID:      1,
					AggregateType: "InputTask",
					AggregateID:   "task-123",
					Topic:         "io.tasks",
					MessageValue:  []byte(`{}`),
					EventType:     "TaskCreated",
					// Priority and MaxRetries not set
				}
				return tx, msg
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, msg := tt.setup(t)
			defer func() {
				if tx != nil {
					_ = tx.Rollback()
				}
			}()

			result, err := publisher.SaveToOutbox(ctx, tx, msg)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)

				// Verify saved message
				assert.Equal(t, msg.TenantID, result.TenantID)
				assert.Equal(t, msg.AggregateType, result.AggregateType)
				assert.Equal(t, msg.Topic, result.Topic)
				assert.Equal(t, msg.EventType, result.EventType)
				assert.Equal(t, "pending", result.SendStatus)
				assert.Equal(t, 0, result.RetryCount)

				// Check defaults
				if msg.Priority == 0 {
					assert.Equal(t, 5, result.Priority)
				}
				if msg.MaxRetries == 0 {
					assert.Equal(t, 3, result.MaxRetries)
				}

				// Commit to verify no constraint violations
				err = tx.Commit()
				assert.NoError(t, err)
				tx = nil // Prevent double rollback
			}
		})
	}
}

func TestSaveTaskCreatedMessage(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	publisher := NewOutboxPublisher(client)
	require.NotNil(t, publisher)

	ctx := context.Background()
	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer tx.Rollback()

	taskID := uint64(12345)
	taskName := "Import Assets Task"
	tenantID := uint64(1)

	err = publisher.SaveTaskCreatedMessage(ctx, tx, taskID, taskName, tenantID)
	require.NoError(t, err)

	// Verify message was saved
	messages, err := tx.OutboxMessage.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, messages, 1)

	msg := messages[0]
	assert.Equal(t, tenantID, msg.TenantID)
	assert.Equal(t, "InputTask", msg.AggregateType)
	assert.Equal(t, "io.input.jobs", msg.Topic)
	assert.Equal(t, "TaskCreated", msg.EventType)
	assert.Equal(t, "pending", msg.SendStatus)

	// Verify message value
	var payload map[string]interface{}
	err = json.Unmarshal(msg.MessageValue, &payload)
	require.NoError(t, err)
	assert.Equal(t, float64(taskID), payload["task_id"])
	assert.Equal(t, taskName, payload["task_name"])
	assert.Equal(t, "TaskCreated", payload["event_type"])
}

func TestSaveTaskCompletedMessage(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	publisher := NewOutboxPublisher(client)
	require.NotNil(t, publisher)

	ctx := context.Background()
	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer tx.Rollback()

	taskID := uint64(67890)
	taskName := "Export Data Task"
	tenantID := uint64(2)
	status := "completed"

	err = publisher.SaveTaskCompletedMessage(ctx, tx, taskID, taskName, tenantID, status)
	require.NoError(t, err)

	// Verify message was saved
	messages, err := tx.OutboxMessage.Query().All(ctx)
	require.NoError(t, err)
	require.Len(t, messages, 1)

	msg := messages[0]
	assert.Equal(t, tenantID, msg.TenantID)
	assert.Equal(t, "InputTask", msg.AggregateType)
	assert.Equal(t, "io.input.jobs", msg.Topic)
	assert.Equal(t, "TaskCompleted", msg.EventType)

	// Verify message value
	var payload map[string]interface{}
	err = json.Unmarshal(msg.MessageValue, &payload)
	require.NoError(t, err)
	assert.Equal(t, float64(taskID), payload["task_id"])
	assert.Equal(t, taskName, payload["task_name"])
	assert.Equal(t, status, payload["status"])
}

func TestSaveCustomMessage(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	publisher := NewOutboxPublisher(client)
	require.NotNil(t, publisher)

	ctx := context.Background()

	tests := []struct {
		name     string
		tenantID uint64
		topic    string
		eventType string
		payload  interface{}
		options  []func(*OutboxMessage)
		wantErr  bool
	}{
		{
			name:      "simple custom message",
			tenantID:  1,
			topic:     "io.custom",
			eventType: "CustomEvent",
			payload: map[string]interface{}{
				"data": "test",
			},
			wantErr: false,
		},
		{
			name:      "custom message with options",
			tenantID:  1,
			topic:     "io.notifications",
			eventType: "NotificationSent",
			payload: map[string]string{
				"user_id": "user-123",
				"message": "Hello",
			},
			options: []func(*OutboxMessage){
				func(msg *OutboxMessage) {
					msg.Priority = 9
					msg.MaxRetries = 5
					msg.AggregateType = "Notification"
					msg.AggregateID = "notif-456"
				},
			},
			wantErr: false,
		},
		{
			name:      "missing tenant_id",
			tenantID:  0,
			topic:     "io.test",
			eventType: "TestEvent",
			payload:   map[string]string{"test": "data"},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := client.Tx(ctx)
			require.NoError(t, err)
			defer tx.Rollback()

			err = publisher.SaveCustomMessage(ctx, tx, tt.tenantID, tt.topic, tt.eventType, tt.payload, tt.options...)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify message
				messages, err := tx.OutboxMessage.Query().All(ctx)
				require.NoError(t, err)
				require.Len(t, messages, 1)

				msg := messages[0]
				assert.Equal(t, tt.tenantID, msg.TenantID)
				assert.Equal(t, tt.topic, msg.Topic)
				assert.Equal(t, tt.eventType, msg.EventType)

				// Check options were applied
				if len(tt.options) > 0 {
					assert.Equal(t, 9, msg.Priority)
					assert.Equal(t, 5, msg.MaxRetries)
					assert.Equal(t, "Notification", msg.AggregateType)
				}
			}
		})
	}
}

func TestOutboxMessageTimestamps(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	publisher := NewOutboxPublisher(client)
	require.NotNil(t, publisher)

	ctx := context.Background()
	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	defer tx.Rollback()

	beforeCreate := time.Now()
	time.Sleep(10 * time.Millisecond)

	msg := &OutboxMessage{
		TenantID:      1,
		AggregateType: "Test",
		AggregateID:   "test-timestamp",
		Topic:         "io.test",
		MessageValue:  []byte(`{}`),
		EventType:     "TestEvent",
	}

	result, err := publisher.SaveToOutbox(ctx, tx, msg)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)
	afterCreate := time.Now()

	// Verify timestamps
	assert.True(t, result.CreatedAt.After(beforeCreate))
	assert.True(t, result.CreatedAt.Before(afterCreate))
	assert.True(t, result.UpdatedAt.After(beforeCreate))
	assert.True(t, result.UpdatedAt.Before(afterCreate))
}
