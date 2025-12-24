package transform

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/zeromicro/go-zero/core/logx"
)

// TypeConverter 类型转换器
// 职责:
// 1. 基础类型转换（string, int, float, bool, time）
// 2. 模板转换
// 3. 字符串操作（拼接、拆分）
// 4. 数值计算
type TypeConverter struct {
	logger logx.Logger
}

// NewTypeConverter 创建类型转换器
func NewTypeConverter(logger logx.Logger) *TypeConverter {
	return &TypeConverter{
		logger: logger,
	}
}

// Convert 基础类型转换
// 根据FieldMapping的target_data_type执行转换
func (c *TypeConverter) Convert(value interface{}, mapping *ent.FieldMapping) (interface{}, error) {
	if value == nil {
		if mapping.AllowNull {
			return nil, nil
		}
		return nil, fmt.Errorf("null value not allowed")
	}

	// 如果没有指定目标类型，直接返回
	if mapping.TargetDataType == "" {
		return value, nil
	}

	targetType := mapping.TargetDataType

	c.logger.Debugw("Converting value",
		logx.Field("source_value", value),
		logx.Field("target_type", targetType))

	switch targetType {
	case "string":
		return c.toString(value)
	case "int", "integer":
		return c.toInt(value)
	case "float", "double":
		return c.toFloat(value)
	case "bool", "boolean":
		return c.toBool(value)
	case "datetime", "timestamp":
		return c.toTime(value, mapping)
	case "json":
		return c.toJSON(value)
	default:
		return nil, fmt.Errorf("unsupported target type: %s", targetType)
	}
}

// TransformWithConfig 复杂转换（使用transform_config）
func (c *TypeConverter) TransformWithConfig(value interface{}, mapping *ent.FieldMapping) (interface{}, error) {
	transformType := TransformType(mapping.TransformType)

	switch transformType {
	case TransformTypeTemplate:
		return c.applyTemplate(value, mapping)
	case TransformTypeConcat:
		return c.concat(value, mapping)
	case TransformTypeSplit:
		return c.split(value, mapping)
	case TransformTypeCalculate:
		return c.calculate(value, mapping)
	case TransformTypeScript:
		return nil, fmt.Errorf("script transform not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported transform type: %s", transformType)
	}
}

// toString 转换为字符串
func (c *TypeConverter) toString(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case int, int32, int64:
		return fmt.Sprintf("%d", v), nil
	case float32, float64:
		return fmt.Sprintf("%f", v), nil
	case bool:
		return strconv.FormatBool(v), nil
	case time.Time:
		return v.Format(time.RFC3339), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// toInt 转换为整数
func (c *TypeConverter) toInt(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int", value)
	}
}

// toFloat 转换为浮点数
func (c *TypeConverter) toFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float", value)
	}
}

// toBool 转换为布尔值
func (c *TypeConverter) toBool(value interface{}) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		v = strings.ToLower(strings.TrimSpace(v))
		switch v {
		case "true", "yes", "1", "y", "t":
			return true, nil
		case "false", "no", "0", "n", "f", "":
			return false, nil
		default:
			return false, fmt.Errorf("cannot convert string '%s' to bool", v)
		}
	case int, int32, int64:
		return v != 0, nil
	case float32, float64:
		return v != 0, nil
	default:
		return false, fmt.Errorf("cannot convert %T to bool", value)
	}
}

// toTime 转换为时间
func (c *TypeConverter) toTime(value interface{}, mapping *ent.FieldMapping) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case string:
		// 尝试解析时间格式
		formats := c.getTimeFormats(mapping)
		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("cannot parse time string: %s", v)
	case int64:
		// Unix时间戳（秒）
		return time.Unix(v, 0), nil
	default:
		return time.Time{}, fmt.Errorf("cannot convert %T to time", value)
	}
}

// toJSON 转换为JSON字符串
func (c *TypeConverter) toJSON(value interface{}) (string, error) {
	if str, ok := value.(string); ok {
		// 验证是否已经是有效的JSON
		var js interface{}
		if err := json.Unmarshal([]byte(str), &js); err == nil {
			return str, nil
		}
	}

	// 转换为JSON
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("failed to convert to JSON: %w", err)
	}
	return string(bytes), nil
}

// applyTemplate 模板转换
// 使用 {{field}} 语法引用字段
func (c *TypeConverter) applyTemplate(value interface{}, mapping *ent.FieldMapping) (string, error) {
	if mapping.TransformConfig == "" {
		return "", fmt.Errorf("template config is empty")
	}

	var config TransformConfig
	if err := json.Unmarshal([]byte(mapping.TransformConfig), &config); err != nil {
		return "", fmt.Errorf("failed to parse template config: %w", err)
	}

	if config.Template == "" {
		return "", fmt.Errorf("template is empty")
	}

	// 简单的模板替换（实际项目中可以使用 text/template）
	result := config.Template
	valueStr := fmt.Sprintf("%v", value)
	result = strings.ReplaceAll(result, "{{value}}", valueStr)

	// 如果有额外的参数，也进行替换
	if config.Params != nil {
		for key, val := range config.Params {
			placeholder := fmt.Sprintf("{{%s}}", key)
			result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", val))
		}
	}

	return result, nil
}

