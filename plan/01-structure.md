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


# Task Lists

## Phase 1: Setup and Dependencies - COMPLETED ✅
1. Initialize Go module for the project
   - [x] Create go.mod file with module name
   - [x] Set Go version requirement
   - [x] Verify module initialization works correctly

2. Add required dependencies (bubbletea, pgx, lipgloss, etc.)
   - [x] Add github.com/charmbracelet/bubbletea
   - [x] Add github.com/charmbracelet/bubbles
   - [x] Add github.com/charmbracelet/lipgloss
   - [x] Add github.com/jackc/pgx/v5
   - [x] Run go mod tidy to resolve dependencies

3. Create basic directory structure
   - [x] Create cmd/ directory
   - [x] Create internal/ directory
   - [x] Create pkg/ directory
   - [x] Create configs/ directory
   - [x] Create testdata/ directory

4. Set up configuration management with defaults
   - [x] Define Config struct with fields for DB connections
   - [x] Create default configuration values
   - [x] Implement config validation function
   - [x] Add config loading from file/environment

## Phase 2: Core Structure
5. Implement the main Bubble Tea model with state management
   - [ ] Define Model struct with state fields
   - [ ] Implement Init() function
   - [ ] Implement Update() function with state transitions
   - [ ] Implement View() function for UI rendering

6. Create the Config struct with validation
   - [ ] Define all necessary configuration fields
   - [ ] Add validation methods to Config
   - [ ] Implement config loading from multiple sources
   - [ ] Add error handling for invalid configurations

7. Implement basic UI elements (spinner, viewport, status display)
   - [ ] Add spinner bubble for loading states
   - [ ] Add viewport bubble for log display
   - [ ] Implement status display elements
   - [ ] Style UI elements with lipgloss

8. Set up the main event loop and state transitions
   - [ ] Define state constants (StateChecking, StateDumping, etc.)
   - [ ] Implement state transition logic
   - [ ] Handle user input for navigation
   - [ ] Test state transitions work properly

## Phase 3: Database Operations
9. Implement database connection checking in checks.go
   - [ ] Create CheckDatabaseConnections function
   - [ ] Use exec.LookPath to verify pg_dump/pg_restore exist
   - [ ] Test database connectivity with pgx
   - [ ] Return appropriate error messages for failures

10. Create utility functions for pgx connections
    - [ ] Implement getConnection function for source DB
    - [ ] Implement getConnection function for target DB
    - [ ] Add connection pooling configuration
    - [ ] Handle connection cleanup with defer statements

11. Implement database validation functions in validation.go
    - [ ] Create ValidateDatabaseSchema function
    - [ ] Implement row count comparison functions
    - [ ] Add concurrent validation with goroutines
    - [ ] Handle validation errors appropriately

12. Add error handling for all database operations
    - [ ] Create custom error types for database errors
    - [ ] Wrap errors with context using fmt.Errorf
    - [ ] Implement retry logic for transient failures
    - [ ] Add comprehensive error logging

## Phase 4: System Operations
13. Implement pg_dump execution in target-backup.go
    - [ ] Create BackupTargetDatabase function
    - [ ] Build proper pg_dump command with options
    - [ ] Stream command output to UI
    - [ ] Handle command execution errors

14. Implement pg_dump execution for structure and data in source-dump.go
    - [ ] Create DumpStructure function with -s flag
    - [ ] Create DumpData function with -a flag
    - [ ] Stream outputs to UI for real-time updates
    - [ ] Handle both functions with proper error handling

15. Implement pg_restore execution in restore-to-target.go
    - [ ] Create RestoreToTarget function
    - [ ] Build proper pg_restore command with options
    - [ ] Stream command output to UI
    - [ ] Handle restore-specific error conditions

16. Add process monitoring and output streaming
    - [ ] Capture stdout and stderr from subprocesses
    - [ ] Stream output to the Bubble Tea model
    - [ ] Monitor process status and report to UI
    - [ ] Handle process cancellation if needed

