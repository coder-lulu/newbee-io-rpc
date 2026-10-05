# Unified-IO Cron 定时任务功能

> **版本**: v1.0.0 (Phase 2 Complete)
> **状态**: ✅ Production Ready
> **最后更新**: 2025-12-24

---

## 📖 快速导航

### 对于用户
- 🚀 [5分钟快速开始](#5分钟快速开始)
- 📘 [完整用户指南](./CRON_USER_GUIDE.md)
- 💡 [常见场景示例](#常见场景示例)
- ❓ [常见问题](#常见问题)

### 对于开发者
- 🏗️ [架构设计](./PHASE2_CRON_IMPLEMENTATION_SUMMARY.md)
- 🧪 [测试报告](./PHASE2_CRON_TESTS_SUMMARY.md)
- 📋 [完成报告](./PHASE2_FINAL_REPORT.md)
- 🔧 [API参考](#api-参考)

---

## 什么是 Cron 定时任务？

Unified-IO 的 Cron 定时任务功能允许您使用标准 Cron 表达式创建周期性的自动化任务，实现：

- ⏰ **定时执行** - 按Cron表达式自动触发
- 🔄 **自动化** - 无需人工干预
- 📊 **统计跟踪** - 完整的执行历史和统计
- 🎛️ **灵活控制** - 启用/禁用/手动触发

### 典型应用场景

| 场景 | Cron表达式 | 说明 |
|------|-----------|------|
| 💾 数据库备份 | `0 2 * * *` | 每天凌晨2点 |
| 🧹 日志清理 | `0 2 * * 0` | 每周日凌晨2点 |
| 📊 性能监控 | `*/5 * * * *` | 每5分钟 |
| 📈 报表生成 | `0 2 1 * *` | 每月1日凌晨2点 |
| 🔍 健康检查 | `*/10 9-18 * * 1-5` | 工作日9-18点每10分钟 |

---

## 5分钟快速开始

### 步骤1: 创建定时任务

```go
package main

import (
    "context"
    "fmt"
    "github.com/coder-lulu/newbee-io-rpc/ioclient"
    "github.com/coder-lulu/newbee-io-rpc/types/io"
    "github.com/zeromicro/go-zero/core/conf"
    "github.com/samber/lo"
)

func main() {
    // 1. 连接RPC服务
    var c ioclient.Config
    conf.MustLoad("etc/io-client.yaml", &c)
    client := ioclient.MustNewClient(c)

    // 2. 创建Cron任务
    resp, err := client.Io().CreateCronTask(context.Background(), &io.CronTaskInfo{
        TaskName:       lo.ToPtr("每日数据库备份"),
        CronExpression: lo.ToPtr("0 2 * * *"), // 每天凌晨2点
        InputSource:    lo.ToPtr("mysql_backup"),
        SourceConfig:   lo.ToPtr(`{
            "host": "mysql.example.com",
            "database": "production",
            "backup_path": "/backups/daily"
        }`),
        Description:    lo.ToPtr("自动备份生产数据库"),
        Enabled:        lo.ToPtr(true), // 立即启用
    })

    if err != nil {
        panic(err)
    }

    fmt.Printf("✅ CronTask创建成功! ID: %d\n", resp.Id)
}
```

### 步骤2: 查询任务状态

```go
// 查询任务详情
task, err := client.Io().GetCronTaskById(context.Background(), &io.IDReq{
    Id: resp.Id,
})

if err != nil {
    panic(err)
}

fmt.Printf("📋 任务信息:\n")
fmt.Printf("   名称: %s\n", *task.TaskName)
fmt.Printf("   表达式: %s\n", *task.CronExpression)
fmt.Printf("   状态: %s\n", map[bool]string{true: "✅ 启用", false: "❌ 禁用"}[*task.Enabled])
fmt.Printf("   已执行: %d次 (成功:%d, 失败:%d)\n",
    *task.ExecutionCount,
    *task.SuccessCount,
    *task.FailureCount)
fmt.Printf("   下次执行: %s\n",
    time.Unix(int64(*task.NextRunTime), 0).Format("2006-01-02 15:04:05"))
```

### 步骤3: 监控执行结果

```go
// 查询由该CronTask创建的InputTask
inputTasks, err := client.Io().GetInputTaskList(context.Background(), &io.InputTaskListReq{
    Page:       lo.ToPtr(uint64(1)),
    PageSize:   lo.ToPtr(uint64(10)),
    TaskType:   lo.ToPtr("cron"), // 只查询cron类型
    CronTaskId: lo.ToPtr(task.Id),
})

fmt.Printf("\n📊 执行历史:\n")
for _, inputTask := range inputTasks.Data {
    fmt.Printf("   %s - %s - %s\n",
        time.Unix(int64(*inputTask.ExecutionTime), 0).Format("2006-01-02 15:04:05"),
        *inputTask.TaskStatus,
        *inputTask.TaskName)
}
```

---

## 常见场景示例

### 场景1: 每小时数据汇总

```go
task := &io.CronTaskInfo{
    TaskName:       lo.ToPtr("用户行为小时汇总"),
    CronExpression: lo.ToPtr("0 * * * *"), // 每小时整点
    InputSource:    lo.ToPtr("clickhouse_aggregator"),
    SourceConfig:   lo.ToPtr(`{
        "source_table": "user_events",
        "target_table": "user_events_hourly",
        "agg_columns": ["user_id", "event_type"],
        "time_window": "1h"
    }`),
    Enabled:        lo.ToPtr(true),
}
```

### 场景2: 工作时间监控

```go
task := &io.CronTaskInfo{
    TaskName:       lo.ToPtr("业务高峰监控"),
    CronExpression: lo.ToPtr("*/5 9-18 * * 1-5"), // 工作日9-18点每5分钟
    InputSource:    lo.ToPtr("business_monitor"),
    SourceConfig:   lo.ToPtr(`{
        "metrics": ["order_count", "payment_count", "error_rate"],
        "alert_threshold": {"error_rate": 0.01}
    }`),
    Enabled:        lo.ToPtr(true),
}
```

### 场景3: 月末报表

```go
task := &io.CronTaskInfo{
    TaskName:       lo.ToPtr("月度销售报表"),
    CronExpression: lo.ToPtr("0 2 L * *"), // 每月最后一天凌晨2点
    InputSource:    lo.ToPtr("report_generator"),
    SourceConfig:   lo.ToPtr(`{
        "report_type": "sales_monthly",
        "format": "pdf",
        "recipients": ["sales@example.com"]
    }`),
    Enabled:        lo.ToPtr(true),
}
```

**注意**: `L` 表示最后一天，这是扩展语法，标准5字段格式使用 `0 2 28-31 * *` 近似实现。

---

## Cron 表达式速查

### 基本格式
```
分钟  小时  日  月  星期
 │    │   │   │    │
 │    │   │   │    └─ 0-7 (0和7都是星期日)
 │    │   │   └────── 1-12
 │    │   └────────── 1-31
 │    └────────────── 0-23
 └─────────────────── 0-59
```

### 常用表达式

| 表达式 | 说明 | 示例场景 |
|--------|------|---------|
| `*/5 * * * *` | 每5分钟 | 实时监控 |
| `0 * * * *` | 每小时 | 小时汇总 |
| `0 */2 * * *` | 每2小时 | 定期检查 |
| `0 2 * * *` | 每天2点 | 日常备份 |
| `0 2 * * 0` | 每周日2点 | 周报 |
| `0 2 1 * *` | 每月1日2点 | 月报 |
| `0 9-17 * * 1-5` | 工作日9-17点 | 工作时间任务 |
| `30 9 * * 1-5` | 工作日9:30 | 早会提醒 |

### 在线验证工具
- https://crontab.guru/ - 最受欢迎的Cron验证工具
- https://crontab.cronhub.io/ - 可视化Cron编辑器

---

## API 参考

### 核心API

| RPC方法 | 说明 | 文档 |
|---------|------|------|
| `CreateCronTask` | 创建定时任务 | [详情](#createcrontask) |
| `UpdateCronTask` | 更新任务配置 | [详情](#updatecrontask) |
| `EnableCronTask` | 启用任务 | [详情](#enablecrontask) |
| `DisableCronTask` | 禁用任务 | [详情](#disablecrontask) |
| `TriggerCronTaskNow` | 手动立即执行 | [详情](#triggercrontasknow) |
| `GetCronTaskById` | 查询单个任务 | [详情](#getcrontaskbyid) |
| `GetCronTaskList` | 查询任务列表 | [详情](#getcrontasklist) |
| `DeleteCronTask` | 删除任务 | [详情](#deletecrontask) |

### CreateCronTask

创建一个新的Cron定时任务。

**请求参数**:
```protobuf
message CronTaskInfo {
    optional string task_name = 1;        // 必填: 任务名称
    optional string cron_expression = 2;  // 必填: Cron表达式
    optional string input_source = 3;     // 必填: 数据源类型
    optional string source_config = 4;    // 必填: 数据源配置(JSON)
    optional string description = 5;      // 可选: 描述
    optional bool enabled = 6;            // 可选: 是否启用(默认false)
}
```

**响应**:
```protobuf
message BaseIDResp {
    uint64 id = 1;    // CronTask ID
    string msg = 2;   // 消息
}
```

**示例**:
```go
resp, err := client.Io().CreateCronTask(ctx, &io.CronTaskInfo{
    TaskName:       lo.ToPtr("测试任务"),
    CronExpression: lo.ToPtr("*/5 * * * *"),
    InputSource:    lo.ToPtr("test_provider"),
    SourceConfig:   lo.ToPtr(`{"key": "value"}`),
    Enabled:        lo.ToPtr(true),
})
```

### EnableCronTask

启用一个已禁用的Cron任务，任务将被添加到调度器并开始按计划执行。

**请求参数**:
```protobuf
message IDReq {
    uint64 id = 1;  // CronTask ID
}
```

**示例**:
```go
_, err := client.Io().EnableCronTask(ctx, &io.IDReq{
    Id: 123,
})
```

### DisableCronTask

禁用一个已启用的Cron任务，任务将从调度器移除并停止执行。

**示例**:
```go
_, err := client.Io().DisableCronTask(ctx, &io.IDReq{
    Id: 123,
})
```

### TriggerCronTaskNow

手动触发Cron任务立即执行一次，不影响正常的定时调度。

**示例**:
```go
resp, err := client.Io().TriggerCronTaskNow(ctx, &io.IDReq{
    Id: 123,
})
// resp.Id 是创建的InputTask ID
```

---

## 常见问题

### Q1: Cron表达式验证失败怎么办？
**A**:
1. 确保使用标准5字段格式：`分 时 日 月 周`
2. 使用在线工具验证：https://crontab.guru/
3. 查看错误信息中的具体提示

### Q2: 任务未按时执行？
**A**: 检查以下几点：
1. 任务是否已启用（`enabled=true`）
2. 查看CronScheduler日志确认任务已加载
3. 检查TaskWorker是否正常运行
4. 验证Cron表达式是否正确

### Q3: 如何查看任务执行历史？
**A**:
```go
// 查询由CronTask创建的所有InputTask
inputTasks, err := client.Io().GetInputTaskList(ctx, &io.InputTaskListReq{
    TaskType:   lo.ToPtr("cron"),
    CronTaskId: lo.ToPtr(cronTaskId),
})
```

### Q4: 可以创建秒级任务吗？
**A**: 不可以。最小间隔是1分钟，因为使用标准5字段Cron格式（不包含秒）。

### Q5: 任务执行失败会自动重试吗？
**A**: 当前版本不支持自动重试。计划在Phase 3实现此功能。

### Q6: 如何避免任务重复执行？
**A**: CronScheduler使用`SkipIfStillRunning`中间件，如果上次执行未完成，本次会自动跳过。

### Q7: 时区如何处理？
**A**: 所有时间使用UTC时区。在显示时需要转换为本地时区：
```go
localTime := time.Unix(*task.NextRunTime, 0).In(time.Local)
```

### Q8: 如何监控任务健康状态？
**A**:
```go
task, _ := client.Io().GetCronTaskById(ctx, &io.IDReq{Id: 123})

// 计算失败率
failureRate := float64(*task.FailureCount) / float64(*task.ExecutionCount)
if failureRate > 0.1 {
    // 失败率超过10%，发送告警
}
```

---

## 最佳实践

### ✅ 推荐做法

1. **合理选择执行频率**
   ```go
   // ✅ 好：根据业务需求选择
   "*/5 * * * *"   // 实时监控
   "0 2 * * *"     // 日常任务

   // ❌ 不好：过于频繁
   "* * * * *"     // 每分钟，压力大
   ```

2. **错开执行时间**
   ```go
   // ✅ 好：错开时间，避免扎堆
   "5 2 * * *"    // 02:05
   "15 2 * * *"   // 02:15
   "25 2 * * *"   // 02:25

   // ❌ 不好：同时执行
   "0 2 * * *"    // 所有任务都在02:00
   ```

3. **清晰的任务命名**
   ```go
   // ✅ 好：描述清晰
   TaskName: "财务-日报生成-每日凌晨2点"

   // ❌ 不好：含糊不清
   TaskName: "task1"
   ```

4. **合理的SourceConfig**
   ```go
   // ✅ 好：使用有效的JSON
   SourceConfig: `{
       "endpoint": "http://api.example.com",
       "timeout": 30,
       "retry": 3
   }`

   // ❌ 不好：无效JSON
   SourceConfig: `{endpoint: "...", timeout: 30}`
   ```

### ⚠️ 注意事项

1. **不要在SourceConfig中存储明文密码**
2. **建议CronTask总数不超过500个**
3. **单个任务执行时间建议不超过5分钟**
4. **最小执行间隔建议≥5分钟**
5. **定期监控失败率和执行延迟**

---

## 性能指标

| 指标 | 数值 | 说明 |
|------|------|------|
| 最大CronTask数 | ~1000 | 理论上限 |
| 建议CronTask数 | ≤500 | 推荐值 |
| 最小执行间隔 | 1分钟 | 技术限制 |
| 建议最小间隔 | ≥5分钟 | 性能考虑 |
| AddTask耗时 | ~250µs | 平均值 |
| 并发操作 | 30+无阻塞 | 测试验证 |

---

## 故障排查

### 问题：任务未执行
1. 检查任务是否启用：`enabled=true`
2. 验证Cron表达式合法性
3. 查看CronScheduler日志
4. 确认TaskWorker运行状态

### 问题：执行失败
1. 查询失败的InputTask
2. 检查SourceConfig配置
3. 使用TriggerCronTaskNow手动测试
4. 查看错误日志

### 问题：执行延迟
1. 检查pending队列长度
2. 增加TaskWorker并发数
3. 优化任务执行时长

**详细排查指南**: 参见 [用户指南 - 故障排查](./CRON_USER_GUIDE.md#6-故障排查)

---

## 完整文档

| 文档 | 适合人群 | 内容 |
|------|---------|------|
| **[用户指南](./CRON_USER_GUIDE.md)** | 👥 所有用户 | 完整使用手册，含最佳实践和示例 |
| **[实现总结](./PHASE2_CRON_IMPLEMENTATION_SUMMARY.md)** | 👨‍💻 开发者 | 架构设计、实现细节、技术决策 |
| **[测试报告](./PHASE2_CRON_TESTS_SUMMARY.md)** | 🧪 测试人员 | 测试覆盖、性能基准、质量指标 |
| **[完成报告](./PHASE2_FINAL_REPORT.md)** | 📊 项目经理 | 项目总结、交付物、成本效益 |

---

## 示例代码

完整示例代码请参见：
- [examples/cron_basic.go](../examples/cron_basic.go) - 基础使用
- [examples/cron_monitoring.go](../examples/cron_monitoring.go) - 监控采集
- [examples/cron_backup.go](../examples/cron_backup.go) - 数据备份

---

## 版本历史

| 版本 | 日期 | 说明 |
|------|------|------|
| v1.0.0 | 2025-12-24 | Phase 2完成，生产就绪 |

---

## 支持与反馈

- 📧 **Email**: support@example.com
- 💬 **Slack**: #unified-io-support
- 📖 **Wiki**: https://wiki.example.com/unified-io
- 🐛 **Issues**: https://github.com/coder-lulu/newbee-io-rpc/issues

---

## 许可证

Copyright © 2025 NewBee Team. All rights reserved.

---

**文档维护**: Unified-IO Team
**最后更新**: 2025-12-24
**文档版本**: v1.0.0
