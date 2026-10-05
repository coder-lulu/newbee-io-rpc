package core

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/ent/cichangehistory"
	"github.com/coder-lulu/newbee-io-rpc/ent/cilifecyclestate"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-cmdb-rpc/types/cmdb"
	"github.com/coder-lulu/newbee-common/v2/utils/uuidx"
	"github.com/gofrs/uuid/v5"
	"github.com/zeromicro/go-zero/core/logx"
)

// permissionCheckerImpl 权限检查器实现
type permissionCheckerImpl struct {
	svcCtx      *svc.ServiceContext
	config      *Configuration
	logger      logx.Logger
	initialized bool
}

func (p *permissionCheckerImpl) Name() string {
	return "PermissionChecker"
}

func (p *permissionCheckerImpl) Initialize(ctx context.Context) error {
	p.logger = logx.WithContext(ctx)
	p.initialized = true
	p.logger.Info("权限检查器初始化完成")
	return nil
}

func (p *permissionCheckerImpl) Shutdown(ctx context.Context) error {
	p.initialized = false
	p.logger.Info("权限检查器已关闭")
	return nil
}

func (p *permissionCheckerImpl) HealthCheck(ctx context.Context) error {
	if !p.initialized {
		return fmt.Errorf("权限检查器未初始化")
	}
	return nil
}

func (p *permissionCheckerImpl) CheckPermission(ctx context.Context, operation *CiOperationContext) (*PermissionResult, error) {
	startTime := time.Now()

	// 🎯 实现基于操作类型的权限检查逻辑
	result := &PermissionResult{
		Granted:    false,
		Level:      PermissionNone,
		Operations: []string{},
		Performance: &PermissionPerformance{
			TotalTime: time.Since(startTime),
			RuleCount: 0,
		},
	}

	// 1. 快速缓存检查（Redis）
	cacheKey := fmt.Sprintf("io:permission:%d:%s:%s", operation.UserID, operation.CiType, operation.Operation)
	if cachedResult := p.getFromCache(ctx, cacheKey); cachedResult != nil {
		cachedResult.Performance = &PermissionPerformance{
			TotalTime: time.Since(startTime),
			RuleCount: 1,
		}
		return cachedResult, nil
	}

	// 2. 基于操作类型的权限判断
	granted := false
	level := PermissionNone
	operations := []string{}

	switch operation.Operation {
	case "read", "list", "view", "get", "query":
		// 读取操作：最低权限级别，默认允许
		granted = true
		level = PermissionRead
		operations = []string{"read", "list", "view", "get", "query"}

	case "create", "import", "add":
		// 创建操作：需要写入权限
		granted = p.hasWritePermission(ctx, operation)
		if granted {
			level = PermissionWrite
			operations = []string{"read", "list", "view", "create", "import", "add"}
		}

	case "update", "edit", "modify", "transform":
		// 更新操作：需要写入权限
		granted = p.hasWritePermission(ctx, operation)
		if granted {
			level = PermissionWrite
			operations = []string{"read", "list", "view", "update", "edit", "modify", "transform"}
		}

	case "delete", "remove":
		// 删除操作：需要管理员权限
		granted = p.hasAdminPermission(ctx, operation)
		if granted {
			level = PermissionAdmin
			operations = []string{"read", "list", "view", "create", "update", "delete", "remove"}
		}

	case "approve", "reject", "review":
		// 审批操作：需要审批权限
		granted = p.hasApprovalPermission(ctx, operation)
		if granted {
			level = PermissionApprove
			operations = []string{"read", "list", "view", "approve", "reject", "review"}
		}

	case "export", "download":
		// 导出操作：需要读取权限，默认允许
		granted = true
		level = PermissionRead
		operations = []string{"read", "list", "view", "export", "download"}

	case "execute", "run":
		// 执行操作：需要执行权限
		granted = p.hasExecutePermission(ctx, operation)
		if granted {
			level = PermissionExecute
			operations = []string{"read", "execute", "run"}
		}

	default:
		// 未知操作类型：拒绝访问
		granted = false
		level = PermissionNone
		p.logger.Warnw("Unknown operation type",
			logx.Field("operation", operation.Operation),
			logx.Field("ci_type", operation.CiType),
			logx.Field("user_id", operation.UserID))
	}

	result.Granted = granted
	result.Level = level
	result.Operations = operations
	result.Performance = &PermissionPerformance{
		TotalTime: time.Since(startTime),
		RuleCount: 1,
	}

	// 3. 缓存结果（缓存5分钟）
	if granted {
		p.setCache(ctx, cacheKey, result, 5*time.Minute)
	}

	p.logger.Infow("Permission check completed",
		logx.Field("user_id", operation.UserID),
		logx.Field("ci_type", operation.CiType),
		logx.Field("operation", operation.Operation),
		logx.Field("granted", granted),
		logx.Field("level", level),
		logx.Field("duration_ms", time.Since(startTime).Milliseconds()))

	return result, nil
}

// hasWritePermission 检查是否有写入权限
func (p *permissionCheckerImpl) hasWritePermission(ctx context.Context, operation *CiOperationContext) bool {
	// 简化实现：暂时允许所有写入操作
	// TODO: 集成Casbin或调用core服务的权限接口进行真实检查
	return true
}

// hasAdminPermission 检查是否有管理员权限
func (p *permissionCheckerImpl) hasAdminPermission(ctx context.Context, operation *CiOperationContext) bool {
	// 简化实现：暂时允许所有管理员操作
	// TODO: 集成Casbin或调用core服务的权限接口进行真实检查
	return true
}

// hasApprovalPermission 检查是否有审批权限
func (p *permissionCheckerImpl) hasApprovalPermission(ctx context.Context, operation *CiOperationContext) bool {
	// 简化实现：暂时允许所有审批操作
	// TODO: 集成Casbin或调用core服务的权限接口进行真实检查
	return true
}

