package producer

import (
	"context"
	"fmt"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/internal/queue/types"
)

// ExampleNewProducer 示例：创建Producer并发送消息
func ExampleNewProducer() {
	// 1. 创建Producer配置
	config := &Config{
		Brokers:      []string{"192.168.26.130:9092"},
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		BatchSize:    100,
		MaxAttempts:  3,
		Compression:  "snappy",
	}

	// 2. 创建Producer
	producer, err := NewProducer(config)
	if err != nil {
		panic(err)
	}
	defer producer.Close()

	// 3. 发送消息
	ctx := context.Background()
	topic := "io.input.jobs"
	key := []byte("tenant-1:connector-1")
	value := []byte(`{"task_type":"sync_users","tenant_id":1}`)
	headers := map[string]string{
		"X-Tenant-ID": "1",
		"X-Trace-ID":  "trace-123",
	}

	err = producer.Publish(ctx, topic, key, value, headers)
	if err != nil {
		panic(err)
	}

	fmt.Println("Message published successfully")
}

// ExampleSecureProducer_PublishTaskMessage 示例：使用安全Producer发送任务消息
func ExampleSecureProducer_PublishTaskMessage() {
	// 1. 创建基础Producer
	config := DefaultConfig([]string{"192.168.26.130:9092"})
	baseProducer, err := NewProducer(config)
	if err != nil {
		panic(err)
	}
	defer baseProducer.Close()

	// 2. 创建安全Producer（启用严格模式）
	secureProducer := NewSecureProducer(baseProducer, true)

	// 3. 构建任务消息
	taskMsg := &types.TaskMessage{
		TenantID:        1,
		TaskRunID:       12345,
		TaskType:        "sync_users",
		ProviderID:      "ldap-provider",
		CredentialRefID: "cred-001", // 使用凭证引用，不传递实际密码
		UserID:          100,
		IdempotencyKey:  "idem-key-12345",
		CreatedAt:       time.Now(),
		MappingRules: []types.FieldMapping{
			{
				SourceField: "uid",
				TargetField: "username",
			},
			{
				SourceField: "mail",
				TargetField: "email",
			},
		},
	}

	// 4. 发送消息
	ctx := context.Background()
	err = secureProducer.PublishTaskMessage(ctx, "io.input.jobs", taskMsg, 1, "ldap-provider")
	if err != nil {
		panic(err)
	}

	fmt.Println("Task message published successfully")
}

// ExampleSecureProducer_PublishWithIdempotency 示例：发送带幂等性的消息
func ExampleSecureProducer_PublishWithIdempotency() {
	// 1. 创建基础Producer
	config := DefaultConfig([]string{"192.168.26.130:9092"})
	baseProducer, err := NewProducer(config)
	if err != nil {
		panic(err)
	}
	defer baseProducer.Close()

	// 2. 创建安全Producer（宽松模式）
	secureProducer := NewSecureProducer(baseProducer, false)

	// 3. 构建输出任务消息
	taskResult := map[string]interface{}{
		"tenant_id":   uint64(1),
		"task_run_id": uint64(12345),
		"task_type":   "write_users",
		"result": map[string]interface{}{
			"created": 10,
			"updated": 5,
			"failed":  1,
		},
	}

	// 4. 发送带幂等性的消息
	ctx := context.Background()
	err = secureProducer.PublishWithIdempotency(ctx, "io.output.jobs", taskResult, 1, 12345)
	if err != nil {
		panic(err)
	}

	fmt.Println("Idempotent message published successfully")
}
