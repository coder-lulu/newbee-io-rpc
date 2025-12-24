package inputtask

import (
	"context"
	"errors"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

    "github.com/coder-lulu/newbee-common/v2/msg/errormsg"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateInputTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateInputTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateInputTaskLogic {
	return &CreateInputTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateInputTaskLogic) CreateInputTask(in *io.InputTaskInfo) (*io.BaseIDResp, error) {
	// ⭐ 验证定时任务参数
	if in.TaskType != nil && *in.TaskType == "scheduled" {
		if in.ScheduledAt == nil {
			return nil, errors.New("scheduled任务必须提供ScheduledAt时间")
		}

		scheduledTime := time.UnixMilli(*in.ScheduledAt)
		if scheduledTime.Before(time.Now()) {
			return nil, errors.New("ScheduledAt不能是过去的时间")
		}

		// 记录定时任务创建日志
		taskName := ""
		if in.TaskName != nil {
			taskName = *in.TaskName
		}
		logx.Infow("Creating scheduled task",
			logx.Field("task_name", taskName),
			logx.Field("scheduled_at", scheduledTime.Format(time.RFC3339)))
	}

    query := l.svcCtx.DB.InputTask.Create().
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

	result, err := query.Save(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseIDResp{Id: result.ID, Msg: errormsg.CreateSuccess }, nil
}
