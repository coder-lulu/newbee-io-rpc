package crontask

import (
	"context"

    "github.com/coder-lulu/newbee-io-rpc/ent/crontask"
    "github.com/coder-lulu/newbee-io-rpc/internal/svc"
    "github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
    "github.com/coder-lulu/newbee-io-rpc/types/io"

    "github.com/suyuan32/simple-admin-common/msg/errormsg"
    "github.com/zeromicro/go-zero/core/logx"
)

type DeleteCronTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteCronTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteCronTaskLogic {
	return &DeleteCronTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteCronTaskLogic) DeleteCronTask(in *io.IDsReq) (*io.BaseResp, error) {
	_, err := l.svcCtx.DB.CronTask.Delete().Where(crontask.IDIn(in.Ids...)).Exec(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseResp{Msg: errormsg.DeleteSuccess }, nil
}
