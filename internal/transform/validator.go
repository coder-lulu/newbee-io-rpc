package transform

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/zeromicro/go-zero/core/logx"
)

// Validator 数据验证器
// 职责:
// 1. 解析validation_rules JSON配置
// 2. 执行各种验证规则（required, regex, range, enum, length）
// 3. 返回详细的验证错误信息
type Validator struct {
	logger logx.Logger
}

// NewValidator 创建验证器
func NewValidator(logger logx.Logger) *Validator {
	return &Validator{
		logger: logger,
	}
}

// Validate 验证字段值
// 根据FieldMapping的validation_rules执行验证
func (v *Validator) Validate(value interface{}, mapping *ent.FieldMapping) error {
	// 如果没有验证规则，直接通过
	if mapping.ValidationRules == "" {
		return nil
	}

	// 解析验证规则
	var rules []ValidationRule
	if err := json.Unmarshal([]byte(mapping.ValidationRules), &rules); err != nil {
		return fmt.Errorf("failed to parse validation rules: %w", err)
	}

	// 执行每个验证规则
	for _, rule := range rules {
		if err := v.validateRule(value, rule); err != nil {
			// 如果规则有自定义错误消息，使用它
			if rule.Message != "" {
				return fmt.Errorf("%s", rule.Message)
			}
			return err
		}
	}

	return nil
}

// validateRule 验证单个规则
func (v *Validator) validateRule(value interface{}, rule ValidationRule) error {
	v.logger.Debugw("Validating rule",
		logx.Field("rule_type", rule.Type),
		logx.Field("value", value))

	switch rule.Type {
	case "required":
		return v.validateRequired(value)
	case "regex":
		return v.validateRegex(value, rule)
	case "range":
		return v.validateRange(value, rule)
	case "enum":
		return v.validateEnum(value, rule)
	case "length":
		return v.validateLength(value, rule)
	case "email":
		return v.validateEmail(value)
	case "url":
		return v.validateURL(value)
	case "custom":
		return v.validateCustom(value, rule)
	default:
		return fmt.Errorf("unsupported validation type: %s", rule.Type)
	}
}

// validateRequired 验证必填
func (v *Validator) validateRequired(value interface{}) error {
	if value == nil {
		return fmt.Errorf("value is required")
	}

	// 检查空字符串
	if str, ok := value.(string); ok {
		if strings.TrimSpace(str) == "" {
			return fmt.Errorf("value cannot be empty")
		}
	}

	// 检查空数组
	if arr, ok := value.([]interface{}); ok {
		if len(arr) == 0 {
			return fmt.Errorf("array cannot be empty")
		}
	}

	return nil
}

// validateRegex 正则表达式验证
func (v *Validator) validateRegex(value interface{}, rule ValidationRule) error {
	// 转换为字符串
	str := fmt.Sprintf("%v", value)

	// 获取正则表达式
	pattern, ok := rule.Params.(string)
	if !ok {
		if params, ok := rule.Params.(map[string]interface{}); ok {
			pattern, _ = params["pattern"].(string)
		}
	}

	if pattern == "" {
		return fmt.Errorf("regex pattern is empty")
	}

	// 编译和匹配
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}

	if !re.MatchString(str) {
		return fmt.Errorf("value does not match pattern: %s", pattern)
	}

	return nil
}

// validateRange 范围验证
func (v *Validator) validateRange(value interface{}, rule ValidationRule) error {
	// 转换为数字
	var numValue float64
	switch v := value.(type) {
	case int:
		numValue = float64(v)
	case int32:
		numValue = float64(v)
	case int64:
		numValue = float64(v)
	case float32:
		numValue = float64(v)
	case float64:
		numValue = v
	default:
		return fmt.Errorf("value is not a number")
	}

	// 获取范围参数
	params, ok := rule.Params.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid range params")
	}

	// 检查最小值
	if min, ok := params["min"].(float64); ok {
		if numValue < min {
			return fmt.Errorf("value %v is less than minimum %v", numValue, min)
		}
	}

	// 检查最大值
	if max, ok := params["max"].(float64); ok {
		if numValue > max {
			return fmt.Errorf("value %v is greater than maximum %v", numValue, max)
		}
	}

	return nil
}

