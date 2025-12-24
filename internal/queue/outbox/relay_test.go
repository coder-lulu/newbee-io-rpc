package outbox

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/ent/outboxmessage"
	"github.com/coder-lulu/newbee-io-rpc/internal/queue/producer"
)

// mockProducer implements Producer interface for testing
type mockProducer struct {
	messages      []mockMessage
	shouldFail    bool
	failureCount  int
	callCount     int
}

type mockMessage struct {
	topic   string
	key     []byte
	value   []byte
	headers map[string]string
}

func (m *mockProducer) Publish(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error {
	m.callCount++

	if m.shouldFail {
		if m.failureCount > 0 && m.callCount <= m.failureCount {
			return errors.New("mock producer error")
		}
	}

	m.messages = append(m.messages, mockMessage{
		topic:   topic,
		key:     key,
		value:   value,
		headers: headers,
	})
	return nil
}

func (m *mockProducer) Close() error {
	return nil
}

func TestNewOutboxRelay(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	mockProd := &mockProducer{}

	tests := []struct {
		name     string
		db       *ent.Client
		producer producer.Producer
		wantNil  bool
	}{
		{
			name:     "valid parameters with default config",
			db:       client,
			producer: mockProd,
			wantNil:  false,
		},
		{
			name:     "valid parameters with custom config",
			db:       client,
			producer: mockProd,
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relay := NewOutboxRelay(tt.db, tt.producer, nil, nil)
			assert.NotNil(t, relay)
			assert.Equal(t, tt.db, relay.db)
			assert.Equal(t, tt.producer, relay.producer)
			// Check defaults
			assert.Equal(t, 5*time.Second, relay.interval)
			assert.Equal(t, 100, relay.batchSize)
		})
	}
}

func TestOutboxRelayOptions(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	mockProd := &mockProducer{}

	config := &RelayConfig{
		Interval:  10 * time.Second,
		BatchSize: 50,
	}

	relay := NewOutboxRelay(client, mockProd, nil, config)

	require.NotNil(t, relay)
	assert.Equal(t, 10*time.Second, relay.interval)
	assert.Equal(t, 50, relay.batchSize)
}

func TestFetchPendingMessages(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	mockProd := &mockProducer{}
	relay := NewOutboxRelay(client, mockProd, nil, nil)
	require.NotNil(t, relay)

	ctx := context.Background()

	// Create test messages
	now := time.Now()
	futureTime := now.Add(1 * time.Hour)

	testCases := []struct {
		sendStatus  string
		nextRetryAt *time.Time
		priority    int
		shouldFetch bool
	}{
		{"pending", nil, 5, true},                // pending, no retry time
		{"pending", &now, 8, true},               // pending, retry time reached
		{"pending", &futureTime, 3, false},       // pending, retry time not reached
		{"sent", nil, 5, false},                  // already sent
		{"failed", nil, 2, false},                // failed (max retries exceeded)
	}

	for i, tc := range testCases {
		_, err := client.OutboxMessage.Create().
			SetTenantID(1).
			SetAggregateType("Test").
			SetAggregateID("test-" + string(rune(i))).
			SetTopic("io.test").
			SetMessageKey("key").
			SetMessageValue([]byte(`{}`)).
			SetEventType("TestEvent").
			SetSendStatus(tc.sendStatus).
			SetNillableNextRetryAt(tc.nextRetryAt).
			SetPriority(tc.priority).
			SetRetryCount(0).
			SetMaxRetries(3).
			Save(ctx)
		require.NoError(t, err)
	}

	// Fetch pending messages
	messages, err := relay.fetchPendingMessages(ctx)
	require.NoError(t, err)

	// Should fetch only messages with shouldFetch=true
	expectedCount := 0
	for _, tc := range testCases {
		if tc.shouldFetch {
			expectedCount++
		}
	}
	assert.Equal(t, expectedCount, len(messages))

	// Verify order (priority desc, created_at asc)
	if len(messages) > 1 {
		assert.GreaterOrEqual(t, messages[0].Priority, messages[1].Priority)
	}
}

func TestSendMessage_Success(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	mockProd := &mockProducer{}
	relay := NewOutboxRelay(client, mockProd, nil, nil)
	require.NotNil(t, relay)

	ctx := context.Background()

	// Create a test message
	msg, err := client.OutboxMessage.Create().
		SetTenantID(1).
		SetAggregateType("InputTask").
		SetAggregateID("task-123").
		SetTopic("io.tasks").
		SetMessageKey("task-123").
		SetMessageValue([]byte(`{"id":"task-123","name":"test"}`)).
		SetMessageHeaders(map[string]string{
			"X-Tenant-ID": "1",
		}).
		SetEventType("TaskCreated").
		SetSendStatus("pending").
		SetPriority(5).
		SetRetryCount(0).
		SetMaxRetries(3).
		Save(ctx)
	require.NoError(t, err)

	// Send the message
	err = relay.sendMessage(ctx, msg)
	require.NoError(t, err)

	// Mark as sent (as processOnce() would do)
	relay.markAsSent(ctx, msg)

	// Verify message was sent to producer
	assert.Equal(t, 1, len(mockProd.messages))
	sentMsg := mockProd.messages[0]
	assert.Equal(t, "io.tasks", sentMsg.topic)
	assert.Equal(t, []byte("task-123"), sentMsg.key)
	assert.Equal(t, msg.MessageValue, sentMsg.value)

	// Verify message status updated
	updated, err := client.OutboxMessage.Get(ctx, msg.ID)
	require.NoError(t, err)
	assert.Equal(t, "sent", updated.SendStatus)
	assert.NotNil(t, updated.SentAt)
}

