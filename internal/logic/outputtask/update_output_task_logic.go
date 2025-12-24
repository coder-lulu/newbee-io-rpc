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

type UpdateOutputTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateOutputTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateOutputTaskLogic {
	return &UpdateOutputTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateOutputTaskLogic) UpdateOutputTask(in *io.OutputTaskInfo) (*io.BaseResp, error) {
	query:= l.svcCtx.DB.OutputTask.UpdateOneID(*in.Id).
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

	 err := query.Exec(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseResp{Msg: errormsg.UpdateSuccess }, nil
}
