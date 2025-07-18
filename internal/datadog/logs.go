package datadog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/jedipunkz/datadog-log-tail/internal/config"
	"github.com/jedipunkz/datadog-log-tail/internal/output"
	"github.com/jedipunkz/datadog-log-tail/pkg/utils"
)

// LogEntry represents a Datadog v2 log entry
type LogEntry struct {
	ID         string                 `json:"id"`
	Timestamp  int64                  `json:"timestamp"`
	Message    string                 `json:"message"`
	Service    string                 `json:"service"`
	Status     string                 `json:"status"`
	Tags       []string               `json:"tags"`
	Attributes map[string]interface{} `json:"attributes"`
}

// v2 API response structure
// https://docs.datadoghq.com/ja/api/latest/logs/#search-logs

type v2Log struct {
	ID    string          `json:"id"`
	Type  string          `json:"type"`
	Attrs v2LogAttributes `json:"attributes"`
}

type v2LogAttributes struct {
	Timestamp  string                 `json:"timestamp"`
	Message    string                 `json:"message"`
	Service    string                 `json:"service"`
	Status     string                 `json:"status"`
	Tags       []string               `json:"tags"`
	Attributes map[string]interface{} `json:"attributes"`
	// Additional fields
	Host     string `json:"host"`
	Source   string `json:"source"`
	LogLevel string `json:"level"`
	// Alternative message fields
	Content string `json:"content"`
	Text    string `json:"text"`
	Log     string `json:"log"`
}

type v2LogsResponse struct {
	Data []v2Log `json:"data"`
	Meta struct {
		Page struct {
			After string `json:"after"`
		} `json:"page"`
	} `json:"meta"`
}

// Implement LogEntry interface methods
func (l LogEntry) GetID() string                         { return l.ID }
func (l LogEntry) GetTimestamp() int64                   { return l.Timestamp }
func (l LogEntry) GetMessage() string                    { return l.Message }
func (l LogEntry) GetService() string                    { return l.Service }
func (l LogEntry) GetStatus() string                     { return l.Status }
func (l LogEntry) GetTags() []string                     { return l.Tags }
func (l LogEntry) GetAttributes() map[string]interface{} { return l.Attributes }

// TailLogs tails logs in real-time
func (c *Client) TailLogs() error {
	ctx := context.Background()
	formatter := output.NewFormatter(c.config.GetOutputFormat())

	fmt.Println("Starting Datadog Logs tail...")
	fmt.Printf("Output format: %s\n", c.config.GetOutputFormat())
	if c.config.GetTags() != "" {
		fmt.Printf("Tag filter: %s\n", c.config.GetTags())
	}
	if c.config.GetLogLevel() != "" {
		fmt.Printf("Log level: %s\n", c.config.GetLogLevel())
	}
	fmt.Println("---")

	var lastTimestamp time.Time
	retryCount := 0
	maxRetries := c.config.GetRetryCount()
	baseInterval := 3 * time.Second // Conservative base interval to avoid rate limits
	currentInterval := baseInterval
	maxInterval := 30 * time.Second     // Reasonable maximum interval
	minInterval := 2 * time.Second      // Safer minimum interval to respect rate limits
	rateLimitBackoff := 5 * time.Second // Initial backoff when rate limited
	consecutiveSuccesses := 0           // Track consecutive successful requests
	searchWindow := 30 * time.Second    // Dynamic search window

	for {
		if retryCount >= maxRetries {
			return fmt.Errorf("maximum retry count (%d) reached", maxRetries)
		}

		from := lastTimestamp
		if from.IsZero() {
			from = time.Now().Add(-searchWindow) // Start with dynamic search window
		} else {
			// Add 1 nanosecond to avoid duplicate logs
			from = lastTimestamp.Add(1 * time.Nanosecond)
		}
		to := time.Now()

		logs, latest, err := c.fetchLogsV2(ctx, from, to)
		if err != nil {
			// Smart rate limit handling with adaptive backoff
			if strings.Contains(err.Error(), "429") {
				// Exponential backoff with jitter for rate limiting
				rateLimitBackoff = time.Duration(math.Min(float64(60*time.Second), float64(rateLimitBackoff)*1.5))
				jitter := time.Duration(rand.Intn(int(rateLimitBackoff / 10))) // Small jitter
				waitTime := rateLimitBackoff + jitter

				fmt.Fprintf(os.Stderr, "Rate limit reached. Backing off for %v...\n", waitTime)
				time.Sleep(waitTime)

				// After rate limit, use conservative interval and reset success counter
				currentInterval = time.Duration(math.Max(float64(baseInterval*2), float64(currentInterval)))
				if currentInterval > maxInterval {
					currentInterval = maxInterval
				}
				consecutiveSuccesses = 0
				continue
			}

			retryCount++
			fmt.Fprintf(os.Stderr, "Failed to fetch logs (attempt %d/%d): %v\n", retryCount, maxRetries, err)
			backoff := utils.CalculateBackoff(retryCount)
			fmt.Fprintf(os.Stderr, "Retrying in %v...\n", backoff)
			time.Sleep(backoff)
			continue
		}

		// Reset retry counter on success and increment consecutive successes
		retryCount = 0
		consecutiveSuccesses++

		// Reset rate limit backoff on successful requests
		if consecutiveSuccesses >= 3 {
			rateLimitBackoff = 5 * time.Second // Reset to initial backoff
		}

		// Smart adaptive interval and search window based on log activity and consecutive successes
		if len(logs) > 0 {
			// Logs found: optimize for real-time response
			if consecutiveSuccesses >= 5 {
				currentInterval = time.Duration(float64(currentInterval) * 0.85) // Moderate reduction
				if currentInterval < minInterval {
					currentInterval = minInterval
				}
			}
			// Reduce search window when finding logs frequently
			if len(logs) >= 5 && searchWindow > 15*time.Second {
				searchWindow = time.Duration(float64(searchWindow) * 0.9)
			}
		} else {
			// No logs: gradually increase interval to reduce API calls
			currentInterval = time.Duration(float64(currentInterval) * 1.05) // Very gentle increase
			if currentInterval > maxInterval {
				currentInterval = maxInterval
			}
			// Increase search window when no logs are found
			if searchWindow < 60*time.Second {
				searchWindow = time.Duration(float64(searchWindow) * 1.1)
			}
		}

		// Output logs immediately as they arrive for better real-time experience
		for _, log := range logs {
			formatted, err := formatter.Format(log)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to format log: %v\n", err)
				continue
			}
			// Flush output immediately for real-time display
			fmt.Println(formatted)
			if err := os.Stdout.Sync(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to sync stdout: %v\n", err)
			}
		}

		// Update lastTimestamp to avoid duplicate logs
		if !latest.IsZero() {
			if lastTimestamp.IsZero() || latest.After(lastTimestamp) {
				lastTimestamp = latest
			}
		} else if len(logs) == 0 {
			// If no logs returned, advance time slightly to avoid infinite loop
			if lastTimestamp.IsZero() {
				lastTimestamp = time.Now().Add(-30 * time.Second)
			} else {
				// Move forward by a small amount when no new logs
				lastTimestamp = time.Now().Add(-10 * time.Second)
			}
		}
		time.Sleep(currentInterval)
	}
}

