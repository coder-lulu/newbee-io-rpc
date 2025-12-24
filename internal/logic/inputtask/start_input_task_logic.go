package inputtask

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/zeromicro/go-zero/core/logx"
)

type StartInputTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStartInputTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartInputTaskLogic {
	return &StartInputTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *StartInputTaskLogic) StartInputTask(in *io.IDReq) (*io.BaseResp, error) {
	task, err := l.svcCtx.DB.InputTask.Get(l.ctx, in.Id)
	if err != nil {
		return &io.BaseResp{Msg: "Task not found"}, err
	}

	if task.TaskStatus != "approved" && task.TaskStatus != "paused" {
		return &io.BaseResp{Msg: "Task not approved or paused"}, nil
	}

	now := time.Now()
	err = l.svcCtx.DB.InputTask.UpdateOne(task).
		SetTaskStatus("running").
		SetStartedAt(now).
		Exec(l.ctx)
	if err != nil {
		return &io.BaseResp{Msg: "Update failed"}, err
	}

	return &io.BaseResp{Msg: "Task started successfully"}, nil
}
