# Quickstart: pgcov

**Feature**: Core Test Runner and Coverage  
**Date**: 2026-01-05

## Prerequisites

- Go 1.21 or later
- PostgreSQL 13 or later (running and accessible)
- PostgreSQL connection credentials

---

## Installation

```bash
# Clone repository
git clone https://github.com/yourorg/pgcov.git
cd pgcov

# Build binary
go build -o pgcov ./cmd/pgcov

# (Optional) Install to PATH
go install ./cmd/pgcov
```

---

## Quick Start

### 1. Set up PostgreSQL connection

```bash
export PGHOST=localhost
export PGPORT=5432
export PGUSER=postgres
export PGPASSWORD=yourpassword
export PGDATABASE=postgres  # Template database for creating test databases
```

### 2. Create test files

Create a test file `auth/auth_test.sql`:

```sql
-- Source file: authenticate.sql (in same directory)
CREATE FUNCTION authenticate(user_id INT) RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS(SELECT 1 FROM users WHERE id = user_id);
END;
$$ LANGUAGE plpgsql;

-- Test case
DO $$
BEGIN
    -- Setup
    CREATE TABLE users (id INT PRIMARY KEY, name TEXT);
    INSERT INTO users VALUES (1, 'Alice'), (2, 'Bob');
    
    -- Test assertions
    IF NOT authenticate(1) THEN
        RAISE EXCEPTION 'Test failed: authenticate(1) should return true';
    END IF;
    
    IF authenticate(999) THEN
        RAISE EXCEPTION 'Test failed: authenticate(999) should return false';
    END IF;
    
    RAISE NOTICE 'All tests passed';
END;
$$;
```

### 3. Create source files

Create `auth/authenticate.sql` (same directory as test, will be instrumented for coverage):

```sql
CREATE OR REPLACE FUNCTION validate_user(user_id INT) RETURNS BOOLEAN AS $$
BEGIN
    RETURN user_id > 0;
END;
$$ LANGUAGE plpgsql;
```

**Note**: Source files must be in the same directory as their test files!

### 4. Run tests

```bash
# Discover and run tests in current directory
pgcov run .

# Recursive discovery (Go-style)
pgcov run ./...

# Specific directory
pgcov run ./tests/
```

**Expected output**:

```
Discovering tests...
Found 1 test file(s), 1 source file(s)

Running tests...
✓ auth_test.sql (1.2s)

Tests: 1 passed, 0 failed
Coverage: 85.7% (6/7 lines)
Coverage data written to .pgcov/coverage.json
```

### 5. Generate coverage reports

```bash
# JSON format (default)
pgcov report --format=json

# LCOV format (for CI integration)
pgcov report --format=lcov -o coverage.lcov
```

---

## Configuration Options

### Connection

```bash
# Via environment variables
export PGHOST=localhost
export PGPORT=5432
export PGUSER=pgcov_user
export PGPASSWORD=secret
export PGDATABASE=postgres

# Via command-line flags (overrides env vars)
pgcov run . --host=localhost --port=5432 --user=pgcov_user
```

### Execution

```bash
# Set per-test timeout (default: 30s)
pgcov run . --timeout=60s

# Enable parallel execution (4 concurrent tests)
pgcov run . --parallel=4

# Verbose output (show SQL queries and coverage signals)
pgcov run . --verbose
```

### Coverage

```bash
# Custom coverage file location
pgcov run . --coverage-file=./coverage/data.json

# Generate report from custom location
pgcov report --coverage-file=./coverage/data.json --format=lcov
```

---

## Project Structure

Recommended project layout:

```
myproject/
├── auth/
│   ├── authenticate.sql      # Source file (instrumented)
│   ├── authorize.sql         # Source file (instrumented)
│   └── auth_test.sql         # Test file
├── users/
│   ├── user_crud.sql         # Source file (instrumented)
│   └── user_test.sql         # Test file
├── .pgcov/
│   └── coverage.json         # Coverage data (auto-generated)
└── .gitignore                # Add .pgcov/ to ignore list
```

**Important**: Source files MUST be in the same directory as their test files. pgcov will instrument all `.sql` files (excluding `*_test.sql`) in each test directory.

Run tests:

```bash
pgcov run ./...
```

---

## Writing Tests

### Test File Naming

Test files must match `*_test.sql` pattern:

✅ `auth_test.sql`  
✅ `user_functions_test.sql`  
❌ `test_auth.sql` (wrong prefix)  
❌ `auth-test.sql` (wrong suffix)

### Test Structure

Each test file should:

1. Create necessary schema (tables, functions)
2. Execute test logic
3. Assert expected outcomes
4. Clean up (optional - temporary database handles this)

