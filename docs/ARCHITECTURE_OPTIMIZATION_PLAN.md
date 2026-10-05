# Unified-IO 架构优化方案

> **文档版本**: v1.0
> **创建日期**: 2025-12-25
> **状态**: 📋 规划中

---

## 📊 执行摘要

本文档基于Phase 2完成后的现状,提出系统性的架构优化方案,重点关注:
1. **职责分离** - 明确各组件边界
2. **配置管理** - 统一配置存储和访问
3. **功能分配** - 合理分配干活vs配置职责
4. **可扩展性** - 支持新功能快速接入
5. **可观测性** - 完善监控和运维能力

---

## 目录

1. [当前架构分析](#1-当前架构分析)
2. [核心问题识别](#2-核心问题识别)
3. [优化架构设计](#3-优化架构设计)
4. [职责分离方案](#4-职责分离方案)
5. [配置管理架构](#5-配置管理架构)
6. [功能分配矩阵](#6-功能分配矩阵)
7. [实施路线图](#7-实施路线图)

---

## 1. 当前架构分析

### 1.1 组件现状

```
┌────────────────────────────────────────────────────────────┐
│                    当前架构 (Phase 2)                       │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  ┌─────────────────────┐      ┌─────────────────────┐   │
│  │  CronScheduler      │      │   TaskWorker        │   │
│  │  (337 lines)        │      │   (576 lines)       │   │
│  ├─────────────────────┤      ├─────────────────────┤   │
│  │ ❌ 职责混淆:         │      │ ❌ 职责混淆:         │   │
│  │ - Cron调度          │      │ - 任务拉取          │   │
│  │ - InputTask创建     │      │ - 任务执行          │   │
│  │ - 统计更新          │      │ - 状态更新          │   │
│  │ - next_run计算      │      │ - 业务逻辑(TODO)    │   │
│  └────────┬────────────┘      └────────┬────────────┘   │
│           │                            │                 │
│           └────────────┬───────────────┘                 │
│                        ▼                                 │
│              ┌─────────────────┐                         │
│              │  InputTask DB   │                         │
│              │  (混合存储)     │                         │
│              └─────────────────┘                         │
│                        ▲                                 │
│                        │                                 │
│              ┌─────────┴─────────┐                       │
│              │   CronTask DB     │                       │
│              │   (配置+状态)     │                       │
│              └───────────────────┘                       │
└────────────────────────────────────────────────────────────┘

⚠️ 问题识别:
1. 职责边界模糊 - 调度器做了太多事情
2. 配置分散 - DB、YAML、代码三处配置
3. 无Provider抽象 - 业务逻辑硬编码在Worker
4. 监控缺失 - 没有统一的指标采集点
5. 扩展困难 - 新增任务类型需要改多处
```

### 1.2 代码度量

| 组件 | 代码行数 | 职责数量 | 评级 |
|------|---------|---------|------|
| **CronScheduler** | 337 | 6个 | ⚠️ 职责过多 |
| **TaskWorker** | 576 | 7个 | ⚠️ 职责过多 |
| **ServiceContext** | 163 | 5个 | ⚠️ 初始化集中 |
| **Logic层** | ~600 (8个文件) | 8个API | ✅ 合理 |

**问题**: 核心组件职责数量超过单一职责原则建议的1-3个

---

## 2. 核心问题识别

### 2.1 职责混淆问题

#### 问题1: CronScheduler职责过多

**当前职责** (6个):
```go
type CronScheduler struct {
    // 职责1: Cron调度管理
    cron *cron.Cron

    // 职责2: 任务映射维护
    taskMap map[uint64]cron.EntryID

    // 职责3: InputTask创建 ❌
    // 职责4: 统计数据更新 ❌
    // 职责5: 时间计算 ❌
    // 职责6: 数据库操作 ❌
}
```

**违反原则**:
- ❌ **单一职责原则** - 一个类应该只有一个改变的理由
- ❌ **关注点分离** - 调度器不应关心业务逻辑

**影响**:
- 难以测试 - 需要Mock数据库、InputTask、统计等
- 难以扩展 - 新增执行逻辑需要修改调度器
- 职责耦合 - 调度失败会影响统计更新

#### 问题2: TaskWorker职责过多

**当前职责** (7个):
```go
type TaskWorker struct {
    // 职责1: 任务拉取
    // 职责2: 并发控制
    // 职责3: 超时管理
    // 职责4: 状态更新
    // 职责5: 业务执行 ❌
    // 职责6: 指标统计
    // 职责7: Stale检查
}
```

**违反原则**:
- ❌ **开闭原则** - 新增Provider需要修改Worker代码
- ❌ **依赖倒置原则** - Worker直接依赖具体实现

### 2.2 配置管理问题

#### 配置分散在三处

```yaml
# 1. YAML配置 (etc/io.yaml)
TaskWorker:
  Enabled: true
  PullInterval: 5s
  BatchSize: 10
  MaxConcurrent: 5
```

```sql
-- 2. 数据库配置 (cron_tasks表)
CREATE TABLE cron_tasks (
    cron_expression VARCHAR(50),  -- 业务配置
    enabled BOOLEAN,              -- 运行时配置
    input_source VARCHAR(50),     -- 执行配置
    source_config TEXT            -- Provider配置
);
```

```go
// 3. 代码配置 (hard-coded)
config := &WorkerConfig{
    TaskTimeout:    5 * time.Minute,  // 硬编码超时
    StaleThreshold: 1 * time.Hour,    // 硬编码阈值
}
```

**问题**:
- ❌ 配置修改需要重启服务 (YAML)
- ❌ 配置修改需要手动SQL (DB)
- ❌ 配置修改需要重新编译 (代码)
- ❌ 没有配置版本管理
- ❌ 没有配置校验机制

### 2.3 扩展性问题

#### 新增任务类型的痛点

**当前流程** (需要修改5处):
```go
// 1. 修改 ent/schema/inputtask.go - 添加新task_type
field.String("task_type").Comment("manual|scheduled|cron|cron_manual|新类型")

// 2. 修改 TaskWorker.processOnce() - 添加过滤逻辑
inputtask.TaskTypeNEQ("新类型"),

// 3. 修改 TaskWorker.executeTask() - 添加处理分支
switch task.TaskType {
    case "新类型":
        return w.handle新类型(ctx, task)
}

// 4. 修改 Logic层 - 添加创建逻辑
func (l *Create新类型Logic) Create新类型(...) { ... }

// 5. 修改 Proto - 添加RPC方法
rpc create新类型(...) returns (...)
```

**问题**:
- ❌ 高耦合 - 新功能影响现有代码
- ❌ 易出错 - 容易遗漏某个修改点
- ❌ 难测试 - 需要回归测试所有路径

---

## 3. 优化架构设计

### 3.1 整体架构图

```
┌───────────────────────────────────────────────────────────────────┐
│                     优化后架构 (目标架构)                           │
├───────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              🎯 API层 (职责: 接口暴露)                    │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │   │
│  │  │ CronTask API│  │ InputTask   │  │ Config API  │    │   │
│  │  │   (CRUD)    │  │    API      │  │  (管理配置) │    │   │
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘    │   │
│  └─────────┼─────────────────┼─────────────────┼──────────────┘   │
│            │                 │                 │                  │
│  ┌─────────┼─────────────────┼─────────────────┼──────────────┐   │
│  │         │    🧠 业务编排层 (职责: 业务逻辑)  │              │   │
│  │  ┌──────▼──────┐   ┌──────▼──────┐   ┌─────▼──────┐    │   │
│  │  │CronTaskMgr  │   │InputTaskMgr │   │ConfigCenter│    │   │
│  │  │(任务管理)   │   │(执行管理)   │   │(配置管理)  │    │   │
│  │  └──────┬──────┘   └──────┬──────┘   └─────┬──────┘    │   │
│  └─────────┼───────────────────┼───────────────┼───────────────┘   │
│            │                   │               │                  │
│  ┌─────────┼───────────────────┼───────────────┼───────────────┐   │
│  │         │    ⚙️ 调度执行层 (职责: 定时+执行)  │              │   │
│  │  ┌──────▼──────┐   ┌───────▼────────┐  ┌──┴──────┐      │   │
│  │  │Scheduler    │   │  TaskExecutor  │  │Registry │      │   │
│  │  │(只负责调度) │───▶│  (只负责执行)  │◀─│(Provider│      │   │
│  │  └──────┬──────┘   └───────┬────────┘  │注册中心)│      │   │
│  │         │                   │           └─────────┘      │   │
│  │         │    ┌──────────────▼──────────────┐             │   │
│  │         └───▶│    TaskQueue (队列抽象)     │             │   │
│  │              │  - Priority Queue           │             │   │
│  │              │  - Delayed Queue            │             │   │
│  │              │  - Dead Letter Queue        │             │   │
│  │              └─────────────────────────────┘             │   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │           💾 数据存储层 (职责: 持久化)                      │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐│   │
│  │  │CronTask  │  │InputTask │  │Config DB │  │Metrics  ││   │
│  │  │  (配置)  │  │ (执行日志)│  │ (运行配置)│  │ (指标)  ││   │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘│   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │           🔌 Provider插件层 (职责: 业务实现)               │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐│   │
│  │  │MySQL     │  │API Caller│  │File      │  │Custom   ││   │
│  │  │Provider  │  │Provider  │  │Provider  │  │Provider ││   │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘│   │
│  └────────────────────────────────────────────────────────────┘   │
└───────────────────────────────────────────────────────────────────┘

✅ 优势:
1. 职责清晰 - 每层只做一件事
2. 解耦合理 - 层间通过接口交互
3. 易扩展 - 新Provider只需注册即可
4. 可测试 - 每层独立可测
5. 可观测 - 每层都有监控点
```

### 3.2 分层职责定义

| 层级 | 职责 | 输入 | 输出 | 示例组件 |
|------|------|------|------|---------|
| **API层** | 接口暴露 | gRPC请求 | gRPC响应 | `CronTaskAPI`, `ConfigAPI` |
| **业务编排层** | 业务逻辑+事务管理 | API调用 | 数据变更 | `CronTaskManager`, `InputTaskManager` |
| **调度执行层** | 定时调度+任务执行 | 配置+规则 | 任务队列 | `Scheduler`, `TaskExecutor` |
| **数据存储层** | 持久化 | 实体对象 | 查询结果 | `CronTaskRepo`, `ConfigRepo` |
| **Provider层** | 业务实现 | 任务配置 | 执行结果 | `MySQLProvider`, `APIProvider` |

---

## 4. 职责分离方案

### 4.1 Scheduler职责重新定义

#### 优化前 (❌ 职责过多)

```go
// 当前CronScheduler - 337行, 6个职责
type CronScheduler struct {
    db          *ent.Client       // ❌ 直接操作DB
    cron        *cron.Cron
    taskMap     map[uint64]cron.EntryID
    taskMapLock sync.RWMutex
}

func (cs *CronScheduler) executeTask(cronTaskID uint64) {
    // ❌ 职责1: 查询CronTask
    cronTask, _ := cs.db.CronTask.Get(systemCtx, cronTaskID)

    // ❌ 职责2: 创建InputTask
    inputTask, _ := cs.db.InputTask.Create()...

    // ❌ 职责3: 更新统计
    cs.db.CronTask.UpdateOneID(cronTaskID).
        SetExecutionCount(count + 1)...

    // ❌ 职责4: 计算next_run_time
    nextRunTime := schedule.Next(time.Now())

    // ❌ 职责5: 错误处理
    // ❌ 职责6: 日志记录
}
```

#### 优化后 (✅ 单一职责)

```go
// ✅ 新Scheduler - 只负责调度触发
type Scheduler struct {
    cron        *cron.Cron
    taskMap     map[uint64]cron.EntryID
    taskMapLock sync.RWMutex
    executor    TaskExecutor  // 依赖抽象,不依赖具体实现
}

// ✅ 唯一职责: 在正确的时间触发任务
func (s *Scheduler) executeTask(cronTaskID uint64) {
    // 只负责调用executor,不关心具体如何执行
    s.executor.Execute(cronTaskID)
}

// ✅ 配置分离
func (s *Scheduler) AddTask(task ScheduleConfig) error {
    entryID, err := s.cron.AddFunc(task.CronExpression, func() {
        s.executeTask(task.ID)
    })
    s.taskMap[task.ID] = entryID
    return nil
}
```

### 4.2 TaskExecutor职责定义

```go
// ✅ 新TaskExecutor - 只负责执行任务
type TaskExecutor struct {
    queue        TaskQueue         // 任务队列
    providerMgr  ProviderManager  // Provider管理器
    metrics      MetricsCollector  // 指标收集器
}

// ✅ 唯一职责: 执行任务并管理生命周期
func (e *TaskExecutor) Execute(cronTaskID uint64) error {
    // 1. 创建任务实例 (委托给TaskManager)
    taskID, err := e.taskManager.CreateInputTask(cronTaskID)
    if err != nil {
        return err
    }

    // 2. 加入队列
    e.queue.Push(&TaskItem{
        ID:       taskID,
        Priority: e.calculatePriority(cronTaskID),
    })

    return nil
}

// ✅ Worker循环 - 从队列消费
func (e *TaskExecutor) Run(ctx context.Context) {
    for {
        task := e.queue.Pop(ctx)
        go e.processTask(ctx, task)  // 异步执行
    }
}

// ✅ 任务处理 - 委托给Provider
func (e *TaskExecutor) processTask(ctx context.Context, task *TaskItem) {
    // 1. 更新状态为processing
    e.taskManager.UpdateStatus(task.ID, "processing")

    // 2. 获取Provider
    provider := e.providerMgr.GetProvider(task.InputSource)

    // 3. 执行
    result, err := provider.Execute(ctx, task.SourceConfig)

    // 4. 更新最终状态
    if err != nil {
        e.taskManager.UpdateStatus(task.ID, "failed")
        e.metrics.RecordFailure(task)
    } else {
        e.taskManager.UpdateStatus(task.ID, "completed")
        e.metrics.RecordSuccess(task)
    }
}
```

### 4.3 配置vs干活职责分离

```
┌─────────────────────────────────────────────────────────┐
│                 职责分离矩阵                             │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  📋 配置服务 (Configuration Services)                   │
│  ┌────────────────────────────────────────────────┐   │
│  │ ConfigCenter - 统一配置管理                     │   │
│  │  - 存储: config_items表                       │   │
│  │  - 功能: 增删改查、版本管理、校验             │   │
│  │  - 示例:                                      │   │
│  │    • Worker并发数: MaxConcurrent=5            │   │
│  │    • 拉取间隔: PullInterval=5s                │   │
│  │    • 任务超时: TaskTimeout=5m                 │   │
│  └────────────────────────────────────────────────┘   │
│                        │                               │
│                        │ 读取配置                       │
│                        ▼                               │
│  ⚙️ 执行服务 (Execution Services)                      │
│  ┌────────────────────────────────────────────────┐   │
│  │ Scheduler - 定时调度                           │   │
│  │  - 读取: CronTask配置                         │   │
│  │  - 执行: 按时间触发任务                       │   │
│  │  - 不写: 不修改任何配置                       │   │
│  ├────────────────────────────────────────────────┤   │
│  │ TaskExecutor - 任务执行                        │   │
│  │  - 读取: InputTask配置、Provider配置          │   │
│  │  - 执行: 调用Provider执行业务逻辑             │   │
│  │  - 写入: 执行日志、状态更新                   │   │
│  ├────────────────────────────────────────────────┤   │
│  │ Provider - 业务实现                            │   │
│  │  - 读取: SourceConfig (业务配置)              │   │
│  │  - 执行: 实际业务逻辑 (DB操作、API调用等)    │   │
│  │  - 写入: 业务数据                             │   │
│  └────────────────────────────────────────────────┘   │
│                        │                               │
│                        │ 写入执行结果                   │
│                        ▼                               │
│  💾 存储服务 (Storage Services)                        │
│  ┌────────────────────────────────────────────────┐   │
│  │ TaskManager - 任务状态管理                     │   │
│  │  - 写入: InputTask状态变更                    │   │
│  │  - 写入: CronTask统计更新                     │   │
│  │  - 查询: 执行历史                             │   │
│  └────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘

🔑 核心原则:
✅ 配置服务: 只读 + 管理
✅ 执行服务: 只读配置 + 执行 + 写日志
✅ 存储服务: 纯数据访问层
```

---

## 5. 配置管理架构

### 5.1 统一配置中心设计

#### 配置分类

```yaml
# 配置类型分类
配置类型:
  系统配置 (System Config):
    - 存储位置: config_items表 + YAML备份
    - 修改方式: API + 热重载
    - 示例:
      • task_worker.max_concurrent: 5
      • scheduler.timezone: "UTC"
      • executor.timeout_default: "5m"

  业务配置 (Business Config):
    - 存储位置: cron_tasks表
    - 修改方式: API
    - 示例:
      • task_name: "每日备份"
      • cron_expression: "0 2 * * *"
      • input_source: "mysql_backup"

  Provider配置 (Provider Config):
    - 存储位置: source_config字段 (JSON)
    - 修改方式: API
    - 示例:
      • mysql: {"host": "...", "db": "..."}
      • api: {"endpoint": "...", "method": "POST"}

  运行时配置 (Runtime Config):
    - 存储位置: Redis
    - 修改方式: 管理API + 自动过期
    - 示例:
      • rate_limit.api_calls: 100/minute
      • circuit_breaker.threshold: 50%
```

#### 配置Schema设计

```sql
-- =====================================================
-- 新表: config_items (统一配置中心)
-- =====================================================
CREATE TABLE config_items (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,

    -- 配置标识
    config_key VARCHAR(100) NOT NULL COMMENT '配置键 (eg: task_worker.max_concurrent)',
    config_category VARCHAR(50) NOT NULL COMMENT '配置类别 (system|business|provider)',

    -- 配置值
    config_value TEXT NOT NULL COMMENT '配置值 (支持JSON)',
    value_type VARCHAR(20) NOT NULL COMMENT '值类型 (int|string|duration|json)',
    default_value TEXT COMMENT '默认值',

    -- 约束和校验
    validation_rule TEXT COMMENT '校验规则 (JSON)',
    is_required BOOLEAN DEFAULT false,
    is_sensitive BOOLEAN DEFAULT false COMMENT '是否敏感配置 (密码、密钥等)',

    -- 版本管理
    version INT UNSIGNED DEFAULT 1,
    previous_value TEXT COMMENT '上一个版本的值',

    -- 生效控制
    is_enabled BOOLEAN DEFAULT true,
    effective_at TIMESTAMP NULL COMMENT '生效时间 (支持定时生效)',

    -- 元数据
    description TEXT COMMENT '配置说明',
    config_group VARCHAR(50) COMMENT '配置分组',
    tags JSON COMMENT '标签',

    -- 审计
    created_by BIGINT UNSIGNED,
    updated_by BIGINT UNSIGNED,
    tenant_id BIGINT UNSIGNED NOT NULL,
    status TINYINT UNSIGNED DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE KEY uk_config_key_tenant (config_key, tenant_id),
    INDEX idx_category (config_category),
    INDEX idx_group (config_group),
    INDEX idx_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='统一配置中心';

-- 示例配置数据
INSERT INTO config_items (config_key, config_category, config_value, value_type, description, config_group, tenant_id) VALUES
-- TaskWorker配置
('task_worker.enabled', 'system', 'true', 'boolean', 'TaskWorker是否启用', 'worker', 1),
('task_worker.pull_interval', 'system', '5s', 'duration', '任务拉取间隔', 'worker', 1),
('task_worker.max_concurrent', 'system', '5', 'int', '最大并发任务数', 'worker', 1),
('task_worker.batch_size', 'system', '10', 'int', '每次拉取任务数', 'worker', 1),

-- Scheduler配置
('scheduler.timezone', 'system', 'UTC', 'string', '调度器时区', 'scheduler', 1),
('scheduler.max_tasks', 'system', '1000', 'int', '最大Cron任务数', 'scheduler', 1),

-- Provider默认配置
('provider.mysql.timeout', 'provider', '30s', 'duration', 'MySQL Provider超时时间', 'provider', 1),
('provider.api.retry_count', 'provider', '3', 'int', 'API Provider重试次数', 'provider', 1);
```

#### ConfigCenter服务接口

```go
// =====================================================
// 新服务: ConfigCenter
// =====================================================
type ConfigCenter struct {
    db    *ent.Client
    cache *ConfigCache    // Redis缓存
    watcher *ConfigWatcher // 配置变更监听
}

// 配置项模型
type ConfigItem struct {
    Key          string
    Value        interface{}
    ValueType    string
    Category     string
    Version      int
    IsEnabled    bool
    ValidationRule *ValidationRule
}

// =====================================================
// API: 配置读取 (执行服务使用)
// =====================================================
func (cc *ConfigCenter) Get(key string) (interface{}, error) {
    // 1. 尝试从缓存读取
    if val, ok := cc.cache.Get(key); ok {
        return val, nil
    }

    // 2. 从DB读取
    item, err := cc.db.ConfigItem.Query().
        Where(configitem.ConfigKeyEQ(key)).
        First(ctx)

    // 3. 写入缓存
    cc.cache.Set(key, item.ConfigValue, 5*time.Minute)

    return item.ConfigValue, nil
}

// GetInt - 类型安全的读取
func (cc *ConfigCenter) GetInt(key string) (int, error) {
    val, err := cc.Get(key)
    if err != nil {
        return 0, err
    }
    return cast.ToInt(val), nil
}

// GetDuration - 读取时间间隔
func (cc *ConfigCenter) GetDuration(key string) (time.Duration, error) {
    val, err := cc.Get(key)
    if err != nil {
        return 0, err
    }
    return time.ParseDuration(cast.ToString(val))
}

// =====================================================
// API: 配置更新 (配置服务使用)
// =====================================================
func (cc *ConfigCenter) Set(key string, value interface{}) error {
    // 1. 校验配置值
    item, _ := cc.db.ConfigItem.Query().
        Where(configitem.ConfigKeyEQ(key)).
        First(ctx)

    if item.ValidationRule != nil {
        if err := cc.validate(value, item.ValidationRule); err != nil {
            return fmt.Errorf("validation failed: %w", err)
        }
    }

    // 2. 更新数据库 (保存旧版本)
    _, err := cc.db.ConfigItem.UpdateOneID(item.ID).
        SetPreviousValue(item.ConfigValue).  // 保存历史版本
        SetConfigValue(value).
        SetVersion(item.Version + 1).
        Save(ctx)

    // 3. 更新缓存
    cc.cache.Set(key, value, 5*time.Minute)

    // 4. 发布变更通知 (通过Redis Pub/Sub)
    cc.watcher.Publish(&ConfigChangeEvent{
        Key:      key,
        OldValue: item.ConfigValue,
        NewValue: value,
        Version:  item.Version + 1,
    })

    return nil
}

// =====================================================
// API: 配置热重载 (监听配置变更)
// =====================================================
type ConfigWatcher struct {
    redis    redis.UniversalClient
    handlers map[string][]ConfigChangeHandler
}

type ConfigChangeHandler func(event *ConfigChangeEvent)

// Watch - 注册配置变更监听器
func (cw *ConfigWatcher) Watch(key string, handler ConfigChangeHandler) {
    cw.handlers[key] = append(cw.handlers[key], handler)
}

// 使用示例:
func (te *TaskExecutor) init(cc *ConfigCenter) {
    // 监听max_concurrent变更,自动调整并发数
    cc.watcher.Watch("task_worker.max_concurrent", func(event *ConfigChangeEvent) {
        newMax := cast.ToInt(event.NewValue)
        te.semaphore = semaphore.NewWeighted(int64(newMax))
        logx.Infow("MaxConcurrent updated",
            logx.Field("old", event.OldValue),
            logx.Field("new", newMax))
    })
}
```

### 5.2 配置管理API

```protobuf
// =====================================================
// 新Proto: config.proto
// =====================================================
syntax = "proto3";

package config;

// 配置管理服务
service Config {
    // 查询配置
    rpc getConfig(ConfigKeyReq) returns (ConfigItem);
    rpc listConfigs(ConfigListReq) returns (ConfigListResp);

    // 更新配置
    rpc updateConfig(ConfigUpdateReq) returns (BaseResp);
    rpc batchUpdateConfigs(BatchConfigUpdateReq) returns (BaseResp);

    // 配置版本管理
    rpc getConfigHistory(ConfigKeyReq) returns (ConfigHistoryResp);
    rpc rollbackConfig(ConfigRollbackReq) returns (BaseResp);

    // 配置校验
    rpc validateConfig(ConfigValidateReq) returns (ValidationResult);

    // 配置导入导出
    rpc exportConfigs(ConfigExportReq) returns (ConfigExportResp);
    rpc importConfigs(ConfigImportReq) returns (BaseResp);
}

message ConfigItem {
    string config_key = 1;
    string config_value = 2;
    string value_type = 3;
    string config_category = 4;
    int32 version = 5;
    bool is_enabled = 6;
    string description = 7;
}

message ConfigUpdateReq {
    string config_key = 1;
    string config_value = 2;
    optional string change_reason = 3;  // 变更原因
    optional int64 effective_at = 4;     // 定时生效
}
```

---

## 6. 功能分配矩阵

### 6.1 组件功能对照表

| 功能 | 负责组件 | 角色 | 输入 | 输出 | 依赖 |
|------|---------|------|------|------|------|
| **配置管理** | ConfigCenter | 配置服务 | API请求 | 配置值 | DB+Redis |
| **定时触发** | Scheduler | 执行服务 | CronTask配置 | 触发事件 | ConfigCenter |
| **任务创建** | TaskManager | 业务服务 | CronTask元数据 | InputTask实例 | DB |
| **任务入队** | TaskQueue | 执行服务 | InputTask | - | Redis Queue |
| **任务拉取** | TaskExecutor | 执行服务 | - | 待执行任务 | TaskQueue |
| **任务执行** | Provider | 业务实现 | SourceConfig | 执行结果 | 外部系统 |
| **状态更新** | TaskManager | 业务服务 | 执行结果 | - | DB |
| **指标收集** | MetricsCollector | 监控服务 | 执行事件 | Prometheus指标 | - |
| **告警通知** | AlertManager | 监控服务 | 异常事件 | 通知 | 钉钉/邮件 |

### 6.2 数据流转图

```
用户API请求
    │
    ├──> [ConfigCenter] 更新配置
    │         │
    │         └──> [Redis Pub/Sub] 发布变更通知
    │                   │
    │                   └──> [Scheduler/TaskExecutor] 热重载
    │
    ├──> [CronTaskManager] 创建/更新CronTask
    │         │
    │         └──> [Scheduler] 添加/更新调度规则
    │                   │
    │                   └──> [定时触发]
    │                           │
    │                           ▼
    │                   [TaskManager] 创建InputTask
    │                           │
    │                           ▼
    │                   [TaskQueue] 任务入队
    │                           │
    └──> [手动触发API]          │
            │                   │
            └───────────────────┤
                                │
                                ▼
                    [TaskExecutor] 拉取任务
                                │
                                ├──> [ProviderRegistry] 获取Provider
                                │           │
                                │           └──> [MySQLProvider]
                                │           └──> [APIProvider]
                                │           └──> [CustomProvider]
                                │
                                ├──> [Provider.Execute()] 执行业务逻辑
                                │           │
                                │           └──> 外部系统 (MySQL/API/etc)
                                │
                                ├──> [TaskManager] 更新状态
                                │
                                └──> [MetricsCollector] 记录指标
                                            │
                                            └──> Prometheus
```

---

## 7. 实施路线图

### 7.1 Phase 3: 架构重构 (预估4周)

#### Week 1: 配置中心建设

**目标**: 建立统一配置管理基础设施

**任务清单**:
- [ ] 创建config_items表和Schema
- [ ] 实现ConfigCenter服务
- [ ] 实现ConfigWatcher (配置变更监听)
- [ ] 迁移现有配置到统一配置中心
- [ ] 编写配置管理API
- [ ] 配置热重载测试

**交付物**:
- `ent/schema/configitem.go`
- `internal/service/config_center.go`
- `internal/service/config_watcher.go`
- `config.proto` + 生成代码
- 配置迁移脚本

#### Week 2: Scheduler职责分离

**目标**: 将Scheduler从337行简化到150行

**任务清单**:
- [ ] 创建TaskManager服务 (负责InputTask CRUD)
- [ ] 从CronScheduler移除InputTask创建逻辑
- [ ] 从CronScheduler移除统计更新逻辑
- [ ] 重构executeTask方法 (只触发,不执行)
- [ ] 单元测试 (Mock TaskManager)

**交付物**:
- `internal/service/task_manager.go` (新)
- `internal/worker/scheduler.go` (重构)
- 单元测试

#### Week 3: Provider抽象层

**目标**: 建立可扩展的Provider体系

**任务清单**:
- [ ] 定义Provider接口
- [ ] 实现ProviderRegistry (Provider注册中心)
- [ ] 实现内置Providers:
  - [ ] MySQLProvider
  - [ ] APIProvider
  - [ ] FileProvider
- [ ] TaskExecutor集成ProviderRegistry
- [ ] Provider配置schema定义

**交付物**:
- `internal/provider/interface.go`
- `internal/provider/registry.go`
- `internal/provider/mysql_provider.go`
- `internal/provider/api_provider.go`
- `internal/provider/file_provider.go`

#### Week 4: 队列抽象+监控

**目标**: 完善任务队列和可观测性

**任务清单**:
- [ ] TaskQueue抽象接口
- [ ] Redis Queue实现
- [ ] Priority Queue支持
- [ ] MetricsCollector实现
- [ ] Prometheus指标暴露
- [ ] Grafana仪表板

**交付物**:
- `internal/queue/task_queue.go`
- `internal/queue/redis_queue.go`
- `internal/metrics/collector.go`
- Grafana Dashboard JSON

### 7.2 Phase 4: 高级特性 (预估4周)

**Week 5-6**: 任务依赖+失败重试
**Week 7-8**: 分布式调度+负载均衡

### 7.3 里程碑验证

| 里程碑 | 验证标准 | 目标日期 |
|--------|---------|---------|
| **M1: 配置中心上线** | 所有配置可通过API管理 | Week 1结束 |
| **M2: Scheduler重构完成** | 代码行数<150, 职责≤3个 | Week 2结束 |
| **M3: Provider体系建立** | 新Provider接入<1小时 | Week 3结束 |
| **M4: 监控体系完善** | Grafana可视化所有指标 | Week 4结束 |

---

## 8. 风险与挑战

### 8.1 技术风险

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| **配置热重载失败** | 高 | 保留YAML备份,支持降级 |
| **Provider性能问题** | 中 | 实现超时+熔断机制 |
| **队列堆积** | 高 | 监控队列长度,自动扩容 |
| **分布式锁冲突** | 中 | 使用Redlock算法 |

### 8.2 兼容性风险

| 场景 | 问题 | 解决方案 |
|------|------|---------|
| **现有CronTask迁移** | 配置格式变更 | 提供自动迁移脚本 |
| **API变更** | 客户端不兼容 | 保持向后兼容,版本共存 |
| **数据库Schema变更** | 停机时间 | 使用在线DDL工具 |

---

## 9. 成功指标

### 9.1 代码质量指标

| 指标 | 当前值 | 目标值 | 评估周期 |
|------|--------|--------|---------|
| **Scheduler代码行数** | 337 | <150 | Phase 3结束 |
| **TaskWorker代码行数** | 576 | <300 | Phase 3结束 |
| **单元测试覆盖率** | 60% | >80% | Phase 3结束 |
| **平均方法行数** | ~50 | <30 | Phase 3结束 |

### 9.2 运维指标

| 指标 | 当前值 | 目标值 | 评估周期 |
|------|--------|--------|---------|
| **配置变更生效时间** | 重启(分钟级) | 热重载(秒级) | Phase 3结束 |
| **新Provider接入时间** | 2天 | <1小时 | Phase 4结束 |
| **任务执行延迟P99** | 未知 | <1s | Phase 4结束 |
| **系统可用性** | 未知 | >99.9% | Phase 4结束 |

---

## 10. 总结

### 10.1 优化核心思路

1. **职责分离** - 单一职责原则,每个组件只做一件事
2. **配置统一** - 建立ConfigCenter,支持热重载
3. **抽象解耦** - Provider插件化,易扩展
4. **可观测性** - 全链路监控,Prometheus+Grafana
5. **渐进式演进** - 保持向后兼容,逐步迁移

### 10.2 预期收益

| 维度 | 收益 |
|------|------|
| **开发效率** | 新Provider接入时间从2天降至1小时 ⬇95% |
| **运维效率** | 配置变更无需重启,生效时间从分钟降至秒级 ⬇99% |
| **系统稳定性** | 故障隔离,单个Provider故障不影响全局 ⬆50% |
| **可测试性** | Mock容易,测试覆盖率从60%提升至80% ⬆33% |
| **代码质量** | 核心组件代码量减半,职责更清晰 ⬆100% |

---

**文档维护**:
- **作者**: Claude Code AI Assistant
- **创建日期**: 2025-12-25
- **最后更新**: 2025-12-25
- **文档版本**: v1.0
- **审阅状态**: 待审阅

**相关文档**:
- [Phase 2 实现总结](./PHASE2_CRON_IMPLEMENTATION_SUMMARY.md)
- [Phase 2 测试报告](./PHASE2_CRON_TESTS_SUMMARY.md)
- [用户指南](./CRON_USER_GUIDE.md)
