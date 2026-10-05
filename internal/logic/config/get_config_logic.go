package config

import (
	"context"
	"fmt"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/ent/configitem"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConfigLogic {
	return &GetConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetConfigLogic) GetConfig(in *io.GetConfigReq) (*io.GetConfigResp, error) {
	if in.ConfigKey == nil || *in.ConfigKey == "" {
		return nil, fmt.Errorf("config_key is required")
	}

	// 获取租户ID
	tenantID := l.ctx.Value("tenantId").(uint64)

	// 查询配置
	cfg, err := l.svcCtx.DB.ConfigItem.Query().
		Where(
			configitem.TenantIDEQ(tenantID),
			configitem.ConfigKeyEQ(*in.ConfigKey),
			configitem.StatusEQ(1),
		).
		First(l.ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("config not found: %s", *in.ConfigKey)
		}
		l.Logger.Errorw("Failed to get config", logx.Field("error", err))
		return nil, err
	}

	// 转换为Proto消息
	version := int64(cfg.Version)
	status := uint32(cfg.Status)
	createdAt := cfg.CreatedAt.Unix()
	updatedAt := cfg.UpdatedAt.Unix()

	item := &io.ConfigItem{
		Id:           &cfg.ID,
		TenantId:     &cfg.TenantID,
		ConfigKey:    &cfg.ConfigKey,
		ConfigValue:  &cfg.ConfigValue,
		ValueType:    &cfg.ValueType,
		Category:     &cfg.Category,
		ServiceName:  &cfg.ServiceName,
		Description:  &cfg.Description,
		DefaultValue: &cfg.DefaultValue,
		Version:      &version,
		Status:       &status,
		IsReadonly:   &cfg.IsReadonly,
		IsSensitive:  &cfg.IsSensitive,
		Scope:        &cfg.Scope,
		ConfigGroup:  &cfg.ConfigGroup,
		CreatedAt:    &createdAt,
		UpdatedAt:    &updatedAt,
	}

	return &io.GetConfigResp{
		Data: item,
	}, nil
}
