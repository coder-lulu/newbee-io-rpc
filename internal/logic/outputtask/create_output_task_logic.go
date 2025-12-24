package outputtask

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

    "github.com/coder-lulu/newbee-common/v2/msg/errormsg"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateOutputTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateOutputTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateOutputTaskLogic {
	return &CreateOutputTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateOutputTaskLogic) CreateOutputTask(in *io.OutputTaskInfo) (*io.BaseIDResp, error) {
    query := l.svcCtx.DB.OutputTask.Create().
			SetNotNilTaskName(in.TaskName).
			SetNotNilTaskType(in.TaskType).
			SetNotNilOutputTarget(in.OutputTarget).
			SetNotNilTargetConfig(in.TargetConfig).
			SetNotNilTaskStatus(in.TaskStatus).
			SetNotNilDataTargetID(in.DataTargetId).
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

	result, err := query.Save(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseIDResp{Id: result.ID, Msg: errormsg.CreateSuccess }, nil
}
