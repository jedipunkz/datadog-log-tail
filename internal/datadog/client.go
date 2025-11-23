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

// RateLimitInfo tracks rate limit information from API responses.
// This struct is thread-safe for concurrent access through its internal mutex.
// The rate limit information is automatically extracted from Datadog API response
// headers (X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset) and can
// be safely accessed by multiple goroutines.
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

// Rate limit backoff thresholds and durations
const (
	rateLimitLowThreshold      = 20.0 // Percentage threshold for low quota warning
	rateLimitVeryLowThreshold  = 10.0 // Percentage threshold for very low quota
	rateLimitCriticalThreshold = 5.0  // Percentage threshold for critical quota

	rateLimitLowBackoff      = 2 * time.Second  // Backoff duration for low quota
	rateLimitVeryLowBackoff  = 5 * time.Second  // Backoff duration for very low quota
	rateLimitCriticalBackoff = 10 * time.Second // Backoff duration for critical quota
	rateLimitResetBuffer     = 1 * time.Second  // Buffer time after rate limit reset
)

// shouldBackoffForRateLimit checks if we should preemptively back off based on remaining quota.
// Returns whether to backoff, the backoff duration, and the current rate limit info.
func (c *Client) shouldBackoffForRateLimit() (bool, time.Duration, int, int, time.Time) {
	limit, remaining, reset := c.getRateLimitInfo()

	// If we don't have rate limit info yet, don't back off
	if limit == 0 {
		return false, 0, limit, remaining, reset
	}

	// Calculate the percentage of quota remaining
	percentRemaining := float64(remaining) / float64(limit) * 100

	// If we're below threshold, introduce preemptive backoff
	if percentRemaining < rateLimitLowThreshold && percentRemaining >= rateLimitVeryLowThreshold {
		return true, rateLimitLowBackoff, limit, remaining, reset
	} else if percentRemaining < rateLimitVeryLowThreshold && percentRemaining >= rateLimitCriticalThreshold {
		return true, rateLimitVeryLowBackoff, limit, remaining, reset
	} else if percentRemaining < rateLimitCriticalThreshold && remaining > 0 {
		return true, rateLimitCriticalBackoff, limit, remaining, reset
	} else if remaining == 0 && !reset.IsZero() && time.Now().Before(reset) {
		// If we've exhausted the quota, wait until reset
		waitTime := time.Until(reset) + rateLimitResetBuffer
		return true, waitTime, limit, remaining, reset
	}

	return false, 0, limit, remaining, reset
}

// GetBaseURL returns the base URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// GetConfig returns the configuration
func (c *Client) GetConfig() *config.Config {
	return c.config
}
