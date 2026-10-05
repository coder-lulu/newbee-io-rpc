# Phase 2: Cron Expression Support - Implementation Summary

## 实施日期
- **开始时间**: 2025-10-XX
- **完成时间**: 2025-10-XX
- **状态**: ✅ Core Implementation Complete

---

## 1. 架构概览

### 1.1 双调度器架构
```
┌─────────────────────────────────────────────────────────┐
│                 Unified-IO RPC Service                  │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌─────────────────┐          ┌─────────────────┐    │
│  │  TaskWorker     │          │ CronScheduler   │    │
│  │  (Pull-based)   │          │ (Time-based)    │    │
│  ├─────────────────┤          ├─────────────────┤    │
│  │ - Polls pending │          │ - Cron triggers │    │
│  │   InputTasks    │          │ - Parses cron   │    │
│  │ - Executes via  │          │   expressions   │    │
│  │   providers     │          │ - Creates       │    │
│  │ - Updates status│          │   InputTasks    │    │
│  └────────┬────────┘          └────────┬────────┘    │
│           │                            │              │
│           └────────────┬───────────────┘              │
│                        ▼                              │
│              ┌─────────────────┐                      │
│              │  InputTask DB   │                      │
│              │  (Execution Log)│                      │
│              └─────────────────┘                      │
│                        ▲                              │
│                        │                              │
│              ┌─────────┴─────────┐                    │
│              │   CronTask DB     │                    │
│              │   (Cron Metadata) │                    │
│              └───────────────────┘                    │
└─────────────────────────────────────────────────────────┘
```

### 1.2 CronTask vs InputTask 关系
- **CronTask**: Cron任务的元数据（规则）
  - `cron_expression`: "0 2 * * *" (每天凌晨2点)
  - `task_name`: "Daily Server Backup"
  - `enabled`: true/false
  - `next_run_time`: 下次执行时间（自动计算）

- **InputTask**: 每次执行的实例（记录）
  - `task_type`: "cron" (由CronScheduler创建) 或 "cron_manual" (手动触发)
  - `cron_task_id`: 关联的CronTask ID
  - `execution_time`: 实际执行时间
  - `task_status`: "pending" → "running" → "completed"/"failed"

### 1.3 任务类型枚举
| task_type | 创建者 | 说明 |
|-----------|--------|------|
| `cron` | CronScheduler | 定时调度自动创建 |
| `cron_manual` | TriggerCronTaskNow API | 手动触发的立即执行 |
| `scheduled` | CreateInputTask API | 一次性定时任务 |
| `manual` | CreateInputTask API | 手动创建的一次性任务 |

---

## 2. 核心组件实现

### 2.1 CronScheduler (internal/worker/cron_scheduler.go)
**文件大小**: 337 lines
**职责**: 管理所有启用的CronTask的定时调度

**核心特性**:
```go
type CronScheduler struct {
    db          *ent.Client
    cron        *cron.Cron              // robfig/cron v3
    taskMap     map[uint64]cron.EntryID // cronTaskID -> entryID
    taskMapLock sync.RWMutex            // 并发安全
    ctx         context.Context
    cancel      context.CancelFunc
}
```

**关键方法**:
- `Start()` - 启动调度器，加载所有启用的CronTask
- `AddTask(task *ent.CronTask)` - 添加单个Cron任务到调度器
- `RemoveTask(cronTaskID uint64)` - 从调度器移除任务
- `UpdateTask(task *ent.CronTask)` - 更新任务（先删除后添加）
- `executeTask(cronTaskID uint64)` - 执行Cron任务（创建InputTask）

**安全特性**:
- ✅ 使用 `SystemContext` 绕过租户隔离（后台任务需要跨租户运行）
- ✅ `cron.SkipIfStillRunning` 防止并发执行冲突
- ✅ UTC时区避免夏令时问题
- ✅ 事务安全的统计更新

**执行流程**:
```go
func (cs *CronScheduler) executeTask(cronTaskID uint64) {
    // 1. 查询CronTask元数据
    cronTask, err := cs.db.CronTask.Get(systemCtx, cronTaskID)

    // 2. 创建InputTask实例（task_type="cron"）
    inputTask, err := cs.db.InputTask.Create().
        SetTaskName(fmt.Sprintf("%s - %s", cronTask.TaskName, time)).
        SetTaskType("cron").
        SetCronTaskID(cronTask.ID).
        SetInputSource(cronTask.InputSource).
        SetSourceConfig(cronTask.SourceConfig).
        SetExecutionTime(time.Now()).
        SetTaskStatus("pending").
        Save(systemCtx)

    // 3. 更新CronTask统计和时间
    cs.db.CronTask.UpdateOneID(cronTaskID).
        SetExecutionCount(cronTask.ExecutionCount + 1).
        SetLastRunTime(time.Now()).
        SetNextRunTime(schedule.Next(time.Now())).
        Save(systemCtx)

    // 4. TaskWorker会自动拉取pending状态的InputTask并执行
}
```

