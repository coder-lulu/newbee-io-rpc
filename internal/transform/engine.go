package transform

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/zeromicro/go-zero/core/logx"
)

// Engine Transform引擎
// 职责:
// 1. 协调字段映射、类型转换、验证等组件
// 2. 管理Transform执行流程
// 3. 记录映射日志和统计信息
type Engine struct {
	config    *EngineConfig
	db        *ent.Client
	logger    logx.Logger

	// 组件
	mapper    *FieldMapper
	converter *TypeConverter
	validator *Validator
	lookup    *LookupResolver

	// 缓存
	cacheMu   sync.RWMutex
	cache     map[string]interface{}
}

// NewEngine 创建Transform引擎
func NewEngine(config *EngineConfig, db *ent.Client, logger logx.Logger) *Engine {
	if config == nil {
		config = DefaultEngineConfig()
	}

	e := &Engine{
		config: config,
		db:     db,
		logger: logger,
		cache:  make(map[string]interface{}),
	}

	// 初始化组件
	e.mapper = NewFieldMapper(logger)
	e.converter = NewTypeConverter(logger)
	e.validator = NewValidator(logger)
	e.lookup = NewLookupResolver(config.EnableLookupCache, config.LookupCacheTTL, logger)

	return e
}

// Transform 执行数据转换
// 这是Transform Engine的主入口函数
func (e *Engine) Transform(ctx *TransformContext) *TransformResult {
	startTime := time.Now()

	// 初始化统计
	ctx.Stats = &TransformStats{
		TotalFields:   len(ctx.Mappings),
		StartTime:     startTime,
		ErrorMessages: make([]string, 0),
	}

	// 初始化输出数据
	if ctx.TargetData == nil {
		ctx.TargetData = make(map[string]interface{})
	}

	// 初始化日志
	logs := make([]*MappingLogEntry, 0, len(ctx.Mappings))

	e.logger.Infow("Starting transform",
		logx.Field("tenant_id", ctx.TenantID),
		logx.Field("task_id", ctx.TaskID),
		logx.Field("total_mappings", len(ctx.Mappings)))

	// 按优先级排序映射规则
	sortedMappings := e.sortMappingsByPriority(ctx.Mappings)

	// 执行映射
	for _, mapping := range sortedMappings {
		// 检查是否启用
		if !mapping.IsActive {
			ctx.Stats.SkippedFields++
			e.logger.Debugw("Skipping inactive mapping",
				logx.Field("mapping_id", mapping.ID),
				logx.Field("mapping_name", mapping.MappingName))
			continue
		}

		// 执行单个字段映射
		logEntry := e.transformField(ctx, mapping)

		// 更新统计
		switch logEntry.Status {
		case MappingStatusSuccess:
			ctx.Stats.SuccessFields++
		case MappingStatusFailed:
			ctx.Stats.FailedFields++
			ctx.Stats.ErrorMessages = append(ctx.Stats.ErrorMessages, logEntry.ErrorMessage)
		case MappingStatusSkipped:
			ctx.Stats.SkippedFields++
		}

		// 记录日志
		if e.shouldLogMapping(logEntry) {
			logs = append(logs, logEntry)
		}

		// 检查错误限制
		if e.config.MaxErrors > 0 && ctx.Stats.FailedFields >= e.config.MaxErrors {
			e.logger.Errorw("Maximum errors exceeded",
				logx.Field("max_errors", e.config.MaxErrors),
				logx.Field("failed_fields", ctx.Stats.FailedFields))
			break
		}

		// 检查严格模式
		if e.config.StrictMode && logEntry.Status == MappingStatusFailed {
			e.logger.Errorw("Strict mode enabled, stopping on first error",
				logx.Field("mapping_id", mapping.ID),
				logx.Field("error", logEntry.ErrorMessage))
			break
		}
	}

	// 计算执行时长
	ctx.Stats.EndTime = time.Now()
	ctx.Stats.Duration = ctx.Stats.EndTime.Sub(ctx.Stats.StartTime)
	ctx.Stats.MappedFields = ctx.Stats.SuccessFields + ctx.Stats.FailedFields

	e.logger.Infow("Transform completed",
		logx.Field("total_fields", ctx.Stats.TotalFields),
		logx.Field("success_fields", ctx.Stats.SuccessFields),
		logx.Field("failed_fields", ctx.Stats.FailedFields),
		logx.Field("skipped_fields", ctx.Stats.SkippedFields),
		logx.Field("duration_ms", ctx.Stats.Duration.Milliseconds()))

	// 构建结果
	result := &TransformResult{
		Success:    ctx.Stats.FailedFields == 0,
		TargetData: ctx.TargetData,
		Stats:      ctx.Stats,
		Logs:       logs,
	}

	if ctx.Stats.FailedFields > 0 {
		result.ErrorMessage = fmt.Sprintf("%d field(s) failed to transform", ctx.Stats.FailedFields)
	}

	// 持久化映射日志
	if e.config.EnableMappingLog && len(logs) > 0 {
		go e.saveMappingLogs(ctx.Ctx, ctx.TenantID, logs)
	}

	return result
}

