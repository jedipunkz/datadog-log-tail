# dlt - Datadog Logs Tail

A Go command-line tool for tailing Datadog Logs in real-time

## Features

- Real-time log retrieval using Datadog Logs API
- Authentication via environment variables
- Log filtering by tags
- Log level filtering
- Multiple output formats (JSON, plain text)
- Error handling and retry functionality

## Installation

### Using Pre-built Binary

```bash
# Build
make build

# Install to system PATH (optional)
make install
```

### Build from Source

```bash
# Install dependencies
make deps

# Build for development
make build-dev

# Build for release (optimized)
make build-release

# Build for specific platform
make build-linux    # Linux
make build-darwin   # macOS
make build-windows  # Windows

# Build for all platforms
make build-all
```

## Configuration

### Environment Variables

Required environment variables:

```bash
export DD_API_KEY="your-datadog-api-key"
export DD_APP_KEY="your-datadog-application-key"
export DD_SITE="datadoghq.com"  # Default: datadoghq.com
```

### Configuration File

Create a configuration file at `~/.dlt/config.yaml`:

```yaml
# Log filtering (optional)
tags: "service:web,env:production"
log_level: "info"

# Output settings (optional)
output_format: "text"  # json or text

# Connection settings (optional)
timeout: 30
retry_count: 3
```

## Usage

### Basic Usage

```bash
# Basic usage
dlt

# Filter by tags
dlt --tags "service:web,env:production"

# Filter by log level
dlt --tags "service:api" --level error

# Specify output format
dlt --format json
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--tags` | Tag filter (comma-separated) | - |
| `--level` | Log level (debug, info, warn, error) | - |
| `--format` | Output format (json, text) | text |
| `--timeout` | Connection timeout (seconds) | 30 |
| `--retry` | Retry count | 3 |
| `--config` | Configuration file path | ~/.dlt/config.yaml |

### Examples

```bash
# Get error logs for a specific service in JSON format
dlt --tags "service:web" --level error --format json

# Filter by multiple tags
dlt --tags "service:api,env:staging,version:v1.0"

# Adjust timeout and retry count
dlt --timeout 60 --retry 5

# Use custom configuration file
dlt --config /path/to/config.yaml
```

## Output Formats

### Text Format (Default)

```
[2024-01-15 10:30:45] [ERROR] [web-service] Database connection failed [service:web, env:prod]
[2024-01-15 10:30:46] [INFO] [api-service] Request processed successfully [service:api, env:prod]
```

### JSON Format

```json
{
  "id": "log-id-123",
  "timestamp": 1705311045,
  "message": "Database connection failed",
  "service": "web-service",
  "status": "error",
  "tags": ["service:web", "env:prod"],
  "attributes": {
    "host": "web-server-1",
    "port": 5432
  }
}
```

## Error Handling

### Authentication Errors

```bash
Error: API key not set (DD_API_KEY)
Error: Application key not set (DD_APP_KEY)
```

### Network Errors

- Connection timeout
- DNS resolution error
- HTTP status error

### API Errors

- Rate limiting
- Query syntax error
- Server error

## Development

### Project Structure

```
.
├── main.go                 # Entry point
├── Makefile               # Build automation
├── cmd/
│   └── tail.go            # Main command
├── internal/
│   ├── config/
│   │   └── config.go      # Configuration management
│   ├── datadog/
│   │   ├── client.go      # Datadog API client
│   │   └── logs.go        # Log retrieval logic
│   └── output/
│       └── formatter.go   # Output formatter
├── pkg/
│   └── utils/
│       └── retry.go       # Retry utility
└── go.mod
```

### Makefile Commands

The project includes a comprehensive Makefile for common development tasks:

```bash
# Build targets
make build              # Build the application (default)
make build-dev          # Build for development (with debug info)
make build-release      # Build for release (optimized)
make build-linux        # Build for Linux
make build-darwin       # Build for macOS
make build-windows      # Build for Windows
make build-all          # Build for all platforms

# Installation
make install            # Install to /usr/local/bin
make uninstall          # Uninstall from /usr/local/bin

# Testing
make test               # Run all tests
make test-coverage      # Run tests with coverage report

# Code quality
make fmt                # Format code
make lint               # Run linter (requires golangci-lint)
make deps               # Download and tidy dependencies

# Utilities
make clean              # Clean build artifacts
make run                # Build and run the application
make help               # Show all available commands
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Test specific packages
go test ./internal/config
go test ./internal/datadog
```

### Build

```bash
# Development build
make build-dev

# Release build
make build-release

# Cross-platform builds
make build-all
```

### Code Quality

```bash
# Format code
make fmt

# Run linter (install golangci-lint first)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
make lint
```

## License

MIT License

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Create a Pull Request 