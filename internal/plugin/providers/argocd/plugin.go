package argocd

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/platformfoundry/pf-ce/pkg/plugin"
)

// Plugin implements GitOps deployments via ArgoCD
type Plugin struct {
	name    string
	version string

	client      *http.Client
	serverURL   string
	authToken   string
	initialized bool
	mu          sync.RWMutex

	// Track applications
	apps map[string]*ApplicationInfo
}

// ApplicationInfo tracks deployed ArgoCD applications
type ApplicationInfo struct {
	Name       string
	Namespace  string
	Project    string
	RepoURL    string
	Path       string
	Revision   string
	SyncStatus string
	Health     string
	CreatedAt  time.Time
}

// Config represents ArgoCD plugin configuration
type Config struct {
	ServerURL   string            `yaml:"serverUrl" json:"serverUrl"`
	AuthToken   string            `yaml:"authToken,omitempty" json:"authToken,omitempty"`
	Username    string            `yaml:"username,omitempty" json:"username,omitempty"`
	Password    string            `yaml:"password,omitempty" json:"password,omitempty"`
	Insecure    bool              `yaml:"insecure,omitempty" json:"insecure,omitempty"`
	Application ApplicationConfig `yaml:"application" json:"application"`
}

// ApplicationConfig represents ArgoCD Application spec
type ApplicationConfig struct {
	Name            string            `yaml:"name" json:"name"`
	Namespace       string            `yaml:"namespace" json:"namespace"`
	Project         string            `yaml:"project,omitempty" json:"project,omitempty"`
	RepoURL         string            `yaml:"repoUrl" json:"repoUrl"`
	Path            string            `yaml:"path" json:"path"`
	TargetRevision  string            `yaml:"targetRevision,omitempty" json:"targetRevision,omitempty"`
	DestServer      string            `yaml:"destServer,omitempty" json:"destServer,omitempty"`
	DestNamespace   string            `yaml:"destNamespace,omitempty" json:"destNamespace,omitempty"`
	SyncPolicy      *SyncPolicy       `yaml:"syncPolicy,omitempty" json:"syncPolicy,omitempty"`
	Helm            *HelmConfig       `yaml:"helm,omitempty" json:"helm,omitempty"`
	Kustomize       *KustomizeConfig  `yaml:"kustomize,omitempty" json:"kustomize,omitempty"`
	Labels          map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations     map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

// SyncPolicy defines auto-sync behavior
type SyncPolicy struct {
	Automated   *AutomatedPolicy `yaml:"automated,omitempty" json:"automated,omitempty"`
	SyncOptions []string         `yaml:"syncOptions,omitempty" json:"syncOptions,omitempty"`
	Retry       *RetryPolicy     `yaml:"retry,omitempty" json:"retry,omitempty"`
}

// AutomatedPolicy controls automatic syncing
type AutomatedPolicy struct {
	Prune      bool `yaml:"prune" json:"prune"`
	SelfHeal   bool `yaml:"selfHeal" json:"selfHeal"`
	AllowEmpty bool `yaml:"allowEmpty,omitempty" json:"allowEmpty,omitempty"`
}

// RetryPolicy controls sync retry behavior
type RetryPolicy struct {
	Limit       int `yaml:"limit,omitempty" json:"limit,omitempty"`
	BackoffSecs int `yaml:"backoff,omitempty" json:"backoff,omitempty"`
}

// HelmConfig for Helm-based applications
type HelmConfig struct {
	ValueFiles []string               `yaml:"valueFiles,omitempty" json:"valueFiles,omitempty"`
	Values     map[string]interface{} `yaml:"values,omitempty" json:"values,omitempty"`
	Parameters []HelmParameter        `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	ReleaseName string               `yaml:"releaseName,omitempty" json:"releaseName,omitempty"`
}

// HelmParameter represents a Helm parameter
type HelmParameter struct {
	Name  string `yaml:"name" json:"name"`
	Value string `yaml:"value" json:"value"`
}

// KustomizeConfig for Kustomize-based applications
type KustomizeConfig struct {
	NamePrefix string            `yaml:"namePrefix,omitempty" json:"namePrefix,omitempty"`
	NameSuffix string            `yaml:"nameSuffix,omitempty" json:"nameSuffix,omitempty"`
	Images     []string          `yaml:"images,omitempty" json:"images,omitempty"`
	CommonLabels map[string]string `yaml:"commonLabels,omitempty" json:"commonLabels,omitempty"`
}

// ArgoCD API types
type argoApplication struct {
	APIVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"`
	Metadata   argoMetadata      `json:"metadata"`
	Spec       argoAppSpec       `json:"spec"`
	Status     *argoAppStatus    `json:"status,omitempty"`
}

type argoMetadata struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type argoAppSpec struct {
	Project     string         `json:"project"`
	Source      argoSource     `json:"source"`
	Destination argoDestination `json:"destination"`
	SyncPolicy  *argoSyncPolicy `json:"syncPolicy,omitempty"`
}

type argoSource struct {
	RepoURL        string          `json:"repoURL"`
	Path           string          `json:"path,omitempty"`
	TargetRevision string          `json:"targetRevision,omitempty"`
	Helm           *argoHelm       `json:"helm,omitempty"`
	Kustomize      *argoKustomize  `json:"kustomize,omitempty"`
}

type argoHelm struct {
	ValueFiles  []string          `json:"valueFiles,omitempty"`
	Values      string            `json:"values,omitempty"`
	Parameters  []argoHelmParam   `json:"parameters,omitempty"`
	ReleaseName string            `json:"releaseName,omitempty"`
}

type argoHelmParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type argoKustomize struct {
	NamePrefix   string            `json:"namePrefix,omitempty"`
	NameSuffix   string            `json:"nameSuffix,omitempty"`
	Images       []string          `json:"images,omitempty"`
	CommonLabels map[string]string `json:"commonLabels,omitempty"`
}

type argoDestination struct {
	Server    string `json:"server"`
	Namespace string `json:"namespace"`
}

type argoSyncPolicy struct {
	Automated   *argoAutomated `json:"automated,omitempty"`
	SyncOptions []string       `json:"syncOptions,omitempty"`
	Retry       *argoRetry     `json:"retry,omitempty"`
}

type argoAutomated struct {
	Prune      bool `json:"prune"`
	SelfHeal   bool `json:"selfHeal"`
	AllowEmpty bool `json:"allowEmpty,omitempty"`
}

type argoRetry struct {
	Limit   int             `json:"limit,omitempty"`
	Backoff *argoBackoff    `json:"backoff,omitempty"`
}

type argoBackoff struct {
	Duration    string `json:"duration,omitempty"`
	Factor      int    `json:"factor,omitempty"`
	MaxDuration string `json:"maxDuration,omitempty"`
}

type argoAppStatus struct {
	Sync   argoSyncStatus   `json:"sync"`
	Health argoHealthStatus `json:"health"`
	Resources []argoResource `json:"resources,omitempty"`
}

type argoSyncStatus struct {
	Status   string `json:"status"`
	Revision string `json:"revision,omitempty"`
}

type argoHealthStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type argoResource struct {
	Group     string `json:"group,omitempty"`
	Version   string `json:"version"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
	Status    string `json:"status,omitempty"`
	Health    *argoHealthStatus `json:"health,omitempty"`
}

type argoSyncRequest struct {
	Revision string   `json:"revision,omitempty"`
	Prune    bool     `json:"prune"`
	DryRun   bool     `json:"dryRun"`
	Resources []argoSyncResource `json:"resources,omitempty"`
}

type argoSyncResource struct {
	Group     string `json:"group"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type argoSessionRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type argoSessionResponse struct {
	Token string `json:"token"`
}

// New creates a new ArgoCD plugin
func New() *Plugin {
	return &Plugin{
		name:    "argocd",
		version: "1.0.0",
		apps:    make(map[string]*ApplicationInfo),
	}
}

func (p *Plugin) Name() string    { return p.name }
func (p *Plugin) Type() string    { return "Orchestrator" }
func (p *Plugin) Version() string { return p.version }

func (p *Plugin) ConfigType() interface{} {
	return Config{}
}

// Initialize sets up the ArgoCD client
func (p *Plugin) Initialize(ctx context.Context, cfg *Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return nil
	}

	transport := &http.Transport{}
	if cfg.Insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	p.client = &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	p.serverURL = strings.TrimSuffix(cfg.ServerURL, "/")

	// Authenticate if credentials provided
	if cfg.AuthToken != "" {
		p.authToken = cfg.AuthToken
	} else if cfg.Username != "" && cfg.Password != "" {
		token, err := p.authenticate(ctx, cfg.Username, cfg.Password)
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
		p.authToken = token
	}

	p.initialized = true
	return nil
}

func (p *Plugin) authenticate(ctx context.Context, username, password string) (string, error) {
	reqBody := argoSessionRequest{
		Username: username,
		Password: password,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.serverURL+"/api/v1/session", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("authentication failed with status %d", resp.StatusCode)
	}

	var session argoSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return "", err
	}

	return session.Token, nil
}

