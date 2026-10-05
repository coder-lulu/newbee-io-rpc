package crontask

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent/crontask"
	"github.com/coder-lulu/newbee-io-rpc/ent/predicate"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/suyuan32/simple-admin-common/utils/pointy"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetCronTaskListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetCronTaskListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCronTaskListLogic {
	return &GetCronTaskListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetCronTaskListLogic) GetCronTaskList(in *io.CronTaskListReq) (*io.CronTaskListResp, error) {
	var predicates []predicate.CronTask
	if in.CreatedAt != nil {
		predicates = append(predicates, crontask.CreatedAtGTE(time.UnixMilli(*in.CreatedAt)))
	}
	if in.UpdatedAt != nil {
		predicates = append(predicates, crontask.UpdatedAtGTE(time.UnixMilli(*in.UpdatedAt)))
	}
	if in.Status != nil {
		predicates = append(predicates, crontask.StatusEQ(uint8(*in.Status)))
	}
	if in.TaskName != nil {
		predicates = append(predicates, crontask.TaskNameContains(*in.TaskName))
	}
	if in.CronExpression != nil {
		predicates = append(predicates, crontask.CronExpressionContains(*in.CronExpression))
	}
	if in.InputSource != nil {
		predicates = append(predicates, crontask.InputSourceContains(*in.InputSource))
	}
	if in.SourceConfig != nil {
		predicates = append(predicates, crontask.SourceConfigContains(*in.SourceConfig))
	}
	if in.Enabled != nil {
		predicates = append(predicates, crontask.EnabledEQ(*in.Enabled))
	}
	if in.NextRunTime != nil {
		predicates = append(predicates, crontask.NextRunTimeGTE(time.UnixMilli(*in.NextRunTime)))
	}
	if in.LastRunTime != nil {
		predicates = append(predicates, crontask.LastRunTimeGTE(time.UnixMilli(*in.LastRunTime)))
	}
	if in.ExecutionCount != nil {
		predicates = append(predicates, crontask.ExecutionCountEQ(int(*in.ExecutionCount)))
	}
	if in.SuccessCount != nil {
		predicates = append(predicates, crontask.SuccessCountEQ(int(*in.SuccessCount)))
	}
	if in.FailureCount != nil {
		predicates = append(predicates, crontask.FailureCountEQ(int(*in.FailureCount)))
	}
	if in.Description != nil {
		predicates = append(predicates, crontask.DescriptionContains(*in.Description))
	}
	result, err := l.svcCtx.DB.CronTask.Query().Where(predicates...).Page(l.ctx, in.Page, in.PageSize)

	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	resp := &io.CronTaskListResp{}
	resp.Total = result.PageDetails.Total

	for _, v := range result.List {
		resp.Data = append(resp.Data, &io.CronTaskInfo{
			Id:             &v.ID,
			CreatedAt:      pointy.GetPointer(v.CreatedAt.UnixMilli()),
			UpdatedAt:      pointy.GetPointer(v.UpdatedAt.UnixMilli()),
			Status:         pointy.GetPointer(uint32(v.Status)),
			TaskName:       &v.TaskName,
			CronExpression: &v.CronExpression,
			InputSource:    &v.InputSource,
			SourceConfig:   &v.SourceConfig,
			Enabled:        &v.Enabled,
			NextRunTime:    pointy.GetUnixMilliPointer(v.NextRunTime.UnixMilli()),
			LastRunTime:    utils.TimeToUnixMilli(v.LastRunTime),
			ExecutionCount: pointy.GetPointer(int64(v.ExecutionCount)),
			SuccessCount:   pointy.GetPointer(int64(v.SuccessCount)),
			FailureCount:   pointy.GetPointer(int64(v.FailureCount)),
			Description:    &v.Description,
		})
	}

	return resp, nil
}
