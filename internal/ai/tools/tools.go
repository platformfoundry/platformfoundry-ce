package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/platformfoundry/platformfoundry-ce/internal/ai"
)

// Tool represents a callable tool for the AI assistant
type Tool struct {
	Definition ai.ToolDefinition
	Handler    ToolHandler
}

// ToolHandler is a function that executes a tool
type ToolHandler func(ctx context.Context, args map[string]interface{}) (string, error)

// ToolRegistry manages available tools
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a new tool registry with platform tools
func NewToolRegistry() *ToolRegistry {
	r := &ToolRegistry{
		tools: make(map[string]Tool),
	}

	// Register all platform tools
	r.registerPlatformTools()

	return r
}

// Register adds a tool to the registry
func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Definition.Name] = tool
}

// Get returns a tool by name
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// GetDefinitions returns all tool definitions for LLM requests
func (r *ToolRegistry) GetDefinitions() []ai.ToolDefinition {
	defs := make([]ai.ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		defs = append(defs, tool.Definition)
	}
	return defs
}

// Execute runs a tool by name with the given arguments
func (r *ToolRegistry) Execute(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	tool, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return tool.Handler(ctx, args)
}

// registerPlatformTools registers all built-in platform tools
func (r *ToolRegistry) registerPlatformTools() {
	// List services tool
	r.Register(Tool{
		Definition: ai.ToolDefinition{
			Name:        "list_services",
			Description: "List all services managed by the platform with their current status",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"environment": map[string]interface{}{
						"type":        "string",
						"description": "Filter by environment (e.g., production, staging, development)",
					},
					"status": map[string]interface{}{
						"type":        "string",
						"description": "Filter by status (healthy, degraded, unhealthy)",
						"enum":        []string{"healthy", "degraded", "unhealthy", "all"},
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of services to return",
						"default":     20,
					},
				},
				"required": []string{},
			},
		},
		Handler: handleListServices,
	})

	// Get health score tool
	r.Register(Tool{
		Definition: ai.ToolDefinition{
			Name:        "get_health_score",
			Description: "Get detailed health score and metrics for a specific service or the entire platform",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"service": map[string]interface{}{
						"type":        "string",
						"description": "Service name (optional, omit for platform-wide health)",
					},
					"environment": map[string]interface{}{
						"type":        "string",
						"description": "Environment to check",
					},
				},
				"required": []string{},
			},
		},
		Handler: handleGetHealthScore,
	})

	// Check drift tool
	r.Register(Tool{
		Definition: ai.ToolDefinition{
			Name:        "check_drift",
			Description: "Check for configuration drift between actual and desired state",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"resource": map[string]interface{}{
						"type":        "string",
						"description": "Specific resource to check (optional)",
					},
					"environment": map[string]interface{}{
						"type":        "string",
						"description": "Environment to check",
					},
					"include_resolved": map[string]interface{}{
						"type":        "boolean",
						"description": "Include recently resolved drift",
						"default":     false,
					},
				},
				"required": []string{},
			},
		},
		Handler: handleCheckDrift,
	})

	// Analyze costs tool
	r.Register(Tool{
		Definition: ai.ToolDefinition{
			Name:        "analyze_costs",
			Description: "Analyze infrastructure costs and identify optimization opportunities",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"environment": map[string]interface{}{
						"type":        "string",
						"description": "Environment to analyze",
					},
					"service": map[string]interface{}{
						"type":        "string",
						"description": "Specific service to analyze",
					},
					"period": map[string]interface{}{
						"type":        "string",
						"description": "Time period for analysis (7d, 30d, 90d)",
						"default":     "30d",
					},
				},
				"required": []string{},
			},
		},
		Handler: handleAnalyzeCosts,
	})

	// Compare environments tool
	r.Register(Tool{
		Definition: ai.ToolDefinition{
			Name:        "compare_environments",
			Description: "Compare configuration and state between two environments",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"source": map[string]interface{}{
						"type":        "string",
						"description": "Source environment",
					},
					"target": map[string]interface{}{
						"type":        "string",
						"description": "Target environment",
					},
					"service": map[string]interface{}{
						"type":        "string",
						"description": "Specific service to compare (optional)",
					},
				},
				"required": []string{"source", "target"},
			},
		},
		Handler: handleCompareEnvironments,
	})

	// List promises tool
	r.Register(Tool{
		Definition: ai.ToolDefinition{
			Name:        "list_promises",
			Description: "List available platform promises (self-service infrastructure offerings)",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Filter by category (database, cache, storage, messaging)",
					},
				},
				"required": []string{},
			},
		},
		Handler: handleListPromises,
	})

	// List workloads tool
	r.Register(Tool{
		Definition: ai.ToolDefinition{
			Name:        "list_workloads",
			Description: "List deployed workloads and their status",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"environment": map[string]interface{}{
						"type":        "string",
						"description": "Filter by environment",
					},
					"team": map[string]interface{}{
						"type":        "string",
						"description": "Filter by team",
					},
				},
				"required": []string{},
			},
		},
		Handler: handleListWorkloads,
	})

	// Get recent events tool
	r.Register(Tool{
		Definition: ai.ToolDefinition{
			Name:        "get_recent_events",
			Description: "Get recent platform events, alerts, and notifications",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Event type filter (deployment, alert, drift, remediation)",
					},
					"severity": map[string]interface{}{
						"type":        "string",
						"description": "Minimum severity (info, warning, error, critical)",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of events",
						"default":     20,
					},
				},
				"required": []string{},
			},
		},
		Handler: handleGetRecentEvents,
	})

	// Get recommendations tool
	r.Register(Tool{
		Definition: ai.ToolDefinition{
			Name:        "get_recommendations",
			Description: "Get AI-powered recommendations for improving platform health, security, and efficiency",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Recommendation category (security, performance, cost, reliability)",
					},
					"environment": map[string]interface{}{
						"type":        "string",
						"description": "Environment to analyze",
					},
				},
				"required": []string{},
			},
		},
		Handler: handleGetRecommendations,
	})

	// Request promise tool
	r.Register(Tool{
		Definition: ai.ToolDefinition{
			Name:        "request_promise",
			Description: "Submit a request for a platform promise (infrastructure resource)",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"promise_name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the promise to request",
					},
					"instance_name": map[string]interface{}{
						"type":        "string",
						"description": "Name for the new instance",
					},
					"environment": map[string]interface{}{
						"type":        "string",
						"description": "Target environment",
					},
					"parameters": map[string]interface{}{
						"type":        "object",
						"description": "Promise-specific parameters",
					},
				},
				"required": []string{"promise_name", "instance_name", "environment"},
			},
		},
		Handler: handleRequestPromise,
	})

	// Describe resource tool
	r.Register(Tool{
		Definition: ai.ToolDefinition{
			Name:        "describe_resource",
			Description: "Get detailed information about a specific resource",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"resource_type": map[string]interface{}{
						"type":        "string",
						"description": "Type of resource (service, workload, promise, database, etc.)",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the resource",
					},
					"environment": map[string]interface{}{
						"type":        "string",
						"description": "Environment",
					},
				},
				"required": []string{"resource_type", "name"},
			},
		},
		Handler: handleDescribeResource,
	})

	// Run diagnostics tool
	r.Register(Tool{
		Definition: ai.ToolDefinition{
			Name:        "run_diagnostics",
			Description: "Run diagnostic checks on a service or the platform",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"target": map[string]interface{}{
						"type":        "string",
						"description": "Target service or 'platform' for overall diagnostics",
					},
					"checks": map[string]interface{}{
						"type":        "array",
						"description": "Specific checks to run (connectivity, resources, configuration)",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"required": []string{"target"},
			},
		},
		Handler: handleRunDiagnostics,
	})
}