func (p *Plugin) Validate(spec map[string]interface{}) error {
	cfg, err := p.parseConfig(spec)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	if cfg.ServerURL == "" {
		return fmt.Errorf("serverUrl is required")
	}

	app := cfg.Application
	if app.Name == "" {
		return fmt.Errorf("application.name is required")
	}
	if app.RepoURL == "" {
		return fmt.Errorf("application.repoUrl is required")
	}
	if app.Path == "" {
		return fmt.Errorf("application.path is required")
	}

	return nil
}

func (p *Plugin) Plan(spec map[string]interface{}) (*plugin.Plan, error) {
	cfg, err := p.parseConfig(spec)
	if err != nil {
		return nil, err
	}

	plan := &plugin.Plan{
		Actions: make([]string, 0),
		Changes: make(map[string]string),
	}

	ctx := context.Background()
	if !p.initialized {
		if err := p.Initialize(ctx, cfg); err != nil {
			return nil, err
		}
	}

	// Check if application exists
	existing, err := p.getApplication(ctx, cfg.Application.Name)
	if err != nil && !isNotFound(err) {
		return nil, err
	}

	if existing == nil {
		plan.Actions = append(plan.Actions, fmt.Sprintf("create Application %s", cfg.Application.Name))
		plan.Changes[cfg.Application.Name] = "create"
	} else {
		// Check for differences
		if existing.Spec.Source.RepoURL != cfg.Application.RepoURL ||
			existing.Spec.Source.Path != cfg.Application.Path ||
			existing.Spec.Source.TargetRevision != cfg.Application.TargetRevision {
			plan.Actions = append(plan.Actions, fmt.Sprintf("update Application %s", cfg.Application.Name))
			plan.Changes[cfg.Application.Name] = "update"
		} else {
			plan.Actions = append(plan.Actions, fmt.Sprintf("sync Application %s (no changes)", cfg.Application.Name))
			plan.Changes[cfg.Application.Name] = "sync"
		}
	}

	return plan, nil
}

