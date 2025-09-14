# dlt - Datadog Logs Tail

A Go command-line tool for tailing Datadog Logs in real-time

## Features

- Real-time log retrieval using Datadog Logs API
- Batch log retrieval from specific timestamps
- Authentication via environment variables
- Log filtering by tags
- Log level filtering
- Multiple output formats (JSON, plain text)

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

## Usage

### Basic Usage

```bash
# Basic usage (real-time tailing)
dlt

# Filter by tags
dlt --query "service:web,env:production"

# Filter by log level
dlt --query "service:api" --level error

# Specify output format
dlt --format json

# Get logs from time range (batch mode)
dlt --timestamp "2024-01-15T10:00:00Z,2024-01-15T11:00:00Z"
```

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--query` | `-q` | Tag filter (comma-separated) | - |
| `--level` | `-l` | Log level (debug, info, warn, error) | - |
| `--format` | `-f` | Output format (json, text) | text |
| `--timestamp` | `-s` | Time range for log search in RFC3339 format (from,to) | - |
| `--timeout` | - | Connection timeout in seconds | 30 |
| `--retry-count` | - | Number of retries for failed requests | 3 |

**Note:** When using `--timestamp` with long time ranges, you may encounter Datadog API rate limits. The tool automatically handles rate limiting with exponential backoff and retries, but large datasets may take longer to retrieve.

### Examples

```bash
# Get error logs for a specific service in JSON format
dlt --query "service:web" --level error --format json

# Filter by multiple tags
dlt --query "service:api,env:staging,version:v1.0"

# Use custom timeout and retry settings
dlt --timeout 60 --retry-count 5

```

## License

MIT License

