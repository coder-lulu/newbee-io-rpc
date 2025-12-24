# Kafka Queue 组件

统一输入输出平台的Kafka消息队列组件，提供安全、可靠的消息发布和消费能力。

## 目录结构

```
queue/
├── types/              # 消息类型定义
│   └── message.go      # TaskMessage, FieldMapping, MessageHeaders
│
├── producer/           # Kafka生产者
│   ├── producer.go         # 基础Producer实现
│   ├── secure_producer.go  # 安全增强Producer
│   ├── producer_test.go    # 单元测试
│   ├── example_test.go     # 使用示例
│   └── integration_test.go # 集成测试
│
└── consumer/           # Kafka消费者 (待实现)
    ├── consumer.go         # 基础Consumer实现
    ├── secure_consumer.go  # 安全增强Consumer
    └── consumer_test.go    # 单元测试
```

## 快速开始

### 1. 安装依赖

```bash
go get github.com/segmentio/kafka-go@v0.4.49
```

### 2. 基础用法

```go
import "github.com/coder-lulu/newbee-io-rpc/internal/queue/producer"

// 创建Producer
config := producer.DefaultConfig([]string{"192.168.26.130:9092"})
producer, err := producer.NewProducer(config)
if err != nil {
    panic(err)
}
defer producer.Close()

// 发送消息
err = producer.Publish(ctx, "io.input.jobs", key, value, headers)
```

### 3. 使用安全Producer（推荐）

```go
// 创建安全Producer
baseProducer, _ := producer.NewProducer(config)
secureProducer := producer.NewSecureProducer(baseProducer, true)

// 发布任务消息（自动处理租户隔离、消息脱敏）
err := secureProducer.PublishTaskMessage(ctx, "io.input.jobs", taskMsg, tenantID, connectorID)
```

## 核心功能

### Producer 功能

- ✅ **基础发布**：消息发送到Kafka
- ✅ **租户隔离**：自动添加 `X-Tenant-ID` Header
- ✅ **消息脱敏**：检测并处理敏感信息（password, api_key, secret, token）
- ✅ **幂等性支持**：自动生成幂等性Key
- ✅ **分区策略**：`tenantID:connectorID` 或 `tenantID:taskRunID`
- ✅ **批量发送**：提升吞吐量
- ✅ **消息压缩**：支持 snappy, lz4, gzip, zstd
- ✅ **自动重试**：可配置重试次数
- ✅ **审计日志**：记录所有发送操作

### Consumer 功能 ✅

- ✅ **基础消费**：从Kafka订阅和消费消息
- ✅ **租户验证**：Header + Body双重验证
- ✅ **幂等性检查**：Redis快速去重 + DB持久化去重（二层架构）
- ✅ **错误处理**：区分临时性错误和永久性错误
- ✅ **优雅关闭**：Context取消和资源清理
- ✅ **安全审计**：详细的租户隔离违规日志
- [ ] DLQ死信队列（计划中）
- [ ] 批量消费优化（计划中）

## 测试

### 单元测试

```bash
go test -v ./internal/queue/producer
```

### 集成测试

```bash
# 前提条件：Kafka运行在 192.168.26.130:9092
go test -tags=integration -v ./internal/queue/producer
```

### 性能测试

```bash
go test -tags=integration -v ./internal/queue/producer -run PerformanceTest
```

## 文档

- [Kafka Producer使用文档](../../docs/KAFKA_PRODUCER_USAGE.md)
- [Kafka Consumer使用文档](../../docs/KAFKA_CONSUMER_USAGE.md) ✅ 新增
- [Kafka集成设计文档](../../docs/unified-io-kafka-queue-design.md)
- [Producer实现报告](../../docs/KAFKA_PRODUCER_IMPLEMENTATION_REPORT.md)
- [Phase验证报告](../../docs/PHASE_VERIFICATION_REPORT.md)

## 配置

### Kafka地址

当前配置的Kafka服务：
- **地址**: 192.168.26.130:9092
- **Topics**:
  - `io.input.jobs` (32分区, 3副本)
  - `io.output.jobs` (32分区, 3副本)

### 压缩算法

支持的压缩算法：
- `snappy` (推荐) - 平衡性能和压缩率
- `lz4` - 最快的压缩速度
- `gzip` - 最高的压缩率
- `zstd` - 新一代压缩算法

## 安全特性

### 租户隔离

所有消息自动包含 `X-Tenant-ID` Header，确保：
- 消息路由到正确的分区
- 审计日志记录租户信息
- Consumer可验证租户权限

### 消息脱敏

自动检测以下敏感字段：
- `password`
- `api_key`
- `secret`
- `token`
- `private_key`
- `access_key`
- `secret_key`

**严格模式**：检测到敏感信息时拒绝发送
**宽松模式**：自动脱敏为 `***REDACTED***`

### 幂等性

使用 `PublishWithIdempotency` 自动生成幂等性Key：
- Header: `X-Idempotency-Key`
- Body: `idempotency_key` 字段
- 配合Consumer的Redis/DB去重机制

## 监控

### 关键指标（待实现）

- `kafka_publish_total` - 发送总数
- `kafka_publish_success_total` - 发送成功数
- `kafka_publish_duration_seconds` - 发送延迟
- `kafka_message_size_bytes` - 消息大小
- `kafka_sensitive_data_detected_total` - 敏感信息检测次数

## 常见问题

### Q1: 如何处理发送失败？

Producer会自动重试（默认3次），如果仍然失败：
1. 返回错误给调用方
2. 记录ERROR级别日志
3. 建议实现Outbox模式确保消息不丢失

### Q2: 如何选择压缩算法？

- 默认使用 `snappy`（平衡性能和压缩率）
- 高吞吐场景使用 `lz4`
- 带宽受限场景使用 `gzip`

### Q3: 如何禁用敏感信息检测？

不推荐禁用，但可以使用宽松模式：
```go
secureProducer := producer.NewSecureProducer(baseProducer, false)
```

## 下一步计划

### Week 5-6: 高级特性 (计划中)
- [ ] 实现Outbox模式（确保消息不丢失）
- [ ] 实现DLQ死信队列（处理失败消息）
- [ ] 添加消息重试机制（指数退避）
- [ ] 实现大消息处理（>900KB分块传输）

### Week 7-8: 安全与运维 (计划中)
- [ ] 添加SASL/SSL支持（生产环境认证）
- [ ] 集成Prometheus监控（完整指标）
- [ ] 实现消息追踪（分布式链路追踪）
- [ ] 性能优化（批量处理、零拷贝）

### Week 9-10: 生产就绪 (计划中)
- [ ] Docker化部署
- [ ] Kubernetes配置
- [ ] 监控告警配置
- [ ] 运维文档完善

---

**更新时间**: 2025-10-21
**维护者**: NewBee IO Team
**状态**: ✅ Producer完成，✅ Consumer完成，进入高级特性阶段
