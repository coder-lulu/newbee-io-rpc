package core

import (
	"context"
	"fmt"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/cmdb"
	"github.com/gofrs/uuid/v5"
)

// ExampleUsage 统一CI数据操作架构使用示例
func ExampleUsage(svcCtx *svc.ServiceContext) {
	ctx := context.Background()

	// 1. 创建并初始化CI数据管理器
	fmt.Println("=== 1. 初始化CI数据管理器 ===")

	config := DefaultConfiguration()
	config.Validation.DefaultLevel = ValidationStrict
	config.Permission.DefaultLevel = PermissionWrite
	config.Performance.EnableMetrics = true

	manager := NewCiDataManager(svcCtx, config)

	// 初始化管理器
	if err := manager.Initialize(ctx); err != nil {
		fmt.Printf("初始化失败: %v\n", err)
		return
	}
	defer manager.Shutdown(ctx)

	// 健康检查
	if err := manager.HealthCheck(ctx); err != nil {
		fmt.Printf("健康检查失败: %v\n", err)
		return
	}
	fmt.Println("CI数据管理器初始化成功")

	// 2. 创建CI实例示例
	fmt.Println("\n=== 2. 创建CI实例 ===")

	operatorID, _ := uuid.NewV4()
	createOperation := &CiOperationContext{
		Type:         OperationCreate,
		Source:       SourceManual,
		Reason:       "创建新的服务器CI实例",
		OperatorID:   operatorID,
		OperatorName: "张三",
		CiTypeID:     1, // 假设服务器类型ID为1
		DataAfter: &cmdb.CisInfo{
			TypeId: func() *uint64 { id := uint64(1); return &id }(),
			// CI名称通过属性值来设置，不是直接的字段
			// 其他属性...
		},
		ValidationLevel:     ValidationStrict,
		RequireApproval:     true,
		SkipPermissionCheck: false,
		AsyncExecution:      false,
		RequestTime:         time.Now(),
		Timeout:             30 * time.Second,
		ClientIP:            "192.168.1.100",
		UserAgent:           "CMDB-Client/1.0",
		Metadata: map[string]interface{}{
			"department": "IT部门",
			"project":    "基础设施建设",
		},
		Tags: []string{"production", "web-server"},
	}

	createResult, err := manager.CreateCi(ctx, createOperation)
	if err != nil {
		fmt.Printf("创建CI失败: %v\n", err)
	} else {
		fmt.Printf("创建CI成功: %+v\n", createResult)

		// 如果需要审批，提交审批
		if createResult.RequireApproval {
			fmt.Println("CI创建需要审批，正在提交审批流程...")
			if err := manager.SubmitForApproval(ctx, createResult.OperationID); err != nil {
				fmt.Printf("提交审批失败: %v\n", err)
			} else {
				fmt.Println("已提交审批，等待审批人处理")
			}
		}
	}

	// 3. 更新CI实例示例
	fmt.Println("\n=== 3. 更新CI实例 ===")

	if createResult != nil && createResult.CreatedCiID != nil {
		updateOperation := &CiOperationContext{
			Type:         OperationUpdate,
			Source:       SourceAPI,
			Reason:       "更新服务器配置信息",
			OperatorID:   operatorID,
			OperatorName: "张三",
			CiID:         createResult.CreatedCiID,
			CiTypeID:     1,
			DataBefore: &cmdb.CisInfo{
				Id:     createResult.CreatedCiID,
				TypeId: func() *uint64 { id := uint64(1); return &id }(),
			},
			DataAfter: &cmdb.CisInfo{
				Id:     createResult.CreatedCiID,
				TypeId: func() *uint64 { id := uint64(1); return &id }(),
			},
			ValidationLevel: ValidationBasic,
			RequireApproval: false,
			RequestTime:     time.Now(),
		}

		updateResult, err := manager.UpdateCi(ctx, updateOperation)
		if err != nil {
			fmt.Printf("更新CI失败: %v\n", err)
		} else {
			fmt.Printf("更新CI成功: %+v\n", updateResult)
		}
	}

	// 4. 批量创建CI实例示例
	fmt.Println("\n=== 4. 批量创建CI实例 ===")

	batchCreateOperation := &CiOperationContext{
		Type:         OperationBatchCreate,
		Source:       SourceImport,
		Reason:       "批量导入服务器资产",
		OperatorID:   operatorID,
		OperatorName: "李四",
		CiTypeID:     1,
		BatchData: []*cmdb.CisInfo{
			{TypeId: func() *uint64 { id := uint64(1); return &id }()},
			{TypeId: func() *uint64 { id := uint64(1); return &id }()},
			{TypeId: func() *uint64 { id := uint64(1); return &id }()},
		},
		ValidationLevel: ValidationFull,
		RequireApproval: true,
		AsyncExecution:  true, // 异步执行批量操作
		RequestTime:     time.Now(),
		Metadata: map[string]interface{}{
			"batch_source": "excel_import",
			"import_file":  "servers_20240101.xlsx",
		},
	}

	batchResult, err := manager.BatchCreate(ctx, batchCreateOperation)
	if err != nil {
		fmt.Printf("批量创建CI失败: %v\n", err)
	} else {
		fmt.Printf("批量创建CI成功: %+v\n", batchResult)

		// 监控异步操作状态
		if batchResult.Status == StatusProcessing {
			fmt.Println("异步操作正在处理中，定期检查状态...")
			go func() {
				for i := 0; i < 10; i++ {
					time.Sleep(2 * time.Second)
					status, err := manager.GetOperationStatus(ctx, batchResult.OperationID)
					if err != nil {
						fmt.Printf("获取操作状态失败: %v\n", err)
						break
					}
					fmt.Printf("操作状态: %s\n", status.Status)
					if status.Status == StatusCompleted || status.Status == StatusFailed {
						break
					}
				}
			}()
		}
	}

	// 5. 权限检查示例
	fmt.Println("\n=== 5. 权限检查示例 ===")

	permissionTestOperation := &CiOperationContext{
		Type:            OperationDelete,
		Source:          SourceManual,
		OperatorID:      operatorID,
		OperatorName:    "王五",
		CiID:            createResult.CreatedCiID,
		CiTypeID:        1,
		ValidationLevel: ValidationBasic,
	}

	// 这里会自动进行权限检查
	deleteResult, err := manager.DeleteCi(ctx, permissionTestOperation)
	if err != nil {
		fmt.Printf("删除操作失败（可能权限不足）: %v\n", err)
	} else {
		fmt.Printf("删除操作成功: %+v\n", deleteResult)
	}

	// 6. 变更历史查询示例
	fmt.Println("\n=== 6. 变更历史查询 ===")

	if createResult != nil && createResult.CreatedCiID != nil {
		history, err := manager.GetChangeHistory(ctx, *createResult.CreatedCiID, 10, 0)
		if err != nil {
			fmt.Printf("查询变更历史失败: %v\n", err)
		} else {
			fmt.Printf("变更历史记录数: %d\n", len(history))
			for i, record := range history {
				fmt.Printf("记录 %d: %s - %s - %v\n",
					i+1, record.OperationType, record.OperatorName, record.OperationTime)
			}
		}
	}

	// 7. 操作日志查询示例
	fmt.Println("\n=== 7. 操作日志查询 ===")

	filters := map[string]interface{}{
		"operator_id": operatorID.String(),
		"time_range": map[string]interface{}{
			"start": time.Now().Add(-1 * time.Hour),
			"end":   time.Now(),
		},
	}

	operationLogs, err := manager.GetOperationLog(ctx, filters)
	if err != nil {
		fmt.Printf("查询操作日志失败: %v\n", err)
	} else {
		fmt.Printf("操作日志记录数: %d\n", len(operationLogs))
		for i, log := range operationLogs {
			fmt.Printf("日志 %d: %s - %s - %s\n",
				i+1, log.OperationID, log.Type, log.OperatorName)
		}
	}

	// 8. 审批工作流示例
	fmt.Println("\n=== 8. 审批工作流示例 ===")

	// 模拟审批人处理审批
	if createResult != nil && createResult.ApprovalFlowID != "" {
		approverID, _ := uuid.NewV4()

		// 审批通过
		err := manager.ProcessApproval(ctx, createResult.OperationID, "approve", "审批通过，符合IT资产管理规范", approverID)
		if err != nil {
			fmt.Printf("处理审批失败: %v\n", err)
		} else {
			fmt.Println("审批处理成功")

			// 检查审批后的操作状态
			finalStatus, err := manager.GetOperationStatus(ctx, createResult.OperationID)
			if err != nil {
				fmt.Printf("获取最终状态失败: %v\n", err)
			} else {
				fmt.Printf("最终操作状态: %s\n", finalStatus.Status)
			}
		}
	}

	// 9. 数据同步示例
	fmt.Println("\n=== 9. 数据同步示例 ===")

	syncOperation := &CiOperationContext{
		Type:            OperationSync,
		Source:          SourceSystem,
		Reason:          "与外部CMDB系统同步数据",
		OperatorID:      operatorID,
		OperatorName:    "系统自动同步",
		CiTypeID:        1,
		ValidationLevel: ValidationBasic,
		AsyncExecution:  true,
		RequestTime:     time.Now(),
		Metadata: map[string]interface{}{
			"sync_source": "external_cmdb",
			"sync_type":   "incremental",
		},
	}

	syncResult, err := manager.SyncData(ctx, syncOperation)
	if err != nil {
		fmt.Printf("数据同步失败: %v\n", err)
	} else {
		fmt.Printf("数据同步启动成功: %+v\n", syncResult)
	}

	fmt.Println("\n=== 示例完成 ===")
	fmt.Println("统一CI数据操作架构示例演示完毕")
}

