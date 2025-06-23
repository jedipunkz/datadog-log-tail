package datadog

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"datadog-log-tail/internal/config"
)

// Client represents a Datadog API client
type Client struct {
	config     *config.Config
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Datadog client
func NewClient(cfg *config.Config) (*Client, error) {
	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: time.Duration(cfg.GetTimeout()) * time.Second,
	}

	// Determine base URL based on site
	baseURL := fmt.Sprintf("https://api.%s", cfg.GetSite())

	return &Client{
		config:     cfg,
		httpClient: httpClient,
		baseURL:    baseURL,
	}, nil
}

// createRequest creates an HTTP request with authentication headers
func (c *Client) createRequest(ctx context.Context, method, endpoint string) (*http.Request, error) {
	url := c.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication headers
	req.Header.Set("DD-API-KEY", c.config.GetAPIKey())
	req.Header.Set("DD-APPLICATION-KEY", c.config.GetAppKey())
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

// doRequest executes an HTTP request
func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	// Output debug information
	fmt.Fprintf(os.Stderr, "API request: %s %s\n", req.Method, req.URL.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Read response body and get error details
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		bodyStr := string(body[:n])

		return nil, fmt.Errorf("API error: %s - %s", resp.Status, bodyStr)
	}

	return resp, nil
}

// GetBaseURL returns the base URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// GetConfig returns the configuration
func (c *Client) GetConfig() *config.Config {
	return c.config
}
