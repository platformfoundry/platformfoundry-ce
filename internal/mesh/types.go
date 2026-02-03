package mesh

import (
	"time"
)

// ServiceMesh defines service mesh configuration
type ServiceMesh struct {
	APIVersion string             `json:"apiVersion" yaml:"apiVersion"`
	Kind       string             `json:"kind" yaml:"kind"`
	Metadata   MeshMetadata       `json:"metadata" yaml:"metadata"`
	Spec       ServiceMeshSpec    `json:"spec" yaml:"spec"`
	Status     *ServiceMeshStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

// MeshMetadata contains mesh identification
type MeshMetadata struct {
	Name        string            `json:"name" yaml:"name"`
	Namespace   string            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	CreatedAt   time.Time         `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
}

// ServiceMeshSpec defines mesh specification
type ServiceMeshSpec struct {
	Provider      MeshProvider        `json:"provider" yaml:"provider"`
	MTLS          MTLSConfig          `json:"mtls,omitempty" yaml:"mtls,omitempty"`
	Traffic       TrafficConfig       `json:"traffic,omitempty" yaml:"traffic,omitempty"`
	Observability ObservabilityConfig `json:"observability,omitempty" yaml:"observability,omitempty"`
	Security      SecurityConfig      `json:"security,omitempty" yaml:"security,omitempty"`
	Ingress       *IngressConfig      `json:"ingress,omitempty" yaml:"ingress,omitempty"`
	Egress        *EgressConfig       `json:"egress,omitempty" yaml:"egress,omitempty"`
}

// MeshProvider defines the service mesh provider
type MeshProvider string

const (
	MeshProviderIstio   MeshProvider = "istio"
	MeshProviderLinkerd MeshProvider = "linkerd"
	MeshProviderCilium  MeshProvider = "cilium"
	MeshProviderConsul  MeshProvider = "consul"
)

// MTLSConfig defines mutual TLS configuration
type MTLSConfig struct {
	Mode                 MTLSMode `json:"mode" yaml:"mode"`
	CertificateAuthority string   `json:"certificateAuthority,omitempty" yaml:"certificateAuthority,omitempty"`
	RootCertTTL          string   `json:"rootCertTTL,omitempty" yaml:"rootCertTTL,omitempty"`
	WorkloadCertTTL      string   `json:"workloadCertTTL,omitempty" yaml:"workloadCertTTL,omitempty"`
	MinTLSVersion        string   `json:"minTLSVersion,omitempty" yaml:"minTLSVersion,omitempty"`
}

// MTLSMode defines mTLS enforcement mode
type MTLSMode string

const (
	MTLSModeStrict     MTLSMode = "strict"
	MTLSModePermissive MTLSMode = "permissive"
	MTLSModeDisabled   MTLSMode = "disabled"
)

// TrafficConfig defines traffic management settings
type TrafficConfig struct {
	Retries        RetryConfig          `json:"retries,omitempty" yaml:"retries,omitempty"`
	CircuitBreaker CircuitBreakerConfig `json:"circuitBreaker,omitempty" yaml:"circuitBreaker,omitempty"`
	Timeout        TimeoutConfig        `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	LoadBalancing  LoadBalancingConfig  `json:"loadBalancing,omitempty" yaml:"loadBalancing,omitempty"`
	RateLimiting   *RateLimitConfig     `json:"rateLimiting,omitempty" yaml:"rateLimiting,omitempty"`
}

// RetryConfig defines retry policy
type RetryConfig struct {
	Attempts      int      `json:"attempts" yaml:"attempts"`
	PerTryTimeout string   `json:"perTryTimeout" yaml:"perTryTimeout"`
	RetryOn       []string `json:"retryOn,omitempty" yaml:"retryOn,omitempty"` // 5xx, gateway-error, etc.
	BackoffBase   string   `json:"backoffBase,omitempty" yaml:"backoffBase,omitempty"`
	BackoffMax    string   `json:"backoffMax,omitempty" yaml:"backoffMax,omitempty"`
}

// CircuitBreakerConfig defines circuit breaker settings
type CircuitBreakerConfig struct {
	ConsecutiveErrors  int    `json:"consecutiveErrors" yaml:"consecutiveErrors"`
	Interval           string `json:"interval" yaml:"interval"`
	BaseEjectionTime   string `json:"baseEjectionTime" yaml:"baseEjectionTime"`
	MaxEjectionPercent int    `json:"maxEjectionPercent,omitempty" yaml:"maxEjectionPercent,omitempty"`
	MinHealthPercent   int    `json:"minHealthPercent,omitempty" yaml:"minHealthPercent,omitempty"`
}

// TimeoutConfig defines timeout settings
type TimeoutConfig struct {
	Request string `json:"request" yaml:"request"`
	Idle    string `json:"idle,omitempty" yaml:"idle,omitempty"`
}

// LoadBalancingConfig defines load balancing strategy
type LoadBalancingConfig struct {
	Algorithm      string                `json:"algorithm" yaml:"algorithm"` // round_robin, least_conn, random
	ConsistentHash *ConsistentHashConfig `json:"consistentHash,omitempty" yaml:"consistentHash,omitempty"`
	LocalityAware  bool                  `json:"localityAware,omitempty" yaml:"localityAware,omitempty"`
}

// ConsistentHashConfig defines consistent hashing for load balancing
type ConsistentHashConfig struct {
	HTTPHeader      string `json:"httpHeader,omitempty" yaml:"httpHeader,omitempty"`
	HTTPCookie      string `json:"httpCookie,omitempty" yaml:"httpCookie,omitempty"`
	SourceIP        bool   `json:"sourceIP,omitempty" yaml:"sourceIP,omitempty"`
	MinimumRingSize int    `json:"minimumRingSize,omitempty" yaml:"minimumRingSize,omitempty"`
}

// RateLimitConfig defines rate limiting
type RateLimitConfig struct {
	RequestsPerUnit int    `json:"requestsPerUnit" yaml:"requestsPerUnit"`
	Unit            string `json:"unit" yaml:"unit"` // second, minute, hour
	BurstSize       int    `json:"burstSize,omitempty" yaml:"burstSize,omitempty"`
}

// ObservabilityConfig defines mesh observability settings
type ObservabilityConfig struct {
	Tracing TracingConfig `json:"tracing,omitempty" yaml:"tracing,omitempty"`
	Metrics MetricsConfig `json:"metrics,omitempty" yaml:"metrics,omitempty"`
	Logging LoggingConfig `json:"logging,omitempty" yaml:"logging,omitempty"`
}

// TracingConfig defines distributed tracing settings
type TracingConfig struct {
	Enabled       bool    `json:"enabled" yaml:"enabled"`
	Sampling      float64 `json:"sampling" yaml:"sampling"`                     // 0.0 to 100.0
	Provider      string  `json:"provider,omitempty" yaml:"provider,omitempty"` // jaeger, zipkin, tempo
	CollectorAddr string  `json:"collectorAddr,omitempty" yaml:"collectorAddr,omitempty"`
}

// MetricsConfig defines metrics collection
type MetricsConfig struct {
	Enabled        bool     `json:"enabled" yaml:"enabled"`
	PrometheusPort int      `json:"prometheusPort,omitempty" yaml:"prometheusPort,omitempty"`
	IncludeLabels  []string `json:"includeLabels,omitempty" yaml:"includeLabels,omitempty"`
	ExcludeLabels  []string `json:"excludeLabels,omitempty" yaml:"excludeLabels,omitempty"`
}

// LoggingConfig defines access logging
type LoggingConfig struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Format  string `json:"format,omitempty" yaml:"format,omitempty"` // json, text
	Level   string `json:"level,omitempty" yaml:"level,omitempty"`   // debug, info, warn, error
}

