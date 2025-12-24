package transform

import (
	"fmt"
	"strings"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/zeromicro/go-zero/core/logx"
)

// FieldMapper 字段映射器
// 职责:
// 1. 从源数据中获取字段值（支持嵌套路径）
// 2. 将值设置到目标数据（支持嵌套路径）
type FieldMapper struct {
	logger logx.Logger
}

// NewFieldMapper 创建字段映射器
func NewFieldMapper(logger logx.Logger) *FieldMapper {
	return &FieldMapper{
		logger: logger,
	}
}

// GetSourceValue 从源数据中获取字段值
// 支持简单字段名和嵌套路径（如 "user.profile.name"）
func (m *FieldMapper) GetSourceValue(sourceData map[string]interface{}, mapping *ent.FieldMapping) (interface{}, bool) {
	// 使用字段路径或字段名
	fieldPath := mapping.SourceField
	if mapping.SourceFieldPath != "" {
		fieldPath = mapping.SourceFieldPath
	}

	return m.getValueByPath(sourceData, fieldPath)
}

// SetTargetValue 将值设置到目标数据
// 支持简单字段名和嵌套路径（如 "address.city"）
func (m *FieldMapper) SetTargetValue(targetData map[string]interface{}, mapping *ent.FieldMapping, value interface{}) {
	// 使用字段路径或字段名
	fieldPath := mapping.TargetField
	if mapping.TargetFieldPath != "" {
		fieldPath = mapping.TargetFieldPath
	}

	m.setValueByPath(targetData, fieldPath, value)
}

// getValueByPath 通过路径获取值
// 支持点分隔的嵌套路径，如 "user.profile.name"
func (m *FieldMapper) getValueByPath(data map[string]interface{}, path string) (interface{}, bool) {
	// 如果路径不包含点，直接返回
	if !strings.Contains(path, ".") {
		value, exists := data[path]
		return value, exists
	}

	// 拆分路径
	parts := strings.Split(path, ".")
	current := interface{}(data)

	// 遍历路径
	for i, part := range parts {
		// 检查当前值是否为map
		currentMap, ok := current.(map[string]interface{})
		if !ok {
			m.logger.Debugw("Path navigation failed: not a map",
				logx.Field("path", path),
				logx.Field("current_part", part),
				logx.Field("part_index", i))
			return nil, false
		}

		// 获取下一级值
		nextValue, exists := currentMap[part]
		if !exists {
			m.logger.Debugw("Path navigation failed: key not found",
				logx.Field("path", path),
				logx.Field("missing_key", part),
				logx.Field("part_index", i))
			return nil, false
		}

		// 如果是最后一个部分，返回值
		if i == len(parts)-1 {
			return nextValue, true
		}

		// 否则继续下一级
		current = nextValue
	}

	return nil, false
}

// setValueByPath 通过路径设置值
// 支持点分隔的嵌套路径，如 "address.city"
// 会自动创建中间map
func (m *FieldMapper) setValueByPath(data map[string]interface{}, path string, value interface{}) {
	// 如果路径不包含点，直接设置
	if !strings.Contains(path, ".") {
		data[path] = value
		return
	}

	// 拆分路径
	parts := strings.Split(path, ".")
	current := data

	// 遍历路径，创建中间map
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]

		// 检查当前键是否存在
		nextValue, exists := current[part]
		if !exists {
			// 创建新的map
			nextMap := make(map[string]interface{})
			current[part] = nextMap
			current = nextMap
		} else {
			// 检查现有值是否为map
			nextMap, ok := nextValue.(map[string]interface{})
			if !ok {
				// 类型不匹配，覆盖为map
				m.logger.Errorw("Path conflict: overwriting non-map value with map",
					logx.Field("path", path),
					logx.Field("conflicting_key", part))
				nextMap = make(map[string]interface{})
				current[part] = nextMap
			}
			current = nextMap
		}
	}

	// 设置最后一个部分的值
	lastPart := parts[len(parts)-1]
	current[lastPart] = value

	m.logger.Debugw("Value set successfully",
		logx.Field("path", path),
		logx.Field("value", value))
}

// GetNestedValue 获取嵌套值的辅助函数（公开方法）
func (m *FieldMapper) GetNestedValue(data map[string]interface{}, path string) (interface{}, bool) {
	return m.getValueByPath(data, path)
}

// SetNestedValue 设置嵌套值的辅助函数（公开方法）
func (m *FieldMapper) SetNestedValue(data map[string]interface{}, path string, value interface{}) {
	m.setValueByPath(data, path, value)
}

// ExtractMultipleFields 批量提取多个字段
func (m *FieldMapper) ExtractMultipleFields(sourceData map[string]interface{}, fieldPaths []string) map[string]interface{} {
	result := make(map[string]interface{})

	for _, path := range fieldPaths {
		if value, exists := m.getValueByPath(sourceData, path); exists {
			result[path] = value
		}
	}

	return result
}

// MergeData 合并两个map（深度合并）
func (m *FieldMapper) MergeData(target, source map[string]interface{}) map[string]interface{} {
	if target == nil {
		target = make(map[string]interface{})
	}

	for key, sourceValue := range source {
		targetValue, exists := target[key]

		// 如果两边都是map，递归合并
		sourceMap, sourceIsMap := sourceValue.(map[string]interface{})
		targetMap, targetIsMap := targetValue.(map[string]interface{})

		if exists && sourceIsMap && targetIsMap {
			target[key] = m.MergeData(targetMap, sourceMap)
		} else {
			// 否则直接覆盖
			target[key] = sourceValue
		}
	}

	return target
}

// FlattenData 扁平化嵌套数据
// 将 {"user": {"name": "John"}} 转换为 {"user.name": "John"}
func (m *FieldMapper) FlattenData(data map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range data {
		// 构建新的键名
		newKey := key
		if prefix != "" {
			newKey = prefix + "." + key
		}

		// 如果值是map，递归扁平化
		if valueMap, ok := value.(map[string]interface{}); ok {
			flattened := m.FlattenData(valueMap, newKey)
			for k, v := range flattened {
				result[k] = v
			}
		} else {
			result[newKey] = value
		}
	}

	return result
}

// UnflattenData 反扁平化数据
// 将 {"user.name": "John"} 转换为 {"user": {"name": "John"}}
func (m *FieldMapper) UnflattenData(flatData map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range flatData {
		m.setValueByPath(result, key, value)
	}

	return result
}

// ValidatePath 验证路径格式
func (m *FieldMapper) ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// 检查是否以点开始或结束
	if strings.HasPrefix(path, ".") || strings.HasSuffix(path, ".") {
		return fmt.Errorf("path cannot start or end with a dot: %s", path)
	}

	// 检查是否有连续的点
	if strings.Contains(path, "..") {
		return fmt.Errorf("path cannot contain consecutive dots: %s", path)
	}

	return nil
}