// hasExecutePermission 检查是否有执行权限
func (p *permissionCheckerImpl) hasExecutePermission(ctx context.Context, operation *CiOperationContext) bool {
	// 简化实现：暂时允许所有执行操作
	// TODO: 集成Casbin或调用core服务的权限接口进行真实检查
	return true
}

// getFromCache 从Redis获取缓存的权限结果
func (p *permissionCheckerImpl) getFromCache(ctx context.Context, key string) *PermissionResult {
	// 简化实现：暂不使用缓存
	// TODO: 实现Redis缓存
	return nil
}

// setCache 设置权限结果缓存
func (p *permissionCheckerImpl) setCache(ctx context.Context, key string, result *PermissionResult, ttl time.Duration) {
	// 简化实现：暂不使用缓存
	// TODO: 实现Redis缓存
}

func (p *permissionCheckerImpl) CheckFieldPermission(ctx context.Context, operatorID uuid.UUID, ciTypeID uint64, attrID uint64, operation string) (*PermissionResult, error) {
	// TODO: 实现字段级权限检查
	return &PermissionResult{
		Granted: true,
		Level:   PermissionWrite,
	}, nil
}

func (p *permissionCheckerImpl) CheckBatchPermission(ctx context.Context, operation *CiOperationContext) (*PermissionResult, error) {
	// TODO: 实现批量权限检查
	return &PermissionResult{
		Granted: true,
		Level:   PermissionWrite,
	}, nil
}

func (p *permissionCheckerImpl) GetUserPermissions(ctx context.Context, userID uuid.UUID, ciTypeID uint64) (*PermissionResult, error) {
	// TODO: 实现获取用户权限
	return &PermissionResult{
		Granted: true,
		Level:   PermissionWrite,
	}, nil
}

// changeRecorderImpl 变更记录器实现
type changeRecorderImpl struct {
	svcCtx      *svc.ServiceContext
	config      *Configuration
	logger      logx.Logger
	initialized bool
}

func (c *changeRecorderImpl) Name() string {
	return "ChangeRecorder"
}

func (c *changeRecorderImpl) Initialize(ctx context.Context) error {
	c.logger = logx.WithContext(ctx)
	c.initialized = true
	c.logger.Info("变更记录器初始化完成")
	return nil
}

func (c *changeRecorderImpl) Shutdown(ctx context.Context) error {
	c.initialized = false
	c.logger.Info("变更记录器已关闭")
	return nil
}

func (c *changeRecorderImpl) HealthCheck(ctx context.Context) error {
	if !c.initialized {
		return fmt.Errorf("变更记录器未初始化")
	}
	return nil
}

func (c *changeRecorderImpl) RecordCreate(ctx context.Context, operation *CiOperationContext, result *CiOperationResult) (*ChangeRecord, error) {
	record := &ChangeRecord{
		RecordID:      uuidx.NewUUID().String(),
		OperationID:   operation.OperationID,
		CiTypeID:      operation.CiTypeID,
		OperationType: operation.Type,
		OperationTime: time.Now(),
		OperatorID:    operation.OperatorID,
		OperatorName:  operation.OperatorName,
		Source:        operation.Source,
		SourceDetail:  operation.SourceDetail,
		ChangeReason:  operation.Reason,
	}

	if operation.CiID != nil {
		record.CiID = *operation.CiID
	}

	// 保存到数据库
	startTime := time.Now()

	// 准备新数据JSON
	var newValuesJSON string
	if operation.DataAfter != nil {
		if jsonBytes, err := json.Marshal(operation.DataAfter); err == nil {
			newValuesJSON = string(jsonBytes)
		}
	}

	// 创建数据库记录
	builder := c.svcCtx.DB.CiChangeHistory.Create().
		SetOperationID(operation.OperationID).
		SetCiTypeID(operation.CiTypeID).
		SetOperationType(string(operation.Type)).
		SetOperationName(fmt.Sprintf("创建CI: %s", operation.OperatorName)).
		SetOperatorID(uint64(operation.OperatorID.IntPart())).
		SetOperatorName(operation.OperatorName).
		SetSource(string(operation.Source)).
		SetSourceDetail(operation.SourceDetail).
		SetChangeReason(operation.Reason).
		SetIPAddress(operation.ClientIP).
		SetUserAgent(operation.UserAgent).
		SetStatus("success").
		SetAffectedCount(1).
		SetDurationMs(int(time.Since(startTime).Milliseconds()))

	// 设置CI ID（如果有）
	if operation.CiID != nil {
		builder.SetCiID(*operation.CiID)
	} else if result != nil && result.CreatedCiID != nil {
		builder.SetCiID(*result.CreatedCiID)
		record.CiID = *result.CreatedCiID
	}

	// 设置新数据
	if newValuesJSON != "" {
		builder.SetNewValues(newValuesJSON)
	}

	// 设置审批信息
	if operation.RequireApproval {
		builder.SetNeedsApproval(true)
	}

	// 设置元数据
	if len(operation.Metadata) > 0 {
		if metadataJSON, err := json.Marshal(operation.Metadata); err == nil {
			builder.SetMetadata(string(metadataJSON))
		}
	}

	dbRecord, err := builder.Save(ctx)
	if err != nil {
		c.logger.Errorw("Failed to save change record",
			logx.Field("operation_id", operation.OperationID),
			logx.Field("error", err))
		return nil, err
	}

	c.logger.Infow("Change record created",
		logx.Field("record_id", dbRecord.ID),
		logx.Field("operation_id", operation.OperationID),
		logx.Field("ci_id", dbRecord.CiID))

	return record, nil
}

