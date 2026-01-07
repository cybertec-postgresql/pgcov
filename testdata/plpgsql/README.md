# PL/pgSQL Test Fixtures

This directory contains PL/pgSQL functions, procedures, and triggers for testing code coverage in procedural PostgreSQL code.

## Files

### function_source.sql
Contains various PL/pgSQL functions:
- `add_numbers(a, b)`: Simple arithmetic function
- `get_grade(score)`: Function with conditional logic (IF/ELSIF/ELSE)
- `sum_to_n(n)`: Function with loop (FOR)
- `safe_divide(a, b)`: Function with exception handling

**Purpose**: Test coverage tracking in PL/pgSQL functions with different control flow patterns.

### function_test.sql
Test cases for the functions in `function_source.sql`.
Uses anonymous DO blocks with ASSERT statements to verify function behavior.

**Purpose**: Execute tests that trigger coverage instrumentation in the functions.

### procedure_source.sql
Contains stored procedures:
- `log_event(event_name)`: Simple procedure with RAISE NOTICE
- `update_user_status(user_id, new_status)`: Procedure simulating table operations
- `batch_insert(start_id, end_id)`: Procedure with loop and COMMIT

**Purpose**: Test coverage tracking in stored procedures (PostgreSQL 11+).

### procedure_test.sql
Test cases for the procedures in `procedure_source.sql`.
Uses CALL statements to execute procedures.

**Purpose**: Execute tests that trigger coverage instrumentation in procedures.

### trigger_source.sql
Contains trigger definitions and trigger functions:
- `update_timestamp()`: Trigger function to update timestamp on row updates
- `log_change()`: Trigger function to log operations to audit table
- Trigger definitions for BEFORE UPDATE and AFTER INSERT events

**Purpose**: Test coverage tracking in trigger functions.

### trigger_test.sql
Test cases for triggers in `trigger_source.sql`.
Performs INSERT and UPDATE operations to fire triggers, then verifies results.

**Purpose**: Execute tests that trigger coverage instrumentation in trigger functions.

## Usage

These fixtures test PL/pgSQL code coverage:

```bash
# Run PL/pgSQL function tests
pgcov run testdata/plpgsql/function_test.sql --database postgres

# Run procedure tests
pgcov run testdata/plpgsql/procedure_test.sql --database postgres

# Run trigger tests
pgcov run testdata/plpgsql/trigger_test.sql --database postgres
```

## Coverage Expectations

For PL/pgSQL code, pgcov instruments:
- Each executable line within BEGIN...END blocks
- Lines inside loops (FOR, WHILE, LOOP)
- Lines inside conditional branches (IF, ELSIF, ELSE, CASE)
- Lines with RAISE, RETURN, PERFORM statements
- Exception handlers (EXCEPTION WHEN clauses)

Skipped (not instrumented):
- Comments
- BEGIN and END keywords
- Declaration sections (DECLARE)
- Control flow keywords (ELSIF, ELSE, LOOP, END IF, etc.)

## Testing Different Code Paths

### Conditional Coverage (get_grade)
To achieve full coverage of `get_grade`, run tests with scores:
- >= 90 (A branch)
- 80-89 (B branch)
- 70-79 (C branch)
- < 70 (F branch)

### Loop Coverage (sum_to_n)
To achieve full coverage of `sum_to_n`, test:
- n > 0 (loop executes)
- n = 0 (loop doesn't execute)

### Exception Coverage (safe_divide)
To achieve full coverage of `safe_divide`, test:
- Normal division (no exception)
- Division by zero (exception path)

## Adding New PL/pgSQL Fixtures

When adding new PL/pgSQL test fixtures:
1. Create the source file with function/procedure/trigger definitions
2. Create corresponding test file with test cases
3. Document the fixtures in this README
4. Ensure test cases cover all code paths for maximum coverage
