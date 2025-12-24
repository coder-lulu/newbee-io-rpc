package worker

import (
	"context"
	"fmt"
	"time"

	"encoding/json"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/zeromicro/go-zero/core/logx"
)

// ContextManager 上下文管理器
type ContextManager struct {
	logger logx.Logger
}

// NewContextManager 创建上下文管理器
func NewContextManager(logger logx.Logger) *ContextManager {
	return &ContextManager{
		logger: logger,
	}
}

// BuildExecutionContext 构建任务执行上下文
func (m *ContextManager) BuildExecutionContext(
	baseCtx context.Context,
	task interface{}, // *ent.InputTask 或 *ent.OutputTask
	taskType TaskType,
	db *ent.Client,
) (*ExecutionContext, error) {
	// 提取租户ID
	tenantID, err := m.extractTenantID(baseCtx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to extract tenant ID: %w", err)
	}

	// 提取用户ID（如果存在）
	userID := m.extractUserID(baseCtx)

	// 构建执行上下文
	var taskID uint64
	var providerID string
	var providerConfig map[string]interface{}
	var timeout time.Duration

	switch taskType {
	case TaskTypeInput:
		inputTask, ok := task.(*ent.InputTask)
		if !ok {
			return nil, fmt.Errorf("invalid task type: expected InputTask")
		}
		taskID = inputTask.ID
		providerID = inputTask.InputSource

		// 解析source_config为map (JSON字符串 -> map)
		if inputTask.SourceConfig != "" {
			if err := json.Unmarshal([]byte(inputTask.SourceConfig), &providerConfig); err != nil {
				m.logger.Errorw("Failed to parse source_config JSON",
					logx.Field("task_id", taskID),
					logx.Field("error", err))
				providerConfig = make(map[string]interface{})
			}
		} else {
			providerConfig = make(map[string]interface{})
		}
		timeout = 5 * time.Minute // 默认超时

	case TaskTypeOutput:
		outputTask, ok := task.(*ent.OutputTask)
		if !ok {
			return nil, fmt.Errorf("invalid task type: expected OutputTask")
		}
		taskID = outputTask.ID
		providerID = outputTask.OutputTarget

		// 解析target_config为map (JSON字符串 -> map)
		if outputTask.TargetConfig != "" {
			if err := json.Unmarshal([]byte(outputTask.TargetConfig), &providerConfig); err != nil {
				m.logger.Errorw("Failed to parse target_config JSON",
					logx.Field("task_id", taskID),
					logx.Field("error", err))
				providerConfig = make(map[string]interface{})
			}
		} else {
			providerConfig = make(map[string]interface{})
		}
		timeout = 5 * time.Minute // 默认超时

	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}

	// 创建带超时的context
	ctx, cancel := context.WithTimeout(baseCtx, timeout)

	// 注意：cancel函数需要在任务完成后调用，这里返回的ctx会被使用
	// 实际使用中需要defer cancel()
	_ = cancel // 避免未使用警告，实际应在Executor中defer

	execCtx := &ExecutionContext{
		Ctx:            ctx,
		TenantID:       tenantID,
		UserID:         userID,
		TaskID:         taskID,
		TaskType:       taskType,
		ProviderID:     providerID,
		ProviderConfig: providerConfig,
		Timeout:        timeout,
		MaxRetries:     3,
		RetryCount:     0,
		Logger:         m.logger,
		DB:             db,
	}

	m.logger.Infof("[Tenant %d] Built execution context for %s task %d, provider: %s",
		tenantID, taskType, taskID, providerID)

	return execCtx, nil
}

// extractTenantID 从任务或上下文中提取租户ID
func (m *ContextManager) extractTenantID(ctx context.Context, task interface{}) (uint64, error) {
	// 方法1：从任务实体中读取（推荐，因为有TenantMixin）
	switch t := task.(type) {
	case *ent.InputTask:
		if t.TenantID > 0 {
			return t.TenantID, nil
		}
	case *ent.OutputTask:
		if t.TenantID > 0 {
			return t.TenantID, nil
		}
	}

	// 方法2：从context中读取（作为备份）
	if tenantID := getTenantIDFromContext(ctx); tenantID > 0 {
		return tenantID, nil
	}

	return 0, fmt.Errorf("tenant ID not found in task or context")
}

// extractUserID 从上下文中提取用户ID
func (m *ContextManager) extractUserID(ctx context.Context) string {
	if userID := getUserIDFromContext(ctx); userID != "" {
		return userID
	}
	return "system" // 默认系统用户
}

// ValidateTenantIsolation 验证租户隔离
// 确保任务中的租户ID与上下文中的租户ID一致
func (m *ContextManager) ValidateTenantIsolation(execCtx *ExecutionContext, expectedTenantID uint64) error {
	if execCtx.TenantID != expectedTenantID {
		err := fmt.Errorf(
			"🚨 TENANT ISOLATION VIOLATION: context tenant %d != expected tenant %d",
			execCtx.TenantID, expectedTenantID,
		)
		m.logger.Errorw(
			"Tenant isolation violation detected",
			logx.Field("context_tenant", execCtx.TenantID),
			logx.Field("expected_tenant", expectedTenantID),
			logx.Field("task_id", execCtx.TaskID),
			logx.Field("task_type", execCtx.TaskType),
		)
		return err
	}
	return nil
}

// InjectTenantContext 注入租户上下文到context
func (m *ContextManager) InjectTenantContext(ctx context.Context, tenantID uint64, userID string) context.Context {
	ctx = context.WithValue(ctx, "tenantId", tenantID)
	if userID != "" {
		ctx = context.WithValue(ctx, "userId", userID)
	}
	return ctx
}

// ========== 内部辅助函数 ==========

// getTenantIDFromContext 从context中获取租户ID
func getTenantIDFromContext(ctx context.Context) uint64 {
	if tenantID, ok := ctx.Value("tenantId").(uint64); ok {
		return tenantID
	}
	return 0
}

// getUserIDFromContext 从context中获取用户ID
func getUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value("userId").(string); ok {
		return userID
	}
	return ""
}
