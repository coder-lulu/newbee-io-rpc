package mappinglog

import (
	"context"

    "github.com/coder-lulu/newbee-io-rpc/ent/mappinglog"
    "github.com/coder-lulu/newbee-io-rpc/internal/svc"
    "github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
    "github.com/coder-lulu/newbee-io-rpc/types/io"

    "github.com/coder-lulu/newbee-common/v2/msg/errormsg"
    "github.com/zeromicro/go-zero/core/logx"
)

type DeleteMappingLogLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteMappingLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteMappingLogLogic {
	return &DeleteMappingLogLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteMappingLogLogic) DeleteMappingLog(in *io.IDsReq) (*io.BaseResp, error) {
	_, err := l.svcCtx.DB.MappingLog.Delete().Where(mappinglog.IDIn(in.Ids...)).Exec(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseResp{Msg: errormsg.DeleteSuccess }, nil
}
