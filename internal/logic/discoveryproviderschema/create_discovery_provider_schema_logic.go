package discoveryproviderschema

import (
	"context"
	"encoding/json"

	"github.com/coder-lulu/newbee-io-rpc/ent/discoveryproviderschema"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/internal/utils/dberrorhandler"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

    "github.com/coder-lulu/newbee-common/v2/msg/errormsg"

	"github.com/coder-lulu/newbee-io-rpc/internal/utils"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateDiscoveryProviderSchemaLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateDiscoveryProviderSchemaLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDiscoveryProviderSchemaLogic {
	return &CreateDiscoveryProviderSchemaLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateDiscoveryProviderSchemaLogic) CreateDiscoveryProviderSchema(in *io.DiscoveryProviderSchemaInfo) (*io.BaseIDResp, error) {
    query := l.svcCtx.DB.DiscoveryProviderSchema.Create().
			SetNotNilProviderID(in.ProviderId).
			SetNotNilProviderName(in.ProviderName).
			SetNotNilDescription(in.Description).
			SetNotNilVersion(in.Version).
			SetNotNilIconURL(in.IconUrl).
			SetNotNilIsBuiltin(in.IsBuiltin).
			SetNotNilIsActive(in.IsActive)

	if in.Category != nil {
		query.SetCategory(discoveryproviderschema.Category(*in.Category))
	}
	if in.ExecutionMode != nil {
		query.SetExecutionMode(discoveryproviderschema.ExecutionMode(*in.ExecutionMode))
	}
	if in.ParameterSchema != nil {
		var paramSchema []interface{}
		if err := json.Unmarshal(in.ParameterSchema, &paramSchema); err == nil {
			query.SetParameterSchema(paramSchema)
		}
	}
	if in.FieldSchema != nil {
		var fieldSchema []interface{}
		if err := json.Unmarshal(in.FieldSchema, &fieldSchema); err == nil {
			query.SetFieldSchema(fieldSchema)
		}
	}
	if in.Status != nil {
		query.SetNotNilStatus(utils.GetPointer(uint8(*in.Status)))
	}

	result, err := query.Save(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseIDResp{Id: result.ID, Msg: errormsg.CreateSuccess }, nil
}
