package outputtask

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/zeromicro/go-zero/core/logx"
)

type PauseOutputTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPauseOutputTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PauseOutputTaskLogic {
	return &PauseOutputTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PauseOutputTaskLogic) PauseOutputTask(in *io.IDReq) (*io.BaseResp, error) {
	task, err := l.svcCtx.DB.OutputTask.Get(l.ctx, in.Id)
	if err != nil {
		return &io.BaseResp{Msg: "Task not found"}, err
	}

	if task.TaskStatus != "running" {
		return &io.BaseResp{Msg: "Task is not running"}, nil
	}

	err = l.svcCtx.DB.OutputTask.UpdateOne(task).
		SetTaskStatus("paused").
		Exec(l.ctx)
	if err != nil {
		return &io.BaseResp{Msg: "Update failed"}, err
	}

	return &io.BaseResp{Msg: "Task paused successfully"}, nil
}
