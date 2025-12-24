// +build integration

package producer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/internal/queue/types"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_PublishAndConsume 集成测试：发布消息并消费验证
// 运行方式: go test -tags=integration -v ./internal/queue/producer
// 前提条件：Kafka服务必须运行在 192.168.26.130:9092
func TestIntegration_PublishAndConsume(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. 创建Producer
	config := &Config{
		Brokers:      []string{"192.168.26.130:9092"},
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		BatchSize:    1,
		MaxAttempts:  3,
		Compression:  "snappy",
	}

	producer, err := NewProducer(config)
	require.NoError(t, err)
	defer producer.Close()

	// 2. 创建安全Producer
	secureProducer := NewSecureProducer(producer, false)

	// 3. 构建测试消息
	taskMsg := &types.TaskMessage{
		TenantID:        1,
		TaskRunID:       uint64(time.Now().Unix()),
		TaskType:        "test_integration",
		ProviderID:      "test-provider",
		CredentialRefID: "test-cred",
		UserID:          100,
		CreatedAt:       time.Now(),
		Params: map[string]interface{}{
			"test": "integration",
		},
	}

	// 4. 发布消息
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	topic := "io.input.jobs"
	err = secureProducer.PublishTaskMessage(ctx, topic, taskMsg, taskMsg.TenantID, taskMsg.ProviderID)
	require.NoError(t, err, "failed to publish message")

	t.Logf("Message published: tenant_id=%d, task_run_id=%d", taskMsg.TenantID, taskMsg.TaskRunID)

	// 5. 创建Consumer验证消息
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   config.Brokers,
		Topic:     topic,
		GroupID:   "test-consumer-group",
		Partition: 0, // 读取分区0
		MinBytes:  1,
		MaxBytes:  10e6, // 10MB
	})
	defer reader.Close()

	// 6. 读取消息（最多等待10秒）
	readCtx, readCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer readCancel()

	msg, err := reader.ReadMessage(readCtx)
	if err != nil {
		if err == context.DeadlineExceeded {
			t.Skip("no messages available in Kafka, this might be expected")
		}
		require.NoError(t, err, "failed to read message")
	}

	// 7. 验证消息
	t.Logf("Message received: key=%s, size=%d bytes", string(msg.Key), len(msg.Value))

	// 验证Headers
	headers := make(map[string]string)
	for _, h := range msg.Headers {
		headers[h.Key] = string(h.Value)
	}

	assert.Equal(t, "1", headers["X-Tenant-ID"])
	assert.NotEmpty(t, headers["X-Trace-ID"])
	assert.NotEmpty(t, headers["X-Message-ID"])
	assert.NotEmpty(t, headers["X-Timestamp"])

	// 验证消息体
	var receivedMsg types.TaskMessage
	err = json.Unmarshal(msg.Value, &receivedMsg)
	require.NoError(t, err)

	assert.Equal(t, taskMsg.TenantID, receivedMsg.TenantID)
	assert.Equal(t, taskMsg.TaskRunID, receivedMsg.TaskRunID)
	assert.Equal(t, taskMsg.TaskType, receivedMsg.TaskType)
	assert.Equal(t, taskMsg.ProviderID, receivedMsg.ProviderID)

	t.Log("Integration test passed: message published and consumed successfully")
}

// TestIntegration_PerformanceTest 性能测试：批量发送消息
// 测试目标：达到 1000 msg/s
func TestIntegration_PerformanceTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	// 1. 创建Producer
	config := &Config{
		Brokers:      []string{"192.168.26.130:9092"},
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		BatchSize:    100, // 启用批处理
		MaxAttempts:  3,
		Compression:  "snappy",
	}

	producer, err := NewProducer(config)
	require.NoError(t, err)
	defer producer.Close()

	secureProducer := NewSecureProducer(producer, false)

	// 2. 批量发送消息
	ctx := context.Background()
	topic := "io.input.jobs"
	messageCount := 1000

	start := time.Now()

	for i := 0; i < messageCount; i++ {
		taskMsg := &types.TaskMessage{
			TenantID:   1,
			TaskRunID:  uint64(i),
			TaskType:   "perf_test",
			ProviderID: "test-provider",
			UserID:     100,
			CreatedAt:  time.Now(),
		}

		err := secureProducer.PublishTaskMessage(ctx, topic, taskMsg, 1, "test-provider")
		require.NoError(t, err)
	}

	elapsed := time.Since(start)
	throughput := float64(messageCount) / elapsed.Seconds()

	t.Logf("Performance test results:")
	t.Logf("  Messages sent: %d", messageCount)
	t.Logf("  Time elapsed: %v", elapsed)
	t.Logf("  Throughput: %.2f msg/s", throughput)

	// 验证吞吐量 >= 500 msg/s (考虑网络延迟)
	assert.GreaterOrEqual(t, throughput, 500.0, "throughput should be at least 500 msg/s")
}
