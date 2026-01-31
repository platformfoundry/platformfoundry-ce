package providers

import (
	"github.com/platformfoundry/platformfoundry-ce/internal/plugin"
	"github.com/platformfoundry/platformfoundry-ce/internal/plugin/providers/aws"
	"github.com/platformfoundry/platformfoundry-ce/internal/plugin/providers/kubernetes"
	"github.com/platformfoundry/platformfoundry-ce/internal/plugin/providers/terraform"
	pluginpkg "github.com/platformfoundry/platformfoundry-ce/pkg/plugin"
)

// BuiltinProviders returns all built-in plugin providers
func BuiltinProviders() []pluginpkg.Plugin {
	return []pluginpkg.Plugin{
		aws.New(),
		kubernetes.New(),
		terraform.New(),
	}
}

// RegisterBuiltins registers all built-in providers with the manager
func RegisterBuiltins(manager *plugin.Manager) error {
	for _, p := range BuiltinProviders() {
		if err := manager.Register(p); err != nil {
			return err
		}
	}
	return nil
}

// ProviderNames returns the names of all built-in providers
func ProviderNames() []string {
	providers := BuiltinProviders()
	names := make([]string, len(providers))
	for i, p := range providers {
		names[i] = p.Name()
	}
	return names
}
