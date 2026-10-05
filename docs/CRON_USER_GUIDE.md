# Unified-IO Cron 定时任务 - 用户指南

## 目录
- [1. 快速入门](#1-快速入门)
- [2. Cron表达式语法](#2-cron表达式语法)
- [3. API使用指南](#3-api使用指南)
- [4. 最佳实践](#4-最佳实践)
- [5. 常见场景示例](#5-常见场景示例)
- [6. 故障排查](#6-故障排查)
- [7. 高级功能](#7-高级功能)

---

## 1. 快速入门

### 1.1 什么是 CronTask？

CronTask 是 Unified-IO 提供的定时任务功能，允许您：
- 📅 按照 Cron 表达式定期执行数据采集任务
- 🔄 自动创建 InputTask 实例并由 TaskWorker 执行
- 📊 跟踪执行统计（成功/失败次数）
- ⏱️ 自动计算下次执行时间

### 1.2 5分钟快速开始

#### 步骤1: 创建定时备份任务
```go
// 通过 RPC 调用
resp, err := ioClient.CreateCronTask(ctx, &io.CronTaskInfo{
    TaskName:       pointy.String("每日数据库备份"),
    CronExpression: pointy.String("0 2 * * *"), // 每天凌晨2点
    InputSource:    pointy.String("mysql_backup"),
    SourceConfig:   pointy.String(`{"database": "main_db", "path": "/backups"}`),
    Description:    pointy.String("自动备份主数据库"),
    Enabled:        pointy.Bool(true), // 立即启用
})
```

#### 步骤2: 验证任务状态
```go
// 查询任务详情
task, err := ioClient.GetCronTaskById(ctx, &io.IDReq{
    Id: resp.Id,
})

fmt.Printf("任务: %s\n", *task.TaskName)
fmt.Printf("下次执行: %s\n", time.Unix(int64(*task.NextRunTime), 0).Format("2006-01-02 15:04:05"))
fmt.Printf("已执行: %d次\n", *task.ExecutionCount)
```

#### 步骤3: 监控执行结果
```go
// 查询由该CronTask创建的所有InputTask
inputTasks, err := ioClient.GetInputTaskList(ctx, &io.InputTaskListReq{
    Page:     pointy.Uint64(1),
    PageSize: pointy.Uint64(10),
    TaskType: pointy.String("cron"), // 只查询cron类型
})

for _, task := range inputTasks.Data {
    fmt.Printf("执行时间: %s, 状态: %s\n",
        time.Unix(int64(*task.ExecutionTime), 0).Format("2006-01-02 15:04:05"),
        *task.TaskStatus)
}
```

---

## 2. Cron表达式语法

### 2.1 标准格式（5字段）

```
分钟 小时 日 月 星期
 │   │   │  │   │
 │   │   │  │   └─ 星期几 (0-7, 0和7都代表星期日)
 │   │   │  └───── 月份 (1-12)
 │   │   └──────── 月中的第几天 (1-31)
 │   └──────────── 小时 (0-23)
 └──────────────── 分钟 (0-59)
```

### 2.2 特殊字符

| 字符 | 说明 | 示例 |
|------|------|------|
| `*` | 任意值 | `* * * * *` = 每分钟 |
| `,` | 列举多个值 | `0,15,30,45 * * * *` = 每小时的0/15/30/45分 |
| `-` | 范围 | `0 9-17 * * *` = 9点到17点整点 |
| `/` | 步长 | `*/5 * * * *` = 每5分钟 |

### 2.3 常用表达式示例

| 表达式 | 说明 | 适用场景 |
|--------|------|---------|
| `*/5 * * * *` | 每5分钟 | 高频监控、实时数据采集 |
| `0 * * * *` | 每小时整点 | 小时级数据汇总 |
| `0 2 * * *` | 每天凌晨2点 | 日常备份、数据清理 |
| `0 2 * * 0` | 每周日凌晨2点 | 周报生成 |
| `0 2 1 * *` | 每月1日凌晨2点 | 月报生成、账单结算 |
| `0 9-17 * * 1-5` | 工作日9-17点整点 | 工作时间监控 |
| `*/10 9-17 * * 1-5` | 工作日9-17点每10分钟 | 业务高峰监控 |
| `0 0 1 1 *` | 每年1月1日零点 | 年度数据归档 |

### 2.4 表达式验证工具

在创建CronTask前，可以使用在线工具验证表达式：
- https://crontab.guru/
- https://crontab.cronhub.io/

或在Go代码中验证：
```go
import "github.com/robfig/cron/v3"

parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
schedule, err := parser.Parse("0 2 * * *")
if err != nil {
    fmt.Printf("无效的Cron表达式: %v\n", err)
    return
}

// 计算未来5次执行时间
now := time.Now()
for i := 0; i < 5; i++ {
    next := schedule.Next(now)
    fmt.Printf("第%d次执行: %s\n", i+1, next.Format("2006-01-02 15:04:05"))
    now = next
}
```

---

## 3. API使用指南

### 3.1 创建 CronTask

#### 请求示例
```go
req := &io.CronTaskInfo{
    TaskName:       pointy.String("服务器性能监控"),
    CronExpression: pointy.String("*/5 * * * *"), // 每5分钟
    InputSource:    pointy.String("prometheus"),
    SourceConfig:   pointy.String(`{
        "endpoint": "http://prometheus:9090",
        "query": "node_cpu_usage",
        "labels": ["host", "cpu"]
    }`),
    Description:    pointy.String("采集所有服务器CPU使用率"),
    Enabled:        pointy.Bool(true),
}

resp, err := ioClient.CreateCronTask(ctx, req)
if err != nil {
    log.Fatalf("创建失败: %v", err)
}
fmt.Printf("CronTask ID: %d\n", resp.Id)
```

#### 字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `task_name` | string | ✅ | 任务名称（唯一，便于识别） |
| `cron_expression` | string | ✅ | Cron表达式（5字段格式） |
| `input_source` | string | ✅ | 数据源类型（如prometheus, mysql, api） |
| `source_config` | string | ✅ | 数据源配置（JSON格式） |
| `description` | string | ❌ | 任务描述 |
| `enabled` | bool | ❌ | 是否启用（默认false） |

#### 响应示例
```json
{
    "id": 123,
    "msg": "success"
}
```

#### 自动计算
创建时系统会自动：
- ✅ 验证Cron表达式合法性
- ✅ 计算 `next_run_time`（下次执行时间）
- ✅ 如果 `enabled=true`，立即添加到调度器

---

### 3.2 更新 CronTask

#### 更新Cron表达式
```go
req := &io.CronTaskInfo{
    Id:             pointy.Uint64(123),
    CronExpression: pointy.String("*/10 * * * *"), // 从5分钟改为10分钟
}

_, err := ioClient.UpdateCronTask(ctx, req)
if err != nil {
    log.Fatalf("更新失败: %v", err)
}
```

#### 更新数据源配置
```go
req := &io.CronTaskInfo{
    Id:           pointy.Uint64(123),
    SourceConfig: pointy.String(`{
        "endpoint": "http://new-prometheus:9090",
        "query": "node_memory_usage"
    }`),
}

_, err := ioClient.UpdateCronTask(ctx, req)
```

#### 重要提示
- 更新Cron表达式时，系统会重新验证并重新计算 `next_run_time`
- 如果任务已启用，调度器会自动更新（先移除旧任务，再添加新任务）
- 不影响已创建的 InputTask 实例

---

### 3.3 启用/禁用 CronTask

#### 启用任务
```go
_, err := ioClient.EnableCronTask(ctx, &io.IDReq{
    Id: 123,
})
if err != nil {
    log.Fatalf("启用失败: %v", err)
}
fmt.Println("任务已启用，开始定时执行")
```

#### 禁用任务
```go
_, err := ioClient.DisableCronTask(ctx, &io.IDReq{
    Id: 123,
})
if err != nil {
    log.Fatalf("禁用失败: %v", err)
}
fmt.Println("任务已禁用，停止定时执行")
```

#### 状态变化流程
```
创建(enabled=false) → 禁用状态（不会执行）
     │
     ├─ EnableCronTask()
     │
     ▼
启用状态 → 添加到CronScheduler → 按Cron表达式定时执行
     │
     ├─ DisableCronTask()
     │
     ▼
禁用状态 → 从CronScheduler移除 → 停止执行
```

---

### 3.4 手动触发立即执行

#### 使用场景
- 🧪 测试Cron任务配置是否正确
- 🔧 调试数据采集逻辑
- 🚀 紧急执行（不等待下次定时触发）

#### 请求示例
```go
resp, err := ioClient.TriggerCronTaskNow(ctx, &io.IDReq{
    Id: 123,
})
if err != nil {
    log.Fatalf("触发失败: %v", err)
}

// 返回创建的InputTask ID
fmt.Printf("已创建InputTask: %d\n", resp.Id)
fmt.Println("任务将立即由TaskWorker执行")
```

#### 行为说明
- ✅ 创建 `task_type="cron_manual"` 的 InputTask
- ✅ 更新 `execution_count++`
- ✅ 更新 `last_run_time`
- ❌ **不影响** `next_run_time`（定时调度不受影响）

#### 区别对比

| 特性 | 定时触发 | 手动触发 |
|------|---------|---------|
| 创建方式 | CronScheduler自动 | TriggerCronTaskNow API |
| task_type | `cron` | `cron_manual` |
| 影响next_run_time | ✅ 是 | ❌ 否 |
| 更新统计 | ✅ 是 | ✅ 是 |

---

### 3.5 查询 CronTask

#### 查询单个任务
```go
task, err := ioClient.GetCronTaskById(ctx, &io.IDReq{
    Id: 123,
})

fmt.Printf("任务: %s\n", *task.TaskName)
fmt.Printf("Cron: %s\n", *task.CronExpression)
fmt.Printf("状态: %s\n", map[bool]string{true: "启用", false: "禁用"}[*task.Enabled])
fmt.Printf("已执行: %d次 (成功:%d, 失败:%d)\n",
    *task.ExecutionCount,
    *task.SuccessCount,
    *task.FailureCount)
fmt.Printf("上次执行: %s\n",
    time.Unix(int64(*task.LastRunTime), 0).Format("2006-01-02 15:04:05"))
fmt.Printf("下次执行: %s\n",
    time.Unix(int64(*task.NextRunTime), 0).Format("2006-01-02 15:04:05"))
```

#### 查询任务列表
```go
list, err := ioClient.GetCronTaskList(ctx, &io.CronTaskListReq{
    Page:     pointy.Uint64(1),
    PageSize: pointy.Uint64(10),
    Enabled:  pointy.Bool(true), // 只查询启用的任务
})

for _, task := range list.Data {
    fmt.Printf("%d. %s - %s\n", *task.Id, *task.TaskName, *task.CronExpression)
}
```

---

### 3.6 删除 CronTask

```go
_, err := ioClient.DeleteCronTask(ctx, &io.IDReq{
    Id: 123,
})
if err != nil {
    log.Fatalf("删除失败: %v", err)
}
fmt.Println("CronTask已删除")
```

**⚠️ 警告**:
- 删除CronTask会自动从调度器移除
- 已创建的InputTask不会被删除（保留执行历史）
- 删除操作不可逆，请谨慎操作

---

## 4. 最佳实践

### 4.1 Cron表达式选择

#### ✅ 推荐做法

**1. 避免整点扎堆**
```go
// ❌ 不好：所有任务都在整点执行
"0 0 * * *"  // 零点
"0 1 * * *"  // 1点
"0 2 * * *"  // 2点

// ✅ 好：错开执行时间
"5 0 * * *"   // 00:05
"15 1 * * *"  // 01:15
"25 2 * * *"  // 02:25
```

**2. 合理选择频率**
```go
// ❌ 不好：过于频繁（每分钟）
"* * * * *"  // 高数据库压力

// ✅ 好：根据业务需求选择
"*/5 * * * *"   // 实时监控：每5分钟
"0 */2 * * *"   // 定期检查：每2小时
"0 2 * * *"     // 日常任务：每天凌晨2点
```

**3. 使用凌晨时段**
```go
// ✅ 推荐：在业务低峰期执行重任务
"0 2-4 * * *"  // 凌晨2-4点（每小时整点）
"0 3 * * *"    // 凌晨3点（数据备份）
```

---

### 4.2 任务命名规范

#### 命名模板
```
[业务领域]-[操作类型]-[执行频率]

示例：
- "财务-日报生成-每日凌晨2点"
- "监控-CPU采集-每5分钟"
- "数据库-备份-每周日"
```

#### ✅ 好的命名
```go
taskName: "用户行为分析-每小时汇总"
taskName: "订单系统-过期订单清理-每日"
taskName: "日志系统-归档-每周"
```

#### ❌ 不好的命名
```go
taskName: "task1"          // 太模糊
taskName: "测试"           // 无实际意义
taskName: "cron"           // 太通用
```

---

### 4.3 SourceConfig 配置

#### JSON格式规范
```go
// ✅ 正确：使用有效的JSON
SourceConfig: `{
    "endpoint": "http://api.example.com",
    "timeout": 30,
    "retry": 3,
    "params": {
        "key1": "value1",
        "key2": "value2"
    }
}`

// ❌ 错误：无效的JSON
SourceConfig: `{endpoint: "...", timeout: 30}`  // 缺少引号
SourceConfig: `{"key": "value",}`               // 多余的逗号
```

#### 敏感信息处理
```go
// ❌ 不安全：明文密码
SourceConfig: `{
    "database": "mysql://root:password123@localhost/db"
}`

// ✅ 安全：使用环境变量或密钥管理服务
SourceConfig: `{
    "database": "mysql://root:${DB_PASSWORD}@localhost/db",
    "secret_ref": "aws-secrets-manager://db-credentials"
}`
```

---

### 4.4 监控与告警

#### 执行统计监控
```go
// 定期检查失败率
task, _ := ioClient.GetCronTaskById(ctx, &io.IDReq{Id: 123})

failureRate := float64(*task.FailureCount) / float64(*task.ExecutionCount) * 100

if failureRate > 10.0 {
    alert.Send(fmt.Sprintf("CronTask %s 失败率过高: %.2f%%",
        *task.TaskName, failureRate))
}
```

#### 执行延迟监控
```go
// 检查任务是否按时执行
task, _ := ioClient.GetCronTaskById(ctx, &io.IDReq{Id: 123})

lastRun := time.Unix(int64(*task.LastRunTime), 0)
nextRun := time.Unix(int64(*task.NextRunTime), 0)
now := time.Now()

if now.After(nextRun.Add(5 * time.Minute)) {
    alert.Send(fmt.Sprintf("CronTask %s 执行延迟超过5分钟", *task.TaskName))
}
```

---

### 4.5 错误处理

#### 创建时验证
```go
resp, err := ioClient.CreateCronTask(ctx, req)
if err != nil {
    // 检查是否是Cron表达式错误
    if strings.Contains(err.Error(), "invalid cron expression") {
        log.Printf("Cron表达式无效: %s", *req.CronExpression)
        // 提示用户修正表达式
        return
    }

    // 其他错误
    log.Printf("创建失败: %v", err)
    return
}
```

#### 重试机制
```go
func createCronTaskWithRetry(client io.IoClient, req *io.CronTaskInfo, maxRetries int) (*io.BaseIDResp, error) {
    var resp *io.BaseIDResp
    var err error

    for i := 0; i < maxRetries; i++ {
        resp, err = client.CreateCronTask(ctx, req)
        if err == nil {
            return resp, nil
        }

        // 如果是验证错误，不重试
        if strings.Contains(err.Error(), "invalid") {
            return nil, err
        }

        log.Printf("重试 %d/%d: %v", i+1, maxRetries, err)
        time.Sleep(time.Second * time.Duration(i+1)) // 指数退避
    }

    return nil, fmt.Errorf("达到最大重试次数: %w", err)
}
```

---

## 5. 常见场景示例

### 5.1 数据库备份

#### 需求
每天凌晨3点自动备份MySQL数据库

#### 实现
```go
task := &io.CronTaskInfo{
    TaskName:       pointy.String("MySQL主库备份"),
    CronExpression: pointy.String("0 3 * * *"),
    InputSource:    pointy.String("mysql_backup"),
    SourceConfig:   pointy.String(`{
        "host": "mysql-master.example.com",
        "port": 3306,
        "database": "production_db",
        "backup_path": "/backups/mysql",
        "compress": true,
        "keep_days": 7
    }`),
    Description:    pointy.String("每日凌晨3点备份生产数据库，保留7天"),
    Enabled:        pointy.Bool(true),
}

resp, err := ioClient.CreateCronTask(ctx, task)
```

---

### 5.2 定期数据清理

#### 需求
每周日凌晨2点清理30天前的日志

#### 实现
```go
task := &io.CronTaskInfo{
    TaskName:       pointy.String("日志清理-30天"),
    CronExpression: pointy.String("0 2 * * 0"), // 每周日凌晨2点
    InputSource:    pointy.String("log_cleaner"),
    SourceConfig:   pointy.String(`{
        "log_path": "/var/log/application",
        "retention_days": 30,
        "file_pattern": "*.log",
        "dry_run": false
    }`),
    Description:    pointy.String("每周清理30天前的应用日志"),
    Enabled:        pointy.Bool(true),
}

resp, err := ioClient.CreateCronTask(ctx, task)
```

---

### 5.3 监控数据采集

#### 需求
每5分钟采集服务器CPU、内存、磁盘使用率

#### 实现
```go
task := &io.CronTaskInfo{
    TaskName:       pointy.String("服务器性能监控"),
    CronExpression: pointy.String("*/5 * * * *"), // 每5分钟
    InputSource:    pointy.String("prometheus"),
    SourceConfig:   pointy.String(`{
        "endpoint": "http://prometheus:9090/api/v1/query",
        "queries": [
            {
                "name": "cpu_usage",
                "query": "100 - (avg by (instance) (irate(node_cpu_seconds_total{mode='idle'}[5m])) * 100)"
            },
            {
                "name": "memory_usage",
                "query": "(node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / node_memory_MemTotal_bytes * 100"
            },
            {
                "name": "disk_usage",
                "query": "(node_filesystem_size_bytes - node_filesystem_avail_bytes) / node_filesystem_size_bytes * 100"
            }
        ],
        "alert_threshold": {
            "cpu": 80,
            "memory": 85,
            "disk": 90
        }
    }`),
    Description:    pointy.String("高频采集服务器性能指标"),
    Enabled:        pointy.Bool(true),
}

resp, err := ioClient.CreateCronTask(ctx, task)
```

---

### 5.4 报表生成

#### 需求
每月1日凌晨生成上月财务报表

#### 实现
```go
task := &io.CronTaskInfo{
    TaskName:       pointy.String("月度财务报表"),
    CronExpression: pointy.String("0 1 1 * *"), // 每月1日凌晨1点
    InputSource:    pointy.String("report_generator"),
    SourceConfig:   pointy.String(`{
        "report_type": "financial",
        "period": "last_month",
        "format": "pdf",
        "recipients": [
            "finance@example.com",
            "ceo@example.com"
        ],
        "include_charts": true,
        "output_path": "/reports/financial"
    }`),
    Description:    pointy.String("自动生成并发送月度财务报表"),
    Enabled:        pointy.Bool(true),
}

resp, err := ioClient.CreateCronTask(ctx, task)
```

---

### 5.5 API健康检查

#### 需求
工作日每10分钟检查核心API可用性

#### 实现
```go
task := &io.CronTaskInfo{
    TaskName:       pointy.String("API健康检查"),
    CronExpression: pointy.String("*/10 9-18 * * 1-5"), // 工作日9-18点每10分钟
    InputSource:    pointy.String("api_health_check"),
    SourceConfig:   pointy.String(`{
        "endpoints": [
            {
                "name": "用户服务",
                "url": "https://api.example.com/health/user",
                "method": "GET",
                "timeout": 5
            },
            {
                "name": "订单服务",
                "url": "https://api.example.com/health/order",
                "method": "GET",
                "timeout": 5
            },
            {
                "name": "支付服务",
                "url": "https://api.example.com/health/payment",
                "method": "GET",
                "timeout": 10
            }
        ],
        "alert_on_failure": true,
        "alert_webhook": "https://alerts.example.com/webhook"
    }`),
    Description:    pointy.String("工作时间监控核心API健康状态"),
    Enabled:        pointy.Bool(true),
}

resp, err := ioClient.CreateCronTask(ctx, task)
```

---

## 6. 故障排查

### 6.1 任务未执行

#### 症状
CronTask已启用，但没有创建InputTask

#### 排查步骤

**1. 检查任务状态**
```go
task, err := ioClient.GetCronTaskById(ctx, &io.IDReq{Id: 123})

fmt.Printf("启用状态: %v\n", *task.Enabled)
fmt.Printf("下次执行: %s\n",
    time.Unix(int64(*task.NextRunTime), 0).Format("2006-01-02 15:04:05"))
```

**2. 检查CronScheduler日志**
```bash
# 查看unified-io RPC服务日志
tail -f /var/log/unified-io/rpc.log | grep -E "CronScheduler|cron_task"

# 期望看到的日志
{"level":"info","content":"CronScheduler started successfully","loaded_tasks":5}
{"level":"info","content":"Added cron task to scheduler","task_id":123}
```

**3. 验证Cron表达式**
```go
import "github.com/robfig/cron/v3"

parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
schedule, err := parser.Parse("0 2 * * *")
if err != nil {
    fmt.Printf("表达式无效: %v\n", err)
}
```

**4. 检查TaskWorker状态**
```yaml
# etc/io.yaml
TaskWorker:
  Enabled: true   # 确保已启用
```

---

### 6.2 任务执行失败

#### 症状
InputTask被创建，但状态为"failed"

#### 排查步骤

**1. 查询失败的InputTask**
```go
inputTasks, err := ioClient.GetInputTaskList(ctx, &io.InputTaskListReq{
    Page:       pointy.Uint64(1),
    PageSize:   pointy.Uint64(10),
    TaskStatus: pointy.String("failed"),
    TaskType:   pointy.String("cron"),
})

for _, task := range inputTasks.Data {
    fmt.Printf("失败任务: %s\n", *task.TaskName)
    fmt.Printf("错误信息: %s\n", *task.ErrorMessage)
}
```

**2. 检查SourceConfig配置**
```go
task, _ := ioClient.GetCronTaskById(ctx, &io.IDReq{Id: 123})

var config map[string]interface{}
json.Unmarshal([]byte(*task.SourceConfig), &config)

fmt.Printf("数据源配置: %+v\n", config)
// 验证endpoint、认证信息、参数等是否正确
```

**3. 手动触发测试**
```go
// 手动触发，观察执行结果
resp, err := ioClient.TriggerCronTaskNow(ctx, &io.IDReq{Id: 123})

// 等待执行完成
time.Sleep(30 * time.Second)

// 查询执行结果
inputTask, _ := ioClient.GetInputTaskById(ctx, &io.IDReq{Id: resp.Id})
fmt.Printf("执行状态: %s\n", *inputTask.TaskStatus)
fmt.Printf("错误信息: %s\n", *inputTask.ErrorMessage)
```

---

### 6.3 执行延迟

#### 症状
任务执行时间比预期晚

#### 可能原因

**1. TaskWorker资源不足**
```yaml
# etc/io.yaml
TaskWorker:
  MaxConcurrent: 5  # 增加并发数
  PullInterval: 5s  # 减少轮询间隔
```

**2. 大量pending任务堆积**
```go
// 检查pending队列长度
inputTasks, err := ioClient.GetInputTaskList(ctx, &io.InputTaskListReq{
    TaskStatus: pointy.String("pending"),
})

fmt.Printf("待处理任务数: %d\n", *inputTasks.Total)

// 如果超过100，考虑增加TaskWorker实例或MaxConcurrent
```

**3. 任务执行时间过长**
```go
// 查询平均执行时长
// 如果单个任务执行超过5分钟，考虑优化或拆分
```

---

### 6.4 统计数据不准确

#### 症状
execution_count、success_count不匹配

#### 排查步骤

**1. 验证统计逻辑**
```go
task, _ := ioClient.GetCronTaskById(ctx, &io.IDReq{Id: 123})

fmt.Printf("总执行: %d\n", *task.ExecutionCount)
fmt.Printf("成功: %d\n", *task.SuccessCount)
fmt.Printf("失败: %d\n", *task.FailureCount)
fmt.Printf("计算验证: %d = %d + %d? %v\n",
    *task.ExecutionCount,
    *task.SuccessCount,
    *task.FailureCount,
    *task.ExecutionCount == *task.SuccessCount + *task.FailureCount)
```

**2. 检查是否有运行中的任务**
```go
// 查询running状态的InputTask
runningTasks, err := ioClient.GetInputTaskList(ctx, &io.InputTaskListReq{
    TaskStatus:  pointy.String("running"),
    CronTaskId:  pointy.Uint64(123),
})

// running状态的任务还未计入success/failure统计
```

---

## 7. 高级功能

### 7.1 动态调整执行频率

#### 场景
根据业务负载动态调整监控频率

#### 实现
```go
// 高峰期：每5分钟
peakHourTask := &io.CronTaskInfo{
    Id:             pointy.Uint64(123),
    CronExpression: pointy.String("*/5 9-18 * * 1-5"), // 工作日9-18点
}
ioClient.UpdateCronTask(ctx, peakHourTask)

// 低峰期：每30分钟
offPeakTask := &io.CronTaskInfo{
    Id:             pointy.Uint64(123),
    CronExpression: pointy.String("*/30 0-8,19-23 * * *"), // 其他时段
}
ioClient.UpdateCronTask(ctx, offPeakTask)
```

---

### 7.2 任务依赖链（规划中）

#### 场景
任务B必须在任务A完成后执行

#### 未来设计
```go
// Phase 3功能预览
task := &io.CronTaskInfo{
    TaskName:       pointy.String("数据汇总"),
    CronExpression: pointy.String("0 4 * * *"),
    DependsOn:      []uint64{123}, // 依赖任务123
    WaitTimeout:    pointy.Uint32(3600), // 最多等待1小时
}
```

---

### 7.3 失败重试（规划中）

#### 场景
任务执行失败后自动重试

#### 未来设计
```go
// Phase 3功能预览
task := &io.CronTaskInfo{
    TaskName:       pointy.String("API数据同步"),
    CronExpression: pointy.String("0 * * * *"),
    RetryConfig:    &io.RetryConfig{
        MaxRetries:    pointy.Uint32(3),
        RetryInterval: pointy.Uint32(300), // 5分钟
        BackoffType:   pointy.String("exponential"),
    },
}
```

---

## 8. FAQ

### Q1: CronTask和InputTask的关系？
**A**: CronTask是定时任务的"规则"，InputTask是每次执行的"实例"。一个CronTask可以创建多个InputTask。

### Q2: 修改Cron表达式会影响已创建的InputTask吗？
**A**: 不会。修改只影响未来的执行，已创建的InputTask不受影响。

### Q3: 如何确保任务不重复执行？
**A**: CronScheduler使用`SkipIfStillRunning`中间件，如果上次执行未完成，本次会跳过。

### Q4: 任务可以跨租户执行吗？
**A**: 不可以。每个CronTask属于特定租户，只能访问该租户的数据。

### Q5: 时区如何处理？
**A**: 所有时间使用UTC时区，避免夏令时问题。显示时需要转换为本地时区。

### Q6: 如何备份CronTask配置？
**A**: 建议定期导出CronTask列表和配置，或使用数据库备份。

### Q7: 最多可以创建多少个CronTask？
**A**: 没有硬性限制，但建议不超过1000个。过多任务会影响CronScheduler性能。

### Q8: TaskWorker挂了会影响CronTask吗？
**A**: CronScheduler会继续创建InputTask，但TaskWorker恢复前不会执行。

---

## 附录

### A. 相关文档
- **实现总结**: `PHASE2_CRON_IMPLEMENTATION_SUMMARY.md`
- **测试报告**: `PHASE2_CRON_TESTS_SUMMARY.md`
- **API参考**: `io.proto`

### B. 示例代码仓库
```bash
# 克隆示例代码
git clone https://github.com/example/unified-io-examples.git
cd unified-io-examples/cron-tasks

# 运行示例
go run examples/cron_basic.go
go run examples/cron_monitoring.go
go run examples/cron_backup.go
```

### C. 联系我们
- 📧 Email: support@example.com
- 💬 Slack: #unified-io-support
- 📖 Wiki: https://wiki.example.com/unified-io

---

**文档版本**: v1.0.0
**最后更新**: 2025-12-24
**维护人员**: Unified-IO Team
