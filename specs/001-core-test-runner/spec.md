# Feature Specification: Core Test Runner and Coverage

**Feature Branch**: `001-core-test-runner`  
**Created**: 2026-01-05  
**Status**: Draft  
**Input**: User description: "Specify the functional and non-functional requirements of pgcov."

## Clarifications

### Session 2026-01-05

- Q: How should pgcov implement test isolation to prevent state leakage between tests? → A: Each test runs in a temporary database created/destroyed per test
- Q: Where should pgcov store coverage data collected during test execution? → A: Store coverage data in a local file (JSON, LCOV, or other preferred format)
- Q: Where should pgcov discover instrumentable source files for coverage tracking? → A: Search the same directory tree where test files are found (same root path provided to `pgcov run`)
- Q: When and how should pgcov apply instrumentation to SQL source code? → A: Instrument source files in-memory before deploying to the temporary test database
- Q: How should pgcov handle parallel test execution? → A: Support parallel execution via opt-in flag with configurable concurrency limit (e.g., `--parallel=N`)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Run Tests and See Coverage (Priority: P1)

A developer working on a PostgreSQL-backed application wants to verify that their database tests execute correctly and understand which parts of their SQL code are covered by tests. They have written test files following the `*_test.sql` naming convention and want to run them against a PostgreSQL instance.

**Why this priority**: This is the MVP—the core value proposition of pgcov. Without the ability to discover, run tests, and report coverage, the tool provides no value.

**Independent Test**: Can be fully tested by placing `*_test.sql` files in a directory, running `pgcov run [path]`, and verifying that tests execute and produce a coverage report showing which lines were executed.

**Acceptance Scenarios**:

1. **Given** a directory containing `user_test.sql` and `auth_test.sql` files, **When** the developer runs `pgcov run ./tests`, **Then** pgcov discovers both test files and executes them against the configured PostgreSQL instance
2. **Given** test files that pass, **When** pgcov finishes execution, **Then** the tool exits with status code 0 and displays a summary showing tests passed and coverage percentage
3. **Given** test files with SQL source code files in the project, **When** pgcov runs tests, **Then** it generates a coverage report showing which lines of source code were executed during tests
4. **Given** a test file that fails (assertion fails or SQL error), **When** pgcov runs the test, **Then** the tool exits with non-zero status code and displays the failure details

---

### User Story 2 - Isolated Test Execution (Priority: P2)

A developer wants each test to run in complete isolation, with its own setup and teardown, so tests can run in any order and in parallel without interfering with each other. Each test file should be self-contained.

**Why this priority**: Essential for reliable CI/CD integration and parallel execution. Without isolation, tests become flaky and order-dependent.

**Independent Test**: Can be tested by running multiple test files in different orders and verifying results are identical regardless of execution order.

**Acceptance Scenarios**:

1. **Given** two test files A and B that modify database state, **When** pgcov runs them in order A→B versus B→A, **Then** both runs produce identical test results
2. **Given** a test file that creates tables and data, **When** the test completes, **Then** the temporary database is destroyed and no artifacts remain
3. **Given** multiple test files, **When** pgcov runs tests, **Then** each test executes in its own temporary database with full isolation

---

### User Story 3 - Flexible Configuration (Priority: P3)

A developer wants to configure PostgreSQL connection details via command-line flags, environment variables, or a config file so they can easily run tests locally or in CI without hardcoding credentials.

**Why this priority**: Enables different environments (local, CI, staging) without code changes. Important for real-world usage but not blocking for MVP.

**Independent Test**: Can be tested by connecting to PostgreSQL using various configuration methods (flags, env vars) and verifying successful test execution.

**Acceptance Scenarios**:

1. **Given** environment variables `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE` are set, **When** the developer runs `pgcov run ./tests`, **Then** pgcov connects using those credentials
2. **Given** command-line flags `--host`, `--port`, `--user`, **When** the developer runs `pgcov run ./tests --host=localhost --port=5432`, **Then** command-line flags override environment variables
3. **Given** no configuration provided, **When** pgcov runs, **Then** it displays a clear error message explaining what connection details are required

