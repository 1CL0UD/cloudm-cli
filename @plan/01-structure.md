# CloudM-CLI: PostgreSQL Migration Tool - Implementation Plan

## The Strategy: "Go as the Orchestrator"

We will **rewrite the logic** in Go, but **reuse the binary tools**.
*   **Bash** previously decided *when* to run `pg_dump`.
*   **Go** will now decide *when* to run `pg_dump`.
*   **Bubble Tea** will visualize *what* Go is doing.

## The Stack / Libraries

Since we are keeping it simple and flat, we will use a lean set of standard and standard-industry libraries.

### 1. UI & Interaction (The "Bubble Tea" part)
*   **`github.com/charmbracelet/bubbletea`**: The main event loop. It will handle the state machine (e.g., `StateChecking`, `StateDumping`, `StateRestoring`).
*   **`github.com/charmbracelet/bubbles`**:
    *   **`spinner`**: To show activity during long dumps/restores.
    *   **`viewport`**: To show the scrolling logs (replacing `tail -f`).
*   **`github.com/charmbracelet/lipgloss`**: For basic styling (colors, bold text) so it looks professional.

### 2. Database Operations (The "SQL" part)
*   **`github.com/jackc/pgx/v5`**:
    *   **Why?** The Bash script uses `psql` for things like `SELECT count(*)`, `terminate_backend`, and setting ownership.
    *   Parsing text output from `psql` in Go is brittle and painful.
    *   Using `pgx` lets us run these queries natively in Go. It is faster, safer, and cleaner than `exec.Command("psql", ...)`.

### 3. System Operations (The "Heavy Lifting" part)
*   **`os/exec` (Standard Lib)**:
    *   We will use this to call `pg_dump` and `pg_restore`. There is no need to rewrite these protocols in Go.
    *   We will capture their `Stdout` and `Stderr` to pipe into our Bubble Tea UI logs.

## Enhanced Features

### Error Handling Strategy
*   Comprehensive error wrapping using `fmt.Errorf()` with `%w` verb for proper error propagation
*   Structured error types for different operation failures (database connection, dump/restore, file system)
*   Graceful degradation with clear error messages to the UI
*   Detailed error logging to both terminal and persistent log files

### Configuration Management
*   Support for different environments (development, staging, production) through configuration files
*   Environment variables as fallback for sensitive data (passwords, connection strings)
*   Command-line flag overrides for ad-hoc adjustments
*   Validation of configuration values before starting operations

### Logging Strategy
*   Real-time logging displayed in the terminal UI during tool execution
*   Simultaneous logging to persistent files with timestamped names (`migration_YYYYMMDD_HHMMSS.log`)
*   Different log levels (INFO, WARN, ERROR) for appropriate filtering
*   Structured logging format for easier parsing and analysis

### Testing Approach
*   Unit tests for individual functions (database operations, utility functions)
*   Integration tests for end-to-end migration scenarios
*   Mock implementations for database connections during testing
*   Test coverage for error handling paths

## The Proposed File Logic

Here is how we map your Bash logic to the Go files:

*   **`main.go`**:
    *   Contains the `Config` struct (hardcoded values for now).
    *   Contains the Bubble Tea `Model` (which holds the state: `CurrentStep`, `Logs`, `Error`).
    *   The `Update()` function acts as the "Controller," triggering the next step when one finishes.

*   **`checks.go`**:
    *   Uses `exec.LookPath` to ensure `pg_dump`/`pg_restore` exist.
    *   Uses `pgx.Connect` to test DB credentials (instead of running `psql -c "SELECT 1"`).

*   **`target-backup.go`**:
    *   Runs `exec.Command("pg_dump", ...)` pointed at the Target.
    *   Streams output to the UI.

*   **`source-dump.go`**:
    *   Function 1: `DumpStructure()` -> `exec.Command("pg_dump", "-s", ...)`
    *   Function 2: `DumpData()` -> `exec.Command("pg_dump", "-a", ...)`

*   **`restore-to-target.go`**:
    *   **Hybrid approach:**
    *   Uses `pgx` to execute the "Terminate Connections" SQL (safer than shelling out).
    *   Uses `pgx` to `DROP SCHEMA public CASCADE`.
    *   Uses `exec.Command("pg_restore")` for the actual file restoration.

*   **`op-setup.go`**:
    *   **Pure `pgx`**. We will take that massive SQL block (ownership transfer) and run it as a Go SQL transaction. This is much less error-prone than passing a heredoc to `psql`.

*   **`validation.go`**:
    *   **Pure `pgx`**. We run `SELECT count(*)` on both DBs concurrently using Go routines and compare the integers directly.

*   **`logging.go`**:
    *   A helper to append strings to a slice in the Bubble Tea model and simultaneously append to a `migration_timestamp.log` file.