## Phase 5: Business Logic
17. Implement connection termination in restore-to-target.go
    - [ ] Create TerminateConnections function using pgx
    - [ ] Execute pg_terminate_backend SQL commands
    - [ ] Verify connections are terminated before proceeding
    - [ ] Handle any errors during termination

18. Create schema cleanup operations using pgx
    - [ ] Implement DROP SCHEMA public CASCADE function
    - [ ] Add safety checks before schema deletion
    - [ ] Handle cleanup errors appropriately
    - [ ] Verify schema is properly cleared

19. Implement ownership transfer in op-setup.go
    - [ ] Convert SQL ownership transfer block to Go
    - [ ] Execute as transaction using pgx
    - [ ] Handle ownership transfer errors
    - [ ] Verify ownership is properly transferred

20. Add concurrent validation with goroutines
    - [ ] Create concurrent validation function
    - [ ] Run validation on both databases simultaneously
    - [ ] Collect results from goroutines
    - [ ] Compare results and report discrepancies

## Phase 6: Logging and Error Handling
21. Implement dual logging (terminal and file) in logging.go
    - [ ] Create Logger struct with writers for terminal and file
    - [ ] Implement Info, Warn, Error level logging
    - [ ] Write logs simultaneously to terminal and file
    - [ ] Format logs consistently across outputs

22. Add structured error types and wrapping
    - [ ] Define custom error types for different operations
    - [ ] Use fmt.Errorf with %w verb for error wrapping
    - [ ] Create error helper functions
    - [ ] Ensure all errors provide sufficient context

23. Create error display in the UI
    - [ ] Add error display area in the Bubble Tea view
    - [ ] Style error messages appropriately
    - [ ] Show error details without overwhelming the user
    - [ ] Allow users to see full error details if needed

24. Implement log file rotation and naming convention
    - [ ] Create timestamped log file names (migration_YYYYMMDD_HHMMSS.log)
    - [ ] Implement log file rotation to prevent huge files
    - [ ] Store logs in appropriate directory
    - [ ] Add log retention policy

## Phase 7: Configuration and Testing
25. Implement environment-specific configuration loading
    - [ ] Create config loading from environment variables
    - [ ] Implement config loading from files (config.yaml, .env)
    - [ ] Add command-line flag overrides
    - [ ] Merge configurations with proper precedence

26. Add command-line flag parsing
    - [ ] Define command-line flags for key options
    - [ ] Parse flags in main function
    - [ ] Override config values with flag values
    - [ ] Display help text for available flags

27. Create unit tests for core functions
    - [ ] Write tests for database operations
    - [ ] Write tests for configuration loading
    - [ ] Write tests for logging functions
    - [ ] Achieve good test coverage (>80%)

28. Add integration tests for migration workflows
    - [ ] Create test database containers if needed
    - [ ] Test end-to-end migration scenarios
    - [ ] Verify data integrity after migrations
    - [ ] Test error handling in integration tests

## Phase 8: Polish and Documentation
29. Enhance UI with lipgloss styling
    - [ ] Create consistent color scheme
    - [ ] Style all UI elements professionally
    - [ ] Add visual feedback for different states
    - [ ] Ensure accessibility and readability

30. Add progress indicators and status messages
    - [ ] Implement progress bars for long operations
    - [ ] Add estimated time remaining for operations
    - [ ] Show current operation status clearly
    - [ ] Update progress in real-time

31. Create usage documentation and examples
    - [ ] Write README with installation instructions
    - [ ] Document all command-line options
    - [ ] Provide usage examples for common scenarios
    - [ ] Include troubleshooting section

32. Perform end-to-end testing and debugging
    - [ ] Run complete migration scenario
    - [ ] Test error recovery procedures
    - [ ] Verify data integrity after migration
    - [ ] Debug and fix any remaining issues