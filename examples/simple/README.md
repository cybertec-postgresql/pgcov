# Simple pgcov Example

This is a minimal example showing how to use pgcov to test SQL code.

## Structure

```
simple/
├── calculate.sql       # Source file with function to test
├── calculate_test.sql  # Test file (must be in same directory)
└── README.md          # This file
```

## Source File

`calculate.sql` contains a PL/pgSQL function that calculates the total price:
- Takes quantity and price as inputs
- Validates inputs (no negative values)
- Returns the total (quantity × price)

## Test File

`calculate_test.sql` contains tests for the calculate_total function:
- Normal calculation test
- Edge case: zero quantity
- Error case: negative quantity
- Error case: negative price

## Running Tests

### Prerequisites

Set up PostgreSQL connection:

```bash
export PGHOST=localhost
export PGPORT=5432
export PGUSER=postgres
export PGPASSWORD=yourpassword
```

### Run the tests

```bash
# From the simple/ directory
pgcov run .

# From the repository root
pgcov run examples/simple/
```

### Expected Output

```
Discovering tests...
Found 1 test file(s), 1 source file(s)

Running tests...
✓ calculate_test.sql (0.5s)

Tests: 1 passed, 0 failed
Coverage: 100.0% (10/10 lines)
Coverage data written to .pgcov/coverage.json
```

## Generate Reports

### JSON Report

```bash
pgcov report --format=json
```

### LCOV Report

```bash
pgcov report --format=lcov -o coverage.lcov
```

### HTML Report (if genhtml is installed)

```bash
pgcov report --format=lcov -o coverage.lcov
genhtml coverage.lcov -o coverage_html/
```

## Key Concepts

1. **Co-location**: Test files and source files must be in the same directory
2. **Naming convention**: Test files must end with `_test.sql`
3. **Source files**: Any `.sql` file without `_test` suffix is treated as source
4. **Assertions**: Use `ASSERT` or `RAISE EXCEPTION` to verify behavior
5. **Isolation**: Each test runs in a temporary database (automatically created and destroyed)

## Next Steps

- Add more test files in the same directory
- Test different code paths to increase coverage
- Use `--verbose` flag to see detailed execution logs
- Try parallel execution with `--parallel=4`