func TestSendMessage_Failure(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	mockProd := &mockProducer{
		shouldFail:   true,
		failureCount: 999, // Always fail
	}
	relay := NewOutboxRelay(client, mockProd, nil, nil)
	require.NotNil(t, relay)

	ctx := context.Background()

	// Create a test message
	msg, err := client.OutboxMessage.Create().
		SetTenantID(1).
		SetAggregateType("InputTask").
		SetAggregateID("task-failure").
		SetTopic("io.tasks").
		SetMessageKey("key").
		SetMessageValue([]byte(`{}`)).
		SetEventType("TestEvent").
		SetSendStatus("pending").
		SetPriority(5).
		SetRetryCount(0).
		SetMaxRetries(3).
		Save(ctx)
	require.NoError(t, err)

	initialRetryCount := msg.RetryCount

	// Send the message (should fail)
	err = relay.sendMessage(ctx, msg)
	require.Error(t, err)

	// Handle failure (as processOnce() would do)
	relay.handleSendFailure(ctx, msg, err)

	// Verify retry count increased
	updated, err := client.OutboxMessage.Get(ctx, msg.ID)
	require.NoError(t, err)
	assert.Equal(t, initialRetryCount+1, updated.RetryCount)
	assert.NotNil(t, updated.NextRetryAt)
	assert.Equal(t, "pending", updated.SendStatus)
}

func TestHandleSendFailure_MaxRetriesExceeded(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	mockProd := &mockProducer{}
	relay := NewOutboxRelay(client, mockProd, nil, nil)
	require.NotNil(t, relay)

	ctx := context.Background()

	// Create message that already has max retries
	msg, err := client.OutboxMessage.Create().
		SetTenantID(1).
		SetAggregateType("Test").
		SetAggregateID("test-max-retries").
		SetTopic("io.test").
		SetMessageKey("key").
		SetMessageValue([]byte(`{}`)).
		SetEventType("TestEvent").
		SetSendStatus("pending").
		SetPriority(5).
		SetRetryCount(2).
		SetMaxRetries(3).
		Save(ctx)
	require.NoError(t, err)

	sendErr := errors.New("send failed")
	relay.handleSendFailure(ctx, msg, sendErr)

	// Verify message marked as failed
	updated, err := client.OutboxMessage.Get(ctx, msg.ID)
	require.NoError(t, err)
	assert.Equal(t, "failed", updated.SendStatus)
	assert.Equal(t, 3, updated.RetryCount)
}

func TestHandleSendFailure_ExponentialBackoff(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	mockProd := &mockProducer{}
	relay := NewOutboxRelay(client, mockProd, nil, nil)
	require.NotNil(t, relay)

	ctx := context.Background()

	tests := []struct {
		retryCount     int
		expectedBackoff time.Duration
	}{
		{0, 2 * time.Second},  // 2^1 = 2s
		{1, 4 * time.Second},  // 2^2 = 4s
		{2, 8 * time.Second},  // 2^3 = 8s
		{3, 16 * time.Second}, // 2^4 = 16s
	}

	for _, tt := range tests {
		t.Run("retry_"+string(rune(tt.retryCount)), func(t *testing.T) {
			msg, err := client.OutboxMessage.Create().
				SetTenantID(1).
				SetAggregateType("Test").
				SetAggregateID(fmt.Sprintf("test-retry-%d", tt.retryCount)).
				SetTopic("io.test").
				SetMessageKey("key").
				SetMessageValue([]byte(`{}`)).
				SetEventType("TestEvent").
				SetSendStatus("pending").
				SetPriority(5).
				SetRetryCount(tt.retryCount).
				SetMaxRetries(5).
				Save(ctx)
			require.NoError(t, err)

			beforeFailure := time.Now()
			sendErr := errors.New("send failed")
			relay.handleSendFailure(ctx, msg, sendErr)

			// Verify next retry time
			updated, err := client.OutboxMessage.Get(ctx, msg.ID)
			require.NoError(t, err)
			require.NotNil(t, updated.NextRetryAt)

			actualBackoff := updated.NextRetryAt.Sub(beforeFailure)
			// Allow 1 second tolerance for test execution time
			assert.InDelta(t, tt.expectedBackoff.Seconds(), actualBackoff.Seconds(), 1.0)
		})
	}
}

