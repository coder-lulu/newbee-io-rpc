# 配置中心使用指南

## 概述

Unified-IO配置中心提供统一的配置管理能力，支持：
- **三级缓存**: 内存(1µs) → Redis(1ms) → MySQL(10ms)
- **热重载**: 配置变更实时生效（< 1秒）
- **多租户隔离**: 每个租户独立配置空间
- **类型安全**: 支持string/int/float/bool/json多种类型
- **版本管理**: 配置变更历史可追溯

---

## 1. 初始化配置

### 1.1 运行初始化脚本

```bash
cd /opt/code/newbee/unified-io/rpc

# 方式1: 使用默认租户ID (1)
go run scripts/init_configs.go 'root:password@tcp(localhost:3306)/unified_io?parseTime=true'

# 方式2: 指定租户ID
go run scripts/init_configs.go 'root:password@tcp(localhost:3306)/unified_io?parseTime=true' 100
```

### 1.2 初始化输出示例

```
🚀 开始初始化配置中心...
   DSN: root:password@tcp(localhost:3306)/unified_io?parseTime=true
   租户ID: 1

✅ 创建配置成功: worker.pull_interval = 10s
✅ 创建配置成功: worker.batch_size = 10
✅ 创建配置成功: worker.max_concurrent = 5
✅ 创建配置成功: worker.task_timeout = 5m
...

============================================================
📊 配置初始化完成！
   ✅ 成功: 16
   ⏭️  跳过: 0
   ❌ 失败: 0
   📦 总计: 16
============================================================
```

---

## 2. 配置加载机制

### 2.1 配置优先级

配置加载遵循以下优先级（从高到低）：

```
1. YAML配置文件 (etc/io.yaml)
   ↓
2. ConfigCenter (MySQL + Redis + 内存缓存)
   ↓
3. 代码默认值 (DefaultWorkerConfig)
```

**示例**:
```go
// 配置加载逻辑 (service_context.go)
workerConfig = worker.LoadWorkerConfigFromCenter(ctx, configCenter, tenantID)

// YAML覆盖（如果显式配置）
if c.TaskWorker.PullInterval > 0 {
    workerConfig.PullInterval = c.TaskWorker.PullInterval
}
```

### 2.2 配置生效流程

```
服务启动
  ↓
初始化ConfigCenter
  ↓
加载Worker配置 (从ConfigCenter)
  ↓
YAML配置覆盖（如果有）
  ↓
启动TaskWorker/CronScheduler
```

---

## 3. 配置使用示例

### 3.1 在Logic中读取配置

```go
package logic

import (
    "github.com/coder-lulu/newbee-io-rpc/internal/service"
)

func (l *SomeLogic) SomeMethod(in *req) (*resp, error) {
    opts := &service.ConfigOptions{
        TenantID:    1,
        ServiceName: "unified-io",
        Category:    "service",
    }

    // 1. 读取字符串配置
    pullInterval, _ := l.svcCtx.ConfigCenter.Get(l.ctx, "worker.pull_interval", opts)

    // 2. 读取整数配置
    batchSize, _ := l.svcCtx.ConfigCenter.GetInt(l.ctx, "worker.batch_size", opts)

    // 3. 读取布尔配置
    maintenanceMode, _ := l.svcCtx.ConfigCenter.GetBool(l.ctx, "system.maintenance_mode", opts)

    // 4. 读取JSON配置
    var dbConfig DatabaseConfig
    _ = l.svcCtx.ConfigCenter.GetJSON(l.ctx, "database.config", opts, &dbConfig)

    // 5. 带默认值读取
    timeout := l.svcCtx.ConfigCenter.GetWithDefault(l.ctx, "api.timeout", "30s", opts)

    return &resp{}, nil
}
```

### 3.2 更新配置

```go
// 更新配置（会自动清除缓存+发布变更通知）
err := l.svcCtx.ConfigCenter.Set(l.ctx, "worker.batch_size", "20", opts)
if err != nil {
    return nil, fmt.Errorf("failed to update config: %w", err)
}

// 所有实例会在1秒内自动加载新配置
```

### 3.3 监听配置变更

```go
// 注册配置变更监听器
l.svcCtx.ConfigCenter.Watch("worker.batch_size", func(key, oldValue, newValue string) {
    logx.Infow("Config changed",
        logx.Field("key", key),
        logx.Field("old", oldValue),
        logx.Field("new", newValue))

    // 执行热重载逻辑
    // TODO: 更新Worker运行时配置
})
```

---

## 4. 已支持的配置项

### 4.1 TaskWorker配置

