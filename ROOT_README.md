# CMDB-RPC 统一CI数据操作架构

> **版本**: 1.0.0  
> **更新时间**: 2024-12-19  
> **状态**: ✅ 已完成核心架构实现  

## 🚀 项目概览

本项目实现了CMDB系统的统一CI数据操作架构，提供标准化、安全、高性能的CI数据管理基础设施。

### 🎯 核心价值

- **统一操作流程**: 所有CI数据操作通过统一接口进行
- **完善权限控制**: 基于RBAC+ABAC的细粒度权限管理
- **全程审计追踪**: 完整的操作日志和变更记录
- **灵活审批流程**: 支持多种审批模式和自定义流程
- **高性能架构**: 支持批量操作、异步执行、缓存优化

## 🏗️ 项目框架

```
cmdb-rpc/
├── internal/core/              # 核心架构实现
│   ├── types.go               # 类型定义
│   ├── ci_data_manager.go     # 数据管理器
│   ├── component_impls.go     # 组件实现
│   ├── data_validator.go      # 数据校验器
│   ├── components.go          # 组件接口
│   └── example_usage.go       # 使用示例
├── ent/                       # 数据实体 (自动生成)
│   ├── cioperation.go         # CI操作记录实体
│   ├── cipermission.go        # CI权限配置实体
│   ├── ciapprovalflow.go      # CI审批流程实体
│   └── cilifecyclestate.go    # CI生命周期状态实体
├── doc/                       # 项目文档
│   └── ci_data_architecture.md # 架构设计文档
└── README.md                  # 项目说明
```

## 🔧 核心功能 (版本 1.0)

### ✅ 已实现功能

1. **资产管理**
   - ✅ CI增删改查操作
   - ✅ 批量数据处理
   - ✅ 异步操作支持

2. **数据校验**
   - ✅ Schema结构校验
   - ✅ 业务规则校验
   - ✅ 唯一性约束检查
   - ✅ 集成现有ValidateCisAttributesLogic

3. **变更记录**
   - ✅ 操作日志记录
   - ✅ 数据变更追踪
   - ✅ 变更历史查询

4. **生命周期管理**
   - ✅ 状态机管理
   - ✅ 状态转换控制
   - ✅ 超时处理

5. **权限控制**
   - ✅ 操作级权限控制
   - ✅ 字段级权限管理
   - ✅ 批量操作权限验证
   - ✅ Casbin集成框架

6. **审批流程**
   - ✅ 审批需求检查
   - ✅ 审批流程提交
   - ✅ 审批状态管理
   - 🔄 待完成: 工单系统集成

### 🚧 规划中功能

- **监控告警**: 性能监控和异常告警
- **数据同步**: 与外部系统的数据同步
- **备份恢复**: 数据备份和恢复机制

## 🛠️ 技术架构

### 核心组件

| 组件 | 职责 | 状态 |
|------|------|------|
| CiDataManager | 统一数据管理器，协调所有组件 | ✅ 完成 |
| DataValidator | 数据校验，集成现有校验逻辑 | ✅ 完成 |
| PermissionChecker | 权限检查，支持多级权限控制 | ✅ 完成 |
| ChangeRecorder | 变更记录，完整操作审计 | ✅ 完成 |
| LifecycleManager | 生命周期管理，状态控制 | ✅ 完成 |
| ApprovalManager | 审批管理，流程控制 | ✅ 完成 |
| DataPersister | 数据持久化，数据库操作 | ✅ 完成 |

### 技术栈

- **语言**: Go 1.19+
- **框架**: go-zero
- **ORM**: Ent
- **数据库**: MySQL 8.0+
- **权限**: Casbin
- **协议**: gRPC + Protocol Buffers

## 📚 已知问题

### 🟡 中等优先级
1. **示例文档**: 需要更多实际业务场景的使用示例
2. **性能测试**: 需要进行压力测试和性能基准测试
3. **错误处理**: 需要完善错误分类和错误码定义

### 🟢 低优先级
1. **国际化**: 支持多语言错误信息
2. **配置热更新**: 支持配置的动态更新
3. **插件系统**: 支持第三方插件扩展

## 📖 文档索引

### 核心文档
- [架构设计文档](doc/ci_data_architecture.md) - 详细的架构设计和实现说明
- [使用示例](internal/core/example_usage.go) - 完整的代码使用示例

### API文档
- [接口定义](internal/core/types.go) - 核心类型和接口定义
- [组件接口](internal/core/components.go) - 各组件的接口规范

### 开发文档
- [数据实体](ent/) - 数据库实体定义
- [组件实现](internal/core/component_impls.go) - 组件具体实现

## 🚀 快速开始

### 1. 初始化数据管理器

```go
import "gitee.com/link234/cmdb-rpc/internal/core"

// 创建配置
config := core.DefaultConfiguration()
config.Validation.DefaultLevel = core.ValidationStrict

// 创建管理器
manager := core.NewCiDataManager(svcCtx, config)

// 初始化
ctx := context.Background()
err := manager.Initialize(ctx)
```

### 2. 基本操作示例

```go
// 创建CI
operation := &core.CiOperationContext{
    Type:            core.OperationCreate,
    Source:          core.SourceManual,
    OperatorID:      userID,
    CiTypeID:        typeID,
    ValidationLevel: core.ValidationStrict,
    DataAfter:       ciData,
}

result, err := manager.CreateCi(ctx, operation)
```

### 3. 批量操作示例

```go
// 批量创建
batchOp := &core.CiOperationContext{
    Type:           core.OperationBatchCreate,
    BatchData:      ciList,
    AsyncExecution: true,
}

result, err := manager.BatchCreate(ctx, batchOp)
```

## 🔧 配置说明

### 关键配置项

```go
type Configuration struct {
    Validation struct {
        DefaultLevel ValidationLevel  // 默认校验级别
        EnableCache  bool            // 启用校验缓存
        Timeout      time.Duration   // 校验超时时间
    }
    
    Permission struct {
        DefaultLevel PermissionLevel  // 默认权限级别
        EnableCache  bool            // 启用权限缓存
    }
    
    Performance struct {
        BatchSize      int           // 批量操作大小
        EnableMetrics  bool          // 启用性能监控
    }
}
```

## 🤝 开发指南

### 兼容性承诺
- ✅ 100%向后兼容现有gRPC接口
- ✅ 平滑集成现有ValidateCisAttributesLogic
- ✅ 支持逐步迁移现有代码

### 扩展开发
- 🔧 支持自定义校验器
- 🔧 支持自定义权限检查器
- 🔧 支持自定义审批流程

### 测试覆盖
- 📝 单元测试: 待完善
- 📝 集成测试: 待完善
- 📝 性能测试: 待完善

## 📞 联系方式

- **团队**: CMDB开发团队
- **文档维护**: 架构组
- **问题反馈**: 请提交Issue或联系开发团队

---

**最后更新**: 2024-12-19  
**下次计划更新**: 2024-12-26 (工单系统集成完成后) 