package discoveryprovider

import (
	"context"

	"github.com/coder-lulu/newbee-io-rpc/internal/provider"
	"github.com/coder-lulu/newbee-io-rpc/internal/svc"
	"github.com/coder-lulu/newbee-io-rpc/types/io"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListDiscoveryProvidersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListDiscoveryProvidersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListDiscoveryProvidersLogic {
	return &ListDiscoveryProvidersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Discovery Provider 业务接口
func (l *ListDiscoveryProvidersLogic) ListDiscoveryProviders(in *io.Empty) (*io.ListProvidersResp, error) {
	var providers []*io.ProviderMetadata

	registry := provider.GetRegistry()
	builtinProviders := registry.List()
	for _, p := range builtinProviders {
		metadata := p.GetMetadata()
		providerMeta := &io.ProviderMetadata{
			Id:       metadata.ID,
			Name:     metadata.Name,
			Category: metadata.Category,
		}
		if metadata.Description != "" {
			providerMeta.Description = &metadata.Description
		}
		if metadata.Version != "" {
			providerMeta.Version = &metadata.Version
		}
		if metadata.Icon != "" {
			providerMeta.Icon = &metadata.Icon
		}
		providers = append(providers, providerMeta)
	}

	schemas, err := l.svcCtx.DB.DiscoveryProviderSchema.Query().All(l.ctx)
	if err != nil {
		return nil, err
	}

	for _, schema := range schemas {
		if registry.Exists(schema.ProviderID) {
			continue
		}

		categoryStr := string(schema.Category)
		provider := &io.ProviderMetadata{
			Id:       schema.ProviderID,
			Name:     schema.ProviderName,
			Category: categoryStr,
		}
		if schema.Description != "" {
			provider.Description = &schema.Description
		}
		if schema.Version != "" {
			provider.Version = &schema.Version
		}
		if schema.IconURL != "" {
			provider.Icon = &schema.IconURL
		}
		providers = append(providers, provider)
	}

	return &io.ListProvidersResp{
		Data: providers,
	}, nil
}