func (c *changeRecorderImpl) RecordUpdate(ctx context.Context, operation *CiOperationContext, result *CiOperationResult) (*ChangeRecord, error) {
	record := &ChangeRecord{
		RecordID:      uuidx.NewUUID().String(),
		OperationID:   operation.OperationID,
		CiID:          *operation.CiID,
		CiTypeID:      operation.CiTypeID,
		OperationType: operation.Type,
		OperationTime: time.Now(),
		OperatorID:    operation.OperatorID,
		OperatorName:  operation.OperatorName,
		Source:        operation.Source,
		SourceDetail:  operation.SourceDetail,
		ChangeReason:  operation.Reason,
	}

	// 保存到数据库
	startTime := time.Now()

	// 准备旧数据和新数据JSON
	var oldValuesJSON, newValuesJSON string
	if operation.DataBefore != nil {
		if jsonBytes, err := json.Marshal(operation.DataBefore); err == nil {
			oldValuesJSON = string(jsonBytes)
		}
	}
	if operation.DataAfter != nil {
		if jsonBytes, err := json.Marshal(operation.DataAfter); err == nil {
			newValuesJSON = string(jsonBytes)
		}
	}

	// 计算变更字段
	changedFields := c.calculateChangedFields(operation.DataBefore, operation.DataAfter)
	var changedFieldsJSON string
	if len(changedFields) > 0 {
		if jsonBytes, err := json.Marshal(changedFields); err == nil {
			changedFieldsJSON = string(jsonBytes)
		}
	}

	// 创建数据库记录
	builder := c.svcCtx.DB.CiChangeHistory.Create().
		SetOperationID(operation.OperationID).
		SetCiID(*operation.CiID).
		SetCiTypeID(operation.CiTypeID).
		SetOperationType(string(operation.Type)).
		SetOperationName(fmt.Sprintf("更新CI: %s", operation.OperatorName)).
		SetOperatorID(uint64(operation.OperatorID.IntPart())).
		SetOperatorName(operation.OperatorName).
		SetSource(string(operation.Source)).
		SetSourceDetail(operation.SourceDetail).
		SetChangeReason(operation.Reason).
		SetIPAddress(operation.ClientIP).
		SetUserAgent(operation.UserAgent).
		SetStatus("success").
		SetAffectedCount(1).
		SetDurationMs(int(time.Since(startTime).Milliseconds()))

	// 设置变更数据
	if oldValuesJSON != "" {
		builder.SetOldValues(oldValuesJSON)
	}
	if newValuesJSON != "" {
		builder.SetNewValues(newValuesJSON)
	}
	if changedFieldsJSON != "" {
		builder.SetChangedFields(changedFieldsJSON)
	}

	// 设置审批信息
	if operation.RequireApproval {
		builder.SetNeedsApproval(true)
	}

	// 设置元数据
	if len(operation.Metadata) > 0 {
		if metadataJSON, err := json.Marshal(operation.Metadata); err == nil {
			builder.SetMetadata(string(metadataJSON))
		}
	}

	dbRecord, err := builder.Save(ctx)
	if err != nil {
		c.logger.Errorw("Failed to save change record for update",
			logx.Field("operation_id", operation.OperationID),
			logx.Field("ci_id", *operation.CiID),
			logx.Field("error", err))
		return nil, err
	}

	c.logger.Infow("Change record created for update",
		logx.Field("record_id", dbRecord.ID),
		logx.Field("operation_id", operation.OperationID),
		logx.Field("ci_id", *operation.CiID),
		logx.Field("changed_fields_count", len(changedFields)))

	return record, nil
}

// calculateChangedFields 计算变更字段
func (c *changeRecorderImpl) calculateChangedFields(before, after *cmdb.CisInfo) []map[string]interface{} {
	var changes []map[string]interface{}

	if before == nil || after == nil {
		return changes
	}

	// 比较基础字段
	if before.Name != nil && after.Name != nil && *before.Name != *after.Name {
		changes = append(changes, map[string]interface{}{
			"field":     "name",
			"old_value": *before.Name,
			"new_value": *after.Name,
		})
	}

	if before.Description != nil && after.Description != nil && *before.Description != *after.Description {
		changes = append(changes, map[string]interface{}{
			"field":     "description",
			"old_value": *before.Description,
			"new_value": *after.Description,
		})
	}

	if before.Status != nil && after.Status != nil && *before.Status != *after.Status {
		changes = append(changes, map[string]interface{}{
			"field":     "status",
			"old_value": *before.Status,
			"new_value": *after.Status,
		})
	}

	// 比较属性值（attributes是map）
	beforeAttrs := make(map[string]interface{})
	afterAttrs := make(map[string]interface{})

	if before.Attributes != nil {
		beforeAttrs = before.Attributes
	}
	if after.Attributes != nil {
		afterAttrs = after.Attributes
	}

	// 检查变更和新增的属性
	for key, afterVal := range afterAttrs {
		if beforeVal, exists := beforeAttrs[key]; exists {
			// 属性存在，检查是否变更
			if fmt.Sprintf("%v", beforeVal) != fmt.Sprintf("%v", afterVal) {
				changes = append(changes, map[string]interface{}{
					"field":     key,
					"old_value": beforeVal,
					"new_value": afterVal,
				})
			}
		} else {
			// 新增属性
			changes = append(changes, map[string]interface{}{
				"field":     key,
				"old_value": nil,
				"new_value": afterVal,
			})
		}
	}

	// 检查删除的属性
	for key, beforeVal := range beforeAttrs {
		if _, exists := afterAttrs[key]; !exists {
			changes = append(changes, map[string]interface{}{
				"field":     key,
				"old_value": beforeVal,
				"new_value": nil,
			})
		}
	}

	return changes
}

