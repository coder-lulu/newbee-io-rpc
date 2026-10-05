package config

import (
	"context"
	"fmt"

	"github.com/coder-lulu/newbee-io-rpc/internal/service"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateConfigLogic {
	return &CreateConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateConfigLogic) CreateConfig(in *io.CreateConfigReq) (*io.BaseResp, error) {
	if in.ConfigKey == nil || *in.ConfigKey == "" {
		return nil, fmt.Errorf("config_key is required")
	}
	if in.ConfigValue == nil {
		return nil, fmt.Errorf("config_value is required")
	}

	// 获取租户ID
	tenantID := l.ctx.Value("tenantId").(uint64)

	// 构建配置选项
	opts := &service.ConfigOptions{
		TenantID: tenantID,
	}
	if in.ServiceName != nil {
		opts.ServiceName = *in.ServiceName
	}
	if in.Category != nil {
		opts.Category = *in.Category
	}

	// 使用ConfigCenter.Set创建配置（会自动记录审计日志）
	err := l.svcCtx.ConfigCenter.Set(l.ctx, *in.ConfigKey, *in.ConfigValue, opts)
	if err != nil {
		l.Logger.Errorw("Failed to create config",
			logx.Field("key", *in.ConfigKey),
			logx.Field("error", err))
		return nil, err
	}

	msg := "Configuration created successfully"
	return &io.BaseResp{
		Msg: msg,
	}, nil
}
