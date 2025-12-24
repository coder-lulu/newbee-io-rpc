package crontask

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/suyuan32/simple-admin-common/utils/pointy"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetCronTaskByIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetCronTaskByIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCronTaskByIdLogic {
	return &GetCronTaskByIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetCronTaskByIdLogic) GetCronTaskById(in *io.IDReq) (*io.CronTaskInfo, error) {
	result, err := l.svcCtx.DB.CronTask.Get(l.ctx, in.Id)
	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	return &io.CronTaskInfo{
		Id:          &result.ID,
		CreatedAt:    pointy.GetPointer(result.CreatedAt.UnixMilli()),
		UpdatedAt:    pointy.GetPointer(result.UpdatedAt.UnixMilli()),
		Status:	pointy.GetPointer(uint32(result.Status)),
		TaskName:	&result.TaskName,
		CronExpression:	&result.CronExpression,
		InputSource:	&result.InputSource,
		SourceConfig:	&result.SourceConfig,
		Enabled:	&result.Enabled,
		NextRunTime:	pointy.GetUnixMilliPointer(result.NextRunTime.UnixMilli()),
		LastRunTime:	pointy.GetUnixMilliPointer(result.LastRunTime.UnixMilli()),
		ExecutionCount:	pointy.GetPointer(int64(result.ExecutionCount)),
		SuccessCount:	pointy.GetPointer(int64(result.SuccessCount)),
		FailureCount:	pointy.GetPointer(int64(result.FailureCount)),
		Description:	&result.Description,
	}, nil
}