// SecurityConfig defines mesh security settings
type SecurityConfig struct {
	AuthorizationPolicy *AuthorizationPolicy `json:"authorizationPolicy,omitempty" yaml:"authorizationPolicy,omitempty"`
	PeerAuthentication  *PeerAuthentication  `json:"peerAuthentication,omitempty" yaml:"peerAuthentication,omitempty"`
}

// AuthorizationPolicy defines access control
type AuthorizationPolicy struct {
	Action string     `json:"action" yaml:"action"` // ALLOW, DENY
	Rules  []AuthRule `json:"rules,omitempty" yaml:"rules,omitempty"`
}

// AuthRule defines an authorization rule
type AuthRule struct {
	From []AuthSource    `json:"from,omitempty" yaml:"from,omitempty"`
	To   []AuthTarget    `json:"to,omitempty" yaml:"to,omitempty"`
	When []AuthCondition `json:"when,omitempty" yaml:"when,omitempty"`
}

// AuthSource defines allowed sources
type AuthSource struct {
	Principals []string `json:"principals,omitempty" yaml:"principals,omitempty"`
	Namespaces []string `json:"namespaces,omitempty" yaml:"namespaces,omitempty"`
	IPBlocks   []string `json:"ipBlocks,omitempty" yaml:"ipBlocks,omitempty"`
}

