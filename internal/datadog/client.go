package datadog

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/jedipunkz/datadog-log-tail/internal/config"
)

// RateLimitInfo tracks rate limit information from API responses
type RateLimitInfo struct {
	Limit     int       // Maximum number of requests allowed
	Remaining int       // Number of requests remaining
	Reset     time.Time // Time when the rate limit resets
	mu        sync.RWMutex
}

// Client represents a Datadog API client
type Client struct {
	config        *config.Config
	httpClient    *http.Client
	baseURL       string
	rateLimitInfo *RateLimitInfo
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
		config:        cfg,
		httpClient:    httpClient,
		baseURL:       baseURL,
		rateLimitInfo: &RateLimitInfo{},
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

	// Update rate limit info from response headers
	c.updateRateLimitInfo(resp)

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

// updateRateLimitInfo extracts rate limit information from response headers
func (c *Client) updateRateLimitInfo(resp *http.Response) {
	c.rateLimitInfo.mu.Lock()
	defer c.rateLimitInfo.mu.Unlock()

	if limitStr := resp.Header.Get("X-RateLimit-Limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			c.rateLimitInfo.Limit = limit
		}
	}

	if remainingStr := resp.Header.Get("X-RateLimit-Remaining"); remainingStr != "" {
		if remaining, err := strconv.Atoi(remainingStr); err == nil {
			c.rateLimitInfo.Remaining = remaining
		}
	}

	if resetStr := resp.Header.Get("X-RateLimit-Reset"); resetStr != "" {
		if resetUnix, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
			c.rateLimitInfo.Reset = time.Unix(resetUnix, 0)
		}
	}
}

// getRateLimitInfo returns a copy of the current rate limit info
func (c *Client) getRateLimitInfo() (limit, remaining int, reset time.Time) {
	c.rateLimitInfo.mu.RLock()
	defer c.rateLimitInfo.mu.RUnlock()
	return c.rateLimitInfo.Limit, c.rateLimitInfo.Remaining, c.rateLimitInfo.Reset
}

// shouldBackoffForRateLimit checks if we should preemptively back off based on remaining quota
func (c *Client) shouldBackoffForRateLimit() (bool, time.Duration) {
	limit, remaining, reset := c.getRateLimitInfo()

	// If we don't have rate limit info yet, don't back off
	if limit == 0 {
		return false, 0
	}

	// Calculate the percentage of quota remaining
	percentRemaining := float64(remaining) / float64(limit) * 100

	// If we're below 20% of quota, introduce preemptive backoff
	if percentRemaining < 20 && percentRemaining >= 10 {
		return true, 2 * time.Second
	} else if percentRemaining < 10 && percentRemaining >= 5 {
		return true, 5 * time.Second
	} else if percentRemaining < 5 && remaining > 0 {
		return true, 10 * time.Second
	} else if remaining == 0 && !reset.IsZero() && time.Now().Before(reset) {
		// If we've exhausted the quota, wait until reset
		waitTime := time.Until(reset) + time.Second // Add 1 second buffer
		return true, waitTime
	}

	return false, 0
}

// GetBaseURL returns the base URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// GetConfig returns the configuration
func (c *Client) GetConfig() *config.Config {
	return c.config
}
