# Data Model: Core Test Runner and Coverage

**Feature**: Core Test Runner and Coverage  
**Branch**: 001-core-test-runner  
**Date**: 2026-01-05

## Overview

This document defines the core data structures used throughout pgcov for representing discovered files, coverage data, test results, and configuration.

---

## 1. File Discovery

### DiscoveredFile

Represents a SQL file discovered during filesystem traversal.

```go
type DiscoveredFile struct {
    Path         string    // Absolute path to file
    RelativePath string    // Path relative to search root
    Type         FileType  // Test or Source
    ModTime      time.Time // Last modification time
}

type FileType int

const (
    FileTypeTest   FileType = iota // Matches *_test.sql
    FileTypeSource                  // Does not match *_test.sql
)
```

**Validation Rules**:
- `Path` must be absolute
- `RelativePath` must be relative to project root
- `Type` determined by `*_test.sql` pattern match

**Relationships**:
- Test files reference source files by relative path in coverage reports

---

## 2. SQL Parsing

### ParsedSQL

Represents a successfully parsed SQL file with AST.

```go
type ParsedSQL struct {
    File       *DiscoveredFile
    AST        *pg_query.ParseResult // From pg_query_go
    Statements []*Statement
}

type Statement struct {
    RawSQL    string // Original SQL text
    StartLine int    // 1-indexed line number
    EndLine   int
    Type      StatementType
}

type StatementType int

const (
    StmtUnknown StatementType = iota
    StmtFunction              // CREATE FUNCTION
    StmtProcedure             // CREATE PROCEDURE
    StmtTrigger               // CREATE TRIGGER
    StmtView                  // CREATE VIEW
    StmtOther                 // Any other statement
)
```

**Validation Rules**:
- `AST` must be non-nil for successful parse
- `StartLine` and `EndLine` must be positive integers
- `StartLine` ≤ `EndLine`

---

## 3. Instrumentation

### InstrumentedSQL

Represents SQL code that has been instrumented for coverage tracking.

```go
type InstrumentedSQL struct {
    Original         *ParsedSQL
    InstrumentedText string          // Rewritten SQL with NOTIFY calls
    Locations        []CoveragePoint // All instrumented locations
}

type CoveragePoint struct {
    File     string // Relative file path
    Line     int    // Line number (1-indexed)
    Branch   string // Branch identifier (optional, e.g., "if_true", "if_false")
    SignalID string // Unique signal identifier sent via NOTIFY
}
```

**Validation Rules**:
- `SignalID` must be unique across all coverage points
- `Line` must be within bounds of original file
- `Branch` empty for statement coverage, non-empty for branch coverage

**Signal ID Format**: `{file}:{line}` or `{file}:{line}:{branch}`

Example: `src/auth.sql:42` or `src/auth.sql:44:if_true`

---

## 4. Test Execution

### TestRun

Represents a single test execution.

```go
type TestRun struct {
    Test        *DiscoveredFile
    Database    *TempDatabase
    StartTime   time.Time
    EndTime     time.Time
    Status      TestStatus
    Error       error          // Non-nil if test failed
    CoverageSigs []CoverageSignal // Signals collected during test
}

type TestStatus int

const (
    TestPending TestStatus = iota
    TestRunning
    TestPassed
    TestFailed
    TestTimeout
)

type TempDatabase struct {
    Name         string // e.g., "pgcov_test_20260105_a3f9c2b1"
    CreatedAt    time.Time
    ConnectionString string
}
```

**Validation Rules**:
- `Database.Name` must match pattern `pgcov_test_{timestamp}_{random}`
- `EndTime` ≥ `StartTime`
- `Status` must be `TestFailed` or `TestTimeout` if `Error` is non-nil

**State Transitions**:
```
TestPending → TestRunning → {TestPassed | TestFailed | TestTimeout}
```

---

## 5. Coverage Signals

### CoverageSignal

Represents a single coverage signal emitted via NOTIFY.

```go
type CoverageSignal struct {
    SignalID  string    // Matches CoveragePoint.SignalID
    Timestamp time.Time // When signal received
    TestRun   *TestRun  // Associated test
}
```

**Parsing**:
```go
// From NOTIFY payload: "src/auth.sql:42"
func ParseSignal(payload string) (*CoverageSignal, error) {
    parts := strings.Split(payload, ":")
    // Validate format and extract file, line, optional branch
}
```

---

## 6. Coverage Aggregation

### Coverage

Aggregated coverage data across all tests.

```go
type Coverage struct {
    Version   string                 // Schema version (e.g., "1.0")
    Timestamp time.Time              // When coverage collected
    Files     map[string]*FileCoverage
}

type FileCoverage struct {
    Path     string              // Relative file path
    Lines    map[int]*LineCoverage
    Branches map[string]*BranchCoverage
}

type LineCoverage struct {
    LineNumber int
    HitCount   int    // Number of times line executed
    Covered    bool   // true if HitCount > 0
}

type BranchCoverage struct {
    BranchID string // e.g., "44:if_true"
    HitCount int
    Covered  bool
}
```

**Validation Rules**:
- `HitCount` must be non-negative
- `Covered` must be `true` iff `HitCount > 0`
- All line numbers must be positive integers

**Aggregation Logic**:
```go
func (c *Coverage) AddSignal(sig *CoverageSignal) {
    file, line, branch := parseSignalID(sig.SignalID)
    if branch == "" {
        c.Files[file].Lines[line].HitCount++
    } else {
        c.Files[file].Branches[branch].HitCount++
    }
}
```

