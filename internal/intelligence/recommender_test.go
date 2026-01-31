package intelligence

import (
	"testing"
)

func TestRecommender_Recommend(t *testing.T) {
	recommender, err := NewRecommender("")
	if err != nil {
		t.Fatalf("Failed to create recommender: %v", err)
	}

	tests := []struct {
		name           string
		techStack      *TechStack
		wantTemplate   string
		wantConfidence float64
	}{
		{
			name: "AWS with full monitoring",
			techStack: &TechStack{
				CloudProvider:      "aws",
				Orchestrator:       "argocd",
				ObservabilityTools: []string{"prometheus", "grafana"},
				HasMonitoring:      true,
			},
			wantTemplate:   "aws-k8s-full",
			wantConfidence: 0.9,
		},
		{
			name: "GCP with monitoring",
			techStack: &TechStack{
				CloudProvider: "gcp",
				HasMonitoring: true,
			},
			wantTemplate:   "gcp-k8s-full",
			wantConfidence: 0.9,
		},
		{
			name: "Azure with monitoring",
			techStack: &TechStack{
				CloudProvider: "azure",
				HasMonitoring: true,
			},
			wantTemplate:   "azure-k8s-full",
			wantConfidence: 0.9,
		},
		{
			name: "Basic cloud without monitoring",
			techStack: &TechStack{
				CloudProvider: "aws",
				HasMonitoring: false,
			},
			wantTemplate:   "k8s-basic",
			wantConfidence: 0.7,
		},
		{
			name: "No cloud provider",
			techStack: &TechStack{
				CloudProvider: "",
			},
			wantTemplate:   "minimal",
			wantConfidence: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recommender.Recommend(tt.techStack)

			if got.Template != tt.wantTemplate {
				t.Errorf("Template = %v, want %v", got.Template, tt.wantTemplate)
			}
			if got.Confidence != tt.wantConfidence {
				t.Errorf("Confidence = %v, want %v", got.Confidence, tt.wantConfidence)
			}
			if got.Reason == "" {
				t.Error("Reason should not be empty")
			}
			if len(got.Features) == 0 {
				t.Error("Features should not be empty")
			}
		})
	}
}

func TestRecommender_MatchRules(t *testing.T) {
	recommender, _ := NewRecommender("")

	techStack := &TechStack{
		CloudProvider: "aws",
		HasMonitoring: true,
	}

	matched := recommender.matchRules(techStack)

	if len(matched) == 0 {
		t.Error("Should have matched at least one rule")
	}

	// Should match AWS rule first (highest priority)
	if matched[0].Name != "AWS Kubernetes Full Stack" {
		t.Errorf("First matched rule = %v, want 'AWS Kubernetes Full Stack'", matched[0].Name)
	}
}

func TestMatchString(t *testing.T) {
	tests := []struct {
		pattern string
		value   string
		want    bool
	}{
		{"aws", "aws", true},
		{"aws", "AWS", true},
		{"*", "anything", true},
		{"*", "", false},
		{"gcp", "aws", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.value, func(t *testing.T) {
			got := matchString(tt.pattern, tt.value)
			if got != tt.want {
				t.Errorf("matchString(%s, %s) = %v, want %v", tt.pattern, tt.value, got, tt.want)
			}
		})
	}
}

func TestMatchAny(t *testing.T) {
	tests := []struct {
		name     string
		required []interface{}
		actual   []string
		want     bool
	}{
		{
			name:     "Match found",
			required: []interface{}{"prometheus", "grafana"},
			actual:   []string{"prometheus", "loki"},
			want:     true,
		},
		{
			name:     "No match",
			required: []interface{}{"tempo"},
			actual:   []string{"prometheus", "grafana"},
			want:     false,
		},
		{
			name:     "Empty required",
			required: []interface{}{},
			actual:   []string{"prometheus"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchAny(tt.required, tt.actual)
			if got != tt.want {
				t.Errorf("matchAny() = %v, want %v", got, tt.want)
			}
		})
	}
}
