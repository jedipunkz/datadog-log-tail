package output

import (
	"encoding/json"
	"strings"
	"testing"
)

// Mock LogEntry for testing
type mockLogEntry struct {
	id         string
	timestamp  int64
	message    string
	service    string
	status     string
	tags       []string
	attributes map[string]interface{}
}

func (m *mockLogEntry) GetID() string                         { return m.id }
func (m *mockLogEntry) GetTimestamp() int64                   { return m.timestamp }
func (m *mockLogEntry) GetMessage() string                    { return m.message }
func (m *mockLogEntry) GetService() string                    { return m.service }
func (m *mockLogEntry) GetStatus() string                     { return m.status }
func (m *mockLogEntry) GetTags() []string                     { return m.tags }
func (m *mockLogEntry) GetAttributes() map[string]interface{} { return m.attributes }

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{"JSON formatter", "json", "*output.JSONFormatter"},
		{"Text formatter", "text", "*output.TextFormatter"},
		{"Mixed case JSON", "JSON", "*output.JSONFormatter"},
		{"Mixed case text", "TEXT", "*output.TextFormatter"},
		{"Invalid format defaults to text", "invalid", "*output.TextFormatter"},
		{"Empty format defaults to text", "", "*output.TextFormatter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFormatter(tt.format)
			formatterType := getFormatterType(formatter)
			if formatterType != tt.expected {
				t.Errorf("NewFormatter(%v) = %v, want %v", tt.format, formatterType, tt.expected)
			}
		})
	}
}

func getFormatterType(formatter Formatter) string {
	switch formatter.(type) {
	case *JSONFormatter:
		return "*output.JSONFormatter"
	case *TextFormatter:
		return "*output.TextFormatter"
	default:
		return "unknown"
	}
}

