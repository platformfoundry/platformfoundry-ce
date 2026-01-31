package service

import (
	"strings"
	"testing"

	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
)

func TestTemplateManager_Create(t *testing.T) {
	backend := NewMockBackend()
	manager := NewTemplateManager(backend)

	template := &types.ServiceTemplate{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "ServiceTemplate",
		Metadata: types.Metadata{
			Name:         "nodejs-express",
			Organization: "acme-corp",
		},
		Spec: types.ServiceTemplateSpec{
			DisplayName: "Node.js Express API",
			Description: "A REST API using Node.js and Express",
			Category:    types.TemplateCategoryBackend,
			Parameters: []types.TemplateParameter{
				{
					Name:     "serviceName",
					Type:     types.ParameterTypeString,
					Required: true,
				},
			},
			Template: `{"metadata":{"name":"{{.serviceName}}"}}`,
		},
	}

	err := manager.Create(template)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify template was saved
	retrieved, err := manager.Get("nodejs-express", "acme-corp")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if retrieved.Metadata.Name != "nodejs-express" {
		t.Errorf("Expected name 'nodejs-express', got '%s'", retrieved.Metadata.Name)
	}
}

func TestTemplateManager_List(t *testing.T) {
	backend := NewMockBackend()
	manager := NewTemplateManager(backend)

	templates := []*types.ServiceTemplate{
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "ServiceTemplate",
			Metadata: types.Metadata{
				Name:         "nodejs-express",
				Organization: "acme-corp",
			},
			Spec: types.ServiceTemplateSpec{
				DisplayName: "Node.js Express",
				Category:    types.TemplateCategoryBackend,
				Template:    `{}`,
			},
		},
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "ServiceTemplate",
			Metadata: types.Metadata{
				Name:         "react-app",
				Organization: "acme-corp",
			},
			Spec: types.ServiceTemplateSpec{
				DisplayName: "React App",
				Category:    types.TemplateCategoryFrontend,
				Template:    `{}`,
			},
		},
	}

	for _, tmpl := range templates {
		err := manager.Create(tmpl)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}
	}

	// List templates
	listed, err := manager.List("acme-corp")
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(listed) != 2 {
		t.Errorf("Expected 2 templates, got %d", len(listed))
	}
}

func TestTemplateManager_ListByCategory(t *testing.T) {
	backend := NewMockBackend()
	manager := NewTemplateManager(backend)

	templates := []*types.ServiceTemplate{
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "ServiceTemplate",
			Metadata: types.Metadata{
				Name:         "nodejs-express",
				Organization: "acme-corp",
			},
			Spec: types.ServiceTemplateSpec{
				DisplayName: "Node.js Express",
				Category:    types.TemplateCategoryBackend,
				Template:    `{}`,
			},
		},
		{
			APIVersion: "platformfoundry.io/v1",
			Kind:       "ServiceTemplate",
			Metadata: types.Metadata{
				Name:         "react-app",
				Organization: "acme-corp",
			},
			Spec: types.ServiceTemplateSpec{
				DisplayName: "React App",
				Category:    types.TemplateCategoryFrontend,
				Template:    `{}`,
			},
		},
	}

	for _, tmpl := range templates {
		err := manager.Create(tmpl)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}
	}

	// List by category
	backendTemplates, err := manager.ListByCategory(types.TemplateCategoryBackend, "acme-corp")
	if err != nil {
		t.Fatalf("ListByCategory() failed: %v", err)
	}

	if len(backendTemplates) != 1 {
		t.Errorf("Expected 1 backend template, got %d", len(backendTemplates))
	}

	if backendTemplates[0].Metadata.Name != "nodejs-express" {
		t.Errorf("Expected 'nodejs-express', got '%s'", backendTemplates[0].Metadata.Name)
	}
}