// Tool handler implementations

func handleListServices(ctx context.Context, args map[string]interface{}) (string, error) {
	env := getStringArg(args, "environment", "")
	status := getStringArg(args, "status", "all")
	limit := getIntArg(args, "limit", 20)

	// Simulated service data
	services := []map[string]interface{}{
		{"name": "api-gateway", "environment": "production", "status": "healthy", "replicas": "3/3", "cpu": "45%", "memory": "62%"},
		{"name": "user-service", "environment": "production", "status": "healthy", "replicas": "2/2", "cpu": "32%", "memory": "48%"},
		{"name": "order-service", "environment": "production", "status": "degraded", "replicas": "2/3", "cpu": "78%", "memory": "85%"},
		{"name": "payment-service", "environment": "production", "status": "healthy", "replicas": "2/2", "cpu": "25%", "memory": "40%"},
		{"name": "notification-service", "environment": "staging", "status": "healthy", "replicas": "1/1", "cpu": "15%", "memory": "30%"},
		{"name": "analytics-service", "environment": "staging", "status": "unhealthy", "replicas": "0/2", "cpu": "0%", "memory": "0%"},
	}

	// Filter services
	filtered := make([]map[string]interface{}, 0)
	for _, svc := range services {
		if env != "" && svc["environment"] != env {
			continue
		}
		if status != "all" && svc["status"] != status {
			continue
		}
		filtered = append(filtered, svc)
		if len(filtered) >= limit {
			break
		}
	}

	result := formatServiceList(filtered)
	return result, nil
}

