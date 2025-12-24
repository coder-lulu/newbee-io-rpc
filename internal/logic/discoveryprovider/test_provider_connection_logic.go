package discoveryprovider

import (
	"context"
	"encoding/json"

	"github.com/coder-lulu/newbee-io-rpc/ent/discoveryproviderschema"
	"github.com/coder-lulu/newbee-io-rpc/internal/provider"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/zeromicro/go-zero/core/logx"
)

type TestProviderConnectionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTestProviderConnectionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TestProviderConnectionLogic {
	return &TestProviderConnectionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *TestProviderConnectionLogic) TestProviderConnection(in *io.TestConnectionReq) (*io.TestConnectionResp, error) {
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(in.Config), &config); err != nil {
		msg := "Invalid config JSON: " + err.Error()
		return &io.TestConnectionResp{
			Success: false,
			Message: &msg,
		}, nil
	}

	registry := provider.GetRegistry()
	if p, err := registry.Get(in.ProviderId); err == nil {
		result, err := p.TestConnection(config)
		if err != nil {
			msg := "Test failed: " + err.Error()
			return &io.TestConnectionResp{
				Success: false,
				Message: &msg,
			}, nil
		}
		return &io.TestConnectionResp{
			Success: result.Success,
			Message: &result.Message,
		}, nil
	}

	schema, err := l.svcCtx.DB.DiscoveryProviderSchema.Query().
		Where(discoveryproviderschema.ProviderIDEQ(in.ProviderId)).
		First(l.ctx)
	if err != nil {
		msg := "Provider not found: " + err.Error()
		return &io.TestConnectionResp{
			Success: false,
			Message: &msg,
		}, nil
	}

	var paramSchema []interface{}
	if len(schema.ParameterSchema) > 0 {
		paramSchema = schema.ParameterSchema
	}

	for _, param := range paramSchema {
		paramMap, ok := param.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := paramMap["name"].(string)
		required, _ := paramMap["required"].(bool)
		if required {
			if _, exists := config[name]; !exists {
				msg := "Required parameter missing: " + name
				return &io.TestConnectionResp{
					Success: false,
					Message: &msg,
				}, nil
			}
		}
	}

	msg := "Configuration validation passed"
	return &io.TestConnectionResp{
		Success: true,
		Message: &msg,
	}, nil
}
