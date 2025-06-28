package datadog

import (
	"testing"

	"github.com/jedipunkz/datadog-log-tail/internal/config"
)


func TestLogEntry_Interface(t *testing.T) {
	log := &LogEntry{
		ID:         "test-id",
		Timestamp:  1642694400, // 2022-01-20 12:00:00 UTC
		Message:    "Test message",
		Service:    "test-service",
		Status:     "info",
		Tags:       []string{"env:test", "service:api"},
		Attributes: map[string]interface{}{"host": "localhost"},
	}

	// Test that LogEntry implements the interface correctly
	if log.GetID() != "test-id" {
		t.Errorf("GetID() = %v, want test-id", log.GetID())
	}

	if log.GetTimestamp() != 1642694400 {
		t.Errorf("GetTimestamp() = %v, want 1642694400", log.GetTimestamp())
	}

	if log.GetMessage() != "Test message" {
		t.Errorf("GetMessage() = %v, want Test message", log.GetMessage())
	}

	if log.GetService() != "test-service" {
		t.Errorf("GetService() = %v, want test-service", log.GetService())
	}

	if log.GetStatus() != "info" {
		t.Errorf("GetStatus() = %v, want info", log.GetStatus())
	}

	tags := log.GetTags()
	if len(tags) != 2 || tags[0] != "env:test" || tags[1] != "service:api" {
		t.Errorf("GetTags() = %v, want [env:test service:api]", tags)
	}

	attrs := log.GetAttributes()
	if attrs["host"] != "localhost" {
		t.Errorf("GetAttributes()[host] = %v, want localhost", attrs["host"])
	}
}

func TestBuildQueryV2(t *testing.T) {
	tests := []struct {
		name     string
		tags     string
		logLevel string
		expected string
	}{
		{
			name:     "No filters",
			tags:     "",
			logLevel: "",
			expected: "",
		},
		{
			name:     "Tags only",
			tags:     "service:web,env:prod",
			logLevel: "",
			expected: "service:web env:prod",
		},
		{
			name:     "Log level only",
			tags:     "",
			logLevel: "error",
			expected: "status:error",
		},
		{
			name:     "Tags and log level",
			tags:     "service:api,version:v1.0",
			logLevel: "info",
			expected: "service:api version:v1.0 status:info",
		},
		{
			name:     "Tags with spaces",
			tags:     " service:web , env:staging ",
			logLevel: "warn",
			expected: "service:web env:staging status:warn",
		},
		{
			name:     "Single tag",
			tags:     "service:database",
			logLevel: "",
			expected: "service:database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Tags:     tt.tags,
				LogLevel: tt.logLevel,
			}

			client := &Client{config: cfg}
			result := client.buildQueryV2()

			if result != tt.expected {
				t.Errorf("buildQueryV2() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestV2LogAttributes_MessageExtraction(t *testing.T) {
	tests := []struct {
		name     string
		attrs    v2LogAttributes
		expected string
	}{
		{
			name: "Message field",
			attrs: v2LogAttributes{
				Message: "This is a message",
			},
			expected: "This is a message",
		},
		{
			name: "Content field when message is empty",
			attrs: v2LogAttributes{
				Content: "This is content",
			},
			expected: "This is content",
		},
		{
			name: "Text field when message and content are empty",
			attrs: v2LogAttributes{
				Text: "This is text",
			},
			expected: "This is text",
		},
		{
			name: "Log field when others are empty",
			attrs: v2LogAttributes{
				Log: "This is log",
			},
			expected: "This is log",
		},
		{
			name:     "No message content",
			attrs:    v2LogAttributes{},
			expected: "No message content",
		},
		{
			name: "Message takes precedence",
			attrs: v2LogAttributes{
				Message: "Primary message",
				Content: "Secondary content",
				Text:    "Tertiary text",
				Log:     "Quaternary log",
			},
			expected: "Primary message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := tt.attrs.Message
			if message == "" {
				message = tt.attrs.Content
			}
			if message == "" {
				message = tt.attrs.Text
			}
			if message == "" {
				message = tt.attrs.Log
			}
			if message == "" {
				message = "No message content"
			}

			if message != tt.expected {
				t.Errorf("Message extraction = %v, want %v", message, tt.expected)
			}
		})
	}
}

func TestV2LogAttributes_ServiceExtraction(t *testing.T) {
	tests := []struct {
		name     string
		attrs    v2LogAttributes
		expected string
	}{
		{
			name: "Service field",
			attrs: v2LogAttributes{
				Service: "api-service",
			},
			expected: "api-service",
		},
		{
			name: "Host field when service is empty",
			attrs: v2LogAttributes{
				Host: "web-server-01",
			},
			expected: "web-server-01",
		},
		{
			name:     "No service or host",
			attrs:    v2LogAttributes{},
			expected: "",
		},
		{
			name: "Service takes precedence",
			attrs: v2LogAttributes{
				Service: "primary-service",
				Host:    "secondary-host",
			},
			expected: "primary-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := tt.attrs.Service
			if service == "" {
				service = tt.attrs.Host
			}

			if service != tt.expected {
				t.Errorf("Service extraction = %v, want %v", service, tt.expected)
			}
		})
	}
}

func TestV2LogAttributes_StatusExtraction(t *testing.T) {
	tests := []struct {
		name     string
		attrs    v2LogAttributes
		expected string
	}{
		{
			name: "Status field",
			attrs: v2LogAttributes{
				Status: "error",
			},
			expected: "error",
		},
		{
			name: "LogLevel field when status is empty",
			attrs: v2LogAttributes{
				LogLevel: "info",
			},
			expected: "info",
		},
		{
			name:     "No status or log level",
			attrs:    v2LogAttributes{},
			expected: "",
		},
		{
			name: "Status takes precedence",
			attrs: v2LogAttributes{
				Status:   "warn",
				LogLevel: "debug",
			},
			expected: "warn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := tt.attrs.Status
			if status == "" {
				status = tt.attrs.LogLevel
			}

			if status != tt.expected {
				t.Errorf("Status extraction = %v, want %v", status, tt.expected)
			}
		})
	}
}

