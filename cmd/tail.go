package cmd

import (
	"fmt"
	"os"

	"github.com/jedipunkz/datadog-log-tail/internal/config"
	"github.com/jedipunkz/datadog-log-tail/internal/datadog"
	"github.com/jedipunkz/datadog-log-tail/internal/tui"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	tags    string
	level   string
	format  string
	timeout int
	retry   int
	tuiMode bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dlt",
	Short: "Datadog Logs Tail - Real-time log tailing tool",
	Long: `dlt is a command-line tool for tailing Datadog Logs in real-time.

Authentication is configured via environment variables, and log filtering is available via tags.

Examples:
  dlt                                    # Basic usage
  dlt --tags "service:web,env:prod"     # Filter by tags
  dlt --level error --format json       # Filter by log level and output format
  dlt --level error,warn --tags "env:prod" # Filter by multiple log levels and tags`,
	RunE: runTail,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Configuration file (default: ~/.dlt/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&tags, "tags", "", "Tag filter (comma-separated)")
	rootCmd.PersistentFlags().StringVar(&level, "level", "", "Log level (debug, info, warn, error) - supports comma-separated values")
	rootCmd.PersistentFlags().StringVar(&format, "format", "text", "Output format (json, text)")
	rootCmd.PersistentFlags().IntVar(&timeout, "timeout", 30, "Connection timeout (seconds)")
	rootCmd.PersistentFlags().IntVar(&retry, "retry", 3, "Retry count")
	rootCmd.PersistentFlags().BoolVar(&tuiMode, "tui", false, "Enable TUI mode for interactive log viewing")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use specified config file
		config.SetConfigFile(cfgFile)
	} else {
		// Default config file path
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			return
		}
		config.SetConfigFile(home + "/.dlt/config.yaml")
	}

	config.AutomaticEnv() // Read environment variables
}

func runTail(cmd *cobra.Command, args []string) error {
	// Load and validate configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Apply flag values to configuration
	if tags != "" {
		cfg.Tags = tags
	}
	if level != "" {
		cfg.LogLevel = level
	}
	if format != "" {
		cfg.OutputFormat = format
	}
	if timeout > 0 {
		cfg.Timeout = timeout
	}
	if retry > 0 {
		cfg.RetryCount = retry
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Check if TUI mode is enabled
	if tuiMode {
		// Create and run TUI
		tuiApp, err := tui.New(cfg)
		if err != nil {
			return fmt.Errorf("failed to create TUI: %w", err)
		}

		if err := tuiApp.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		return nil
	}

	// Create Datadog client
	client, err := datadog.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Start tailing logs
	if err := client.TailLogs(); err != nil {
		return fmt.Errorf("failed to tail logs: %w", err)
	}

	return nil
}