func handleGetHealthScore(ctx context.Context, args map[string]interface{}) (string, error) {
	service := getStringArg(args, "service", "")
	env := getStringArg(args, "environment", "production")

	if service == "" {
		// Platform-wide health
		return fmt.Sprintf(`Platform Health Summary (%s)
================================
Overall Score: 87/100

Component Scores:
  Services:      92/100 (18/20 healthy)
  Infrastructure: 85/100 (minor drift detected)
  Security:      88/100 (2 medium findings)
  Performance:   83/100 (elevated latency in 2 services)

Recent Changes:
  - order-service scaled down (5 min ago)
  - prometheus config updated (2 hours ago)
  - New deployment: notification-service v2.3.1 (6 hours ago)

Active Alerts: 3
  - WARN: order-service high memory usage
  - WARN: analytics-service pod crash loop
  - INFO: Scheduled maintenance window in 4 hours`, env), nil
	}

	// Service-specific health
	return fmt.Sprintf(`Service Health: %s (%s)
================================
Health Score: 85/100

Status: Degraded
Replicas: 2/3 ready

Metrics (last 15 min):
  CPU Usage:     78%% avg (elevated)
  Memory Usage:  85%% avg (warning threshold)
  Request Rate:  1,250 req/s
  Error Rate:    0.3%%
  P99 Latency:   245ms

Recent Events:
  - Pod %s-abc123 OOMKilled (3 min ago)
  - Horizontal autoscaler triggered (5 min ago)
  - Config reload completed (1 hour ago)

Dependencies:
  - postgresql: healthy
  - redis: healthy
  - kafka: healthy

Recommendations:
  1. Increase memory limits (current: 512Mi, suggested: 768Mi)
  2. Review recent code changes for memory leaks
  3. Consider adding pod disruption budget`, service, env, service), nil
}

