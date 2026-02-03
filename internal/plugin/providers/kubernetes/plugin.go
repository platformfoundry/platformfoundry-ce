package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/platformfoundry/pf-ce/pkg/plugin"
)

// Plugin implements the Plugin interface for Kubernetes resources
type Plugin struct {
	name    string
	version string

	clientset   *kubernetes.Clientset
	initialized bool
	mu          sync.RWMutex

	// Track deployed resources
	resources map[string]resourceInfo
}

type resourceInfo struct {
	Kind      string
	Namespace string
	Name      string
}

// Config represents the configuration schema for Kubernetes resources
type Config struct {
	Kubeconfig string     `yaml:"kubeconfig,omitempty" json:"kubeconfig,omitempty"`
	Context    string     `yaml:"context,omitempty" json:"context,omitempty"`
	Namespace  string     `yaml:"namespace" json:"namespace"`
	Manifests  []Manifest `yaml:"manifests" json:"manifests"`
}

// Manifest represents a Kubernetes manifest to deploy
type Manifest struct {
	Kind        string                 `yaml:"kind" json:"kind"`
	Name        string                 `yaml:"name" json:"name"`
	Spec        map[string]interface{} `yaml:"spec" json:"spec"`
	Labels      map[string]string      `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string      `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

// New creates a new Kubernetes plugin
func New() *Plugin {
	return &Plugin{
		name:      "kubernetes",
		version:   "1.0.0",
		resources: make(map[string]resourceInfo),
	}
}

func (p *Plugin) Name() string    { return p.name }
func (p *Plugin) Type() string    { return "Cluster" }
func (p *Plugin) Version() string { return p.version }

func (p *Plugin) ConfigType() interface{} {
	return Config{}
}

// Initialize sets up the Kubernetes client
func (p *Plugin) Initialize(ctx context.Context, kubeconfig, kubeContext string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return nil
	}

	var cfg *rest.Config
	var err error

	if kubeconfig != "" {
		// Use provided kubeconfig
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		loadingRules.ExplicitPath = kubeconfig

		configOverrides := &clientcmd.ConfigOverrides{}
		if kubeContext != "" {
			configOverrides.CurrentContext = kubeContext
		}

		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loadingRules, configOverrides,
		)

		cfg, err = kubeConfig.ClientConfig()
	} else {
		// Try in-cluster config first, fall back to default kubeconfig
		cfg, err = rest.InClusterConfig()
		if err != nil {
			cfg, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	p.clientset, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	p.initialized = true
	return nil
}

func (p *Plugin) Validate(spec map[string]interface{}) error {
	cfg, err := p.parseConfig(spec)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	supportedKinds := map[string]bool{
		"Deployment":              true,
		"Service":                 true,
		"ConfigMap":               true,
		"Secret":                  true,
		"Ingress":                 true,
		"HorizontalPodAutoscaler": true,
		"PersistentVolumeClaim":   true,
		"ServiceAccount":          true,
		"Namespace":               true,
	}

	for _, m := range cfg.Manifests {
		if !supportedKinds[m.Kind] {
			return fmt.Errorf("unsupported kind: %s", m.Kind)
		}
		if m.Name == "" {
			return fmt.Errorf("manifest name is required")
		}
	}

	if cfg.Namespace == "" {
		return fmt.Errorf("namespace is required")
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

	for _, m := range cfg.Manifests {
		action := fmt.Sprintf("create %s: %s/%s", m.Kind, cfg.Namespace, m.Name)
		plan.Actions = append(plan.Actions, action)
		plan.Changes[m.Name] = m.Kind
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
		if err := p.Initialize(ctx, cfg.Kubeconfig, cfg.Context); err != nil {
			return nil, err
		}
	}

	result := &plugin.Result{
		Status:    "success",
		Resources: make([]string, 0),
		Outputs:   make(map[string]string),
	}

	// Ensure namespace exists
	if err := p.ensureNamespace(ctx, cfg.Namespace); err != nil {
		return nil, fmt.Errorf("failed to ensure namespace: %w", err)
	}

	// Sort manifests by deployment order
	sorted := p.sortByKindOrder(cfg.Manifests)

	// Apply manifests
	for _, m := range sorted {
		output, err := p.applyManifest(ctx, cfg.Namespace, m)
		if err != nil {
			result.Status = "partial"
			result.Message = fmt.Sprintf("failed at %s/%s: %v", m.Kind, m.Name, err)
			return result, err
		}

		result.Resources = append(result.Resources, fmt.Sprintf("%s:%s/%s", m.Kind, cfg.Namespace, m.Name))
		for k, v := range output {
			result.Outputs[fmt.Sprintf("%s.%s.%s", m.Kind, m.Name, k)] = v
		}
	}

	result.Message = fmt.Sprintf("Deployed %d resources to namespace %s", len(result.Resources), cfg.Namespace)
	return result, nil
}

func (p *Plugin) Delete(name string) error {
	p.mu.RLock()
	info, exists := p.resources[name]
	p.mu.RUnlock()

	if !exists {
		return fmt.Errorf("resource %s not found", name)
	}

	ctx := context.Background()

	switch info.Kind {
	case "Deployment":
		return p.clientset.AppsV1().Deployments(info.Namespace).Delete(ctx, info.Name, metav1.DeleteOptions{})
	case "Service":
		return p.clientset.CoreV1().Services(info.Namespace).Delete(ctx, info.Name, metav1.DeleteOptions{})
	case "ConfigMap":
		return p.clientset.CoreV1().ConfigMaps(info.Namespace).Delete(ctx, info.Name, metav1.DeleteOptions{})
	case "Secret":
		return p.clientset.CoreV1().Secrets(info.Namespace).Delete(ctx, info.Name, metav1.DeleteOptions{})
	case "Ingress":
		return p.clientset.NetworkingV1().Ingresses(info.Namespace).Delete(ctx, info.Name, metav1.DeleteOptions{})
	case "HorizontalPodAutoscaler":
		return p.clientset.AutoscalingV2().HorizontalPodAutoscalers(info.Namespace).Delete(ctx, info.Name, metav1.DeleteOptions{})
	default:
		return fmt.Errorf("unsupported kind for deletion: %s", info.Kind)
	}
}

func (p *Plugin) Status(name string) (*plugin.Status, error) {
	p.mu.RLock()
	info, exists := p.resources[name]
	p.mu.RUnlock()

	if !exists {
		return &plugin.Status{
			State:   "unknown",
			Ready:   false,
			Message: "Resource not tracked",
		}, nil
	}

	ctx := context.Background()

	switch info.Kind {
	case "Deployment":
		deploy, err := p.clientset.AppsV1().Deployments(info.Namespace).Get(ctx, info.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		ready := deploy.Status.ReadyReplicas == *deploy.Spec.Replicas
		return &plugin.Status{
			State:   "running",
			Ready:   ready,
			Message: fmt.Sprintf("%d/%d replicas ready", deploy.Status.ReadyReplicas, *deploy.Spec.Replicas),
			Details: map[string]string{
				"replicas":      fmt.Sprintf("%d", *deploy.Spec.Replicas),
				"readyReplicas": fmt.Sprintf("%d", deploy.Status.ReadyReplicas),
			},
		}, nil
	default:
		return &plugin.Status{
			State:   "unknown",
			Ready:   true,
			Message: "Status check not implemented for this kind",
		}, nil
	}
}

func (p *Plugin) ensureNamespace(ctx context.Context, namespace string) error {
	_, err := p.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	if !errors.IsNotFound(err) {
		return err
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"managed-by": "platformfoundry",
			},
		},
	}

	_, err = p.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	return err
}

func (p *Plugin) applyManifest(ctx context.Context, namespace string, m Manifest) (map[string]string, error) {
	switch m.Kind {
	case "Deployment":
		return p.applyDeployment(ctx, namespace, m)
	case "Service":
		return p.applyService(ctx, namespace, m)
	case "ConfigMap":
		return p.applyConfigMap(ctx, namespace, m)
	case "Secret":
		return p.applySecret(ctx, namespace, m)
	case "Ingress":
		return p.applyIngress(ctx, namespace, m)
	case "HorizontalPodAutoscaler":
		return p.applyHPA(ctx, namespace, m)
	default:
		return nil, fmt.Errorf("unsupported kind: %s", m.Kind)
	}
}

func (p *Plugin) applyDeployment(ctx context.Context, namespace string, m Manifest) (map[string]string, error) {
	replicas := int32(getIntProp(m.Spec, "replicas", 1))
	image := getStringProp(m.Spec, "image", "")
	containerPort := int32(getIntProp(m.Spec, "port", 80))

	if image == "" {
		return nil, fmt.Errorf("image is required for Deployment")
	}

	labels := map[string]string{
		"app":        m.Name,
		"managed-by": "platformfoundry",
	}
	for k, v := range m.Labels {
		labels[k] = v
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        m.Name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: m.Annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": m.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  m.Name,
							Image: image,
							Ports: []corev1.ContainerPort{
								{ContainerPort: containerPort},
							},
						},
					},
				},
			},
		},
	}

	// Add resource limits if specified
	if resources, ok := m.Spec["resources"].(map[string]interface{}); ok {
		container := &deploy.Spec.Template.Spec.Containers[0]
		container.Resources = p.buildResourceRequirements(resources)
	}

	// Add environment variables if specified
	if envs, ok := m.Spec["env"].([]interface{}); ok {
		container := &deploy.Spec.Template.Spec.Containers[0]
		for _, e := range envs {
			if env, ok := e.(map[string]interface{}); ok {
				envVar := corev1.EnvVar{
					Name:  getStringProp(env, "name", ""),
					Value: getStringProp(env, "value", ""),
				}
				container.Env = append(container.Env, envVar)
			}
		}
	}

	// Create or update
	existing, err := p.clientset.AppsV1().Deployments(namespace).Get(ctx, m.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	var result *appsv1.Deployment
	if errors.IsNotFound(err) {
		result, err = p.clientset.AppsV1().Deployments(namespace).Create(ctx, deploy, metav1.CreateOptions{})
	} else {
		deploy.ResourceVersion = existing.ResourceVersion
		result, err = p.clientset.AppsV1().Deployments(namespace).Update(ctx, deploy, metav1.UpdateOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to apply deployment: %w", err)
	}

	p.trackResource(m.Name, "Deployment", namespace, m.Name)

	return map[string]string{
		"name":      result.Name,
		"namespace": result.Namespace,
		"replicas":  fmt.Sprintf("%d", *result.Spec.Replicas),
	}, nil
}

func (p *Plugin) applyService(ctx context.Context, namespace string, m Manifest) (map[string]string, error) {
	port := int32(getIntProp(m.Spec, "port", 80))
	targetPort := int32(getIntProp(m.Spec, "targetPort", int(port)))
	svcType := getStringProp(m.Spec, "type", "ClusterIP")

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        m.Name,
			Namespace:   namespace,
			Labels:      m.Labels,
			Annotations: m.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": m.Name},
			Ports: []corev1.ServicePort{
				{
					Port:       port,
					TargetPort: intstr.FromInt32(targetPort),
				},
			},
			Type: corev1.ServiceType(svcType),
		},
	}

	existing, err := p.clientset.CoreV1().Services(namespace).Get(ctx, m.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	var result *corev1.Service
	if errors.IsNotFound(err) {
		result, err = p.clientset.CoreV1().Services(namespace).Create(ctx, svc, metav1.CreateOptions{})
	} else {
		svc.ResourceVersion = existing.ResourceVersion
		svc.Spec.ClusterIP = existing.Spec.ClusterIP // Preserve ClusterIP
		result, err = p.clientset.CoreV1().Services(namespace).Update(ctx, svc, metav1.UpdateOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to apply service: %w", err)
	}

	p.trackResource(m.Name, "Service", namespace, m.Name)

	return map[string]string{
		"name":      result.Name,
		"namespace": result.Namespace,
		"clusterIP": result.Spec.ClusterIP,
		"port":      fmt.Sprintf("%d", port),
	}, nil
}

func (p *Plugin) applyConfigMap(ctx context.Context, namespace string, m Manifest) (map[string]string, error) {
	data := make(map[string]string)
	if d, ok := m.Spec["data"].(map[string]interface{}); ok {
		for k, v := range d {
			if vs, ok := v.(string); ok {
				data[k] = vs
			}
		}
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        m.Name,
			Namespace:   namespace,
			Labels:      m.Labels,
			Annotations: m.Annotations,
		},
		Data: data,
	}

	existing, err := p.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, m.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	var result *corev1.ConfigMap
	if errors.IsNotFound(err) {
		result, err = p.clientset.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
	} else {
		cm.ResourceVersion = existing.ResourceVersion
		result, err = p.clientset.CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to apply configmap: %w", err)
	}

	p.trackResource(m.Name, "ConfigMap", namespace, m.Name)

	return map[string]string{
		"name":      result.Name,
		"namespace": result.Namespace,
	}, nil
}

