package discoverypool

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/zeromicro/go-zero/core/logx"
)

type ApproveDiscoveryPoolLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewApproveDiscoveryPoolLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApproveDiscoveryPoolLogic {
	return &ApproveDiscoveryPoolLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DiscoveryPool lifecycle operations
func (l *ApproveDiscoveryPoolLogic) ApproveDiscoveryPool(in *io.ApprovalReq) (*io.BaseResp, error) {
	pool, err := l.svcCtx.DB.DiscoveryPool.Get(l.ctx, in.Id)
	if err != nil {
		return &io.BaseResp{Msg: "DiscoveryPool not found"}, err
	}

	if pool.ApprovalStatus == "approved" {
		return &io.BaseResp{Msg: "Already approved"}, nil
	}

	now := time.Now()
	update := l.svcCtx.DB.DiscoveryPool.UpdateOne(pool).
		SetApprovedAt(now)

	// Handle approve or reject action
	if in.Action == "approve" {
		update.SetApprovalStatus("approved")
	} else if in.Action == "reject" {
		update.SetApprovalStatus("rejected")
		if in.Reason != nil {
			update.SetRejectionReason(*in.Reason)
		}
	} else {
		return &io.BaseResp{Msg: "Invalid action, must be 'approve' or 'reject'"}, nil
	}

	// Set approver if provided
	if in.ApprovedBy != nil {
		update.SetApprovedBy(*in.ApprovedBy)
	}

	err = update.Exec(l.ctx)
	if err != nil {
		return &io.BaseResp{Msg: "Update failed"}, err
	}

	return &io.BaseResp{Msg: "Success"}, nil
}