// FetchLogsV2 fetches logs from Datadog Logs API v2 (public method for TUI)
func (c *Client) FetchLogsV2(ctx context.Context, from, to time.Time) ([]LogEntry, time.Time, error) {
	return c.fetchLogsV2(ctx, from, to)
}

// fetchLogsV2 fetches logs from Datadog Logs API v2
func (c *Client) fetchLogsV2(ctx context.Context, from, to time.Time) ([]LogEntry, time.Time, error) {
	endpoint := "/api/v2/logs/events/search"

	query := c.buildQueryV2()
	body := map[string]interface{}{
		"filter": map[string]interface{}{
			"from":  from.UTC().Format(time.RFC3339),
			"to":    to.UTC().Format(time.RFC3339),
			"query": query,
		},
		"page": map[string]interface{}{
			"limit": 100, // Balanced limit to avoid overwhelming the API
		},
		"sort": "timestamp",
	}
	jsonBody, _ := json.Marshal(body)

	req, err := c.createRequest(ctx, "POST", endpoint)
	if err != nil {
		return nil, time.Time{}, err
	}
	req.Body = io.NopCloser(bytes.NewReader(jsonBody))
	req.ContentLength = int64(len(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, time.Time{}, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var v2resp v2LogsResponse
	if err := json.NewDecoder(resp.Body).Decode(&v2resp); err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to parse response: %w", err)
	}

	var logs []LogEntry
	var latest time.Time
	for _, d := range v2resp.Data {
		ts, _ := time.Parse(time.RFC3339Nano, d.Attrs.Timestamp)

		// Extract message from multiple possible fields
		message := d.Attrs.Message
		if message == "" {
			message = d.Attrs.Content
		}
		if message == "" {
			message = d.Attrs.Text
		}
		if message == "" {
			message = d.Attrs.Log
		}
		if message == "" {
			message = "No message content"
		}

		// Extract service name
		service := d.Attrs.Service
		if service == "" {
			service = d.Attrs.Host
		}

		// Extract status/log level
		status := d.Attrs.Status
		if status == "" {
			status = d.Attrs.LogLevel
		}

		log := LogEntry{
			ID:         d.ID,
			Timestamp:  ts.Unix(),
			Message:    message,
			Service:    service,
			Status:     status,
			Tags:       d.Attrs.Tags,
			Attributes: d.Attrs.Attributes,
		}
		logs = append(logs, log)
		if ts.After(latest) {
			latest = ts
		}
	}
	return logs, latest, nil
}

// GetLogs fetches logs for TUI mode with custom config
func (c *Client) GetLogs(cfg *config.Config) ([]map[string]interface{}, error) {
	ctx := context.Background()
	from := time.Now().Add(-5 * time.Second) // Further reduced to 5s for better real-time
	to := time.Now()

	// Temporarily update client config
	originalTags := c.config.GetTags()
	originalLevel := c.config.GetLogLevel()

	// Apply temporary config
	if cfg.GetTags() != "" {
		c.config.Tags = cfg.GetTags()
	}
	if cfg.GetLogLevel() != "" {
		c.config.LogLevel = cfg.GetLogLevel()
		c.config.LogLevels = cfg.GetLogLevels()
	}

	logs, _, err := c.fetchLogsV2(ctx, from, to)

	// Restore original config
	c.config.Tags = originalTags
	c.config.LogLevel = originalLevel

	if err != nil {
		return nil, err
	}

	// Convert LogEntry to map for TUI
	var result []map[string]interface{}
	for _, log := range logs {
		logMap := map[string]interface{}{
			"id":         log.ID,
			"timestamp":  time.Unix(log.Timestamp, 0).Format("15:04:05"),
			"message":    log.Message,
			"service":    log.Service,
			"level":      log.Status,
			"tags":       log.Tags,
			"attributes": log.Attributes,
		}
		result = append(result, logMap)
	}

	return result, nil
}

// GetLogsFromTimestamp retrieves logs from a time range (batch mode)
func (c *Client) GetLogsFromTimestamp() error {
	ctx := context.Background()
	formatter := output.NewFormatter(c.config.GetOutputFormat())

	// Parse the time range from config
	timestampStr := c.config.GetTimestamp()
	
	// Ensure it's a range format (must contain comma)
	if !strings.Contains(timestampStr, ",") {
		return fmt.Errorf("timestamp must be a time range in format: from,to (e.g. 2024-01-15T10:00:00Z,2024-01-15T11:00:00Z)")
	}

	// Parse time range
	parts := strings.Split(timestampStr, ",")
	if len(parts) != 2 {
		return fmt.Errorf("invalid timestamp range format (use: from,to in RFC3339, e.g. 2024-01-15T10:00:00Z,2024-01-15T11:00:00Z)")
	}

	from, err := time.Parse(time.RFC3339, strings.TrimSpace(parts[0]))
	if err != nil {
		return fmt.Errorf("invalid start timestamp format (use RFC3339, e.g. 2024-01-15T10:00:00Z): %w", err)
	}

	to, err := time.Parse(time.RFC3339, strings.TrimSpace(parts[1]))
	if err != nil {
		return fmt.Errorf("invalid end timestamp format (use RFC3339, e.g. 2024-01-15T10:00:00Z): %w", err)
	}

	if !to.After(from) {
		return fmt.Errorf("end timestamp must be after start timestamp")
	}

	fmt.Printf("Retrieving logs from %s to %s...\n", from.Format(time.RFC3339), to.Format(time.RFC3339))
	fmt.Printf("Output format: %s\n", c.config.GetOutputFormat())
	if c.config.GetTags() != "" {
		fmt.Printf("Tag filter: %s\n", c.config.GetTags())
	}
	if c.config.GetLogLevel() != "" {
		fmt.Printf("Log level: %s\n", c.config.GetLogLevel())
	}
	fmt.Println("---")

	// Fetch all logs using pagination
	allLogs, err := c.fetchAllLogsV2(ctx, from, to)
	if err != nil {
		return fmt.Errorf("failed to fetch logs: %w", err)
	}

	if len(allLogs) == 0 {
		fmt.Println("No logs found for the specified time range.")
		return nil
	}

	// Output all logs
	for _, log := range allLogs {
		formatted, err := formatter.Format(log)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to format log: %v\n", err)
			continue
		}
		fmt.Println(formatted)
	}

	fmt.Printf("\nRetrieved %d log entries.\n", len(allLogs))
	return nil
}

