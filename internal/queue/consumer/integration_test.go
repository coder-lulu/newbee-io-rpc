// +build integration

package consumer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/internal/idempotency"
	"github.com/coder-lulu/newbee-io-rpc/internal/queue/producer"
	"github.com/coder-lulu/newbee-io-rpc/internal/queue/types"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_ProducerConsumer 集成测试：Producer -> Consumer完整流程
// 运行方式: go test -tags=integration -v ./internal/queue/consumer
// 前提条件：
// 1. Kafka服务运行在 192.168.26.130:9092
// 2. Redis服务运行在 localhost:6379
func TestIntegration_ProducerConsumer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. 创建Redis客户端
	rds := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // 使用测试数据库
	})
	defer rds.Close()

	// 2. 创建幂等性守护者
	idempotencyGuard := idempotency.NewGuard(rds, &idempotency.Config{
		WindowTime: 5 * time.Minute,
	})

	// 3. 创建测试任务处理器
	taskHandler := &testTaskHandler{
		received: make(chan *types.TaskMessage, 10),
	}

	// 4. 创建安全Consumer
	secureConsumer := NewSecureConsumer(
		1, // tenantID
		"test-consumer-group",
		taskHandler,
		idempotencyGuard,
	)

	// 5. 创建Consumer配置
	consumerConfig := DefaultConfig(
		[]string{"192.168.26.130:9092"},
		"io.input.jobs",
		"test-consumer-group",
	)
	consumerConfig.StartOffset = kafka.FirstOffset // 从头开始读取

	// 6. 创建Consumer
	consumer, err := NewConsumer(consumerConfig, secureConsumer)
	require.NoError(t, err)
	defer consumer.Close()

	// 7. 启动Consumer（后台）
	consumerCtx, consumerCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer consumerCancel()

	consumerDone := make(chan error, 1)
	go func() {
		consumerDone <- consumer.Start(consumerCtx)
	}()

	// 8. 创建Producer
	producerConfig := producer.DefaultConfig([]string{"192.168.26.130:9092"})
	baseProducer, err := producer.NewProducer(producerConfig)
	require.NoError(t, err)
	defer baseProducer.Close()

	secureProducer := producer.NewSecureProducer(baseProducer, false)

	// 9. 发送测试消息
	taskMsg := &types.TaskMessage{
		TenantID:   1,
		TaskRunID:  uint64(time.Now().Unix()),
		TaskType:   "integration_test",
		ProviderID: "test-provider",
		UserID:     100,
		CreatedAt:  time.Now(),
	}

	err = secureProducer.PublishTaskMessage(
		context.Background(),
		"io.input.jobs",
		taskMsg,
		taskMsg.TenantID,
		taskMsg.ProviderID,
	)
	require.NoError(t, err)

	t.Logf("Message sent: tenant_id=%d, task_run_id=%d", taskMsg.TenantID, taskMsg.TaskRunID)

	// 10. 等待接收消息（最多10秒）
	select {
	case receivedMsg := <-taskHandler.received:
		t.Logf("Message received: tenant_id=%d, task_run_id=%d", receivedMsg.TenantID, receivedMsg.TaskRunID)
		assert.Equal(t, taskMsg.TenantID, receivedMsg.TenantID)
		assert.Equal(t, taskMsg.TaskRunID, receivedMsg.TaskRunID)
		assert.Equal(t, taskMsg.TaskType, receivedMsg.TaskType)

	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	t.Log("Integration test passed")
}

// TestIntegration_Idempotency 集成测试：幂等性验证
func TestIntegration_Idempotency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. 创建Redis客户端
	rds := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	defer rds.Close()

	// 2. 创建幂等性守护者
	idempotencyGuard := idempotency.NewGuard(rds, idempotency.DefaultConfig())

	// 3. 创建任务处理器
	taskHandler := &testTaskHandler{
		received: make(chan *types.TaskMessage, 10),
	}

	// 4. 创建安全Consumer
	secureConsumer := NewSecureConsumer(1, "test-group", taskHandler, idempotencyGuard)

	// 5. 创建相同的测试消息
	taskMsg := &types.TaskMessage{
		TenantID:  1,
		TaskRunID: uint64(time.Now().Unix()),
		TaskType:  "idempotency_test",
	}

	body, _ := json.Marshal(taskMsg)
	msg := kafka.Message{
		Headers: []kafka.Header{
			{Key: "X-Tenant-ID", Value: []byte("1")},
		},
		Value: body,
	}

	ctx := context.Background()

	// 6. 第一次处理
	err := secureConsumer.Handle(ctx, &msg)
	require.NoError(t, err)

	select {
	case <-taskHandler.received:
		t.Log("First message processed")
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for first message")
	}

	// 7. 第二次处理（重复消息）
	err = secureConsumer.Handle(ctx, &msg)
	require.NoError(t, err)

	// 8. 验证不会再次处理
	select {
	case <-taskHandler.received:
		t.Fatal("duplicate message should not be processed")
	case <-time.After(1 * time.Second):
		t.Log("Duplicate message correctly skipped")
	}

	t.Log("Idempotency test passed")
}

// testTaskHandler 测试用的任务处理器
type testTaskHandler struct {
	received chan *types.TaskMessage
}

func (h *testTaskHandler) HandleTask(ctx context.Context, msg *types.TaskMessage) error {
	select {
	case h.received <- msg:
	default:
	}
	return nil
}
