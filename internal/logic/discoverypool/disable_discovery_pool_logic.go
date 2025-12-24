package discoverypool

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/zeromicro/go-zero/core/logx"
)

type DisableDiscoveryPoolLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDisableDiscoveryPoolLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableDiscoveryPoolLogic {
	return &DisableDiscoveryPoolLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DisableDiscoveryPoolLogic) DisableDiscoveryPool(in *io.IDReq) (*io.BaseResp, error) {
	pool, err := l.svcCtx.DB.DiscoveryPool.Get(l.ctx, in.Id)
	if err != nil {
		return &io.BaseResp{Msg: "DiscoveryPool not found"}, err
	}

	err = l.svcCtx.DB.DiscoveryPool.UpdateOne(pool).
		SetPoolStatus("inactive").
		SetStatus(2).
		Exec(l.ctx)
	if err != nil {
		return &io.BaseResp{Msg: "Update failed"}, err
	}

	return &io.BaseResp{Msg: "Success"}, nil
}