| 配置键 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `worker.pull_interval` | string | 10s | 任务拉取间隔 |
| `worker.batch_size` | int | 10 | 每次拉取任务数量 |
| `worker.max_concurrent` | int | 5 | 最大并发处理任务数 |
| `worker.task_timeout` | string | 5m | 单个任务超时时间 |
| `worker.stale_threshold` | string | 1h | 任务过期阈值 |
| `worker.stale_check_interval` | string | 10m | Stale任务检查间隔 |

### 4.2 CronScheduler配置

| 配置键 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `cron.default_timeout` | string | 30m | Cron任务默认超时 |
| `cron.max_retry` | int | 3 | 最大重试次数 |

### 4.3 分布式锁配置

| 配置键 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `lock.default_expiry` | string | 30s | 锁默认过期时间 |
| `lock.retry_times` | int | 3 | 获取锁重试次数 |
| `lock.retry_delay` | string | 500ms | 重试延迟 |

### 4.4 Provider配置

| 配置键 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `provider.mysql.timeout` | string | 30s | MySQL Provider超时 |
| `provider.api.timeout` | string | 60s | API Provider超时 |
| `provider.api.max_retries` | int | 3 | API最大重试次数 |

### 4.5 系统级配置

| 配置键 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `system.maintenance_mode` | bool | false | 维护模式 |
| `system.log_level` | string | info | 日志级别 |

---

## 5. 配置管理API

### 5.1 数据库直接管理

```sql
-- 查询配置
SELECT config_key, config_value, value_type, version
FROM config_items
WHERE tenant_id = 1
  AND status = 1
  AND config_group = 'worker';

-- 更新配置（不推荐，使用API更安全）
UPDATE config_items
SET config_value = '20', version = version + 1
WHERE tenant_id = 1
  AND config_key = 'worker.batch_size';

-- 删除配置（软删除）
UPDATE config_items
SET status = 2
WHERE tenant_id = 1
  AND config_key = 'deprecated.config';
```

### 5.2 通过ConfigCenter API

```go
// 推荐：使用ConfigCenter API（自动处理缓存+通知）
cc := svcCtx.ConfigCenter

// 创建/更新配置
cc.Set(ctx, "worker.batch_size", "20", opts)

// 删除配置
cc.Delete(ctx, "deprecated.config", tenantID)

// 批量查询
configs := []string{"worker.pull_interval", "worker.batch_size"}
for _, key := range configs {
    val, _ := cc.Get(ctx, key, opts)
    fmt.Printf("%s = %s\n", key, val)
}
```

---

## 6. 性能指标

### 6.1 缓存命中率

```
L1 (Memory Cache):
  - 命中延迟: ~1µs
  - TTL: 5分钟
  - 预期命中率: 95%+

L2 (Redis Cache):
  - 命中延迟: ~1ms
  - TTL: 30分钟
  - 预期命中率: 99%

L3 (MySQL):
  - 查询延迟: ~10ms
  - 冷启动或缓存失效时访问
```

### 6.2 热重载延迟

```
配置更新 → Redis Pub/Sub → 所有实例清除缓存 → 下次访问加载新配置

预期延迟: < 1秒
```

---

## 7. 故障排查

### 7.1 配置未生效

**问题**: 更新了配置但服务未使用新值

**检查步骤**:
1. 确认配置已写入数据库
   ```sql
   SELECT * FROM config_items WHERE config_key = 'xxx';
   ```

2. 检查Redis Pub/Sub通道
   ```bash
   redis-cli SUBSCRIBE unified-io:config:update
   ```

3. 查看服务日志
   ```bash
   grep "Config" io-rpc.log | tail -20
   ```

### 7.2 配置缓存不一致

**问题**: 不同实例配置值不同

**解决方案**:
```bash
# 1. 清除Redis缓存
redis-cli DEL "unified-io:config:tenant:1:*"

# 2. 重启服务（清除内存缓存）
systemctl restart unified-io-rpc
```

### 7.3 配置初始化失败

**问题**: init_configs.go执行报错

**常见原因**:
- 数据库连接失败
- 表不存在（未执行ent migration）
- 配置键重复

**解决方案**:
```bash
# 1. 检查数据库连接
mysql -u root -p -h localhost unified_io -e "SELECT 1"

# 2. 检查表是否存在
mysql -u root -p unified_io -e "SHOW TABLES LIKE 'config_items'"

# 3. 如果表不存在，运行ent generate
go run entgo.io/ent/cmd/ent generate ./ent/schema
```

---

## 8. 最佳实践

### 8.1 配置命名规范

```
格式: <service>.<module>.<key>

示例:
  worker.pull_interval      ✅ 正确
  worker_pull_interval      ❌ 错误（应使用点号分隔）
  WorkerPullInterval        ❌ 错误（应使用小写+下划线）
```