func (c *changeRecorderImpl) RecordDelete(ctx context.Context, operation *CiOperationContext, result *CiOperationResult) (*ChangeRecord, error) {
	record := &ChangeRecord{
		RecordID:      uuidx.NewUUID().String(),
		OperationID:   operation.OperationID,
		CiID:          *operation.CiID,
		CiTypeID:      operation.CiTypeID,
		OperationType: operation.Type,
		OperationTime: time.Now(),
		OperatorID:    operation.OperatorID,
		OperatorName:  operation.OperatorName,
		Source:        operation.Source,
		SourceDetail:  operation.SourceDetail,
		ChangeReason:  operation.Reason,
	}

	// 保存到数据库
	startTime := time.Now()

	// 准备删除前的数据JSON
	var oldValuesJSON string
	if operation.DataBefore != nil {
		if jsonBytes, err := json.Marshal(operation.DataBefore); err == nil {
			oldValuesJSON = string(jsonBytes)
		}
	}

	// 创建数据库记录
	builder := c.svcCtx.DB.CiChangeHistory.Create().
		SetOperationID(operation.OperationID).
		SetCiID(*operation.CiID).
		SetCiTypeID(operation.CiTypeID).
		SetOperationType(string(operation.Type)).
		SetOperationName(fmt.Sprintf("删除CI: %s", operation.OperatorName)).
		SetOperatorID(uint64(operation.OperatorID.IntPart())).
		SetOperatorName(operation.OperatorName).
		SetSource(string(operation.Source)).
		SetSourceDetail(operation.SourceDetail).
		SetChangeReason(operation.Reason).
		SetIPAddress(operation.ClientIP).
		SetUserAgent(operation.UserAgent).
		SetStatus("success").
		SetAffectedCount(1).
		SetCanRollback(true). // 删除操作可以回滚（重新创建）
		SetDurationMs(int(time.Since(startTime).Milliseconds()))

	// 设置删除前的数据
	if oldValuesJSON != "" {
		builder.SetOldValues(oldValuesJSON)
	}

	// 设置审批信息
	if operation.RequireApproval {
		builder.SetNeedsApproval(true)
	}

	// 设置元数据
	if len(operation.Metadata) > 0 {
		if metadataJSON, err := json.Marshal(operation.Metadata); err == nil {
			builder.SetMetadata(string(metadataJSON))
		}
	}

	dbRecord, err := builder.Save(ctx)
	if err != nil {
		c.logger.Errorw("Failed to save change record for delete",
			logx.Field("operation_id", operation.OperationID),
			logx.Field("ci_id", *operation.CiID),
			logx.Field("error", err))
		return nil, err
	}

	c.logger.Infow("Change record created for delete",
		logx.Field("record_id", dbRecord.ID),
		logx.Field("operation_id", operation.OperationID),
		logx.Field("ci_id", *operation.CiID))

	return record, nil
}

func (c *changeRecorderImpl) RecordBatch(ctx context.Context, operation *CiOperationContext, result *CiOperationResult) ([]*ChangeRecord, error) {
	var records []*ChangeRecord
	startTime := time.Now()

	// 为每个受影响的CI创建变更记录
	for _, ciID := range result.AffectedCiIDs {
		record := &ChangeRecord{
			RecordID:      uuidx.NewUUID().String(),
			OperationID:   operation.OperationID,
			CiID:          ciID,
			CiTypeID:      operation.CiTypeID,
			OperationType: operation.Type,
			OperationTime: time.Now(),
			OperatorID:    operation.OperatorID,
			OperatorName:  operation.OperatorName,
			Source:        operation.Source,
			SourceDetail:  operation.SourceDetail,
			ChangeReason:  operation.Reason,
		}

		// 保存到数据库
		builder := c.svcCtx.DB.CiChangeHistory.Create().
			SetOperationID(operation.OperationID).
			SetCiID(ciID).
			SetCiTypeID(operation.CiTypeID).
			SetOperationType(string(operation.Type)).
			SetOperationName(fmt.Sprintf("批量操作: %s", string(operation.Type))).
			SetOperatorID(uint64(operation.OperatorID.IntPart())).
			SetOperatorName(operation.OperatorName).
			SetSource(string(operation.Source)).
			SetSourceDetail(operation.SourceDetail).
			SetChangeReason(operation.Reason).
			SetIPAddress(operation.ClientIP).
			SetUserAgent(operation.UserAgent).
			SetStatus("success").
			SetAffectedCount(len(result.AffectedCiIDs)).
			SetDurationMs(int(time.Since(startTime).Milliseconds()))

		// 设置审批信息
		if operation.RequireApproval {
			builder.SetNeedsApproval(true)
		}

		// 设置元数据
		if len(operation.Metadata) > 0 {
			if metadataJSON, err := json.Marshal(operation.Metadata); err == nil {
				builder.SetMetadata(string(metadataJSON))
			}
		}

		_, err := builder.Save(ctx)
		if err != nil {
			c.logger.Errorw("Failed to save batch change record",
				logx.Field("operation_id", operation.OperationID),
				logx.Field("ci_id", ciID),
				logx.Field("error", err))
			continue // 继续处理其他记录
		}

		records = append(records, record)
	}

	c.logger.Infow("Batch change records created",
		logx.Field("operation_id", operation.OperationID),
		logx.Field("total_records", len(records)),
		logx.Field("affected_ci_count", len(result.AffectedCiIDs)))

	return records, nil
}

