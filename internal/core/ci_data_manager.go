package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/cmdb"
	"github.com/coder-lulu/newbee-common/v2/utils/uuidx"
	"github.com/gofrs/uuid/v5"
	"github.com/zeromicro/go-zero/core/logx"
)

// ciDataManagerImpl CI数据管理器实现
type ciDataManagerImpl struct {
	// 基础组件
	svcCtx *svc.ServiceContext
	config *Configuration
	logger logx.Logger

	// 核心组件
	validator         DataValidator
	permissionChecker PermissionChecker
	changeRecorder    ChangeRecorder
	lifecycleManager  LifecycleManager
	approvalManager   ApprovalManager
	dataPersister     DataPersister

	// 状态管理
	initialized bool
	mu          sync.RWMutex

	// 操作缓存
	operationCache sync.Map // map[string]*CiOperationResult

	// 性能监控
	metrics *PerformanceMetrics
}

// PerformanceMetrics 性能监控指标
type PerformanceMetrics struct {
	mu sync.RWMutex

	// 操作统计
	TotalOperations   int64
	SuccessOperations int64
	FailedOperations  int64

	// 执行时间统计
	AvgExecutionTime time.Duration
	MaxExecutionTime time.Duration
	MinExecutionTime time.Duration

	// 组件性能
	ValidationTime  time.Duration
	PermissionTime  time.Duration
	PersistenceTime time.Duration

	// 缓存统计
	CacheHits   int64
	CacheMisses int64

	// 最后更新时间
	LastUpdate time.Time
}

// NewCiDataManager 创建CI数据管理器
func NewCiDataManager(svcCtx *svc.ServiceContext, config *Configuration) CiDataManager {
	if config == nil {
		config = DefaultConfiguration()
	}

	manager := &ciDataManagerImpl{
		svcCtx: svcCtx,
		config: config,
		logger: logx.WithContext(context.Background()),
		metrics: &PerformanceMetrics{
			LastUpdate: time.Now(),
		},
	}

	return manager
}

// Name 返回组件名称
func (m *ciDataManagerImpl) Name() string {
	return "CiDataManager"
}

// Initialize 初始化组件
func (m *ciDataManagerImpl) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initialized {
		return nil
	}

	m.logger.Info("正在初始化CI数据管理器...")

	// 初始化子组件
	if err := m.initializeComponents(ctx); err != nil {
		return fmt.Errorf("初始化子组件失败: %w", err)
	}

	// 启动性能监控
	if m.config.Performance.EnableMetrics {
		go m.startMetricsCollection(ctx)
	}

	m.initialized = true
	m.logger.Info("CI数据管理器初始化完成")

	return nil
}

// Shutdown 关闭组件
func (m *ciDataManagerImpl) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return nil
	}

	m.logger.Info("正在关闭CI数据管理器...")

	// 关闭子组件
	if err := m.shutdownComponents(ctx); err != nil {
		m.logger.Errorf("关闭子组件失败: %v", err)
	}

	m.initialized = false
	m.logger.Info("CI数据管理器已关闭")

	return nil
}

// HealthCheck 健康检查
func (m *ciDataManagerImpl) HealthCheck(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return fmt.Errorf("CI数据管理器未初始化")
	}

	// 检查子组件健康状态
	components := []ComponentInterface{
		m.validator,
		m.permissionChecker,
		m.changeRecorder,
		m.lifecycleManager,
		m.approvalManager,
		m.dataPersister,
	}

	for _, component := range components {
		if component != nil {
			if err := component.HealthCheck(ctx); err != nil {
				return fmt.Errorf("组件 %s 健康检查失败: %w", component.Name(), err)
			}
		}
	}

	return nil
}

// CreateCi 创建CI实例
func (m *ciDataManagerImpl) CreateCi(ctx context.Context, operation *CiOperationContext) (*CiOperationResult, error) {
	if err := m.ensureInitialized(); err != nil {
		return nil, err
	}

	// 记录开始时间
	startTime := time.Now()

	// 生成操作ID
	if operation.OperationID == "" {
		operation.OperationID = uuidx.NewUUID().String()
	}

	// 创建操作结果
	result := &CiOperationResult{
		OperationID:    operation.OperationID,
		Status:         StatusPending,
		LifecycleStage: StageDraft,
	}

	// 缓存操作结果
	m.operationCache.Store(operation.OperationID, result)
	defer func() {
		result.ExecutionTime = time.Since(startTime)
		m.updateMetrics("create", result.Success, result.ExecutionTime)
	}()

	// 执行创建流程
	if err := m.executeCreateOperation(ctx, operation, result); err != nil {
		result.Success = false
		result.Status = StatusFailed
		result.Error = err
		result.Message = fmt.Sprintf("创建CI失败: %v", err)
		return result, err
	}

	result.Success = true
	result.Status = StatusCompleted
	result.LifecycleStage = StageCompleted
	result.Message = "CI创建成功"

	return result, nil
}