---

### User Story 4 - Multiple Report Formats (Priority: P4)

A developer wants to export coverage reports in machine-readable formats (JSON, LCOV) so they can integrate with CI systems, code review tools, and coverage visualization platforms.

**Why this priority**: Enables integration with existing tooling ecosystem. Valuable but not essential for initial testing workflow.

**Independent Test**: Can be tested by running `pgcov report --format=json` and `pgcov report --format=lcov` and verifying output format correctness.

**Acceptance Scenarios**:

1. **Given** completed test execution with coverage data, **When** the developer runs `pgcov report --format=json`, **Then** pgcov outputs a JSON file with coverage data including file paths, line numbers, and execution counts
2. **Given** completed test execution, **When** the developer runs `pgcov report --format=lcov`, **Then** pgcov outputs an LCOV-formatted file compatible with standard coverage tools
3. **Given** no prior test execution, **When** the developer runs `pgcov report`, **Then** the tool displays an error indicating no coverage data is available

---

### Edge Cases

- What happens when a test file has syntax errors? Tool should report the specific file and line with error details.
- What happens when PostgreSQL is unreachable? Tool should display clear connection error with troubleshooting hints.
- What happens when no `*_test.sql` files are found? Tool should report "0 tests discovered" and exit with appropriate status.
- What happens when a test hangs indefinitely? Tool should enforce reasonable timeout (configurable, default 30 seconds per test).
- What happens when instrumentation cannot parse a SQL file? Tool should report which file failed parsing and provide error context.
- What happens when coverage data storage fails? Tool should still complete test execution but warn about coverage data loss.
- What happens when running tests on PostgreSQL 12 (below minimum 13+)? Tool should detect version and display clear error about minimum requirements.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST discover all files matching the `*_test.sql` pattern under a specified directory path (recursive search, similar to `go test ./...` or `go test ./foldername/`)
- **FR-002**: System MUST parse non-test SQL files (`.sql` extension, not matching `*_test.sql`) found in the same directory tree as instrumentable source code using PostgreSQL-compatible parser
- **FR-003**: System MUST instrument SQL and PL/pgSQL source code in-memory by rewriting the Abstract Syntax Tree (AST) to insert execution tracking hooks before deploying to the temporary test database
- **FR-004**: System MUST execute each discovered test file in a temporary database that is created before the test runs and destroyed after completion, ensuring complete isolation with no shared state between tests
- **FR-021**: System MUST create temporary databases with unique names to enable parallel test execution without conflicts
- **FR-023**: System MUST support optional parallel test execution via `--parallel=N` flag where N specifies maximum concurrent tests (default: sequential execution)
- **FR-005**: System MUST collect execution signals during test runs using PostgreSQL's LISTEN/NOTIFY mechanism to capture which instrumented locations were executed
- **FR-006**: System MUST aggregate execution counts per source location (file, line, branch) from collected signals and persist to a local coverage data file
- **FR-007**: System MUST generate coverage reports showing line coverage and branch coverage (where identifiable in SQL/PL/pgSQL)
- **FR-008**: System MUST exit with non-zero status code if any test fails (SQL error, assertion failure, or timeout)
- **FR-009**: System MUST exit with zero status code when all tests pass, regardless of coverage percentage
- **FR-010**: System MUST support PostgreSQL connection configuration via environment variables (`PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`)
- **FR-011**: System MUST support PostgreSQL connection configuration via command-line flags that override environment variables
- **FR-012**: System MUST provide `pgcov run [path]` command to discover and execute tests with coverage collection (supports Go-style path patterns like `./...` for recursive, `./foldername/` for specific directory)
- **FR-013**: System MUST provide `pgcov report [options]` command to read coverage data from the local file and output in specified format
- **FR-014**: System MUST provide `pgcov help` command showing usage documentation and available commands
- **FR-015**: System MUST support output formats: JSON (structured coverage data) and LCOV (for tool integration)
- **FR-016**: System MUST work with PostgreSQL 13 and later versions
- **FR-017**: System MUST support plain SQL and PL/pgSQL procedural language syntax
- **FR-018**: System MUST ensure instrumentation is transparent (does not change SQL semantics), performed in-memory, and produces no permanent artifacts (automatically removed when temporary database is destroyed)
- **FR-019**: System MUST enforce test timeout (default 30 seconds per test file, configurable via flag)
- **FR-020**: System MUST validate PostgreSQL version on connection and fail with clear error if version < 13
- **FR-022**: System MUST store coverage data in a default location (e.g., `.pgcov/coverage.json`) with option to specify custom path via flag