func (c *changeRecorderImpl) QueryChangeHistory(ctx context.Context, ciID uint64, limit int, offset int) ([]*ChangeRecord, error) {
	// 设置默认值
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// 查询数据库
	dbRecords, err := c.svcCtx.DB.CiChangeHistory.Query().
		Where(
			cichangehistory.CiIDEQ(ciID),
		).
		Order(ent.Desc(cichangehistory.FieldCreatedAt)).
		Limit(limit).
		Offset(offset).
		All(ctx)

	if err != nil {
		c.logger.Errorw("Failed to query change history",
			logx.Field("ci_id", ciID),
			logx.Field("error", err))
		return nil, err
	}

	// 转换为ChangeRecord结构
	records := make([]*ChangeRecord, 0, len(dbRecords))
	for _, dbRecord := range dbRecords {
		// 解析UUID
		operatorUUID, _ := uuid.FromString(fmt.Sprintf("%d", dbRecord.OperatorID))

		// 解析changed_fields JSON
		var changedFields []string
		if dbRecord.ChangedFields != nil {
			var fields []map[string]interface{}
			if err := json.Unmarshal([]byte(*dbRecord.ChangedFields), &fields); err == nil {
				for _, field := range fields {
					if fieldName, ok := field["field"].(string); ok {
						changedFields = append(changedFields, fieldName)
					}
				}
			}
		}

		// 解析data_before和data_after
		var dataBefore, dataAfter map[string]interface{}
		if dbRecord.OldValues != nil {
			json.Unmarshal([]byte(*dbRecord.OldValues), &dataBefore)
		}
		if dbRecord.NewValues != nil {
			json.Unmarshal([]byte(*dbRecord.NewValues), &dataAfter)
		}

		record := &ChangeRecord{
			RecordID:      fmt.Sprintf("%d", dbRecord.ID),
			OperationID:   dbRecord.OperationID,
			CiID:          *dbRecord.CiID,
			CiTypeID:      dbRecord.CiTypeID,
			OperationType: OperationType(dbRecord.OperationType),
			OperationTime: dbRecord.CreatedAt,
			OperatorID:    operatorUUID,
			OperatorName:  dbRecord.OperatorName,
			Source:        OperationSource(*dbRecord.Source),
			SourceDetail:  *dbRecord.SourceDetail,
			DataBefore:    dataBefore,
			DataAfter:     dataAfter,
			ChangedFields: changedFields,
			ChangeReason:  *dbRecord.ChangeReason,
		}

		// 设置审批信息
		if dbRecord.NeedsApproval {
			record.RequireApproval = true
			if dbRecord.IsApproved != nil {
				if *dbRecord.IsApproved {
					record.ApprovalStatus = "approved"
				} else {
					record.ApprovalStatus = "rejected"
				}
			} else {
				record.ApprovalStatus = "pending"
			}
		}

		records = append(records, record)
	}

	c.logger.Infow("Change history queried",
		logx.Field("ci_id", ciID),
		logx.Field("total_records", len(records)),
		logx.Field("limit", limit),
		logx.Field("offset", offset))

	return records, nil
}

// lifecycleManagerImpl 生命周期管理器实现
type lifecycleManagerImpl struct {
	svcCtx      *svc.ServiceContext
	config      *Configuration
	logger      logx.Logger
	initialized bool
}

func (l *lifecycleManagerImpl) Name() string {
	return "LifecycleManager"
}

func (l *lifecycleManagerImpl) Initialize(ctx context.Context) error {
	l.logger = logx.WithContext(ctx)
	l.initialized = true
	l.logger.Info("生命周期管理器初始化完成")
	return nil
}

func (l *lifecycleManagerImpl) Shutdown(ctx context.Context) error {
	l.initialized = false
	l.logger.Info("生命周期管理器已关闭")
	return nil
}

func (l *lifecycleManagerImpl) HealthCheck(ctx context.Context) error {
	if !l.initialized {
		return fmt.Errorf("生命周期管理器未初始化")
	}
	return nil
}

func (l *lifecycleManagerImpl) CreateState(ctx context.Context, operation *CiOperationContext) (*LifecycleState, error) {
	state := &LifecycleState{
		StateID:     uuidx.NewUUID().String(),
		StateName:   string(StageDraft),
		StateType:   string(StageDraft),
		CiTypeID:    operation.CiTypeID,
		EnteredAt:   time.Now(),
		TriggerType: "manual",
		TriggeredBy: operation.OperatorID,
	}

	if operation.CiID != nil {
		state.CiID = *operation.CiID
	}

	// 保存到数据库
	startTime := time.Now()

	// 准备state_data JSON
	var stateDataJSON string
	if len(operation.Metadata) > 0 {
		if jsonBytes, err := json.Marshal(operation.Metadata); err == nil {
			stateDataJSON = string(jsonBytes)
		}
	}

	// 创建数据库记录
	builder := l.svcCtx.DB.CiLifecycleState.Create().
		SetStateID(state.StateID).
		SetCiTypeID(operation.CiTypeID).
		SetStateName(string(StageDraft)).
		SetStateType(string(StageDraft)).
		SetEnteredAt(state.EnteredAt).
		SetTriggerType("manual").
		SetTriggeredBy(uint64(operation.OperatorID.IntPart())).
		SetOperationID(operation.OperationID).
		SetIsCurrent(true).
		SetIsFinal(false).
		SetCanRetry(true)

	// 设置CI ID
	if operation.CiID != nil {
		builder.SetCiID(*operation.CiID)
	}

	// 设置状态数据
	if stateDataJSON != "" {
		builder.SetStateData(stateDataJSON)
	}

	// 设置预期超时时间（根据配置）
	if l.config.Lifecycle.EnableTimeout {
		expectedExitAt := time.Now().Add(l.config.Lifecycle.DefaultTimeout)
		builder.SetExpectedExitAt(expectedExitAt)
		state.ExpectedExitAt = &expectedExitAt
	}

	dbState, err := builder.Save(ctx)
	if err != nil {
		l.logger.Errorw("Failed to create lifecycle state",
			logx.Field("state_id", state.StateID),
			logx.Field("error", err))
		return nil, err
	}

	l.logger.Infow("Lifecycle state created",
		logx.Field("state_id", dbState.StateID),
		logx.Field("state_type", dbState.StateType),
		logx.Field("ci_id", dbState.CiID),
		logx.Field("duration_ms", time.Since(startTime).Milliseconds()))

	return state, nil
}