### 8.2 配置变更流程

```
1. 开发环境测试
   ↓
2. 在ConfigCenter更新配置
   ↓
3. 观察日志确认热重载生效
   ↓
4. 生产环境灰度发布
   ↓
5. 全量发布
```

### 8.3 敏感配置处理

```go
// 标记为敏感配置（is_sensitive=true）
_, err := db.ConfigItem.Create().
    SetConfigKey("database.password").
    SetConfigValue(encrypted).  // 存储加密后的值
    SetIsSensitive(true).        // 标记为敏感
    Save(ctx)

// 读取时自动解密（需要实现解密逻辑）
password, _ := cc.Get(ctx, "database.password", opts)
decrypted := decrypt(password)  // 自定义解密函数
```

---

## 9. 配置变更审计日志 ✅

### 9.1 功能概述

配置变更审计日志自动记录所有配置的创建、更新和删除操作，提供完整的变更历史追踪：

**记录内容**:
- ✅ 配置键和变更类型（create/update/delete）
- ✅ 变更前后的值（old_value → new_value）
- ✅ 版本号变更（old_version → new_version）
- ✅ 变更人信息（用户ID、用户名）
- ✅ 操作上下文（IP地址、User-Agent）
- ✅ 变更原因（可选）
- ✅ 回滚标记（用于跟踪回滚操作）

**特性**:
- 🚀 **异步记录** - 不阻塞配置变更主流程
- 🔒 **自动记录** - 无需手动调用，Set/Delete自动触发
- 📊 **可查询** - 提供多种查询API
- 🔄 **支持回滚追踪** - 可标记回滚操作来源

### 9.2 审计日志自动记录

**配置变更时自动记录**:

```go
// 配置创建 - 自动记录 create 类型审计日志
cc.Set(ctx, "worker.batch_size", "10", opts)
// 📝 审计日志: changeType=create, oldValue="", newValue="10", oldVersion=0, newVersion=1

// 配置更新 - 自动记录 update 类型审计日志
cc.Set(ctx, "worker.batch_size", "20", opts)
// 📝 审计日志: changeType=update, oldValue="10", newValue="20", oldVersion=1, newVersion=2

// 配置删除 - 自动记录 delete 类型审计日志
cc.Delete(ctx, "worker.batch_size", tenantID)
// 📝 审计日志: changeType=delete, oldValue="20", newValue="", oldVersion=2, newVersion=0
```

### 9.3 审计上下文传递

**通过context传递审计信息**:

```go
// 创建包含用户信息的context
ctx := context.WithValue(context.Background(), "userId", uint64(123))
ctx = context.WithValue(ctx, "username", "admin")
ctx = context.WithValue(ctx, "clientIp", "192.168.1.100")
ctx = context.WithValue(ctx, "userAgent", "Mozilla/5.0...")

// 配置变更会自动提取审计信息
cc.Set(ctx, "worker.batch_size", "30", opts)
// 📝 审计日志自动包含: changed_by=123, changed_by_name="admin", ip_address="192.168.1.100"
```

**RPC层自动提取**:

在RPC Logic中，context通常已包含用户信息（由JWT中间件注入），无需额外设置：

```go
func (l *SomeLogic) UpdateConfig(in *req) (*resp, error) {
    // l.ctx 已包含用户信息（userId, username等）
    err := l.svcCtx.ConfigCenter.Set(l.ctx, "key", "value", opts)
    // 审计日志自动记录变更人信息 ✅
    return &resp{}, nil
}
```

### 9.4 查询审计日志

**方法1: 查询最近变更**

```go
// 查询租户1的最近100条审计日志
logs, err := cc.QueryAuditLogs(ctx, tenantID, "", 100)

for _, log := range logs {
    fmt.Printf("[%s] %s: %s (v%d → v%d) by %s\n",
        log.CreatedAt.Format("2006-01-02 15:04:05"),
        log.ChangeType,
        log.ConfigKey,
        log.OldVersion,
        log.NewVersion,
        log.ChangedByName,
    )
}
```

**输出示例**:
```
[2025-12-25 10:30:15] update: worker.batch_size (v1 → v2) by admin
[2025-12-25 10:25:30] create: worker.max_concurrent (v0 → v1) by system
[2025-12-25 10:20:45] delete: deprecated.config (v3 → v0) by admin
```

**方法2: 查询特定配置的历史**

```go
// 查询worker.batch_size的完整变更历史（按时间正序）
history, err := cc.GetConfigHistory(ctx, tenantID, "worker.batch_size")

fmt.Printf("配置 'worker.batch_size' 的变更历史:\n")
for _, log := range history {
    fmt.Printf("  %s: %s → %s (by %s)\n",
        log.CreatedAt.Format("2006-01-02 15:04:05"),
        log.OldValue,
        log.NewValue,
        log.ChangedByName,
    )
}
```