// UpdateCi 更新CI实例
func (m *ciDataManagerImpl) UpdateCi(ctx context.Context, operation *CiOperationContext) (*CiOperationResult, error) {
	if err := m.ensureInitialized(); err != nil {
		return nil, err
	}

	startTime := time.Now()

	if operation.OperationID == "" {
		operation.OperationID = uuidx.NewUUID().String()
	}

	result := &CiOperationResult{
		OperationID:    operation.OperationID,
		Status:         StatusPending,
		LifecycleStage: StageDraft,
	}

	m.operationCache.Store(operation.OperationID, result)
	defer func() {
		result.ExecutionTime = time.Since(startTime)
		m.updateMetrics("update", result.Success, result.ExecutionTime)
	}()

	if err := m.executeUpdateOperation(ctx, operation, result); err != nil {
		result.Success = false
		result.Status = StatusFailed
		result.Error = err
		result.Message = fmt.Sprintf("更新CI失败: %v", err)
		return result, err
	}

	result.Success = true
	result.Status = StatusCompleted
	result.LifecycleStage = StageCompleted
	result.Message = "CI更新成功"

	return result, nil
}

// DeleteCi 删除CI实例
func (m *ciDataManagerImpl) DeleteCi(ctx context.Context, operation *CiOperationContext) (*CiOperationResult, error) {
	if err := m.ensureInitialized(); err != nil {
		return nil, err
	}

	startTime := time.Now()

	if operation.OperationID == "" {
		operation.OperationID = uuidx.NewUUID().String()
	}

	result := &CiOperationResult{
		OperationID:    operation.OperationID,
		Status:         StatusPending,
		LifecycleStage: StageDraft,
	}

	m.operationCache.Store(operation.OperationID, result)
	defer func() {
		result.ExecutionTime = time.Since(startTime)
		m.updateMetrics("delete", result.Success, result.ExecutionTime)
	}()

	if err := m.executeDeleteOperation(ctx, operation, result); err != nil {
		result.Success = false
		result.Status = StatusFailed
		result.Error = err
		result.Message = fmt.Sprintf("删除CI失败: %v", err)
		return result, err
	}

	result.Success = true
	result.Status = StatusCompleted
	result.LifecycleStage = StageCompleted
	result.Message = "CI删除成功"

	return result, nil
}

// GetCi 获取CI实例
func (m *ciDataManagerImpl) GetCi(ctx context.Context, ciID uint64, operatorID uuid.UUID) (*cmdb.CisInfo, error) {
	if err := m.ensureInitialized(); err != nil {
		return nil, err
	}

	// 检查读取权限
	if m.permissionChecker != nil {
		operation := &CiOperationContext{
			Type:       "read",
			CiID:       &ciID,
			OperatorID: operatorID,
		}

		permResult, err := m.permissionChecker.CheckPermission(ctx, operation)
		if err != nil {
			return nil, fmt.Errorf("权限检查失败: %w", err)
		}

		if !permResult.Granted {
			return nil, fmt.Errorf("无权限读取CI %d", ciID)
		}
	}

	// 获取数据
	return m.dataPersister.Get(ctx, ciID)
}

