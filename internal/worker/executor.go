package worker

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/ent/fieldmapping"
	"github.com/coder-lulu/newbee-io-rpc/internal/provider"
	"github.com/coder-lulu/newbee-io-rpc/internal/transform"
	"github.com/zeromicro/go-zero/core/logx"
)

// Executor 任务执行器
type Executor struct {
	config           *ExecutorConfig
	logger           logx.Logger
	stateMachine     *StateMachine
	contextManager   *ContextManager
	providerRegistry *provider.ProviderRegistry
	transformEngine  *transform.Engine  // Transform引擎
	outputProcessor  *OutputProcessor   // 输出处理器
	db               *ent.Client        // 数据库客户端（用于查询FieldMapping）
}

// NewExecutor 创建任务执行器
func NewExecutor(
	config *ExecutorConfig,
	db *ent.Client,
	outputProcessor *OutputProcessor,
	logger logx.Logger,
) *Executor {
	if config == nil {
		config = DefaultExecutorConfig()
	}

	// 初始化 Transform Engine（使用默认配置）
	transformEngine := transform.NewEngine(nil, db, logger)

	return &Executor{
		config:           config,
		logger:           logger,
		stateMachine:     NewStateMachine(db, logger),
		contextManager:   NewContextManager(logger),
		providerRegistry: provider.GetRegistry(),
		transformEngine:  transformEngine,
		outputProcessor:  outputProcessor,
		db:               db,
	}
}

// Execute 执行任务
func (e *Executor) Execute(execCtx *ExecutionContext) *TaskResult {
	startTime := time.Now()
	result := &TaskResult{
		Status:    TaskStatusFailed, // 默认失败，成功后更新
		StartedAt: startTime,
	}

	// Panic恢复
	if e.config.PanicRecovery {
		defer func() {
			if r := recover(); r != nil {
				result.Status = TaskStatusFailed
				result.ErrorMessage = fmt.Sprintf("PANIC: %v\nStack: %s", r, debug.Stack())
				result.CompletedAt = time.Now()
				result.Duration = time.Since(startTime)

				e.logger.Errorw(
					"Task execution panicked",
					logx.Field("task_id", execCtx.TaskID),
					logx.Field("task_type", execCtx.TaskType),
					logx.Field("panic", r),
					logx.Field("stack", string(debug.Stack())),
				)
			}
		}()
	}

	// 1. 验证租户隔离
	if err := e.validateTenantIsolation(execCtx); err != nil {
		result.ErrorMessage = err.Error()
		result.CompletedAt = time.Now()
		result.Duration = time.Since(startTime)
		return result
	}

	// 2. 获取Provider
	discoveryProvider, err := e.getProvider(execCtx)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to get provider: %v", err)
		result.CompletedAt = time.Now()
		result.Duration = time.Since(startTime)
		return result
	}

	// 3. 执行Provider发现逻辑
	e.logger.Infow(
		"Executing provider discovery",
		logx.Field("task_id", execCtx.TaskID),
		logx.Field("provider_id", execCtx.ProviderID),
		logx.Field("tenant_id", execCtx.TenantID),
	)

	discoveryResult, err := e.executeProviderDiscovery(execCtx, discoveryProvider)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Provider execution failed: %v", err)
		result.CompletedAt = time.Now()
		result.Duration = time.Since(startTime)
		return result
	}

	// 4. 应用Transform转换
	transformedRecords, transformStats, err := e.applyTransform(execCtx, discoveryResult.Records)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Transform failed: %v", err)
		result.CompletedAt = time.Now()
		result.Duration = time.Since(startTime)
		return result
	}

	// 5. 处理发现结果
	result.Status = TaskStatusCompleted
	result.TotalRecords = discoveryResult.TotalRecords
	result.SuccessRecords = int64(len(transformedRecords))
	result.ProcessedRecords = discoveryResult.TotalRecords
	result.FailedRecords = int64(transformStats.FailedFields)
	result.Data = transformedRecords
	result.Metadata = discoveryResult.Metadata

	// 添加Transform统计到元数据
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	result.Metadata["transform_stats"] = map[string]interface{}{
		"total_fields":   transformStats.TotalFields,
		"success_fields": transformStats.SuccessFields,
		"failed_fields":  transformStats.FailedFields,
		"skipped_fields": transformStats.SkippedFields,
		"duration_ms":    transformStats.Duration.Milliseconds(),
	}

	result.CompletedAt = time.Now()
	result.Duration = time.Since(startTime)

	e.logger.Infow(
		"Task execution completed successfully",
		logx.Field("task_id", execCtx.TaskID),
		logx.Field("total_records", result.TotalRecords),
		logx.Field("duration", result.Duration.Seconds()),
	)

	return result
}

