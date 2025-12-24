package discoverypool

import (
	"context"

    "github.com/coder-lulu/newbee-io-rpc/ent/discoverypool"
    "github.com/coder-lulu/newbee-io-rpc/internal/svc"
    "github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
    "github.com/coder-lulu/newbee-io-rpc/types/io"

    "github.com/coder-lulu/newbee-common/v2/msg/errormsg"
    "github.com/zeromicro/go-zero/core/logx"
)

type DeleteDiscoveryPoolLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteDiscoveryPoolLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteDiscoveryPoolLogic {
	return &DeleteDiscoveryPoolLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteDiscoveryPoolLogic) DeleteDiscoveryPool(in *io.IDsReq) (*io.BaseResp, error) {
	_, err := l.svcCtx.DB.DiscoveryPool.Delete().Where(discoverypool.IDIn(in.Ids...)).Exec(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseResp{Msg: errormsg.DeleteSuccess }, nil
}