// BatchCreate 批量创建CI实例
func (m *ciDataManagerImpl) BatchCreate(ctx context.Context, operation *CiOperationContext) (*CiOperationResult, error) {
	if err := m.ensureInitialized(); err != nil {
		return nil, err
	}

	startTime := time.Now()

	if operation.OperationID == "" {
		operation.OperationID = uuidx.NewUUID().String()
	}

	result := &CiOperationResult{
		OperationID:    operation.OperationID,
		Status:         StatusPending,
		LifecycleStage: StageDraft,
		BatchTotal:     len(operation.BatchData),
	}

	m.operationCache.Store(operation.OperationID, result)
	defer func() {
		result.ExecutionTime = time.Since(startTime)
		m.updateMetrics("batch_create", result.Success, result.ExecutionTime)
	}()

	if err := m.executeBatchCreateOperation(ctx, operation, result); err != nil {
		result.Success = false
		result.Status = StatusFailed
		result.Error = err
		result.Message = fmt.Sprintf("批量创建CI失败: %v", err)
		return result, err
	}

	result.Success = true
	result.Status = StatusCompleted
	result.LifecycleStage = StageCompleted
	result.Message = fmt.Sprintf("批量创建完成，成功: %d，失败: %d", result.BatchSuccess, result.BatchFailed)

	return result, nil
}

// BatchUpdate 批量更新CI实例
func (m *ciDataManagerImpl) BatchUpdate(ctx context.Context, operation *CiOperationContext) (*CiOperationResult, error) {
	// 实现逻辑类似 BatchCreate
	return m.executeBatchOperation(ctx, operation, "batch_update")
}

// BatchDelete 批量删除CI实例
func (m *ciDataManagerImpl) BatchDelete(ctx context.Context, operation *CiOperationContext) (*CiOperationResult, error) {
	// 实现逻辑类似 BatchCreate
	return m.executeBatchOperation(ctx, operation, "batch_delete")
}

// BulkImport 批量导入CI实例
func (m *ciDataManagerImpl) BulkImport(ctx context.Context, operation *CiOperationContext) (*CiOperationResult, error) {
	// 导入操作需要特殊处理，包含数据转换、清洗等步骤
	operation.Type = OperationImport
	return m.BatchCreate(ctx, operation)
}

// SyncData 同步数据
func (m *ciDataManagerImpl) SyncData(ctx context.Context, operation *CiOperationContext) (*CiOperationResult, error) {
	// 同步操作需要特殊处理，包含冲突解决等步骤
	operation.Type = OperationSync
	return m.BatchUpdate(ctx, operation)
}

// GetOperationStatus 获取操作状态
func (m *ciDataManagerImpl) GetOperationStatus(ctx context.Context, operationID string) (*CiOperationResult, error) {
	if result, ok := m.operationCache.Load(operationID); ok {
		return result.(*CiOperationResult), nil
	}

	return nil, fmt.Errorf("操作 %s 不存在", operationID)
}

// CancelOperation 取消操作
func (m *ciDataManagerImpl) CancelOperation(ctx context.Context, operationID string, operatorID uuid.UUID) error {
	if result, ok := m.operationCache.Load(operationID); ok {
		operationResult := result.(*CiOperationResult)

		// 只能取消pending或processing状态的操作
		if operationResult.Status == StatusPending || operationResult.Status == StatusProcessing {
			operationResult.Status = StatusFailed
			operationResult.Message = "操作已被取消"
			return nil
		}

		return fmt.Errorf("操作 %s 当前状态 %s 无法取消", operationID, operationResult.Status)
	}

	return fmt.Errorf("操作 %s 不存在", operationID)
}

// RetryOperation 重试操作
func (m *ciDataManagerImpl) RetryOperation(ctx context.Context, operationID string, operatorID uuid.UUID) (*CiOperationResult, error) {
	// TODO: 实现重试逻辑
	return nil, fmt.Errorf("重试功能暂未实现")
}

// SubmitForApproval 提交审批
func (m *ciDataManagerImpl) SubmitForApproval(ctx context.Context, operationID string) error {
	if m.approvalManager == nil {
		return fmt.Errorf("审批管理器未初始化")
	}

	// TODO: 实现提交审批逻辑
	return fmt.Errorf("提交审批功能暂未实现")
}

// ProcessApproval 处理审批
func (m *ciDataManagerImpl) ProcessApproval(ctx context.Context, operationID string, action string, comment string, approverID uuid.UUID) error {
	if m.approvalManager == nil {
		return fmt.Errorf("审批管理器未初始化")
	}

	// TODO: 实现处理审批逻辑
	return fmt.Errorf("处理审批功能暂未实现")
}

// GetChangeHistory 获取变更历史
func (m *ciDataManagerImpl) GetChangeHistory(ctx context.Context, ciID uint64, limit int, offset int) ([]*ChangeRecord, error) {
	if m.changeRecorder == nil {
		return nil, fmt.Errorf("变更记录器未初始化")
	}

	return m.changeRecorder.QueryChangeHistory(ctx, ciID, limit, offset)
}