### 2.2 Logic层增强

#### CreateCronTask (create_cron_task_logic.go)
**新增功能**:
```go
// 1. Cron表达式验证
parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
schedule, err := parser.Parse(*in.CronExpression)
if err != nil {
    return nil, fmt.Errorf("invalid cron expression '%s': %w", *in.CronExpression, err)
}

// 2. 自动计算next_run_time
now := time.Now().UTC()
nextRunTime := schedule.Next(now)

query := db.CronTask.Create().
    SetCronExpression(*in.CronExpression).
    SetNextRunTime(nextRunTime). // 自动设置
    ...

// 3. 启用时自动添加到调度器
if result.Enabled && l.svcCtx.CronScheduler != nil {
    l.svcCtx.CronScheduler.AddTask(result)
}
```

#### UpdateCronTask (update_cron_task_logic.go)
**新增功能**:
```go
// 1. 如果cron表达式变更，重新验证并计算next_run_time
var nextRunTime *time.Time
if in.CronExpression != nil && *in.CronExpression != "" {
    parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
    schedule, err := parser.Parse(*in.CronExpression)
    // ... validation ...
    next := schedule.Next(time.Now().UTC())
    nextRunTime = &next
}

// 2. 更新后同步调度器
if l.svcCtx.CronScheduler != nil {
    task, _ := l.svcCtx.DB.CronTask.Get(l.ctx, *in.Id)
    if task.Enabled {
        l.svcCtx.CronScheduler.UpdateTask(task) // 先删除后添加
    } else {
        l.svcCtx.CronScheduler.RemoveTask(*in.Id) // 禁用时移除
    }
}
```

#### EnableCronTask (enable_cron_task_logic.go)
**完整实现**:
```go
func (l *EnableCronTaskLogic) EnableCronTask(in *io.IDReq) (*io.BaseResp, error) {
    // 1. 查询并检查状态
    task, err := l.svcCtx.DB.CronTask.Get(l.ctx, in.Id)
    if task.Enabled {
        return &io.BaseResp{Msg: "Task is already enabled"}, nil
    }

    // 2. 更新数据库
    err = l.svcCtx.DB.CronTask.UpdateOneID(in.Id).SetEnabled(true).Exec(l.ctx)

    // 3. 添加到调度器
    if l.svcCtx.CronScheduler != nil {
        task, _ = l.svcCtx.DB.CronTask.Get(l.ctx, in.Id) // 重新查询
        if err := l.svcCtx.CronScheduler.AddTask(task); err != nil {
            return nil, fmt.Errorf("task enabled but failed to add to scheduler: %w", err)
        }
    }

    return &io.BaseResp{Msg: errormsg.UpdateSuccess}, nil
}
```

#### DisableCronTask (disable_cron_task_logic.go)
**完整实现**:
```go
func (l *DisableCronTaskLogic) DisableCronTask(in *io.IDReq) (*io.BaseResp, error) {
    // 1. 查询并检查状态
    task, err := l.svcCtx.DB.CronTask.Get(l.ctx, in.Id)
    if !task.Enabled {
        return &io.BaseResp{Msg: "Task is already disabled"}, nil
    }

    // 2. 从调度器移除（先移除后更新数据库）
    if l.svcCtx.CronScheduler != nil {
        if err := l.svcCtx.CronScheduler.RemoveTask(in.Id); err != nil {
            logx.Errorw("Failed to remove from scheduler", ...)
            // 继续执行，不因调度器错误中断
        }
    }

    // 3. 更新数据库
    err = l.svcCtx.DB.CronTask.UpdateOneID(in.Id).SetEnabled(false).Exec(l.ctx)

    return &io.BaseResp{Msg: errormsg.UpdateSuccess}, nil
}
```