func (p *Plugin) Apply(spec map[string]interface{}) (*plugin.Result, error) {
	cfg, err := p.parseConfig(spec)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	if !p.initialized {
		if err := p.Initialize(ctx, cfg); err != nil {
			return nil, err
		}
	}

	app := p.buildApplication(cfg)

	// Check if application exists
	existing, err := p.getApplication(ctx, cfg.Application.Name)
	if err != nil && !isNotFound(err) {
		return nil, err
	}

	var action string
	if existing == nil {
		// Create application
		if err := p.createApplication(ctx, app); err != nil {
			return &plugin.Result{
				Status:  "failed",
				Message: fmt.Sprintf("failed to create application: %v", err),
			}, err
		}
		action = "created"
	} else {
		// Update application
		if err := p.updateApplication(ctx, app); err != nil {
			return &plugin.Result{
				Status:  "failed",
				Message: fmt.Sprintf("failed to update application: %v", err),
			}, err
		}
		action = "updated"
	}

	// Trigger sync
	if err := p.syncApplication(ctx, cfg.Application.Name); err != nil {
		return &plugin.Result{
			Status:  "partial",
			Message: fmt.Sprintf("application %s but sync failed: %v", action, err),
		}, nil
	}

	// Wait for sync to complete
	status, err := p.waitForSync(ctx, cfg.Application.Name, 5*time.Minute)
	if err != nil {
		return &plugin.Result{
			Status:  "partial",
			Message: fmt.Sprintf("application %s but sync status unknown: %v", action, err),
		}, nil
	}

	// Track application
	p.mu.Lock()
	p.apps[cfg.Application.Name] = &ApplicationInfo{
		Name:       cfg.Application.Name,
		Namespace:  cfg.Application.Namespace,
		Project:    cfg.Application.Project,
		RepoURL:    cfg.Application.RepoURL,
		Path:       cfg.Application.Path,
		Revision:   cfg.Application.TargetRevision,
		SyncStatus: status.Sync.Status,
		Health:     status.Health.Status,
		CreatedAt:  time.Now(),
	}
	p.mu.Unlock()

	return &plugin.Result{
		Status:    "success",
		Message:   fmt.Sprintf("Application %s %s and synced successfully", cfg.Application.Name, action),
		Resources: []string{fmt.Sprintf("Application:%s/%s", cfg.Application.Namespace, cfg.Application.Name)},
		Outputs: map[string]string{
			"name":       cfg.Application.Name,
			"namespace":  cfg.Application.Namespace,
			"syncStatus": status.Sync.Status,
			"health":     status.Health.Status,
			"revision":   status.Sync.Revision,
		},
	}, nil
}

