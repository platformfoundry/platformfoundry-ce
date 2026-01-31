package cli_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/platformfoundry/platformfoundry-ce/internal/cli"
	"github.com/platformfoundry/platformfoundry-ce/pkg/types"
	"github.com/stretchr/testify/assert"
)

var (
	testOrg  = "acme-corp"
	testTeam = "platform-team"
)

// setupTestServer creates a test HTTP server to mock API responses
func setupTestServer(_ *testing.T, handler http.Handler) *httptest.Server {
	return httptest.NewServer(handler)
}

// TestRunServiceList tests the service list command
func TestRunServiceList(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/services", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		services := []types.Service{
			{
				Metadata: types.Metadata{Name: "user-api", Organization: testOrg},
				Spec:     types.ServiceSpec{Type: "microservice", Owner: types.ServiceOwner{Team: testTeam}},
				Status:   types.ServiceStatus{State: "active", Health: "healthy"},
			},
			{
				Metadata: types.Metadata{Name: "payment-api", Organization: testOrg},
				Spec:     types.ServiceSpec{Type: "microservice", Owner: types.ServiceOwner{Team: "finance-team"}},
				Status:   types.ServiceStatus{State: "active", Health: "healthy"},
			},
		}

		resp := struct {
			Success bool            `json:"success"`
			Data    []types.Service `json:"data"`
		}{
			Success: true,
			Data:    services,
		}

		_ = json.NewEncoder(w).Encode(resp)
	})

	server := setupTestServer(t, handler)
	defer server.Close()

	// Set environment variables for API client
	t.Setenv("PF_API_URL", server.URL)
	t.Setenv("PF_API_TOKEN", "test-token")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	assert.NoError(t, err)
	os.Stdout = w

	// Run command
	cmdErr := cli.RunServiceList(nil, nil)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Check results
	assert.NoError(t, cmdErr)

	var output strings.Builder
	_, err = io.Copy(&output, r)
	assert.NoError(t, err)

	assert.Contains(t, output.String(), "user-api")
	assert.Contains(t, output.String(), "payment-api")
	assert.Contains(t, output.String(), "platform-team")
}

// TestRunServiceGet tests the service get command
func TestRunServiceGet(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/services/user-api", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		service := types.Service{
			Metadata: types.Metadata{Name: "user-api", Organization: testOrg},
			Spec: types.ServiceSpec{
				Type:  "microservice",
				Owner: types.ServiceOwner{Team: testTeam, Email: "team@example.com"},
				Repository: &types.RepositoryConfig{
					URL:    "https://github.com/acme/user-api",
					Branch: "main",
				},
				Dependencies: []types.ServiceDependency{
					{Name: "postgres-db", Type: "database"},
				},
			},
			Status: types.ServiceStatus{State: "active", Health: "healthy"},
		}

		resp := struct {
			Success bool          `json:"success"`
			Data    types.Service `json:"data"`
		}{
			Success: true,
			Data:    service,
		}

		_ = json.NewEncoder(w).Encode(resp)
	})

	server := setupTestServer(t, handler)
	defer server.Close()

	t.Setenv("PF_API_URL", server.URL)
	t.Setenv("PF_API_TOKEN", "test-token")

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	assert.NoError(t, err)
	os.Stdout = w

	cmdErr := cli.RunServiceGet(nil, []string{"user-api"})

	w.Close()
	os.Stdout = oldStdout

	assert.NoError(t, cmdErr)

	var output strings.Builder
	_, err = io.Copy(&output, r)
	assert.NoError(t, err)

	assert.Contains(t, output.String(), "Name:         user-api")
	assert.Contains(t, output.String(), "Team:         platform-team")
	assert.Contains(t, output.String(), "Dependencies:")
	assert.Contains(t, output.String(), "postgres-db (database)")
}

