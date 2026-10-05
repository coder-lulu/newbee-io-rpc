package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coder-lulu/newbee-cmdb-rpc/cmdbclient"
	"github.com/coder-lulu/newbee-cmdb-rpc/types/cmdb"
	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-common/v2/utils/pointy"
	"github.com/zeromicro/go-zero/core/logx"
)

// OutputProcessor 输出处理器
// 负责将转换后的数据输出到不同的目标系统
type OutputProcessor struct {
	db      *ent.Client
	cmdbRpc cmdbclient.Cmdb
	logger  logx.Logger
}

// NewOutputProcessor 创建输出处理器
func NewOutputProcessor(db *ent.Client, cmdbRpc cmdbclient.Cmdb, logger logx.Logger) *OutputProcessor {
	return &OutputProcessor{
		db:      db,
		cmdbRpc: cmdbRpc,
		logger:  logger,
	}
}

// ProcessOutput 处理输出任务
func (p *OutputProcessor) ProcessOutput(ctx context.Context, task *ent.OutputTask, data []map[string]interface{}) error {
	p.logger.Infow("Processing output task",
		logx.Field("task_id", task.ID),
		logx.Field("output_target", task.OutputTarget),
		logx.Field("records_count", len(data)))

	// 根据output_target类型选择不同的处理方式
	switch task.OutputTarget {
	case "cmdb":
		return p.outputToCMDB(ctx, task, data)
	case "database":
		return p.outputToDatabase(ctx, task, data)
	case "file":
		return p.outputToFile(ctx, task, data)
	case "api":
		return p.outputToAPI(ctx, task, data)
	default:
		return fmt.Errorf("unsupported output target: %s", task.OutputTarget)
	}
}

// outputToCMDB 输出数据到CMDB
func (p *OutputProcessor) outputToCMDB(ctx context.Context, task *ent.OutputTask, data []map[string]interface{}) error {
	// 1. 解析target_config
	var config CMDBOutputConfig
	if task.TargetConfig != "" {
		if err := json.Unmarshal([]byte(task.TargetConfig), &config); err != nil {
			return fmt.Errorf("failed to parse target_config: %w", err)
		}
	}

	// 2. 验证配置
	if config.CITypeID == 0 {
		return fmt.Errorf("ci_type_id is required in target_config")
	}

	// 设置默认值
	if config.UniqueKeyField == "" {
		config.UniqueKeyField = "unique_key"
	}
	if config.AutoCreate == nil {
		config.AutoCreate = pointy.GetPointer(true)
	}
	if config.AutoUpdate == nil {
		config.AutoUpdate = pointy.GetPointer(true)
	}
	if config.ConflictResolution == "" {
		config.ConflictResolution = "merge"
	}

	p.logger.Infow("Output to CMDB configuration",
		logx.Field("ci_type_id", config.CITypeID),
		logx.Field("unique_key_field", config.UniqueKeyField),
		logx.Field("auto_create", *config.AutoCreate),
		logx.Field("auto_update", *config.AutoUpdate),
		logx.Field("conflict_resolution", config.ConflictResolution))

	// 3. 遍历数据，调用CMDB RPC
	successCount := 0
	failedCount := 0
	var lastError error

	for i, record := range data {
		// 提取unique_key
		uniqueKey := p.extractUniqueKey(record, config.UniqueKeyField)
		if uniqueKey == "" {
			logx.WithContext(ctx).Infow("Record missing unique key, skipping",
				logx.Field("index", i),
				logx.Field("unique_key_field", config.UniqueKeyField))
			failedCount++
			continue
		}

		// 转换记录为string map（CMDB RPC需要）
		attributes := p.convertToStringMap(record)

		// 构建RPC请求
		req := &cmdb.DiscoveredCIData{
			CiTypeId:           pointy.GetPointer(config.CITypeID),
			UniqueKey:          pointy.GetPointer(uniqueKey),
			Attributes:         attributes,
			Source:             pointy.GetPointer(task.TaskName),
			SourceType:         pointy.GetPointer("unified-io"),
			AutoCreate:         config.AutoCreate,
			AutoUpdate:         config.AutoUpdate,
			ConflictResolution: pointy.GetPointer(config.ConflictResolution),
		}

		// 调用CMDB RPC
		resp, err := p.cmdbRpc.WriteDiscoveredCI(ctx, req)
		if err != nil {
			p.logger.Errorw("Failed to write CI to CMDB",
				logx.Field("index", i),
				logx.Field("unique_key", uniqueKey),
				logx.Field("error", err))
			failedCount++
			lastError = err
			continue
		}

		p.logger.Infow("Successfully wrote CI to CMDB",
			logx.Field("index", i),
			logx.Field("ci_id", resp.CiId),
			logx.Field("action", resp.Action),
			logx.Field("message", resp.Message))
		successCount++
	}

	// 4. 记录结果
	p.logger.Infow("CMDB output completed",
		logx.Field("task_id", task.ID),
		logx.Field("total", len(data)),
		logx.Field("success", successCount),
		logx.Field("failed", failedCount))

	// 如果有失败的记录，返回错误
	if failedCount > 0 {
		return fmt.Errorf("failed to write %d/%d records to CMDB, last error: %v", failedCount, len(data), lastError)
	}

	return nil
}

