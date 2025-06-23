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
# Clone the repository
git clone <repository-url>
cd datadog-log-tail

# Build
go build -o dlt

# Add executable to PATH (optional)
sudo cp dlt /usr/local/bin/
```

### Build from Source

```bash
# Install dependencies
go mod tidy

# Build
go build -o dlt
```

## Configuration

### Environment Variables

Required environment variables:

```bash
export DD_API_KEY="your-datadog-api-key"
export DD_APP_KEY="your-datadog-application-key"
```

Optional environment variables:

```bash
export DD_SITE="datadoghq.com"  # Default: datadoghq.com
```

### Configuration File

Create a configuration file at `~/.dlt/config.yaml`:

```yaml
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

### Testing

```bash
# Run all tests
go test ./...

# Test specific packages
go test ./internal/config
go test ./internal/datadog
```

### Build

```bash
# Development build
go build -o dlt

# Release build
go build -ldflags="-s -w" -o dlt
```

## License

MIT License

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Create a Pull Request 