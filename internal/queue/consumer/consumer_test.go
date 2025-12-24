package consumer

import (
	"context"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
)

// mockHandler 用于测试的Mock Handler
type mockHandler struct {
	messages []*kafka.Message
	err      error
}

func (m *mockHandler) Handle(ctx context.Context, msg *kafka.Message) error {
	m.messages = append(m.messages, msg)
	return m.err
}

// TestNewConsumer 测试创建Consumer
func TestNewConsumer(t *testing.T) {
	handler := &mockHandler{}

	tests := []struct {
		name    string
		config  *Config
		handler MessageHandler
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			handler: handler,
			wantErr: true,
		},
		{
			name: "empty brokers",
			config: &Config{
				Brokers: []string{},
				Topic:   "test-topic",
				GroupID: "test-group",
			},
			handler: handler,
			wantErr: true,
		},
		{
			name: "empty topic",
			config: &Config{
				Brokers: []string{"localhost:9092"},
				Topic:   "",
				GroupID: "test-group",
			},
			handler: handler,
			wantErr: true,
		},
		{
			name: "empty group ID",
			config: &Config{
				Brokers: []string{"localhost:9092"},
				Topic:   "test-topic",
				GroupID: "",
			},
			handler: handler,
			wantErr: true,
		},
		{
			name: "nil handler",
			config: &Config{
				Brokers: []string{"localhost:9092"},
				Topic:   "test-topic",
				GroupID: "test-group",
			},
			handler: nil,
			wantErr: true,
		},
		{
			name: "valid config",
			config: &Config{
				Brokers:        []string{"localhost:9092"},
				Topic:          "test-topic",
				GroupID:        "test-group",
				Partition:      -1,
				MinBytes:       1,
				MaxBytes:       10e6,
				MaxWait:        500 * time.Millisecond,
				CommitInterval: 1 * time.Second,
				StartOffset:    kafka.LastOffset,
			},
			handler: handler,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumer, err := NewConsumer(tt.config, tt.handler)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, consumer)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, consumer)
				if consumer != nil {
					_ = consumer.Close()
				}
			}
		})
	}
}

// TestDefaultConfig 测试默认配置
func TestDefaultConfig(t *testing.T) {
	brokers := []string{"localhost:9092"}
	topic := "test-topic"
	groupID := "test-group"

	config := DefaultConfig(brokers, topic, groupID)

	assert.Equal(t, brokers, config.Brokers)
	assert.Equal(t, topic, config.Topic)
	assert.Equal(t, groupID, config.GroupID)
	assert.Equal(t, -1, config.Partition)
	assert.Equal(t, 1, config.MinBytes)
	assert.Equal(t, int(10e6), config.MaxBytes)
	assert.Equal(t, 500*time.Millisecond, config.MaxWait)
	assert.Equal(t, 1*time.Second, config.CommitInterval)
	assert.Equal(t, int64(kafka.LastOffset), config.StartOffset)
}

// TestExtractTenantIDFromHeaders 测试从Headers提取租户ID
func TestExtractTenantIDFromHeaders(t *testing.T) {
	tests := []struct {
		name       string
		headers    []kafka.Header
		wantTenant uint64
		wantErr    bool
	}{
		{
			name: "valid tenant ID",
			headers: []kafka.Header{
				{Key: "X-Tenant-ID", Value: []byte("123")},
			},
			wantTenant: 123,
			wantErr:    false,
		},
		{
			name: "missing tenant ID",
			headers: []kafka.Header{
				{Key: "X-Trace-ID", Value: []byte("trace-123")},
			},
			wantTenant: 0,
			wantErr:    true,
		},
		{
			name: "invalid tenant ID format",
			headers: []kafka.Header{
				{Key: "X-Tenant-ID", Value: []byte("invalid")},
			},
			wantTenant: 0,
			wantErr:    true,
		},
		{
			name: "multiple headers",
			headers: []kafka.Header{
				{Key: "X-Trace-ID", Value: []byte("trace-123")},
				{Key: "X-Tenant-ID", Value: []byte("456")},
				{Key: "X-Message-ID", Value: []byte("msg-123")},
			},
			wantTenant: 456,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenantID, err := extractTenantIDFromHeaders(tt.headers)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantTenant, tenantID)
			}
		})
	}
}
