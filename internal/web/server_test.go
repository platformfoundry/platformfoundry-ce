package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewServer(t *testing.T) {
	config := Config{}
	server := NewServer(config)

	if server == nil {
		t.Fatal("NewServer() returned nil")
	}

	if server.port != 8080 {
		t.Errorf("Expected default port 8080, got %d", server.port)
	}

	if server.platforms == nil {
		t.Error("platforms should be initialized")
	}

	if server.jobs == nil {
		t.Error("jobs should be initialized")
	}
}

func TestNewServerCustomPort(t *testing.T) {
	config := Config{
		Port: 9090,
	}
	server := NewServer(config)

	if server.port != 9090 {
		t.Errorf("Expected port 9090, got %d", server.port)
	}
}

func TestHandleHealth(t *testing.T) {
	server := NewServer(Config{})

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result APIResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if !result.Success {
		t.Error("Expected success true")
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be map")
	}

	if data["status"] != "healthy" {
		t.Error("Expected status healthy")
	}
}

func TestHandleStats(t *testing.T) {
	server := NewServer(Config{})

	// Add some test data
	server.platforms["test1"] = map[string]interface{}{"name": "test1"}
	server.platforms["test2"] = map[string]interface{}{"name": "test2"}
	server.jobs["job1"] = map[string]interface{}{"id": "job1"}

	req := httptest.NewRequest("GET", "/api/stats", nil)
	w := httptest.NewRecorder()

	server.handleStats(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result APIResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if !result.Success {
		t.Error("Expected success true")
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be map")
	}

	if platforms, ok := data["platforms"].(float64); !ok || platforms != 2 {
		t.Errorf("Expected 2 platforms, got %v", data["platforms"])
	}

	if jobs, ok := data["jobs"].(float64); !ok || jobs != 1 {
		t.Errorf("Expected 1 job, got %v", data["jobs"])
	}
}

func TestListPlatforms(t *testing.T) {
	server := NewServer(Config{})

	// Add test platforms
	server.platforms["platform1"] = map[string]interface{}{"name": "platform1"}
	server.platforms["platform2"] = map[string]interface{}{"name": "platform2"}

	req := httptest.NewRequest("GET", "/api/platforms", nil)
	w := httptest.NewRecorder()

	server.handlePlatforms(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result APIResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if !result.Success {
		t.Error("Expected success true")
	}

	platforms, ok := result.Data.([]interface{})
	if !ok {
		t.Fatal("Expected data to be array")
	}

	if len(platforms) != 2 {
		t.Errorf("Expected 2 platforms, got %d", len(platforms))
	}
}

func TestCreatePlatform(t *testing.T) {
	server := NewServer(Config{})

	platform := map[string]interface{}{
		"name": "test-platform",
		"type": "Platform",
	}

	body, _ := json.Marshal(platform)
	req := httptest.NewRequest("POST", "/api/platforms", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handlePlatforms(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result APIResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if !result.Success {
		t.Error("Expected success true")
	}

	// Check platform was added
	if _, exists := server.platforms["test-platform"]; !exists {
		t.Error("Platform should be added to server state")
	}
}

func TestCreatePlatformMissingName(t *testing.T) {
	server := NewServer(Config{})

	platform := map[string]interface{}{
		"type": "Platform",
	}

	body, _ := json.Marshal(platform)
	req := httptest.NewRequest("POST", "/api/platforms", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handlePlatforms(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	var result APIResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Success {
		t.Error("Expected success false")
	}
}

func TestGetPlatform(t *testing.T) {
	server := NewServer(Config{})

	// Add test platform
	testPlatform := map[string]interface{}{
		"name": "test-platform",
		"type": "Platform",
	}
	server.platforms["test-platform"] = testPlatform

	req := httptest.NewRequest("GET", "/api/platforms/test-platform", nil)
	w := httptest.NewRecorder()

	server.handlePlatform(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result APIResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if !result.Success {
		t.Error("Expected success true")
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be map")
	}

	if data["name"] != "test-platform" {
		t.Error("Expected platform name test-platform")
	}
}

func TestGetPlatformNotFound(t *testing.T) {
	server := NewServer(Config{})

	req := httptest.NewRequest("GET", "/api/platforms/nonexistent", nil)
	w := httptest.NewRecorder()

	server.handlePlatform(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	var result APIResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Success {
		t.Error("Expected success false")
	}
}

func TestDeletePlatform(t *testing.T) {
	server := NewServer(Config{})

	// Add test platform
	server.platforms["test-platform"] = map[string]interface{}{"name": "test-platform"}

	req := httptest.NewRequest("DELETE", "/api/platforms/test-platform", nil)
	w := httptest.NewRecorder()

	server.handlePlatform(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result APIResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if !result.Success {
		t.Error("Expected success true")
	}

	// Check platform was deleted
	if _, exists := server.platforms["test-platform"]; exists {
		t.Error("Platform should be deleted from server state")
	}
}

func TestListJobs(t *testing.T) {
	server := NewServer(Config{})

	// Add test jobs
	server.jobs["job1"] = map[string]interface{}{"id": "job1"}
	server.jobs["job2"] = map[string]interface{}{"id": "job2"}

	req := httptest.NewRequest("GET", "/api/jobs", nil)
	w := httptest.NewRecorder()

	server.handleJobs(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result APIResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if !result.Success {
		t.Error("Expected success true")
	}

	jobs, ok := result.Data.([]interface{})
	if !ok {
		t.Fatal("Expected data to be array")
	}

	if len(jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(jobs))
	}
}

func TestValidate(t *testing.T) {
	server := NewServer(Config{})

	payload := map[string]interface{}{
		"yaml": "test yaml content",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleValidate(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result APIResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if !result.Success {
		t.Error("Expected success true")
	}
}

func TestApply(t *testing.T) {
	server := NewServer(Config{})

	payload := map[string]interface{}{
		"yaml": "test yaml content",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/apply", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleApply(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result APIResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if !result.Success {
		t.Error("Expected success true")
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be map")
	}

	if data["job_id"] == nil {
		t.Error("Expected job_id in response")
	}

	// Check job was created
	if len(server.jobs) == 0 {
		t.Error("Job should be created")
	}
}

func TestPlan(t *testing.T) {
	server := NewServer(Config{})

	payload := map[string]interface{}{
		"yaml": "test yaml content",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/plan", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handlePlan(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result APIResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if !result.Success {
		t.Error("Expected success true")
	}
}

func TestGetPort(t *testing.T) {
	config := Config{
		Port: 9000,
	}
	server := NewServer(config)

	if server.GetPort() != 9000 {
		t.Errorf("Expected port 9000, got %d", server.GetPort())
	}
}