// AuthTarget defines allowed targets
type AuthTarget struct {
	Hosts   []string `json:"hosts,omitempty" yaml:"hosts,omitempty"`
	Ports   []int    `json:"ports,omitempty" yaml:"ports,omitempty"`
	Methods []string `json:"methods,omitempty" yaml:"methods,omitempty"`
	Paths   []string `json:"paths,omitempty" yaml:"paths,omitempty"`
}

// AuthCondition defines additional conditions
type AuthCondition struct {
	Key    string   `json:"key" yaml:"key"`
	Values []string `json:"values" yaml:"values"`
}

// PeerAuthentication defines peer authentication settings
type PeerAuthentication struct {
	Mode         MTLSMode         `json:"mode" yaml:"mode"`
	PortSelector map[int]MTLSMode `json:"portSelector,omitempty" yaml:"portSelector,omitempty"`
}

// IngressConfig defines ingress gateway settings
type IngressConfig struct {
	Enabled  bool       `json:"enabled" yaml:"enabled"`
	Hosts    []string   `json:"hosts,omitempty" yaml:"hosts,omitempty"`
	TLS      *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty"`
	Replicas int        `json:"replicas,omitempty" yaml:"replicas,omitempty"`
}

// TLSConfig defines TLS settings
type TLSConfig struct {
	Mode           string `json:"mode" yaml:"mode"` // SIMPLE, MUTUAL, PASSTHROUGH
	CredentialName string `json:"credentialName,omitempty" yaml:"credentialName,omitempty"`
	MinVersion     string `json:"minVersion,omitempty" yaml:"minVersion,omitempty"`
}

// EgressConfig defines egress gateway settings
type EgressConfig struct {
	Enabled        bool     `json:"enabled" yaml:"enabled"`
	AllowedHosts   []string `json:"allowedHosts,omitempty" yaml:"allowedHosts,omitempty"`
	BlockByDefault bool     `json:"blockByDefault,omitempty" yaml:"blockByDefault,omitempty"`
}

