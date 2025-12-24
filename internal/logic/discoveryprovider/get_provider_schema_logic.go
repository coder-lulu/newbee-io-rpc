package discoveryprovider

import (
	"context"
	"fmt"

	"github.com/coder-lulu/newbee-io-rpc/ent/discoveryproviderschema"
	"github.com/coder-lulu/newbee-io-rpc/internal/provider"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetProviderSchemaLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetProviderSchemaLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetProviderSchemaLogic {
	return &GetProviderSchemaLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetProviderSchemaLogic) GetProviderSchema(in *io.GetProviderSchemaReq) (*io.GetProviderSchemaResp, error) {
	registry := provider.GetRegistry()
	if p, err := registry.Get(in.ProviderId); err == nil {
		return l.buildSchemaFromProvider(p), nil
	}

	schema, err := l.svcCtx.DB.DiscoveryProviderSchema.Query().
		Where(discoveryproviderschema.ProviderIDEQ(in.ProviderId)).
		First(l.ctx)
	if err != nil {
		return nil, err
	}

	var paramDefs []*io.ParameterDefinition
	for _, param := range schema.ParameterSchema {
		paramMap, ok := param.(map[string]interface{})
		if !ok {
			continue
		}
		paramDef := &io.ParameterDefinition{
			Name:     getString(paramMap, "name"),
			Label:    getString(paramMap, "label"),
			Type:     getString(paramMap, "type"),
			Required: getBool(paramMap, "required"),
		}
		if desc := getString(paramMap, "description"); desc != "" {
			paramDef.Description = &desc
		}
		if def := getString(paramMap, "default_value"); def != "" {
			paramDef.DefaultValue = &def
		}
		if placeholder := getString(paramMap, "placeholder"); placeholder != "" {
			paramDef.Placeholder = &placeholder
		}
		if options, ok := paramMap["options"].([]interface{}); ok {
			for _, opt := range options {
				if optStr, ok := opt.(string); ok {
					paramDef.Options = append(paramDef.Options, optStr)
				}
			}
		}
		paramDefs = append(paramDefs, paramDef)
	}

	var fieldDefs []*io.FieldDefinition
	for _, field := range schema.FieldSchema {
		fieldMap, ok := field.(map[string]interface{})
		if !ok {
			continue
		}
		fieldDef := &io.FieldDefinition{
			Name:     getString(fieldMap, "name"),
			Label:    getString(fieldMap, "label"),
			DataType: getString(fieldMap, "data_type"),
			Required: getBool(fieldMap, "required"),
		}
		if desc := getString(fieldMap, "description"); desc != "" {
			fieldDef.Description = &desc
		}
		if example := getString(fieldMap, "example"); example != "" {
			fieldDef.Example = &example
		}
		fieldDefs = append(fieldDefs, fieldDef)
	}

	idStr := fmt.Sprintf("%d", schema.ID)
	categoryStr := string(schema.Category)
	execModeStr := string(schema.ExecutionMode)

	schemaInfo := &io.ProviderSchemaInfo{
		Id:              idStr,
		ProviderId:      schema.ProviderID,
		ProviderName:    schema.ProviderName,
		Category:        categoryStr,
		ParameterSchema: paramDefs,
		FieldSchema:     fieldDefs,
		IsActive:        schema.IsActive,
		IsBuiltin:       schema.IsBuiltin,
		ExecutionMode:   execModeStr,
	}

	if schema.Description != "" {
		schemaInfo.Description = &schema.Description
	}
	if schema.Version != "" {
		schemaInfo.Version = &schema.Version
	}
	if schema.IconURL != "" {
		schemaInfo.IconUrl = &schema.IconURL
	}

	return &io.GetProviderSchemaResp{
		Data: schemaInfo,
	}, nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func (l *GetProviderSchemaLogic) buildSchemaFromProvider(p provider.IDiscoveryProvider) *io.GetProviderSchemaResp {
	metadata := p.GetMetadata()
	paramSchema := p.GetParameterSchema()
	fieldSchema := p.GetFieldSchema()

	var paramDefs []*io.ParameterDefinition
	for _, param := range paramSchema {
		paramDef := &io.ParameterDefinition{
			Name:     param.Name,
			Label:    param.Name,
			Type:     param.Type,
			Required: param.Required,
		}
		if param.Description != "" {
			paramDef.Description = &param.Description
		}
		if param.Default != nil {
			if defStr, ok := param.Default.(string); ok {
				paramDef.DefaultValue = &defStr
			}
		}
		if len(param.Options) > 0 {
			for _, option := range param.Options {
				paramDef.Options = append(paramDef.Options, option.Value)
			}
		}
		paramDefs = append(paramDefs, paramDef)
	}

	var fieldDefs []*io.FieldDefinition
	for _, field := range fieldSchema {
		fieldDef := &io.FieldDefinition{
			Name:     field.Name,
			Label:    field.Name,
			DataType: field.Type,
			Required: false,
		}
		if field.Description != "" {
			fieldDef.Description = &field.Description
		}
		fieldDefs = append(fieldDefs, fieldDef)
	}

	schemaInfo := &io.ProviderSchemaInfo{
		Id:              metadata.ID,
		ProviderId:      metadata.ID,
		ProviderName:    metadata.Name,
		Category:        metadata.Category,
		ParameterSchema: paramDefs,
		FieldSchema:     fieldDefs,
		IsActive:        true,
		IsBuiltin:       true,
		ExecutionMode:   "direct",
	}

	if metadata.Description != "" {
		schemaInfo.Description = &metadata.Description
	}
	if metadata.Version != "" {
		schemaInfo.Version = &metadata.Version
	}
	if metadata.Icon != "" {
		schemaInfo.IconUrl = &metadata.Icon
	}

	return &io.GetProviderSchemaResp{
		Data: schemaInfo,
	}
}
