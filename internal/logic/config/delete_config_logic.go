package config

import (
	"context"
	"fmt"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteConfigLogic {
	return &DeleteConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteConfigLogic) DeleteConfig(in *io.DeleteConfigReq) (*io.BaseResp, error) {
	if in.ConfigKey == nil || *in.ConfigKey == "" {
		return nil, fmt.Errorf("config_key is required")
	}

	// 获取租户ID
	tenantID := l.ctx.Value("tenantId").(uint64)

	// 使用ConfigCenter.Delete删除配置（会自动记录审计日志）
	err := l.svcCtx.ConfigCenter.Delete(l.ctx, *in.ConfigKey, tenantID)
	if err != nil {
		l.Logger.Errorw("Failed to delete config",
			logx.Field("key", *in.ConfigKey),
			logx.Field("error", err))
		return nil, err
	}

	msg := "Configuration deleted successfully"
	return &io.BaseResp{
		Msg: msg,
	}, nil
}