// fetchAllLogsV2 fetches all logs from Datadog Logs API v2 using pagination with rate limiting
func (c *Client) fetchAllLogsV2(ctx context.Context, from, to time.Time) ([]LogEntry, error) {
	var allLogs []LogEntry
	var cursor string
	pageSize := 500 // Reduce page size to be more conservative
	retryCount := 0
	maxRetries := 5
	baseDelay := 2 * time.Second
	
	for {
		logs, nextCursor, err := c.fetchLogsV2WithPagination(ctx, from, to, cursor, pageSize)
		if err != nil {
			// Handle rate limiting with exponential backoff
			if strings.Contains(err.Error(), "429") {
				if retryCount >= maxRetries {
					return nil, fmt.Errorf("maximum retry count reached due to rate limiting: %w", err)
				}
				
				// Exponential backoff with jitter
				backoffDelay := time.Duration(float64(baseDelay) * math.Pow(2, float64(retryCount)))
				jitter := time.Duration(rand.Intn(int(backoffDelay / 4))) // Add up to 25% jitter
				totalDelay := backoffDelay + jitter
				
				fmt.Fprintf(os.Stderr, "Rate limit reached. Retrying in %v... (attempt %d/%d)\n", totalDelay, retryCount+1, maxRetries)
				time.Sleep(totalDelay)
				retryCount++
				continue
			}
			return nil, err
		}
		
		// Reset retry count on successful request
		retryCount = 0
		allLogs = append(allLogs, logs...)
		
		// If no next cursor, we've reached the end
		if nextCursor == "" {
			break
		}
		cursor = nextCursor
		
		// Show progress for large datasets
		if len(allLogs)%500 == 0 {
			fmt.Fprintf(os.Stderr, "Retrieved %d log entries so far...\n", len(allLogs))
		}
		
		// Add a small delay between requests to avoid hitting rate limits
		time.Sleep(500 * time.Millisecond)
	}
	
	return allLogs, nil
}