func (p *Plugin) applySecret(ctx context.Context, namespace string, m Manifest) (map[string]string, error) {
	data := make(map[string][]byte)
	if d, ok := m.Spec["data"].(map[string]interface{}); ok {
		for k, v := range d {
			if vs, ok := v.(string); ok {
				data[k] = []byte(vs)
			}
		}
	}

	secretType := corev1.SecretType(getStringProp(m.Spec, "type", "Opaque"))

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        m.Name,
			Namespace:   namespace,
			Labels:      m.Labels,
			Annotations: m.Annotations,
		},
		Type: secretType,
		Data: data,
	}

	existing, err := p.clientset.CoreV1().Secrets(namespace).Get(ctx, m.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	var result *corev1.Secret
	if errors.IsNotFound(err) {
		result, err = p.clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	} else {
		secret.ResourceVersion = existing.ResourceVersion
		result, err = p.clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to apply secret: %w", err)
	}

	p.trackResource(m.Name, "Secret", namespace, m.Name)

	return map[string]string{
		"name":      result.Name,
		"namespace": result.Namespace,
	}, nil
}

func (p *Plugin) applyIngress(ctx context.Context, namespace string, m Manifest) (map[string]string, error) {
	host := getStringProp(m.Spec, "host", "")
	path := getStringProp(m.Spec, "path", "/")
	serviceName := getStringProp(m.Spec, "serviceName", m.Name)
	servicePort := int32(getIntProp(m.Spec, "servicePort", 80))
	tls := getBoolProp(m.Spec, "tls", false)

	pathType := networkingv1.PathTypePrefix

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        m.Name,
			Namespace:   namespace,
			Labels:      m.Labels,
			Annotations: m.Annotations,
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     path,
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: serviceName,
											Port: networkingv1.ServiceBackendPort{
												Number: servicePort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if tls && host != "" {
		ingress.Spec.TLS = []networkingv1.IngressTLS{
			{
				Hosts:      []string{host},
				SecretName: fmt.Sprintf("%s-tls", m.Name),
			},
		}
	}

	existing, err := p.clientset.NetworkingV1().Ingresses(namespace).Get(ctx, m.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	var result *networkingv1.Ingress
	if errors.IsNotFound(err) {
		result, err = p.clientset.NetworkingV1().Ingresses(namespace).Create(ctx, ingress, metav1.CreateOptions{})
	} else {
		ingress.ResourceVersion = existing.ResourceVersion
		result, err = p.clientset.NetworkingV1().Ingresses(namespace).Update(ctx, ingress, metav1.UpdateOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to apply ingress: %w", err)
	}

	p.trackResource(m.Name, "Ingress", namespace, m.Name)

	return map[string]string{
		"name":      result.Name,
		"namespace": result.Namespace,
		"host":      host,
	}, nil
}

func (p *Plugin) applyHPA(ctx context.Context, namespace string, m Manifest) (map[string]string, error) {
	minReplicas := int32(getIntProp(m.Spec, "minReplicas", 1))
	maxReplicas := int32(getIntProp(m.Spec, "maxReplicas", 10))
	targetCPU := int32(getIntProp(m.Spec, "targetCPUUtilization", 70))
	scaleTargetRef := getStringProp(m.Spec, "scaleTargetRef", m.Name)

	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:        m.Name,
			Namespace:   namespace,
			Labels:      m.Labels,
			Annotations: m.Annotations,
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       scaleTargetRef,
			},
			MinReplicas: &minReplicas,
			MaxReplicas: maxReplicas,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: corev1.ResourceCPU,
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &targetCPU,
						},
					},
				},
			},
		},
	}

	existing, err := p.clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(ctx, m.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	var result *autoscalingv2.HorizontalPodAutoscaler
	if errors.IsNotFound(err) {
		result, err = p.clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).Create(ctx, hpa, metav1.CreateOptions{})
	} else {
		hpa.ResourceVersion = existing.ResourceVersion
		result, err = p.clientset.AutoscalingV2().HorizontalPodAutoscalers(namespace).Update(ctx, hpa, metav1.UpdateOptions{})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to apply hpa: %w", err)
	}

	p.trackResource(m.Name, "HorizontalPodAutoscaler", namespace, m.Name)

	return map[string]string{
		"name":        result.Name,
		"namespace":   result.Namespace,
		"minReplicas": fmt.Sprintf("%d", minReplicas),
		"maxReplicas": fmt.Sprintf("%d", maxReplicas),
	}, nil
}