// transformField 转换单个字段
func (e *Engine) transformField(ctx *TransformContext, mapping *ent.FieldMapping) *MappingLogEntry {
	logEntry := &MappingLogEntry{
		FieldMappingID: mapping.ID,
		MappingName:    mapping.MappingName,
		SourceField:    mapping.SourceField,
		TargetField:    mapping.TargetField,
		Status:         MappingStatusSuccess,
		LoggedAt:       time.Now(),
	}

	// 1. 获取源字段值
	sourceValue, exists := e.mapper.GetSourceValue(ctx.SourceData, mapping)
	logEntry.SourceValue = sourceValue

	// 处理字段不存在的情况
	if !exists {
		if mapping.IsRequired {
			logEntry.Status = MappingStatusFailed
			logEntry.ErrorMessage = fmt.Sprintf("Required field '%s' not found in source data", mapping.SourceField)
			e.logger.Errorw("Required field missing",
				logx.Field("mapping_id", mapping.ID),
				logx.Field("source_field", mapping.SourceField))
			return logEntry
		}

		// 使用默认值
		if mapping.DefaultValue != "" {
			sourceValue = mapping.DefaultValue
			e.logger.Debugw("Using default value",
				logx.Field("mapping_id", mapping.ID),
				logx.Field("field", mapping.SourceField),
				logx.Field("default_value", sourceValue))
		} else if !mapping.AllowNull {
			logEntry.Status = MappingStatusFailed
			logEntry.ErrorMessage = fmt.Sprintf("Field '%s' is null but null values are not allowed", mapping.SourceField)
			return logEntry
		} else {
			// 字段不存在且允许null，跳过
			logEntry.Status = MappingStatusSkipped
			return logEntry
		}
	}

	// 2. 检查条件规则
	if mapping.ConditionRules != "" {
		shouldProcess, err := e.evaluateCondition(ctx, mapping, sourceValue)
		if err != nil {
			logEntry.Status = MappingStatusFailed
			logEntry.ErrorMessage = fmt.Sprintf("Condition evaluation error: %v", err)
			return logEntry
		}
		if !shouldProcess {
			logEntry.Status = MappingStatusSkipped
			e.logger.Debugw("Skipping field due to condition",
				logx.Field("mapping_id", mapping.ID),
				logx.Field("field", mapping.SourceField))
			return logEntry
		}
	}

	// 3. 执行查找表转换
	if mapping.LookupTable != "" {
		transformedValue, err := e.lookup.Resolve(sourceValue, mapping)
		if err != nil {
			logEntry.Status = MappingStatusFailed
			logEntry.ErrorMessage = fmt.Sprintf("Lookup failed: %v", err)
			return logEntry
		}
		sourceValue = transformedValue
	}

	// 4. 执行类型转换
	var targetValue interface{}
	var err error

	transformType := TransformType(mapping.TransformType)
	switch transformType {
	case TransformTypeDirect:
		// 直接映射
		targetValue = sourceValue

	case TransformTypeConvert:
		// 类型转换
		targetValue, err = e.converter.Convert(sourceValue, mapping)
		if err != nil {
			logEntry.Status = MappingStatusFailed
			logEntry.ErrorMessage = fmt.Sprintf("Type conversion failed: %v", err)
			return logEntry
		}

	case TransformTypeTemplate, TransformTypeScript, TransformTypeConcat, TransformTypeSplit, TransformTypeCalculate:
		// 复杂转换 - 委托给converter
		targetValue, err = e.converter.TransformWithConfig(sourceValue, mapping)
		if err != nil {
			logEntry.Status = MappingStatusFailed
			logEntry.ErrorMessage = fmt.Sprintf("Transform failed: %v", err)
			return logEntry
		}

	default:
		logEntry.Status = MappingStatusFailed
		logEntry.ErrorMessage = fmt.Sprintf("Unknown transform type: %s", transformType)
		return logEntry
	}

	logEntry.TargetValue = targetValue

	// 5. 数据验证
	if e.config.EnableValidation && mapping.ValidationRules != "" {
		if err := e.validator.Validate(targetValue, mapping); err != nil {
			logEntry.Status = MappingStatusFailed
			logEntry.ErrorMessage = fmt.Sprintf("Validation failed: %v", err)
			return logEntry
		}
	}

	// 6. 设置目标字段值
	e.mapper.SetTargetValue(ctx.TargetData, mapping, targetValue)

	e.logger.Debugw("Field transformed successfully",
		logx.Field("mapping_id", mapping.ID),
		logx.Field("source_field", mapping.SourceField),
		logx.Field("target_field", mapping.TargetField),
		logx.Field("transform_type", mapping.TransformType))

	return logEntry
}

