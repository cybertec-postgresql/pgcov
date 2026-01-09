# AI Agent Instructions for pgcov

This document provides comprehensive guidance for AI agents working on the pgcov project. Read this carefully before making any contributions.

## Table of Contents

- [Project Overview](#project-overview)
- [Core Principles & Constitution](#core-principles--constitution)
- [Architecture & Design](#architecture--design)
- [Development Environment](#development-environment)
- [Code Standards](#code-standards)
- [Testing Requirements](#testing-requirements)
- [Common Tasks](#common-tasks)
- [CI/CD Integration](#cicd-integration)
- [Troubleshooting](#troubleshooting)

---

## Project Overview

**pgcov** is a PostgreSQL test runner and coverage tool written in Go. It discovers `*_test.sql` files, instruments SQL/PL/pgSQL source code for coverage tracking, executes tests in isolated temporary databases, and generates coverage reports.

### Key Facts

- **Language**: Go 1.21+
- **Build System**: CGO-enabled (requires C compiler)
- **Target**: Cross-platform (Linux, macOS, Windows)
- **PostgreSQL Support**: Version 13+
- **Output Formats**: JSON, LCOV, HTML

### Project Structure

```
pgcov/
├── cmd/pgcov/              # CLI entry point
├── internal/               # Core implementation
│   ├── cli/               # Command handlers (run, report)
│   ├── coverage/          # Coverage collection & storage
│   ├── database/          # PostgreSQL connection & temp DB management
│   ├── discovery/         # Test/source file discovery
│   ├── instrument/        # SQL/PL/pgSQL instrumentation
│   ├── parser/            # SQL parsing (pg_query_go wrapper)
│   ├── report/            # Report formatters (JSON, LCOV, HTML)
│   ├── runner/            # Test execution & parallelization
│   └── logger/            # Logging utilities
├── testdata/              # Integration test fixtures
├── examples/              # Usage examples
```

---

## Core Principles & Constitution

**CRITICAL**: All development MUST comply with the project constitution. Read [`.specify/memory/constitution.md`](../.specify/memory/constitution.md) before making changes.

### The Six Principles

#### I. Direct Protocol Access

- MUST use native PostgreSQL protocol (via `pgx`)
- NEVER depend on `psql`, shell execution, or external CLI tools
- Ensures consistent behavior across environments

#### II. Test Isolation

- Each test MUST run in a temporary database (created/destroyed per test)
- NO shared state between tests
- Tests MUST be order-independent and parallelizable

#### III. Instrumentation Transparency

- Instrument SQL/PL/pgSQL in-memory via AST rewriting
- NO permanent artifacts or database modifications
- NO PostgreSQL extensions required
- MUST NOT change semantic behavior of tested code

#### IV. CLI-First Design

- Standalone CLI tool with no runtime dependencies (except PostgreSQL)
- Configuration via flags, env vars, or config files
- Clear exit codes for CI/CD integration
- Familiar to Go developers (`go test`-like UX)

#### V. Coverage Accuracy Over Speed

- Deterministic results (same code → same coverage)
- NO false positives or negatives
- Speed optimizations are secondary to correctness

#### VI. Go Development Ergonomics

- Idiomatic Go code patterns
- Command structure similar to Go toolchain
- Standard output formats (JSON, LCOV)
- Clear, actionable error messages

### Non-Goals (Do NOT Implement)

- Database migration management (use external tools)
- Global schema lifecycle management
- Assertion libraries (integrate with pgTAP/others)
- PostgreSQL extensions

---

## Architecture & Design

### Key Technologies

- **CLI Framework**: `urfave/cli/v3` (subcommands, flags, env var binding)
- **SQL Parser**: `pganalyze/pg_query_go/v6` (official PostgreSQL parser)
- **PostgreSQL Driver**: `jackc/pgx/v5` (native protocol, LISTEN/NOTIFY)

### Core Workflows

#### Test Execution Flow

```
1. Discovery: Find *_test.sql files recursively
2. Source Discovery: Find co-located .sql files (same directory)
3. Parsing: Parse source files with pg_query_go
4. Instrumentation: Rewrite AST to inject NOTIFY calls
5. Temp DB Creation: CREATE DATABASE pgcov_test_{timestamp}_{random}
6. Deployment: Load instrumented sources into temp DB
7. Execution: Run test file in temp DB
8. Signal Collection: LISTEN coverage_signal for execution data
9. Cleanup: DROP DATABASE (automatic artifact removal)
10. Reporting: Aggregate coverage and generate reports
```

#### Coverage Signal Format

```
Channel: coverage_signal
Payload: file:line[:branch]
Example: src/auth.sql:42 or src/auth.sql:42:branch_1
```

#### Temporary Database Naming

```
pgcov_test_{timestamp}_{random}
Example: pgcov_test_20260105_a3f9c2b1
```

### Source File Discovery Rules

**Co-located with test files** (same directory only):

For each `path/to/dir/test_name_test.sql`:

1. Get parent directory: `path/to/dir/`
2. Find all `.sql` files in that directory (non-recursive)
3. Exclude files matching `*_test.sql`
4. Remaining files are instrumentable sources

**Example**:

```
myproject/
├── auth/
│   ├── authenticate.sql    # ✅ Instrumented when auth_test.sql runs
│   ├── authorize.sql       # ✅ Instrumented when auth_test.sql runs
│   └── auth_test.sql       # Test file
└── users/
    ├── user_crud.sql       # ✅ Instrumented when user_test.sql runs
    └── user_test.sql       # Test file
```

---

## Development Environment

### Prerequisites

1. **Go 1.21+**: `go version`
2. **C Compiler**:
   - Linux: `sudo apt-get install build-essential`
   - macOS: `xcode-select --install`
   - Windows: MSYS2 + MinGW-w64 (see [BUILD.md](../BUILD.md))
3. **Docker**: For integration tests (PostgreSQL 13-16)
4. **PostgreSQL** (optional): Local instance for manual testing

### Setup

```bash
# Clone repository
git clone https://github.com/cybertec-postgresql/pgcov.git
cd pgcov

# Install dependencies
go mod download

# Build
export CGO_ENABLED=1  # Linux/macOS
# $env:CGO_ENABLED = "1"  # Windows PowerShell
go build -o pgcov ./cmd/pgcov

# Verify
./pgcov --version
```

### VS Code Tasks

Use VS Code tasks for common operations:

- **Build pgcov**: `Ctrl+Shift+B` (or `Cmd+Shift+B` on macOS)
- **Unit Test**: Run tests in current directory
- **Coverage Report**: Generate HTML coverage report
- **Run pgcov**: Execute against testdata/simple
- **Format Code**: Run `go fmt ./...`
- **Go Vet**: Static analysis with `go vet ./...`

---

## Code Standards

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` (enforced in CI)
- Run `go vet` before committing
- Prefer standard library over dependencies
- Keep functions small and focused (< 50 lines ideal)

### Error Handling

```go
// ✅ Good: Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to connect to database %s: %w", dbName, err)
}

// ❌ Bad: Generic error without context
if err != nil {
    return err
}
```

### Logging

```go
// Use internal/logger package
logger := logger.New(verbose)
logger.Info("Discovering tests in %s", path)
logger.Debug("Found test file: %s", filePath)
logger.Error("Failed to parse: %v", err)
```

### Naming Conventions

- **Files**: `snake_case.go` (e.g., `database_pool.go`)
- **Test Files**: `*_test.go` (e.g., `database_pool_test.go`)
- **Packages**: Short, lowercase, no underscores (e.g., `package coverage`)
- **Interfaces**: Noun or adjective (e.g., `type Executor interface {}`)
- **Implementations**: Descriptive (e.g., `type ParallelExecutor struct {}`)

### Documentation

- All exported functions/types MUST have godoc comments
- Start comments with the item name: `// Execute runs the test file...`
- Include usage examples for complex APIs

```go
// Execute runs a test file in an isolated temporary database.
// It returns the test result and any error encountered during execution.
//
// Example:
//
// result, err := executor.Execute(ctx, testFile)
// if err != nil {
//     return fmt.Errorf("test execution failed: %w", err)
// }
func (e *Executor) Execute(ctx context.Context, testFile string) (*Result, error) {
    // ...
}
```

---

## Testing Requirements

### Test Organization

- **Unit Tests**: Alongside source files (`*_test.go`)
- **Integration Tests**: `internal/*_integration_test.go`
- **Test Fixtures**: `testdata/` directory

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/parser

# Verbose output
go test -v ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Integration tests only
go test ./... -run Integration
```

### Unit Test Guidelines

- Use table-driven tests for multiple cases
- Mock external dependencies (database, filesystem)
- Test error paths explicitly
- Avoid testing implementation details

**Example**:

```go
func TestParseSQL(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    *AST
        wantErr bool
    }{
        {"simple select", "SELECT 1", &AST{...}, false},
        {"syntax error", "SLECT 1", nil, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseSQL(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseSQL() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("ParseSQL() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Integration Test Guidelines

- Use Docker for PostgreSQL instances (via `testcontainers-go`)
- Test against multiple PostgreSQL versions (13, 14, 15, 16)
- Create/destroy test databases for each test
- Verify actual SQL execution and coverage collection

**Example**:

```go
func TestIntegration_RunSimpleTest(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // Setup PostgreSQL container
    pg, err := setupPostgres(t)
    require.NoError(t, err)
    defer pg.Cleanup()
    
    // Run test
    result, err := runTest(pg.ConnectionString(), "testdata/simple/math_test.sql")
    require.NoError(t, err)
    assert.True(t, result.Passed)
    assert.Greater(t, result.Coverage, 0.8) // 80% coverage
}
```

### Test Fixtures

**Location**: `testdata/`

**Structure**:

```
testdata/
├── simple/              # Basic test cases
│   ├── math.sql        # Source file
│   ├── math_test.sql   # Test file
│   └── README.md       # Fixture documentation
├── plpgsql/            # PL/pgSQL-specific tests
├── edge_cases/         # Error conditions, empty files, etc.
├── isolation/          # Test isolation verification
└── parallel/           # Parallel execution tests
```

---

## Common Tasks

### Adding a New Command

1. Add command in `internal/cli/`:

```go
// internal/cli/mycommand.go
func MyCommand() *cli.Command {
    return &cli.Command{
        Name:  "mycommand",
        Usage: "Does something useful",
        Flags: []cli.Flag{
            &cli.StringFlag{Name: "option", Usage: "Option description"},
        },
        Action: func(c *cli.Context) error {
            // Implementation
            return nil
        },
    }
}
```

1. Register in `cmd/pgcov/main.go`:

```go
app := &cli.App{
    Commands: []*cli.Command{
        cli.RunCommand(),
        cli.ReportCommand(),
        cli.MyCommand(), // Add here
    },
}
```

1. Add tests in `internal/cli/mycommand_test.go`
2. Update documentation in README.md

### Adding a New Report Format

1. Implement `Reporter` interface:

```go
// internal/report/myformat.go
type MyFormatReporter struct{}

func (r *MyFormatReporter) Generate(coverage *coverage.Coverage) ([]byte, error) {
    // Format conversion logic
    return data, nil
}
```

1. Register in `internal/report/formatter.go`:

```go
func NewReporter(format string) (Reporter, error) {
    switch format {
    case "json":
        return &JSONReporter{}, nil
    case "lcov":
        return &LCOVReporter{}, nil
    case "myformat": // Add here
        return &MyFormatReporter{}, nil
    default:
        return nil, fmt.Errorf("unsupported format: %s", format)
    }
}
```

1. Add tests in `internal/report/myformat_test.go`
2. Update CLI flag documentation

### Modifying Instrumentation

**IMPORTANT**: Changes to instrumentation MUST maintain transparency (Principle III).

1. Understand current AST traversal in `internal/instrument/instrumenter.go`
2. Add new node visitors for additional coverage points
3. Test with multiple PostgreSQL versions (13-16)
4. Verify deterministic output (same input → same instrumented code)
5. Update `specs/001-core-test-runner/research.md` with design rationale

### Adding New Configuration Options

1. Add flag in `internal/cli/config.go`:

```go
type Config struct {
    // Existing fields...
    NewOption string
}

var configFlags = []cli.Flag{
    // Existing flags...
    &cli.StringFlag{
        Name:    "new-option",
        Usage:   "Description of new option",
        EnvVars: []string{"PGCOV_NEW_OPTION"},
        Value:   "default_value",
    },
}
```

1. Add validation in `LoadConfig()` if needed
2. Update documentation in README.md and quickstart.md
3. Add tests for new configuration behavior

---

## CI/CD Integration

### GitHub Actions Workflow

The project uses GitHub Actions for CI/CD (see `.github/workflows/`):

- **Build**: Verify compilation on Linux, macOS, Windows
- **Test**: Run unit and integration tests
- **Lint**: `golangci-lint` with strict rules
- **Coverage**: Upload to Codecov
- **Release**: Build binaries for all platforms

### Pre-commit Checks

Before pushing:

```bash
# Format code
go fmt ./...

# Run linters
go vet ./...
golangci-lint run

# Run tests
go test ./...

# Build for all platforms (if changing build process)
GOOS=linux GOARCH=amd64 go build ./cmd/pgcov
GOOS=darwin GOARCH=amd64 go build ./cmd/pgcov
GOOS=windows GOARCH=amd64 go build ./cmd/pgcov
```

### Commit Message Format

Use conventional commits:

```
feat: Add support for HTML coverage reports
fix: Correct line number calculation in LCOV output
docs: Update quickstart with parallel execution examples
test: Add integration tests for PostgreSQL 16
refactor: Simplify test discovery logic
chore: Update dependencies
```

---

## Troubleshooting

### Build Issues

#### "CGO not enabled"

**Solution**:

```bash
export CGO_ENABLED=1  # Linux/macOS
$env:CGO_ENABLED = "1"  # Windows PowerShell
```

#### "gcc: command not found" (Windows)

**Solution**: Install MSYS2 and add MinGW to PATH:

```powershell
$env:PATH = "$env:PATH;C:\msys64\mingw64\bin"
$env:CC = "C:\msys64\mingw64\bin\gcc.exe"
```

#### "undefined reference to libpg_query"

**Solution**: Clean and rebuild:

```bash
go clean -cache
go build ./cmd/pgcov
```

### Test Issues

#### "no PostgreSQL server running"

**Solution**: Integration tests use Docker. Ensure Docker is running:

```bash
docker ps
```

#### "permission denied to create database"

**Solution**: Grant CREATEDB privilege:

```sql
ALTER USER testuser CREATEDB;
```

### Runtime Issues

#### "failed to instrument source file"

**Cause**: SQL syntax error or unsupported PostgreSQL feature

**Solution**:

1. Check file with `pg_query_go` directly
2. Verify PostgreSQL version compatibility
3. Enable verbose logging: `pgcov run --verbose`

#### "test timeout after 30s"

**Solution**: Increase timeout:

```bash
pgcov run --timeout=60s ./tests/
```

---

## Reference Documentation

### External Resources

- [pg_query_go Documentation](https://pkg.go.dev/github.com/pganalyze/pg_query_go/v6)
- [pgx Driver Documentation](https://pkg.go.dev/github.com/jackc/pgx/v5)
- [PostgreSQL SQL Syntax](https://www.postgresql.org/docs/current/sql.html)
- [LCOV Format Specification](https://linux.die.net/man/1/geninfo)

### Key Packages

- `internal/cli`: Command-line interface handlers
- `internal/coverage`: Coverage data collection and storage
- `internal/database`: PostgreSQL connection management
- `internal/discovery`: Test/source file discovery
- `internal/instrument`: SQL instrumentation via AST rewriting
- `internal/parser`: SQL parsing wrapper (pg_query_go)
- `internal/report`: Coverage report formatters
- `internal/runner`: Test execution and parallelization

---

## Quick Reference

### Build Commands

```bash
# Development build
CGO_ENABLED=1 go build -o pgcov ./cmd/pgcov

# Release build
CGO_ENABLED=1 go build -ldflags="-s -w" -o pgcov ./cmd/pgcov

# Run tests
go test ./...

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run Commands

```bash
# Basic usage
pgcov run ./tests/

# Recursive discovery
pgcov run ./...

# Parallel execution
pgcov run --parallel=4 ./...

# Custom connection
pgcov run --host=db.example.com --port=5432 --user=pgcov ./...

# With timeout
pgcov run --timeout=60s ./...

# Verbose output
pgcov run --verbose ./...

# Generate report
pgcov report --format=lcov -o coverage.lcov
```

### PostgreSQL Setup

```bash
# Environment variables
export PGHOST=localhost
export PGPORT=5432
export PGUSER=postgres
export PGPASSWORD=yourpassword
export PGDATABASE=postgres

# Grant privileges
psql -c "ALTER USER youruser CREATEDB;"

# Verify connection
psql -h localhost -p 5432 -U postgres -c "SELECT version();"
```

---

## Support

- **Issues**: <https://github.com/cybertec-postgresql/pgcov/issues>
- **Discussions**: <https://github.com/cybertec-postgresql/pgcov/discussions>
- **Documentation**: <https://github.com/cybertec-postgresql/pgcov/tree/main/docs>

---

**Last Updated**: 2026-01-09  
**Version**: 1.0.0
