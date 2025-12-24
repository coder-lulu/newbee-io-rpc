package inputtask

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/zeromicro/go-zero/core/logx"
)

type PauseInputTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPauseInputTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PauseInputTaskLogic {
	return &PauseInputTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PauseInputTaskLogic) PauseInputTask(in *io.IDReq) (*io.BaseResp, error) {
	task, err := l.svcCtx.DB.InputTask.Get(l.ctx, in.Id)
	if err != nil {
		return &io.BaseResp{Msg: "Task not found"}, err
	}

	if task.TaskStatus != "running" {
		return &io.BaseResp{Msg: "Task is not running"}, nil
	}

	err = l.svcCtx.DB.InputTask.UpdateOne(task).
		SetTaskStatus("paused").
		Exec(l.ctx)
	if err != nil {
		return &io.BaseResp{Msg: "Update failed"}, err
	}

	return &io.BaseResp{Msg: "Task paused successfully"}, nil
}
