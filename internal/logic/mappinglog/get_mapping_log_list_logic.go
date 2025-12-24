package mappinglog

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent/mappinglog"
	"github.com/coder-lulu/newbee-io-rpc/ent/predicate"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
    "github.com/zeromicro/go-zero/core/logx"
)

type GetMappingLogListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMappingLogListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMappingLogListLogic {
	return &GetMappingLogListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetMappingLogListLogic) GetMappingLogList(in *io.MappingLogListReq) (*io.MappingLogListResp, error) {
	var predicates []predicate.MappingLog
	if in.CreatedAt != nil {
		predicates = append(predicates, mappinglog.CreatedAtGTE(time.UnixMilli(*in.CreatedAt)))
	}
	if in.UpdatedAt != nil {
		predicates = append(predicates, mappinglog.UpdatedAtGTE(time.UnixMilli(*in.UpdatedAt)))
	}
	if in.FieldMappingId != nil {
		predicates = append(predicates, mappinglog.FieldMappingIDEQ(*in.FieldMappingId))
	}
	if in.SourceValue != nil {
		predicates = append(predicates, mappinglog.SourceValueContains(*in.SourceValue))
	}
	if in.TargetValue != nil {
		predicates = append(predicates, mappinglog.TargetValueContains(*in.TargetValue))
	}
	if in.TransformStatus != nil {
		predicates = append(predicates, mappinglog.TransformStatusContains(*in.TransformStatus))
	}
	if in.ErrorMessage != nil {
		predicates = append(predicates, mappinglog.ErrorMessageContains(*in.ErrorMessage))
	}
	if in.LoggedAt != nil {
		predicates = append(predicates, mappinglog.LoggedAtGTE(time.UnixMilli(*in.LoggedAt)))
	}
	result, err := l.svcCtx.DB.MappingLog.Query().Where(predicates...).Page(l.ctx, in.Page, in.PageSize)

	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	resp := &io.MappingLogListResp{}
	resp.Total = result.PageDetails.Total

	for _, v := range result.List {
		resp.Data = append(resp.Data, &io.MappingLogInfo{
			Id:          &v.ID,
			CreatedAt:   utils.GetPointer(v.CreatedAt.UnixMilli()),
			UpdatedAt:   utils.GetPointer(v.UpdatedAt.UnixMilli()),
			FieldMappingId:	&v.FieldMappingID,
			SourceValue:	&v.SourceValue,
			TargetValue:	&v.TargetValue,
			TransformStatus:	&v.TransformStatus,
			ErrorMessage:	&v.ErrorMessage,
			LoggedAt:	utils.GetPointer(v.LoggedAt.UnixMilli()),
		})
	}

	return resp, nil
}