// TestRunServiceCreate tests the service create command
func TestRunServiceCreate(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/services", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var svc types.Service
		err := json.NewDecoder(r.Body).Decode(&svc)
		assert.NoError(t, err)
		assert.Equal(t, "new-api", svc.Metadata.Name)

		resp := struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}{
			Success: true,
			Message: "Service created",
		}

		_ = json.NewEncoder(w).Encode(resp)
	})

	server := setupTestServer(t, handler)
	defer server.Close()

	t.Setenv("PF_API_URL", server.URL)
	t.Setenv("PF_API_TOKEN", "test-token")

	// Create a temporary service file
	serviceFileContent := `
apiVersion: platformfoundry.io/v1
kind: Service
metadata:
  name: new-api
spec:
  type: microservice
  owner:
    team: platform-team
`
	tmpFile, err := os.CreateTemp("", "service-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(serviceFileContent)
	assert.NoError(t, err)
	tmpFile.Close()

	cli.ServiceFile = tmpFile.Name() // Set global for the command

	oldStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	assert.NoError(t, pipeErr)
	os.Stdout = w

	cmdErr := cli.RunServiceCreate(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	assert.NoError(t, cmdErr)

	var output strings.Builder
	_, err = io.Copy(&output, r)
	assert.NoError(t, err)
	fmt.Println(output.String())
	assert.Contains(t, output.String(), "✓ Service 'new-api' created successfully")
}

// TestRunServiceUpdate tests the service update command
func TestRunServiceUpdate(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/services/updated-api", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)

		var svc types.Service
		err := json.NewDecoder(r.Body).Decode(&svc)
		assert.NoError(t, err)
		assert.Equal(t, "updated-api", svc.Metadata.Name)
		assert.Equal(t, "The A-Team", svc.Spec.Owner.Team)

		resp := struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}{
			Success: true,
			Message: "Service updated",
		}

		_ = json.NewEncoder(w).Encode(resp)
	})

	server := setupTestServer(t, handler)
	defer server.Close()

	t.Setenv("PF_API_URL", server.URL)
	t.Setenv("PF_API_TOKEN", "test-token")

	// Create a temporary service file
	serviceFileContent := `
apiVersion: platformfoundry.io/v1
kind: Service
metadata:
  name: updated-api
spec:
  type: microservice
  owner:
    team: The A-Team
`
	tmpFile, err := os.CreateTemp("", "service-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(serviceFileContent)
	assert.NoError(t, err)
	tmpFile.Close()

	cli.ServiceFile = tmpFile.Name()

	oldStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	assert.NoError(t, pipeErr)
	os.Stdout = w

	cmdErr := cli.RunServiceUpdate(nil, nil)

	w.Close()
	os.Stdout = oldStdout

	assert.NoError(t, cmdErr)

	var output strings.Builder
	_, err = io.Copy(&output, r)
	assert.NoError(t, err)

	assert.Contains(t, output.String(), "✓ Service 'updated-api' updated successfully")
}

// TestRunServiceDelete tests the service delete command
func TestRunServiceDelete(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/services/deleted-api", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		resp := struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}{
			Success: true,
			Message: "Service deleted",
		}

		_ = json.NewEncoder(w).Encode(resp)
	})

	server := setupTestServer(t, handler)
	defer server.Close()

	t.Setenv("PF_API_URL", server.URL)
	t.Setenv("PF_API_TOKEN", "test-token")

	// Simulate user input
	input := "yes\n"
	r, w, err := os.Pipe()
	assert.NoError(t, err)

	_, err = w.WriteString(input)
	assert.NoError(t, err)
	w.Close()

	oldStdin := os.Stdin
	os.Stdin = r

	oldStdout := os.Stdout
	rout, wout, pipeErr := os.Pipe()
	assert.NoError(t, pipeErr)
	os.Stdout = wout

	cmdErr := cli.RunServiceDelete(nil, []string{"deleted-api"})

	os.Stdin = oldStdin
	wout.Close()
	os.Stdout = oldStdout

	assert.NoError(t, cmdErr)

	var output strings.Builder
	_, err = io.Copy(&output, rout)
	assert.NoError(t, err)

	assert.Contains(t, output.String(), "✓ Service 'deleted-api' deleted successfully")
}