func (l *lifecycleManagerImpl) UpdateState(ctx context.Context, stateID string, newStage LifecycleStage) (*LifecycleState, error) {
	// 1. 查询当前状态
	currentState, err := l.svcCtx.DB.CiLifecycleState.Query().
		Where(
			cilifecyclestate.StateIDEQ(stateID),
			cilifecyclestate.IsCurrentEQ(true),
		).
		First(ctx)

	if err != nil {
		l.logger.Errorw("Failed to find current state",
			logx.Field("state_id", stateID),
			logx.Field("error", err))
		return nil, fmt.Errorf("未找到状态: %s", stateID)
	}

	// 2. 检查状态转换是否合法
	currentStage := LifecycleStage(currentState.StateType)
	allowed, err := l.CheckStateTransition(ctx, currentStage, newStage)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, fmt.Errorf("不允许从 %s 转换到 %s", currentStage, newStage)
	}

	// 3. 标记当前状态为非当前
	exitTime := time.Now()
	duration := int(exitTime.Sub(currentState.EnteredAt).Seconds())

	_, err = l.svcCtx.DB.CiLifecycleState.UpdateOneID(currentState.ID).
		SetIsCurrent(false).
		SetExitedAt(exitTime).
		SetDurationSeconds(duration).
		Save(ctx)

	if err != nil {
		l.logger.Errorw("Failed to update current state",
			logx.Field("state_id", stateID),
			logx.Field("error", err))
		return nil, err
	}

	// 4. 创建新状态记录
	newStateID := uuidx.NewUUID().String()
	newState := l.svcCtx.DB.CiLifecycleState.Create().
		SetStateID(newStateID).
		SetCiID(*currentState.CiID).
		SetCiTypeID(currentState.CiTypeID).
		SetStateName(string(newStage)).
		SetStateType(string(newStage)).
		SetPreviousState(currentState.StateType).
		SetEnteredAt(exitTime).
		SetTriggerType("auto").
		SetOperationID(*currentState.OperationID).
		SetIsCurrent(true).
		SetIsFinal(l.isFinalStage(newStage)).
		SetCanRetry(l.canRetryStage(newStage)).
		SetTenantID(currentState.TenantID)

	// 设置预期超时时间
	if l.config.Lifecycle.EnableTimeout && !l.isFinalStage(newStage) {
		expectedExitAt := time.Now().Add(l.config.Lifecycle.DefaultTimeout)
		newState.SetExpectedExitAt(expectedExitAt)
	}

	dbNewState, err := newState.Save(ctx)
	if err != nil {
		l.logger.Errorw("Failed to create new state",
			logx.Field("new_stage", newStage),
			logx.Field("error", err))
		return nil, err
	}

	// 5. 构造返回结果
	operatorUUID, _ := uuid.FromString(fmt.Sprintf("%d", *dbNewState.TriggeredBy))
	result := &LifecycleState{
		StateID:     dbNewState.StateID,
		StateName:   dbNewState.StateName,
		StateType:   dbNewState.StateType,
		CiID:        *dbNewState.CiID,
		CiTypeID:    dbNewState.CiTypeID,
		EnteredAt:   dbNewState.EnteredAt,
		TriggerType: dbNewState.TriggerType,
		TriggeredBy: operatorUUID,
		IsTimeout:   dbNewState.IsTimeout,
		HasError:    dbNewState.HasError,
	}

	if dbNewState.ExpectedExitAt != nil {
		result.ExpectedExitAt = dbNewState.ExpectedExitAt
	}

	l.logger.Infow("Lifecycle state updated",
		logx.Field("state_id", stateID),
		logx.Field("previous_stage", currentStage),
		logx.Field("new_stage", newStage),
		logx.Field("new_state_id", newStateID))

	return result, nil
}

// isFinalStage 判断是否为最终状态
func (l *lifecycleManagerImpl) isFinalStage(stage LifecycleStage) bool {
	switch stage {
	case StageCompleted, StageCancelled, StageExpired:
		return true
	default:
		return false
	}
}

// canRetryStage 判断该状态是否可以重试
func (l *lifecycleManagerImpl) canRetryStage(stage LifecycleStage) bool {
	switch stage {
	case StageDraft, StageSubmitted:
		return true
	default:
		return false
	}
}

func (l *lifecycleManagerImpl) GetCurrentState(ctx context.Context, ciID uint64) (*LifecycleState, error) {
	// 查询当前状态
	dbState, err := l.svcCtx.DB.CiLifecycleState.Query().
		Where(
			cilifecyclestate.CiIDEQ(ciID),
			cilifecyclestate.IsCurrentEQ(true),
		).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("CI %d 没有当前状态", ciID)
		}
		l.logger.Errorw("Failed to query current state",
			logx.Field("ci_id", ciID),
			logx.Field("error", err))
		return nil, err
	}

	// 转换为LifecycleState结构
	operatorUUID, _ := uuid.FromString(fmt.Sprintf("%d", *dbState.TriggeredBy))

	// 解析state_data JSON
	var stateData map[string]interface{}
	if dbState.StateData != nil {
		json.Unmarshal([]byte(*dbState.StateData), &stateData)
	}

	state := &LifecycleState{
		StateID:      dbState.StateID,
		StateName:    dbState.StateName,
		StateType:    dbState.StateType,
		CiID:         *dbState.CiID,
		CiTypeID:     dbState.CiTypeID,
		EnteredAt:    dbState.EnteredAt,
		TriggerType:  dbState.TriggerType,
		TriggeredBy:  operatorUUID,
		StateData:    stateData,
		IsTimeout:    dbState.IsTimeout,
		HasError:     dbState.HasError,
		ErrorMessage: *dbState.ErrorMessage,
	}

	if dbState.ExpectedExitAt != nil {
		state.ExpectedExitAt = dbState.ExpectedExitAt
	}

	return state, nil
}

