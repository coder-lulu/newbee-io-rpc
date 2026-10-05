package config

import (
	"context"
	"fmt"

	"github.com/coder-lulu/newbee-io-rpc/internal/service"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateConfigLogic {
	return &UpdateConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateConfigLogic) UpdateConfig(in *io.UpdateConfigReq) (*io.BaseResp, error) {
	if in.ConfigKey == nil || *in.ConfigKey == "" {
		return nil, fmt.Errorf("config_key is required")
	}
	if in.ConfigValue == nil {
		return nil, fmt.Errorf("config_value is required")
	}

	// 获取租户ID
	tenantID, tenantErr := requireTenant(l.ctx)
	if tenantErr != nil {
		return nil, tenantErr
	}

	// 构建配置选项
	opts := &service.ConfigOptions{
		TenantID: tenantID,
	}

	// 使用ConfigCenter.Set更新配置（会自动记录审计日志）
	err := l.svcCtx.ConfigCenter.Set(l.ctx, *in.ConfigKey, *in.ConfigValue, opts)
	if err != nil {
		l.Logger.Errorw("Failed to update config",
			logx.Field("key", *in.ConfigKey),
			logx.Field("error", err))
		return nil, err
	}

	msg := "Configuration updated successfully"
	return &io.BaseResp{
		Msg: msg,
	}, nil
}
