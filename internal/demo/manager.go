package demo

import (
	"fmt"
	"strings"
	"time"
)

// Config represents demo configuration
type Config struct {
	ClusterName string
	Components  []string
	QuickMode   bool
}

// Manager manages the demo lifecycle
type Manager struct {
	config     *Config
	cluster    *KindCluster
	portForwards []*PortForward
}

// PortForward represents a port forwarding session
type PortForward struct {
	Service    string
	Namespace  string
	LocalPort  int
	TargetPort int
	URL        string
}

// NewManager creates a new demo manager
func NewManager(config *Config) *Manager {
	return &Manager{
		config:       config,
		portForwards: make([]*PortForward, 0),
	}
}

// Setup sets up the complete demo environment
func (m *Manager) Setup() error {
	// 1. Create kind cluster
	if err := m.createCluster(); err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	// 2. Wait for cluster to be ready
	if err := m.waitForCluster(); err != nil {
		return fmt.Errorf("cluster not ready: %w", err)
	}

	// 3. Install components
	for _, component := range m.config.Components {
		if err := m.installComponent(component); err != nil {
			return fmt.Errorf("failed to install %s: %w", component, err)
		}
	}

	// 4. Configure integrations
	if err := m.configureIntegrations(); err != nil {
		return fmt.Errorf("failed to configure integrations: %w", err)
	}

	// 5. Setup port forwarding
	if err := m.setupPortForwarding(); err != nil {
		return fmt.Errorf("failed to setup port forwarding: %w", err)
	}

	return nil
}

func (m *Manager) createCluster() error {
	fmt.Println("📦 Creating kind cluster...")

	m.cluster = NewKindCluster(m.config.ClusterName)

	if err := m.cluster.Create(); err != nil {
		return err
	}

	fmt.Println("✓ Kind cluster created")
	return nil
}

func (m *Manager) waitForCluster() error {
	fmt.Println("⏳ Waiting for cluster to be ready...")

	if err := m.cluster.WaitReady(); err != nil {
		return err
	}

	fmt.Println("✓ Cluster ready")
	return nil
}

func (m *Manager) installComponent(component string) error {
	fmt.Printf("📦 Installing %s...\n", component)

	var err error
	switch component {
	case "prometheus":
		err = m.installPrometheus()
	case "grafana":
		err = m.installGrafana()
	case "argocd":
		err = m.installArgoCD()
	case "backstage":
		err = m.installBackstage()
	default:
		return fmt.Errorf("unknown component: %s", component)
	}

	if err != nil {
		return err
	}

	fmt.Printf("✓ %s installed\n", component)
	return nil
}

func (m *Manager) installPrometheus() error {
	// Create namespace
	if err := m.cluster.CreateNamespace("monitoring"); err != nil {
		return err
	}

	// Install using kubectl (simplified version)
	manifest := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prometheus
  template:
    metadata:
      labels:
        app: prometheus
    spec:
      containers:
      - name: prometheus
        image: prom/prometheus:latest
        ports:
        - containerPort: 9090
        args:
        - '--config.file=/etc/prometheus/prometheus.yml'
        - '--storage.tsdb.path=/prometheus'
        - '--web.console.libraries=/usr/share/prometheus/console_libraries'
        - '--web.console.templates=/usr/share/prometheus/consoles'
---
apiVersion: v1
kind: Service
metadata:
  name: prometheus
  namespace: monitoring
spec:
  selector:
    app: prometheus
  ports:
  - port: 9090
    targetPort: 9090
`

	return m.cluster.ApplyManifest(manifest)
}

func (m *Manager) installGrafana() error {
	// Create namespace if not exists
	m.cluster.CreateNamespace("monitoring")

	manifest := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: grafana
  template:
    metadata:
      labels:
        app: grafana
    spec:
      containers:
      - name: grafana
        image: grafana/grafana:latest
        ports:
        - containerPort: 3000
        env:
        - name: GF_SECURITY_ADMIN_PASSWORD
          value: admin
        - name: GF_SECURITY_ADMIN_USER
          value: admin
---
apiVersion: v1
kind: Service
metadata:
  name: grafana
  namespace: monitoring
spec:
  selector:
    app: grafana
  ports:
  - port: 3000
    targetPort: 3000
`

	return m.cluster.ApplyManifest(manifest)
}

