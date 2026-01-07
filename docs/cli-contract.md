# CLI Contract: pgcov

**Version**: 1.0  
**Date**: 2026-01-05

## Overview

This document defines the command-line interface contract for pgcov, including commands, flags, exit codes, and output formats.

---

## Commands

### `pgcov run [path]`

Discover tests and source files, execute tests with coverage tracking, and generate coverage data.

**Arguments**:
- `[path]`: Directory or pattern to search (default: `.`)
  - `.` - Current directory only
  - `./...` - Recursive from current directory (Go-style)
  - `./tests/` - Specific directory

**Flags**:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--host` | string | `localhost` | PostgreSQL host |
| `--port` | int | `5432` | PostgreSQL port |
| `--user` | string | current user | PostgreSQL user |
| `--password` | string | (empty) | PostgreSQL password |
| `--database` | string | `postgres` | Template database for test databases |
| `--timeout` | duration | `30s` | Per-test timeout |
| `--parallel` | int | `1` | Maximum concurrent tests (1 = sequential) |
| `--coverage-file` | string | `.pgcov/coverage.json` | Coverage data output path |
| `--verbose` | bool | `false` | Enable debug output |

**Exit Codes**:
- `0`: All tests passed
- `1`: One or more tests failed
- `2`: Configuration error (e.g., invalid flags, connection failure)
- `3`: No tests discovered

**stdout Output**:

```
Discovering tests...
Found 3 test file(s), 5 source file(s)

Running tests...
✓ auth_test.sql (1.2s)
✓ user_test.sql (0.8s)
✗ payment_test.sql (2.1s)
  ERROR: relation "payments" does not exist
  Line: 15

Tests: 2 passed, 1 failed
Coverage: 78.5% (22/28 lines)
Coverage data written to .pgcov/coverage.json
```

**stderr Output** (errors only):

```
Error: failed to connect to PostgreSQL
  Host: localhost:5432
  User: postgres
  Error: password authentication failed

Suggestion: Set PGPASSWORD environment variable or use --password flag
```

**Environment Variables**:
- `PGHOST`: PostgreSQL host (overridden by `--host`)
- `PGPORT`: PostgreSQL port (overridden by `--port`)
- `PGUSER`: PostgreSQL user (overridden by `--user`)
- `PGPASSWORD`: PostgreSQL password (overridden by `--password`)
- `PGDATABASE`: Template database (overridden by `--database`)

---

### `pgcov report`

Generate coverage report from existing coverage data.

**Arguments**: None

**Flags**:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | string | `json` | Output format (`json` or `lcov`) |
| `--output`, `-o` | string | stdout | Output file path (use `-` for stdout) |
| `--coverage-file` | string | `.pgcov/coverage.json` | Coverage data input path |

**Exit Codes**:
- `0`: Report generated successfully
- `1`: Coverage data file not found
- `2`: Invalid format or output path

**stdout Output** (JSON format):

```json
{
  "version": "1.0",
  "timestamp": "2026-01-05T16:00:00Z",
  "files": {
    "src/auth.sql": {
      "path": "src/auth.sql",
      "lines": {
        "42": {"line_number": 42, "hit_count": 5, "covered": true}
      }
    }
  }
}
```

**stdout Output** (LCOV format):

```
TN:
SF:src/auth.sql
DA:42,5
DA:43,0
end_of_record
```

---

### `pgcov help [command]`

Display help information.

**Arguments**:
- `[command]`: Optional command name for detailed help

**Exit Codes**:
- `0`: Always

**stdout Output**:

```
NAME:
   pgcov - PostgreSQL test runner and coverage tool

USAGE:
   pgcov [global options] command [command options] [arguments...]

VERSION:
   1.0.0

COMMANDS:
   run      Run tests and collect coverage
   report   Generate coverage report
   help     Show help

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

---

### `pgcov --version`

Display version information.

**Exit Codes**:
- `0`: Always

**stdout Output**:

```
pgcov version 1.0.0
```

---

## Coverage Data File Contract

### File Path

Default: `.pgcov/coverage.json`  
Configurable via: `--coverage-file` flag

