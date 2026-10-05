package config

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent"
	"github.com/coder-lulu/newbee-io-rpc/ent/configauditlog"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListAuditLogLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAuditLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAuditLogLogic {
	return &ListAuditLogLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListAuditLogLogic) ListAuditLog(in *io.ListAuditLogReq) (*io.ListAuditLogResp, error) {
	// 获取租户ID
	tenantID, tenantErr := requireTenant(l.ctx)
	if tenantErr != nil {
		return nil, tenantErr
	}

	// 构建查询
	query := l.svcCtx.DB.ConfigAuditLog.Query().
		Where(configauditlog.TenantIDEQ(tenantID))

	// 添加可选过滤条件
	if in.ConfigKey != nil && *in.ConfigKey != "" {
		query = query.Where(configauditlog.ConfigKeyEQ(*in.ConfigKey))
	}
	if in.ChangeType != nil && *in.ChangeType != "" {
		query = query.Where(configauditlog.ChangeTypeEQ(*in.ChangeType))
	}
	if in.StartTime != nil {
		query = query.Where(configauditlog.CreatedAtGTE(timeFromUnix(*in.StartTime)))
	}
	if in.EndTime != nil {
		query = query.Where(configauditlog.CreatedAtLTE(timeFromUnix(*in.EndTime)))
	}

	// 获取总数
	total, err := query.Count(l.ctx)
	if err != nil {
		l.Logger.Errorw("Failed to count audit logs", logx.Field("error", err))
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

	logs, err := query.
		Order(ent.Desc(configauditlog.FieldCreatedAt)).
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		All(l.ctx)
	if err != nil {
		l.Logger.Errorw("Failed to list audit logs", logx.Field("error", err))
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

	totalUint := uint64(total)
	return &io.ListAuditLogResp{
		Total: &totalUint,
		Data:  items,
	}, nil
}

func timeFromUnix(unix int64) time.Time {
	return time.Unix(unix, 0)
}