func handleCheckDrift(ctx context.Context, args map[string]interface{}) (string, error) {
	resource := getStringArg(args, "resource", "")
	env := getStringArg(args, "environment", "production")
	includeResolved := getBoolArg(args, "include_resolved", false)

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Drift Detection Report (%s)\n", env))
	result.WriteString("================================\n\n")

	if resource != "" {
		result.WriteString(fmt.Sprintf("Resource: %s\n\n", resource))
	}

	result.WriteString("Active Drift:\n")
	result.WriteString("  1. argocd/application-controller\n")
	result.WriteString("     - Field: spec.replicas\n")
	result.WriteString("     - Expected: 2, Actual: 1\n")
	result.WriteString("     - Detected: 15 minutes ago\n")
	result.WriteString("     - Severity: Medium\n\n")

	result.WriteString("  2. prometheus/retention\n")
	result.WriteString("     - Field: spec.retention\n")
	result.WriteString("     - Expected: 30d, Actual: 7d\n")
	result.WriteString("     - Detected: 2 hours ago\n")
	result.WriteString("     - Severity: Low\n\n")

	if includeResolved {
		result.WriteString("Recently Resolved:\n")
		result.WriteString("  1. grafana/dashboards (auto-remediated 1 hour ago)\n")
		result.WriteString("  2. vault/seal-config (manually fixed 3 hours ago)\n\n")
	}

	result.WriteString("Summary: 2 active drift items, 0 critical\n")
	result.WriteString("Auto-remediation: Enabled for low/medium severity\n")

	return result.String(), nil
}

func handleAnalyzeCosts(ctx context.Context, args map[string]interface{}) (string, error) {
	env := getStringArg(args, "environment", "production")
	service := getStringArg(args, "service", "")
	period := getStringArg(args, "period", "30d")

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Cost Analysis Report (%s, %s)\n", env, period))
	result.WriteString("================================\n\n")

	if service != "" {
		result.WriteString(fmt.Sprintf("Service: %s\n", service))
		result.WriteString("Current Monthly Cost: $1,245\n")
		result.WriteString("Previous Month: $1,180 (+5.5%)\n\n")
		result.WriteString("Breakdown:\n")
		result.WriteString("  Compute:  $850 (68%)\n")
		result.WriteString("  Storage:  $245 (20%)\n")
		result.WriteString("  Network:  $150 (12%)\n\n")
	} else {
		result.WriteString("Total Monthly Cost: $45,230\n")
		result.WriteString("Previous Month: $42,100 (+7.4%)\n\n")
		result.WriteString("Top 5 Services by Cost:\n")
		result.WriteString("  1. order-service:    $8,450 (18.7%)\n")
		result.WriteString("  2. api-gateway:      $6,230 (13.8%)\n")
		result.WriteString("  3. analytics-db:     $5,890 (13.0%)\n")
		result.WriteString("  4. user-service:     $4,120 (9.1%)\n")
		result.WriteString("  5. payment-service:  $3,980 (8.8%)\n\n")
	}

	result.WriteString("Optimization Opportunities:\n")
	result.WriteString("  1. Right-size analytics-db instances: ~$1,200/mo savings\n")
	result.WriteString("  2. Use spot instances for batch jobs: ~$800/mo savings\n")
	result.WriteString("  3. Archive cold storage data: ~$450/mo savings\n")
	result.WriteString("  4. Consolidate unused load balancers: ~$200/mo savings\n\n")
	result.WriteString("Estimated Total Savings: $2,650/mo (5.9%)\n")

	return result.String(), nil
}

