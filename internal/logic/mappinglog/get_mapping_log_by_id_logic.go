package mappinglog

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetMappingLogByIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMappingLogByIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMappingLogByIdLogic {
	return &GetMappingLogByIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetMappingLogByIdLogic) GetMappingLogById(in *io.IDReq) (*io.MappingLogInfo, error) {
	result, err := l.svcCtx.DB.MappingLog.Get(l.ctx, in.Id)
	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	return &io.MappingLogInfo{
		Id:          &result.ID,
		CreatedAt:    utils.GetPointer(result.CreatedAt.UnixMilli()),
		UpdatedAt:    utils.GetPointer(result.UpdatedAt.UnixMilli()),
		FieldMappingId:	&result.FieldMappingID,
		SourceValue:	&result.SourceValue,
		TargetValue:	&result.TargetValue,
		TransformStatus:	&result.TransformStatus,
		ErrorMessage:	&result.ErrorMessage,
		LoggedAt:	utils.GetPointer(result.LoggedAt.UnixMilli()),
	}, nil
}