// sortMappingsByPriority 按优先级和排序顺序排序映射规则
func (e *Engine) sortMappingsByPriority(mappings []*ent.FieldMapping) []*ent.FieldMapping {
	// 复制切片避免修改原始数据
	sorted := make([]*ent.FieldMapping, len(mappings))
	copy(sorted, mappings)

	// 使用简单的冒泡排序（映射规则通常不多）
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			// 先按优先级降序，再按排序顺序升序
			if sorted[j].Priority < sorted[j+1].Priority ||
				(sorted[j].Priority == sorted[j+1].Priority && sorted[j].SortOrder > sorted[j+1].SortOrder) {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted
}

// evaluateCondition 评估条件规则
func (e *Engine) evaluateCondition(ctx *TransformContext, mapping *ent.FieldMapping, value interface{}) (bool, error) {
	// TODO: 实现条件表达式评估
	// 这里需要一个简单的表达式引擎或使用第三方库
	// 暂时返回true，后续实现
	return true, nil
}

// shouldLogMapping 判断是否应该记录映射日志
func (e *Engine) shouldLogMapping(logEntry *MappingLogEntry) bool {
	if !e.config.EnableMappingLog {
		return false
	}

	switch logEntry.Status {
	case MappingStatusSuccess:
		return e.config.LogSuccessMapping
	case MappingStatusFailed:
		return e.config.LogFailedMapping
	case MappingStatusSkipped:
		return false // 默认不记录跳过的
	default:
		return false
	}
}

// saveMappingLogs 持久化映射日志到数据库
func (e *Engine) saveMappingLogs(ctx context.Context, tenantID uint64, logs []*MappingLogEntry) {
	// 检查db是否为nil（测试环境下可能为nil）
	if e.db == nil {
		e.logger.Debugw("Skip saving mapping logs: db is nil")
		return
	}

	for _, log := range logs {
		// 转换sourceValue和targetValue为字符串
		sourceValueStr := fmt.Sprintf("%v", log.SourceValue)
		targetValueStr := fmt.Sprintf("%v", log.TargetValue)

		// 限制长度
		if len(sourceValueStr) > 1000 {
			sourceValueStr = sourceValueStr[:1000]
		}
		if len(targetValueStr) > 1000 {
			targetValueStr = targetValueStr[:1000]
		}

		_, err := e.db.MappingLog.Create().
			SetFieldMappingID(log.FieldMappingID).
			SetSourceValue(sourceValueStr).
			SetTargetValue(targetValueStr).
			SetTransformStatus(string(log.Status)).
			SetNillableErrorMessage(&log.ErrorMessage).
			SetLoggedAt(log.LoggedAt).
			SetTenantID(tenantID).
			Save(ctx)

		if err != nil {
			e.logger.Errorw("Failed to save mapping log",
				logx.Field("mapping_id", log.FieldMappingID),
				logx.Field("error", err))
		}
	}
}
