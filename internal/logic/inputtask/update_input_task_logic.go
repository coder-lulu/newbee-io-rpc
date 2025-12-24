package inputtask

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

    "github.com/coder-lulu/newbee-common/v2/msg/errormsg"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateInputTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateInputTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateInputTaskLogic {
	return &UpdateInputTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateInputTaskLogic) UpdateInputTask(in *io.InputTaskInfo) (*io.BaseResp, error) {
	query:= l.svcCtx.DB.InputTask.UpdateOneID(*in.Id).
			SetNotNilTaskName(in.TaskName).
			SetNotNilTaskType(in.TaskType).
			SetNotNilInputSource(in.InputSource).
			SetNotNilSourceConfig(in.SourceConfig).
			SetNotNilTaskStatus(in.TaskStatus).
			SetNotNilDiscoveryPoolID(in.DiscoveryPoolId).
			SetNotNilScheduledAt(utils.GetTimeMilliPointer(in.ScheduledAt)).
			SetNotNilStartedAt(utils.GetTimeMilliPointer(in.StartedAt)).
			SetNotNilCompletedAt(utils.GetTimeMilliPointer(in.CompletedAt)).
			SetNotNilTotalRecords(in.TotalRecords).
			SetNotNilProcessedRecords(in.ProcessedRecords).
			SetNotNilSuccessRecords(in.SuccessRecords).
			SetNotNilFailedRecords(in.FailedRecords).
			SetNotNilErrorMessage(in.ErrorMessage).
			SetNotNilMetadata(in.Metadata)

	if in.Status != nil {
		query.SetNotNilStatus(utils.GetPointer(uint8(*in.Status)))
	}

	 err := query.Exec(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseResp{Msg: errormsg.UpdateSuccess }, nil
}
