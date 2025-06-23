package utils

import (
	"math"
	"math/rand"
	"time"
)

// CalculateBackoff calculates exponential backoff wait time
func CalculateBackoff(retryCount int) time.Duration {
	// Exponential backoff: 2^retryCount seconds (max 30 seconds)
	backoffSeconds := math.Pow(2, float64(retryCount))
	if backoffSeconds > 30 {
		backoffSeconds = 30
	}

	// Appropriate jitter (±20% randomness)
	jitter := backoffSeconds * 0.2 * (rand.Float64() - 0.5) // -10% to +10%
	backoffSeconds = backoffSeconds + jitter

	// Minimum 1 second, maximum 30 seconds
	if backoffSeconds < 1 {
		backoffSeconds = 1
	}
	if backoffSeconds > 30 {
		backoffSeconds = 30
	}

	return time.Duration(backoffSeconds) * time.Second
}

// ShouldRetry determines whether to retry based on the error
func ShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Determine retryable errors from error message
	errMsg := err.Error()

	// Network errors
	if contains(errMsg, "timeout") ||
		contains(errMsg, "connection refused") ||
		contains(errMsg, "network") ||
		contains(errMsg, "temporary") {
		return true
	}

	// HTTP errors (5xx are retryable)
	if contains(errMsg, "500") ||
		contains(errMsg, "502") ||
		contains(errMsg, "503") ||
		contains(errMsg, "504") {
		return true
	}

	// Rate limiting
	if contains(errMsg, "rate limit") ||
		contains(errMsg, "429") {
		return true
	}

	return false
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr))))
}

// containsSubstring performs substring search
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
