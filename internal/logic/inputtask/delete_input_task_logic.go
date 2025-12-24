package inputtask

import (
	"context"

    "github.com/coder-lulu/newbee-io-rpc/ent/inputtask"
    "github.com/coder-lulu/newbee-io-rpc/internal/svc"
    "github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
    "github.com/coder-lulu/newbee-io-rpc/types/io"

    "github.com/coder-lulu/newbee-common/v2/msg/errormsg"
    "github.com/zeromicro/go-zero/core/logx"
)

type DeleteInputTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteInputTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteInputTaskLogic {
	return &DeleteInputTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteInputTaskLogic) DeleteInputTask(in *io.IDsReq) (*io.BaseResp, error) {
	_, err := l.svcCtx.DB.InputTask.Delete().Where(inputtask.IDIn(in.Ids...)).Exec(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseResp{Msg: errormsg.DeleteSuccess }, nil
}