func (l *lifecycleManagerImpl) CheckStateTransition(ctx context.Context, currentStage, targetStage LifecycleStage) (bool, error) {
	// 定义状态转换规则（状态机）
	// 格式: currentStage -> 允许转换的目标状态列表
	allowedTransitions := map[LifecycleStage][]LifecycleStage{
		StageDraft: {
			StageSubmitted,  // draft -> submitted
			StageCancelled,  // draft -> cancelled
		},
		StageSubmitted: {
			StageValidated,  // submitted -> validated
			StageDraft,      // submitted -> draft (退回修改)
			StageCancelled,  // submitted -> cancelled
		},
		StageValidated: {
			StageApproved,   // validated -> approved
			StageSubmitted,  // validated -> submitted (退回)
			StageCancelled,  // validated -> cancelled
		},
		StageApproved: {
			StageExecuted,   // approved -> executed
			StageCancelled,  // approved -> cancelled
		},
		StageExecuted: {
			StageCompleted,  // executed -> completed
			StageCancelled,  // executed -> cancelled (异常取消)
		},
		StageCompleted: {
			// 已完成状态是最终状态，不允许转换
		},
		StageCancelled: {
			// 已取消状态是最终状态，不允许转换
		},
		StageExpired: {
			// 已过期状态是最终状态，不允许转换
		},
	}

	// 检查当前状态是否有定义的转换规则
	allowed, exists := allowedTransitions[currentStage]
	if !exists {
		return false, fmt.Errorf("未知的状态: %s", currentStage)
	}

	// 检查目标状态是否在允许列表中
	for _, allowedStage := range allowed {
		if allowedStage == targetStage {
			l.logger.Infow("State transition allowed",
				logx.Field("current_stage", currentStage),
				logx.Field("target_stage", targetStage))
			return true, nil
		}
	}

	l.logger.Warnw("State transition not allowed",
		logx.Field("current_stage", currentStage),
		logx.Field("target_stage", targetStage),
		logx.Field("allowed_transitions", allowed))

	return false, nil
}

func (l *lifecycleManagerImpl) HandleTimeout(ctx context.Context, stateID string) error {
	// 1. 查询状态
	state, err := l.svcCtx.DB.CiLifecycleState.Query().
		Where(
			cilifecyclestate.StateIDEQ(stateID),
			cilifecyclestate.IsCurrentEQ(true),
		).
		First(ctx)

	if err != nil {
		l.logger.Errorw("Failed to find state for timeout handling",
			logx.Field("state_id", stateID),
			logx.Field("error", err))
		return err
	}

	// 2. 检查是否真的超时
	if state.ExpectedExitAt == nil {
		return fmt.Errorf("状态 %s 没有设置预期退出时间", stateID)
	}

	if time.Now().Before(*state.ExpectedExitAt) {
		return fmt.Errorf("状态 %s 尚未超时", stateID)
	}

	// 3. 标记为超时并转换到expired状态
	_, err = l.svcCtx.DB.CiLifecycleState.UpdateOneID(state.ID).
		SetIsTimeout(true).
		SetErrorMessage("状态超时").
		Save(ctx)

	if err != nil {
		l.logger.Errorw("Failed to mark state as timeout",
			logx.Field("state_id", stateID),
			logx.Field("error", err))
		return err
	}

	// 4. 转换到expired状态
	_, err = l.UpdateState(ctx, stateID, StageExpired)
	if err != nil {
		l.logger.Errorw("Failed to transition to expired state",
			logx.Field("state_id", stateID),
			logx.Field("error", err))
		return err
	}

	l.logger.Infow("State timeout handled",
		logx.Field("state_id", stateID),
		logx.Field("ci_id", state.CiID),
		logx.Field("expected_exit_at", state.ExpectedExitAt))

	return nil
}

func (l *lifecycleManagerImpl) QueryStates(ctx context.Context, filters map[string]interface{}) ([]*LifecycleState, error) {
	query := l.svcCtx.DB.CiLifecycleState.Query()

	// 应用过滤条件
	if ciID, ok := filters["ci_id"].(uint64); ok {
		query = query.Where(cilifecyclestate.CiIDEQ(ciID))
	}

	if ciTypeID, ok := filters["ci_type_id"].(uint64); ok {
		query = query.Where(cilifecyclestate.CiTypeIDEQ(ciTypeID))
	}

	if stateType, ok := filters["state_type"].(string); ok {
		query = query.Where(cilifecyclestate.StateTypeEQ(stateType))
	}

	if isCurrent, ok := filters["is_current"].(bool); ok {
		query = query.Where(cilifecyclestate.IsCurrentEQ(isCurrent))
	}

	if isTimeout, ok := filters["is_timeout"].(bool); ok {
		query = query.Where(cilifecyclestate.IsTimeoutEQ(isTimeout))
	}

	if hasError, ok := filters["has_error"].(bool); ok {
		query = query.Where(cilifecyclestate.HasErrorEQ(hasError))
	}

	// 排序和分页
	limit := 20
	offset := 0
	if l, ok := filters["limit"].(int); ok && l > 0 {
		limit = l
	}
	if o, ok := filters["offset"].(int); ok && o >= 0 {
		offset = o
	}

	dbStates, err := query.
		Order(ent.Desc(cilifecyclestate.FieldCreatedAt)).
		Limit(limit).
		Offset(offset).
		All(ctx)

	if err != nil {
		l.logger.Errorw("Failed to query lifecycle states",
			logx.Field("filters", filters),
			logx.Field("error", err))
		return nil, err
	}

	// 转换为LifecycleState结构
	states := make([]*LifecycleState, 0, len(dbStates))
	for _, dbState := range dbStates {
		operatorUUID, _ := uuid.FromString(fmt.Sprintf("%d", *dbState.TriggeredBy))

		var stateData map[string]interface{}
		if dbState.StateData != nil {
			json.Unmarshal([]byte(*dbState.StateData), &stateData)
		}

		state := &LifecycleState{
			StateID:     dbState.StateID,
			StateName:   dbState.StateName,
			StateType:   dbState.StateType,
			CiID:        *dbState.CiID,
			CiTypeID:    dbState.CiTypeID,
			EnteredAt:   dbState.EnteredAt,
			TriggerType: dbState.TriggerType,
			TriggeredBy: operatorUUID,
			StateData:   stateData,
			IsTimeout:   dbState.IsTimeout,
			HasError:    dbState.HasError,
		}

		if dbState.ExpectedExitAt != nil {
			state.ExpectedExitAt = dbState.ExpectedExitAt
		}

		if dbState.ErrorMessage != nil {
			state.ErrorMessage = *dbState.ErrorMessage
		}

		states = append(states, state)
	}

	l.logger.Infow("Lifecycle states queried",
		logx.Field("total_states", len(states)),
		logx.Field("filters", filters))

	return states, nil
}

