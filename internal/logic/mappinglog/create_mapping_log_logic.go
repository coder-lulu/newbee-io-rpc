package mappinglog

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

    "github.com/coder-lulu/newbee-common/v2/msg/errormsg"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateMappingLogLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateMappingLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateMappingLogLogic {
	return &CreateMappingLogLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateMappingLogLogic) CreateMappingLog(in *io.MappingLogInfo) (*io.BaseIDResp, error) {
    result, err := l.svcCtx.DB.MappingLog.Create().
			SetNotNilFieldMappingID(in.FieldMappingId).
			SetNotNilSourceValue(in.SourceValue).
			SetNotNilTargetValue(in.TargetValue).
			SetNotNilTransformStatus(in.TransformStatus).
			SetNotNilErrorMessage(in.ErrorMessage).
			SetNotNilLoggedAt(utils.GetTimeMilliPointer(in.LoggedAt)).
			Save(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseIDResp{Id: result.ID, Msg: errormsg.CreateSuccess }, nil
}