// validateEnum 枚举值验证
func (v *Validator) validateEnum(value interface{}, rule ValidationRule) error {
	valueStr := fmt.Sprintf("%v", value)

	// 获取允许的值列表
	var allowedValues []string

	switch params := rule.Params.(type) {
	case []interface{}:
		for _, val := range params {
			allowedValues = append(allowedValues, fmt.Sprintf("%v", val))
		}
	case map[string]interface{}:
		if values, ok := params["values"].([]interface{}); ok {
			for _, val := range values {
				allowedValues = append(allowedValues, fmt.Sprintf("%v", val))
			}
		}
	default:
		return fmt.Errorf("invalid enum params")
	}

	if len(allowedValues) == 0 {
		return fmt.Errorf("enum values list is empty")
	}

	// 检查值是否在允许列表中
	for _, allowed := range allowedValues {
		if valueStr == allowed {
			return nil
		}
	}

	return fmt.Errorf("value '%s' is not in allowed values: %v", valueStr, allowedValues)
}

// validateLength 长度验证
func (v *Validator) validateLength(value interface{}, rule ValidationRule) error {
	var length int

	switch v := value.(type) {
	case string:
		length = len(v)
	case []interface{}:
		length = len(v)
	default:
		return fmt.Errorf("value does not have length")
	}

	// 获取长度参数
	params, ok := rule.Params.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid length params")
	}

	// 检查最小长度
	if min, ok := params["min"].(float64); ok {
		if length < int(min) {
			return fmt.Errorf("length %d is less than minimum %d", length, int(min))
		}
	}

	// 检查最大长度
	if max, ok := params["max"].(float64); ok {
		if length > int(max) {
			return fmt.Errorf("length %d is greater than maximum %d", length, int(max))
		}
	}

	// 检查精确长度
	if exact, ok := params["exact"].(float64); ok {
		if length != int(exact) {
			return fmt.Errorf("length %d does not equal required length %d", length, int(exact))
		}
	}

	return nil
}

// validateEmail 邮箱验证
func (v *Validator) validateEmail(value interface{}) error {
	str := fmt.Sprintf("%v", value)

	// 简单的邮箱正则
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(emailRegex)

	if !re.MatchString(str) {
		return fmt.Errorf("invalid email format: %s", str)
	}

	return nil
}

// validateURL URL验证
func (v *Validator) validateURL(value interface{}) error {
	str := fmt.Sprintf("%v", value)

	// 简单的URL正则
	urlRegex := `^https?://[a-zA-Z0-9\-._~:/?#\[\]@!$&'()*+,;=]+$`
	re := regexp.MustCompile(urlRegex)

	if !re.MatchString(str) {
		return fmt.Errorf("invalid URL format: %s", str)
	}

	return nil
}

// validateCustom 自定义验证
func (v *Validator) validateCustom(value interface{}, rule ValidationRule) error {
	// 自定义验证需要通过脚本或表达式引擎实现
	// 这里暂时返回未实现错误
	return fmt.Errorf("custom validation not implemented yet")
}

// ValidateBatch 批量验证
func (v *Validator) ValidateBatch(values []interface{}, mapping *ent.FieldMapping) error {
	for i, value := range values {
		if err := v.Validate(value, mapping); err != nil {
			return fmt.Errorf("validation failed at index %d: %w", i, err)
		}
	}
	return nil
}

// ValidateMultipleRules 验证多个规则（用于自定义场景）
func (v *Validator) ValidateMultipleRules(value interface{}, rules []ValidationRule) error {
	for _, rule := range rules {
		if err := v.validateRule(value, rule); err != nil {
			if rule.Message != "" {
				return fmt.Errorf("%s", rule.Message)
			}
			return err
		}
	}
	return nil
}

// BuildRuleFromConfig 从配置构建验证规则（工具方法）
func BuildRuleFromConfig(ruleType string, params interface{}, message string) ValidationRule {
	return ValidationRule{
		Type:    ruleType,
		Params:  params,
		Message: message,
	}
}

// CommonRules 常用验证规则（工具方法）
var CommonRules = struct {
	Required      ValidationRule
	Email         ValidationRule
	URL           ValidationRule
	PositiveInt   ValidationRule
	NonEmptyString ValidationRule
}{
	Required: ValidationRule{
		Type:    "required",
		Message: "This field is required",
	},
	Email: ValidationRule{
		Type:    "email",
		Message: "Invalid email format",
	},
	URL: ValidationRule{
		Type:    "url",
		Message: "Invalid URL format",
	},
	PositiveInt: ValidationRule{
		Type:    "range",
		Params:  map[string]interface{}{"min": 0.0},
		Message: "Value must be a positive integer",
	},
	NonEmptyString: ValidationRule{
		Type:    "length",
		Params:  map[string]interface{}{"min": 1.0},
		Message: "String cannot be empty",
	},
}