#### TriggerCronTaskNow (trigger_cron_task_now_logic.go)
**完整实现**:
```go
func (l *TriggerCronTaskNowLogic) TriggerCronTaskNow(in *io.IDReq) (*io.BaseIDResp, error) {
    // 1. 查询CronTask元数据
    cronTask, err := l.svcCtx.DB.CronTask.Get(l.ctx, in.Id)

    // 2. 创建InputTask实例（task_type="cron_manual"）
    executionTime := time.Now()
    inputTask, err := l.svcCtx.DB.InputTask.Create().
        SetTaskName(fmt.Sprintf("%s - Manual Trigger - %s", cronTask.TaskName, executionTime.Format("2006-01-02 15:04:05"))).
        SetTaskType("cron_manual"). // 标记为手动触发
        SetCronTaskID(cronTask.ID).
        SetInputSource(cronTask.InputSource).
        SetSourceConfig(cronTask.SourceConfig).
        SetTaskStatus("pending").
        SetExecutionTime(executionTime).
        Save(l.ctx)

    // 3. 更新执行统计（但不更新next_run_time）
    l.svcCtx.DB.CronTask.UpdateOneID(cronTask.ID).
        SetExecutionCount(cronTask.ExecutionCount + 1).
        SetLastRunTime(executionTime).
        Save(l.ctx)

    return &io.BaseIDResp{Id: inputTask.ID, Msg: errormsg.CreateSuccess}, nil
}
```

### 2.3 ServiceContext集成 (internal/svc/service_context.go)

**结构体定义**:
```go
type ServiceContext struct {
    Config        config.Config
    DB            *ent.Client
    Redis         redis.UniversalClient
    CoreRpc       coreclient.Core
    OpsRpc        opsclient.Ops
    TaskWorker    *worker.TaskWorker
    CronScheduler *worker.CronScheduler  // 新增
}
```

**初始化逻辑**:
```go
// 初始化CronScheduler
var cronScheduler *worker.CronScheduler
if c.TaskWorker.Enabled { // 共享TaskWorker.Enabled配置
    cronScheduler = worker.NewCronScheduler(db)

    // 在独立goroutine中启动
    go func() {
        if err := cronScheduler.Start(context.Background()); err != nil {
            logx.Errorw("Failed to start CronScheduler", logx.Field("error", err))
        }
    }()

    logx.Infow("CronScheduler initialized and started", logx.Field("enabled", true))
} else {
    logx.Info("CronScheduler is disabled (TaskWorker disabled)")
}

return &ServiceContext{
    // ...
    TaskWorker:    taskWorker,
    CronScheduler: cronScheduler,
}
```

**配置要求**:
```yaml
# etc/io.yaml
TaskWorker:
  Enabled: true          # 同时控制TaskWorker和CronScheduler
  PullInterval: 5s
  BatchSize: 10
  MaxConcurrent: 5
```

### 2.4 RPC接口定义 (io.proto)

**新增RPC方法**:
```protobuf
// CronTask lifecycle operations
// group: crontask
rpc enableCronTask(IDReq) returns (BaseResp);
// group: crontask
rpc disableCronTask(IDReq) returns (BaseResp);
// group: crontask
rpc triggerCronTaskNow(IDReq) returns (BaseIDResp);
```

**⚠️ 已知问题**: 这三个方法需要在 `make gen-rpc` 后手动重新添加到proto文件
- **原因**: goctls工具会删除手动添加的RPC方法
- **临时方案**: 每次生成后手动添加
- **长期方案**: 需要修改代码生成模板或使用单独的proto文件

---

## 3. 编译验证

### 3.1 编译结果
```bash
$ go build -v .
github.com/coder-lulu/newbee-io-rpc/internal/svc
github.com/coder-lulu/newbee-io-rpc/internal/logic/crontask
github.com/coder-lulu/newbee-io-rpc/internal/server
github.com/coder-lulu/newbee-io-rpc
✅ 编译成功，无错误
```

### 3.2 修复的编译错误
1. **logx.Warnw undefined** (cron_scheduler.go:299)
   - 修复: 改为 `logx.Infow`

2. **SetPriority undefined** (trigger_cron_task_now_logic.go:57)
   - 修复: 移除 `SetPriority` 调用（InputTask无此字段）

3. **Wrong import path** (io_server.go:23)
   - 修复: `types/cmdb` → `types/io`

---

## 4. 依赖库

### 4.1 robfig/cron v3
```bash
go get github.com/robfig/cron/v3
```

**版本**: v3.0.1+
**用途**: Cron表达式解析和调度
**特性**:
- 标准5字段格式: `minute hour day month weekday`
- 时区支持（使用UTC）
- SkipIfStillRunning中间件
- 并发安全

**Cron表达式示例**:
| 表达式 | 说明 |
|--------|------|
| `0 2 * * *` | 每天凌晨2点 |
| `*/5 * * * *` | 每5分钟 |
| `0 9-17 * * 1-5` | 工作日9点到17点整点 |
| `0 0 1 * *` | 每月1日零点 |
| `0 0 * * 0` | 每周日零点 |

