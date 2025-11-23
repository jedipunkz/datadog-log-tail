# Datadog Logs Tail - dlt

A Go command-line tool for tailing Datadog Logs in real-time

## Installation

```bash
# Build
make build

# Install to system PATH (optional)
make install
```


## Environment Variables

Required environment variables:

```bash
export DD_API_KEY="your-datadog-api-key"
export DD_APP_KEY="your-datadog-application-key"
export DD_SITE="datadoghq.com"  # Default: datadoghq.com
```

## Usage


```bash
# Basic usage (real-time tailing)
dlt

# Filter by tags
dlt -q "service:web,env:production"

# Filter by log level
dlt -q "service:api" -l error

# Specify output format
dlt -f json

# Get logs from time range (batch mode)
dlt -s "2025-01-15T10:00:00Z,2025-01-15T11:00:00Z"
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


**Note:** When using `--timestamp` with long time ranges, you may encounter Datadog API rate limits. The tool automatically handles rate limiting with intelligent strategies:
- Monitors Datadog's rate limit headers (X-RateLimit-*) in real-time
- Implements preemptive backoff when quota is low (< 20%)
- Uses exponential backoff with jitter when rate limited
- Adjusts request frequency based on log activity
- Conservative default intervals to minimize rate limit hits

## License

MIT License

## Author

[jedipunkz](https://github.com/jedipunkz)
