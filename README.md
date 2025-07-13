# dlt - Datadog Logs Tail

A Go command-line tool for tailing Datadog Logs in real-time

## Features

- Real-time log retrieval using Datadog Logs API
- Authentication via environment variables
- Log filtering by tags
- Log level filtering
- Multiple output formats (JSON, plain text)
- Interactive TUI (Terminal User Interface) mode

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
dlt --query "service:web,env:production"

# Filter by log level
dlt --query "service:api" --level error

# Specify output format
dlt --format json

# Enable TUI mode
dlt --tui
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--query` | Tag filter (comma-separated) | - |
| `--level` | Log level (debug, info, warn, error) | - |
| `--format` | Output format (json, text) | text |
| `--tui` | Enable TUI mode for interactive log viewing | false |
| `--config` | Configuration file path | ~/.dlt/config.yaml |

### Examples

```bash
# Get error logs for a specific service in JSON format
dlt --query "service:web" --level error --format json

# Filter by multiple tags
dlt --query "service:api,env:staging,version:v1.0"

# Use custom configuration file
dlt --config /path/to/config.yaml

# Start TUI mode with filters
dlt --tui --query "service:web" --level error
```

## TUI Mode

TUI (Terminal User Interface) mode provides an interactive interface for viewing logs with enhanced navigation and filtering capabilities.

```bash
# Enable TUI mode
dlt --tui

# TUI mode with filters
dlt --tui --query "service:web" --level error
```

TUI mode features:
- Interactive log viewing with scrolling
- Real-time log updates
- Enhanced visual formatting
- Keyboard navigation

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

## License

MIT License

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Create a Pull Request
