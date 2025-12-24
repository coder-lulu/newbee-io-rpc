package discoveryproviderschema

import (
	"context"
	"encoding/json"
	"time"

	"github.com/coder-lulu/newbee-io-rpc/ent/discoveryproviderschema"
	"github.com/coder-lulu/newbee-io-rpc/ent/predicate"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
    "github.com/zeromicro/go-zero/core/logx"
)

type GetDiscoveryProviderSchemaListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDiscoveryProviderSchemaListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDiscoveryProviderSchemaListLogic {
	return &GetDiscoveryProviderSchemaListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetDiscoveryProviderSchemaListLogic) GetDiscoveryProviderSchemaList(in *io.DiscoveryProviderSchemaListReq) (*io.DiscoveryProviderSchemaListResp, error) {
	var predicates []predicate.DiscoveryProviderSchema
	if in.CreatedAt != nil {
		predicates = append(predicates, discoveryproviderschema.CreatedAtGTE(time.UnixMilli(*in.CreatedAt)))
	}
	if in.UpdatedAt != nil {
		predicates = append(predicates, discoveryproviderschema.UpdatedAtGTE(time.UnixMilli(*in.UpdatedAt)))
	}
	if in.Status != nil {
		predicates = append(predicates, discoveryproviderschema.StatusEQ(uint8(*in.Status)))
	}
	if in.ProviderId != nil {
		predicates = append(predicates, discoveryproviderschema.ProviderIDContains(*in.ProviderId))
	}
	if in.ProviderName != nil {
		predicates = append(predicates, discoveryproviderschema.ProviderNameContains(*in.ProviderName))
	}
	if in.Category != nil {
		predicates = append(predicates, discoveryproviderschema.CategoryEQ(discoveryproviderschema.Category(*in.Category)))
	}
	if in.Description != nil {
		predicates = append(predicates, discoveryproviderschema.DescriptionContains(*in.Description))
	}
	if in.Version != nil {
		predicates = append(predicates, discoveryproviderschema.VersionContains(*in.Version))
	}
	if in.IconUrl != nil {
		predicates = append(predicates, discoveryproviderschema.IconURLContains(*in.IconUrl))
	}
	if in.IsBuiltin != nil {
		predicates = append(predicates, discoveryproviderschema.IsBuiltinEQ(*in.IsBuiltin))
	}
	if in.ExecutionMode != nil {
		predicates = append(predicates, discoveryproviderschema.ExecutionModeEQ(discoveryproviderschema.ExecutionMode(*in.ExecutionMode)))
	}
	if in.IsActive != nil {
		predicates = append(predicates, discoveryproviderschema.IsActiveEQ(*in.IsActive))
	}
	result, err := l.svcCtx.DB.DiscoveryProviderSchema.Query().Where(predicates...).Page(l.ctx, in.Page, in.PageSize)

	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	resp := &io.DiscoveryProviderSchemaListResp{}
	resp.Total = result.PageDetails.Total

	for _, v := range result.List {
		info := &io.DiscoveryProviderSchemaInfo{
			Id:          &v.ID,
			CreatedAt:   utils.GetPointer(v.CreatedAt.UnixMilli()),
			UpdatedAt:   utils.GetPointer(v.UpdatedAt.UnixMilli()),
			Status:	utils.GetPointer(uint32(v.Status)),
			ProviderId:	&v.ProviderID,
			ProviderName:	&v.ProviderName,
			Description:	&v.Description,
			Version:	&v.Version,
			IconUrl:	&v.IconURL,
			IsBuiltin:	&v.IsBuiltin,
			IsActive:	&v.IsActive,
		}

		category := string(v.Category)
		info.Category = &category
		executionMode := string(v.ExecutionMode)
		info.ExecutionMode = &executionMode

		if v.ParameterSchema != nil {
			paramBytes, _ := json.Marshal(v.ParameterSchema)
			info.ParameterSchema = paramBytes
		}
		if v.FieldSchema != nil {
			fieldBytes, _ := json.Marshal(v.FieldSchema)
			info.FieldSchema = fieldBytes
		}

		resp.Data = append(resp.Data, info)
	}

	return resp, nil
}
