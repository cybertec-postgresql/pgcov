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
- **PostgreSQL**: 13 or later (running and accessible)
- **Permissions**: CREATEDB privilege for test isolation

## Installation

```bash
# Clone repository
git clone https://github.com/pashagolub/pgcov.git
cd pgcov

# Build binary
go build -o pgcov ./cmd/pgcov

# (Optional) Install to PATH
go install ./cmd/pgcov
```

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
- `--port`: PostgreSQL port (default: `5432`)
- `--user`: PostgreSQL user (default: current user)
- `--password`: PostgreSQL password
- `--database`: Template database (default: `postgres`)

**Execution**:
- `--timeout`: Per-test timeout (default: `30s`)
- `--parallel`: Concurrent tests (default: `1`)
- `--verbose`: Enable debug output

**Output**:
- `--coverage-file`: Coverage data path (default: `.pgcov/coverage.json`)

### Environment Variables

pgcov respects standard PostgreSQL environment variables:
- `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`

Command-line flags override environment variables.

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

```bash
# Run tests
go test ./...

# Build
go build -o pgcov ./cmd/pgcov

# Format code
go fmt ./...

# Lint
go vet ./...
```

## License

MIT

## Contributing

Contributions welcome! Please open an issue or pull request.

## Support

- **Documentation**: [Full docs](https://github.com/pashagolub/pgcov/docs)
- **Issues**: [GitHub Issues](https://github.com/pashagolub/pgcov/issues)
- **Examples**: [Examples directory](./examples)