func (p *Plugin) buildResourceRequirements(resources map[string]interface{}) corev1.ResourceRequirements {
	req := corev1.ResourceRequirements{}

	if limits, ok := resources["limits"].(map[string]interface{}); ok {
		req.Limits = corev1.ResourceList{}
		if cpu, ok := limits["cpu"].(string); ok {
			req.Limits[corev1.ResourceCPU] = mustParseQuantity(cpu)
		}
		if mem, ok := limits["memory"].(string); ok {
			req.Limits[corev1.ResourceMemory] = mustParseQuantity(mem)
		}
	}

	if requests, ok := resources["requests"].(map[string]interface{}); ok {
		req.Requests = corev1.ResourceList{}
		if cpu, ok := requests["cpu"].(string); ok {
			req.Requests[corev1.ResourceCPU] = mustParseQuantity(cpu)
		}
		if mem, ok := requests["memory"].(string); ok {
			req.Requests[corev1.ResourceMemory] = mustParseQuantity(mem)
		}
	}

	return req
}

func mustParseQuantity(s string) resource.Quantity {
	q, _ := resource.ParseQuantity(s)
	return q
}

func (p *Plugin) trackResource(key, kind, namespace, name string) {
	p.mu.Lock()
	p.resources[key] = resourceInfo{Kind: kind, Namespace: namespace, Name: name}
	p.mu.Unlock()
}

