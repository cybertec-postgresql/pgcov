# Research: Core Test Runner and Coverage

**Feature**: Core Test Runner and Coverage  
**Branch**: 001-core-test-runner  
**Date**: 2026-01-05

## Overview

This document consolidates research findings and technical decisions for implementing pgcov's core functionality: test discovery, SQL instrumentation, test execution with isolation, and coverage reporting.

---

## 1. CLI Framework Selection

### Decision: `urfave/cli/v3`

### Rationale

- **Go-idiomatic**: Widely used in Go ecosystem (Docker, Terraform CLI components)
- **Feature-complete**: Supports subcommands, flags, env var binding, help generation
- **Maintained**: Active development, stable v3 API with improved ergonomics
- **Familiar patterns**: Similar to stdlib `flag` but more powerful

### Alternatives Considered

- **`spf13/cobra`**: More feature-rich but heavier; overkill for pgcov's simple command structure
- **stdlib `flag`**: Too basic; requires manual subcommand routing and help generation
- **`alecthomas/kong`**: Struct-tag based; less common pattern in database tooling
- **`urfave/cli/v2`**: Older version; v3 preferred for latest features and improvements

### Implementation Notes

```go
app := &cli.App{
    Name: "pgcov",
    Commands: []*cli.Command{
        {Name: "run", Action: runCommand},
        {Name: "report", Action: reportCommand},
    },
}
```

---

## 2. PostgreSQL SQL Parser

### Decision: `pganalyze/pg_query_go/v6`

### Rationale

- **Official PostgreSQL parser**: Based on `libpg_query`, wraps actual PostgreSQL parser code
- **AST access**: Provides full Abstract Syntax Tree for instrumentation
- **Version support**: Handles PostgreSQL 13-16 syntax with latest improvements
- **Battle-tested**: Used by pganalyze production systems

### Alternatives Considered

- **Custom parser**: Prohibitively complex; SQL grammar is massive and version-specific
- **`auxten/postgresql-parser`**: Less mature, incomplete AST coverage
- **Regex-based**: Insufficient for accurate instrumentation (comments, strings, nested blocks)
- **`pganalyze/pg_query_go/v5`**: Older version; v6 preferred for latest PostgreSQL support

### Implementation Notes

- Parsing produces protobuf-based AST
- Must handle parse errors gracefully (report file + line number)
- Protobuf messages require proto knowledge for traversal

```go
import pg_query "github.com/pganalyze/pg_query_go/v6"

tree, err := pg_query.Parse(sqlText)
// tree.Stmts contains parsed statements
```

---

## 3. Source File Discovery Strategy

### Decision: Co-located with test files (same directory only)

### Rationale

- **Explicit relationships**: Source files and test files in same directory make dependencies clear
- **Simple discovery**: No complex path resolution or recursive searching needed
- **Test isolation**: Each test directory is self-contained with its own sources
- **Go-like pattern**: Similar to `_test.go` files living alongside source `.go` files

### Discovery Rules

For each test file discovered at `path/to/dir/test_name_test.sql`:
1. Get parent directory: `path/to/dir/`
2. Find all `.sql` files in that directory (non-recursive)
3. Exclude files matching `*_test.sql` pattern
4. Remaining files are instrumentable sources for that test

### Example Structure

```
myproject/
├── auth/
│   ├── authenticate.sql       # Instrumented when auth_test.sql runs
│   ├── authorize.sql          # Instrumented when auth_test.sql runs
│   └── auth_test.sql          # Test file
├── users/
│   ├── user_crud.sql          # Instrumented when user_test.sql runs
│   └── user_test.sql          # Test file
└── payments/
    └── payment_test.sql       # No source files in this directory
```

Run `pgcov run ./...` discovers all three test files and instruments source files in each directory.

### Alternatives Considered

- **Recursive subdirectories**: Ambiguous which sources belong to which tests; complex dependency graph
- **Separate `--source-dir` flag**: Requires explicit configuration; breaks "minimal configuration" principle
- **Project root search**: Instruments all sources for every test (inefficient, confusing coverage attribution)

### Implementation Notes

```go
// For each discovered test file
testDir := filepath.Dir(testFile.Path)
entries, _ := os.ReadDir(testDir)

var sources []string
for _, entry := range entries {
    if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" {
        if !strings.HasSuffix(entry.Name(), "_test.sql") {
            sources = append(sources, filepath.Join(testDir, entry.Name()))
        }
    }
}
// sources now contains all instrumentable files for this test
```

---

## 4. PostgreSQL Driver

### Decision: `jackc/pgx/v5`

### Rationale

