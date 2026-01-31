package integration

import (
	"testing"

	"github.com/platformfoundry/platformfoundry-ce/internal/plugin"
	"github.com/platformfoundry/platformfoundry-ce/internal/plugins/clusterexisting"
	"github.com/platformfoundry/platformfoundry-ce/internal/plugins/devex/backstage"
	"github.com/platformfoundry/platformfoundry-ce/internal/plugins/infrastructure/terraform"
	"github.com/platformfoundry/platformfoundry-ce/internal/plugins/observability/grafana"
	"github.com/platformfoundry/platformfoundry-ce/internal/plugins/observability/prometheus"
	"github.com/platformfoundry/platformfoundry-ce/internal/plugins/orchestrator/argocd"
	pkgplugin "github.com/platformfoundry/platformfoundry-ce/pkg/plugin"
)

func TestPlugins_RegistrationAndDiscovery(t *testing.T) {
	pm := plugin.NewManager()

	// Test plugin registration
	t.Run("Register All Plugins", func(t *testing.T) {
		// Register infrastructure plugins
		terraformPlugin := terraform.NewPlugin()
		if err := pm.Register(terraformPlugin); err != nil {
			t.Errorf("Failed to register terraform plugin: %v", err)
		}

		// Register cluster plugins
		existingClusterPlugin := clusterexisting.NewPlugin()
		if err := pm.Register(existingClusterPlugin); err != nil {
			t.Errorf("Failed to register existing cluster plugin: %v", err)
		}

		// Register orchestrator plugins
		argocdPlugin := argocd.NewPlugin()
		if err := pm.Register(argocdPlugin); err != nil {
			t.Errorf("Failed to register argocd plugin: %v", err)
		}

		// Register observability plugins
		prometheusPlugin := prometheus.NewPlugin()
		if err := pm.Register(prometheusPlugin); err != nil {
			t.Errorf("Failed to register prometheus plugin: %v", err)
		}

		grafanaPlugin := grafana.NewPlugin()
		if err := pm.Register(grafanaPlugin); err != nil {
			t.Errorf("Failed to register grafana plugin: %v", err)
		}

		// Register devex plugins
		backstagePlugin := backstage.NewPlugin()
		if err := pm.Register(backstagePlugin); err != nil {
			t.Errorf("Failed to register backstage plugin: %v", err)
		}
	})

	t.Run("Discover Registered Plugins", func(t *testing.T) {
		expectedPlugins := []struct {
			kind     string
			provider string
		}{
			{"Infrastructure", "terraform"},
			{"Cluster", "existing"},
			{"Orchestrator", "argocd"},
			{"Observability", "prometheus"},
			{"Observability", "grafana"},
			{"DevEx", "backstage"},
		}

		registeredPlugins := pm.List()

		if len(registeredPlugins) < 5 {
			t.Errorf("Expected at least 5 registered plugins, got %d", len(registeredPlugins))
		}

		// Verify we can retrieve each expected plugin
		for _, expected := range expectedPlugins {
			_, err := pm.Get(expected.kind, expected.provider)
			if err != nil {
				t.Errorf("Expected plugin %s:%s not found: %v", expected.kind, expected.provider, err)
			}
		}
	})

	t.Run("Get Plugin by Type and Provider", func(t *testing.T) {
		p, err := pm.Get("Infrastructure", "terraform")
		if err != nil {
			t.Errorf("Failed to get terraform plugin: %v", err)
		}

		if p == nil {
			t.Error("Expected terraform plugin, got nil")
		}
	})
}

