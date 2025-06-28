# CLAUDE.md

This file provides guidance for Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is `dlt` (Datadog Logs Tail), a Go command-line application for real-time tailing of Datadog Logs. It provides authentication through environment variables, log filtering by tags and levels, and multiple output formats.

## Key Development Commands

### Build
```bash
make build           # Standard build
make build-dev       # Development build with debug information
make build-release   # Optimized release build
make build-all       # Cross-platform build for Linux, macOS, Windows
```

### Testing
```bash
make test            # Run all tests
make test-coverage   # Run tests with coverage report (generates coverage.html)
go test ./internal/config     # Test specific package
go test ./internal/datadog    # Test specific package
```

### Code Quality
```bash
make fmt             # Format code with go fmt
make lint            # Run golangci-lint (requires golangci-lint installation)
make deps            # Download and tidy dependencies
```

### Execution
```bash
make run             # Build and run the application
./build/dlt          # Run the built binary directly
```

### Utilities
```bash
make clean           # Clean build artifacts and coverage files
make install         # Install to /usr/local/bin (requires sudo)
make uninstall       # Remove from /usr/local/bin (requires sudo)
```

## Architecture

The project follows Go's standard project layout:

- **main.go**: Entry point that delegates to the cmd package
- **cmd/tail.go**: Cobra-based CLI implementation with flag handling and configuration
- **internal/config/**: Configuration management with Viper (YAML config + environment variables)
- **internal/datadog/**: Datadog API client and log retrieval logic
- **internal/output/**: Output formatting (JSON and text formats)
- **pkg/utils/**: Reusable utilities (including retry logic)

### Key Components

1. **CLI Layer**: Cobra-based command-line interface with flags for tags, log levels, format, timeout, and retry count
2. **Configuration**: Viper-based configuration supporting both YAML files (~/.dlt/config.yaml) and environment variables (DD_API_KEY, DD_APP_KEY, DD_SITE)
3. **Datadog Integration**: Custom client for Datadog Logs API with authentication and real-time log streaming
4. **Output Formatting**: Pluggable formatters for text and JSON output

### Dependencies

- **github.com/spf13/cobra**: CLI framework
- **github.com/spf13/viper**: Configuration management
- Uses Go 1.24.3

## Environment Variables

Required for operation:
- `DD_API_KEY`: Datadog API key
- `DD_APP_KEY`: Datadog application key
- `DD_SITE`: Datadog site (default: datadoghq.com)

## Testing Strategy

When adding tests, follow the existing pattern of testing individual packages in isolation. Coverage reports are generated as HTML files for easy viewing.