### Assertions

Use SQL `RAISE EXCEPTION` for failures:

```sql
-- Simple assertion
IF NOT condition THEN
    RAISE EXCEPTION 'Test failed: expected X but got Y';
END IF;

-- Using pgTAP (if installed)
SELECT plan(3);
SELECT ok(authenticate(1), 'User 1 should authenticate');
SELECT ok(NOT authenticate(999), 'Invalid user should not authenticate');
SELECT finish();
```

---

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
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432
    
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Install pgcov
        run: go install github.com/yourorg/pgcov/cmd/pgcov@latest
      
      - name: Run tests
        env:
          PGHOST: localhost
          PGPORT: 5432
          PGUSER: postgres
          PGPASSWORD: postgres
        run: pgcov run ./sql/...
      
      - name: Generate coverage report
        run: pgcov report --format=lcov -o coverage.lcov
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: coverage.lcov
```

---

## Troubleshooting

### Tests Not Discovered

**Problem**: `Found 0 test file(s)`

**Solution**: Ensure files match `*_test.sql` pattern and are in specified directory.

```bash
# Check file naming
ls -la *_test.sql

# Verify search path
pgcov run ./tests/ --verbose
```

### Connection Failed

**Problem**: `failed to connect to PostgreSQL`

**Solution**: Verify connection details and PostgreSQL is running.

```bash
# Test connection manually
psql -h localhost -p 5432 -U postgres -d postgres -c "SELECT version();"

# Check environment variables
env | grep PG
```

### Permission Denied

**Problem**: `ERROR: permission denied to create database`

**Solution**: User needs CREATEDB privilege.

```sql
-- Grant privilege
ALTER USER pgcov_user CREATEDB;
```

### Test Timeout

**Problem**: `test timeout after 30s`

**Solution**: Increase timeout or optimize test.

```bash
# Increase timeout to 60 seconds
pgcov run . --timeout=60s
```

### Instrumentation Failed

**Problem**: `failed to instrument source file`

**Solution**: Check SQL syntax and PostgreSQL version compatibility.

```bash
# Test parse manually
pgcov run . --verbose  # Shows parse errors with line numbers
```

---

## Next Steps

- **Parallel Execution**: Use `--parallel=N` to speed up large test suites
- **Coverage Thresholds**: Integrate with CI to enforce minimum coverage (future feature)
- **Branch Coverage**: Analyze PL/pgSQL control flow coverage (future feature)
- **HTML Reports**: Generate visual coverage reports (future feature)

---

## Getting Help

- **Documentation**: [Full documentation](https://github.com/yourorg/pgcov/docs)
- **Issues**: [GitHub Issues](https://github.com/yourorg/pgcov/issues)
- **Examples**: [Example projects](https://github.com/yourorg/pgcov/tree/main/examples)

---

## Example: Complete Workflow

```bash
# 1. Setup
export PGHOST=localhost PGPORT=5432 PGUSER=postgres PGPASSWORD=secret

# 2. Write tests
cat > user_test.sql << 'EOF'
CREATE TABLE users (id INT, name TEXT);
INSERT INTO users VALUES (1, 'Alice');

DO $$
BEGIN
    IF NOT EXISTS(SELECT 1 FROM users WHERE id = 1) THEN
        RAISE EXCEPTION 'User 1 not found';
    END IF;
END $$;
EOF

# 3. Write source
cat > user_functions.sql << 'EOF'
CREATE FUNCTION get_user_name(user_id INT) RETURNS TEXT AS $$
DECLARE
    user_name TEXT;
BEGIN
    SELECT name INTO user_name FROM users WHERE id = user_id;
    RETURN user_name;
END;
$$ LANGUAGE plpgsql;
EOF

# 4. Run tests
pgcov run .

# 5. Check coverage
pgcov report --format=json | jq '.files."user_functions.sql".lines'

# 6. Generate LCOV for CI
pgcov report --format=lcov -o coverage.lcov
```

**Output**:

```
Discovering tests...
Found 1 test file(s), 1 source file(s)

Running tests...
✓ user_test.sql (0.8s)

Tests: 1 passed, 0 failed
Coverage: 100.0% (7/7 lines)
Coverage data written to .pgcov/coverage.json
```

---

## Summary

You now have pgcov running! Key points:

- ✅ Test files must match `*_test.sql` pattern
- ✅ Source files in same directory tree get instrumented automatically
- ✅ Each test runs in isolated temporary database
- ✅ Coverage data persists in `.pgcov/coverage.json`
- ✅ Use `pgcov report` to export JSON or LCOV formats

Happy testing! 🎉
