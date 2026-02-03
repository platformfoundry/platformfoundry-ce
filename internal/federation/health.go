package federation

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// DefaultHealthMonitor implements health monitoring for federated clusters
type DefaultHealthMonitor struct {
	clusters      map[string]*monitoredCluster
	callback      HealthCallback
	httpClient    *http.Client
	checkInterval time.Duration
	mu            sync.RWMutex
	stopCh        chan struct{}
	running       bool
}

type monitoredCluster struct {
	cluster *Cluster
	health  *ClusterHealth
	stopCh  chan struct{}
	running bool
}

// HealthMonitorConfig configures the health monitor
type HealthMonitorConfig struct {
	CheckInterval    time.Duration `json:"checkInterval"`
	Timeout          time.Duration `json:"timeout"`
	FailureThreshold int           `json:"failureThreshold"`
	SuccessThreshold int           `json:"successThreshold"`
}

// DefaultHealthMonitorConfig returns default health monitor configuration
func DefaultHealthMonitorConfig() *HealthMonitorConfig {
	return &HealthMonitorConfig{
		CheckInterval:    30 * time.Second,
		Timeout:          10 * time.Second,
		FailureThreshold: 3,
		SuccessThreshold: 2,
	}
}

// NewDefaultHealthMonitor creates a new health monitor
func NewDefaultHealthMonitor(config *HealthMonitorConfig) *DefaultHealthMonitor {
	if config == nil {
		config = DefaultHealthMonitorConfig()
	}

	return &DefaultHealthMonitor{
		clusters:      make(map[string]*monitoredCluster),
		checkInterval: config.CheckInterval,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		stopCh: make(chan struct{}),
	}
}

// StartMonitoring starts monitoring a cluster
func (m *DefaultHealthMonitor) StartMonitoring(cluster *Cluster) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.clusters[cluster.Name]; exists {
		return fmt.Errorf("cluster %s is already being monitored", cluster.Name)
	}

	mc := &monitoredCluster{
		cluster: cluster,
		health: &ClusterHealth{
			Status:    ClusterStatusUnknown,
			CheckedAt: time.Now(),
		},
		stopCh:  make(chan struct{}),
		running: true,
	}

	m.clusters[cluster.Name] = mc

	// Start monitoring goroutine
	go m.monitorCluster(mc)

	return nil
}

// StopMonitoring stops monitoring a cluster
func (m *DefaultHealthMonitor) StopMonitoring(clusterName string) error {
	m.mu.Lock()
	mc, exists := m.clusters[clusterName]
	if !exists {
		m.mu.Unlock()
		return ErrClusterNotFound
	}

	if mc.running {
		close(mc.stopCh)
		mc.running = false
	}
	delete(m.clusters, clusterName)
	m.mu.Unlock()

	return nil
}

// GetHealth returns the current health of a cluster
func (m *DefaultHealthMonitor) GetHealth(clusterName string) (*ClusterHealth, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mc, exists := m.clusters[clusterName]
	if !exists {
		return nil, ErrClusterNotFound
	}

	return mc.health, nil
}

// SetHealthCallback sets the callback for health changes
func (m *DefaultHealthMonitor) SetHealthCallback(callback HealthCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callback = callback
}

// monitorCluster runs the monitoring loop for a cluster
func (m *DefaultHealthMonitor) monitorCluster(mc *monitoredCluster) {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	consecutiveFailures := 0
	consecutiveSuccesses := 0

	for {
		select {
		case <-mc.stopCh:
			return
		case <-ticker.C:
			health := m.checkClusterHealth(mc.cluster)
			oldStatus := mc.health.Status

			if health.Status == ClusterStatusHealthy {
				consecutiveSuccesses++
				consecutiveFailures = 0
			} else {
				consecutiveFailures++
				consecutiveSuccesses = 0
			}

			// Update health status with hysteresis
			if consecutiveFailures >= 3 {
				health.Status = ClusterStatusUnhealthy
			} else if consecutiveFailures >= 1 {
				health.Status = ClusterStatusDegraded
			}

			m.mu.Lock()
			mc.health = health
			m.mu.Unlock()

			// Notify callback if status changed
			if oldStatus != health.Status && m.callback != nil {
				m.callback(mc.cluster.Name, health)
			}
		}
	}
}

// checkClusterHealth performs a health check on a cluster
func (m *DefaultHealthMonitor) checkClusterHealth(cluster *Cluster) *ClusterHealth {
	start := time.Now()

	health := &ClusterHealth{
		Status:    ClusterStatusHealthy,
		CheckedAt: start,
	}

	// Check API server connectivity
	endpoint := cluster.Endpoint
	if endpoint == "" {
		health.Status = ClusterStatusUnknown
		health.Message = "No endpoint configured"
		return health
	}

	// Try to reach the health endpoint
	healthURL := fmt.Sprintf("%s/healthz", endpoint)
	resp, err := m.httpClient.Get(healthURL)
	if err != nil {
		health.Status = ClusterStatusUnhealthy
		health.Message = fmt.Sprintf("Health check failed: %v", err)
		health.Latency = time.Since(start)
		return health
	}
	defer resp.Body.Close()

	health.Latency = time.Since(start)

	if resp.StatusCode != http.StatusOK {
		health.Status = ClusterStatusDegraded
		health.Message = fmt.Sprintf("Health endpoint returned %d", resp.StatusCode)
		return health
	}

	health.Message = "Cluster is healthy"
	return health
}

// GetAllHealth returns health status for all monitored clusters
func (m *DefaultHealthMonitor) GetAllHealth() map[string]*ClusterHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*ClusterHealth)
	for name, mc := range m.clusters {
		result[name] = mc.health
	}
	return result
}

// Stop stops the health monitor
func (m *DefaultHealthMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, mc := range m.clusters {
		if mc.running {
			close(mc.stopCh)
			mc.running = false
		}
	}
	m.clusters = make(map[string]*monitoredCluster)
}
