package outputtask

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/zeromicro/go-zero/core/logx"
)

type ApproveOutputTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewApproveOutputTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApproveOutputTaskLogic {
	return &ApproveOutputTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// OutputTask lifecycle operations
func (l *ApproveOutputTaskLogic) ApproveOutputTask(in *io.ApprovalReq) (*io.BaseResp, error) {
	task, err := l.svcCtx.DB.OutputTask.Get(l.ctx, in.Id)
	if err != nil {
		return &io.BaseResp{Msg: "Task not found"}, err
	}

	if task.TaskStatus == "approved" {
		return &io.BaseResp{Msg: "Already approved"}, nil
	}

	update := l.svcCtx.DB.OutputTask.UpdateOne(task)

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