func TestPlugins_TerraformIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping terraform integration test in short mode")
	}

	pm := plugin.NewManager()
	terraformPlugin := terraform.NewPlugin()
	if err := pm.Register(terraformPlugin); err != nil {
		t.Fatalf("Failed to register terraform plugin: %v", err)
	}

	t.Run("Verify Terraform Plugin", func(t *testing.T) {
		if terraformPlugin.Type() != "Infrastructure" {
			t.Errorf("Expected type 'Infrastructure', got '%s'", terraformPlugin.Type())
		}

		if terraformPlugin.Name() != "terraform" {
			t.Errorf("Expected name 'terraform', got '%s'", terraformPlugin.Name())
		}

		// Verify plugin was registered correctly
		retrieved, err := pm.Get("Infrastructure", "terraform")
		if err != nil {
			t.Fatalf("Failed to retrieve registered terraform plugin: %v", err)
		}

		if retrieved != terraformPlugin {
			t.Error("Retrieved plugin does not match registered plugin")
		}
	})

	t.Run("Terraform Config Type", func(t *testing.T) {
		configType := terraformPlugin.ConfigType()
		if configType == nil {
			t.Error("Expected non-nil config type")
		}
		t.Logf("Terraform plugin config type: %T", configType)
	})
}

func TestPlugins_ArgoCDIntegration(t *testing.T) {
	pm := plugin.NewManager()
	argocdPlugin := argocd.NewPlugin()
	if err := pm.Register(argocdPlugin); err != nil {
		t.Fatalf("Failed to register argocd plugin: %v", err)
	}

	t.Run("Verify ArgoCD Plugin", func(t *testing.T) {
		if argocdPlugin.Type() != "Orchestrator" {
			t.Errorf("Expected type 'Orchestrator', got '%s'", argocdPlugin.Type())
		}

		if argocdPlugin.Name() != "argocd" {
			t.Errorf("Expected name 'argocd', got '%s'", argocdPlugin.Name())
		}

		// Verify plugin was registered correctly
		retrieved, err := pm.Get("Orchestrator", "argocd")
		if err != nil {
			t.Fatalf("Failed to retrieve registered argocd plugin: %v", err)
		}

		if retrieved != argocdPlugin {
			t.Error("Retrieved plugin does not match registered plugin")
		}
	})

	t.Run("ArgoCD Config Type", func(t *testing.T) {
		configType := argocdPlugin.ConfigType()
		if configType == nil {
			t.Error("Expected non-nil config type")
		}
		t.Logf("ArgoCD plugin config type: %T", configType)
	})
}

func TestPlugins_PrometheusIntegration(t *testing.T) {
	pm := plugin.NewManager()
	prometheusPlugin := prometheus.NewPlugin()
	if err := pm.Register(prometheusPlugin); err != nil {
		t.Fatalf("Failed to register prometheus plugin: %v", err)
	}

	t.Run("Verify Prometheus Plugin", func(t *testing.T) {
		if prometheusPlugin.Type() != "Observability" {
			t.Errorf("Expected type 'Observability', got '%s'", prometheusPlugin.Type())
		}

		if prometheusPlugin.Name() != "prometheus" {
			t.Errorf("Expected name 'prometheus', got '%s'", prometheusPlugin.Name())
		}

		// Verify plugin was registered correctly
		retrieved, err := pm.Get("Observability", "prometheus")
		if err != nil {
			t.Fatalf("Failed to retrieve registered prometheus plugin: %v", err)
		}

		if retrieved != prometheusPlugin {
			t.Error("Retrieved plugin does not match registered plugin")
		}
	})

	t.Run("Prometheus Config Type", func(t *testing.T) {
		configType := prometheusPlugin.ConfigType()
		if configType == nil {
			t.Error("Expected non-nil config type")
		}
		t.Logf("Prometheus plugin config type: %T", configType)
	})
}