func handleCompareEnvironments(ctx context.Context, args map[string]interface{}) (string, error) {
	source := getStringArg(args, "source", "staging")
	target := getStringArg(args, "target", "production")
	service := getStringArg(args, "service", "")

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Environment Comparison: %s vs %s\n", source, target))
	result.WriteString("================================\n\n")

	if service != "" {
		result.WriteString(fmt.Sprintf("Service: %s\n\n", service))
	}

	result.WriteString("Configuration Differences:\n")
	result.WriteString("┌─────────────────────┬──────────────┬──────────────┐\n")
	result.WriteString("│ Setting             │ staging      │ production   │\n")
	result.WriteString("├─────────────────────┼──────────────┼──────────────┤\n")
	result.WriteString("│ replicas            │ 1            │ 3            │\n")
	result.WriteString("│ cpu_limit           │ 500m         │ 1000m        │\n")
	result.WriteString("│ memory_limit        │ 512Mi        │ 1Gi          │\n")
	result.WriteString("│ log_level           │ debug        │ info         │\n")
	result.WriteString("│ rate_limit          │ 1000/min     │ 10000/min    │\n")
	result.WriteString("└─────────────────────┴──────────────┴──────────────┘\n\n")

	result.WriteString("Version Differences:\n")
	result.WriteString("  api-gateway:    v2.4.0 (staging) vs v2.3.1 (production)\n")
	result.WriteString("  user-service:   v1.8.0 (staging) vs v1.8.0 (production) ✓\n")
	result.WriteString("  order-service:  v3.1.0 (staging) vs v3.0.2 (production)\n\n")

	result.WriteString("Summary:\n")
	result.WriteString("  - 5 configuration differences\n")
	result.WriteString("  - 2 version differences\n")
	result.WriteString("  - 0 missing resources\n")

	return result.String(), nil
}

func handleListPromises(ctx context.Context, args map[string]interface{}) (string, error) {
	category := getStringArg(args, "category", "")

	promises := []map[string]interface{}{
		{"name": "postgresql-database", "category": "database", "description": "Managed PostgreSQL database", "status": "available"},
		{"name": "mysql-database", "category": "database", "description": "Managed MySQL database", "status": "available"},
		{"name": "redis-cache", "category": "cache", "description": "Managed Redis cache cluster", "status": "available"},
		{"name": "memcached-cache", "category": "cache", "description": "Managed Memcached cluster", "status": "available"},
		{"name": "s3-bucket", "category": "storage", "description": "S3-compatible object storage", "status": "available"},
		{"name": "kafka-topic", "category": "messaging", "description": "Kafka topic with schema registry", "status": "available"},
		{"name": "rabbitmq-queue", "category": "messaging", "description": "RabbitMQ virtual host", "status": "available"},
		{"name": "dynamodb-table", "category": "database", "description": "DynamoDB table with autoscaling", "status": "available"},
	}

	var result strings.Builder
	result.WriteString("Available Platform Promises\n")
	result.WriteString("================================\n\n")

	for _, p := range promises {
		if category != "" && p["category"] != category {
			continue
		}
		result.WriteString(fmt.Sprintf("• %s [%s]\n", p["name"], p["category"]))
		result.WriteString(fmt.Sprintf("  %s\n", p["description"]))
		result.WriteString(fmt.Sprintf("  Status: %s\n\n", p["status"]))
	}

	result.WriteString("Use 'request_promise' to provision a new instance.\n")

	return result.String(), nil
}

func handleListWorkloads(ctx context.Context, args map[string]interface{}) (string, error) {
	env := getStringArg(args, "environment", "")
	team := getStringArg(args, "team", "")

	workloads := []map[string]interface{}{
		{"name": "api-gateway", "team": "platform", "environment": "production", "version": "v2.3.1", "status": "running"},
		{"name": "user-service", "team": "identity", "environment": "production", "version": "v1.8.0", "status": "running"},
		{"name": "order-service", "team": "commerce", "environment": "production", "version": "v3.0.2", "status": "degraded"},
		{"name": "payment-service", "team": "commerce", "environment": "production", "version": "v2.1.0", "status": "running"},
		{"name": "api-gateway", "team": "platform", "environment": "staging", "version": "v2.4.0", "status": "running"},
		{"name": "notification-service", "team": "engagement", "environment": "staging", "version": "v2.3.1", "status": "running"},
	}

	var result strings.Builder
	result.WriteString("Deployed Workloads\n")
	result.WriteString("================================\n\n")

	result.WriteString("┌────────────────────────┬────────────┬─────────────┬──────────┬──────────┐\n")
	result.WriteString("│ Name                   │ Team       │ Environment │ Version  │ Status   │\n")
	result.WriteString("├────────────────────────┼────────────┼─────────────┼──────────┼──────────┤\n")

	for _, w := range workloads {
		if env != "" && w["environment"] != env {
			continue
		}
		if team != "" && w["team"] != team {
			continue
		}
		result.WriteString(fmt.Sprintf("│ %-22s │ %-10s │ %-11s │ %-8s │ %-8s │\n",
			w["name"], w["team"], w["environment"], w["version"], w["status"]))
	}

	result.WriteString("└────────────────────────┴────────────┴─────────────┴──────────┴──────────┘\n")

	return result.String(), nil
}

