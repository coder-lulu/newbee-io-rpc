package consumer

import (
	"context"
	"fmt"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/internal/idempotency"
	"github.com/coder-lulu/newbee-io-rpc/internal/queue/types"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

// ExampleNewConsumer 示例：创建Consumer并消费消息
func ExampleNewConsumer() {
	// 1. 创建Handler
	handler := &SimpleHandler{}

	// 2. 创建Consumer配置
	config := DefaultConfig(
		[]string{"192.168.26.130:9092"},
		"io.input.jobs",
		"unified-io-worker",
	)

	// 3. 创建Consumer
	consumer, err := NewConsumer(config, handler)
	if err != nil {
		panic(err)
	}
	defer consumer.Close()

	// 4. 启动Consumer（阻塞）
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := consumer.Start(ctx); err != nil {
		if err != context.DeadlineExceeded {
			panic(err)
		}
	}

	fmt.Println("Consumer stopped")
}

// SimpleHandler 简单的消息处理器
type SimpleHandler struct{}

func (h *SimpleHandler) Handle(ctx context.Context, msg *kafka.Message) error {
	fmt.Printf("Received message: topic=%s, partition=%d, offset=%d\n",
		msg.Topic, msg.Partition, msg.Offset)
	return nil
}

// ExampleSecureConsumer 示例：使用安全Consumer处理任务消息
func ExampleSecureConsumer() {
	// 1. 创建Redis客户端
	rds := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rds.Close()

	// 2. 创建幂等性守护者
	idempotencyGuard := idempotency.NewGuard(rds, idempotency.DefaultConfig())

	// 3. 创建任务处理器
	taskHandler := &TaskProcessor{}

	// 4. 创建安全Consumer
	secureConsumer := NewSecureConsumer(
		1,                  // allowedTenantID
		"unified-io-worker", // groupID
		taskHandler,
		idempotencyGuard,
	)

	// 5. 创建基础Consumer配置
	config := DefaultConfig(
		[]string{"192.168.26.130:9092"},
		"io.input.jobs",
		"unified-io-worker",
	)

	// 6. 创建基础Consumer
	consumer, err := NewConsumer(config, secureConsumer)
	if err != nil {
		panic(err)
	}
	defer consumer.Close()

	// 7. 启动Consumer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("Secure consumer started")
	if err := consumer.Start(ctx); err != nil {
		fmt.Printf("Consumer stopped: %v\n", err)
	}
}

// TaskProcessor 任务处理器
type TaskProcessor struct{}

func (p *TaskProcessor) HandleTask(ctx context.Context, msg *types.TaskMessage) error {
	fmt.Printf("Processing task: tenant_id=%d, task_run_id=%d, task_type=%s\n",
		msg.TenantID, msg.TaskRunID, msg.TaskType)

	// 实际的业务逻辑
	// ...

	return nil
}
