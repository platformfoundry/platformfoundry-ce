package rbac

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"
)

// ABACEngine provides attribute-based access control
type ABACEngine struct {
	policyEngine   PolicyEngine
	attributeStore AttributeStore
	policies       map[string]*ABACPolicy
	mu             sync.RWMutex
}

// PolicyEngine interface for evaluating policies
type PolicyEngine interface {
	Evaluate(ctx context.Context, policyName string, input map[string]interface{}) (*PolicyResult, error)
	LoadPolicy(ctx context.Context, name string, policy string) error
}

// PolicyResult represents the result of a policy evaluation
type PolicyResult struct {
	Allow   bool     `json:"allow"`
	Deny    bool     `json:"deny"`
	Reasons []string `json:"reasons,omitempty"`
}

// AttributeStore interface for storing and retrieving attributes
type AttributeStore interface {
	GetSubjectAttributes(ctx context.Context, subjectID string) (map[string]interface{}, error)
	SetSubjectAttributes(ctx context.Context, subjectID string, attrs map[string]interface{}) error
	GetResourceAttributes(ctx context.Context, resourceID string) (map[string]interface{}, error)
	SetResourceAttributes(ctx context.Context, resourceID string, attrs map[string]interface{}) error
}

// ABACPolicy represents an attribute-based access control policy
type ABACPolicy struct {
	ID          string      `json:"id" yaml:"id"`
	Name        string      `json:"name" yaml:"name"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Rules       []ABACRule  `json:"rules" yaml:"rules"`
	Priority    int         `json:"priority" yaml:"priority"` // Higher priority policies evaluated first
	Enabled     bool        `json:"enabled" yaml:"enabled"`
	CreatedAt   time.Time   `json:"createdAt" yaml:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt" yaml:"updatedAt"`
}

// ABACRule represents a single ABAC rule
type ABACRule struct {
	Name        string              `json:"name" yaml:"name"`
	Description string              `json:"description,omitempty" yaml:"description,omitempty"`
	Subject     SubjectMatcher      `json:"subject" yaml:"subject"`
	Resource    ResourceMatcher     `json:"resource" yaml:"resource"`
	Action      ActionMatcher       `json:"action" yaml:"action"`
	Environment EnvironmentMatcher  `json:"environment,omitempty" yaml:"environment,omitempty"`
	Effect      string              `json:"effect" yaml:"effect"` // allow, deny
	Condition   string              `json:"condition,omitempty" yaml:"condition,omitempty"` // CEL or Rego expression
}

// SubjectMatcher defines conditions for matching subjects
type SubjectMatcher struct {
	Attributes map[string]AttributeMatcher `json:"attributes" yaml:"attributes"`
}

// ResourceMatcher defines conditions for matching resources
type ResourceMatcher struct {
	Type       string                      `json:"type,omitempty" yaml:"type,omitempty"`
	Attributes map[string]AttributeMatcher `json:"attributes" yaml:"attributes"`
}

// ActionMatcher defines conditions for matching actions
type ActionMatcher struct {
	Actions []string `json:"actions" yaml:"actions"` // List of actions or "*" for all
}

// EnvironmentMatcher defines conditions for matching environment
type EnvironmentMatcher struct {
	TimeRange   *TimeRange  `json:"timeRange,omitempty" yaml:"timeRange,omitempty"`
	IPRange     []string    `json:"ipRange,omitempty" yaml:"ipRange,omitempty"`
	Locations   []string    `json:"locations,omitempty" yaml:"locations,omitempty"`
	Attributes  map[string]AttributeMatcher `json:"attributes,omitempty" yaml:"attributes,omitempty"`
}

// TimeRange defines a time window
type TimeRange struct {
	StartHour int      `json:"startHour" yaml:"startHour"` // 0-23
	EndHour   int      `json:"endHour" yaml:"endHour"`     // 0-23
	Days      []string `json:"days,omitempty" yaml:"days,omitempty"` // Monday, Tuesday, etc.
	Timezone  string   `json:"timezone,omitempty" yaml:"timezone,omitempty"`
}

// AttributeMatcher defines how to match an attribute value
type AttributeMatcher struct {
	Equals     interface{}   `json:"equals,omitempty" yaml:"equals,omitempty"`
	NotEquals  interface{}   `json:"notEquals,omitempty" yaml:"notEquals,omitempty"`
	In         []interface{} `json:"in,omitempty" yaml:"in,omitempty"`
	NotIn      []interface{} `json:"notIn,omitempty" yaml:"notIn,omitempty"`
	Contains   string        `json:"contains,omitempty" yaml:"contains,omitempty"`
	StartsWith string        `json:"startsWith,omitempty" yaml:"startsWith,omitempty"`
	EndsWith   string        `json:"endsWith,omitempty" yaml:"endsWith,omitempty"`
	Regex      string        `json:"regex,omitempty" yaml:"regex,omitempty"`
	GreaterThan   *float64   `json:"greaterThan,omitempty" yaml:"greaterThan,omitempty"`
	LessThan      *float64   `json:"lessThan,omitempty" yaml:"lessThan,omitempty"`
}

