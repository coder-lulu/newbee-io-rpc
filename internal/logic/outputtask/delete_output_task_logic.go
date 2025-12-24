package outputtask

import (
	"context"

    "github.com/coder-lulu/newbee-io-rpc/ent/outputtask"
    "github.com/coder-lulu/newbee-io-rpc/internal/svc"
    "github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
    "github.com/coder-lulu/newbee-io-rpc/types/io"

    "github.com/coder-lulu/newbee-common/v2/msg/errormsg"
    "github.com/zeromicro/go-zero/core/logx"
)

type DeleteOutputTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteOutputTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteOutputTaskLogic {
	return &DeleteOutputTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteOutputTaskLogic) DeleteOutputTask(in *io.IDsReq) (*io.BaseResp, error) {
	_, err := l.svcCtx.DB.OutputTask.Delete().Where(outputtask.IDIn(in.Ids...)).Exec(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseResp{Msg: errormsg.DeleteSuccess }, nil
}