func handleGetRecentEvents(ctx context.Context, args map[string]interface{}) (string, error) {
	eventType := getStringArg(args, "type", "")
	severity := getStringArg(args, "severity", "")
	limit := getIntArg(args, "limit", 20)

	events := []map[string]interface{}{
		{"time": "2 min ago", "type": "alert", "severity": "warning", "message": "order-service high memory usage (85%)"},
		{"time": "5 min ago", "type": "deployment", "severity": "info", "message": "notification-service v2.3.1 deployed to staging"},
		{"time": "15 min ago", "type": "drift", "severity": "warning", "message": "argocd replicas drifted from desired state"},
		{"time": "1 hour ago", "type": "remediation", "severity": "info", "message": "Auto-remediated grafana dashboard drift"},
		{"time": "2 hours ago", "type": "alert", "severity": "error", "message": "analytics-service pod crash loop detected"},
		{"time": "3 hours ago", "type": "deployment", "severity": "info", "message": "api-gateway v2.3.1 deployed to production"},
		{"time": "6 hours ago", "type": "alert", "severity": "critical", "message": "Database connection pool exhausted (resolved)"},
	}

	severityOrder := map[string]int{"info": 0, "warning": 1, "error": 2, "critical": 3}

	var result strings.Builder
	result.WriteString("Recent Platform Events\n")
	result.WriteString("================================\n\n")

	count := 0
	for _, e := range events {
		if eventType != "" && e["type"] != eventType {
			continue
		}
		if severity != "" && severityOrder[e["severity"].(string)] < severityOrder[severity] {
			continue
		}
		if count >= limit {
			break
		}

		icon := "ℹ️"
		switch e["severity"] {
		case "warning":
			icon = "⚠️"
		case "error":
			icon = "❌"
		case "critical":
			icon = "🔴"
		}

		result.WriteString(fmt.Sprintf("%s [%s] %s\n", icon, e["time"], e["message"]))
		result.WriteString(fmt.Sprintf("   Type: %s | Severity: %s\n\n", e["type"], e["severity"]))
		count++
	}

	if count == 0 {
		result.WriteString("No events matching the specified filters.\n")
	}

	return result.String(), nil
}

func handleGetRecommendations(ctx context.Context, args map[string]interface{}) (string, error) {
	category := getStringArg(args, "category", "")
	env := getStringArg(args, "environment", "production")

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Platform Recommendations (%s)\n", env))
	result.WriteString("================================\n\n")

	recommendations := []map[string]interface{}{
		{
			"category": "reliability",
			"priority": "high",
			"title":    "Add pod disruption budget to order-service",
			"impact":   "Prevent downtime during node maintenance",
			"effort":   "Low",
		},
		{
			"category": "security",
			"priority": "high",
			"title":    "Rotate database credentials (90+ days old)",
			"impact":   "Reduce risk of credential compromise",
			"effort":   "Low",
		},
		{
			"category": "cost",
			"priority": "medium",
			"title":    "Right-size analytics-db instances",
			"impact":   "Save ~$1,200/month",
			"effort":   "Medium",
		},
		{
			"category": "performance",
			"priority": "medium",
			"title":    "Enable connection pooling for user-service",
			"impact":   "Reduce latency by ~30ms",
			"effort":   "Low",
		},
		{
			"category": "reliability",
			"priority": "low",
			"title":    "Implement circuit breaker for payment-service",
			"impact":   "Improve fault tolerance",
			"effort":   "Medium",
		},
	}

	for _, r := range recommendations {
		if category != "" && r["category"] != category {
			continue
		}

		priorityIcon := "🟡"
		switch r["priority"] {
		case "high":
			priorityIcon = "🔴"
		case "low":
			priorityIcon = "🟢"
		}

		result.WriteString(fmt.Sprintf("%s [%s] %s\n", priorityIcon, r["category"], r["title"]))
		result.WriteString(fmt.Sprintf("   Impact: %s\n", r["impact"]))
		result.WriteString(fmt.Sprintf("   Effort: %s | Priority: %s\n\n", r["effort"], r["priority"]))
	}

	return result.String(), nil
}

