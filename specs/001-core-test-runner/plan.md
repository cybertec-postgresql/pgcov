# Implementation Plan: Core Test Runner and Coverage

**Branch**: `001-core-test-runner` | **Date**: 2026-01-05 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-core-test-runner/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

pgcov is a PostgreSQL test runner and coverage tool that discovers `*_test.sql` files, instruments SQL/PL/pgSQL source code for coverage tracking, executes tests in isolated temporary databases, and generates coverage reports in JSON and LCOV formats. The tool provides a Go-like CLI experience (`pgcov run`, `pgcov report`) with direct PostgreSQL protocol access via pgx, SQL parsing via pg_query_go, and optional parallel test execution.

## Technical Context

**Language/Version**: Go 1.21+  
**Primary Dependencies**: 
- `github.com/urfave/cli/v3` (CLI framework)
- `github.com/pganalyze/pg_query_go/v6` (PostgreSQL SQL parser)
- `github.com/jackc/pgx/v5` (PostgreSQL driver and protocol)

**Storage**: 
- Coverage data file: `.pgcov/coverage.json` (default, configurable)
- Temporary PostgreSQL databases (created/destroyed per test)

**Testing**: 
- Go standard testing (`go test`)
- Integration tests against PostgreSQL 13, 14, 15, 16

**Target Platform**: Linux, macOS, Windows (single static binary per platform)

**Project Type**: Single CLI application

**Performance Goals**: 
- 100 test files in <5 minutes on standard hardware
- Deterministic coverage (identical results across runs)
- Minimal overhead for instrumentation (<10% execution time increase)

**Constraints**: 
- No PostgreSQL extensions required
- No superuser privileges required
- Must work with vanilla PostgreSQL 13+
- Single static binary distribution

**Scale/Scope**: 
- Support projects with up to 10,000 lines of SQL source code
- Handle up to 100 concurrent test files (with `--parallel` flag)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Initial Check (Pre-Research)

| Principle | Status | Compliance Notes |
|-----------|--------|------------------|
| **I. Direct Protocol Access** | ✅ Pass | Using `pgx` for direct PostgreSQL protocol access; no psql/shell dependencies |
| **II. Test Isolation** | ✅ Pass | Temporary database per test ensures complete isolation; tests are order-independent |
| **III. Instrumentation Transparency** | ✅ Pass | In-memory AST rewriting via `pg_query_go`; no extensions, no permanent artifacts |
| **IV. CLI-First Design** | ✅ Pass | CLI via `urfave/cli`; supports flags, env vars; Go-like commands (`pgcov run`, `pgcov report`) |
| **V. Coverage Accuracy Over Speed** | ✅ Pass | Deterministic execution; LISTEN/NOTIFY for accurate signal collection; reproducible results |
| **VI. Go Development Ergonomics** | ✅ Pass | Command patterns mirror `go test`; LCOV/JSON output; idiomatic Go error handling |

**Verdict**: ✅ All constitutional principles satisfied. No violations. Proceed to Phase 0.

### Post-Design Check (After Phase 1)

| Principle | Status | Validation |
|-----------|--------|------------|
| **I. Direct Protocol Access** | ✅ Pass | `pgx` connection pool; no shell commands; `database/pool.go` handles direct protocol access |
| **II. Test Isolation** | ✅ Pass | `database/tempdb.go` creates unique database per test; `runner/isolation.go` ensures no state leakage |
| **III. Instrumentation Transparency** | ✅ Pass | `instrument/instrumenter.go` performs in-memory AST rewriting; temporary database cleanup removes all artifacts |
| **IV. CLI-First Design** | ✅ Pass | `cli/run.go` and `cli/report.go` provide Go-style commands; `cli/config.go` handles flags/env vars; exit codes match contract |
| **V. Coverage Accuracy Over Speed** | ✅ Pass | `coverage/collector.go` aggregates signals deterministically; `coverage/store.go` ensures reproducible persistence |
| **VI. Go Development Ergonomics** | ✅ Pass | `internal/` package structure; standard Go idioms; error types defined in data-model.md; LCOV/JSON reporters |