// GetOperationLog 获取操作日志
func (m *ciDataManagerImpl) GetOperationLog(ctx context.Context, filters map[string]interface{}) ([]*CiOperationContext, error) {
	// TODO: 实现操作日志查询
	return nil, fmt.Errorf("操作日志查询功能暂未实现")
}

// 私有方法

// ensureInitialized 确保组件已初始化
func (m *ciDataManagerImpl) ensureInitialized() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return fmt.Errorf("CI数据管理器未初始化")
	}

	return nil
}

// initializeComponents 初始化子组件
func (m *ciDataManagerImpl) initializeComponents(ctx context.Context) error {
	// 创建各个组件实例
	m.validator = NewDataValidator(m.svcCtx, m.config)
	m.permissionChecker = NewPermissionChecker(m.svcCtx, m.config)
	m.changeRecorder = NewChangeRecorder(m.svcCtx, m.config)
	m.lifecycleManager = NewLifecycleManager(m.svcCtx, m.config)
	m.approvalManager = NewApprovalManager(m.svcCtx, m.config)
	m.dataPersister = NewDataPersister(m.svcCtx, m.config)

	// 初始化各个组件
	components := []ComponentInterface{
		m.validator,
		m.permissionChecker,
		m.changeRecorder,
		m.lifecycleManager,
		m.approvalManager,
		m.dataPersister,
	}

	for _, component := range components {
		if component != nil {
			if err := component.Initialize(ctx); err != nil {
				return fmt.Errorf("初始化组件 %s 失败: %w", component.Name(), err)
			}
		}
	}

	return nil
}

// shutdownComponents 关闭子组件
func (m *ciDataManagerImpl) shutdownComponents(ctx context.Context) error {
	components := []ComponentInterface{
		m.dataPersister,
		m.approvalManager,
		m.lifecycleManager,
		m.changeRecorder,
		m.permissionChecker,
		m.validator,
	}

	for _, component := range components {
		if component != nil {
			if err := component.Shutdown(ctx); err != nil {
				m.logger.Errorf("关闭组件 %s 失败: %v", component.Name(), err)
			}
		}
	}

	return nil
}

// executeCreateOperation 执行创建操作
func (m *ciDataManagerImpl) executeCreateOperation(ctx context.Context, operation *CiOperationContext, result *CiOperationResult) error {
	// 1. 创建生命周期状态
	result.Status = StatusProcessing
	result.LifecycleStage = StageSubmitted

	if m.lifecycleManager != nil {
		state, err := m.lifecycleManager.CreateState(ctx, operation)
		if err != nil {
			return fmt.Errorf("创建生命周期状态失败: %w", err)
		}
		result.LifecycleStateID = state.StateID
	}

	// 2. 权限检查
	if !operation.SkipPermissionCheck && m.permissionChecker != nil {
		result.LifecycleStage = StageSubmitted

		permResult, err := m.permissionChecker.CheckPermission(ctx, operation)
		if err != nil {
			return fmt.Errorf("权限检查失败: %w", err)
		}

		result.PermissionResult = permResult
		if !permResult.Granted {
			return fmt.Errorf("权限不足，无法创建CI")
		}
	}

	// 3. 数据校验
	if !operation.SkipValidation && m.validator != nil {
		result.Status = StatusValidating
		result.LifecycleStage = StageValidated

		validResult, err := m.validator.ValidateComplete(ctx, operation)
		if err != nil {
			return fmt.Errorf("数据校验失败: %w", err)
		}

		result.ValidationResult = validResult
		if !validResult.Valid {
			return fmt.Errorf("数据校验未通过，错误: %v", validResult.Errors)
		}
	}

	// 4. 检查是否需要审批
	if operation.RequireApproval && m.approvalManager != nil {
		requireApproval, flowID, err := m.approvalManager.CheckRequireApproval(ctx, operation)
		if err != nil {
			return fmt.Errorf("检查审批需求失败: %w", err)
		}

		if requireApproval {
			result.Status = StatusWaitingApproval
			result.RequireApproval = true
			result.ApprovalFlowID = flowID

			// 提交审批（这里暂时跳过，实际应该等待审批完成）
			m.logger.Infof("操作 %s 需要审批，流程ID: %s", operation.OperationID, flowID)
			return nil // 暂停执行，等待审批
		}
	}

	// 5. 数据持久化
	result.Status = StatusProcessing
	result.LifecycleStage = StageExecuted

	if m.dataPersister != nil {
		createdData, err := m.dataPersister.Create(ctx, operation.DataAfter)
		if err != nil {
			return fmt.Errorf("数据持久化失败: %w", err)
		}

		if createdData.Id != nil {
			result.CreatedCiID = createdData.Id
			result.AffectedCiIDs = []uint64{*createdData.Id}
			operation.CiID = createdData.Id
		}

		result.Data = createdData
	}

	// 6. 记录变更
	if m.changeRecorder != nil {
		changeRecord, err := m.changeRecorder.RecordCreate(ctx, operation, result)
		if err != nil {
			m.logger.Errorf("记录变更失败: %v", err)
		} else {
			result.ChangeRecordID = changeRecord.RecordID
		}
	}

	return nil
}

