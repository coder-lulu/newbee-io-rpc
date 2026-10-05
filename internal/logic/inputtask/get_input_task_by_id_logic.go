package inputtask

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetInputTaskByIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetInputTaskByIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetInputTaskByIdLogic {
	return &GetInputTaskByIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetInputTaskByIdLogic) GetInputTaskById(in *io.IDReq) (*io.InputTaskInfo, error) {
	result, err := l.svcCtx.DB.InputTask.Get(l.ctx, in.Id)
	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	return &io.InputTaskInfo{
		Id:               &result.ID,
		CreatedAt:        utils.GetPointer(result.CreatedAt.UnixMilli()),
		UpdatedAt:        utils.GetPointer(result.UpdatedAt.UnixMilli()),
		Status:           utils.GetPointer(uint32(result.Status)),
		TaskName:         &result.TaskName,
		TaskType:         &result.TaskType,
		InputSource:      &result.InputSource,
		SourceConfig:     &result.SourceConfig,
		TaskStatus:       &result.TaskStatus,
		DiscoveryPoolId:  result.DiscoveryPoolID,
		ScheduledAt:      utils.TimeToUnixMilli(result.ScheduledAt),
		StartedAt:        utils.TimeToUnixMilli(result.StartedAt),
		CompletedAt:      utils.TimeToUnixMilli(result.CompletedAt),
		TotalRecords:     &result.TotalRecords,
		ProcessedRecords: &result.ProcessedRecords,
		SuccessRecords:   &result.SuccessRecords,
		FailedRecords:    &result.FailedRecords,
		ErrorMessage:     &result.ErrorMessage,
		Metadata:         &result.Metadata,
	}, nil
}
