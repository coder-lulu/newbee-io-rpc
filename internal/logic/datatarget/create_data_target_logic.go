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

type CreateDataTargetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateDataTargetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDataTargetLogic {
	return &CreateDataTargetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateDataTargetLogic) CreateDataTarget(in *io.DataTargetInfo) (*io.BaseIDResp, error) {
    query := l.svcCtx.DB.DataTarget.Create().
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

	result, err := query.Save(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseIDResp{Id: result.ID, Msg: errormsg.CreateSuccess }, nil
}
