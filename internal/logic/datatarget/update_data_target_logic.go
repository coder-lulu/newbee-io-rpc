package datatarget

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

    "github.com/coder-lulu/newbee-common/v2/msg/errormsg"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateDataTargetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateDataTargetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateDataTargetLogic {
	return &UpdateDataTargetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateDataTargetLogic) UpdateDataTarget(in *io.DataTargetInfo) (*io.BaseResp, error) {
	query:= l.svcCtx.DB.DataTarget.UpdateOneID(*in.Id).
			SetNotNilTargetName(in.TargetName).
			SetNotNilTargetCode(in.TargetCode).
			SetNotNilDescription(in.Description).
			SetNotNilTargetType(in.TargetType).
			SetNotNilTargetSystem(in.TargetSystem).
			SetNotNilConnectionConfig(in.ConnectionConfig).
			SetNotNilAuthConfig(in.AuthConfig).
			SetNotNilIsActive(in.IsActive).
			SetNotNilMetadata(in.Metadata)

	if in.Status != nil {
		query.SetNotNilStatus(utils.GetPointer(uint8(*in.Status)))
	}

	 err := query.Exec(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseResp{Msg: errormsg.UpdateSuccess }, nil
}