- **Pure Go**: No C dependencies; cross-platform compilation
- **Protocol-level control**: Supports LISTEN/NOTIFY, connection pooling, query cancellation
- **Performance**: Faster than `lib/pq` due to binary protocol usage
- **Maintained**: Active development, PostgreSQL 16 support

### Alternatives Considered

- **`lib/pq`**: Older, maintenance-mode; no binary protocol support
- **`go-pg/pg`**: ORM-focused; unnecessary abstraction for pgcov use case
- **database/sql + driver**: Loses LISTEN/NOTIFY support and fine-grained control

### Implementation Notes

```go
conn, err := pgx.Connect(ctx, connString)
defer conn.Close(ctx)

// LISTEN support
_, err = conn.Exec(ctx, "LISTEN coverage_signal")
```

---

## 5. AST Instrumentation Strategy

### Decision: In-memory AST rewriting with NOTIFY injection

### Rationale

- **Transparent**: No filesystem pollution
- **Deterministic**: Same source → same instrumented code
- **Removable**: Temporary databases destroy all artifacts automatically

### Instrumentation Technique

1. Parse SQL file with `pg_query_go`
2. Traverse AST to identify instrumentable locations:
   - Function entry points (PL/pgSQL functions)
   - Statement boundaries (SQL statements)
   - Branch points (IF/CASE/LOOP in PL/pgSQL)
3. Inject `PERFORM pg_notify('coverage_signal', 'file:line')` calls
4. Serialize instrumented AST back to SQL
5. Execute instrumented SQL in temporary database

### Challenges & Mitigations

- **Challenge**: AST modification requires deep protobuf knowledge
  - **Mitigation**: Focus on statement-level injection first; defer branch coverage to future iteration
  
- **Challenge**: Serializing modified AST back to SQL
  - **Mitigation**: Use `pg_query.Deparse()` if available; fallback to template-based generation

---

## 6. Test Isolation via Temporary Databases

### Decision: CREATE DATABASE per test, DROP DATABASE after completion

### Rationale

- **Strongest isolation**: No schema/data leakage between tests
- **Parallel-safe**: Unique database names prevent conflicts
- **Simple cleanup**: DROP DATABASE removes all objects atomically

### Naming Strategy

```
pgcov_test_{timestamp}_{random_suffix}
```

Example: `pgcov_test_20260105_a3f9c2b1`

### Connection Workflow

1. Connect to template database (`template1` or user-specified)
2. Execute `CREATE DATABASE pgcov_test_...`
3. Connect to new database
4. Load instrumented source files
5. Execute test file
6. Collect coverage signals
7. Disconnect
8. Connect to template database
9. Execute `DROP DATABASE pgcov_test_... WITH (FORCE)` (PostgreSQL 13+)

### Challenges & Mitigations

- **Challenge**: CREATE DATABASE requires elevated privileges on some systems
  - **Mitigation**: Document privilege requirements; provide schema-based isolation alternative for restricted environments (future)

---

## 7. Coverage Signal Collection

### Decision: LISTEN/NOTIFY for real-time signal aggregation

### Rationale

- **Built-in**: No extensions required
- **Real-time**: Signals arrive as code executes
- **Asynchronous**: Non-blocking for test execution

### Signal Format

```
channel: coverage_signal
payload: file:line[:branch]
```

Example: `src/auth.sql:42` or `src/auth.sql:42:branch_1`

### Collection Workflow

1. Before test execution: `LISTEN coverage_signal`
2. Instrumented code emits: `NOTIFY coverage_signal, 'src/auth.sql:42'`
3. Collector goroutine receives notifications asynchronously
4. Aggregate signals into coverage map: `map[string]map[int]int` (file → line → hitcount)
5. After test completion: `UNLISTEN coverage_signal`

### Challenges & Mitigations

- **Challenge**: NOTIFY payload size limit (8KB)
  - **Mitigation**: Use short file paths (relative to project root) and compact format

---

## 8. Coverage Data Persistence

### Decision: JSON file at `.pgcov/coverage.json`

### Rationale

- **Human-readable**: Easy debugging
- **Diff-friendly**: Git-compatible for tracking coverage changes
- **Extensible**: Schema can evolve with additional metadata

### File Schema

```json
{
  "version": "1.0",
  "timestamp": "2026-01-05T16:00:00Z",
  "files": {
    "src/auth.sql": {
      "lines": {
        "42": 5,
        "43": 5,
        "50": 0
      },
      "branches": {
        "44:1": 3,
        "44:2": 2
      }
    }
  }
}
```

### Alternatives Considered