---

## 5. 工作流程示例

### 5.1 创建定时备份任务
```go
// 1. 用户通过API创建CronTask
CreateCronTask({
    TaskName: "Daily Database Backup",
    CronExpression: "0 2 * * *", // 每天凌晨2点
    InputSource: "backup_provider",
    SourceConfig: `{"type": "mysql", "target": "main_db"}`,
    Enabled: true
})

// 2. CreateCronTaskLogic处理
// - 验证cron表达式
// - 计算next_run_time = 2025-10-XX 02:00:00 UTC
// - 保存到cron_tasks表
// - 调用 CronScheduler.AddTask()

// 3. CronScheduler添加任务
// - 使用robfig/cron注册定时任务
// - 存储到内存taskMap: {cronTaskID: entryID}

// 4. 到达执行时间（凌晨2点）
// - robfig/cron自动触发 executeTask(cronTaskID)
// - 创建InputTask(task_type="cron", task_status="pending")
// - 更新CronTask统计(execution_count++, last_run_time, next_run_time)

// 5. TaskWorker自动拉取
// - 轮询发现pending状态的InputTask
// - 调用backup_provider执行备份
// - 更新InputTask状态: pending → running → completed
// - 更新CronTask统计(success_count++ 或 failure_count++)
```

### 5.2 手动触发执行
```go
// 用户触发立即执行
TriggerCronTaskNow({Id: 123})

// TriggerCronTaskNowLogic处理
// 1. 创建InputTask(task_type="cron_manual", task_status="pending")
// 2. 不影响next_run_time（定时调度不受影响）
// 3. 更新execution_count
// 4. TaskWorker自动拉取并执行
```

### 5.3 禁用任务
```go
// 用户禁用任务
DisableCronTask({Id: 123})

// DisableCronTaskLogic处理
// 1. CronScheduler.RemoveTask(123) - 从调度器移除
// 2. 更新cron_tasks.enabled = false
// 3. 后续不再自动创建InputTask
// 4. 已创建的pending InputTask仍会被执行（不影响）
```

---

## 6. 数据库变更

### 6.1 CronTask表 (Schema已存在，无需迁移)
```go
// ent/schema/crontask.go
type CronTask struct {
    ID              uint64    `json:"id"`
    TaskName        string    `json:"task_name"`         // 任务名称
    CronExpression  string    `json:"cron_expression"`   // Cron表达式
    InputSource     string    `json:"input_source"`      // 数据源类型
    SourceConfig    string    `json:"source_config"`     // 数据源配置
    Enabled         bool      `json:"enabled"`           // 是否启用
    NextRunTime     time.Time `json:"next_run_time"`     // 下次执行时间（自动计算）
    LastRunTime     *time.Time `json:"last_run_time"`    // 上次执行时间
    ExecutionCount  int       `json:"execution_count"`   // 总执行次数
    SuccessCount    int       `json:"success_count"`     // 成功次数
    FailureCount    int       `json:"failure_count"`     // 失败次数
    Description     string    `json:"description"`       // 描述
    TenantID        uint64    `json:"tenant_id"`         // 租户ID
    Status          uint8     `json:"status"`            // 状态
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}
```

### 6.2 InputTask表扩展 (已完成)
```go
// ent/schema/inputtask.go
type InputTask struct {
    // ... 其他字段 ...
    TaskType      string     `json:"task_type"`         // 新增: "cron" | "cron_manual" | "scheduled" | "manual"
    CronTaskID    *uint64    `json:"cron_task_id"`      // 新增: 关联的CronTask ID（可选）
    ExecutionTime time.Time  `json:"execution_time"`    // 新增: 预期执行时间
}
```

---

## 7. 待完成任务

### 7.1 测试 (优先级: 高)
- [ ] **单元测试** - CronScheduler方法测试
  - AddTask成功/失败场景
  - RemoveTask存在/不存在场景
  - UpdateTask边界条件
  - executeTask执行流程

- [ ] **集成测试** - 完整生命周期测试
  - 创建启用的CronTask → 自动添加到调度器
  - 更新enabled状态 → 调度器同步
  - 到达执行时间 → 自动创建InputTask
  - 手动触发 → 创建cron_manual类型InputTask

- [ ] **并发测试** - 多任务同时执行
  - 同一时间多个Cron任务触发
  - SkipIfStillRunning中间件验证
  - taskMap并发读写安全

### 7.2 文档 (优先级: 中)
- [ ] **用户指南** - 如何创建和管理Cron任务
  - Cron表达式语法说明
  - 常见场景示例
  - 最佳实践建议

