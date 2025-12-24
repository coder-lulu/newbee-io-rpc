package validator

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
	"github.com/zeromicro/go-zero/core/errorx"
)

var (
	validate *validator.Validate
	trans    ut.Translator
)

// 初始化验证器
func init() {
	// 创建验证器
	validate = validator.New()

	// 注册自定义的标签处理函数，用于从json标签获取字段名
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return fld.Name
		}
		return name
	})

	// 设置中文翻译器
	zhCn := zh.New()
	uni := ut.New(zhCn, zhCn)
	trans, _ = uni.GetTranslator("zh")
	zh_translations.RegisterDefaultTranslations(validate, trans)

	// 注册自定义验证规则
	registerCustomValidations()
}

// registerCustomValidations 注册自定义验证规则
func registerCustomValidations() {
	// 注册手机号验证
	_ = validate.RegisterValidation("mobile", validateMobile)
	_ = validate.RegisterTranslation("mobile", trans, func(ut ut.Translator) error {
		return ut.Add("mobile", "{0}必须是有效的手机号码", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("mobile", fe.Field())
		return t
	})

	// 注册字母数字下划线验证
	_ = validate.RegisterValidation("alphanumdash", validateAlphaNumDash)
	_ = validate.RegisterTranslation("alphanumdash", trans, func(ut ut.Translator) error {
		return ut.Add("alphanumdash", "{0}只能包含字母、数字和下划线", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("alphanumdash", fe.Field())
		return t
	})
}

// validateMobile 验证手机号
func validateMobile(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	pattern := `^1[3-9]\d{9}$`
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(value)
}

// validateAlphaNumDash 验证字母数字下划线
func validateAlphaNumDash(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	pattern := `^[a-zA-Z0-9_]+$`
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(value)
}

// Validate 验证结构体
func Validate(s interface{}) error {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		return errorx.NewInvalidArgumentError("验证错误")
	}

	errMsgs := make([]string, 0, len(errs))
	for _, e := range errs {
		errMsgs = append(errMsgs, e.Translate(trans))
	}

	return errorx.NewInvalidArgumentError("参数验证失败: " + strings.Join(errMsgs, "; "))
}

// ValidateVar 验证单个变量
func ValidateVar(field interface{}, tag string) error {
	err := validate.Var(field, tag)
	if err == nil {
		return nil
	}

	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		return errorx.NewInvalidArgumentError("验证错误")
	}

	errMsgs := make([]string, 0, len(errs))
	for _, e := range errs {
		errMsgs = append(errMsgs, e.Translate(trans))
	}

	return errorx.NewInvalidArgumentError("字段验证失败: " + strings.Join(errMsgs, "; "))
}

// RegisterValidation 注册自定义验证规则
func RegisterValidation(tag string, fn validator.Func, errMsg string) error {
	err := validate.RegisterValidation(tag, fn)
	if err != nil {
		return err
	}

	// 注册翻译
	_ = validate.RegisterTranslation(tag, trans, func(ut ut.Translator) error {
		return ut.Add(tag, "{0}"+errMsg, true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T(tag, fe.Field())
		return t
	})

	return nil
}

// HTTPValidator 用于HTTP请求的验证器，集成了go-zero的httpx
func HTTPValidator(r interface{}, v interface{}) error {
	// 暂时简化实现，只进行验证
	return Validate(v)
}

// GetValidate 获取验证器实例，用于高级自定义
func GetValidate() *validator.Validate {
	return validate
}

// GetTranslator 获取翻译器实例，用于高级自定义
func GetTranslator() ut.Translator {
	return trans
}

// FormatError 格式化错误信息
func FormatError(err error) string {
	if err == nil {
		return ""
	}

	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		return err.Error()
	}

	var errMsgs []string
	for _, e := range errs {
		errMsgs = append(errMsgs, fmt.Sprintf("%s", e.Translate(trans)))
	}

	return strings.Join(errMsgs, "; ")
}
