package intelligence

import (
	"testing"
)

func TestTemplateRepository_LoadDefaults(t *testing.T) {
	repo := NewTemplateRepository()
	repo.LoadDefaults()

	templates := repo.List()
	if len(templates) == 0 {
		t.Error("No templates loaded")
	}

	// Verify expected templates exist
	expectedTemplates := []string{
		"aws-k8s-full",
		"gcp-k8s-full",
		"azure-k8s-full",
		"multi-cloud",
		"k8s-basic",
		"minimal",
	}

	for _, id := range expectedTemplates {
		template, err := repo.Get(id)
		if err != nil {
			t.Errorf("Expected template %s not found: %v", id, err)
		}
		if template.ID != id {
			t.Errorf("Template ID = %s, want %s", template.ID, id)
		}
		if template.Name == "" {
			t.Errorf("Template %s has empty name", id)
		}
		if template.Description == "" {
			t.Errorf("Template %s has empty description", id)
		}
	}
}

func TestTemplateRepository_Get(t *testing.T) {
	repo := NewTemplateRepository()
	repo.LoadDefaults()

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "Existing template",
			id:      "aws-k8s-full",
			wantErr: false,
		},
		{
			name:    "Non-existent template",
			id:      "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.Get(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.ID != tt.id {
				t.Errorf("Get() ID = %v, want %v", got.ID, tt.id)
			}
		})
	}
}

func TestTemplateRepository_Search(t *testing.T) {
	repo := NewTemplateRepository()
	repo.LoadDefaults()

	tests := []struct {
		name      string
		query     string
		wantCount int
	}{
		{
			name:      "Search by tag 'aws'",
			query:     "aws",
			wantCount: 2, // aws-k8s-full and multi-cloud
		},
		{
			name:      "Search by tag 'kubernetes'",
			query:     "kubernetes",
			wantCount: 5, // All except minimal
		},
		{
			name:      "Search by tag 'minimal'",
			query:     "minimal",
			wantCount: 1,
		},
		{
			name:      "Search non-existent",
			query:     "nonexistent",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := repo.Search(tt.query)
			if len(results) != tt.wantCount {
				t.Errorf("Search() returned %d results, want %d", len(results), tt.wantCount)
			}
		})
	}
}

func TestGetDefaultTemplates(t *testing.T) {
	templates := getDefaultTemplates()

	if len(templates) < 5 {
		t.Errorf("Expected at least 5 default templates, got %d", len(templates))
	}

	// Verify each template has required fields
	for _, template := range templates {
		if template.ID == "" {
			t.Error("Template has empty ID")
		}
		if template.Name == "" {
			t.Error("Template has empty name")
		}
		if template.Description == "" {
			t.Error("Template has empty description")
		}
		if len(template.Features) == 0 {
			t.Errorf("Template %s has no features", template.ID)
		}
		if len(template.Plugins) == 0 {
			t.Errorf("Template %s has no plugins", template.ID)
		}
	}
}
