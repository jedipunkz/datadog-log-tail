package datadog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jedipunkz/datadog-log-tail/internal/config"
)

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		Timeout: 30,
		Site:    "datadoghq.com",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	if client.GetBaseURL() != "https://api.datadoghq.com" {
		t.Errorf("BaseURL = %v, want https://api.datadoghq.com", client.GetBaseURL())
	}

	if client.GetConfig() != cfg {
		t.Error("Config not properly set")
	}
}

func TestNewClient_CustomSite(t *testing.T) {
	cfg := &config.Config{
		Timeout: 30,
		Site:    "us3.datadoghq.com",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client.GetBaseURL() != "https://api.us3.datadoghq.com" {
		t.Errorf("BaseURL = %v, want https://api.us3.datadoghq.com", client.GetBaseURL())
	}
}

func TestClient_createRequest(t *testing.T) {
	cfg := &config.Config{
		APIKey:  "test-api-key",
		AppKey:  "test-app-key",
		Timeout: 30,
		Site:    "datadoghq.com",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	req, err := client.createRequest(ctx, "GET", "/api/v2/logs/events/search")
	if err != nil {
		t.Fatalf("createRequest() error = %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("Method = %v, want GET", req.Method)
	}

	expectedURL := "https://api.datadoghq.com/api/v2/logs/events/search"
	if req.URL.String() != expectedURL {
		t.Errorf("URL = %v, want %v", req.URL.String(), expectedURL)
	}

	if req.Header.Get("DD-API-KEY") != "test-api-key" {
		t.Errorf("DD-API-KEY header = %v, want test-api-key", req.Header.Get("DD-API-KEY"))
	}

	if req.Header.Get("DD-APPLICATION-KEY") != "test-app-key" {
		t.Errorf("DD-APPLICATION-KEY header = %v, want test-app-key", req.Header.Get("DD-APPLICATION-KEY"))
	}

	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type header = %v, want application/json", req.Header.Get("Content-Type"))
	}
}

func TestClient_doRequest_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data": []}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		APIKey:  "test-api-key",
		AppKey:  "test-app-key",
		Timeout: 30,
	}

	client := &Client{
		config:     cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    server.URL,
	}

	req, err := client.createRequest(context.Background(), "GET", "/test")
	if err != nil {
		t.Fatalf("createRequest() error = %v", err)
	}

	resp, err := client.doRequest(req)
	if err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %v, want %v", resp.StatusCode, http.StatusOK)
	}
}

func TestClient_doRequest_Error(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errors":["Forbidden"]}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		APIKey:  "test-api-key",
		AppKey:  "test-app-key",
		Timeout: 30,
	}

	client := &Client{
		config:     cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    server.URL,
	}

	req, err := client.createRequest(context.Background(), "GET", "/test")
	if err != nil {
		t.Fatalf("createRequest() error = %v", err)
	}

	_, err = client.doRequest(req)
	if err == nil {
		t.Fatal("doRequest() expected error but got none")
	}

	expectedError := "403 Forbidden"
	if !contains(err.Error(), expectedError) {
		t.Errorf("Error = %v, want to contain %v", err.Error(), expectedError)
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