func TestProcessOnce(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	mockProd := &mockProducer{}
	config := &RelayConfig{
		BatchSize: 2,
	}
	relay := NewOutboxRelay(client, mockProd, nil, config)
	require.NotNil(t, relay)

	ctx := context.Background()

	// Create multiple pending messages
	for i := 0; i < 5; i++ {
		_, err := client.OutboxMessage.Create().
			SetTenantID(1).
			SetAggregateType("Test").
			SetAggregateID(fmt.Sprintf("test-process-%d", i)).
			SetTopic("io.test").
			SetMessageKey("key").
			SetMessageValue([]byte(`{}`)).
			SetEventType("TestEvent").
			SetSendStatus("pending").
			SetPriority(5).
			SetRetryCount(0).
			SetMaxRetries(3).
			Save(ctx)
		require.NoError(t, err)
	}

	// Process once (should process batch of 2)
	relay.processOnce(ctx)

	// Verify 2 messages were sent
	assert.Equal(t, 2, len(mockProd.messages))

	// Verify messages marked as sent
	sentCount, err := client.OutboxMessage.Query().
		Where(outboxmessage.SendStatusEQ("sent")).
		Count(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, sentCount, 2)
}

func TestStartStop(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	mockProd := &mockProducer{}
	config := &RelayConfig{
		Interval: 100 * time.Millisecond,
	}
	relay := NewOutboxRelay(client, mockProd, nil, config)
	require.NotNil(t, relay)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Create some pending messages
	for i := 0; i < 3; i++ {
		_, err := client.OutboxMessage.Create().
			SetTenantID(1).
			SetAggregateType("Test").
			SetAggregateID(fmt.Sprintf("test-startstop-%d", i)).
			SetTopic("io.test").
			SetMessageKey("key").
			SetMessageValue([]byte(`{}`)).
			SetEventType("TestEvent").
			SetSendStatus("pending").
			SetPriority(5).
			SetRetryCount(0).
			SetMaxRetries(3).
			Save(context.Background())
		require.NoError(t, err)
	}

	// Start relay in goroutine
	go relay.Start(ctx)

	// Wait for processing
	time.Sleep(300 * time.Millisecond)

	// Stop relay
	relay.Stop()

	// Give it time to stop
	time.Sleep(100 * time.Millisecond)

	// Verify messages were processed
	assert.Greater(t, len(mockProd.messages), 0)
}

func TestMessageHeadersConversion(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	mockProd := &mockProducer{}
	relay := NewOutboxRelay(client, mockProd, nil, nil)
	require.NotNil(t, relay)

	ctx := context.Background()

	headers := map[string]string{
		"X-Tenant-ID":      "123",
		"X-Correlation-ID": "corr-456",
		"X-User-ID":        "user-789",
	}

	msg, err := client.OutboxMessage.Create().
		SetTenantID(1).
		SetAggregateType("Test").
		SetAggregateID("test-headers").
		SetTopic("io.test").
		SetMessageKey("key").
		SetMessageValue([]byte(`{}`)).
		SetMessageHeaders(headers).
		SetEventType("TestEvent").
		SetSendStatus("pending").
		SetPriority(5).
		SetRetryCount(0).
		SetMaxRetries(3).
		Save(ctx)
	require.NoError(t, err)

	err = relay.sendMessage(ctx, msg)
	require.NoError(t, err)

	// Verify headers were passed correctly
	require.Len(t, mockProd.messages, 1)
	sentHeaders := mockProd.messages[0].headers
	assert.Equal(t, "123", sentHeaders["X-Tenant-ID"])
	assert.Equal(t, "corr-456", sentHeaders["X-Correlation-ID"])
	assert.Equal(t, "user-789", sentHeaders["X-User-ID"])
}

func TestMetadataHandling(t *testing.T) {
	client := setupTestDB(t)
	defer client.Close()

	mockProd := &mockProducer{}
	relay := NewOutboxRelay(client, mockProd, nil, nil)
	require.NotNil(t, relay)

	ctx := context.Background()

	metadata := map[string]interface{}{
		"user_id":        "user-123",
		"source_system":  "api",
		"request_id":     "req-456",
		"custom_field":   123.45,
		"boolean_field":  true,
	}

	msg, err := client.OutboxMessage.Create().
		SetTenantID(1).
		SetAggregateType("Test").
		SetAggregateID("test-metadata").
		SetTopic("io.test").
		SetMessageKey("key").
		SetMessageValue([]byte(`{}`)).
		SetMetadata(metadata).
		SetEventType("TestEvent").
		SetSendStatus("pending").
		SetPriority(5).
		SetRetryCount(0).
		SetMaxRetries(3).
		Save(ctx)
	require.NoError(t, err)

	// Verify metadata was saved correctly
	retrieved, err := client.OutboxMessage.Get(ctx, msg.ID)
	require.NoError(t, err)
	assert.Equal(t, "user-123", retrieved.Metadata["user_id"])
	assert.Equal(t, "api", retrieved.Metadata["source_system"])
	assert.Equal(t, float64(123.45), retrieved.Metadata["custom_field"])
	assert.Equal(t, true, retrieved.Metadata["boolean_field"])
}
