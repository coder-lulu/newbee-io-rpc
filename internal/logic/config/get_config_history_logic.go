package config

import (
	"context"
	"fmt"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/ent/configauditlog"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetConfigHistoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetConfigHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConfigHistoryLogic {
	return &GetConfigHistoryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetConfigHistoryLogic) GetConfigHistory(in *io.GetConfigHistoryReq) (*io.GetConfigHistoryResp, error) {
	if in.ConfigKey == nil || *in.ConfigKey == "" {
		return nil, fmt.Errorf("config_key is required")
	}

	// 获取租户ID
	tenantID, tenantErr := requireTenant(l.ctx)
	if tenantErr != nil {
		return nil, tenantErr
	}

	// 查询配置历史（按时间正序）
	logs, err := l.svcCtx.DB.ConfigAuditLog.Query().
		Where(
			configauditlog.TenantIDEQ(tenantID),
			configauditlog.ConfigKeyEQ(*in.ConfigKey),
		).
		Order(ent.Asc(configauditlog.FieldCreatedAt)).
		All(l.ctx)

	if err != nil {
		l.Logger.Errorw("Failed to get config history",
			logx.Field("config_key", *in.ConfigKey),
			logx.Field("error", err))
		return nil, err
	}

	// 转换为Proto消息
	items := make([]*io.ConfigAuditLog, 0, len(logs))
	for _, log := range logs {
		oldVersion := int64(log.OldVersion)
		newVersion := int64(log.NewVersion)
		createdAt := log.CreatedAt.Unix()

		items = append(items, &io.ConfigAuditLog{
			Id:                &log.ID,
			TenantId:          &log.TenantID,
			ConfigKey:         &log.ConfigKey,
			OldValue:          &log.OldValue,
			NewValue:          &log.NewValue,
			ChangeType:        &log.ChangeType,
			ChangedBy:         &log.ChangedBy,
			ChangedByName:     &log.ChangedByName,
			ServiceName:       &log.ServiceName,
			Category:          &log.Category,
			ConfigGroup:       &log.ConfigGroup,
			ChangeReason:      &log.ChangeReason,
			IpAddress:         &log.IPAddress,
			UserAgent:         &log.UserAgent,
			OldVersion:        &oldVersion,
			NewVersion:        &newVersion,
			IsRollback:        &log.IsRollback,
			RollbackFromLogId: &log.RollbackFromLogID,
			CreatedAt:         &createdAt,
		})
	}

	return &io.GetConfigHistoryResp{
		Data: items,
	}, nil
}