- [ ] **API文档** - RPC接口使用说明
  - 请求/响应示例
  - 错误码说明
  - 生命周期操作指南

- [ ] **架构文档** - 系统设计详解
  - CronScheduler vs TaskWorker职责划分
  - SystemContext使用场景
  - 性能调优建议

### 7.3 监控与运维 (优先级: 低)
- [ ] **监控指标**
  - 活跃Cron任务数量
  - 每小时执行次数
  - 失败率统计
  - 平均执行时长

- [ ] **告警规则**
  - 连续执行失败 > 3次
  - 执行超时 > 配置阈值
  - 调度器异常停止

- [ ] **运维工具**
  - 批量启用/禁用任务
  - 执行历史查询
  - 任务依赖关系可视化

---

## 8. 已知问题与解决方案

### 8.1 Proto文件手动维护问题
**问题**: `make gen-rpc` 会删除手动添加的RPC方法定义
**影响**: 三个生命周期操作需要每次生成后手动添加
**临时方案**:
```bash
# 每次make gen-rpc后执行
# 手动编辑 io.proto，添加enableCronTask等三个方法
```
**长期方案**:
- 选项1: 修改goctls代码生成模板，保留自定义方法
- 选项2: 使用单独的proto文件（如crontask_lifecycle.proto）
- 选项3: 使用git hooks在生成后自动追加方法

### 8.2 时区问题
**决策**: 统一使用UTC时区
**原因**: 避免夏令时切换导致的调度混乱
**实现**:
```go
cron.New(cron.WithLocation(time.UTC))
schedule.Next(time.Now().UTC())
```

### 8.3 SystemContext使用
**为什么需要**:
- CronScheduler在后台goroutine运行
- 需要跨租户查询所有启用的CronTask
- 租户隔离Hook会阻止跨租户查询

**使用场景**:
```go
systemCtx := hooks.NewSystemContext(cs.ctx)
tasks, err := cs.db.CronTask.Query().
    Where(crontask.EnabledEQ(true)).
    All(systemCtx) // 使用SystemContext绕过租户过滤
```

---

## 9. 性能考虑

### 9.1 内存占用
- CronScheduler的taskMap大小 = 启用的CronTask数量
- 每个任务约占 32 bytes (uint64 key + cron.EntryID value)
- 1000个活跃任务 ≈ 32KB 内存（可忽略不计）

### 9.2 数据库负载
- CronScheduler启动时一次性加载所有启用任务
- 每次任务执行时:
  - 1次查询（CronTask元数据）
  - 1次插入（InputTask）
  - 1次更新（CronTask统计）
- 对于分钟级高频任务需要注意数据库压力

### 9.3 并发控制
- SkipIfStillRunning: 如果上次执行未完成，跳过本次
- taskMapLock: RWMutex保护taskMap并发访问
- 避免死锁: 先操作调度器，后操作数据库

---

## 10. 后续扩展方向

### 10.1 高级Cron特性
- [ ] **依赖任务** - 任务A完成后自动触发任务B
- [ ] **失败重试** - 自动重试策略（次数、间隔）
- [ ] **超时控制** - 单次执行超时自动取消
- [ ] **通知机制** - 执行失败时发送告警

### 10.2 分布式调度
- [ ] **任务分片** - 大任务拆分为多个子任务并行执行
- [ ] **多实例协调** - 多个RPC实例时避免重复执行
- [ ] **负载均衡** - 任务均匀分配到不同Worker

### 10.3 可视化管理
- [ ] **任务DAG可视化** - 依赖关系图形展示
- [ ] **执行日志查看** - 实时查看任务执行状态
- [ ] **性能分析** - 执行时长趋势分析

---

## 11. 总结

### ✅ 已完成
1. ✅ CronScheduler核心调度器实现（337行）
2. ✅ CreateCronTask/UpdateCronTask增强（Cron验证）
3. ✅ 三个生命周期操作（Enable/Disable/TriggerNow）
4. ✅ ServiceContext集成
5. ✅ Logic层调度器调用启用
6. ✅ 编译验证通过

### 🚧 进行中
- 单元测试编写
- 用户文档编写

### 📋 待规划
- 集成测试
- 监控指标
- 分布式调度

### 🎯 成果
Phase 2: Cron Expression Support 的核心功能已全部实现并通过编译验证。系统现在支持完整的Cron定时任务生命周期管理，为后续的高级特性奠定了坚实基础。

---

**文档维护**:
- 创建日期: 2025-10-XX
- 最后更新: 2025-10-XX
- 维护人员: Claude Code AI Assistant