// approvalManagerImpl 审批管理器实现
type approvalManagerImpl struct {
	svcCtx      *svc.ServiceContext
	config      *Configuration
	logger      logx.Logger
	initialized bool
}

func (a *approvalManagerImpl) Name() string {
	return "ApprovalManager"
}

func (a *approvalManagerImpl) Initialize(ctx context.Context) error {
	a.logger = logx.WithContext(ctx)
	a.initialized = true
	a.logger.Info("审批管理器初始化完成")
	return nil
}

func (a *approvalManagerImpl) Shutdown(ctx context.Context) error {
	a.initialized = false
	a.logger.Info("审批管理器已关闭")
	return nil
}

func (a *approvalManagerImpl) HealthCheck(ctx context.Context) error {
	if !a.initialized {
		return fmt.Errorf("审批管理器未初始化")
	}
	return nil
}

func (a *approvalManagerImpl) CheckRequireApproval(ctx context.Context, operation *CiOperationContext) (bool, string, error) {
	// TODO: 实现审批需求检查
	// 暂时返回不需要审批
	return false, "", nil
}

func (a *approvalManagerImpl) SubmitForApproval(ctx context.Context, operation *CiOperationContext) (string, error) {
	// TODO: 实现提交审批
	return "", fmt.Errorf("提交审批功能暂未实现")
}

func (a *approvalManagerImpl) ProcessApproval(ctx context.Context, approvalID string, action string, comment string, approverID uuid.UUID) error {
	// TODO: 实现处理审批
	return fmt.Errorf("处理审批功能暂未实现")
}

func (a *approvalManagerImpl) GetApprovalStatus(ctx context.Context, approvalID string) (string, error) {
	// TODO: 实现获取审批状态
	return "", fmt.Errorf("获取审批状态功能暂未实现")
}

func (a *approvalManagerImpl) QueryPendingApprovals(ctx context.Context, approverID uuid.UUID) ([]map[string]interface{}, error) {
	// TODO: 实现查询待审批项
	return []map[string]interface{}{}, nil
}

// dataPersisterImpl 数据持久化器实现
type dataPersisterImpl struct {
	svcCtx      *svc.ServiceContext
	config      *Configuration
	logger      logx.Logger
	initialized bool
}

func (d *dataPersisterImpl) Name() string {
	return "DataPersister"
}

func (d *dataPersisterImpl) Initialize(ctx context.Context) error {
	d.logger = logx.WithContext(ctx)
	d.initialized = true
	d.logger.Info("数据持久化器初始化完成")
	return nil
}

func (d *dataPersisterImpl) Shutdown(ctx context.Context) error {
	d.initialized = false
	d.logger.Info("数据持久化器已关闭")
	return nil
}

func (d *dataPersisterImpl) HealthCheck(ctx context.Context) error {
	if !d.initialized {
		return fmt.Errorf("数据持久化器未初始化")
	}
	return nil
}

func (d *dataPersisterImpl) Create(ctx context.Context, data *cmdb.CisInfo) (*cmdb.CisInfo, error) {
	// TODO: 调用现有的CreateCisLogic进行数据创建
	// 这里暂时返回模拟数据
	createdData := *data
	id := uint64(time.Now().UnixNano()) // 模拟生成的ID
	createdData.Id = &id

	return &createdData, nil
}

func (d *dataPersisterImpl) Update(ctx context.Context, ciID uint64, data *cmdb.CisInfo) (*cmdb.CisInfo, error) {
	// TODO: 调用现有的UpdateCisLogic进行数据更新
	updatedData := *data
	updatedData.Id = &ciID

	return &updatedData, nil
}

func (d *dataPersisterImpl) Delete(ctx context.Context, ciID uint64) error {
	// TODO: 调用现有的DeleteCisLogic进行数据删除
	return nil
}

func (d *dataPersisterImpl) BatchCreate(ctx context.Context, dataList []*cmdb.CisInfo) ([]*cmdb.CisInfo, error) {
	var results []*cmdb.CisInfo

	for _, data := range dataList {
		created, err := d.Create(ctx, data)
		if err != nil {
			d.logger.Errorf("批量创建失败: %v", err)
			continue
		}
		results = append(results, created)
	}

	return results, nil
}

func (d *dataPersisterImpl) BatchUpdate(ctx context.Context, updates map[uint64]*cmdb.CisInfo) ([]*cmdb.CisInfo, error) {
	var results []*cmdb.CisInfo

	for ciID, data := range updates {
		updated, err := d.Update(ctx, ciID, data)
		if err != nil {
			d.logger.Errorf("批量更新失败: %v", err)
			continue
		}
		results = append(results, updated)
	}

	return results, nil
}

func (d *dataPersisterImpl) BatchDelete(ctx context.Context, ciIDs []uint64) error {
	for _, ciID := range ciIDs {
		if err := d.Delete(ctx, ciID); err != nil {
			d.logger.Errorf("批量删除失败: %v", err)
		}
	}

	return nil
}

func (d *dataPersisterImpl) Get(ctx context.Context, ciID uint64) (*cmdb.CisInfo, error) {
	// TODO: 调用现有的GetCisByIdLogic进行数据获取
	return nil, fmt.Errorf("获取数据功能暂未实现")
}