// executeUpdateOperation 执行更新操作
func (m *ciDataManagerImpl) executeUpdateOperation(ctx context.Context, operation *CiOperationContext, result *CiOperationResult) error {
	// 类似于创建操作，但需要先获取现有数据
	if operation.CiID == nil {
		return fmt.Errorf("更新操作必须指定CI ID")
	}

	// 获取现有数据
	if m.dataPersister != nil && operation.DataBefore == nil {
		existingData, err := m.dataPersister.Get(ctx, *operation.CiID)
		if err != nil {
			return fmt.Errorf("获取现有数据失败: %w", err)
		}
		operation.DataBefore = existingData
	}

	// 执行类似创建的流程
	result.Status = StatusProcessing

	// 权限检查、数据校验、审批检查...（类似创建流程）

	// 数据持久化
	if m.dataPersister != nil {
		updatedData, err := m.dataPersister.Update(ctx, *operation.CiID, operation.DataAfter)
		if err != nil {
			return fmt.Errorf("数据更新失败: %w", err)
		}

		result.AffectedCiIDs = []uint64{*operation.CiID}
		result.Data = updatedData
	}

	// 记录变更
	if m.changeRecorder != nil {
		changeRecord, err := m.changeRecorder.RecordUpdate(ctx, operation, result)
		if err != nil {
			m.logger.Errorf("记录变更失败: %v", err)
		} else {
			result.ChangeRecordID = changeRecord.RecordID
		}
	}

	return nil
}

// executeDeleteOperation 执行删除操作
func (m *ciDataManagerImpl) executeDeleteOperation(ctx context.Context, operation *CiOperationContext, result *CiOperationResult) error {
	if operation.CiID == nil {
		return fmt.Errorf("删除操作必须指定CI ID")
	}

	// 获取要删除的数据
	if m.dataPersister != nil && operation.DataBefore == nil {
		existingData, err := m.dataPersister.Get(ctx, *operation.CiID)
		if err != nil {
			return fmt.Errorf("获取要删除的数据失败: %w", err)
		}
		operation.DataBefore = existingData
	}

	result.Status = StatusProcessing

	// 权限检查、审批检查...

	// 数据删除
	if m.dataPersister != nil {
		if err := m.dataPersister.Delete(ctx, *operation.CiID); err != nil {
			return fmt.Errorf("数据删除失败: %w", err)
		}

		result.AffectedCiIDs = []uint64{*operation.CiID}
	}

	// 记录变更
	if m.changeRecorder != nil {
		changeRecord, err := m.changeRecorder.RecordDelete(ctx, operation, result)
		if err != nil {
			m.logger.Errorf("记录变更失败: %v", err)
		} else {
			result.ChangeRecordID = changeRecord.RecordID
		}
	}

	return nil
}

