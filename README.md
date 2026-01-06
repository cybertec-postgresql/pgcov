# pgcov

PostgreSQL test runner and coverage tool

## Overview

pgcov is a Go-based CLI tool that discovers `*_test.sql` files, instruments SQL/PL/pgSQL source code for coverage tracking, executes tests in isolated temporary databases, and generates coverage reports in JSON and LCOV formats.

## Features

- 🧪 **Automatic Test Discovery**: Finds `*_test.sql` files and co-located source files
- 🔒 **Complete Test Isolation**: Each test runs in a temporary database
- 📊 **Coverage Tracking**: Statement-level coverage via SQL instrumentation
- 📈 **Multiple Report Formats**: JSON and LCOV output for CI/CD integration
- ⚡ **Parallel Execution**: Optional concurrent test execution with `--parallel` flag
- 🎯 **PostgreSQL Native**: Direct protocol access via pgx, no external dependencies

## Prerequisites

- **Go**: 1.21 or later (for building)
- **C Compiler**: Required for CGO (GCC on Linux, MinGW-w64 on Windows)
- **PostgreSQL**: 13 or later (running and accessible)
- **Permissions**: CREATEDB privilege for test isolation

### C Compiler Setup

**Linux/macOS**:
```bash
# Ubuntu/Debian
sudo apt-get install build-essential

# macOS (Xcode Command Line Tools)
xcode-select --install

# Fedora/RHEL
sudo dnf install gcc
```