func TestPlugins_GrafanaIntegration(t *testing.T) {
	pm := plugin.NewManager()
	grafanaPlugin := grafana.NewPlugin()
	if err := pm.Register(grafanaPlugin); err != nil {
		t.Fatalf("Failed to register grafana plugin: %v", err)
	}

	t.Run("Verify Grafana Plugin", func(t *testing.T) {
		if grafanaPlugin.Type() != "Observability" {
			t.Errorf("Expected type 'Observability', got '%s'", grafanaPlugin.Type())
		}

		if grafanaPlugin.Name() != "grafana" {
			t.Errorf("Expected name 'grafana', got '%s'", grafanaPlugin.Name())
		}

		// Verify plugin was registered correctly
		retrieved, err := pm.Get("Observability", "grafana")
		if err != nil {
			t.Fatalf("Failed to retrieve registered grafana plugin: %v", err)
		}

		if retrieved != grafanaPlugin {
			t.Error("Retrieved plugin does not match registered plugin")
		}
	})

	t.Run("Grafana Config Type", func(t *testing.T) {
		configType := grafanaPlugin.ConfigType()
		if configType == nil {
			t.Error("Expected non-nil config type")
		}
		t.Logf("Grafana plugin config type: %T", configType)
	})
}

func TestPlugins_BackstageIntegration(t *testing.T) {
	pm := plugin.NewManager()
	backstagePlugin := backstage.NewPlugin()
	if err := pm.Register(backstagePlugin); err != nil {
		t.Fatalf("Failed to register backstage plugin: %v", err)
	}

	t.Run("Verify Backstage Plugin", func(t *testing.T) {
		if backstagePlugin.Type() != "DevEx" {
			t.Errorf("Expected type 'DevEx', got '%s'", backstagePlugin.Type())
		}

		if backstagePlugin.Name() != "backstage" {
			t.Errorf("Expected name 'backstage', got '%s'", backstagePlugin.Name())
		}

		// Verify plugin was registered correctly
		retrieved, err := pm.Get("DevEx", "backstage")
		if err != nil {
			t.Fatalf("Failed to retrieve registered backstage plugin: %v", err)
		}

		if retrieved != backstagePlugin {
			t.Error("Retrieved plugin does not match registered plugin")
		}
	})

	t.Run("Backstage Config Type", func(t *testing.T) {
		configType := backstagePlugin.ConfigType()
		if configType == nil {
			t.Error("Expected non-nil config type")
		}
		t.Logf("Backstage plugin config type: %T", configType)
	})
}

func TestPlugins_ExistingClusterIntegration(t *testing.T) {
	pm := plugin.NewManager()
	existingPlugin := clusterexisting.NewPlugin()
	if err := pm.Register(existingPlugin); err != nil {
		t.Fatalf("Failed to register existing cluster plugin: %v", err)
	}

	t.Run("Verify Existing Cluster Plugin", func(t *testing.T) {
		if existingPlugin.Type() != "Cluster" {
			t.Errorf("Expected type 'Cluster', got '%s'", existingPlugin.Type())
		}

		if existingPlugin.Name() != "existing" {
			t.Errorf("Expected name 'existing', got '%s'", existingPlugin.Name())
		}

		// Verify plugin was registered correctly
		retrieved, err := pm.Get("Cluster", "existing")
		if err != nil {
			t.Fatalf("Failed to retrieve registered existing cluster plugin: %v", err)
		}

		if retrieved != existingPlugin {
			t.Error("Retrieved plugin does not match registered plugin")
		}
	})

	t.Run("Existing Cluster Config Type", func(t *testing.T) {
		configType := existingPlugin.ConfigType()
		if configType == nil {
			t.Error("Expected non-nil config type")
		}
		t.Logf("Existing cluster plugin config type: %T", configType)
	})
}

func TestPlugins_ConfigValidation(t *testing.T) {
	// Test that all plugins have proper config types
	tests := []struct {
		name   string
		plugin pkgplugin.Plugin
	}{
		{"Terraform", terraform.NewPlugin()},
		{"ArgoCD", argocd.NewPlugin()},
		{"Prometheus", prometheus.NewPlugin()},
		{"Grafana", grafana.NewPlugin()},
		{"Backstage", backstage.NewPlugin()},
		{"Existing Cluster", clusterexisting.NewPlugin()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configType := tt.plugin.ConfigType()
			if configType == nil {
				t.Errorf("%s plugin has nil config type", tt.name)
			} else {
				t.Logf("%s plugin config type: %T", tt.name, configType)
			}
		})
	}
}