// AccessRequest represents a request to access a resource
type AccessRequest struct {
	Subject     string                 `json:"subject"`
	SubjectType string                 `json:"subjectType"` // user, service, group
	Resource    string                 `json:"resource"`
	ResourceType string                `json:"resourceType"`
	Action      string                 `json:"action"`
	Environment map[string]interface{} `json:"environment,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// AccessDecision represents the result of an access request evaluation
type AccessDecision struct {
	Allowed     bool      `json:"allowed"`
	Reason      string    `json:"reason"`
	PolicyID    string    `json:"policyId,omitempty"`
	RuleID      string    `json:"ruleId,omitempty"`
	EvaluatedAt time.Time `json:"evaluatedAt"`
}

// NewABACEngine creates a new ABACEngine
func NewABACEngine(policyEngine PolicyEngine, attributeStore AttributeStore) *ABACEngine {
	return &ABACEngine{
		policyEngine:   policyEngine,
		attributeStore: attributeStore,
		policies:       make(map[string]*ABACPolicy),
	}
}

// RegisterPolicy registers a new ABAC policy
func (e *ABACEngine) RegisterPolicy(ctx context.Context, policy *ABACPolicy) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if policy.ID == "" {
		policy.ID = fmt.Sprintf("policy-%s-%d", policy.Name, time.Now().UnixNano())
	}

	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()

	e.policies[policy.ID] = policy
	return nil
}

// GetPolicy returns a policy by ID
func (e *ABACEngine) GetPolicy(ctx context.Context, policyID string) (*ABACPolicy, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	policy, ok := e.policies[policyID]
	if !ok {
		return nil, fmt.Errorf("policy not found: %s", policyID)
	}

	return policy, nil
}

// ListPolicies returns all registered policies
func (e *ABACEngine) ListPolicies(ctx context.Context) []*ABACPolicy {
	e.mu.RLock()
	defer e.mu.RUnlock()

	policies := make([]*ABACPolicy, 0, len(e.policies))
	for _, policy := range e.policies {
		policies = append(policies, policy)
	}

	return policies
}

// DeletePolicy removes a policy
func (e *ABACEngine) DeletePolicy(ctx context.Context, policyID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.policies[policyID]; !ok {
		return fmt.Errorf("policy not found: %s", policyID)
	}

	delete(e.policies, policyID)
	return nil
}

// Evaluate evaluates an access request against all policies
func (e *ABACEngine) Evaluate(ctx context.Context, request AccessRequest) (*AccessDecision, error) {
	// Get subject attributes
	subjectAttrs, err := e.getSubjectAttributes(ctx, request.Subject)
	if err != nil {
		return &AccessDecision{
			Allowed:     false,
			Reason:      fmt.Sprintf("failed to get subject attributes: %v", err),
			EvaluatedAt: time.Now(),
		}, nil
	}

	// Get resource attributes
	resourceAttrs, err := e.getResourceAttributes(ctx, request.Resource)
	if err != nil {
		return &AccessDecision{
			Allowed:     false,
			Reason:      fmt.Sprintf("failed to get resource attributes: %v", err),
			EvaluatedAt: time.Now(),
		}, nil
	}

	// Build evaluation context
	evalContext := &EvaluationContext{
		Subject:     subjectAttrs,
		Resource:    resourceAttrs,
		Action:      request.Action,
		Environment: e.getEnvironmentAttributes(ctx, request.Environment),
		RequestContext: request.Context,
	}

	// Evaluate policies in priority order
	e.mu.RLock()
	policies := make([]*ABACPolicy, 0, len(e.policies))
	for _, policy := range e.policies {
		if policy.Enabled {
			policies = append(policies, policy)
		}
	}
	e.mu.RUnlock()

	// Sort by priority (higher first)
	sortPoliciesByPriority(policies)

	// Evaluate each policy
	for _, policy := range policies {
		decision := e.evaluatePolicy(ctx, policy, evalContext, request)
		if decision != nil {
			return decision, nil
		}
	}

	// Default deny if no policy matched
	return &AccessDecision{
		Allowed:     false,
		Reason:      "no matching policy found",
		EvaluatedAt: time.Now(),
	}, nil
}

// EvaluationContext holds all context for policy evaluation
type EvaluationContext struct {
	Subject        map[string]interface{}
	Resource       map[string]interface{}
	Action         string
	Environment    map[string]interface{}
	RequestContext map[string]interface{}
}

// evaluatePolicy evaluates a single policy
func (e *ABACEngine) evaluatePolicy(ctx context.Context, policy *ABACPolicy, evalCtx *EvaluationContext, request AccessRequest) *AccessDecision {
	for _, rule := range policy.Rules {
		if e.ruleMatches(ctx, rule, evalCtx, request) {
			return &AccessDecision{
				Allowed:     rule.Effect == "allow",
				Reason:      fmt.Sprintf("matched rule: %s in policy: %s", rule.Name, policy.Name),
				PolicyID:    policy.ID,
				RuleID:      rule.Name,
				EvaluatedAt: time.Now(),
			}
		}
	}
	return nil
}

// ruleMatches checks if a rule matches the evaluation context
func (e *ABACEngine) ruleMatches(ctx context.Context, rule ABACRule, evalCtx *EvaluationContext, request AccessRequest) bool {
	// Check action match
	if !e.matchAction(rule.Action, request.Action) {
		return false
	}

	// Check subject match
	if !e.matchAttributes(rule.Subject.Attributes, evalCtx.Subject) {
		return false
	}

	// Check resource match
	if rule.Resource.Type != "" && rule.Resource.Type != request.ResourceType {
		return false
	}
	if !e.matchAttributes(rule.Resource.Attributes, evalCtx.Resource) {
		return false
	}

	// Check environment match
	if !e.matchEnvironment(rule.Environment, evalCtx.Environment) {
		return false
	}

	// Check custom condition if present
	if rule.Condition != "" && e.policyEngine != nil {
		input := map[string]interface{}{
			"subject":     evalCtx.Subject,
			"resource":    evalCtx.Resource,
			"action":      evalCtx.Action,
			"environment": evalCtx.Environment,
			"context":     evalCtx.RequestContext,
		}
		result, err := e.policyEngine.Evaluate(ctx, "condition", input)
		if err != nil || !result.Allow {
			return false
		}
	}

	return true
}

// matchAction checks if the action matches
func (e *ABACEngine) matchAction(matcher ActionMatcher, action string) bool {
	for _, a := range matcher.Actions {
		if a == "*" || a == action {
			return true
		}
	}
	return false
}

// matchAttributes checks if attributes match the matchers
func (e *ABACEngine) matchAttributes(matchers map[string]AttributeMatcher, attrs map[string]interface{}) bool {
	for key, matcher := range matchers {
		value, exists := attrs[key]
		if !exists {
			return false
		}
		if !e.matchAttributeValue(matcher, value) {
			return false
		}
	}
	return true
}

// matchAttributeValue checks if a value matches an attribute matcher
func (e *ABACEngine) matchAttributeValue(matcher AttributeMatcher, value interface{}) bool {
	// Equals check
	if matcher.Equals != nil && value != matcher.Equals {
		return false
	}

	// NotEquals check
	if matcher.NotEquals != nil && value == matcher.NotEquals {
		return false
	}

	// In check
	if len(matcher.In) > 0 {
		found := false
		for _, v := range matcher.In {
			if value == v {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// NotIn check
	if len(matcher.NotIn) > 0 {
		for _, v := range matcher.NotIn {
			if value == v {
				return false
			}
		}
	}

	// String operations
	strValue, isString := value.(string)
	if isString {
		if matcher.Contains != "" && !containsString(strValue, matcher.Contains) {
			return false
		}
		if matcher.StartsWith != "" && !startsWithString(strValue, matcher.StartsWith) {
			return false
		}
		if matcher.EndsWith != "" && !endsWithString(strValue, matcher.EndsWith) {
			return false
		}
		if matcher.Regex != "" {
			matched, _ := regexp.MatchString(matcher.Regex, strValue)
			if !matched {
				return false
			}
		}
	}

	// Numeric operations
	numValue, isNum := toFloat64(value)
	if isNum {
		if matcher.GreaterThan != nil && numValue <= *matcher.GreaterThan {
			return false
		}
		if matcher.LessThan != nil && numValue >= *matcher.LessThan {
			return false
		}
	}

	return true
}

// matchEnvironment checks if environment conditions match
func (e *ABACEngine) matchEnvironment(matcher EnvironmentMatcher, env map[string]interface{}) bool {
	// Check time range
	if matcher.TimeRange != nil {
		if !e.matchTimeRange(matcher.TimeRange) {
			return false
		}
	}

	// Check attributes
	if len(matcher.Attributes) > 0 {
		if !e.matchAttributes(matcher.Attributes, env) {
			return false
		}
	}

	return true
}

// matchTimeRange checks if current time is within the specified range
func (e *ABACEngine) matchTimeRange(tr *TimeRange) bool {
	now := time.Now()
	if tr.Timezone != "" {
		loc, err := time.LoadLocation(tr.Timezone)
		if err == nil {
			now = now.In(loc)
		}
	}

	hour := now.Hour()
	if hour < tr.StartHour || hour >= tr.EndHour {
		return false
	}

	if len(tr.Days) > 0 {
		dayName := now.Weekday().String()
		found := false
		for _, d := range tr.Days {
			if d == dayName {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// getSubjectAttributes retrieves subject attributes
func (e *ABACEngine) getSubjectAttributes(ctx context.Context, subjectID string) (map[string]interface{}, error) {
	if e.attributeStore != nil {
		return e.attributeStore.GetSubjectAttributes(ctx, subjectID)
	}
	return make(map[string]interface{}), nil
}

// getResourceAttributes retrieves resource attributes
func (e *ABACEngine) getResourceAttributes(ctx context.Context, resourceID string) (map[string]interface{}, error) {
	if e.attributeStore != nil {
		return e.attributeStore.GetResourceAttributes(ctx, resourceID)
	}
	return make(map[string]interface{}), nil
}

// getEnvironmentAttributes builds environment attributes
func (e *ABACEngine) getEnvironmentAttributes(ctx context.Context, requestEnv map[string]interface{}) map[string]interface{} {
	env := make(map[string]interface{})

	// Add time-based attributes
	now := time.Now()
	env["current_hour"] = now.Hour()
	env["current_day"] = now.Weekday().String()
	env["current_time"] = now.Format(time.RFC3339)

	// Merge request environment
	for k, v := range requestEnv {
		env[k] = v
	}

	return env
}

// Helper functions
func sortPoliciesByPriority(policies []*ABACPolicy) {
	// Simple bubble sort for policies by priority (descending)
	for i := 0; i < len(policies)-1; i++ {
		for j := 0; j < len(policies)-i-1; j++ {
			if policies[j].Priority < policies[j+1].Priority {
				policies[j], policies[j+1] = policies[j+1], policies[j]
			}
		}
	}
}

func containsString(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func startsWithString(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func endsWithString(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case float32:
		return float64(val), true
	case float64:
		return val, true
	default:
		return 0, false
	}
}

// CommonABACPolicies returns commonly used ABAC policies
func CommonABACPolicies() []*ABACPolicy {
	return []*ABACPolicy{
		{
			Name:        "production-protection",
			Description: "Restrict production access to authorized personnel",
			Priority:    100,
			Enabled:     true,
			Rules: []ABACRule{
				{
					Name:        "deny-production-delete",
					Description: "Deny delete operations in production",
					Subject: SubjectMatcher{
						Attributes: map[string]AttributeMatcher{
							"clearance_level": {In: []interface{}{"standard", "basic"}},
						},
					},
					Resource: ResourceMatcher{
						Attributes: map[string]AttributeMatcher{
							"environment": {Equals: "production"},
						},
					},
					Action: ActionMatcher{Actions: []string{"delete", "destroy"}},
					Effect: "deny",
				},
			},
		},
		{
			Name:        "business-hours-only",
			Description: "Restrict certain operations to business hours",
			Priority:    50,
			Enabled:     true,
			Rules: []ABACRule{
				{
					Name:        "allow-business-hours",
					Description: "Allow operations during business hours",
					Subject: SubjectMatcher{
						Attributes: map[string]AttributeMatcher{},
					},
					Resource: ResourceMatcher{
						Attributes: map[string]AttributeMatcher{
							"sensitivity": {Equals: "high"},
						},
					},
					Action: ActionMatcher{Actions: []string{"*"}},
					Environment: EnvironmentMatcher{
						TimeRange: &TimeRange{
							StartHour: 9,
							EndHour:   17,
							Days:      []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},
						},
					},
					Effect: "allow",
				},
			},
		},
		{
			Name:        "team-ownership",
			Description: "Allow teams to manage their own resources",
			Priority:    75,
			Enabled:     true,
			Rules: []ABACRule{
				{
					Name:        "allow-team-resources",
					Description: "Allow users to manage resources owned by their team",
					Subject: SubjectMatcher{
						Attributes: map[string]AttributeMatcher{},
					},
					Resource: ResourceMatcher{
						Attributes: map[string]AttributeMatcher{},
					},
					Action: ActionMatcher{Actions: []string{"*"}},
					Effect:    "allow",
					Condition: "subject.team == resource.owner_team",
				},
			},
		},
	}
}