**Final Verdict**: ✅ All principles satisfied in final design. No architectural violations. Ready for task generation.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
pgcov/
├── cmd/
│   └── pgcov/
│       └── main.go           # CLI entry point
├── internal/
│   ├── cli/
│   │   ├── run.go            # 'pgcov run' command
│   │   ├── report.go         # 'pgcov report' command
│   │   └── config.go         # Configuration management
│   ├── discovery/
│   │   ├── discover.go       # File discovery (tests + sources)
│   │   └── classifier.go     # SQL file classification
│   ├── parser/
│   │   ├── parse.go          # pg_query_go wrapper
│   │   └── ast.go            # AST utilities
│   ├── instrument/
│   │   ├── instrumenter.go   # AST rewriting for coverage
│   │   ├── injector.go       # NOTIFY injection logic
│   │   └── location.go       # Source location tracking
│   ├── database/
│   │   ├── pool.go           # PostgreSQL connection pool
│   │   ├── tempdb.go         # Temporary database lifecycle
│   │   └── listener.go       # LISTEN/NOTIFY handler
│   ├── runner/
│   │   ├── executor.go       # Test execution orchestration
│   │   ├── isolation.go      # Per-test isolation logic
│   │   └── parallel.go       # Parallel execution coordinator
│   ├── coverage/
│   │   ├── collector.go      # Signal aggregation
│   │   ├── model.go          # Coverage data structures
│   │   └── store.go          # File persistence
│   └── report/
│       ├── json.go           # JSON reporter
│       ├── lcov.go           # LCOV reporter
│       └── formatter.go      # Output formatting utilities
├── pkg/
│   └── types/                # Public types (if needed for extensions)
├── testdata/                 # Test fixtures
│   ├── simple/
│   ├── plpgsql/
│   └── edge_cases/
└── go.mod
```

**Structure Decision**: Single Go project using standard layout. `internal/` prevents external imports; `cmd/` for executable; `pkg/` for potential library exports; `testdata/` for integration test fixtures.

## Complexity Tracking

> **No violations detected.** All constitutional principles are satisfied by the current design.

---

## Phase 0: Research - ✅ COMPLETE

**Objective**: Resolve all technical unknowns and document technology choices.

**Deliverable**: [research.md](./research.md)

**Key Decisions**:
1. CLI Framework: `urfave/cli/v3`
2. SQL Parser: `pganalyze/pg_query_go/v6`
3. PostgreSQL Driver: `jackc/pgx/v5`
4. Source File Discovery: Co-located with tests (same directory, not recursive)
5. AST Instrumentation: In-memory rewriting with NOTIFY injection
6. Test Isolation: Temporary database per test (CREATE/DROP)
7. Coverage Signals: LISTEN/NOTIFY mechanism
8. Coverage Persistence: JSON file at `.pgcov/coverage.json`
9. Parallel Execution: Opt-in via `--parallel=N` flag with goroutine pool
10. Error Handling: Structured errors with file/line context
11. Configuration: Layered (flags > env vars > defaults)

**Status**: All unknowns resolved. No blocking issues.

---

## Phase 1: Design - ✅ COMPLETE

**Objective**: Define data model, API contracts, and quickstart guide.

**Deliverables**:
- [data-model.md](./data-model.md) - Core data structures and relationships
- [contracts/cli-contract.md](./contracts/cli-contract.md) - CLI interface specification
- [quickstart.md](./quickstart.md) - User guide and examples

**Artifacts Generated**:
1. **Data Model**: 10 core entities (DiscoveredFile, ParsedSQL, InstrumentedSQL, TestRun, Coverage, etc.)
2. **CLI Contract**: Commands (run, report, help), flags, exit codes, output formats
3. **Coverage Schema**: JSON format with versioning (v1.0)
4. **LCOV Contract**: Standard trace file format
5. **Quickstart Guide**: Installation, configuration, examples, CI/CD integration

**Status**: All design artifacts complete. Constitution check passed.

---

## Phase 2: Task Generation - NEXT STEP

**Command**: `/speckit.tasks`

**What it will generate**:
- 	asks.md with implementation tasks organized by user story
- Setup phase (project init, dependencies)
- Foundational phase (core infrastructure)
- User Story 1 (P1): Run Tests and See Coverage
- User Story 2 (P2): Isolated Test Execution
- User Story 3 (P3): Flexible Configuration
- User Story 4 (P4): Multiple Report Formats

**Blocking items**: None - ready for task generation.

---

## Implementation Overview

### Major Components

1. **CLI Layer** (`internal/cli/`)
   - Command routing and flag parsing
   - Configuration management
   - User-facing output formatting

2. **Discovery Layer** (`internal/discovery/`)
   - Filesystem traversal (Go-style patterns: `./...`)
   - File classification (`*_test.sql` vs source)

3. **Parser Layer** (`internal/parser/`)
   - `pg_query_go` wrapper
   - AST extraction and validation

4. **Instrumentation Layer** (`internal/instrument/`)
   - AST traversal and rewriting
   - NOTIFY injection at statement boundaries
   - Source location tracking

5. **Database Layer** (`internal/database/`)
   - `pgx` connection pooling
   - Temporary database lifecycle (CREATE/DROP)
   - LISTEN/NOTIFY handler

6. **Runner Layer** (`internal/runner/`)
   - Test execution orchestration
   - Isolation management (per-test database)
   - Parallel execution coordinator (goroutine pool)

7. **Coverage Layer** (`internal/coverage/`)
   - Signal aggregation (file → line → hitcount)
   - Coverage data structure
   - File persistence (JSON)

8. **Reporter Layer** (`internal/report/`)
   - JSON reporter
   - LCOV reporter
   - Output formatting utilities

### Control Flow

\\\
CLI (run command)
  ↓
Discovery (find *_test.sql + *.sql)
  ↓
Parser (parse source files)
  ↓
Instrumenter (rewrite AST + inject NOTIFY)
  ↓
Runner (for each test):
    Database (CREATE temp DB)
    Database (LISTEN coverage_signal)
    Database (load instrumented source)
    Database (execute test)
    Coverage (collect NOTIFY signals)
    Database (DROP temp DB)
  ↓
Coverage (aggregate all signals)
  ↓
Coverage (write .pgcov/coverage.json)
  ↓
CLI (display summary + exit code)
\\\

### Key Data Structures

- **DiscoveredFile**: Represents a SQL file (test or source)
- **ParsedSQL**: SQL file + AST + statements
- **InstrumentedSQL**: Rewritten SQL + coverage points
- **TestRun**: Test execution state + coverage signals
- **Coverage**: Aggregated coverage data (file → line → hitcount)
- **Config**: Runtime configuration (flags + env vars)

### Assumptions

1. PostgreSQL 13+ is running and accessible
2. User has CREATEDB privilege for test isolation
3. Test files are self-contained (setup + assertions + teardown)
4. Source files are valid PostgreSQL SQL/PL/pgSQL syntax
5. LISTEN/NOTIFY payloads fit within 8KB limit (reasonable for source location identifiers)
6. Standard hardware (4 CPU cores, 8GB RAM) for performance benchmarks

### Trade-offs

1. **In-memory instrumentation vs file-based**:
   - **Chosen**: In-memory (no filesystem pollution, automatic cleanup)
   - **Trade-off**: Higher memory usage for large source files (acceptable for 10k LOC target)

2. **Temporary database vs schema-based isolation**:
   - **Chosen**: Temporary database (strongest isolation, parallel-safe)
   - **Trade-off**: Requires CREATEDB privilege (documented as requirement)

3. **Sequential vs parallel by default**:
   - **Chosen**: Sequential default, opt-in parallel
   - **Trade-off**: Slower for large test suites (user controls via `--parallel`)

4. **JSON vs binary coverage data format**:
   - **Chosen**: JSON (human-readable, git-friendly)
   - **Trade-off**: Larger file size (acceptable for target scale)

---

## Summary

**Status**: ✅ Planning complete. Ready for implementation.

**Branch**: `001-core-test-runner`

**Artifacts Generated**:
- ✅ `plan.md` (this file)
- ✅ `research.md` (technology decisions)
- ✅ `data-model.md` (core entities)
- ✅ `contracts/cli-contract.md` (CLI specification)
- ✅ `quickstart.md` (user guide)
- ✅ `spec.md` (requirements, from /speckit.specify)
- ✅ `.github/agents/copilot-instructions.md` (updated)

**Constitutional Compliance**: ✅ All 6 principles satisfied

**Next Command**: `/speckit.tasks` to generate implementation tasks organized by user story.

**Estimated Scope**:
- ~15-20 Go packages
- ~2500-3000 lines of implementation code
- ~1500-2000 lines of test code
- 4 user stories (P1-P4)
- Integration tests against PostgreSQL 13-16