func (p *Plugin) sortByKindOrder(manifests []Manifest) []Manifest {
	kindOrder := map[string]int{
		"Namespace":               1,
		"ServiceAccount":          2,
		"ConfigMap":               3,
		"Secret":                  4,
		"PersistentVolumeClaim":   5,
		"Deployment":              6,
		"Service":                 7,
		"HorizontalPodAutoscaler": 8,
		"Ingress":                 9,
	}

	sorted := make([]Manifest, len(manifests))
	copy(sorted, manifests)

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if kindOrder[sorted[i].Kind] > kindOrder[sorted[j].Kind] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

func (p *Plugin) parseConfig(spec map[string]interface{}) (*Config, error) {
	cfg := &Config{
		Namespace: "default",
		Manifests: make([]Manifest, 0),
	}

	if ns, ok := spec["namespace"].(string); ok {
		cfg.Namespace = ns
	}
	if kc, ok := spec["kubeconfig"].(string); ok {
		cfg.Kubeconfig = kc
	}
	if ctx, ok := spec["context"].(string); ok {
		cfg.Context = ctx
	}

	if manifests, ok := spec["manifests"].([]interface{}); ok {
		for _, m := range manifests {
			if mm, ok := m.(map[string]interface{}); ok {
				manifest := Manifest{
					Kind:   getStringProp(mm, "kind", ""),
					Name:   getStringProp(mm, "name", ""),
					Spec:   make(map[string]interface{}),
					Labels: make(map[string]string),
				}
				if s, ok := mm["spec"].(map[string]interface{}); ok {
					manifest.Spec = s
				}
				if l, ok := mm["labels"].(map[string]interface{}); ok {
					for k, v := range l {
						if vs, ok := v.(string); ok {
							manifest.Labels[k] = vs
						}
					}
				}
				if a, ok := mm["annotations"].(map[string]interface{}); ok {
					manifest.Annotations = make(map[string]string)
					for k, v := range a {
						if vs, ok := v.(string); ok {
							manifest.Annotations[k] = vs
						}
					}
				}
				cfg.Manifests = append(cfg.Manifests, manifest)
			}
		}
	}

	// Also handle pre-built manifests from workload translation
	if deployment, ok := spec["deployment"].(map[string]interface{}); ok {
		data, _ := json.Marshal(deployment)
		var m Manifest
		json.Unmarshal(data, &m)
		m.Kind = "Deployment"
		cfg.Manifests = append(cfg.Manifests, m)
	}

	return cfg, nil
}

func getStringProp(m map[string]interface{}, key, defaultVal string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return defaultVal
}

func getIntProp(m map[string]interface{}, key string, defaultVal int) int {
	if v, ok := m[key].(int); ok {
		return v
	}
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return defaultVal
}

func getBoolProp(m map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return defaultVal
}