### JSON Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["version", "timestamp", "files"],
  "properties": {
    "version": {
      "type": "string",
      "description": "Schema version (semantic versioning)"
    },
    "timestamp": {
      "type": "string",
      "format": "date-time",
      "description": "ISO 8601 timestamp of coverage collection"
    },
    "files": {
      "type": "object",
      "additionalProperties": {
        "$ref": "#/definitions/FileCoverage"
      }
    }
  },
  "definitions": {
    "FileCoverage": {
      "type": "object",
      "required": ["path", "lines"],
      "properties": {
        "path": {
          "type": "string",
          "description": "Relative file path"
        },
        "lines": {
          "type": "object",
          "additionalProperties": {
            "$ref": "#/definitions/LineCoverage"
          }
        },
        "branches": {
          "type": "object",
          "additionalProperties": {
            "$ref": "#/definitions/BranchCoverage"
          }
        }
      }
    },
    "LineCoverage": {
      "type": "object",
      "required": ["line_number", "hit_count", "covered"],
      "properties": {
        "line_number": {
          "type": "integer",
          "minimum": 1
        },
        "hit_count": {
          "type": "integer",
          "minimum": 0
        },
        "covered": {
          "type": "boolean"
        }
      }
    },
    "BranchCoverage": {
      "type": "object",
      "required": ["branch_id", "hit_count", "covered"],
      "properties": {
        "branch_id": {
          "type": "string",
          "description": "Branch identifier (e.g., '44:if_true')"
        },
        "hit_count": {
          "type": "integer",
          "minimum": 0
        },
        "covered": {
          "type": "boolean"
        }
      }
    }
  }
}
```

### Example Coverage Data File

```json
{
  "version": "1.0",
  "timestamp": "2026-01-05T16:00:00Z",
  "files": {
    "src/auth.sql": {
      "path": "src/auth.sql",
      "lines": {
        "42": {
          "line_number": 42,
          "hit_count": 5,
          "covered": true
        },
        "43": {
          "line_number": 43,
          "hit_count": 0,
          "covered": false
        }
      },
      "branches": {
        "44:if_true": {
          "branch_id": "44:if_true",
          "hit_count": 3,
          "covered": true
        },
        "44:if_false": {
          "branch_id": "44:if_false",
          "hit_count": 2,
          "covered": true
        }
      }
    }
  }
}
```

---

## LCOV Output Contract

### Format Specification

LCOV trace file format (compatible with genhtml and coverage.py).

### Example Output

```
TN:
SF:src/auth.sql
DA:42,5
DA:43,0
DA:50,1
BRDA:44,0,0,3
BRDA:44,0,1,2
LH:2
LF:3
BRH:2
BRF:2
end_of_record

SF:src/user.sql
DA:10,8
DA:11,8
DA:12,0
LH:2
LF:3
end_of_record
```

**Legend**:
- `TN:` - Test name (empty for pgcov)
- `SF:` - Source file path
- `DA:line,hitcount` - Line coverage data
- `BRDA:line,block,branch,hitcount` - Branch coverage data
- `LH:` - Lines hit
- `LF:` - Lines found (total)
- `BRH:` - Branches hit
- `BRF:` - Branches found (total)
- `end_of_record` - End of file marker

---

## Behavioral Contracts

### Test Discovery

**Contract**: Files matching `*_test.sql` pattern are test files; all other `.sql` files are source files.

**Examples**:
- ✅ `auth_test.sql` → Test
- ✅ `user_functions_test.sql` → Test
- ✅ `auth.sql` → Source
- ❌ `test_auth.sql` → Source (wrong pattern)

### Test Isolation

**Contract**: Each test runs in a unique temporary database.

**Guarantees**:
- Test execution order does not affect results
- Tests can run in parallel without interference
- No database artifacts persist after test completion

### Coverage Accuracy

**Contract**: Same code and tests produce identical coverage results.

**Guarantees**:
- Deterministic hit counts
- Reproducible across runs
- No false positives (covered line must have executed)
- No false negatives (executed line must be marked covered)

### Error Reporting

**Contract**: All errors include actionable context.

**Guarantees**:
- Parse errors show file, line, column
- Connection errors suggest configuration fixes
- Test failures show SQL error code and message
- Timeout errors identify which test timed out

---

## Versioning

**Contract Version**: 1.0  
**Breaking Changes**: Require major version bump

Breaking changes include:
- CLI flag removals or renames
- Exit code changes
- Coverage data JSON schema changes (incompatible with previous parsers)
- LCOV format deviations

**Non-Breaking Changes**: Minor/patch version bumps

Non-breaking changes include:
- New CLI flags
- New output formats
- Additional fields in JSON schema
- Performance improvements

---

## Stability Guarantees

- **CLI Interface**: Stable after v1.0 (flag additions only)
- **Coverage Data Format**: Backward-compatible schema evolution
- **Exit Codes**: Fixed contract (no reassignment)
- **LCOV Format**: Strict adherence to specification

---

## Contract Tests

Implementation must pass these contract validation tests:

1. **CLI Help Output**: `pgcov help` returns exit code 0 and shows all commands
2. **Version Output**: `pgcov --version` shows version string
3. **Exit Code 0**: All passing tests return exit code 0
4. **Exit Code 1**: Any failing test returns exit code 1
5. **Coverage File**: `pgcov run` creates `.pgcov/coverage.json` with valid JSON
6. **LCOV Output**: `pgcov report --format=lcov` produces parseable LCOV format
7. **Test Pattern**: `*_test.sql` files discovered, others treated as source
8. **Parallel Execution**: `--parallel=N` respects concurrency limit
9. **Timeout Enforcement**: `--timeout=Xs` terminates test after X seconds
10. **Deterministic Coverage**: Multiple runs produce identical coverage percentages

---

## Summary

This contract defines:
- ✅ CLI commands and flags
- ✅ Exit codes and their meanings
- ✅ Output formats (text, JSON, LCOV)
- ✅ Coverage data file schema
- ✅ Behavioral guarantees
- ✅ Versioning policy

All implementations must comply with this contract for v1.0 compatibility.
