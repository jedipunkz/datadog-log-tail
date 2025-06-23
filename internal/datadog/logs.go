package datadog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"datadog-log-tail/internal/output"
	"datadog-log-tail/pkg/utils"
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
	baseInterval := 10 * time.Second // Changed to 10 seconds to avoid rate limiting
	currentInterval := baseInterval

	for {
		if retryCount >= maxRetries {
			return fmt.Errorf("maximum retry count (%d) reached", maxRetries)
		}

		from := lastTimestamp
		if from.IsZero() {
			from = time.Now().Add(-5 * time.Minute)
		}
		to := time.Now()

		logs, latest, err := c.fetchLogsV2(ctx, from, to)
		if err != nil {
			// Special handling for 429 errors
			if strings.Contains(err.Error(), "429") {
				fmt.Fprintf(os.Stderr, "Rate limit reached. Waiting 10 seconds...\n")
				time.Sleep(10 * time.Second)
				currentInterval = 10 * time.Second // Temporarily extend interval
				continue
			}

			retryCount++
			fmt.Fprintf(os.Stderr, "Failed to fetch logs (attempt %d/%d): %v\n", retryCount, maxRetries, err)
			backoff := utils.CalculateBackoff(retryCount)
			fmt.Fprintf(os.Stderr, "Retrying in %v...\n", backoff)
			time.Sleep(backoff)
			continue
		}

		// Reset retry counter and restore base interval on success
		retryCount = 0
		currentInterval = baseInterval

		for _, log := range logs {
			formatted, err := formatter.Format(log)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to format log: %v\n", err)
				continue
			}
			fmt.Println(formatted)
		}

		// Update lastTimestamp to avoid duplicate logs
		if !latest.IsZero() {
			if lastTimestamp.IsZero() || latest.After(lastTimestamp) {
				lastTimestamp = latest
			}
		} else {
			// If no logs returned, advance time slightly to avoid infinite loop
			if lastTimestamp.IsZero() {
				lastTimestamp = time.Now().Add(-1 * time.Minute)
			} else {
				lastTimestamp = time.Now().Add(-30 * time.Second)
			}
		}
		time.Sleep(currentInterval)
	}
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
			"limit": 100,
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
	defer resp.Body.Close()

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

// tryAlternativeRequest tries alternative request structures
func (c *Client) tryAlternativeRequest(ctx context.Context, from, to time.Time) ([]LogEntry, time.Time, error) {
	fmt.Fprintf(os.Stderr, "Trying alternative request structure...\n")

	// Try with query inside filter
	query := c.buildQueryV2()
	body := map[string]interface{}{
		"filter": map[string]interface{}{
			"from":  from.UTC().Format(time.RFC3339),
			"to":    to.UTC().Format(time.RFC3339),
			"query": query,
		},
		"page": map[string]interface{}{
			"limit": 100,
		},
		"sort": "timestamp",
	}
	jsonBody, _ := json.Marshal(body)

	fmt.Fprintf(os.Stderr, "Alternative request body: %s\n", string(jsonBody))

	req, err := c.createRequest(ctx, "POST", "/api/v2/logs/events/search")
	if err != nil {
		return nil, time.Time{}, err
	}
	req.Body = io.NopCloser(bytes.NewReader(jsonBody))
	req.ContentLength = int64(len(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to execute alternative HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, time.Time{}, fmt.Errorf("Alternative request also failed: %s - %s", resp.Status, string(body))
	}

	var v2resp v2LogsResponse
	if err := json.NewDecoder(resp.Body).Decode(&v2resp); err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to parse alternative response: %w", err)
	}

	var logs []LogEntry
	var latest time.Time
	for _, d := range v2resp.Data {
		ts, _ := time.Parse(time.RFC3339Nano, d.Attrs.Timestamp)

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

		service := d.Attrs.Service
		if service == "" {
			service = d.Attrs.Host
		}

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

	fmt.Fprintf(os.Stderr, "Alternative request succeeded!\n")
	return logs, latest, nil
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
	if c.config.GetLogLevel() != "" {
		conditions = append(conditions, fmt.Sprintf("status:%s", c.config.GetLogLevel()))
	}
	return strings.Join(conditions, " ")
}
