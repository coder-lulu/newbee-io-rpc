package config

import (
	"context"
	"fmt"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/internal/service"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"
	"github.com/zeromicro/go-zero/core/logx"
)

type RollbackConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRollbackConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RollbackConfigLogic {
	return &RollbackConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RollbackConfigLogic) RollbackConfig(in *io.RollbackConfigReq) (*io.BaseResp, error) {
	if in.AuditLogId == nil {
		return nil, fmt.Errorf("audit_log_id is required")
	}

	// 获取租户ID
	tenantID := l.ctx.Value("tenantId").(uint64)

	// 查询审计日志
	auditLog, err := l.svcCtx.DB.ConfigAuditLog.Get(l.ctx, *in.AuditLogId)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("audit log not found: %d", *in.AuditLogId)
		}
		l.Logger.Errorw("Failed to get audit log",
			logx.Field("id", *in.AuditLogId),
			logx.Field("error", err))
		return nil, err
	}

	// 验证租户ID
	if auditLog.TenantID != tenantID {
		return nil, fmt.Errorf("permission denied: audit log belongs to different tenant")
	}

	// 回滚到旧值
	rollbackValue := auditLog.OldValue
	if rollbackValue == "" {
		return nil, fmt.Errorf("cannot rollback: old value is empty (this was a create operation)")
	}

	// 构建配置选项
	opts := &service.ConfigOptions{
		TenantID:    tenantID,
		ServiceName: auditLog.ServiceName,
		Category:    auditLog.Category,
	}

	// 使用ConfigCenter.Set回滚配置
	err = l.svcCtx.ConfigCenter.Set(l.ctx, auditLog.ConfigKey, rollbackValue, opts)
	if err != nil {
		l.Logger.Errorw("Failed to rollback config",
			logx.Field("key", auditLog.ConfigKey),
			logx.Field("error", err))
		return nil, err
	}

	msg := fmt.Sprintf("Configuration '%s' rolled back successfully to version %d",
		auditLog.ConfigKey, auditLog.OldVersion)
	return &io.BaseResp{
		Msg: msg,
	}, nil
}