---

## 7. Configuration

### Config

Runtime configuration combining flags, environment variables, and defaults.

```go
type Config struct {
    // PostgreSQL connection
    PGHost     string
    PGPort     int
    PGUser     string
    PGPassword string
    PGDatabase string // Template database for creating temp DBs
    
    // Execution
    SearchPath  string        // Root path for test/source discovery
    Timeout     time.Duration // Per-test timeout
    Parallelism int           // Max concurrent tests (1 = sequential)
    
    // Output
    CoverageFile string // Coverage data output path
    Verbose      bool   // Enable debug logging
}
```

**Defaults**:
```go
var DefaultConfig = Config{
    PGHost:       "localhost",
    PGPort:       5432,
    PGDatabase:   "postgres",
    Timeout:      30 * time.Second,
    Parallelism:  1,
    CoverageFile: ".pgcov/coverage.json",
    Verbose:      false,
}
```

**Loading Priority** (highest to lowest):
1. Command-line flags
2. Environment variables (`PGHOST`, `PGPORT`, etc.)
3. Defaults

---

## 8. Test Results Summary

### TestSummary

Summary of all test executions.

```go
type TestSummary struct {
    TotalTests   int
    PassedTests  int
    FailedTests  int
    TimedOutTests int
    TotalDuration time.Duration
    Coverage      *Coverage
}

func (s *TestSummary) AllPassed() bool {
    return s.FailedTests == 0 && s.TimedOutTests == 0
}

func (s *TestSummary) ExitCode() int {
    if s.AllPassed() {
        return 0
    }
    return 1
}
```

---

## 9. Reporter Output

### ReportFormat

Enumeration of supported report formats.

```go
type ReportFormat int

const (
    FormatJSON ReportFormat = iota
    FormatLCOV
)

func (f ReportFormat) Extension() string {
    switch f {
    case FormatJSON:
        return ".json"
    case FormatLCOV:
        return ".lcov"
    default:
        return ""
    }
}
```

---

## 10. Error Types

### Custom Error Types

```go
// ParseError represents SQL parsing failure
type ParseError struct {
    File   string
    Line   int
    Column int
    Message string
}

func (e *ParseError) Error() string {
    return fmt.Sprintf("%s:%d:%d: %s", e.File, e.Line, e.Column, e.Message)
}

// ConnectionError represents PostgreSQL connection failure
type ConnectionError struct {
    Host    string
    Port    int
    Message string
}

func (e *ConnectionError) Error() string {
    return fmt.Sprintf("failed to connect to %s:%d: %s", e.Host, e.Port, e.Message)
}

// TestFailureError represents test execution failure
type TestFailureError struct {
    Test     string
    SQLError *pgconn.PgError // PostgreSQL error details
}

func (e *TestFailureError) Error() string {
    return fmt.Sprintf("test %s failed: [%s] %s", e.Test, e.SQLError.Code, e.SQLError.Message)
}
```

---

## Entity Relationship Diagram

```
DiscoveredFile (1) --> (*) ParsedSQL
    |
    v
ParsedSQL (1) --> (1) InstrumentedSQL
    |
    v
InstrumentedSQL (1) --> (*) CoveragePoint
    |
    v
CoveragePoint (1) <-- (*) CoverageSignal
    |
    v
CoverageSignal (*) --> (1) TestRun
    |
    v
TestRun (*) --> (1) TestSummary
    |
    v
TestSummary (1) --> (1) Coverage
    |
    v
Coverage (1) --> (*) FileCoverage
```

---

## Data Flow

1. **Discovery**: `DiscoveredFile` list built from filesystem
2. **Parsing**: Each source file → `ParsedSQL`
3. **Instrumentation**: `ParsedSQL` → `InstrumentedSQL` with `CoveragePoint` list
4. **Execution**: Each test → `TestRun`, emits `CoverageSignal` instances
5. **Aggregation**: All `CoverageSignal` → `Coverage` (aggregated by file/line)
6. **Summary**: All `TestRun` → `TestSummary` with embedded `Coverage`
7. **Reporting**: `Coverage` → JSON/LCOV output

---

## Persistence

### Coverage Data File Format (JSON)

```json
{
  "version": "1.0",
  "timestamp": "2026-01-05T16:00:00Z",
  "files": {
    "src/auth.sql": {
      "path": "src/auth.sql",
      "lines": {
        "42": {"line_number": 42, "hit_count": 5, "covered": true},
        "43": {"line_number": 43, "hit_count": 5, "covered": true},
        "50": {"line_number": 50, "hit_count": 0, "covered": false}
      },
      "branches": {
        "44:if_true": {"branch_id": "44:if_true", "hit_count": 3, "covered": true},
        "44:if_false": {"branch_id": "44:if_false", "hit_count": 2, "covered": true}
      }
    }
  }
}
```

**Schema Versioning**: `version` field enables future schema evolution.

---

## Summary

All core entities defined with validation rules, relationships, and state transitions. Data model supports:

- File discovery and classification
- SQL parsing and AST access
- Instrumentation with coverage point tracking
- Test execution with isolation
- Coverage signal collection
- Aggregated coverage reporting
- Pluggable output formats

**Next**: Generate contracts and quickstart documentation.
