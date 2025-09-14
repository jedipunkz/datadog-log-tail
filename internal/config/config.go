package config

import (
	"fmt"
	"os"
	"strings"
)

// Config represents the application configuration
type Config struct {
	APIKey       string
	AppKey       string
	Site         string
	Tags         string
	LogLevel     string
	LogLevels    []string
	OutputFormat string
	Timestamp    string
	Timeout      int
	RetryCount   int
}

// New creates a new configuration with default values
func New() *Config {
	return &Config{
		OutputFormat: "text",
		Timeout:      30,
		RetryCount:   3,
		Site:         "datadoghq.com",
	}
}


// Validate validates the configuration
func (c *Config) Validate() error {
	// API key must be set via environment variable
	if c.APIKey == "" {
		apiKey := os.Getenv("DD_API_KEY")
		if apiKey == "" {
			return fmt.Errorf("API key not set (DD_API_KEY)")
		}
		c.APIKey = apiKey
	}

	// App key must be set via environment variable
	if c.AppKey == "" {
		appKey := os.Getenv("DD_APP_KEY")
		if appKey == "" {
			return fmt.Errorf("application key not set (DD_APP_KEY)")
		}
		c.AppKey = appKey
	}

	// Site must be set via environment variable
	site := os.Getenv("DD_SITE")
	if site != "" {
		c.Site = site
	} else {
		// Fallback to default if not set
		c.Site = "datadoghq.com"
	}

	if c.OutputFormat != "json" && c.OutputFormat != "text" {
		return fmt.Errorf("invalid output format: %s (json or text must be specified)", c.OutputFormat)
	}

	if c.LogLevel != "" {
		// Parse comma-separated log levels
		levels := strings.Split(c.LogLevel, ",")
		c.LogLevels = make([]string, 0, len(levels))
		validLevels := []string{"debug", "info", "warn", "error"}

		for _, level := range levels {
			level = strings.TrimSpace(level)
			if level == "" {
				continue
			}
			isValid := false
			for _, validLevel := range validLevels {
				if level == validLevel {
					isValid = true
					break
				}
			}
			if !isValid {
				return fmt.Errorf("invalid log level: %s (must be one of debug, info, warn, error)", level)
			}
			c.LogLevels = append(c.LogLevels, level)
		}
	}

	return nil
}

// GetAPIKey returns the API key
func (c *Config) GetAPIKey() string {
	return c.APIKey
}

// GetAppKey returns the application key
func (c *Config) GetAppKey() string {
	return c.AppKey
}

// GetSite returns the Datadog site
func (c *Config) GetSite() string {
	return c.Site
}

// GetTags returns the tag filter
func (c *Config) GetTags() string {
	return c.Tags
}

// GetLogLevel returns the log level filter
func (c *Config) GetLogLevel() string {
	return c.LogLevel
}

// GetLogLevels returns the parsed log levels as a slice
func (c *Config) GetLogLevels() []string {
	return c.LogLevels
}

// GetOutputFormat returns the output format
func (c *Config) GetOutputFormat() string {
	return c.OutputFormat
}

// GetTimeout returns the connection timeout
func (c *Config) GetTimeout() int {
	return c.Timeout
}

// GetRetryCount returns the retry count
func (c *Config) GetRetryCount() int {
	return c.RetryCount
}

// GetTimestamp returns the timestamp filter
func (c *Config) GetTimestamp() string {
	return c.Timestamp
}

