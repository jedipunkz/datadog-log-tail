package utils

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		name         string
		retryCount   int
		minExpected  time.Duration
		maxExpected  time.Duration
		exactCheck   bool
		exactValue   time.Duration
	}{
		{
			name:        "First retry",
			retryCount:  1,
			minExpected: 900 * time.Millisecond, // 2^1 - 10% jitter = 1.8s
			maxExpected: 2200 * time.Millisecond, // 2^1 + 10% jitter = 2.2s
		},
		{
			name:        "Second retry",
			retryCount:  2,
			minExpected: 3600 * time.Millisecond, // 2^2 - 10% jitter = 3.6s
			maxExpected: 4400 * time.Millisecond, // 2^2 + 10% jitter = 4.4s
		},
		{
			name:        "Third retry",
			retryCount:  3,
			minExpected: 7200 * time.Millisecond, // 2^3 - 10% jitter = 7.2s
			maxExpected: 8800 * time.Millisecond, // 2^3 + 10% jitter = 8.8s
		},
		{
			name:        "Large retry count (should cap at 30s)",
			retryCount:  10,
			minExpected: 27 * time.Second, // 30s - 10% jitter = 27s
			maxExpected: 30 * time.Second, // Capped at 30s
		},
		{
			name:       "Zero retry count (minimum 1s)",
			retryCount: 0,
			exactCheck: true,
			exactValue: 1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateBackoff(tt.retryCount)

			if tt.exactCheck {
				if result != tt.exactValue {
					t.Errorf("CalculateBackoff(%d) = %v, want exactly %v", tt.retryCount, result, tt.exactValue)
				}
				return
			}

			if result < tt.minExpected || result > tt.maxExpected {
				t.Errorf("CalculateBackoff(%d) = %v, want between %v and %v", 
					tt.retryCount, result, tt.minExpected, tt.maxExpected)
			}

			// Ensure result is never more than 30 seconds
			if result > 30*time.Second {
				t.Errorf("CalculateBackoff(%d) = %v, should not exceed 30 seconds", tt.retryCount, result)
			}

			// Ensure result is never less than 1 second
			if result < 1*time.Second {
				t.Errorf("CalculateBackoff(%d) = %v, should not be less than 1 second", tt.retryCount, result)
			}
		})
	}
}

func TestCalculateBackoff_Consistency(t *testing.T) {
	// Test that the function is deterministic for the same input
	// (since we use rand, we need to test multiple calls)
	retryCount := 3
	results := make([]time.Duration, 100)

	for i := 0; i < 100; i++ {
		results[i] = CalculateBackoff(retryCount)
	}

	// All results should be within the expected range
	expectedBase := math.Pow(2, float64(retryCount))
	minExpected := time.Duration(expectedBase*0.9) * time.Second
	maxExpected := time.Duration(expectedBase*1.1) * time.Second

	for i, result := range results {
		if result < minExpected || result > maxExpected {
			t.Errorf("CalculateBackoff(%d) call %d = %v, want between %v and %v", 
				retryCount, i, result, minExpected, maxExpected)
		}
	}

	// Check that we get some variation (jitter is working)
	allSame := true
	firstResult := results[0]
	for _, result := range results[1:] {
		if result != firstResult {
			allSame = false
			break
		}
	}

	if allSame {
		t.Error("CalculateBackoff should produce varying results due to jitter")
	}
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "Timeout error",
			err:      errors.New("connection timeout"),
			expected: true,
		},
		{
			name:     "Connection refused",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "Network error",
			err:      errors.New("network unreachable"),
			expected: true,
		},
		{
			name:     "Temporary error",
			err:      errors.New("temporary failure"),
			expected: true,
		},
		{
			name:     "500 Internal Server Error",
			err:      errors.New("HTTP 500 Internal Server Error"),
			expected: true,
		},
		{
			name:     "502 Bad Gateway",
			err:      errors.New("502 Bad Gateway"),
			expected: true,
		},
		{
			name:     "503 Service Unavailable",
			err:      errors.New("503 Service Unavailable"),
			expected: true,
		},
		{
			name:     "504 Gateway Timeout",
			err:      errors.New("504 Gateway Timeout"),
			expected: true,
		},
		{
			name:     "Rate limit error",
			err:      errors.New("rate limit exceeded"),
			expected: true,
		},
		{
			name:     "429 Too Many Requests",
			err:      errors.New("429 Too Many Requests"),
			expected: true,
		},
		{
			name:     "401 Unauthorized (not retryable)",
			err:      errors.New("401 Unauthorized"),
			expected: false,
		},
		{
			name:     "403 Forbidden (not retryable)",
			err:      errors.New("403 Forbidden"),
			expected: false,
		},
		{
			name:     "404 Not Found (not retryable)",
			err:      errors.New("404 Not Found"),
			expected: false,
		},
		{
			name:     "400 Bad Request (not retryable)",
			err:      errors.New("400 Bad Request"),
			expected: false,
		},
		{
			name:     "Generic error (not retryable)",
			err:      errors.New("something went wrong"),
			expected: false,
		},
		{
			name:     "Parse error (not retryable)",
			err:      errors.New("failed to parse JSON"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldRetry(tt.err)
			if result != tt.expected {
				t.Errorf("ShouldRetry(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "Exact match",
			s:        "timeout",
			substr:   "timeout",
			expected: true,
		},
		{
			name:     "Substring at beginning",
			s:        "timeout error occurred",
			substr:   "timeout",
			expected: true,
		},
		{
			name:     "Substring at end",
			s:        "connection timeout",
			substr:   "timeout",
			expected: true,
		},
		{
			name:     "Substring in middle",
			s:        "network timeout error",
			substr:   "timeout",
			expected: true,
		},
		{
			name:     "Not found",
			s:        "connection refused",
			substr:   "timeout",
			expected: false,
		},
		{
			name:     "Empty substring",
			s:        "any string",
			substr:   "",
			expected: true,
		},
		{
			name:     "Empty string",
			s:        "",
			substr:   "timeout",
			expected: false,
		},
		{
			name:     "Both empty",
			s:        "",
			substr:   "",
			expected: true,
		},
		{
			name:     "Substring longer than string",
			s:        "hi",
			substr:   "hello",
			expected: false,
		},
		{
			name:     "Case sensitive",
			s:        "Timeout",
			substr:   "timeout",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

func TestContainsSubstring(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "Found at start",
			s:        "hello world",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "Found at end",
			s:        "hello world",
			substr:   "world",
			expected: true,
		},
		{
			name:     "Found in middle",
			s:        "hello world test",
			substr:   "world",
			expected: true,
		},
		{
			name:     "Not found",
			s:        "hello world",
			substr:   "xyz",
			expected: false,
		},
		{
			name:     "Empty substring always found",
			s:        "any string",
			substr:   "",
			expected: true,
		},
		{
			name:     "Single character found",
			s:        "hello",
			substr:   "e",
			expected: true,
		},
		{
			name:     "Single character not found",
			s:        "hello",
			substr:   "x",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsSubstring(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("containsSubstring(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkCalculateBackoff(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CalculateBackoff(3)
	}
}

func BenchmarkShouldRetry(b *testing.B) {
	err := errors.New("connection timeout")
	for i := 0; i < b.N; i++ {
		ShouldRetry(err)
	}
}

func BenchmarkContains(b *testing.B) {
	s := "this is a long string with timeout error in the middle"
	substr := "timeout"
	for i := 0; i < b.N; i++ {
		contains(s, substr)
	}
}