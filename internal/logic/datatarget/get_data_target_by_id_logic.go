package datatarget

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetDataTargetByIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDataTargetByIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDataTargetByIdLogic {
	return &GetDataTargetByIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetDataTargetByIdLogic) GetDataTargetById(in *io.IDReq) (*io.DataTargetInfo, error) {
	result, err := l.svcCtx.DB.DataTarget.Get(l.ctx, in.Id)
	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	return &io.DataTargetInfo{
		Id:          &result.ID,
		CreatedAt:    utils.GetPointer(result.CreatedAt.UnixMilli()),
		UpdatedAt:    utils.GetPointer(result.UpdatedAt.UnixMilli()),
		Status:	utils.GetPointer(uint32(result.Status)),
		TargetName:	&result.TargetName,
		TargetCode:	&result.TargetCode,
		Description:	&result.Description,
		TargetType:	&result.TargetType,
		TargetSystem:	&result.TargetSystem,
		ConnectionConfig:	&result.ConnectionConfig,
		AuthConfig:	&result.AuthConfig,
		IsActive:	&result.IsActive,
		Metadata:	&result.Metadata,
	}, nil
}