func (p *Plugin) Delete(name string) error {
	p.mu.RLock()
	if !p.initialized {
		p.mu.RUnlock()
		return fmt.Errorf("plugin not initialized")
	}
	p.mu.RUnlock()

	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, "DELETE",
		fmt.Sprintf("%s/api/v1/applications/%s", p.serverURL, name), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.authToken)

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed with status %d: %s", resp.StatusCode, string(body))
	}

	p.mu.Lock()
	delete(p.apps, name)
	p.mu.Unlock()

	return nil
}

func (p *Plugin) Status(name string) (*plugin.Status, error) {
	p.mu.RLock()
	if !p.initialized {
		p.mu.RUnlock()
		return &plugin.Status{
			State:   "unknown",
			Ready:   false,
			Message: "Plugin not initialized",
		}, nil
	}
	p.mu.RUnlock()

	ctx := context.Background()
	app, err := p.getApplication(ctx, name)
	if err != nil {
		if isNotFound(err) {
			return &plugin.Status{
				State:   "deleted",
				Ready:   false,
				Message: "Application not found",
			}, nil
		}
		return nil, err
	}

	if app.Status == nil {
		return &plugin.Status{
			State:   "unknown",
			Ready:   false,
			Message: "No status available",
		}, nil
	}

	ready := app.Status.Sync.Status == "Synced" && app.Status.Health.Status == "Healthy"
	state := "running"
	if ready {
		state = "ready"
	} else if app.Status.Health.Status == "Degraded" || app.Status.Health.Status == "Unhealthy" {
		state = "degraded"
	}

	details := map[string]string{
		"syncStatus":    app.Status.Sync.Status,
		"healthStatus":  app.Status.Health.Status,
		"revision":      app.Status.Sync.Revision,
		"resourceCount": fmt.Sprintf("%d", len(app.Status.Resources)),
	}

	return &plugin.Status{
		State:   state,
		Ready:   ready,
		Message: fmt.Sprintf("Sync: %s, Health: %s", app.Status.Sync.Status, app.Status.Health.Status),
		Details: details,
	}, nil
}

func (p *Plugin) getApplication(ctx context.Context, name string) (*argoApplication, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/api/v1/applications/%s", p.serverURL, name), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.authToken)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, &notFoundError{name: name}
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get application failed with status %d: %s", resp.StatusCode, string(body))
	}

	var app argoApplication
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, err
	}

	return &app, nil
}

func (p *Plugin) createApplication(ctx context.Context, app *argoApplication) error {
	body, err := json.Marshal(app)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		p.serverURL+"/api/v1/applications", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create application failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (p *Plugin) updateApplication(ctx context.Context, app *argoApplication) error {
	body, err := json.Marshal(app)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT",
		fmt.Sprintf("%s/api/v1/applications/%s", p.serverURL, app.Metadata.Name), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update application failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (p *Plugin) syncApplication(ctx context.Context, name string) error {
	syncReq := argoSyncRequest{
		Prune:  true,
		DryRun: false,
	}

	body, err := json.Marshal(syncReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/v1/applications/%s/sync", p.serverURL, name), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sync failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (p *Plugin) waitForSync(ctx context.Context, name string, timeout time.Duration) (*argoAppStatus, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("timeout waiting for sync")
			}

			app, err := p.getApplication(ctx, name)
			if err != nil {
				return nil, err
			}

			if app.Status == nil {
				continue
			}

			// Check if sync is complete
			if app.Status.Sync.Status == "Synced" || app.Status.Sync.Status == "OutOfSync" {
				if app.Status.Health.Status != "Progressing" {
					return app.Status, nil
				}
			}
		}
	}
}

