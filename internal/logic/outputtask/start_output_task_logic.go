package outputtask

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/zeromicro/go-zero/core/logx"
)

type StartOutputTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStartOutputTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartOutputTaskLogic {
	return &StartOutputTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// OutputTask lifecycle operations
func (l *StartOutputTaskLogic) StartOutputTask(in *io.IDReq) (*io.BaseResp, error) {
	task, err := l.svcCtx.DB.OutputTask.Get(l.ctx, in.Id)
	if err != nil {
		return &io.BaseResp{Msg: "Task not found"}, err
	}

	if task.TaskStatus == "running" {
		return &io.BaseResp{Msg: "Task already running"}, nil
	}

	if task.TaskStatus != "pending" && task.TaskStatus != "paused" {
		return &io.BaseResp{Msg: "Task cannot be started"}, nil
	}

	now := time.Now()
	err = l.svcCtx.DB.OutputTask.UpdateOne(task).
		SetTaskStatus("running").
		SetStartedAt(now).
		Exec(l.ctx)
	if err != nil {
		return &io.BaseResp{Msg: "Update failed"}, err
	}

	return &io.BaseResp{Msg: "Task started successfully"}, nil
}