func TestTemplateManager_Instantiate(t *testing.T) {
	backend := NewMockBackend()
	manager := NewTemplateManager(backend)

	template := &types.ServiceTemplate{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "ServiceTemplate",
		Metadata: types.Metadata{
			Name:         "nodejs-express",
			Organization: "acme-corp",
		},
		Spec: types.ServiceTemplateSpec{
			DisplayName: "Node.js Express API",
			Category:    types.TemplateCategoryBackend,
			Parameters: []types.TemplateParameter{
				{
					Name:     "serviceName",
					Type:     types.ParameterTypeString,
					Required: true,
				},
				{
					Name:     "team",
					Type:     types.ParameterTypeString,
					Required: true,
				},
			},
			Template: `{
				"apiVersion": "platformfoundry.io/v1",
				"kind": "Service",
				"metadata": {
					"name": "{{.serviceName}}"
				},
				"spec": {
					"type": "microservice",
					"owner": {
						"team": "{{.team}}"
					}
				}
			}`,
		},
	}

	err := manager.Create(template)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Instantiate template
	params := map[string]interface{}{
		"serviceName": "user-api",
		"team":        "platform-team",
	}

	service, err := manager.Instantiate("nodejs-express", "acme-corp", params)
	if err != nil {
		t.Fatalf("Instantiate() failed: %v", err)
	}

	if service.Metadata.Name != "user-api" {
		t.Errorf("Expected service name 'user-api', got '%s'", service.Metadata.Name)
	}

	if service.Spec.Owner.Team != "platform-team" {
		t.Errorf("Expected team 'platform-team', got '%s'", service.Spec.Owner.Team)
	}
}

func TestTemplateManager_ValidateParameters(t *testing.T) {
	backend := NewMockBackend()
	manager := NewTemplateManager(backend)

	template := &types.ServiceTemplate{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "ServiceTemplate",
		Metadata: types.Metadata{
			Name:         "test-template",
			Organization: "acme-corp",
		},
		Spec: types.ServiceTemplateSpec{
			DisplayName: "Test Template",
			Category:    types.TemplateCategoryBackend,
			Parameters: []types.TemplateParameter{
				{
					Name:     "requiredParam",
					Type:     types.ParameterTypeString,
					Required: true,
				},
				{
					Name:      "stringParam",
					Type:      types.ParameterTypeString,
					MinLength: 3,
					MaxLength: 10,
				},
				{
					Name: "numberParam",
					Type: types.ParameterTypeNumber,
					Min:  func() *float64 { v := 1.0; return &v }(),
					Max:  func() *float64 { v := 100.0; return &v }(),
				},
				{
					Name: "enumParam",
					Type: types.ParameterTypeString,
					Enum: []string{"option1", "option2", "option3"},
				},
			},
			Template: `{}`,
		},
	}

	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name:    "missing required parameter",
			params:  map[string]interface{}{},
			wantErr: true,
			errMsg:  "required parameter 'requiredParam' is missing",
		},
		{
			name: "string too short",
			params: map[string]interface{}{
				"requiredParam": "test",
				"stringParam":   "ab",
			},
			wantErr: true,
			errMsg:  "string length 2 is less than minimum 3",
		},
		{
			name: "string too long",
			params: map[string]interface{}{
				"requiredParam": "test",
				"stringParam":   "this-is-too-long",
			},
			wantErr: true,
			errMsg:  "string length",
		},
		{
			name: "number too small",
			params: map[string]interface{}{
				"requiredParam": "test",
				"numberParam":   0.5,
			},
			wantErr: true,
			errMsg:  "less than minimum",
		},
		{
			name: "number too large",
			params: map[string]interface{}{
				"requiredParam": "test",
				"numberParam":   101.0,
			},
			wantErr: true,
			errMsg:  "exceeds maximum",
		},
		{
			name: "invalid enum value",
			params: map[string]interface{}{
				"requiredParam": "test",
				"enumParam":     "invalid",
			},
			wantErr: true,
			errMsg:  "not in enum",
		},
		{
			name: "valid parameters",
			params: map[string]interface{}{
				"requiredParam": "test",
				"stringParam":   "valid",
				"numberParam":   50.0,
				"enumParam":     "option2",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.validateParameters(template, tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateParameters() error = %v, should contain %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestTemplateManager_Delete(t *testing.T) {
	backend := NewMockBackend()
	manager := NewTemplateManager(backend)

	template := &types.ServiceTemplate{
		APIVersion: "platformfoundry.io/v1",
		Kind:       "ServiceTemplate",
		Metadata: types.Metadata{
			Name:         "nodejs-express",
			Organization: "acme-corp",
		},
		Spec: types.ServiceTemplateSpec{
			DisplayName: "Node.js Express",
			Category:    types.TemplateCategoryBackend,
			Template:    `{}`,
		},
	}

	err := manager.Create(template)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Delete template
	err = manager.Delete("nodejs-express", "acme-corp")
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	// Verify deletion
	_, err = manager.Get("nodejs-express", "acme-corp")
	if err == nil {
		t.Error("Template still exists after deletion")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