func TestPlugins_EndToEndWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end plugin test in short mode")
	}

	// This test simulates a complete platform deployment workflow
	pm := plugin.NewManager()

	// Define all plugins needed for a complete platform
	pluginSpecs := []struct {
		name   string
		kind   string
		plugin pkgplugin.Plugin
	}{
		{"terraform", "Infrastructure", terraform.NewPlugin()},
		{"existing", "Cluster", clusterexisting.NewPlugin()},
		{"argocd", "Orchestrator", argocd.NewPlugin()},
		{"prometheus", "Observability", prometheus.NewPlugin()},
		{"grafana", "Observability", grafana.NewPlugin()},
		{"backstage", "DevEx", backstage.NewPlugin()},
	}

	t.Run("Register All Platform Components", func(t *testing.T) {
		for _, spec := range pluginSpecs {
			if err := pm.Register(spec.plugin); err != nil {
				t.Fatalf("Failed to register %s plugin: %v", spec.name, err)
			}
			t.Logf("Registered %s:%s plugin", spec.kind, spec.name)
		}

		// Verify all plugins registered
		registeredPlugins := pm.List()
		if len(registeredPlugins) != len(pluginSpecs) {
			t.Errorf("Expected %d registered plugins, got %d", len(pluginSpecs), len(registeredPlugins))
		}
	})

	t.Run("Retrieve Platform Stack Components", func(t *testing.T) {
		// Verify we can retrieve each component of the platform stack
		t.Log("Step 1: Verify infrastructure plugin...")
		infraPlugin, err := pm.Get("Infrastructure", "terraform")
		if err != nil {
			t.Errorf("Failed to get infrastructure plugin: %v", err)
		} else if infraPlugin == nil {
			t.Error("Infrastructure plugin is nil")
		}

		t.Log("Step 2: Verify orchestrator plugin...")
		orchPlugin, err := pm.Get("Orchestrator", "argocd")
		if err != nil {
			t.Errorf("Failed to get orchestrator plugin: %v", err)
		} else if orchPlugin == nil {
			t.Error("Orchestrator plugin is nil")
		}

		t.Log("Step 3: Verify observability plugins...")
		prometheusPlugin, err := pm.Get("Observability", "prometheus")
		if err != nil {
			t.Errorf("Failed to get prometheus plugin: %v", err)
		} else if prometheusPlugin == nil {
			t.Error("Prometheus plugin is nil")
		}

		grafanaPlugin, err := pm.Get("Observability", "grafana")
		if err != nil {
			t.Errorf("Failed to get grafana plugin: %v", err)
		} else if grafanaPlugin == nil {
			t.Error("Grafana plugin is nil")
		}

		t.Log("Step 4: Verify DevEx plugin...")
		devexPlugin, err := pm.Get("DevEx", "backstage")
		if err != nil {
			t.Errorf("Failed to get backstage plugin: %v", err)
		} else if devexPlugin == nil {
			t.Error("Backstage plugin is nil")
		}

		t.Log("Complete platform stack verified successfully")
	})

	t.Run("Verify Plugin Metadata", func(t *testing.T) {
		// Verify each plugin has proper metadata
		for _, spec := range pluginSpecs {
			p, err := pm.Get(spec.kind, spec.name)
			if err != nil {
				t.Errorf("Failed to get %s plugin: %v", spec.name, err)
				continue
			}

			if p.Type() != spec.kind {
				t.Errorf("Plugin %s has wrong type: expected %s, got %s", spec.name, spec.kind, p.Type())
			}

			if p.Name() != spec.name {
				t.Errorf("Plugin has wrong name: expected %s, got %s", spec.name, p.Name())
			}

			if p.ConfigType() == nil {
				t.Errorf("Plugin %s has nil config type", spec.name)
			}
		}
	})
}