func (p *Plugin) buildApplication(cfg *Config) *argoApplication {
	app := &argoApplication{
		APIVersion: "argoproj.io/v1alpha1",
		Kind:       "Application",
		Metadata: argoMetadata{
			Name:        cfg.Application.Name,
			Namespace:   cfg.Application.Namespace,
			Labels:      cfg.Application.Labels,
			Annotations: cfg.Application.Annotations,
		},
		Spec: argoAppSpec{
			Project: cfg.Application.Project,
			Source: argoSource{
				RepoURL:        cfg.Application.RepoURL,
				Path:           cfg.Application.Path,
				TargetRevision: cfg.Application.TargetRevision,
			},
			Destination: argoDestination{
				Server:    cfg.Application.DestServer,
				Namespace: cfg.Application.DestNamespace,
			},
		},
	}

	if app.Spec.Project == "" {
		app.Spec.Project = "default"
	}
	if app.Metadata.Namespace == "" {
		app.Metadata.Namespace = "argocd"
	}
	if app.Spec.Destination.Server == "" {
		app.Spec.Destination.Server = "https://kubernetes.default.svc"
	}
	if app.Spec.Destination.Namespace == "" {
		app.Spec.Destination.Namespace = "default"
	}
	if app.Spec.Source.TargetRevision == "" {
		app.Spec.Source.TargetRevision = "HEAD"
	}

	// Add managed-by label
	if app.Metadata.Labels == nil {
		app.Metadata.Labels = make(map[string]string)
	}
	app.Metadata.Labels["managed-by"] = "platformfoundry"

	// Add Helm config if specified
	if cfg.Application.Helm != nil {
		app.Spec.Source.Helm = &argoHelm{
			ValueFiles:  cfg.Application.Helm.ValueFiles,
			ReleaseName: cfg.Application.Helm.ReleaseName,
		}
		if len(cfg.Application.Helm.Parameters) > 0 {
			for _, param := range cfg.Application.Helm.Parameters {
				app.Spec.Source.Helm.Parameters = append(app.Spec.Source.Helm.Parameters, argoHelmParam{
					Name:  param.Name,
					Value: param.Value,
				})
			}
		}
		if cfg.Application.Helm.Values != nil {
			valuesBytes, _ := json.Marshal(cfg.Application.Helm.Values)
			app.Spec.Source.Helm.Values = string(valuesBytes)
		}
	}

	// Add Kustomize config if specified
	if cfg.Application.Kustomize != nil {
		app.Spec.Source.Kustomize = &argoKustomize{
			NamePrefix:   cfg.Application.Kustomize.NamePrefix,
			NameSuffix:   cfg.Application.Kustomize.NameSuffix,
			Images:       cfg.Application.Kustomize.Images,
			CommonLabels: cfg.Application.Kustomize.CommonLabels,
		}
	}

	// Add sync policy if specified
	if cfg.Application.SyncPolicy != nil {
		app.Spec.SyncPolicy = &argoSyncPolicy{
			SyncOptions: cfg.Application.SyncPolicy.SyncOptions,
		}
		if cfg.Application.SyncPolicy.Automated != nil {
			app.Spec.SyncPolicy.Automated = &argoAutomated{
				Prune:      cfg.Application.SyncPolicy.Automated.Prune,
				SelfHeal:   cfg.Application.SyncPolicy.Automated.SelfHeal,
				AllowEmpty: cfg.Application.SyncPolicy.Automated.AllowEmpty,
			}
		}
		if cfg.Application.SyncPolicy.Retry != nil {
			app.Spec.SyncPolicy.Retry = &argoRetry{
				Limit: cfg.Application.SyncPolicy.Retry.Limit,
			}
			if cfg.Application.SyncPolicy.Retry.BackoffSecs > 0 {
				app.Spec.SyncPolicy.Retry.Backoff = &argoBackoff{
					Duration:    fmt.Sprintf("%ds", cfg.Application.SyncPolicy.Retry.BackoffSecs),
					Factor:      2,
					MaxDuration: "3m",
				}
			}
		}
	}

	return app
}

