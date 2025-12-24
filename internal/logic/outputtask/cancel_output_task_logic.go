package outputtask

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelOutputTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCancelOutputTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelOutputTaskLogic {
	return &CancelOutputTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CancelOutputTaskLogic) CancelOutputTask(in *io.IDReq) (*io.BaseResp, error) {
	task, err := l.svcCtx.DB.OutputTask.Get(l.ctx, in.Id)
	if err != nil {
		return &io.BaseResp{Msg: "Task not found"}, err
	}

	if task.TaskStatus == "completed" || task.TaskStatus == "cancelled" {
		return &io.BaseResp{Msg: "Task already finished"}, nil
	}

	now := time.Now()
	err = l.svcCtx.DB.OutputTask.UpdateOne(task).
		SetTaskStatus("cancelled").
		SetCompletedAt(now).
		Exec(l.ctx)
	if err != nil {
		return &io.BaseResp{Msg: "Update failed"}, err
	}

	return &io.BaseResp{Msg: "Task cancelled successfully"}, nil
}
