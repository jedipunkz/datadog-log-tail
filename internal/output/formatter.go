package output

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// LogEntry interface for log entries
type LogEntry interface {
	GetID() string
	GetTimestamp() int64
	GetMessage() string
	GetService() string
	GetStatus() string
	GetTags() []string
	GetAttributes() map[string]interface{}
}

// Formatter interface for log output formatting
type Formatter interface {
	Format(log LogEntry) (string, error)
}

// JSONFormatter formats logs as JSON
type JSONFormatter struct{}

// TextFormatter formats logs as plain text
type TextFormatter struct{}

// NewFormatter creates a new formatter based on the specified format
func NewFormatter(format string) Formatter {
	switch strings.ToLower(format) {
	case "json":
		return &JSONFormatter{}
	case "text":
		return &TextFormatter{}
	default:
		return &TextFormatter{} // Default to text format
	}
}

// Format formats a log entry as JSON
func (f *JSONFormatter) Format(log LogEntry) (string, error) {
	jsonData, err := json.Marshal(log)
	if err != nil {
		return "", fmt.Errorf("failed to marshal log to JSON: %w", err)
	}
	return string(jsonData), nil
}

// Format formats a log entry as plain text
func (f *TextFormatter) Format(log LogEntry) (string, error) {
	timestamp := time.Unix(log.GetTimestamp(), 0).Format("2006-01-02 15:04:05")

	// Format tags
	tagsStr := ""
	if len(log.GetTags()) > 0 {
		tagsStr = " [" + strings.Join(log.GetTags(), ", ") + "]"
	}

	// Format: [timestamp] [level] [service] message [tags]
	formatted := fmt.Sprintf("[%s] [%s] [%s] %s%s",
		timestamp,
		strings.ToUpper(log.GetStatus()),
		log.GetService(),
		log.GetMessage(),
		tagsStr,
	)

	return formatted, nil
}
