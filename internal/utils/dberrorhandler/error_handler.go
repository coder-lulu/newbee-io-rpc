package dberrorhandler

import (
	"context"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/coder-lulu/newbee-common/v2/msg/logmsg"
	"github.com/coder-lulu/newbee-common/v2/errors"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-common/v2/i18n"
)

// DefaultEntError returns errors dealing with default functions.
func DefaultEntError(logger logx.Logger, err error, detail any) error {
	if err != nil {
		// 首先检查是否是UUID空值错误
		if isUUIDNullError(err) {
			return handleUUIDNullError(logger, err, detail)
		}

		// 检查是否是唯一性校验错误
		if isUniqueValidationError(err) {
			return handleUniqueValidationError(logger, err, detail)
		}

		switch {
		case ent.IsNotFound(err):
			logger.Errorw(err.Error(), logx.Field("detail", detail))
			return errors.NotFound(i18n.TargetNotFound)
		case ent.IsConstraintError(err):
			logger.Errorw(err.Error(), logx.Field("detail", detail))
			return errors.Validation(i18n.ConstraintError)
		case ent.IsValidationError(err):
			logger.Errorw(err.Error(), logx.Field("detail", detail))
			return errors.Validation(i18n.ValidationError)
		case ent.IsNotSingular(err):
			logger.Errorw(err.Error(), logx.Field("detail", detail))
			return errors.Validation(i18n.NotSingularError)
		default:
			logger.Errorw(logmsg.DatabaseError, logx.Field("detail", err.Error()))
			return errors.DatabaseWithCause(i18n.DatabaseError, err)
		}
	}
	return err
}

// DefaultEntErrorWithContext returns errors dealing with default functions with context.
func DefaultEntErrorWithContext(ctx context.Context, logger logx.Logger, err error, detail any) error {
	if err != nil {
		// 首先检查是否是UUID空值错误
		if isUUIDNullError(err) {
			return handleUUIDNullErrorWithContext(ctx, logger, err, detail)
		}

		// 检查是否是唯一性校验错误
		if isUniqueValidationError(err) {
			return handleUniqueValidationErrorWithContext(ctx, logger, err, detail)
		}

		switch {
		case ent.IsNotFound(err):
			logger.Errorw(err.Error(), logx.Field("detail", detail))
			return errors.NotFound(i18n.TargetNotFound)
		case ent.IsConstraintError(err):
			logger.Errorw(err.Error(), logx.Field("detail", detail))
			return errors.Validation(i18n.ConstraintError)
		case ent.IsValidationError(err):
			logger.Errorw(err.Error(), logx.Field("detail", detail))
			return errors.Validation(i18n.ValidationError)
		case ent.IsNotSingular(err):
			logger.Errorw(err.Error(), logx.Field("detail", detail))
			return errors.Validation(i18n.NotSingularError)
		default:
			logger.Errorw(logmsg.DatabaseError, logx.Field("detail", err.Error()))
			return errors.DatabaseWithCause(i18n.DatabaseError, err)
		}
	}
	return err
}

// isUUIDNullError 检查是否是UUID空值错误
func isUUIDNullError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "uuid: cannot convert <nil> to UUID")
}

// handleUUIDNullError 处理UUID空值错误
func handleUUIDNullError(logger logx.Logger, err error, detail any) error {
	errMsg := err.Error()
	fieldName := extractFieldNameFromError(errMsg)

	logger.Errorw("UUID null value error",
		logx.Field("field", fieldName),
		logx.Field("detail", detail),
		logx.Field("error", err.Error()))

	// 如果是updated_by字段，提供特殊处理
	if fieldName == "updated_by" {
		logger.Errorw("updated_by field contains null value, this should be fixed by regenerating ent code with .Nillable()",
			logx.Field("suggestion", "regenerate ent code"))
		return errors.Internal("数据完整性问题：updated_by字段包含空值，请联系管理员修复")
	}

	return errors.Internal("数据完整性问题：UUID字段包含空值")
}

// handleUUIDNullErrorWithContext 处理UUID空值错误（带上下文）
func handleUUIDNullErrorWithContext(ctx context.Context, logger logx.Logger, err error, detail any) error {
	errMsg := err.Error()
	fieldName := extractFieldNameFromError(errMsg)

	logx.WithContext(ctx).Errorw("UUID null value error",
		logx.Field("field", fieldName),
		logx.Field("detail", detail),
		logx.Field("error", err.Error()))

	// 如果是updated_by字段，提供特殊处理
	if fieldName == "updated_by" {
		logx.WithContext(ctx).Errorw("updated_by field contains null value, this should be fixed by regenerating ent code with .Nillable()",
			logx.Field("suggestion", "regenerate ent code"))
		return errors.Internal("数据完整性问题：updated_by字段包含空值，请联系管理员修复")
	}

	return errors.Internal("数据完整性问题：UUID字段包含空值")
}

// extractFieldNameFromError 从错误信息中提取字段名
func extractFieldNameFromError(errMsg string) string {
	// 错误格式通常是: sql: Scan error on column index 8, name "updated_by": uuid: cannot convert <nil> to UUID
	if strings.Contains(errMsg, `name "`) {
		start := strings.Index(errMsg, `name "`) + 6
		end := strings.Index(errMsg[start:], `"`)
		if end > 0 {
			return errMsg[start : start+end]
		}
	}
	return "unknown"
}

// isUniqueValidationError 检查是否是唯一性校验错误
func isUniqueValidationError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "唯一性校验失败") ||
		strings.Contains(errMsg, "在当前CI类型下已存在")
}

// handleUniqueValidationError 处理唯一性校验错误
func handleUniqueValidationError(logger logx.Logger, err error, detail any) error {
	logger.Errorw("Unique validation error",
		logx.Field("detail", detail),
		logx.Field("error", err.Error()))

	// 返回更友好的错误信息
	return errors.Validation(err.Error())
}

// handleUniqueValidationErrorWithContext 处理唯一性校验错误（带上下文）
func handleUniqueValidationErrorWithContext(ctx context.Context, logger logx.Logger, err error, detail any) error {
	logx.WithContext(ctx).Errorw("Unique validation error",
		logx.Field("detail", detail),
		logx.Field("error", err.Error()))

	// 返回更友好的错误信息
	return errors.Validation(err.Error())
}