// validateTenantIsolation 验证租户隔离
func (e *Executor) validateTenantIsolation(execCtx *ExecutionContext) error {
	if execCtx.TenantID == 0 {
		return fmt.Errorf("tenant ID is required but not found in execution context")
	}

	// 记录租户隔离检查
	e.logger.Infow(
		"Tenant isolation validated",
		logx.Field("task_id", execCtx.TaskID),
		logx.Field("tenant_id", execCtx.TenantID),
		logx.Field("user_id", execCtx.UserID),
	)

	return nil
}

// getProvider 获取Provider
func (e *Executor) getProvider(execCtx *ExecutionContext) (provider.IDiscoveryProvider, error) {
	if execCtx.ProviderID == "" {
		return nil, fmt.Errorf("provider ID is empty")
	}

	discoveryProvider, err := e.providerRegistry.Get(execCtx.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("provider %s not found: %w", execCtx.ProviderID, err)
	}

	return discoveryProvider, nil
}

// executeProviderDiscovery 执行Provider发现逻辑
func (e *Executor) executeProviderDiscovery(
	execCtx *ExecutionContext,
	discoveryProvider provider.IDiscoveryProvider,
) (*provider.DiscoveryResult, error) {
	// 创建带超时的context
	ctx, cancel := context.WithTimeout(execCtx.Ctx, e.config.ProviderTimeout)
	defer cancel()

	// 准备Provider配置
	config := execCtx.ProviderConfig
	if config == nil {
		config = make(map[string]interface{})
	}

	// 注入租户信息到配置（用于日志和审计）
	config["_tenant_id"] = execCtx.TenantID
	config["_user_id"] = execCtx.UserID
	config["_task_id"] = execCtx.TaskID

	// 🎯 转换为V2 Provider接口(支持context传递)
	// - 如果Provider原生支持V2,直接使用
	// - 如果是V1 Provider,自动使用适配器包装
	providerV2, err := provider.ToProviderV2(discoveryProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to convert provider to V2: %w", err)
	}

	// 使用V2接口执行发现(支持context取消和超时)
	result, err := providerV2.DiscoverWithContext(ctx, config)
	if err != nil {
		// 检查是否是context超时/取消导致的错误
		if ctx.Err() != nil {
			return nil, fmt.Errorf("provider discovery cancelled/timeout: %w", ctx.Err())
		}
		return nil, fmt.Errorf("provider discovery failed: %w", err)
	}

	return result, nil
}

// ExecuteWithStateUpdate 执行任务并自动更新状态
func (e *Executor) ExecuteWithStateUpdate(execCtx *ExecutionContext) error {
	// 1. 标记任务为运行中
	if err := e.stateMachine.MarkTaskAsRunning(execCtx.Ctx, execCtx.TaskID, execCtx.TaskType); err != nil {
		e.logger.Errorw(
			"Failed to mark task as running",
			logx.Field("task_id", execCtx.TaskID),
			logx.Field("error", err.Error()),
		)
		return err
	}

	// 2. 执行任务
	result := e.Execute(execCtx)

	// 3. 根据执行结果更新任务状态
	var updateErr error
	if result.Status == TaskStatusCompleted {
		updateErr = e.stateMachine.MarkTaskAsCompleted(execCtx.Ctx, execCtx.TaskID, execCtx.TaskType, result)
	} else {
		updateErr = e.stateMachine.MarkTaskAsFailed(execCtx.Ctx, execCtx.TaskID, execCtx.TaskType, result)
	}

	if updateErr != nil {
		e.logger.Errorw(
			"Failed to update task state after execution",
			logx.Field("task_id", execCtx.TaskID),
			logx.Field("final_status", result.Status),
			logx.Field("error", updateErr.Error()),
		)
		return updateErr
	}

	return nil
}