// fetchLogsV2WithPagination fetches logs with pagination support
func (c *Client) fetchLogsV2WithPagination(ctx context.Context, from, to time.Time, cursor string, limit int) ([]LogEntry, string, error) {
	endpoint := "/api/v2/logs/events/search"

	query := c.buildQueryV2()
	body := map[string]interface{}{
		"filter": map[string]interface{}{
			"from":  from.UTC().Format(time.RFC3339),
			"to":    to.UTC().Format(time.RFC3339),
			"query": query,
		},
		"page": map[string]interface{}{
			"limit": limit,
		},
		"sort": "timestamp",
	}
	
	// Add cursor for pagination if available
	if cursor != "" {
		body["page"].(map[string]interface{})["cursor"] = cursor
	}
	
	jsonBody, _ := json.Marshal(body)

	req, err := c.createRequest(ctx, "POST", endpoint)
	if err != nil {
		return nil, "", err
	}
	req.Body = io.NopCloser(bytes.NewReader(jsonBody))
	req.ContentLength = int64(len(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var v2resp v2LogsResponse
	if err := json.NewDecoder(resp.Body).Decode(&v2resp); err != nil {
		return nil, "", fmt.Errorf("failed to parse response: %w", err)
	}

	var logs []LogEntry
	var latest time.Time
	for _, d := range v2resp.Data {
		ts, _ := time.Parse(time.RFC3339Nano, d.Attrs.Timestamp)

		// Extract message from multiple possible fields
		message := d.Attrs.Message
		if message == "" {
			message = d.Attrs.Content
		}
		if message == "" {
			message = d.Attrs.Text
		}
		if message == "" {
			message = d.Attrs.Log
		}
		if message == "" {
			message = "No message content"
		}

		// Extract service name
		service := d.Attrs.Service
		if service == "" {
			service = d.Attrs.Host
		}

		// Extract status/log level
		status := d.Attrs.Status
		if status == "" {
			status = d.Attrs.LogLevel
		}

		log := LogEntry{
			ID:         d.ID,
			Timestamp:  ts.Unix(),
			Message:    message,
			Service:    service,
			Status:     status,
			Tags:       d.Attrs.Tags,
			Attributes: d.Attrs.Attributes,
		}
		logs = append(logs, log)
		if ts.After(latest) {
			latest = ts
		}
	}
	
	// Return the next cursor for pagination
	nextCursor := v2resp.Meta.Page.After
	return logs, nextCursor, nil
}

// buildQueryV2 builds Datadog v2 query
func (c *Client) buildQueryV2() string {
	var conditions []string
	if c.config.GetTags() != "" {
		tags := strings.Split(c.config.GetTags(), ",")
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				conditions = append(conditions, tag)
			}
		}
	}
	
	// Handle multiple log levels
	levels := c.config.GetLogLevels()
	if len(levels) > 0 {
		if len(levels) == 1 {
			conditions = append(conditions, fmt.Sprintf("status:%s", levels[0]))
		} else {
			// Multiple levels: (status:error OR status:warn OR status:info)
			var levelConditions []string
			for _, level := range levels {
				levelConditions = append(levelConditions, fmt.Sprintf("status:%s", level))
			}
			conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(levelConditions, " OR ")))
		}
	} else if c.config.GetLogLevel() != "" {
		// Fallback for backward compatibility
		conditions = append(conditions, fmt.Sprintf("status:%s", c.config.GetLogLevel()))
	}
	
	return strings.Join(conditions, " ")
}