// concat 字符串拼接
func (c *TypeConverter) concat(value interface{}, mapping *ent.FieldMapping) (string, error) {
	if mapping.TransformConfig == "" {
		return "", fmt.Errorf("concat config is empty")
	}

	var config TransformConfig
	if err := json.Unmarshal([]byte(mapping.TransformConfig), &config); err != nil {
		return "", fmt.Errorf("failed to parse concat config: %w", err)
	}

	// 获取分隔符
	separator := config.Separator
	if separator == "" {
		separator = ""
	}

	// 获取要拼接的值列表
	parts := make([]string, 0)

	// 添加当前值
	if value != nil {
		parts = append(parts, fmt.Sprintf("%v", value))
	}

	// 添加配置中的其他值
	if config.Params != nil {
		if values, ok := config.Params["values"].([]interface{}); ok {
			for _, v := range values {
				parts = append(parts, fmt.Sprintf("%v", v))
			}
		}
	}

	return strings.Join(parts, separator), nil
}

// split 字符串拆分
func (c *TypeConverter) split(value interface{}, mapping *ent.FieldMapping) ([]string, error) {
	if mapping.TransformConfig == "" {
		return nil, fmt.Errorf("split config is empty")
	}

	var config TransformConfig
	if err := json.Unmarshal([]byte(mapping.TransformConfig), &config); err != nil {
		return nil, fmt.Errorf("failed to parse split config: %w", err)
	}

	// 转换为字符串
	str := fmt.Sprintf("%v", value)

	// 获取分隔符
	separator := config.Separator
	if separator == "" {
		separator = ","
	}

	// 拆分
	parts := strings.Split(str, separator)

	// 去除空白
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result, nil
}

// calculate 数值计算
func (c *TypeConverter) calculate(value interface{}, mapping *ent.FieldMapping) (float64, error) {
	if mapping.TransformConfig == "" {
		return 0, fmt.Errorf("calculate config is empty")
	}

	var config TransformConfig
	if err := json.Unmarshal([]byte(mapping.TransformConfig), &config); err != nil {
		return 0, fmt.Errorf("failed to parse calculate config: %w", err)
	}

	// 转换当前值为数字
	numValue, err := c.toFloat(value)
	if err != nil {
		return 0, fmt.Errorf("cannot convert value to number: %w", err)
	}

	// 执行表达式计算（简单实现）
	// 实际项目中可以使用 github.com/Knetic/govaluate 等表达式引擎
	expression := config.Expression
	if expression == "" {
		return numValue, nil
	}

	// 简单的运算支持
	if strings.HasPrefix(expression, "+") {
		operand, err := strconv.ParseFloat(strings.TrimSpace(expression[1:]), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid expression: %s", expression)
		}
		return numValue + operand, nil
	} else if strings.HasPrefix(expression, "-") {
		operand, err := strconv.ParseFloat(strings.TrimSpace(expression[1:]), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid expression: %s", expression)
		}
		return numValue - operand, nil
	} else if strings.HasPrefix(expression, "*") {
		operand, err := strconv.ParseFloat(strings.TrimSpace(expression[1:]), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid expression: %s", expression)
		}
		return numValue * operand, nil
	} else if strings.HasPrefix(expression, "/") {
		operand, err := strconv.ParseFloat(strings.TrimSpace(expression[1:]), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid expression: %s", expression)
		}
		if operand == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return numValue / operand, nil
	}

	return 0, fmt.Errorf("unsupported expression: %s", expression)
}

// getTimeFormats 获取时间格式列表
func (c *TypeConverter) getTimeFormats(mapping *ent.FieldMapping) []string {
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
		"01/02/2006",
		"01/02/2006 15:04:05",
	}

	// 从配置中获取自定义格式
	if mapping.TransformConfig != "" {
		var config TransformConfig
		if err := json.Unmarshal([]byte(mapping.TransformConfig), &config); err == nil {
			if customFormat, ok := config.Params["time_format"].(string); ok {
				// 将自定义格式放在最前面
				formats = append([]string{customFormat}, formats...)
			}
		}
	}

	return formats
}

// ConvertBatch 批量转换
func (c *TypeConverter) ConvertBatch(values []interface{}, mapping *ent.FieldMapping) ([]interface{}, error) {
	results := make([]interface{}, len(values))
	for i, value := range values {
		converted, err := c.Convert(value, mapping)
		if err != nil {
			return nil, fmt.Errorf("failed to convert value at index %d: %w", i, err)
		}
		results[i] = converted
	}
	return results, nil
}
