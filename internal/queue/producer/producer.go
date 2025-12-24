package producer

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
	"github.com/zeromicro/go-zero/core/logx"
)

// Producer Kafka生产者接口
type Producer interface {
	// Publish 发布消息到指定topic
	Publish(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error

	// Close 关闭生产者
	Close() error
}

// Config Producer配置
type Config struct {
	Brokers      []string      // Kafka broker地址列表
	WriteTimeout time.Duration // 写入超时时间
	ReadTimeout  time.Duration // 读取超时时间
	BatchSize    int           // 批量大小
	MaxAttempts  int           // 最大重试次数

	// 压缩配置
	Compression string // snappy, lz4, gzip, zstd

	// 可选的SASL/SSL配置
	// TODO: 后续添加SASL/SSL支持
}

// DefaultConfig 返回默认配置
func DefaultConfig(brokers []string) *Config {
	return &Config{
		Brokers:      brokers,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		BatchSize:    100,
		MaxAttempts:  3,
		Compression:  "snappy",
	}
}

// kafkaProducer Kafka生产者实现
type kafkaProducer struct {
	writer *kafka.Writer
	config *Config
}

// NewProducer 创建新的Kafka生产者
func NewProducer(config *Config) (Producer, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if len(config.Brokers) == 0 {
		return nil, fmt.Errorf("brokers list cannot be empty")
	}

	// 解析压缩算法
	compression := getCompression(config.Compression)

	writer := &kafka.Writer{
		Addr:         kafka.TCP(config.Brokers...),
		Balancer:     &kafka.Hash{}, // 使用Hash负载均衡器，保证相同key的消息发送到同一分区
		WriteTimeout: config.WriteTimeout,
		ReadTimeout:  config.ReadTimeout,
		BatchSize:    config.BatchSize,
		MaxAttempts:  config.MaxAttempts,
		Compression:  compression,

		// 异步错误处理
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			logx.Errorw("kafka producer error", logx.Field("msg", fmt.Sprintf(msg, args...)))
		}),

		// 日志处理
		Logger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			logx.Infow("kafka producer", logx.Field("msg", fmt.Sprintf(msg, args...)))
		}),
	}

	return &kafkaProducer{
		writer: writer,
		config: config,
	}, nil
}

// Publish 发布消息到指定topic
func (p *kafkaProducer) Publish(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error {
	// 构建Kafka消息
	kafkaHeaders := make([]kafka.Header, 0, len(headers))
	for k, v := range headers {
		kafkaHeaders = append(kafkaHeaders, kafka.Header{
			Key:   k,
			Value: []byte(v),
		})
	}

	msg := kafka.Message{
		Topic:   topic,
		Key:     key,
		Value:   value,
		Headers: kafkaHeaders,
		Time:    time.Now(),
	}

	// 记录消息大小
	logx.Infow("publishing message",
		logx.Field("topic", topic),
		logx.Field("key", string(key)),
		logx.Field("size", len(value)),
		logx.Field("headers", len(headers)))

	// 发送消息
	err := p.writer.WriteMessages(ctx, msg)
	if err != nil {
		logx.Errorw("failed to publish message",
			logx.Field("topic", topic),
			logx.Field("key", string(key)),
			logx.Field("error", err))
		return fmt.Errorf("publish message: %w", err)
	}

	logx.Infow("message published successfully",
		logx.Field("topic", topic),
		logx.Field("key", string(key)))

	return nil
}

// Close 关闭生产者
func (p *kafkaProducer) Close() error {
	if p.writer != nil {
		logx.Info("closing kafka producer")
		return p.writer.Close()
	}
	return nil
}

// getCompression 获取压缩算法
func getCompression(compression string) kafka.Compression {
	switch compression {
	case "snappy":
		return compress.Snappy
	case "lz4":
		return compress.Lz4
	case "gzip":
		return compress.Gzip
	case "zstd":
		return compress.Zstd
	default:
		logx.Infow("unknown compression type, using snappy", logx.Field("type", compression))
		return compress.Snappy
	}
}
