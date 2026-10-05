package config

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/ent/configitem"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListConfigLogic {
	return &ListConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListConfigLogic) ListConfig(in *io.ListConfigReq) (*io.ListConfigResp, error) {
	// 获取租户ID
	tenantID, tenantErr := requireTenant(l.ctx)
	if tenantErr != nil {
		return nil, tenantErr
	}

	// 构建查询
	query := l.svcCtx.DB.ConfigItem.Query().
		Where(configitem.TenantIDEQ(tenantID), configitem.StatusEQ(1))

	// 添加可选过滤条件
	if in.ServiceName != nil && *in.ServiceName != "" {
		query = query.Where(configitem.ServiceNameEQ(*in.ServiceName))
	}
	if in.Category != nil && *in.Category != "" {
		query = query.Where(configitem.CategoryEQ(*in.Category))
	}
	if in.ConfigGroup != nil && *in.ConfigGroup != "" {
		query = query.Where(configitem.ConfigGroupEQ(*in.ConfigGroup))
	}
	if in.Keyword != nil && *in.Keyword != "" {
		query = query.Where(configitem.ConfigKeyContains(*in.Keyword))
	}

	// 获取总数
	total, err := query.Count(l.ctx)
	if err != nil {
		l.Logger.Errorw("Failed to count configs", logx.Field("error", err))
		return nil, err
	}

	// 分页查询
	page := uint64(1)
	pageSize := uint64(10)
	if in.Page != nil {
		page = *in.Page
	}
	if in.PageSize != nil {
		pageSize = *in.PageSize
	}

	configs, err := query.
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		All(l.ctx)
	if err != nil {
		l.Logger.Errorw("Failed to list configs", logx.Field("error", err))
		return nil, err
	}

	// 转换为Proto消息
	items := make([]*io.ConfigItem, 0, len(configs))
	for _, cfg := range configs {
		version := int64(cfg.Version)
		status := uint32(cfg.Status)
		createdAt := cfg.CreatedAt.Unix()
		updatedAt := cfg.UpdatedAt.Unix()

		items = append(items, &io.ConfigItem{
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
		})
	}

	totalUint := uint64(total)
	return &io.ListConfigResp{
		Total: &totalUint,
		Data:  items,
	}, nil
}
