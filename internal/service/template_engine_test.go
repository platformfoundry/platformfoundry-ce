package service

import (
	"strings"
	"testing"
)

func TestTemplateEngine_Render(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name     string
		template string
		params   map[string]interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "basic parameter substitution",
			template: "Hello {{.name}}!",
			params:   map[string]interface{}{"name": "World"},
			want:     "Hello World!",
			wantErr:  false,
		},
		{
			name:     "multiple parameters",
			template: "{{.greeting}} {{.name}}, your age is {{.age}}",
			params: map[string]interface{}{
				"greeting": "Hi",
				"name":     "Alice",
				"age":      30,
			},
			want:    "Hi Alice, your age is 30",
			wantErr: false,
		},
		{
			name:     "nested object access",
			template: "Team: {{.team.name}}, Owner: {{.team.owner}}",
			params: map[string]interface{}{
				"team": map[string]interface{}{
					"name":  "Platform",
					"owner": "John",
				},
			},
			want:    "Team: Platform, Owner: John",
			wantErr: false,
		},
		{
			name:     "invalid template syntax",
			template: "Hello {{.name",
			params:   map[string]interface{}{"name": "World"},
			wantErr:  true,
		},
		{
			name:     "missing parameter",
			template: "Hello {{.missing}}!",
			params:   map[string]interface{}{"name": "World"},
			want:     "Hello <no value>!",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Render(tt.template, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Render() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplateEngine_CustomFunctions_Upper(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name     string
		template string
		params   map[string]interface{}
		want     string
	}{
		{
			name:     "upper function",
			template: "{{.name | upper}}",
			params:   map[string]interface{}{"name": "hello"},
			want:     "HELLO",
		},
		{
			name:     "upper on sentence",
			template: "{{.text | upper}}",
			params:   map[string]interface{}{"text": "hello world"},
			want:     "HELLO WORLD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Render(tt.template, tt.params)
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("Render() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplateEngine_CustomFunctions_Lower(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name     string
		template string
		params   map[string]interface{}
		want     string
	}{
		{
			name:     "lower function",
			template: "{{.name | lower}}",
			params:   map[string]interface{}{"name": "HELLO"},
			want:     "hello",
		},
		{
			name:     "lower on mixed case",
			template: "{{.text | lower}}",
			params:   map[string]interface{}{"text": "Hello World"},
			want:     "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Render(tt.template, tt.params)
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("Render() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplateEngine_CustomFunctions_Title(t *testing.T) {
	engine := NewTemplateEngine()

	template := "{{.text | title}}"
	params := map[string]interface{}{"text": "hello world"}
	want := "Hello World"

	got, err := engine.Render(template, params)
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}
	if got != want {
		t.Errorf("Render() = %v, want %v", got, want)
	}
}

func TestTemplateEngine_CustomFunctions_Trim(t *testing.T) {
	engine := NewTemplateEngine()

	template := "{{.text | trim}}"
	params := map[string]interface{}{"text": "  hello world  "}
	want := "hello world"

	got, err := engine.Render(template, params)
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}
	if got != want {
		t.Errorf("Render() = %v, want %v", got, want)
	}
}

func TestTemplateEngine_CustomFunctions_Replace(t *testing.T) {
	engine := NewTemplateEngine()

	template := "{{replace .text \"world\" \"Go\"}}"
	params := map[string]interface{}{"text": "hello world"}
	want := "hello Go"

	got, err := engine.Render(template, params)
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}
	if got != want {
		t.Errorf("Render() = %v, want %v", got, want)
	}
}

func TestTemplateEngine_CustomFunctions_Contains(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name     string
		template string
		params   map[string]interface{}
		want     string
	}{
		{
			name:     "contains true",
			template: "{{if contains .text \"world\"}}found{{else}}not found{{end}}",
			params:   map[string]interface{}{"text": "hello world"},
			want:     "found",
		},
		{
			name:     "contains false",
			template: "{{if contains .text \"missing\"}}found{{else}}not found{{end}}",
			params:   map[string]interface{}{"text": "hello world"},
			want:     "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Render(tt.template, tt.params)
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("Render() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplateEngine_CustomFunctions_Split(t *testing.T) {
	engine := NewTemplateEngine()

	template := "{{range split .text \",\"}}{{.}}-{{end}}"
	params := map[string]interface{}{"text": "a,b,c"}
	want := "a-b-c-"

	got, err := engine.Render(template, params)
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}
	if got != want {
		t.Errorf("Render() = %v, want %v", got, want)
	}
}

func TestTemplateEngine_CustomFunctions_Join(t *testing.T) {
	engine := NewTemplateEngine()

	template := "{{join .items \", \"}}"
	params := map[string]interface{}{"items": []string{"a", "b", "c"}}
	want := "a, b, c"

	got, err := engine.Render(template, params)
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}
	if got != want {
		t.Errorf("Render() = %v, want %v", got, want)
	}
}

func TestTemplateEngine_CustomFunctions_Default(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name     string
		template string
		params   map[string]interface{}
		want     string
	}{
		{
			name:     "default with value",
			template: "{{.name | default \"Unknown\"}}",
			params:   map[string]interface{}{"name": "Alice"},
			want:     "Alice",
		},
		{
			name:     "default with missing value",
			template: "{{.name | default \"Unknown\"}}",
			params:   map[string]interface{}{},
			want:     "Unknown",
		},
		{
			name:     "default with empty string",
			template: "{{.name | default \"Unknown\"}}",
			params:   map[string]interface{}{"name": ""},
			want:     "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Render(tt.template, tt.params)
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("Render() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplateEngine_CustomFunctions_Quote(t *testing.T) {
	engine := NewTemplateEngine()

	template := "{{.text | quote}}"
	params := map[string]interface{}{"text": "hello"}
	want := "\"hello\""

	got, err := engine.Render(template, params)
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}
	if got != want {
		t.Errorf("Render() = %v, want %v", got, want)
	}
}

func TestTemplateEngine_CustomFunctions_Indent(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name     string
		template string
		params   map[string]interface{}
		want     string
	}{
		{
			name:     "indent 2 spaces",
			template: "{{indent 2 .text}}",
			params:   map[string]interface{}{"text": "hello\nworld"},
			want:     "  hello\n  world",
		},
		{
			name:     "indent 4 spaces",
			template: "{{indent 4 .text}}",
			params:   map[string]interface{}{"text": "line1\nline2"},
			want:     "    line1\n    line2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Render(tt.template, tt.params)
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("Render() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplateEngine_CustomFunctions_Nindent(t *testing.T) {
	engine := NewTemplateEngine()

	template := "{{nindent 2 .text}}"
	params := map[string]interface{}{"text": "hello\nworld"}
	want := "\n  hello\n  world"

	got, err := engine.Render(template, params)
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}
	if got != want {
		t.Errorf("Render() = %v, want %v", got, want)
	}
}

func TestTemplateEngine_SafeRender_SizeLimits(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name     string
		template string
		params   map[string]interface{}
		maxSize  int
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "template too large",
			template: strings.Repeat("a", 2*1024*1024), // 2MB template
			params:   map[string]interface{}{},
			maxSize:  1024 * 1024, // 1MB limit
			wantErr:  true,
			errMsg:   "template size",
		},
		{
			name:     "output too large",
			template: "{{range .items}}{{.}}{{end}}",
			params: map[string]interface{}{
				"items": func() []string {
					// Create 11000 items of 100 chars each = 1.1MB output
					// With maxSize=100KB, output limit is 1MB, so this should exceed
					items := make([]string, 11000)
					for i := range items {
						items[i] = strings.Repeat("x", 100)
					}
					return items
				}(),
			},
			maxSize: 1024 * 100, // 100KB limit, so output limit is 1MB
			wantErr: true,
			errMsg:  "rendered output size",
		},
		{
			name:     "within size limits",
			template: "Hello {{.name}}!",
			params:   map[string]interface{}{"name": "World"},
			maxSize:  1024,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := engine.SafeRender(tt.template, tt.params, tt.maxSize)
			if (err != nil) != tt.wantErr {
				t.Errorf("SafeRender() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("SafeRender() error = %v, should contain %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestTemplateEngine_ComplexTemplate(t *testing.T) {
	engine := NewTemplateEngine()

	template := `apiVersion: platformfoundry.io/v1
kind: Service
metadata:
  name: {{.serviceName | lower}}
  organization: {{.organization}}
  labels:
    team: {{.team | lower}}
    env: {{.environment | default "production"}}
spec:
  type: {{.serviceType}}
  owner:
    team: {{.team}}
    email: {{.email | quote}}
  repository:
    url: {{.repoUrl}}
  {{- if .dependencies}}
  dependencies:
  {{- range .dependencies}}
    - name: {{.name}}
      type: {{.type}}
  {{- end}}
  {{- end}}`

	params := map[string]interface{}{
		"serviceName": "UserAPI",
		"organization": "acme-corp",
		"team":        "Platform",
		"environment": "",
		"serviceType": "microservice",
		"email":       "platform@acme.com",
		"repoUrl":     "https://github.com/acme/user-api",
		"dependencies": []map[string]interface{}{
			{"name": "auth-service", "type": "microservice"},
			{"name": "postgres", "type": "database"},
		},
	}

	result, err := engine.Render(template, params)
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	// Verify key parts of the output
	if !strings.Contains(result, "name: userapi") {
		t.Error("Expected lowercase service name 'userapi'")
	}
	if !strings.Contains(result, "team: platform") {
		t.Error("Expected lowercase team 'platform'")
	}
	if !strings.Contains(result, "env: production") {
		t.Error("Expected default environment 'production'")
	}
	if !strings.Contains(result, "email: \"platform@acme.com\"") {
		t.Error("Expected quoted email")
	}
	if !strings.Contains(result, "- name: auth-service") {
		t.Error("Expected auth-service dependency")
	}
	if !strings.Contains(result, "- name: postgres") {
		t.Error("Expected postgres dependency")
	}
}

func TestTemplateEngine_Conditionals(t *testing.T) {
	engine := NewTemplateEngine()

	template := `{{if .enabled}}Service is enabled{{else}}Service is disabled{{end}}`

	tests := []struct {
		name   string
		params map[string]interface{}
		want   string
	}{
		{
			name:   "enabled true",
			params: map[string]interface{}{"enabled": true},
			want:   "Service is enabled",
		},
		{
			name:   "enabled false",
			params: map[string]interface{}{"enabled": false},
			want:   "Service is disabled",
		},
		{
			name:   "enabled missing",
			params: map[string]interface{}{},
			want:   "Service is disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Render(template, tt.params)
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}
			if got != tt.want {
				t.Errorf("Render() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplateEngine_Loops(t *testing.T) {
	engine := NewTemplateEngine()

	template := `{{range .items}}{{.name}}: {{.value}}
{{end}}`

	params := map[string]interface{}{
		"items": []map[string]interface{}{
			{"name": "cpu", "value": "2"},
			{"name": "memory", "value": "4Gi"},
			{"name": "disk", "value": "100Gi"},
		},
	}

	want := `cpu: 2
memory: 4Gi
disk: 100Gi
`

	got, err := engine.Render(template, params)
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}
	if got != want {
		t.Errorf("Render() = %v, want %v", got, want)
	}
}

func TestTemplateEngine_EdgeCases(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name     string
		template string
		params   map[string]interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "empty template",
			template: "",
			params:   map[string]interface{}{},
			want:     "",
			wantErr:  false,
		},
		{
			name:     "template with no parameters",
			template: "Hello World",
			params:   map[string]interface{}{},
			want:     "Hello World",
			wantErr:  false,
		},
		{
			name:     "nil params",
			template: "Hello World",
			params:   nil,
			want:     "Hello World",
			wantErr:  false,
		},
		{
			name:     "special characters",
			template: "{{.text}}",
			params:   map[string]interface{}{"text": "Hello! @#$%^&*()"},
			want:     "Hello! @#$%^&*()",
			wantErr:  false,
		},
		{
			name:     "unicode characters",
			template: "{{.text}}",
			params:   map[string]interface{}{"text": "Hello 世界 🌍"},
			want:     "Hello 世界 🌍",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.Render(tt.template, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Render() = %v, want %v", got, tt.want)
			}
		})
	}
}
