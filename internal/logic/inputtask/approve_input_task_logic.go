package inputtask

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/zeromicro/go-zero/core/logx"
)

type ApproveInputTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewApproveInputTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApproveInputTaskLogic {
	return &ApproveInputTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// InputTask lifecycle operations
func (l *ApproveInputTaskLogic) ApproveInputTask(in *io.ApprovalReq) (*io.BaseResp, error) {
	task, err := l.svcCtx.DB.InputTask.Get(l.ctx, in.Id)
	if err != nil {
		return &io.BaseResp{Msg: "Task not found"}, err
	}

	if task.TaskStatus == "approved" {
		return &io.BaseResp{Msg: "Already approved"}, nil
	}

	update := l.svcCtx.DB.InputTask.UpdateOne(task)

	// Handle approve or reject action
	if in.Action == "approve" {
		update.SetTaskStatus("approved")
	} else if in.Action == "reject" {
		update.SetTaskStatus("rejected")
	} else {
		return &io.BaseResp{Msg: "Invalid action, must be 'approve' or 'reject'"}, nil
	}

	err = update.Exec(l.ctx)
	if err != nil {
		return &io.BaseResp{Msg: "Update failed"}, err
	}

	return &io.BaseResp{Msg: "Success"}, nil
}
