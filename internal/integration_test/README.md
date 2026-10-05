# Integration Testing Framework

## 概述

本目录包含 unified-io 服务的集成测试，测试完整的数据流程：
Provider Discovery → Field Mapping → Script Transformation → Output

## 测试场景

### 1. SSH Discovery Integration Test
测试 SSH Provider 完整流程：
- SSH 连接和数据采集
- 字段映射
- 脚本转换
- 数据验证

### 2. Multi-Tenant Isolation Test
测试多租户数据隔离：
- 租户 A 的发现池不能看到租户 B 的数据
- 租户 A 的字段映射不能应用到租户 B 的数据

### 3. Error Handling Test
测试错误处理：
- SSH 连接失败
- 脚本执行超时
- 字段映射失败
- Provider 不存在

### 4. Performance Test
测试性能：
- 100 台主机发现 < 30 秒
- 并发发现测试
- 内存使用监控

## 运行方式

### 运行所有集成测试
```bash
go test -tags=integration -v ./internal/integration_test/...
```

### 运行单个测试场景
```bash
go test -tags=integration -v ./internal/integration_test -run TestSSHDiscoveryIntegration
```

### 跳过集成测试
```bash
go test -short ./...
```

## 前提条件

### 本地测试（使用 Mock）
```bash
# 无需外部依赖，使用内存数据库
go test -tags=integration -v ./internal/integration_test -run TestMock
```

### 完整集成测试（需要真实服务）
```bash
# 启动测试环境
docker-compose -f docker-compose.test.yml up -d

# 运行测试
go test -tags=integration -v ./internal/integration_test

# 清理
docker-compose -f docker-compose.test.yml down
```

## 测试数据

测试使用的示例数据位于 `testdata/` 目录：
- `ssh_discovery_mock.json` - SSH 发现的 Mock 数据
- `field_mappings.json` - 字段映射配置
- `transform_scripts.json` - 转换脚本配置

## CI/CD 集成

在 CI 环境中，集成测试会自动跳过（使用 `-short` 标志）。
要在 CI 中运行集成测试，需要配置测试环境变量：
```yaml
env:
  INTEGRATION_TEST: "true"
  SSH_TEST_HOST: "test-ssh-server"
  SSH_TEST_USER: "testuser"
  SSH_TEST_PASSWORD: "testpass"
```

## 故障排除

### 测试超时
```bash
# 增加超时时间
go test -tags=integration -timeout 5m -v ./internal/integration_test
```

### 查看详细日志
```bash
# 启用详细日志
export LOG_LEVEL=debug
go test -tags=integration -v ./internal/integration_test
```

### 清理测试数据
```bash
# 清理测试生成的数据
rm -rf ./testdata/output/*
```
