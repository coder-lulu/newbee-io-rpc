package inputtask

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelInputTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCancelInputTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelInputTaskLogic {
	return &CancelInputTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CancelInputTaskLogic) CancelInputTask(in *io.IDReq) (*io.BaseResp, error) {
	task, err := l.svcCtx.DB.InputTask.Get(l.ctx, in.Id)
	if err != nil {
		return &io.BaseResp{Msg: "Task not found"}, err
	}

	if task.TaskStatus == "completed" || task.TaskStatus == "cancelled" {
		return &io.BaseResp{Msg: "Task already finished"}, nil
	}

	now := time.Now()
	err = l.svcCtx.DB.InputTask.UpdateOne(task).
		SetTaskStatus("cancelled").
		SetCompletedAt(now).
		Exec(l.ctx)
	if err != nil {
		return &io.BaseResp{Msg: "Update failed"}, err
	}

	return &io.BaseResp{Msg: "Task cancelled successfully"}, nil
}