### Non-Functional Requirements

- **NFR-001**: System MUST run on Linux, macOS, and Windows operating systems
- **NFR-002**: System MUST be distributed as a single static binary with no runtime dependencies (except PostgreSQL server)
- **NFR-003**: System MUST produce deterministic coverage results (same code and tests produce identical coverage data across runs)
- **NFR-004**: System MUST execute medium-sized test suites (100 test files, 10,000 lines of SQL source) in reasonable time (under 5 minutes on standard hardware)
- **NFR-005**: System MUST provide clear, actionable error messages indicating file location, line number, and specific problem for all failure scenarios
- **NFR-006**: System MUST require minimal configuration to run (connection details only; no complex setup files or initialization steps required)
- **NFR-007**: System MUST maintain test isolation via temporary databases such that tests can run in any order with identical results and can execute in parallel (when enabled via `--parallel` flag) without interference
- **NFR-008**: System MUST not require PostgreSQL extensions, superuser privileges, or database-level modifications

### Key Entities

- **Test File**: SQL file matching `*_test.sql` pattern containing test logic (assertions, setup, teardown)
- **Source File**: SQL file containing instrumentable code (functions, procedures, views, triggers) that tests exercise
- **Coverage Data File**: Local file (default: `.pgcov/coverage.json`) storing aggregated execution counts per source location (file path, line number, optional branch identifier)
- **Test Result**: Pass/fail status, execution time, error details (if failed) for each test file
- **Instrumented Source**: In-memory version of source file with execution tracking hooks inserted via AST rewriting, deployed to temporary test database
- **Execution Signal**: LISTEN/NOTIFY message containing source location identifier emitted when instrumented code executes
- **Temporary Test Database**: Uniquely-named PostgreSQL database created for each test execution and destroyed after completion

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Developers can run pgcov on a project with 100 test files and 10,000 lines of SQL source code in under 5 minutes
- **SC-002**: Coverage reports accurately reflect code execution (zero false positives or false negatives for line coverage)
- **SC-003**: Running the same test suite multiple times produces identical coverage percentages (deterministic execution)
- **SC-004**: Test failures are detected immediately with clear error messages showing file, line number, and failure reason
- **SC-005**: Developers can integrate pgcov into CI pipelines without additional dependencies beyond PostgreSQL and the pgcov binary
- **SC-006**: Tool works on all three major platforms (Linux, macOS, Windows) with single binary distribution per platform
- **SC-007**: Test execution respects isolation (tests can run in any order with 100% identical results)
- **SC-008**: Coverage data can be exported to LCOV format and consumed by standard coverage visualization tools without errors
- **SC-009**: Developers require no configuration file to run tests (connection details via env vars or flags sufficient)
- **SC-010**: Tool provides helpful error guidance when PostgreSQL is unreachable, reducing support requests by 80%

## Assumptions

- PostgreSQL instance is accessible on network (local or remote) with provided credentials
- Developers follow naming convention `*_test.sql` for test files consistently
- SQL source files are valid PostgreSQL SQL or PL/pgSQL syntax
- Test files contain logic to verify behavior (assertions via SQL, pgTAP, or custom logic)
- Each test file is self-contained and responsible for its own setup/teardown
- Standard hardware for performance benchmarks: 4 CPU cores, 8GB RAM, SSD storage
- Medium-sized schema assumption: up to 10,000 lines of instrumentable SQL source code
- Default timeout of 30 seconds per test file is reasonable for most test workloads
