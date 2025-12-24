package consumer

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-zero/core/logx"
)

// MessageHandler 消息处理器接口
type MessageHandler interface {
	// Handle 处理单条消息
	// 返回nil表示处理成功，返回error表示处理失败
	Handle(ctx context.Context, msg *kafka.Message) error
}

// Consumer Kafka消费者接口
type Consumer interface {
	// Start 启动消费者（阻塞）
	Start(ctx context.Context) error

	// Close 关闭消费者
	Close() error
}

// Config Consumer配置
type Config struct {
	Brokers   []string      // Kafka broker地址列表
	Topic     string        // Topic名称
	GroupID   string        // Consumer Group ID
	Partition int           // 分区号（-1表示自动分配）

	// 消费配置
	MinBytes      int           // 最小字节数
	MaxBytes      int           // 最大字节数
	MaxWait       time.Duration // 最大等待时间
	CommitInterval time.Duration // 提交间隔

	// 可选配置
	StartOffset int64 // 起始偏移量（-1=最新，-2=最早）
}

// DefaultConfig 返回默认配置
func DefaultConfig(brokers []string, topic string, groupID string) *Config {
	return &Config{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		Partition:      -1, // 自动分配
		MinBytes:       1,
		MaxBytes:       10e6, // 10MB
		MaxWait:        500 * time.Millisecond,
		CommitInterval: 1 * time.Second,
		StartOffset:    kafka.LastOffset, // 从最新消息开始
	}
}

// kafkaConsumer Kafka消费者实现
type kafkaConsumer struct {
	reader  *kafka.Reader
	handler MessageHandler
	config  *Config
}

// NewConsumer 创建新的Kafka消费者
func NewConsumer(config *Config, handler MessageHandler) (Consumer, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if len(config.Brokers) == 0 {
		return nil, fmt.Errorf("brokers list cannot be empty")
	}

	if config.Topic == "" {
		return nil, fmt.Errorf("topic cannot be empty")
	}

	if config.GroupID == "" {
		return nil, fmt.Errorf("group ID cannot be empty")
	}

	if handler == nil {
		return nil, fmt.Errorf("handler cannot be nil")
	}

	// 创建Reader配置
	readerConfig := kafka.ReaderConfig{
		Brokers:        config.Brokers,
		Topic:          config.Topic,
		GroupID:        config.GroupID,
		MinBytes:       config.MinBytes,
		MaxBytes:       config.MaxBytes,
		MaxWait:        config.MaxWait,
		CommitInterval: config.CommitInterval,
		StartOffset:    config.StartOffset,

		// 错误日志
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			logx.Errorw("kafka consumer error", logx.Field("msg", fmt.Sprintf(msg, args...)))
		}),

		// 日志
		Logger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			logx.Infow("kafka consumer", logx.Field("msg", fmt.Sprintf(msg, args...)))
		}),
	}

	// 如果指定了分区，设置分区号
	if config.Partition >= 0 {
		readerConfig.Partition = config.Partition
	}

	reader := kafka.NewReader(readerConfig)

	return &kafkaConsumer{
		reader:  reader,
		handler: handler,
		config:  config,
	}, nil
}

// Start 启动消费者（阻塞）
func (c *kafkaConsumer) Start(ctx context.Context) error {
	logx.Infow("starting kafka consumer",
		logx.Field("topic", c.config.Topic),
		logx.Field("group_id", c.config.GroupID))

	for {
		select {
		case <-ctx.Done():
			logx.Info("consumer stopped by context")
			return ctx.Err()

		default:
			// 读取消息
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if err == context.Canceled || err == context.DeadlineExceeded {
					return err
				}
				logx.Errorw("failed to fetch message",
					logx.Field("error", err))
				continue
			}

			// 记录消息信息
			logx.Infow("message received",
				logx.Field("topic", msg.Topic),
				logx.Field("partition", msg.Partition),
				logx.Field("offset", msg.Offset),
				logx.Field("key", string(msg.Key)),
				logx.Field("size", len(msg.Value)))

			// 处理消息
			if err := c.handler.Handle(ctx, &msg); err != nil {
				logx.Errorw("failed to handle message",
					logx.Field("topic", msg.Topic),
					logx.Field("partition", msg.Partition),
					logx.Field("offset", msg.Offset),
					logx.Field("error", err))

				// 处理失败，不提交offset
				// 下次rebalance或重启后会重新消费
				continue
			}

			// 处理成功，提交offset
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				logx.Errorw("failed to commit message",
					logx.Field("topic", msg.Topic),
					logx.Field("partition", msg.Partition),
					logx.Field("offset", msg.Offset),
					logx.Field("error", err))
			} else {
				logx.Infow("message committed",
					logx.Field("topic", msg.Topic),
					logx.Field("partition", msg.Partition),
					logx.Field("offset", msg.Offset))
			}
		}
	}
}

// Close 关闭消费者
func (c *kafkaConsumer) Close() error {
	if c.reader != nil {
		logx.Info("closing kafka consumer")
		return c.reader.Close()
	}
	return nil
}