- **SQLite**: Overkill; adds dependency and complexity
- **Binary format**: Loses human-readability for debugging
- **LCOV directly**: Not suitable as intermediate format (line-oriented, lacks structure)

---

## 9. Report Generation

### Decision: Pluggable reporters (JSON, LCOV initially)

### Reporter Interface

```go
type Reporter interface {
    Generate(coverage *Coverage) ([]byte, error)
}

type JSONReporter struct{}
type LCOVReporter struct{}
```

### LCOV Format

```
TN:
SF:src/auth.sql
DA:42,5
DA:43,5
DA:50,0
BRDA:44,0,1,3
BRDA:44,0,2,2
end_of_record
```

### Extensibility

Future reporters can be added without modifying core:
- HTML reporter (with syntax highlighting)
- Cobertura XML (for Jenkins/GitLab)
- Terminal pretty-print reporter

---

## 10. Parallel Test Execution

### Decision: Opt-in via `--parallel=N` flag using goroutine pool

### Rationale

- **Safe default**: Sequential execution avoids overwhelming PostgreSQL
- **Controlled concurrency**: User specifies max parallel tests
- **Familiar pattern**: Matches `go test -parallel N`

### Implementation Strategy

```go
// Worker pool pattern
semaphore := make(chan struct{}, parallelism)
for _, test := range tests {
    semaphore <- struct{}{} // Acquire
    go func(t Test) {
        defer func() { <-semaphore }() // Release
        runTest(t)
    }(test)
}
```

### Challenges & Mitigations

- **Challenge**: PostgreSQL connection limit
  - **Mitigation**: Connection pooling; N parallel tests ≤ connection limit

- **Challenge**: Coverage signal aggregation race conditions
  - **Mitigation**: Use `sync.Mutex` or channels for thread-safe aggregation

---

## 11. Error Handling and Diagnostics

### Decision: Structured errors with context

### Error Strategy

- **Parse errors**: Report file, line, column, snippet
- **Connection errors**: Suggest configuration fixes (env vars, connection string format)
- **Test failures**: Show SQL error code, message, query context
- **Timeout errors**: Indicate which test timed out and suggest timeout flag adjustment

### Example Error Output

```
Error: Failed to parse source file
  File: src/auth.sql
  Line: 42
  Column: 15
  Error: syntax error at or near "SLECT"
  
  40 | CREATE FUNCTION authenticate(user_id INT) RETURNS BOOLEAN AS $$
  41 | BEGIN
> 42 |   SLECT COUNT(*) INTO valid_user FROM users WHERE id = user_id;
     |   ^^^^^
  43 |   RETURN valid_user > 0;
  44 | END;
```

### Logging

- **Info**: Test discovery, test execution progress
- **Debug**: SQL queries, coverage signals (via `--verbose` flag)
- **Error**: Failures with full context

---

## 12. Configuration Management

### Decision: Layered configuration (env vars → flags → config file)

### Priority (highest to lowest)

1. Command-line flags (`--host`, `--port`, etc.)
2. Environment variables (`PGHOST`, `PGPORT`, etc.)
3. Config file (`.pgcov.yaml` - future)

### PostgreSQL Connection

Standard environment variables:
- `PGHOST` (default: `localhost`)
- `PGPORT` (default: `5432`)
- `PGUSER` (default: current user)
- `PGPASSWORD` (default: empty)
- `PGDATABASE` (default: `postgres` - used as template)

### Coverage Configuration

- `--coverage-file`: Coverage data output path (default: `.pgcov/coverage.json`)
- `--timeout`: Per-test timeout (default: `30s`)
- `--parallel`: Parallel test count (default: `1` - sequential)

---

## 13. Extensibility Points

### Plugin Architecture (Future)

1. **Custom Reporters**: Implement `Reporter` interface
2. **Custom Parsers**: Support additional languages (e.g., Python procedural language)
3. **Custom Isolation Strategies**: Schema-based isolation for restricted environments

### Current Hooks

- Reporter registry: Add new formats without modifying core
- Instrumentation visitors: Extend AST traversal for new coverage metrics

---

## Summary

All technical decisions are resolved. No blocking unknowns remain. Ready to proceed to Phase 1 (design artifacts).

**Key Technologies**:
- CLI: `urfave/cli/v2`
- SQL Parser: `pganalyze/pg_query_go/v5`
- PostgreSQL Driver: `jackc/pgx/v5`

**Key Patterns**:
- In-memory AST instrumentation
- Temporary database per test
- LISTEN/NOTIFY for coverage signals
- Goroutine pool for parallelism
- Pluggable reporters

**Next Phase**: Data model definition and quickstart documentation.

