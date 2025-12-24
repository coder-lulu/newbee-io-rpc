package tasklog

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

    "github.com/coder-lulu/newbee-common/v2/msg/errormsg"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateTaskLogLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateTaskLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateTaskLogLogic {
	return &CreateTaskLogLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateTaskLogLogic) CreateTaskLog(in *io.TaskLogInfo) (*io.BaseIDResp, error) {
    result, err := l.svcCtx.DB.TaskLog.Create().
			SetNotNilTaskType(in.TaskType).
			SetNotNilTaskID(in.TaskId).
			SetNotNilLogLevel(in.LogLevel).
			SetNotNilLogMessage(in.LogMessage).
			SetNotNilLogDetail(in.LogDetail).
			SetNotNilLoggedAt(utils.GetTimeMilliPointer(in.LoggedAt)).
			Save(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseIDResp{Id: result.ID, Msg: errormsg.CreateSuccess }, nil
}