// ExampleCustomValidator 自定义校验器示例
func ExampleCustomValidator() {
	fmt.Println("=== 自定义校验器示例 ===")

	// 可以创建自定义的校验器，实现特定的业务规则
	// 例如：服务器命名规范校验、IP地址格式校验等

	fmt.Println("自定义校验器可以实现:")
	fmt.Println("1. 业务特定的命名规范")
	fmt.Println("2. 数据完整性检查")
	fmt.Println("3. 跨CI关联性校验")
	fmt.Println("4. 合规性检查")
}

// ExampleCustomPermissionChecker 自定义权限检查器示例
func ExampleCustomPermissionChecker() {
	fmt.Println("=== 自定义权限检查器示例 ===")

	fmt.Println("自定义权限检查器可以实现:")
	fmt.Println("1. 基于角色的访问控制(RBAC)")
	fmt.Println("2. 基于属性的访问控制(ABAC)")
	fmt.Println("3. 数据级权限控制")
	fmt.Println("4. 时间窗口权限限制")
	fmt.Println("5. 地理位置权限限制")
}

// ExampleIntegrationWithExistingLogic 与现有逻辑集成示例
func ExampleIntegrationWithExistingLogic(svcCtx *svc.ServiceContext) {
	fmt.Println("=== 与现有逻辑集成示例 ===")

	// 新架构完全兼容现有的ValidateCisAttributesLogic
	ctx := context.Background()
	manager := NewCiDataManager(svcCtx, nil)

	// 在现有的gRPC接口中使用
	operation := &CiOperationContext{
		Type:            OperationCreate,
		Source:          SourceAPI,
		CiTypeID:        1,
		ValidationLevel: ValidationStrict,
		// ... 其他字段
	}

	result, err := manager.CreateCi(ctx, operation)
	if err != nil {
		fmt.Printf("集成调用失败: %v\n", err)
		return
	}

	fmt.Printf("集成调用成功: %+v\n", result)
	fmt.Println("现有代码可以逐步迁移到新架构")
}