// ExecuteInputTask 执行输入任务（便捷方法）
func (e *Executor) ExecuteInputTask(ctx context.Context, task *ent.InputTask, db *ent.Client) error {
	// 构建执行上下文
	execCtx, err := e.contextManager.BuildExecutionContext(ctx, task, TaskTypeInput, db)
	if err != nil {
		return fmt.Errorf("failed to build execution context: %w", err)
	}

	// 执行并更新状态
	return e.ExecuteWithStateUpdate(execCtx)
}

// ExecuteOutputTask 执行输出任务（便捷方法）
func (e *Executor) ExecuteOutputTask(ctx context.Context, task *ent.OutputTask, db *ent.Client) error {
	// 构建执行上下文
	execCtx, err := e.contextManager.BuildExecutionContext(ctx, task, TaskTypeOutput, db)
	if err != nil {
		return fmt.Errorf("failed to build execution context: %w", err)
	}

	// 1. 标记任务为运行中
	if err := e.stateMachine.MarkTaskAsRunning(execCtx.Ctx, execCtx.TaskID, TaskTypeOutput); err != nil {
		e.logger.Errorw(
			"Failed to mark output task as running",
			logx.Field("task_id", execCtx.TaskID),
			logx.Field("error", err.Error()),
		)
		return err
	}

	// 2. 执行发现和转换（如果OutputTask配置了provider）
	// 或者从关联的InputTask获取数据
	result := e.Execute(execCtx)

	// 3. 如果执行失败，标记任务失败
	if result.Status != TaskStatusCompleted {
		updateErr := e.stateMachine.MarkTaskAsFailed(execCtx.Ctx, execCtx.TaskID, TaskTypeOutput, result)
		if updateErr != nil {
			e.logger.Errorw(
				"Failed to mark output task as failed",
				logx.Field("task_id", execCtx.TaskID),
				logx.Field("error", updateErr.Error()),
			)
		}
		return fmt.Errorf("task execution failed: %s", result.ErrorMessage)
	}

	// 4. 调用OutputProcessor处理输出
	if e.outputProcessor != nil {
		e.logger.Infow(
			"Processing output to target",
			logx.Field("task_id", task.ID),
			logx.Field("output_target", task.OutputTarget),
			logx.Field("records_count", len(result.Data)),
		)

		// 将TaskResult的数据传递给OutputProcessor
		if err := e.outputProcessor.ProcessOutput(ctx, task, result.Data); err != nil {
			// 输出失败，标记任务失败
			result.Status = TaskStatusFailed
			result.ErrorMessage = fmt.Sprintf("Output processing failed: %v", err)
			updateErr := e.stateMachine.MarkTaskAsFailed(execCtx.Ctx, execCtx.TaskID, TaskTypeOutput, result)
			if updateErr != nil {
				e.logger.Errorw(
					"Failed to mark output task as failed after output error",
					logx.Field("task_id", execCtx.TaskID),
					logx.Field("error", updateErr.Error()),
				)
			}
			return fmt.Errorf("output processing failed: %w", err)
		}

		e.logger.Infow(
			"Output processing completed successfully",
			logx.Field("task_id", task.ID),
			logx.Field("output_target", task.OutputTarget),
		)
	}

	// 5. 标记任务完成
	if err := e.stateMachine.MarkTaskAsCompleted(execCtx.Ctx, execCtx.TaskID, TaskTypeOutput, result); err != nil {
		e.logger.Errorw(
			"Failed to mark output task as completed",
			logx.Field("task_id", execCtx.TaskID),
			logx.Field("error", err.Error()),
		)
		return err
	}

	return nil
}

// GetStateMachine 获取状态机（用于外部查询）
func (e *Executor) GetStateMachine() *StateMachine {
	return e.stateMachine
}