**输出示例**:
```
配置 'worker.batch_size' 的变更历史:
  2025-12-25 09:00:00: "" → "10" (by system)       # 初始创建
  2025-12-25 10:30:15: "10" → "20" (by admin)      # 调整批量大小
  2025-12-25 14:45:20: "20" → "15" (by admin)      # 性能优化
```

### 9.5 使用脚本查询审计日志

**查询所有审计日志**:

```bash
cd /opt/code/newbee/unified-io/rpc

# 查询租户1的所有审计日志（最近100条）
go run scripts/query_audit_logs.go 'root:password@tcp(localhost:3306)/unified_io?parseTime=true' 1

# 查询特定配置的审计日志
go run scripts/query_audit_logs.go 'root:password@tcp(localhost:3306)/unified_io?parseTime=true' 1 worker.batch_size
```

**输出示例**:
```
🔍 查询配置审计日志...
   DSN: root:password@tcp(localhost:3306)/unified_io?parseTime=true
   租户ID: 1
   配置键: worker.batch_size

📊 找到 3 条审计日志:

=========================================================================================================================
ID    变更时间              变更类型         配置键                           变更人                旧版本     → 新版本
=========================================================================================================================
105   2025-12-25 14:45:20  update          worker.batch_size              admin               v2         → v3
      旧值: 20
      新值: 15
      原因: 性能优化：降低批量大小以减少内存占用

104   2025-12-25 10:30:15  update          worker.batch_size              admin               v1         → v2
      旧值: 10
      新值: 20

103   2025-12-25 09:00:00  create          worker.batch_size              System              v0         → v1
      旧值:
      新值: 10

=========================================================================================================================

✅ 查询完成！共 3 条记录
```

### 9.6 审计日志表结构

**config_audit_logs表字段**:

| 字段名 | 类型 | 说明 |
|--------|------|------|
| `id` | uint64 | 主键ID |
| `tenant_id` | uint64 | 租户ID |
| `config_key` | string | 配置键 |
| `old_value` | string | 旧值 |
| `new_value` | string | 新值 |
| `change_type` | string | 变更类型（create/update/delete） |
| `changed_by` | uint64 | 变更人用户ID |
| `changed_by_name` | string | 变更人用户名 |
| `service_name` | string | 所属服务 |
| `category` | string | 配置分类 |
| `config_group` | string | 配置组 |
| `change_reason` | string | 变更原因 |
| `ip_address` | string | 操作IP地址 |
| `user_agent` | string | 用户代理 |
| `old_version` | int | 旧版本号 |
| `new_version` | int | 新版本号 |
| `is_rollback` | bool | 是否为回滚操作 |
| `rollback_from_log_id` | string | 回滚来源日志ID |
| `created_at` | time | 创建时间 |
| `updated_at` | time | 更新时间 |

**索引**:
- `(tenant_id, config_key)` - 按租户和配置键查询
- `(created_at)` - 按时间查询
- `(changed_by)` - 按变更人查询
- `(change_type)` - 按变更类型查询
- `(tenant_id, config_key, created_at)` - 复合索引（最常用）

### 9.7 审计日志最佳实践

**1. 审计日志保留策略**:

```sql
-- 定期清理超过90天的审计日志（可选）
DELETE FROM config_audit_logs
WHERE created_at < DATE_SUB(NOW(), INTERVAL 90 DAY);
```

**2. 重要配置变更通知**:

```go
// 监听关键配置变更，发送告警
cc.Watch("system.maintenance_mode", func(key, old, new string) {
    if new == "true" {
        // 发送钉钉/邮件通知
        sendAlert("系统进入维护模式", key, old, new)
    }
})
```

**3. 合规性审计**:

```sql
-- 查询特定时间范围的所有配置变更（用于审计）
SELECT
    created_at,
    config_key,
    change_type,
    old_value,
    new_value,
    changed_by_name,
    ip_address
FROM config_audit_logs
WHERE tenant_id = 1
  AND created_at BETWEEN '2025-12-01' AND '2025-12-31'
ORDER BY created_at DESC;
```

---

## 10. 未来增强

- [x] ✅ Worker热重载支持（无需重启）- **已完成**
- [x] ✅ 配置变更审计日志 - **已完成**
- [ ] Web UI配置管理界面
- [ ] 配置回滚功能（一键回到指定版本，基于审计日志）
- [ ] 配置导入/导出（JSON/YAML格式）
- [ ] 配置模板系统
- [ ] 多环境配置隔离（dev/test/prod）
- [ ] 配置变更通知（钉钉/邮件/Webhook）
