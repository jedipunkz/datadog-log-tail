package cmd

import (
	"fmt"

	"github.com/jedipunkz/datadog-log-tail/internal/config"
	"github.com/jedipunkz/datadog-log-tail/internal/datadog"

	"github.com/spf13/cobra"
)

var (
	query      string
	level      string
	format     string
	timestamp  string
	timeout    int
	retryCount int
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dlt",
	Short: "Datadog Logs Tail - Real-time log tailing tool",
	Long: `dlt is a command-line tool for tailing Datadog Logs in real-time.

Authentication is configured via environment variables, and log filtering is available via tags.

Examples:
  dlt                                    # Basic usage (real-time tailing)
  dlt --query "service:web,env:prod"     # Filter by tags
  dlt --level error --format json       # Filter by log level and output format
  dlt --level error,warn --query "env:prod" # Filter by multiple log levels and tags
  dlt --timestamp "2024-01-15T10:00:00Z,2024-01-15T11:00:00Z" # Get logs from time range (batch mode)`,
	RunE: runTail,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&query, "query", "q", "", "Tag filter (comma-separated)")
	rootCmd.PersistentFlags().StringVarP(&level, "level", "l", "", "Log level (debug, info, warn, error) - supports comma-separated values")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "text", "Output format (json, text)")
	rootCmd.PersistentFlags().StringVarP(&timestamp, "timestamp", "s", "", "Time range for log search in RFC3339 format (from,to): 2024-01-15T10:00:00Z,2024-01-15T11:00:00Z")
	rootCmd.PersistentFlags().IntVar(&timeout, "timeout", 30, "Connection timeout in seconds")
	rootCmd.PersistentFlags().IntVar(&retryCount, "retry-count", 3, "Number of retries for failed requests")
}


func runTail(cmd *cobra.Command, args []string) error {
	// Create configuration from command line flags
	cfg := config.New()
	cfg.Tags = query
	cfg.LogLevel = level
	cfg.OutputFormat = format
	cfg.Timestamp = timestamp
	cfg.Timeout = timeout
	cfg.RetryCount = retryCount

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Create Datadog client
	client, err := datadog.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Datadog client: %w", err)
	}

	// Start tailing logs or batch retrieval based on timestamp
	if cfg.Timestamp != "" {
		// Batch mode: retrieve logs from specific timestamp
		if err := client.GetLogsFromTimestamp(); err != nil {
			return fmt.Errorf("failed to get logs from timestamp: %w", err)
		}
	} else {
		// Tail mode: real-time log streaming
		if err := client.TailLogs(); err != nil {
			return fmt.Errorf("failed to tail logs: %w", err)
		}
	}

	return nil
}