func handleRequestPromise(ctx context.Context, args map[string]interface{}) (string, error) {
	promiseName := getStringArg(args, "promise_name", "")
	instanceName := getStringArg(args, "instance_name", "")
	env := getStringArg(args, "environment", "")
	params := args["parameters"]

	if promiseName == "" || instanceName == "" || env == "" {
		return "", fmt.Errorf("promise_name, instance_name, and environment are required")
	}

	var result strings.Builder
	result.WriteString("Promise Request Submitted\n")
	result.WriteString("================================\n\n")
	result.WriteString(fmt.Sprintf("Promise:      %s\n", promiseName))
	result.WriteString(fmt.Sprintf("Instance:     %s\n", instanceName))
	result.WriteString(fmt.Sprintf("Environment:  %s\n", env))

	if params != nil {
		paramsJSON, _ := json.MarshalIndent(params, "              ", "  ")
		result.WriteString(fmt.Sprintf("Parameters:   %s\n", string(paramsJSON)))
	}

	result.WriteString(fmt.Sprintf("\nRequest ID:   req-%s\n", time.Now().Format("20060102-150405")))
	result.WriteString("Status:       Pending Approval\n\n")
	result.WriteString("The request has been submitted and is awaiting approval.\n")
	result.WriteString("You will be notified once the request is processed.\n")

	return result.String(), nil
}