**Windows**:
- Install [MSYS2](https://www.msys2.org/)
- Open MSYS2 terminal and run:
  ```bash
  pacman -S mingw-w64-x86_64-gcc
  ```
- Add `C:\msys64\mingw64\bin` to your PATH

## Installation

### Building from Source

**Linux/macOS**:
```bash
# Clone repository
git clone https://github.com/pashagolub/pgcov.git
cd pgcov

# Enable CGO and build
export CGO_ENABLED=1
go build -o pgcov ./cmd/pgcov

# (Optional) Install to PATH
go install ./cmd/pgcov
```

**Windows (PowerShell) - Quick Build**:
```powershell
# Clone repository
git clone https://github.com/pashagolub/pgcov.git
cd pgcov

# Use the build script (handles CGO automatically)
.\build.ps1
```

**Windows (PowerShell) - Manual Build**:
```powershell
# Clone repository
git clone https://github.com/pashagolub/pgcov.git
cd pgcov

# Enable CGO and set compiler
$env:CGO_ENABLED = "1"
$env:CC = "C:\msys64\mingw64\bin\gcc.exe"
$env:PATH = "$env:PATH;C:\msys64\mingw64\bin"

# Build
go build -o pgcov.exe .\cmd\pgcov
```

**Windows (CMD)**:
```cmd
REM Clone repository
git clone https://github.com/pashagolub/pgcov.git
cd pgcov

REM Enable CGO and set compiler
set CGO_ENABLED=1
set CC=C:\msys64\mingw64\bin\gcc.exe
set PATH=%PATH%;C:\msys64\mingw64\bin

REM Build
go build -o pgcov.exe .\cmd\pgcov
```

### Why CGO is Required

pgcov uses `pg_query_go` which wraps the PostgreSQL query parser (libpg_query) written in C. This provides native PostgreSQL SQL parsing capabilities but requires CGO to be enabled during compilation.

## Quick Start

### 1. Configure PostgreSQL Connection

```bash
export PGHOST=localhost
export PGPORT=5432
export PGUSER=postgres
export PGPASSWORD=yourpassword
export PGDATABASE=postgres
```

### 2. Create Test Files

Test files must match `*_test.sql` pattern and be co-located with source files:

```
myproject/
├── auth/
│   ├── authenticate.sql      # Source (will be instrumented)
│   └── auth_test.sql          # Test
```

### 3. Run Tests

```bash
# Current directory
pgcov run .

# Recursive (Go-style)
pgcov run ./...

# Specific directory
pgcov run ./tests/
```

### 4. Generate Coverage Reports

```bash
# JSON format (default)
pgcov report --format=json

# LCOV format (for CI)
pgcov report --format=lcov -o coverage.lcov
```

## Usage

### Commands

```bash
# Run tests and collect coverage
pgcov run [path]

# Generate coverage report
pgcov report [--format=json|lcov] [-o output-file]

# Show help
pgcov help [command]

# Show version
pgcov --version
```

### Configuration Flags

**Connection**:
- `--host`: PostgreSQL host (default: `localhost`)
- `--port`: PostgreSQL port (default: `5432`, valid range: 1-65535)
- `--user`: PostgreSQL user (default: current user)
- `--password`: PostgreSQL password
- `--database`: Template database (default: `postgres`)

**Execution**:
- `--timeout`: Per-test timeout (default: `30s`, format: `10s`, `1m`, `90s`)
- `--parallel`: Concurrent tests (default: `1`, valid range: 1-100)
- `--verbose`: Enable debug output

**Output**:
- `--coverage-file`: Coverage data path (default: `.pgcov/coverage.json`)

### Environment Variables

pgcov respects standard PostgreSQL environment variables:
- `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`

**Configuration Priority** (highest to lowest):
1. Command-line flags (e.g., `--host`)
2. Environment variables (e.g., `PGHOST`)
3. Default values

### Configuration Validation

pgcov validates all configuration values and provides helpful error messages:

```bash
# Invalid port
$ pgcov run --port=99999 .
Error: configuration error for port: invalid port number: 99999

Suggestion: Port must be between 1 and 65535. Default PostgreSQL port is 5432.
Set via --port flag or PGPORT environment variable.

# Invalid parallelism
$ pgcov run --parallel=0 .
Error: configuration error for parallel: parallelism must be at least 1, got: 0

Suggestion: Use --parallel=N where N is number of tests to run concurrently.
Use 1 for sequential execution.

# Invalid timeout
$ pgcov run --timeout=-5s .
Error: configuration error for timeout: timeout must be positive

Suggestion: Use --timeout flag with format like '30s', '1m', '90s'. Default is 30s.
```

## Writing Tests

### Test File Structure

```sql
-- auth_test.sql

-- Setup: Create schema and test data
CREATE TABLE users (id INT PRIMARY KEY, name TEXT);
INSERT INTO users VALUES (1, 'Alice'), (2, 'Bob');

-- Test: Verify behavior
DO $$
BEGIN
    IF NOT authenticate(1) THEN
        RAISE EXCEPTION 'Test failed: User 1 should authenticate';
    END IF;
    
    IF authenticate(999) THEN
        RAISE EXCEPTION 'Test failed: Invalid user should not authenticate';
    END IF;
    
    RAISE NOTICE 'All tests passed';
END;
$$;
```

### Source File Structure

Source files in the same directory as test files will be automatically instrumented:

```sql
-- authenticate.sql

CREATE OR REPLACE FUNCTION authenticate(user_id INT) RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(SELECT 1 FROM users WHERE id = user_id);
END;
$$ LANGUAGE plpgsql;
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
        ports:
          - 5432:5432
    
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Install pgcov
        run: go install github.com/pashagolub/pgcov/cmd/pgcov@latest
      
      - name: Run tests
        env:
          PGHOST: localhost
          PGPORT: 5432
          PGUSER: postgres
          PGPASSWORD: postgres
        run: pgcov run ./...
      
      - name: Generate LCOV report
        run: pgcov report --format=lcov -o coverage.lcov
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: coverage.lcov
```

## Architecture

- **CLI Layer**: Command routing and user interface (`urfave/cli/v3`)
- **Discovery Layer**: Test and source file discovery (filesystem traversal)
- **Parser Layer**: SQL parsing and AST access (`pg_query_go`)
- **Instrumentation Layer**: AST rewriting with coverage injection
- **Database Layer**: PostgreSQL connections and temporary databases (`pgx/v5`)
- **Runner Layer**: Test execution orchestration and isolation
- **Coverage Layer**: Signal collection and aggregation (LISTEN/NOTIFY)
- **Reporter Layer**: Output formatting (JSON, LCOV)

## Development

### Running Tests

The project includes comprehensive integration tests that use testcontainers to spin up a PostgreSQL instance.

**Linux/macOS**:
```bash
# Enable CGO
export CGO_ENABLED=1

# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific test
go test -v ./internal -run TestEndToEndWithTestcontainers

# Run with timeout (useful for integration tests)
go test -timeout 5m ./...

# Run tests with coverage
go test -cover ./...
```

**Windows (PowerShell) - Quick Test**:
```powershell
# Use the test script (handles CGO automatically)
.\test.ps1

# Run with verbose output
.\test.ps1 -Verbose

# Run specific tests
.\test.ps1 -Run "TestConfig" -Verbose

# Run short tests only (skips integration tests)
.\test.ps1 -Short

# Run specific package
.\test.ps1 -Package ".\internal\cli\..." -Verbose
```

**Windows (PowerShell) - Manual**:
```powershell
# Enable CGO and set compiler
$env:CGO_ENABLED = "1"
$env:CC = "C:\msys64\mingw64\bin\gcc.exe"
$env:PATH = "$env:PATH;C:\msys64\mingw64\bin"

# Run all tests
go test .\...

# Run with verbose output
go test -v .\...

# Run specific test
go test -v .\internal -run TestEndToEndWithTestcontainers

# Run with timeout
go test -timeout 5m .\...

# Run tests with coverage
go test -cover .\...
```

**Windows (CMD)**:
```cmd
REM Enable CGO and set compiler
set CGO_ENABLED=1
set CC=C:\msys64\mingw64\bin\gcc.exe
set PATH=%PATH%;C:\msys64\mingw64\bin

REM Run all tests
go test .\...

REM Run with verbose output
go test -v .\...
```

### Building

**Linux/macOS**:
```bash
# Development build
export CGO_ENABLED=1
go build -o pgcov ./cmd/pgcov

# Release build with optimizations
go build -ldflags="-s -w" -o pgcov ./cmd/pgcov

# Format code
go fmt ./...

# Lint
go vet ./...

# Clean build cache
go clean -cache
```

**Windows (PowerShell)**:
```powershell
# Development build
$env:CGO_ENABLED = "1"
$env:CC = "C:\msys64\mingw64\bin\gcc.exe"
$env:PATH = "$env:PATH;C:\msys64\mingw64\bin"
go build -o pgcov.exe .\cmd\pgcov

# Release build with optimizations
go build -ldflags="-s -w" -o pgcov.exe .\cmd\pgcov

# Format code
go fmt .\...

# Lint
go vet .\...

# Clean build cache
go clean -cache
```

### Test Requirements

**Docker**: Integration tests use testcontainers-go which requires Docker to be running:
- Linux: Docker Engine
- macOS: Docker Desktop
- Windows: Docker Desktop with WSL2 backend

**PostgreSQL Version**: Tests verify PostgreSQL 13+ compatibility using the `postgres:16-alpine` image.

### Troubleshooting Build Issues

**CGO errors on Linux**:
```bash
# Install build tools
sudo apt-get update
sudo apt-get install build-essential

# Verify GCC is available
gcc --version
```

**CGO errors on Windows**:
```powershell
# Verify GCC is in PATH
gcc --version

# If not found, ensure MSYS2 MinGW64 is in PATH:
$env:PATH = "$env:PATH;C:\msys64\mingw64\bin"
```

**Missing DLL errors on Windows**:
Ensure `C:\msys64\mingw64\bin` is in your PATH to access required MinGW DLLs.

**Test container startup failures**:
```bash
# Verify Docker is running
docker ps

# Pull PostgreSQL image manually
docker pull postgres:16-alpine
```

## VS Code Integration

This project includes complete VS Code configuration for CGO development.

### Features

- ✅ **Automatic CGO Environment** - No manual env var setup required
- ✅ **IntelliSense Support** - Full code completion for CGO code
- ✅ **Debug Configurations** - F5 to debug, with 5 pre-configured scenarios
- ✅ **Build Tasks** - Ctrl+Shift+B to build, plus 9 other tasks
- ✅ **Integrated Terminal** - CGO variables automatically set
- ✅ **Cross-Platform** - Windows, Linux, and macOS configurations

### Quick Start

1. **Open workspace in VS Code**
   ```bash
   code .
   ```

2. **Reload window** (if already open)
   - Press `Ctrl+Shift+P` (or `Cmd+Shift+P` on macOS)
   - Type "Reload Window"
   - Press Enter

3. **Verify gopls is working**
   - Check bottom-right status bar
   - Should show "gopls" without errors

4. **Build the project**
   - Press `Ctrl+Shift+B` (or `Cmd+Shift+B` on macOS)
   - Or: Terminal → Run Build Task

5. **Debug the project**
   - Open Run and Debug sidebar (`Ctrl+Shift+D`)
   - Select "Launch pgcov"
   - Press `F5`

### Configuration Files

See [.vscode/README.md](.vscode/README.md) for detailed documentation:
- **settings.json** - CGO environment for Go tools and terminal
- **launch.json** - Debug configurations
- **tasks.json** - Build and test tasks

### Requirements

**Windows Users**: Ensure MSYS2 MinGW-w64 GCC is installed at `C:\msys64\mingw64\bin\gcc.exe`

If installed elsewhere, update the `CC` path in `.vscode/settings.json`.

## License

MIT

## Contributing

Contributions welcome! Please open an issue or pull request.

## Support

- **Documentation**: [Full docs](https://github.com/pashagolub/pgcov/docs)
- **Issues**: [GitHub Issues](https://github.com/pashagolub/pgcov/issues)
- **Examples**: [Examples directory](./examples)