// ServiceMeshStatus tracks mesh status
type ServiceMeshStatus struct {
	Phase          MeshPhase       `json:"phase" yaml:"phase"`
	LastUpdated    time.Time       `json:"lastUpdated" yaml:"lastUpdated"`
	Services       int             `json:"services" yaml:"services"`
	ProxiesHealthy int             `json:"proxiesHealthy" yaml:"proxiesHealthy"`
	ProxiesTotal   int             `json:"proxiesTotal" yaml:"proxiesTotal"`
	Conditions     []MeshCondition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// MeshPhase defines mesh phase
type MeshPhase string

const (
	MeshPhaseHealthy    MeshPhase = "Healthy"
	MeshPhaseDegraded   MeshPhase = "Degraded"
	MeshPhaseUnhealthy  MeshPhase = "Unhealthy"
	MeshPhaseInstalling MeshPhase = "Installing"
)

// MeshCondition represents a mesh condition
type MeshCondition struct {
	Type               string    `json:"type" yaml:"type"`
	Status             string    `json:"status" yaml:"status"`
	LastTransitionTime time.Time `json:"lastTransitionTime" yaml:"lastTransitionTime"`
	Reason             string    `json:"reason,omitempty" yaml:"reason,omitempty"`
	Message            string    `json:"message,omitempty" yaml:"message,omitempty"`
}

// VirtualService defines traffic routing for a service
type VirtualService struct {
	APIVersion string             `json:"apiVersion" yaml:"apiVersion"`
	Kind       string             `json:"kind" yaml:"kind"`
	Metadata   MeshMetadata       `json:"metadata" yaml:"metadata"`
	Spec       VirtualServiceSpec `json:"spec" yaml:"spec"`
}

// VirtualServiceSpec defines virtual service specification
type VirtualServiceSpec struct {
	Hosts    []string    `json:"hosts" yaml:"hosts"`
	Gateways []string    `json:"gateways,omitempty" yaml:"gateways,omitempty"`
	HTTP     []HTTPRoute `json:"http,omitempty" yaml:"http,omitempty"`
	TCP      []TCPRoute  `json:"tcp,omitempty" yaml:"tcp,omitempty"`
}

// HTTPRoute defines an HTTP route
type HTTPRoute struct {
	Name    string                 `json:"name,omitempty" yaml:"name,omitempty"`
	Match   []HTTPMatchRequest     `json:"match,omitempty" yaml:"match,omitempty"`
	Route   []HTTPRouteDestination `json:"route" yaml:"route"`
	Timeout string                 `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Retries *RetryConfig           `json:"retries,omitempty" yaml:"retries,omitempty"`
	Fault   *FaultInjection        `json:"fault,omitempty" yaml:"fault,omitempty"`
	Headers *HeaderOperations      `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// HTTPMatchRequest defines HTTP match criteria
type HTTPMatchRequest struct {
	URI     *StringMatch           `json:"uri,omitempty" yaml:"uri,omitempty"`
	Headers map[string]StringMatch `json:"headers,omitempty" yaml:"headers,omitempty"`
	Method  *StringMatch           `json:"method,omitempty" yaml:"method,omitempty"`
}

// StringMatch defines string matching
type StringMatch struct {
	Exact  string `json:"exact,omitempty" yaml:"exact,omitempty"`
	Prefix string `json:"prefix,omitempty" yaml:"prefix,omitempty"`
	Regex  string `json:"regex,omitempty" yaml:"regex,omitempty"`
}

// HTTPRouteDestination defines a route destination
type HTTPRouteDestination struct {
	Destination Destination `json:"destination" yaml:"destination"`
	Weight      int         `json:"weight,omitempty" yaml:"weight,omitempty"`
}

// Destination defines a traffic destination
type Destination struct {
	Host   string        `json:"host" yaml:"host"`
	Port   *PortSelector `json:"port,omitempty" yaml:"port,omitempty"`
	Subset string        `json:"subset,omitempty" yaml:"subset,omitempty"`
}

// PortSelector defines a port selection
type PortSelector struct {
	Number int `json:"number" yaml:"number"`
}

// FaultInjection defines fault injection for testing
type FaultInjection struct {
	Delay *FaultDelay `json:"delay,omitempty" yaml:"delay,omitempty"`
	Abort *FaultAbort `json:"abort,omitempty" yaml:"abort,omitempty"`
}

// FaultDelay defines delay injection
type FaultDelay struct {
	Percentage float64 `json:"percentage" yaml:"percentage"`
	FixedDelay string  `json:"fixedDelay" yaml:"fixedDelay"`
}

// FaultAbort defines abort injection
type FaultAbort struct {
	Percentage float64 `json:"percentage" yaml:"percentage"`
	HTTPStatus int     `json:"httpStatus" yaml:"httpStatus"`
}

// HeaderOperations defines header modifications
type HeaderOperations struct {
	Set    map[string]string `json:"set,omitempty" yaml:"set,omitempty"`
	Add    map[string]string `json:"add,omitempty" yaml:"add,omitempty"`
	Remove []string          `json:"remove,omitempty" yaml:"remove,omitempty"`
}

// TCPRoute defines a TCP route
type TCPRoute struct {
	Match []TCPMatchRequest     `json:"match,omitempty" yaml:"match,omitempty"`
	Route []TCPRouteDestination `json:"route" yaml:"route"`
}

// TCPMatchRequest defines TCP match criteria
type TCPMatchRequest struct {
	DestinationSubnets []string `json:"destinationSubnets,omitempty" yaml:"destinationSubnets,omitempty"`
	Port               int      `json:"port,omitempty" yaml:"port,omitempty"`
}

// TCPRouteDestination defines TCP route destination
type TCPRouteDestination struct {
	Destination Destination `json:"destination" yaml:"destination"`
	Weight      int         `json:"weight,omitempty" yaml:"weight,omitempty"`
}

// DestinationRule defines traffic policies for a destination
type DestinationRule struct {
	APIVersion string              `json:"apiVersion" yaml:"apiVersion"`
	Kind       string              `json:"kind" yaml:"kind"`
	Metadata   MeshMetadata        `json:"metadata" yaml:"metadata"`
	Spec       DestinationRuleSpec `json:"spec" yaml:"spec"`
}

// DestinationRuleSpec defines destination rule specification
type DestinationRuleSpec struct {
	Host          string         `json:"host" yaml:"host"`
	TrafficPolicy *TrafficPolicy `json:"trafficPolicy,omitempty" yaml:"trafficPolicy,omitempty"`
	Subsets       []Subset       `json:"subsets,omitempty" yaml:"subsets,omitempty"`
}

// TrafficPolicy defines traffic policies
type TrafficPolicy struct {
	ConnectionPool   *ConnectionPool       `json:"connectionPool,omitempty" yaml:"connectionPool,omitempty"`
	LoadBalancer     *LoadBalancerSettings `json:"loadBalancer,omitempty" yaml:"loadBalancer,omitempty"`
	OutlierDetection *OutlierDetection     `json:"outlierDetection,omitempty" yaml:"outlierDetection,omitempty"`
	TLS              *TLSSettings          `json:"tls,omitempty" yaml:"tls,omitempty"`
}

// ConnectionPool defines connection pool settings
type ConnectionPool struct {
	TCP  *TCPConnectionPool  `json:"tcp,omitempty" yaml:"tcp,omitempty"`
	HTTP *HTTPConnectionPool `json:"http,omitempty" yaml:"http,omitempty"`
}

// TCPConnectionPool defines TCP connection pool
type TCPConnectionPool struct {
	MaxConnections int    `json:"maxConnections" yaml:"maxConnections"`
	ConnectTimeout string `json:"connectTimeout,omitempty" yaml:"connectTimeout,omitempty"`
}

// HTTPConnectionPool defines HTTP connection pool
type HTTPConnectionPool struct {
	HTTP1MaxPendingRequests  int `json:"http1MaxPendingRequests,omitempty" yaml:"http1MaxPendingRequests,omitempty"`
	HTTP2MaxRequests         int `json:"http2MaxRequests,omitempty" yaml:"http2MaxRequests,omitempty"`
	MaxRequestsPerConnection int `json:"maxRequestsPerConnection,omitempty" yaml:"maxRequestsPerConnection,omitempty"`
	MaxRetries               int `json:"maxRetries,omitempty" yaml:"maxRetries,omitempty"`
}

// LoadBalancerSettings defines load balancer settings
type LoadBalancerSettings struct {
	Simple         string            `json:"simple,omitempty" yaml:"simple,omitempty"`
	ConsistentHash *ConsistentHashLB `json:"consistentHash,omitempty" yaml:"consistentHash,omitempty"`
}

// ConsistentHashLB defines consistent hash load balancing
type ConsistentHashLB struct {
	HTTPHeaderName string      `json:"httpHeaderName,omitempty" yaml:"httpHeaderName,omitempty"`
	HTTPCookie     *HTTPCookie `json:"httpCookie,omitempty" yaml:"httpCookie,omitempty"`
	UseSourceIP    bool        `json:"useSourceIp,omitempty" yaml:"useSourceIp,omitempty"`
}

// HTTPCookie defines HTTP cookie for consistent hashing
type HTTPCookie struct {
	Name string `json:"name" yaml:"name"`
	TTL  string `json:"ttl,omitempty" yaml:"ttl,omitempty"`
}

// OutlierDetection defines outlier detection (circuit breaker)
type OutlierDetection struct {
	Consecutive5xxErrors     int    `json:"consecutive5xxErrors,omitempty" yaml:"consecutive5xxErrors,omitempty"`
	ConsecutiveGatewayErrors int    `json:"consecutiveGatewayErrors,omitempty" yaml:"consecutiveGatewayErrors,omitempty"`
	Interval                 string `json:"interval,omitempty" yaml:"interval,omitempty"`
	BaseEjectionTime         string `json:"baseEjectionTime,omitempty" yaml:"baseEjectionTime,omitempty"`
	MaxEjectionPercent       int    `json:"maxEjectionPercent,omitempty" yaml:"maxEjectionPercent,omitempty"`
}

// TLSSettings defines TLS settings for traffic policy
type TLSSettings struct {
	Mode              string `json:"mode" yaml:"mode"` // DISABLE, SIMPLE, MUTUAL, ISTIO_MUTUAL
	ClientCertificate string `json:"clientCertificate,omitempty" yaml:"clientCertificate,omitempty"`
	PrivateKey        string `json:"privateKey,omitempty" yaml:"privateKey,omitempty"`
	CACertificates    string `json:"caCertificates,omitempty" yaml:"caCertificates,omitempty"`
}

// Subset defines a service subset
type Subset struct {
	Name          string            `json:"name" yaml:"name"`
	Labels        map[string]string `json:"labels" yaml:"labels"`
	TrafficPolicy *TrafficPolicy    `json:"trafficPolicy,omitempty" yaml:"trafficPolicy,omitempty"`
}