func (m *Manager) installArgoCD() error {
	// Create namespace
	if err := m.cluster.CreateNamespace("argocd"); err != nil {
		return err
	}

	// For demo, use simplified ArgoCD deployment
	manifest := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: argocd-server
  namespace: argocd
spec:
  replicas: 1
  selector:
    matchLabels:
      app: argocd-server
  template:
    metadata:
      labels:
        app: argocd-server
    spec:
      containers:
      - name: argocd-server
        image: argoproj/argocd:latest
        command:
        - argocd-server
        - --insecure
        ports:
        - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: argocd-server
  namespace: argocd
spec:
  selector:
    app: argocd-server
  ports:
  - port: 8080
    targetPort: 8080
`

	return m.cluster.ApplyManifest(manifest)
}

func (m *Manager) installBackstage() error {
	// Create namespace
	if err := m.cluster.CreateNamespace("backstage"); err != nil {
		return err
	}

	// For demo, use simplified Backstage deployment
	manifest := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backstage
  namespace: backstage
spec:
  replicas: 1
  selector:
    matchLabels:
      app: backstage
  template:
    metadata:
      labels:
        app: backstage
    spec:
      containers:
      - name: backstage
        image: spotify/backstage:latest
        ports:
        - containerPort: 7007
---
apiVersion: v1
kind: Service
metadata:
  name: backstage
  namespace: backstage
spec:
  selector:
    app: backstage
  ports:
  - port: 7007
    targetPort: 7007
`

	return m.cluster.ApplyManifest(manifest)
}

func (m *Manager) configureIntegrations() error {
	fmt.Println("🔗 Configuring integrations...")

	// Configure Grafana → Prometheus datasource
	if m.hasComponent("grafana") && m.hasComponent("prometheus") {
		if err := m.configureGrafanaPrometheus(); err != nil {
			fmt.Printf("⚠ Warning: Could not configure Grafana-Prometheus integration: %v\n", err)
		} else {
			fmt.Println("  ✓ Grafana → Prometheus datasource configured")
		}
	}

	fmt.Println("✓ Integrations configured")
	return nil
}

func (m *Manager) configureGrafanaPrometheus() error {
	// In a real implementation, this would configure Grafana via API
	// For demo purposes, we'll create a ConfigMap
	manifest := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-datasources
  namespace: monitoring
data:
  prometheus.yaml: |
    apiVersion: 1
    datasources:
    - name: Prometheus
      type: prometheus
      access: proxy
      url: http://prometheus:9090
      isDefault: true
`

	return m.cluster.ApplyManifest(manifest)
}

func (m *Manager) setupPortForwarding() error {
	fmt.Println("🌐 Setting up access URLs...")

	// Wait a bit for pods to be ready
	time.Sleep(5 * time.Second)

	portMap := map[string]*PortForward{
		"prometheus": {
			Service:    "prometheus",
			Namespace:  "monitoring",
			LocalPort:  9090,
			TargetPort: 9090,
			URL:        "http://localhost:9090",
		},
		"grafana": {
			Service:    "grafana",
			Namespace:  "monitoring",
			LocalPort:  3000,
			TargetPort: 3000,
			URL:        "http://localhost:3000",
		},
		"argocd": {
			Service:    "argocd-server",
			Namespace:  "argocd",
			LocalPort:  8080,
			TargetPort: 8080,
			URL:        "http://localhost:8080",
		},
		"backstage": {
			Service:    "backstage",
			Namespace:  "backstage",
			LocalPort:  7007,
			TargetPort: 7007,
			URL:        "http://localhost:7007",
		},
	}

	for _, component := range m.config.Components {
		if pf, ok := portMap[component]; ok {
			m.portForwards = append(m.portForwards, pf)
		}
	}

	fmt.Println("✓ Access URLs configured")
	return nil
}

func (m *Manager) hasComponent(name string) bool {
	for _, c := range m.config.Components {
		if c == name {
			return true
		}
	}
	return false
}

// ShowAccessInfo displays access information for all components
func (m *Manager) ShowAccessInfo() {
	fmt.Println("🌐 Access your platform:")
	fmt.Println("")

	for _, pf := range m.portForwards {
		var emoji string
		switch {
		case strings.Contains(pf.Service, "backstage"):
			emoji = "🎯"
		case strings.Contains(pf.Service, "grafana"):
			emoji = "📊"
		case strings.Contains(pf.Service, "argocd"):
			emoji = "🚀"
		case strings.Contains(pf.Service, "prometheus"):
			emoji = "📈"
		default:
			emoji = "🔗"
		}

		componentName := strings.Title(strings.ReplaceAll(pf.Service, "-server", ""))
		fmt.Printf("  %s %-12s %s\n", emoji, componentName+":", pf.URL)

		// Show default credentials if applicable
		if strings.Contains(pf.Service, "grafana") {
			fmt.Printf("     %-12s User: admin / Password: admin\n", "")
		}
		if strings.Contains(pf.Service, "argocd") {
			fmt.Printf("     %-12s User: admin / Password: (run: kubectl -n argocd get secret argocd-initial-admin-secret)\n", "")
		}
	}

	fmt.Println("")
	fmt.Println("💡 Note: Services are accessible via kubectl port-forward")
	fmt.Println("   Run in separate terminals:")
	for _, pf := range m.portForwards {
		fmt.Printf("   kubectl port-forward -n %s svc/%s %d:%d\n",
			pf.Namespace, pf.Service, pf.LocalPort, pf.TargetPort)
	}
}

// Cleanup removes all demo resources
func (m *Manager) Cleanup() error {
	if m.cluster != nil {
		return m.cluster.Delete()
	}
	return nil
}