func handleDescribeResource(ctx context.Context, args map[string]interface{}) (string, error) {
	resourceType := getStringArg(args, "resource_type", "")
	name := getStringArg(args, "name", "")
	env := getStringArg(args, "environment", "production")

	if resourceType == "" || name == "" {
		return "", fmt.Errorf("resource_type and name are required")
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Resource Details: %s/%s (%s)\n", resourceType, name, env))
	result.WriteString("================================\n\n")

	switch resourceType {
	case "service", "workload":
		result.WriteString(fmt.Sprintf("Name:           %s\n", name))
		result.WriteString(fmt.Sprintf("Type:           %s\n", resourceType))
		result.WriteString(fmt.Sprintf("Environment:    %s\n", env))
		result.WriteString("Status:         Running\n")
		result.WriteString("Version:        v2.3.1\n")
		result.WriteString("Replicas:       3/3\n")
		result.WriteString("Created:        2024-01-15 10:30:00 UTC\n")
		result.WriteString("Last Updated:   2024-01-20 14:22:00 UTC\n\n")
		result.WriteString("Resources:\n")
		result.WriteString("  CPU Request:    500m\n")
		result.WriteString("  CPU Limit:      1000m\n")
		result.WriteString("  Memory Request: 512Mi\n")
		result.WriteString("  Memory Limit:   1Gi\n\n")
		result.WriteString("Labels:\n")
		result.WriteString("  app: " + name + "\n")
		result.WriteString("  team: platform\n")
		result.WriteString("  version: v2.3.1\n")

	case "database", "promise":
		result.WriteString(fmt.Sprintf("Name:           %s\n", name))
		result.WriteString(fmt.Sprintf("Type:           %s\n", resourceType))
		result.WriteString(fmt.Sprintf("Environment:    %s\n", env))
		result.WriteString("Status:         Available\n")
		result.WriteString("Engine:         PostgreSQL 15.2\n")
		result.WriteString("Instance Class: db.t3.medium\n")
		result.WriteString("Storage:        100 GB\n")
		result.WriteString("Multi-AZ:       Yes\n")
		result.WriteString("Endpoint:       " + name + ".xxxxx." + env + ".rds.amazonaws.com\n")

	default:
		result.WriteString(fmt.Sprintf("Resource type '%s' details not available.\n", resourceType))
	}

	return result.String(), nil
}

func handleRunDiagnostics(ctx context.Context, args map[string]interface{}) (string, error) {
	target := getStringArg(args, "target", "platform")
	checks := args["checks"]

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Diagnostics Report: %s\n", target))
	result.WriteString("================================\n\n")

	// Default checks if none specified
	checkList := []string{"connectivity", "resources", "configuration"}
	if checks != nil {
		if arr, ok := checks.([]interface{}); ok {
			checkList = make([]string, len(arr))
			for i, v := range arr {
				checkList[i] = fmt.Sprintf("%v", v)
			}
		}
	}

	for _, check := range checkList {
		result.WriteString(fmt.Sprintf("Running: %s check...\n", check))
		switch check {
		case "connectivity":
			result.WriteString("  ✅ API Gateway: Reachable (latency: 12ms)\n")
			result.WriteString("  ✅ Database: Reachable (latency: 3ms)\n")
			result.WriteString("  ✅ Cache: Reachable (latency: 1ms)\n")
			result.WriteString("  ✅ Message Queue: Reachable (latency: 5ms)\n")
		case "resources":
			result.WriteString("  ✅ CPU Usage: 45% (healthy)\n")
			result.WriteString("  ⚠️ Memory Usage: 78% (elevated)\n")
			result.WriteString("  ✅ Disk Usage: 52% (healthy)\n")
			result.WriteString("  ✅ Network I/O: Normal\n")
		case "configuration":
			result.WriteString("  ✅ Environment variables: Valid\n")
			result.WriteString("  ✅ Secrets: Accessible\n")
			result.WriteString("  ✅ ConfigMaps: Synchronized\n")
			result.WriteString("  ⚠️ Certificate expiry: 45 days remaining\n")
		}
		result.WriteString("\n")
	}

	result.WriteString("Summary: 11 checks passed, 2 warnings, 0 failures\n")

	return result.String(), nil
}

// Helper functions

func formatServiceList(services []map[string]interface{}) string {
	var result strings.Builder
	result.WriteString("Services List\n")
	result.WriteString("================================\n\n")
	result.WriteString("┌────────────────────────┬─────────────┬──────────┬─────────┬──────────┬──────────┐\n")
	result.WriteString("│ Name                   │ Environment │ Status   │ Replicas│ CPU      │ Memory   │\n")
	result.WriteString("├────────────────────────┼─────────────┼──────────┼─────────┼──────────┼──────────┤\n")

	for _, svc := range services {
		result.WriteString(fmt.Sprintf("│ %-22s │ %-11s │ %-8s │ %-7s │ %-8s │ %-8s │\n",
			svc["name"], svc["environment"], svc["status"], svc["replicas"], svc["cpu"], svc["memory"]))
	}

	result.WriteString("└────────────────────────┴─────────────┴──────────┴─────────┴──────────┴──────────┘\n")
	result.WriteString(fmt.Sprintf("\nTotal: %d services\n", len(services)))

	return result.String()
}

func getStringArg(args map[string]interface{}, key, defaultVal string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

func getIntArg(args map[string]interface{}, key string, defaultVal int) int {
	if v, ok := args[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case float64:
			return int(val)
		}
	}
	return defaultVal
}

func getBoolArg(args map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := args[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultVal
}
