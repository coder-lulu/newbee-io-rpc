package datatarget

import (
	"context"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent/datatarget"
	"github.com/coder-lulu/newbee-io-rpc/ent/predicate"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
    "github.com/zeromicro/go-zero/core/logx"
)

type GetDataTargetListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDataTargetListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDataTargetListLogic {
	return &GetDataTargetListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetDataTargetListLogic) GetDataTargetList(in *io.DataTargetListReq) (*io.DataTargetListResp, error) {
	var predicates []predicate.DataTarget
	if in.CreatedAt != nil {
		predicates = append(predicates, datatarget.CreatedAtGTE(time.UnixMilli(*in.CreatedAt)))
	}
	if in.UpdatedAt != nil {
		predicates = append(predicates, datatarget.UpdatedAtGTE(time.UnixMilli(*in.UpdatedAt)))
	}
	if in.Status != nil {
		predicates = append(predicates, datatarget.StatusEQ(uint8(*in.Status)))
	}
	if in.TargetName != nil {
		predicates = append(predicates, datatarget.TargetNameContains(*in.TargetName))
	}
	if in.TargetCode != nil {
		predicates = append(predicates, datatarget.TargetCodeContains(*in.TargetCode))
	}
	if in.Description != nil {
		predicates = append(predicates, datatarget.DescriptionContains(*in.Description))
	}
	if in.TargetType != nil {
		predicates = append(predicates, datatarget.TargetTypeContains(*in.TargetType))
	}
	if in.TargetSystem != nil {
		predicates = append(predicates, datatarget.TargetSystemContains(*in.TargetSystem))
	}
	if in.ConnectionConfig != nil {
		predicates = append(predicates, datatarget.ConnectionConfigContains(*in.ConnectionConfig))
	}
	if in.AuthConfig != nil {
		predicates = append(predicates, datatarget.AuthConfigContains(*in.AuthConfig))
	}
	if in.IsActive != nil {
		predicates = append(predicates, datatarget.IsActiveEQ(*in.IsActive))
	}
	if in.Metadata != nil {
		predicates = append(predicates, datatarget.MetadataContains(*in.Metadata))
	}
	result, err := l.svcCtx.DB.DataTarget.Query().Where(predicates...).Page(l.ctx, in.Page, in.PageSize)

	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	resp := &io.DataTargetListResp{}
	resp.Total = result.PageDetails.Total

	for _, v := range result.List {
		resp.Data = append(resp.Data, &io.DataTargetInfo{
			Id:          &v.ID,
			CreatedAt:   utils.GetPointer(v.CreatedAt.UnixMilli()),
			UpdatedAt:   utils.GetPointer(v.UpdatedAt.UnixMilli()),
			Status:	utils.GetPointer(uint32(v.Status)),
			TargetName:	&v.TargetName,
			TargetCode:	&v.TargetCode,
			Description:	&v.Description,
			TargetType:	&v.TargetType,
			TargetSystem:	&v.TargetSystem,
			ConnectionConfig:	&v.ConnectionConfig,
			AuthConfig:	&v.AuthConfig,
			IsActive:	&v.IsActive,
			Metadata:	&v.Metadata,
		})
	}

	return resp, nil
}