func (p *Plugin) parseConfig(spec map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Application: ApplicationConfig{
			Project:        "default",
			Namespace:      "argocd",
			TargetRevision: "HEAD",
		},
	}

	if url, ok := spec["serverUrl"].(string); ok {
		cfg.ServerURL = url
	}
	if token, ok := spec["authToken"].(string); ok {
		cfg.AuthToken = token
	}
	if user, ok := spec["username"].(string); ok {
		cfg.Username = user
	}
	if pass, ok := spec["password"].(string); ok {
		cfg.Password = pass
	}
	if insecure, ok := spec["insecure"].(bool); ok {
		cfg.Insecure = insecure
	}

	if appSpec, ok := spec["application"].(map[string]interface{}); ok {
		if name, ok := appSpec["name"].(string); ok {
			cfg.Application.Name = name
		}
		if ns, ok := appSpec["namespace"].(string); ok {
			cfg.Application.Namespace = ns
		}
		if proj, ok := appSpec["project"].(string); ok {
			cfg.Application.Project = proj
		}
		if url, ok := appSpec["repoUrl"].(string); ok {
			cfg.Application.RepoURL = url
		}
		if path, ok := appSpec["path"].(string); ok {
			cfg.Application.Path = path
		}
		if rev, ok := appSpec["targetRevision"].(string); ok {
			cfg.Application.TargetRevision = rev
		}
		if server, ok := appSpec["destServer"].(string); ok {
			cfg.Application.DestServer = server
		}
		if destNs, ok := appSpec["destNamespace"].(string); ok {
			cfg.Application.DestNamespace = destNs
		}
		if labels, ok := appSpec["labels"].(map[string]interface{}); ok {
			cfg.Application.Labels = make(map[string]string)
			for k, v := range labels {
				if vs, ok := v.(string); ok {
					cfg.Application.Labels[k] = vs
				}
			}
		}
		if annotations, ok := appSpec["annotations"].(map[string]interface{}); ok {
			cfg.Application.Annotations = make(map[string]string)
			for k, v := range annotations {
				if vs, ok := v.(string); ok {
					cfg.Application.Annotations[k] = vs
				}
			}
		}

		// Parse sync policy
		if syncPolicy, ok := appSpec["syncPolicy"].(map[string]interface{}); ok {
			cfg.Application.SyncPolicy = &SyncPolicy{}
			if automated, ok := syncPolicy["automated"].(map[string]interface{}); ok {
				cfg.Application.SyncPolicy.Automated = &AutomatedPolicy{
					Prune:    getBool(automated, "prune", false),
					SelfHeal: getBool(automated, "selfHeal", false),
				}
			}
			if options, ok := syncPolicy["syncOptions"].([]interface{}); ok {
				for _, opt := range options {
					if optStr, ok := opt.(string); ok {
						cfg.Application.SyncPolicy.SyncOptions = append(cfg.Application.SyncPolicy.SyncOptions, optStr)
					}
				}
			}
		}

		// Parse Helm config
		if helm, ok := appSpec["helm"].(map[string]interface{}); ok {
			cfg.Application.Helm = &HelmConfig{}
			if vf, ok := helm["valueFiles"].([]interface{}); ok {
				for _, f := range vf {
					if fs, ok := f.(string); ok {
						cfg.Application.Helm.ValueFiles = append(cfg.Application.Helm.ValueFiles, fs)
					}
				}
			}
			if vals, ok := helm["values"].(map[string]interface{}); ok {
				cfg.Application.Helm.Values = vals
			}
			if rn, ok := helm["releaseName"].(string); ok {
				cfg.Application.Helm.ReleaseName = rn
			}
		}

		// Parse Kustomize config
		if kustomize, ok := appSpec["kustomize"].(map[string]interface{}); ok {
			cfg.Application.Kustomize = &KustomizeConfig{}
			if prefix, ok := kustomize["namePrefix"].(string); ok {
				cfg.Application.Kustomize.NamePrefix = prefix
			}
			if suffix, ok := kustomize["nameSuffix"].(string); ok {
				cfg.Application.Kustomize.NameSuffix = suffix
			}
			if images, ok := kustomize["images"].([]interface{}); ok {
				for _, img := range images {
					if imgStr, ok := img.(string); ok {
						cfg.Application.Kustomize.Images = append(cfg.Application.Kustomize.Images, imgStr)
					}
				}
			}
		}
	}

	return cfg, nil
}

func getBool(m map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return defaultVal
}

type notFoundError struct {
	name string
}

func (e *notFoundError) Error() string {
	return fmt.Sprintf("application %s not found", e.name)
}

func isNotFound(err error) bool {
	_, ok := err.(*notFoundError)
	return ok
}