// applyTransform 应用Transform转换到发现的记录
func (e *Executor) applyTransform(
	execCtx *ExecutionContext,
	records []map[string]interface{},
) ([]map[string]interface{}, *transform.TransformStats, error) {
	// 1. 查询FieldMapping规则
	mappings, err := e.getFieldMappings(execCtx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get field mappings: %w", err)
	}

	// 如果没有映射规则，直接返回原始数据
	if len(mappings) == 0 {
		e.logger.Infow(
			"No field mappings found, skipping transform",
			logx.Field("task_id", execCtx.TaskID),
			logx.Field("task_type", execCtx.TaskType),
		)
		return records, &transform.TransformStats{}, nil
	}

	e.logger.Infow(
		"Applying transform",
		logx.Field("task_id", execCtx.TaskID),
		logx.Field("total_records", len(records)),
		logx.Field("total_mappings", len(mappings)),
	)

	// 2. 对每条记录应用Transform
	transformedRecords := make([]map[string]interface{}, 0, len(records))
	var totalStats transform.TransformStats

	for i, record := range records {
		// 创建Transform上下文
		transformCtx := &transform.TransformContext{
			Ctx:        execCtx.Ctx,
			TenantID:   execCtx.TenantID,
			UserID:     execCtx.UserID,
			TaskID:     execCtx.TaskID,
			TaskType:   string(execCtx.TaskType),
			Logger:     e.logger,
			DB:         e.db,
			SourceData: record,
			Mappings:   mappings,
		}

		// 执行Transform
		result := e.transformEngine.Transform(transformCtx)

		// 累计统计信息
		totalStats.TotalFields += result.Stats.TotalFields
		totalStats.MappedFields += result.Stats.MappedFields
		totalStats.SuccessFields += result.Stats.SuccessFields
		totalStats.FailedFields += result.Stats.FailedFields
		totalStats.SkippedFields += result.Stats.SkippedFields
		totalStats.ErrorMessages = append(totalStats.ErrorMessages, result.Stats.ErrorMessages...)

		// 检查是否成功
		if !result.Success {
			e.logger.Errorw(
				"Transform failed for record",
				logx.Field("task_id", execCtx.TaskID),
				logx.Field("record_index", i),
				logx.Field("error", result.ErrorMessage),
			)

			// 根据配置决定是否继续
			// TODO: 从配置中读取是否宽松模式
			// 暂时采用宽松模式：跳过失败的记录，继续处理
			continue
		}

		// 添加转换后的数据
		transformedRecords = append(transformedRecords, result.TargetData)
	}

	// 计算总时长
	totalStats.Duration = totalStats.Duration / time.Duration(len(records))

	e.logger.Infow(
		"Transform completed",
		logx.Field("task_id", execCtx.TaskID),
		logx.Field("input_records", len(records)),
		logx.Field("output_records", len(transformedRecords)),
		logx.Field("success_fields", totalStats.SuccessFields),
		logx.Field("failed_fields", totalStats.FailedFields),
	)

	return transformedRecords, &totalStats, nil
}

// getFieldMappings 获取任务关联的FieldMapping规则
func (e *Executor) getFieldMappings(execCtx *ExecutionContext) ([]*ent.FieldMapping, error) {
	// 根据任务类型查询不同的关联字段
	var mappings []*ent.FieldMapping
	var err error

	switch execCtx.TaskType {
	case TaskTypeInput:
		// InputTask: 查询 input_task_id = execCtx.TaskID
		mappings, err = e.db.FieldMapping.Query().
			Where(
				fieldmapping.InputTaskIDEQ(execCtx.TaskID),
				fieldmapping.IsActiveEQ(true),
				fieldmapping.StatusEQ(1), // 正常状态
			).
			Order(ent.Desc(fieldmapping.FieldPriority), ent.Asc(fieldmapping.FieldSortOrder)).
			All(execCtx.Ctx)

	case TaskTypeOutput:
		// OutputTask: 查询 output_task_id = execCtx.TaskID
		mappings, err = e.db.FieldMapping.Query().
			Where(
				fieldmapping.OutputTaskIDEQ(execCtx.TaskID),
				fieldmapping.IsActiveEQ(true),
				fieldmapping.StatusEQ(1), // 正常状态
			).
			Order(ent.Desc(fieldmapping.FieldPriority), ent.Asc(fieldmapping.FieldSortOrder)).
			All(execCtx.Ctx)

	default:
		return nil, fmt.Errorf("unsupported task type: %s", execCtx.TaskType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query field mappings: %w", err)
	}

	return mappings, nil
}
