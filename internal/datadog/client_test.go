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
		config:        cfg,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		baseURL:       server.URL,
		rateLimitInfo: &RateLimitInfo{},
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
		config:        cfg,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		baseURL:       server.URL,
		rateLimitInfo: &RateLimitInfo{},
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

func TestClient_updateRateLimitInfo(t *testing.T) {
	// Create test server that returns rate limit headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "1000")
		w.Header().Set("X-RateLimit-Remaining", "500")
		w.Header().Set("X-RateLimit-Reset", "1700000000")
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
		config:        cfg,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		baseURL:       server.URL,
		rateLimitInfo: &RateLimitInfo{},
	}

	req, err := client.createRequest(context.Background(), "GET", "/test")
	if err != nil {
		t.Fatalf("createRequest() error = %v", err)
	}

	_, err = client.doRequest(req)
	if err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}

	limit, remaining, reset := client.getRateLimitInfo()
	if limit != 1000 {
		t.Errorf("Limit = %d, want 1000", limit)
	}
	if remaining != 500 {
		t.Errorf("Remaining = %d, want 500", remaining)
	}
	if reset.Unix() != 1700000000 {
		t.Errorf("Reset = %v, want %v", reset.Unix(), 1700000000)
	}
}

func TestClient_shouldBackoffForRateLimit(t *testing.T) {
	cfg := &config.Config{
		APIKey:  "test-api-key",
		AppKey:  "test-app-key",
		Timeout: 30,
	}

	tests := []struct {
		name              string
		limit             int
		remaining         int
		reset             time.Time
		expectBackoff     bool
		minBackoffSeconds float64
	}{
		{
			name:          "No rate limit info",
			limit:         0,
			remaining:     0,
			reset:         time.Time{},
			expectBackoff: false,
		},
		{
			name:          "High quota remaining (80%)",
			limit:         1000,
			remaining:     800,
			reset:         time.Now().Add(time.Hour),
			expectBackoff: false,
		},
		{
			name:              "Low quota (15%)",
			limit:             1000,
			remaining:         150,
			reset:             time.Now().Add(time.Hour),
			expectBackoff:     true,
			minBackoffSeconds: 2,
		},
		{
			name:              "Very low quota (8%)",
			limit:             1000,
			remaining:         80,
			reset:             time.Now().Add(time.Hour),
			expectBackoff:     true,
			minBackoffSeconds: 5,
		},
		{
			name:              "Critical quota (3%)",
			limit:             1000,
			remaining:         30,
			reset:             time.Now().Add(time.Hour),
			expectBackoff:     true,
			minBackoffSeconds: 10,
		},
		{
			name:              "Quota exhausted",
			limit:             1000,
			remaining:         0,
			reset:             time.Now().Add(10 * time.Second),
			expectBackoff:     true,
			minBackoffSeconds: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(cfg)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			// Set rate limit info
			client.rateLimitInfo.Limit = tt.limit
			client.rateLimitInfo.Remaining = tt.remaining
			client.rateLimitInfo.Reset = tt.reset

			shouldBackoff, backoffDuration, limit, remaining, reset := client.shouldBackoffForRateLimit()

			if shouldBackoff != tt.expectBackoff {
				t.Errorf("shouldBackoff = %v, want %v", shouldBackoff, tt.expectBackoff)
			}

			if tt.expectBackoff && backoffDuration.Seconds() < tt.minBackoffSeconds {
				t.Errorf("backoffDuration = %v, want at least %v seconds", backoffDuration, tt.minBackoffSeconds)
			}

			// Verify returned rate limit info matches what we set
			if limit != tt.limit {
				t.Errorf("returned limit = %d, want %d", limit, tt.limit)
			}
			if remaining != tt.remaining {
				t.Errorf("returned remaining = %d, want %d", remaining, tt.remaining)
			}
			if !reset.Equal(tt.reset) {
				t.Errorf("returned reset = %v, want %v", reset, tt.reset)
			}
		})
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
