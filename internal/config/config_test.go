package config

import (
	"os"
	"strings"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name       string
		config     *Config
		envVars    map[string]string
		wantErr    bool
		errorContains string
	}{
		{
			name: "Valid config with environment variables",
			config: &Config{
				OutputFormat: "json",
				Timeout:      30,
				RetryCount:   3,
			},
			envVars: map[string]string{
				"DD_API_KEY": "test-api-key",
				"DD_APP_KEY": "test-app-key",
				"DD_SITE":    "datadoghq.com",
			},
			wantErr: false,
		},
		{
			name: "Missing API key",
			config: &Config{
				OutputFormat: "json",
			},
			envVars: map[string]string{
				"DD_APP_KEY": "test-app-key",
			},
			wantErr:    true,
			errorContains: "API key not set",
		},
		{
			name: "Missing app key",
			config: &Config{
				OutputFormat: "json",
			},
			envVars: map[string]string{
				"DD_API_KEY": "test-api-key",
			},
			wantErr:    true,
			errorContains: "application key not set",
		},
		{
			name: "Invalid output format",
			config: &Config{
				OutputFormat: "invalid",
			},
			envVars: map[string]string{
				"DD_API_KEY": "test-api-key",
				"DD_APP_KEY": "test-app-key",
			},
			wantErr:    true,
			errorContains: "invalid output format",
		},
		{
			name: "Invalid log level",
			config: &Config{
				OutputFormat: "json",
				LogLevel:     "invalid",
			},
			envVars: map[string]string{
				"DD_API_KEY": "test-api-key",
				"DD_APP_KEY": "test-app-key",
			},
			wantErr:    true,
			errorContains: "invalid log level",
		},
		{
			name: "Valid log levels",
			config: &Config{
				OutputFormat: "text",
				LogLevel:     "debug",
			},
			envVars: map[string]string{
				"DD_API_KEY": "test-api-key",
				"DD_APP_KEY": "test-app-key",
			},
			wantErr: false,
		},
		{
			name: "Default site when not set",
			config: &Config{
				OutputFormat: "text",
			},
			envVars: map[string]string{
				"DD_API_KEY": "test-api-key",
				"DD_APP_KEY": "test-app-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables
			os.Unsetenv("DD_API_KEY")
			os.Unsetenv("DD_APP_KEY")
			os.Unsetenv("DD_SITE")

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Config.Validate() expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Config.Validate() error = %v, want error containing %v", err, tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("Config.Validate() unexpected error = %v", err)
				}
			}

			// Check default site setting
			if !tt.wantErr && tt.envVars["DD_SITE"] == "" {
				if tt.config.GetSite() != "datadoghq.com" {
					t.Errorf("Config.GetSite() = %v, want datadoghq.com", tt.config.GetSite())
				}
			}
		})
	}

	// Clean up
	os.Unsetenv("DD_API_KEY")
	os.Unsetenv("DD_APP_KEY")
	os.Unsetenv("DD_SITE")
}

func TestConfig_Getters(t *testing.T) {
	config := &Config{
		APIKey:       "test-api-key",
		AppKey:       "test-app-key",
		Site:         "us3.datadoghq.com",
		Tags:         "service:web,env:prod",
		LogLevel:     "info",
		OutputFormat: "json",
		Timeout:      60,
		RetryCount:   5,
	}

	tests := []struct {
		name     string
		getter   func() interface{}
		expected interface{}
	}{
		{"GetAPIKey", func() interface{} { return config.GetAPIKey() }, "test-api-key"},
		{"GetAppKey", func() interface{} { return config.GetAppKey() }, "test-app-key"},
		{"GetSite", func() interface{} { return config.GetSite() }, "us3.datadoghq.com"},
		{"GetTags", func() interface{} { return config.GetTags() }, "service:web,env:prod"},
		{"GetLogLevel", func() interface{} { return config.GetLogLevel() }, "info"},
		{"GetOutputFormat", func() interface{} { return config.GetOutputFormat() }, "json"},
		{"GetTimeout", func() interface{} { return config.GetTimeout() }, 60},
		{"GetRetryCount", func() interface{} { return config.GetRetryCount() }, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.getter()
			if result != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, result, tt.expected)
			}
		})
	}
}