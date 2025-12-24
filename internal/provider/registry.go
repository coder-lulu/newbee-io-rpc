package provider

import (
	"errors"
	"sync"
)

type ProviderRegistry struct {
	providers map[string]IDiscoveryProvider
	mu        sync.RWMutex
}

var (
	globalRegistry     *ProviderRegistry
	globalRegistryOnce sync.Once
)

func GetRegistry() *ProviderRegistry {
	globalRegistryOnce.Do(func() {
		globalRegistry = &ProviderRegistry{
			providers: make(map[string]IDiscoveryProvider),
		}
		globalRegistry.registerBuiltinProviders()
	})
	return globalRegistry
}

func (r *ProviderRegistry) registerBuiltinProviders() {
	// 文件导入提供者
	r.Register(NewFileImportProvider())
	
	// NewBee Agent 分布式发现提供者
	r.Register(NewNBAgentProvider())
	
	// 阿里云ECS提供者
	r.Register(NewAliyunECSProvider())
	
	// VMware vCenter提供者
	r.Register(NewVMwareVCenterProvider())
}

func (r *ProviderRegistry) Register(provider IDiscoveryProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	metadata := provider.GetMetadata()
	r.providers[metadata.ID] = provider
}

func (r *ProviderRegistry) Get(providerID string) (IDiscoveryProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[providerID]
	if !exists {
		return nil, errors.New("provider not found: " + providerID)
	}
	return provider, nil
}

func (r *ProviderRegistry) List() []IDiscoveryProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]IDiscoveryProvider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}
	return providers
}

func (r *ProviderRegistry) Exists(providerID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.providers[providerID]
	return exists
}
