package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	APIKey       string   `mapstructure:"api_key"`
	AppKey       string   `mapstructure:"app_key"`
	Site         string   `mapstructure:"site"`
	Tags         string   `mapstructure:"tags"`
	LogLevel     string   `mapstructure:"log_level"`
	LogLevels    []string `mapstructure:"log_levels"`
	OutputFormat string   `mapstructure:"output_format"`
	Timeout      int      `mapstructure:"timeout"`
	RetryCount   int      `mapstructure:"retry_count"`
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	// Set default values (except site, which should come from environment)
	viper.SetDefault("output_format", "text")
	viper.SetDefault("timeout", 30)
	viper.SetDefault("retry_count", 3)

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is not an error
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// SetConfigFile sets the configuration file path
func SetConfigFile(configFile string) {
	viper.SetConfigFile(configFile)
}

// AutomaticEnv enables automatic environment variable binding
func AutomaticEnv() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
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

// CreateDefaultConfig creates a default configuration file
func CreateDefaultConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".dlt")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")
	if _, err := os.Stat(configFile); err == nil {
		return fmt.Errorf("config file already exists: %s", configFile)
	}

	configContent := `# Datadog Logs Tail Configuration
# API credentials are loaded from environment variables:
#   DD_API_KEY - Datadog API key (required)
#   DD_APP_KEY - Datadog application key (required)
#   DD_SITE - Datadog site (optional, default: datadoghq.com)

# Log filtering (optional)
tags: "service:web,env:production"
log_level: "info"

# Output settings (optional)
output_format: "text"  # json or text

# Connection settings (optional)
timeout: 30
retry_count: 3
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Default configuration file created: %s\n", configFile)
	return nil
}
