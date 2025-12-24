package discoverypool

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/zeromicro/go-zero/core/logx"
)

type EnableDiscoveryPoolLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewEnableDiscoveryPoolLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnableDiscoveryPoolLogic {
	return &EnableDiscoveryPoolLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *EnableDiscoveryPoolLogic) EnableDiscoveryPool(in *io.IDReq) (*io.BaseResp, error) {
	pool, err := l.svcCtx.DB.DiscoveryPool.Get(l.ctx, in.Id)
	if err != nil {
		return &io.BaseResp{Msg: "DiscoveryPool not found"}, err
	}

	if pool.ApprovalStatus != "approved" {
		return &io.BaseResp{Msg: "Pool not approved"}, nil
	}

	err = l.svcCtx.DB.DiscoveryPool.UpdateOne(pool).
		SetPoolStatus("active").
		SetStatus(1).
		Exec(l.ctx)
	if err != nil {
		return &io.BaseResp{Msg: "Update failed"}, err
	}

	return &io.BaseResp{Msg: "Success"}, nil
}
