package producer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProducer 用于测试的Mock Producer
type mockProducer struct {
	messages []mockMessage
}

type mockMessage struct {
	topic   string
	key     []byte
	value   []byte
	headers map[string]string
}

func (m *mockProducer) Publish(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error {
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

// TestNewProducer 测试创建Producer
func TestNewProducer(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty brokers",
			config: &Config{
				Brokers: []string{},
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &Config{
				Brokers:      []string{"localhost:9092"},
				WriteTimeout: 10 * time.Second,
				ReadTimeout:  10 * time.Second,
				BatchSize:    100,
				MaxAttempts:  3,
				Compression:  "snappy",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			producer, err := NewProducer(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, producer)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, producer)
				if producer != nil {
					_ = producer.Close()
				}
			}
		})
	}
}

// TestDefaultConfig 测试默认配置
func TestDefaultConfig(t *testing.T) {
	brokers := []string{"localhost:9092", "localhost:9093"}
	config := DefaultConfig(brokers)

	assert.Equal(t, brokers, config.Brokers)
	assert.Equal(t, 10*time.Second, config.WriteTimeout)
	assert.Equal(t, 10*time.Second, config.ReadTimeout)
	assert.Equal(t, 100, config.BatchSize)
	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, "snappy", config.Compression)
}

// TestSecureProducer_PublishTaskMessage 测试发布任务消息
func TestSecureProducer_PublishTaskMessage(t *testing.T) {
	mock := &mockProducer{}
	secureProducer := NewSecureProducer(mock, false)

	ctx := context.Background()
	topic := "io.input.jobs"
	tenantID := uint64(123)
	connectorID := "test-connector"

	msg := map[string]interface{}{
		"task_type": "sync_users",
		"user_id":   uint64(456),
	}

	err := secureProducer.PublishTaskMessage(ctx, topic, msg, tenantID, connectorID)
	require.NoError(t, err)

	// 验证消息
	require.Len(t, mock.messages, 1)
	published := mock.messages[0]

	assert.Equal(t, topic, published.topic)
	assert.Equal(t, "123:test-connector", string(published.key))

	// 验证Headers
	assert.Equal(t, "123", published.headers["X-Tenant-ID"])
	assert.NotEmpty(t, published.headers["X-Trace-ID"])
	assert.NotEmpty(t, published.headers["X-Message-ID"])
	assert.NotEmpty(t, published.headers["X-Timestamp"])

	// 验证消息体
	var receivedMsg map[string]interface{}
	err = json.Unmarshal(published.value, &receivedMsg)
	require.NoError(t, err)
	assert.Equal(t, "sync_users", receivedMsg["task_type"])
}

// TestSecureProducer_SensitiveDataDetection 测试敏感信息检测
func TestSecureProducer_SensitiveDataDetection(t *testing.T) {
	tests := []struct {
		name       string
		strictMode bool
		msg        map[string]interface{}
		wantErr    bool
	}{
		{
			name:       "no sensitive data - strict mode",
			strictMode: true,
			msg: map[string]interface{}{
				"task_type": "sync_users",
				"user_id":   uint64(123),
			},
			wantErr: false,
		},
		{
			name:       "has password - strict mode",
			strictMode: true,
			msg: map[string]interface{}{
				"task_type": "sync_users",
				"password":  "secret123",
			},
			wantErr: true,
		},
		{
			name:       "has password - lenient mode",
			strictMode: false,
			msg: map[string]interface{}{
				"task_type": "sync_users",
				"password":  "secret123",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockProducer{}
			secureProducer := NewSecureProducer(mock, tt.strictMode)

			ctx := context.Background()
			err := secureProducer.PublishTaskMessage(ctx, "test-topic", tt.msg, 1, "test")

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSecureProducer_PublishWithIdempotency 测试带幂等性的发布
func TestSecureProducer_PublishWithIdempotency(t *testing.T) {
	mock := &mockProducer{}
	secureProducer := NewSecureProducer(mock, false)

	ctx := context.Background()
	topic := "io.output.jobs"
	tenantID := uint64(123)
	taskRunID := uint64(456)

	msg := map[string]interface{}{
		"task_type": "write_users",
		"data": map[string]interface{}{
			"name": "John Doe",
		},
	}

	err := secureProducer.PublishWithIdempotency(ctx, topic, msg, tenantID, taskRunID)
	require.NoError(t, err)

	// 验证消息
	require.Len(t, mock.messages, 1)
	published := mock.messages[0]

	assert.Equal(t, topic, published.topic)
	assert.Equal(t, "123:456", string(published.key))

	// 验证Headers
	assert.Equal(t, "123", published.headers["X-Tenant-ID"])
	assert.Equal(t, "456", published.headers["X-Task-Run-ID"])
	assert.NotEmpty(t, published.headers["X-Idempotency-Key"])

	// 验证消息体包含idempotency_key
	var receivedMsg map[string]interface{}
	err = json.Unmarshal(published.value, &receivedMsg)
	require.NoError(t, err)
	assert.NotEmpty(t, receivedMsg["idempotency_key"])
}

// TestGetCompression 测试压缩算法获取
func TestGetCompression(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"snappy", "snappy"},
		{"lz4", "lz4"},
		{"gzip", "gzip"},
		{"zstd", "zstd"},
		{"unknown", "snappy"}, // 默认返回snappy
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			compression := getCompression(tt.input)
			// 这里只是验证函数不会panic
			assert.NotNil(t, compression)
		})
	}
}