func TestJSONFormatter_Format(t *testing.T) {
	formatter := &JSONFormatter{}

	tests := []struct {
		name    string
		log     LogEntry
		wantErr bool
	}{
		{
			name: "Valid log entry",
			log: &mockLogEntry{
				id:        "test-id-123",
				timestamp: 1642694400, // 2022-01-20 12:00:00 UTC
				message:   "Test message",
				service:   "api-service",
				status:    "info",
				tags:      []string{"env:prod", "service:api"},
				attributes: map[string]interface{}{
					"host": "web-server-01",
					"port": 8080,
				},
			},
			wantErr: false,
		},
		{
			name: "Empty log entry",
			log: &mockLogEntry{
				id:         "",
				timestamp:  0,
				message:    "",
				service:    "",
				status:     "",
				tags:       nil,
				attributes: nil,
			},
			wantErr: false,
		},
		{
			name: "Log with special characters",
			log: &mockLogEntry{
				id:      "special-123",
				message: "Error: \"Connection failed\" with special chars: <>",
				service: "test-service",
				status:  "error",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatter.Format(tt.log)

			if tt.wantErr {
				if err == nil {
					t.Error("JSONFormatter.Format() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("JSONFormatter.Format() unexpected error = %v", err)
				return
			}

			// Verify the result is valid JSON
			var jsonData map[string]interface{}
			if err := json.Unmarshal([]byte(result), &jsonData); err != nil {
				t.Errorf("JSONFormatter.Format() result is not valid JSON: %v", err)
				return
			}

			// Verify that the JSON is not empty and has basic structure
			if len(result) == 0 {
				t.Error("JSON result should not be empty")
			}
			
			// Basic structure check - should be a JSON object
			if !strings.HasPrefix(result, "{") || !strings.HasSuffix(result, "}") {
				t.Errorf("JSON result should be a valid JSON object, got: %s", result)
			}
		})
	}
}

func TestTextFormatter_Format(t *testing.T) {
	formatter := &TextFormatter{}

	tests := []struct {
		name     string
		log      LogEntry
		contains []string
		wantErr  bool
	}{
		{
			name: "Complete log entry",
			log: &mockLogEntry{
				id:        "test-id-123",
				timestamp: 1642694400, // 2022-01-20 12:00:00 UTC
				message:   "Database connection established",
				service:   "api-service",
				status:    "info",
				tags:      []string{"env:prod", "service:api"},
			},
			contains: []string{
				"2022-01-20 12:00:00",
				"INFO",
				"api-service",
				"Database connection established",
				"env:prod",
				"service:api",
			},
			wantErr: false,
		},
		{
			name: "Log without tags",
			log: &mockLogEntry{
				id:        "test-id-456",
				timestamp: 1642698000, // 2022-01-20 13:00:00 UTC
				message:   "Operation completed",
				service:   "worker-service",
				status:    "success",
				tags:      nil,
			},
			contains: []string{
				"2022-01-20 13:00:00",
				"SUCCESS",
				"worker-service",
				"Operation completed",
			},
			wantErr: false,
		},
		{
			name: "Error log",
			log: &mockLogEntry{
				id:        "error-789",
				timestamp: 1642701600, // 2022-01-20 14:00:00 UTC
				message:   "Connection timeout",
				service:   "database",
				status:    "error",
				tags:      []string{"urgent"},
			},
			contains: []string{
				"2022-01-20 14:00:00",
				"ERROR",
				"database",
				"Connection timeout",
				"urgent",
			},
			wantErr: false,
		},
		{
			name: "Empty fields",
			log: &mockLogEntry{
				timestamp: 1642694400,
				message:   "",
				service:   "",
				status:    "",
			},
			contains: []string{
				"2022-01-20 12:00:00",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatter.Format(tt.log)

			if tt.wantErr {
				if err == nil {
					t.Error("TextFormatter.Format() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("TextFormatter.Format() unexpected error = %v", err)
				return
			}

			// Check that all expected strings are present (except timezone-dependent timestamps)
			for _, expected := range tt.contains {
				// Skip exact timestamp checks as they depend on local timezone
				if strings.Contains(expected, "2022-01-") {
					continue
				}
				if !strings.Contains(result, expected) {
					t.Errorf("TextFormatter.Format() result doesn't contain %q\nResult: %s", expected, result)
				}
			}

			// Verify the basic format structure
			if !strings.Contains(result, "[") || !strings.Contains(result, "]") {
				t.Errorf("TextFormatter.Format() result doesn't follow expected format: %s", result)
			}
		})
	}
}

func TestTextFormatter_TimestampFormat(t *testing.T) {
	formatter := &TextFormatter{}

	// Test that timestamps are properly formatted (regardless of timezone)
	log := &mockLogEntry{
		timestamp: 1642694400, // Jan 20, 2022 12:00:00 UTC
		message:   "test",
		service:   "test",
		status:    "info",
	}

	result, err := formatter.Format(log)
	if err != nil {
		t.Errorf("TextFormatter.Format() error = %v", err)
		return
	}

	// Just check that the result contains a properly formatted timestamp pattern
	if !strings.Contains(result, "2022-01-") {
		t.Errorf("TextFormatter.Format() should contain a 2022 date, got: %s", result)
	}
	
	// Check for timestamp bracket pattern
	if !strings.Contains(result, "[2022-") {
		t.Errorf("TextFormatter.Format() should contain timestamp in brackets, got: %s", result)
	}
}

func TestTextFormatter_TagsFormat(t *testing.T) {
	formatter := &TextFormatter{}

	tests := []struct {
		name     string
		tags     []string
		expected string
	}{
		{"No tags", nil, ""},
		{"Single tag", []string{"env:prod"}, "[env:prod]"},
		{"Multiple tags", []string{"env:prod", "service:api", "version:v1.0"}, "[env:prod, service:api, version:v1.0]"},
		{"Empty tags slice", []string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := &mockLogEntry{
				timestamp: 1642694400,
				message:   "test message",
				service:   "test-service",
				status:    "info",
				tags:      tt.tags,
			}

			result, err := formatter.Format(log)
			if err != nil {
				t.Errorf("TextFormatter.Format() error = %v", err)
				return
			}

			if tt.expected == "" {
				// Should not contain any brackets if no tags
				if strings.Contains(result, "[]") {
					t.Errorf("TextFormatter.Format() should not contain empty brackets: %s", result)
				}
			} else {
				if !strings.Contains(result, tt.expected) {
					t.Errorf("TextFormatter.Format() should contain %q in result %s", tt.expected, result)
				}
			}
		})
	}
}