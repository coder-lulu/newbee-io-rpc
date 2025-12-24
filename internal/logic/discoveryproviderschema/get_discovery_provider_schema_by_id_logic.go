package discoveryproviderschema

import (
	"context"
	"encoding/json"

	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetDiscoveryProviderSchemaByIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDiscoveryProviderSchemaByIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDiscoveryProviderSchemaByIdLogic {
	return &GetDiscoveryProviderSchemaByIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetDiscoveryProviderSchemaByIdLogic) GetDiscoveryProviderSchemaById(in *io.IDReq) (*io.DiscoveryProviderSchemaInfo, error) {
	result, err := l.svcCtx.DB.DiscoveryProviderSchema.Get(l.ctx, in.Id)
	if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

	resp := &io.DiscoveryProviderSchemaInfo{
		Id:          &result.ID,
		CreatedAt:    utils.GetPointer(result.CreatedAt.UnixMilli()),
		UpdatedAt:    utils.GetPointer(result.UpdatedAt.UnixMilli()),
		Status:	utils.GetPointer(uint32(result.Status)),
		ProviderId:	&result.ProviderID,
		ProviderName:	&result.ProviderName,
		Description:	&result.Description,
		Version:	&result.Version,
		IconUrl:	&result.IconURL,
		IsBuiltin:	&result.IsBuiltin,
		IsActive:	&result.IsActive,
	}

	category := string(result.Category)
	resp.Category = &category
	executionMode := string(result.ExecutionMode)
	resp.ExecutionMode = &executionMode

	if result.ParameterSchema != nil {
		paramBytes, _ := json.Marshal(result.ParameterSchema)
		resp.ParameterSchema = paramBytes
	}
	if result.FieldSchema != nil {
		fieldBytes, _ := json.Marshal(result.FieldSchema)
		resp.FieldSchema = fieldBytes
	}

	return resp, nil
}

