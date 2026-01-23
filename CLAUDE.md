# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

cloudm-cli is a PostgreSQL migration tool that provides a Bubble Tea-powered UI for migrating data between PostgreSQL databases. It's a Go application that orchestrates PostgreSQL operations using `pg_dump` and `pg_restore` while providing a rich terminal user interface.

## Architecture

The application follows a "Go as the Orchestrator" pattern:
- **UI Layer**: Uses `github.com/charmbracelet/bubbletea` for the terminal UI
- **Database Operations**: Uses `github.com/jackc/pgx/v5` for direct database interactions
- **System Operations**: Uses `os/exec` to call `pg_dump` and `pg_restore` binaries
- **Styling**: Uses `github.com/charmbracelet/lipgloss` for UI styling

Key planned files based on implementation plan:
- `main.go`: Contains the Bubble Tea model and configuration
- `checks.go`: Database connectivity and tool availability checks
- `target-backup.go`: Backup operations for target database
- `source-dump.go`: Dump structure and data from source
- `restore-to-target.go`: Restore operations to target database
- `op-setup.go`: Ownership transfer and setup operations
- `validation.go`: Validation of migration success
- `logging.go`: Dual logging to terminal and files

## Development Commands

### Building
```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Clean build artifacts
make clean
```

### Testing
```bash
# Run tests
make test

# Run with verbose output
go test -v ./...
```

### Development
```bash
# Format code
make fmt

# Lint code
make lint

# Run with help
make run

# Install to /usr/local/bin
make install
```

### Release
```bash
# Create a release (interactive)
make release

# Build release binaries locally
make release-local
```

## Key Features

The tool performs PostgreSQL database migrations with the following steps:
1. Pre-flight checks (connection tests, tool availability)
2. Backup target database (optional)
3. Dump structure and data from source database
4. Clean and restore to target database
5. Configure ownership for application user
6. Validate migration success

## Configuration

The tool supports environment variables for database connections:
- Source database: SRC_HOST, SRC_PORT, SRC_DB, SRC_USER, SRC_PASSWORD
- Target database: DST_HOST, DST_PORT, DST_DB, DST_ADMIN_USER, DST_ADMIN_PASSWORD
- Application user: APP_USER
- Options: SKIP_BACKUP, PARALLEL_JOBS, DATA_PARALLEL_JOBS, DRY_RUN

## Planned Enhancements

Based on the implementation plan, the Go version will include:
- Real-time UI with progress indicators
- Concurrent validation
- Comprehensive error handling
- Structured logging with file output
- Configuration management with validation
- Proper error wrapping and propagation