// outputToDatabase 输出数据到数据库（占位符）
func (p *OutputProcessor) outputToDatabase(ctx context.Context, task *ent.OutputTask, data []map[string]interface{}) error {
	// TODO: 实现数据库输出逻辑
	p.logger.Infow("Database output not implemented yet",
		logx.Field("task_id", task.ID),
		logx.Field("records_count", len(data)))
	return fmt.Errorf("database output not implemented yet")
}

// outputToFile 输出数据到文件（占位符）
func (p *OutputProcessor) outputToFile(ctx context.Context, task *ent.OutputTask, data []map[string]interface{}) error {
	// TODO: 实现文件输出逻辑
	p.logger.Infow("File output not implemented yet",
		logx.Field("task_id", task.ID),
		logx.Field("records_count", len(data)))
	return fmt.Errorf("file output not implemented yet")
}

// outputToAPI 输出数据到API（占位符）
func (p *OutputProcessor) outputToAPI(ctx context.Context, task *ent.OutputTask, data []map[string]interface{}) error {
	// TODO: 实现API输出逻辑
	p.logger.Infow("API output not implemented yet",
		logx.Field("task_id", task.ID),
		logx.Field("records_count", len(data)))
	return fmt.Errorf("API output not implemented yet")
}

// extractUniqueKey 从记录中提取唯一键
func (p *OutputProcessor) extractUniqueKey(record map[string]interface{}, keyField string) string {
	value, ok := record[keyField]
	if !ok {
		return ""
	}

	// 转换为字符串
	switch v := value.(type) {
	case string:
		return v
	case int, int32, int64, uint, uint32, uint64:
		return fmt.Sprintf("%v", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// convertToStringMap 将map[string]interface{}转换为map[string]string
func (p *OutputProcessor) convertToStringMap(record map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for key, value := range record {
		if value == nil {
			continue
		}

		// 转换为字符串
		switch v := value.(type) {
		case string:
			result[key] = v
		case int, int32, int64, uint, uint32, uint64:
			result[key] = fmt.Sprintf("%d", v)
		case float32, float64:
			result[key] = fmt.Sprintf("%v", v)
		case bool:
			if v {
				result[key] = "true"
			} else {
				result[key] = "false"
			}
		default:
			// 其他类型转换为JSON字符串
			if jsonBytes, err := json.Marshal(v); err == nil {
				result[key] = string(jsonBytes)
			} else {
				result[key] = fmt.Sprintf("%v", v)
			}
		}
	}
	return result
}

// CMDBOutputConfig CMDB输出配置
type CMDBOutputConfig struct {
	CITypeID           uint64  `json:"ci_type_id"`            // CI类型ID
	UniqueKeyField     string  `json:"unique_key_field"`      // 唯一键字段名
	AutoCreate         *bool   `json:"auto_create"`           // 自动创建CI
	AutoUpdate         *bool   `json:"auto_update"`           // 自动更新属性
	ConflictResolution string  `json:"conflict_resolution"`   // 冲突策略（skip/update/merge）
}