// executeBatchCreateOperation 执行批量创建操作
func (m *ciDataManagerImpl) executeBatchCreateOperation(ctx context.Context, operation *CiOperationContext, result *CiOperationResult) error {
	if len(operation.BatchData) == 0 {
		return fmt.Errorf("批量操作数据不能为空")
	}

	result.Status = StatusProcessing
	result.BatchTotal = len(operation.BatchData)

	// 批量权限检查
	if !operation.SkipPermissionCheck && m.permissionChecker != nil {
		permResult, err := m.permissionChecker.CheckBatchPermission(ctx, operation)
		if err != nil {
			return fmt.Errorf("批量权限检查失败: %w", err)
		}

		if !permResult.Granted {
			return fmt.Errorf("权限不足，无法执行批量创建")
		}
	}

	// 批量数据校验
	if !operation.SkipValidation && m.validator != nil {
		for _, data := range operation.BatchData {
			operation.DataAfter = data
			validResult, err := m.validator.ValidateComplete(ctx, operation)
			if err != nil {
				result.BatchFailed++
				continue
			}

			if !validResult.Valid {
				result.BatchFailed++
				continue
			}
		}
	}

	// 批量数据持久化
	if m.dataPersister != nil {
		createdDataList, err := m.dataPersister.BatchCreate(ctx, operation.BatchData)
		if err != nil {
			return fmt.Errorf("批量数据持久化失败: %w", err)
		}

		result.BatchSuccess = len(createdDataList)
		result.BatchFailed = result.BatchTotal - result.BatchSuccess

		// 收集创建的CI ID
		for _, data := range createdDataList {
			if data.Id != nil {
				result.AffectedCiIDs = append(result.AffectedCiIDs, *data.Id)
			}
		}
	}

	// 记录批量变更
	if m.changeRecorder != nil {
		changeRecords, err := m.changeRecorder.RecordBatch(ctx, operation, result)
		if err != nil {
			m.logger.Errorf("记录批量变更失败: %v", err)
		} else if len(changeRecords) > 0 {
			result.ChangeRecordID = changeRecords[0].RecordID // 使用第一个记录的ID
		}
	}

	return nil
}

// executeBatchOperation 执行批量操作的通用方法
func (m *ciDataManagerImpl) executeBatchOperation(ctx context.Context, operation *CiOperationContext, operationType string) (*CiOperationResult, error) {
	startTime := time.Now()

	if operation.OperationID == "" {
		operation.OperationID = uuidx.NewUUID().String()
	}

	result := &CiOperationResult{
		OperationID:    operation.OperationID,
		Status:         StatusPending,
		LifecycleStage: StageDraft,
	}

	m.operationCache.Store(operation.OperationID, result)
	defer func() {
		result.ExecutionTime = time.Since(startTime)
		m.updateMetrics(operationType, result.Success, result.ExecutionTime)
	}()

	// 根据操作类型执行相应逻辑
	switch operationType {
	case "batch_update":
		// TODO: 实现批量更新逻辑
		return nil, fmt.Errorf("批量更新功能暂未实现")
	case "batch_delete":
		// TODO: 实现批量删除逻辑
		return nil, fmt.Errorf("批量删除功能暂未实现")
	default:
		return nil, fmt.Errorf("不支持的批量操作类型: %s", operationType)
	}
}

// updateMetrics 更新性能指标
func (m *ciDataManagerImpl) updateMetrics(operation string, success bool, duration time.Duration) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	m.metrics.TotalOperations++
	if success {
		m.metrics.SuccessOperations++
	} else {
		m.metrics.FailedOperations++
	}

	// 更新执行时间统计
	if m.metrics.MinExecutionTime == 0 || duration < m.metrics.MinExecutionTime {
		m.metrics.MinExecutionTime = duration
	}

	if duration > m.metrics.MaxExecutionTime {
		m.metrics.MaxExecutionTime = duration
	}

	// 计算平均执行时间
	if m.metrics.TotalOperations > 0 {
		totalDuration := time.Duration(int64(m.metrics.AvgExecutionTime)*(m.metrics.TotalOperations-1) + int64(duration))
		m.metrics.AvgExecutionTime = totalDuration / time.Duration(m.metrics.TotalOperations)
	}

	m.metrics.LastUpdate = time.Now()
}

// startMetricsCollection 启动性能监控收集
func (m *ciDataManagerImpl) startMetricsCollection(ctx context.Context) {
	ticker := time.NewTicker(m.config.Performance.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.collectMetrics()
		}
	}
}

// collectMetrics 收集性能指标
func (m *ciDataManagerImpl) collectMetrics() {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	// 输出性能指标到日志
	m.logger.Infof("性能指标 - 总操作数: %d, 成功: %d, 失败: %d, 平均耗时: %v",
		m.metrics.TotalOperations,
		m.metrics.SuccessOperations,
		m.metrics.FailedOperations,
		m.metrics.AvgExecutionTime,
	)

	// TODO: 可以将指标发送到监控系统
}
