package outputtask

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetOutputTaskByIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetOutputTaskByIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOutputTaskByIdLogic {
	return &GetOutputTaskByIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetOutputTaskByIdLogic) GetOutputTaskById(in *io.IDReq) (*io.OutputTaskInfo, error) {
	result, err := l.svcCtx.DB.OutputTask.Get(l.ctx, in.Id)
	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	return &io.OutputTaskInfo{
		Id:          &result.ID,
		CreatedAt:    utils.GetPointer(result.CreatedAt.UnixMilli()),
		UpdatedAt:    utils.GetPointer(result.UpdatedAt.UnixMilli()),
		Status:	utils.GetPointer(uint32(result.Status)),
		TaskName:	&result.TaskName,
		TaskType:	&result.TaskType,
		OutputTarget:	&result.OutputTarget,
		TargetConfig:	&result.TargetConfig,
		TaskStatus:	&result.TaskStatus,
		DataTargetId:	result.DataTargetID,
		ScheduledAt:	utils.GetUnixMilliPointer(result.ScheduledAt.UnixMilli()),
		StartedAt:	utils.GetUnixMilliPointer(result.StartedAt.UnixMilli()),
		CompletedAt:	utils.GetUnixMilliPointer(result.CompletedAt.UnixMilli()),
		TotalRecords:	&result.TotalRecords,
		ProcessedRecords:	&result.ProcessedRecords,
		SuccessRecords:	&result.SuccessRecords,
		FailedRecords:	&result.FailedRecords,
		ErrorMessage:	&result.ErrorMessage,
		Metadata:	&result.Metadata,
	}, nil
}

