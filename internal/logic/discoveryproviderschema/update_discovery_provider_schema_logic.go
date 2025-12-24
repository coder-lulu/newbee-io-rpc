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

type UpdateDiscoveryProviderSchemaLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateDiscoveryProviderSchemaLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateDiscoveryProviderSchemaLogic {
	return &UpdateDiscoveryProviderSchemaLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateDiscoveryProviderSchemaLogic) UpdateDiscoveryProviderSchema(in *io.DiscoveryProviderSchemaInfo) (*io.BaseResp, error) {
	query:= l.svcCtx.DB.DiscoveryProviderSchema.UpdateOneID(*in.Id).
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

	 err := query.Exec(l.ctx)

    if err != nil {
		return nil, dberrorhandler.DefaultEntError(l.Logger, err, in)
	}

    return &io.BaseResp{Msg: errormsg.UpdateSuccess }, nil
}
