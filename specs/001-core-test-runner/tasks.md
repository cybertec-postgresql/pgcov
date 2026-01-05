---

description: "Task list for Core Test Runner and Coverage implementation"
---

# Tasks: Core Test Runner and Coverage

**Input**: Design documents from `/specs/001-core-test-runner/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli-contract.md

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

Single Go project: `internal/`, `cmd/`, `testdata/` at repository root

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Initialize Go module with go.mod at repository root
- [X] T002 Create project directory structure (cmd/pgcov/, internal/, pkg/types/, testdata/)
- [X] T003 Add dependencies: github.com/urfave/cli/v3, github.com/pganalyze/pg_query_go/v6, github.com/jackc/pgx/v5
- [X] T004 [P] Create .gitignore file (ignore .pgcov/, binaries, IDE files)
- [X] T005 [P] Create README.md with project overview and quick start

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T006 Create Config struct in internal/cli/config.go (PostgreSQL connection, execution, output settings)
- [X] T007 Implement configuration loader in internal/cli/config.go (layered: flags → env vars → defaults)
- [X] T008 Create DiscoveredFile and FileType types in internal/discovery/types.go
- [X] T009 Create ParsedSQL and Statement types in internal/parser/types.go
- [X] T010 Create InstrumentedSQL and CoveragePoint types in internal/instrument/types.go
- [X] T011 Create Coverage, FileCoverage, LineCoverage, BranchCoverage types in internal/coverage/model.go
- [X] T012 Create TestRun, TestStatus, TempDatabase types in internal/runner/types.go
- [X] T013 Create custom error types (ParseError, ConnectionError, TestFailureError) in internal/errors/errors.go
- [X] T014 [P] Create CLI app skeleton in cmd/pgcov/main.go using urfave/cli/v3
- [X] T015 [P] Implement version and help commands in cmd/pgcov/main.go

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Run Tests and See Coverage (Priority: P1) 🎯 MVP

**Goal**: Discover test files, execute them against PostgreSQL, instrument source code, collect coverage signals, and generate coverage report

**Independent Test**: Place `*_test.sql` files in a directory, run `pgcov run [path]`, verify tests execute and produce coverage report showing line execution

### Implementation for User Story 1

**Discovery Layer**

- [ ] T016 [P] [US1] Implement filesystem discovery in internal/discovery/discover.go (find *_test.sql files)
- [ ] T017 [P] [US1] Implement file classifier in internal/discovery/classifier.go (classify test vs source files)
- [ ] T018 [US1] Implement co-location strategy in internal/discovery/discover.go (find source files in same directory as tests)

**Parser Layer**

- [ ] T019 [P] [US1] Create pg_query_go wrapper in internal/parser/parse.go (parse SQL files into AST)
- [ ] T020 [P] [US1] Implement AST utilities in internal/parser/ast.go (extract statements, line numbers, statement types)
- [ ] T021 [US1] Add parse error handling in internal/parser/parse.go (return ParseError with file/line/column context)

**Instrumentation Layer**

- [ ] T022 [P] [US1] Implement AST instrumenter in internal/instrument/instrumenter.go (inject NOTIFY calls for statement coverage)
- [ ] T023 [P] [US1] Implement coverage point tracker in internal/instrument/location.go (track file:line locations)
- [ ] T024 [US1] Implement signal ID generator in internal/instrument/injector.go (format: file:line)
- [ ] T025 [US1] Add instrumentation error handling in internal/instrument/instrumenter.go (handle AST rewrite failures)

**Database Layer**

- [ ] T026 [P] [US1] Implement PostgreSQL connection pool in internal/database/pool.go using pgx/v5
- [ ] T027 [P] [US1] Implement temporary database creator in internal/database/tempdb.go (CREATE DATABASE with unique name)
- [ ] T028 [P] [US1] Implement temporary database destroyer in internal/database/tempdb.go (DROP DATABASE WITH FORCE)
- [ ] T029 [P] [US1] Implement LISTEN/NOTIFY handler in internal/database/listener.go (receive coverage signals)
- [ ] T030 [US1] Add PostgreSQL version check in internal/database/pool.go (validate PostgreSQL 13+)
- [ ] T031 [US1] Add connection error handling in internal/database/pool.go (return ConnectionError with suggestions)

**Runner Layer**

- [ ] T032 [P] [US1] Implement test executor in internal/runner/executor.go (orchestrate test execution)
- [ ] T033 [P] [US1] Implement per-test workflow in internal/runner/executor.go (create temp DB, load instrumented code, run test, collect signals, destroy DB)
- [ ] T034 [US1] Add test timeout enforcement in internal/runner/executor.go (context.WithTimeout)
- [ ] T035 [US1] Add test failure detection in internal/runner/executor.go (capture SQL errors, set TestStatus)

**Coverage Layer**

- [ ] T036 [P] [US1] Implement coverage signal collector in internal/coverage/collector.go (aggregate signals from LISTEN/NOTIFY)
- [ ] T037 [P] [US1] Implement coverage aggregation logic in internal/coverage/collector.go (map file:line to hit counts)
- [ ] T038 [P] [US1] Implement coverage data persistence in internal/coverage/store.go (write JSON to .pgcov/coverage.json)
- [ ] T039 [US1] Add coverage calculation in internal/coverage/model.go (compute line coverage percentages)

**Reporter Layer**

- [ ] T040 [P] [US1] Implement JSON reporter in internal/report/json.go (format coverage data as JSON)
- [ ] T041 [P] [US1] Implement LCOV reporter in internal/report/lcov.go (format coverage data as LCOV)
- [ ] T042 [US1] Implement reporter interface in internal/report/formatter.go (pluggable format selection)

**CLI Integration**

- [ ] T043 [US1] Implement `pgcov run` command in internal/cli/run.go (wire all layers together)
- [ ] T044 [US1] Implement `pgcov report` command in internal/cli/report.go (read coverage file and output report)
- [ ] T045 [US1] Add CLI output formatting in internal/cli/run.go (test summary with pass/fail counts and coverage percentage)
- [ ] T046 [US1] Add exit code logic in internal/cli/run.go (0 for all passed, 1 for failures)

**Testing & Validation**

- [ ] T047 [US1] Create test fixtures in testdata/simple/ (basic SQL test and source files)
- [ ] T048 [US1] Create integration test in internal/integration_test.go (end-to-end test with PostgreSQL)
- [ ] T049 [US1] Verify JSON coverage output schema matches contract in tests
- [ ] T050 [US1] Verify LCOV output format matches specification in tests

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Isolated Test Execution (Priority: P2)

**Goal**: Ensure each test runs in complete isolation with temporary database per test, enabling order-independent and parallel execution

**Independent Test**: Run multiple test files in different orders and verify identical results regardless of execution order

### Implementation for User Story 2

- [ ] T051 [P] [US2] Implement unique database naming in internal/database/tempdb.go (timestamp + random suffix)
- [ ] T052 [P] [US2] Implement isolation validator in internal/runner/isolation.go (verify no state leakage between tests)
- [ ] T053 [US2] Add cleanup verification in internal/database/tempdb.go (ensure database is dropped after test)
- [ ] T054 [US2] Create order-independence test in internal/integration_test.go (run tests A→B and B→A, verify identical results)
- [ ] T055 [US2] Create isolation test fixtures in testdata/isolation/ (tests that modify state)
- [ ] T056 [US2] Verify test independence in integration tests (run same test twice, verify identical coverage)

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Flexible Configuration (Priority: P3)

**Goal**: Enable configuration via command-line flags, environment variables, or defaults for different environments

**Independent Test**: Connect to PostgreSQL using various configuration methods (flags, env vars) and verify successful test execution

### Implementation for User Story 3

- [ ] T057 [P] [US3] Add connection flags to run command in internal/cli/run.go (--host, --port, --user, --password, --database)
- [ ] T058 [P] [US3] Add execution flags to run command in internal/cli/run.go (--timeout, --parallel, --coverage-file, --verbose)
- [ ] T059 [US3] Implement configuration priority logic in internal/cli/config.go (flags override env vars override defaults)
- [ ] T060 [US3] Add environment variable binding in internal/cli/config.go (PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE)
- [ ] T061 [US3] Add configuration validation in internal/cli/config.go (check required fields, validate formats)
- [ ] T062 [US3] Add helpful error messages in internal/cli/config.go (suggest fixes for missing/invalid configuration)
- [ ] T063 [US3] Create configuration test in internal/cli/config_test.go (verify priority order and defaults)
- [ ] T064 [US3] Document all configuration options in README.md

**Checkpoint**: At this point, User Stories 1, 2, AND 3 should all work independently

---

## Phase 6: User Story 4 - Multiple Report Formats (Priority: P4)

**Goal**: Export coverage reports in JSON and LCOV formats for integration with CI systems and coverage tools

**Independent Test**: Run `pgcov report --format=json` and `pgcov report --format=lcov` and verify output format correctness

### Implementation for User Story 4

- [ ] T065 [P] [US4] Add format flag to report command in internal/cli/report.go (--format=json|lcov)
- [ ] T066 [P] [US4] Add output flag to report command in internal/cli/report.go (-o, --output)
- [ ] T067 [US4] Implement format selection logic in internal/cli/report.go (route to appropriate reporter)
- [ ] T068 [US4] Add coverage file validation in internal/cli/report.go (check file exists, valid JSON)
- [ ] T069 [US4] Add output writing in internal/cli/report.go (stdout or file based on --output flag)
- [ ] T070 [US4] Create JSON output test in internal/report/json_test.go (verify schema compliance)
- [ ] T071 [US4] Create LCOV output test in internal/report/lcov_test.go (verify format compliance)
- [ ] T072 [US4] Add report command examples to README.md

**Checkpoint**: All user stories should now be independently functional

---

## Phase 7: Parallel Execution Enhancement

**Goal**: Enable parallel test execution with configurable concurrency via --parallel flag

**Independent Test**: Run tests with `--parallel=4` and verify faster execution with correct coverage results

### Implementation for Parallel Execution

- [ ] T073 [P] Implement goroutine pool in internal/runner/parallel.go (worker pool pattern with semaphore)
- [ ] T074 [P] Implement thread-safe coverage aggregation in internal/coverage/collector.go (use sync.Mutex or channels)
- [ ] T075 Add parallel execution coordinator in internal/runner/parallel.go (distribute tests to workers)
- [ ] T076 Add connection pool sizing in internal/database/pool.go (ensure pool size ≥ parallel limit)
- [ ] T077 Wire parallel flag to executor in internal/cli/run.go (sequential if --parallel=1)
- [ ] T078 Create parallel execution test in internal/integration_test.go (verify coverage accuracy with parallel execution)
- [ ] T079 Create parallel test fixtures in testdata/parallel/ (multiple independent tests)

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T080 [P] Add comprehensive unit tests in internal/parser/parse_test.go (test parsing edge cases)
- [ ] T081 [P] Add unit tests in internal/instrument/instrumenter_test.go (test instrumentation correctness)
- [ ] T082 [P] Add unit tests in internal/database/tempdb_test.go (test database lifecycle)
- [ ] T083 [P] Add unit tests in internal/coverage/collector_test.go (test signal aggregation)
- [ ] T084 [P] Create edge case test fixtures in testdata/edge_cases/ (syntax errors, empty files, large files)
- [ ] T085 [P] Create PL/pgSQL test fixtures in testdata/plpgsql/ (functions, procedures, triggers)
- [ ] T086 Add error handling tests (verify all error types return correct messages)
- [ ] T087 Add integration tests for PostgreSQL 13, 14, 15, 16 (version compatibility)
- [ ] T088 [P] Add verbose logging throughout all layers (use --verbose flag)
- [ ] T089 [P] Optimize coverage data structure for large projects (benchmark 10k LOC)
- [ ] T090 [P] Document CLI contract in docs/cli-contract.md (copy from specs/)
- [ ] T091 [P] Create quickstart guide in docs/quickstart.md (copy from specs/)
- [ ] T092 [P] Create examples in examples/ directory (simple project, CI integration)
- [ ] T093 Run performance benchmark (100 test files in <5 minutes)
- [ ] T094 Run deterministic coverage test (same code/tests produce identical coverage)
- [ ] T095 Verify constitutional compliance (no violations of 6 principles)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational completion - MVP story
- **User Story 2 (Phase 4)**: Depends on Foundational completion - Can be implemented independently after US1
- **User Story 3 (Phase 5)**: Depends on Foundational completion - Can be implemented independently after US1
- **User Story 4 (Phase 6)**: Depends on User Story 1 completion (needs coverage data)
- **Parallel Execution (Phase 7)**: Depends on User Stories 1 & 2 completion
- **Polish (Phase 8)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Enhances US1 isolation but independently testable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - Adds configuration flexibility, independently testable
- **User Story 4 (P4)**: Depends on User Story 1 - Uses coverage data from US1

### Within Each User Story

- Discovery → Parser → Instrumentation → Database → Runner → Coverage → Reporter → CLI
- Models before services
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

**Setup (Phase 1)**:
- T004 (.gitignore) || T005 (README.md)

**Foundational (Phase 2)**:
- T006-T013 (all type definitions can be created in parallel)
- T014 (CLI skeleton) || T015 (version/help commands)

**User Story 1 (Phase 3)**:
- T016 (discovery) || T017 (classifier)
- T019 (parser wrapper) || T020 (AST utilities)
- T022 (instrumenter) || T023 (coverage point tracker)
- T026 (connection pool) || T027 (temp DB creator) || T028 (temp DB destroyer) || T029 (LISTEN/NOTIFY)
- T032 (executor) || T033 (per-test workflow)
- T036 (collector) || T037 (aggregation) || T038 (persistence)
- T040 (JSON reporter) || T041 (LCOV reporter)

**User Story 2 (Phase 4)**:
- T051 (naming) || T052 (isolation validator)

**User Story 3 (Phase 5)**:
- T057 (connection flags) || T058 (execution flags)

**User Story 4 (Phase 6)**:
- T065 (format flag) || T066 (output flag)
- T070 (JSON test) || T071 (LCOV test)

**Parallel Execution (Phase 7)**:
- T073 (goroutine pool) || T074 (thread-safe aggregation)

**Polish (Phase 8)**:
- T080-T085 (all unit tests can run in parallel)
- T088-T092 (documentation tasks can run in parallel)

---

## Parallel Example: User Story 1

```bash
# Launch discovery layer tasks in parallel:
Task T016: "Implement filesystem discovery in internal/discovery/discover.go"
Task T017: "Implement file classifier in internal/discovery/classifier.go"

# Launch parser layer tasks in parallel:
Task T019: "Create pg_query_go wrapper in internal/parser/parse.go"
Task T020: "Implement AST utilities in internal/parser/ast.go"

# Launch instrumentation layer tasks in parallel:
Task T022: "Implement AST instrumenter in internal/instrument/instrumenter.go"
Task T023: "Implement coverage point tracker in internal/instrument/location.go"

# Launch database layer tasks in parallel:
Task T026: "Implement PostgreSQL connection pool in internal/database/pool.go"
Task T027: "Implement temporary database creator in internal/database/tempdb.go"
Task T028: "Implement temporary database destroyer in internal/database/tempdb.go"
Task T029: "Implement LISTEN/NOTIFY handler in internal/database/listener.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T005)
2. Complete Phase 2: Foundational (T006-T015) - CRITICAL - blocks all stories
3. Complete Phase 3: User Story 1 (T016-T050)
4. **STOP and VALIDATE**: Test User Story 1 independently with `pgcov run` and `pgcov report`
5. Deploy/demo if ready

**MVP Scope**: User Story 1 provides complete core functionality - discover tests, run them with isolation, collect coverage, generate reports

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 (T016-T050) → Test independently → Deploy/Demo (MVP! ✅)
3. Add User Story 2 (T051-T056) → Test independently → Deploy/Demo (Better isolation)
4. Add User Story 3 (T057-T064) → Test independently → Deploy/Demo (Flexible config)
5. Add User Story 4 (T065-T072) → Test independently → Deploy/Demo (Multiple formats)
6. Add Parallel Execution (T073-T079) → Test independently → Deploy/Demo (Performance)
7. Polish (T080-T095) → Final validation → Release

Each story adds value without breaking previous stories.

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together (T001-T015)
2. Once Foundational is done:
   - Developer A: User Story 1 (T016-T050)
   - Developer B: User Story 2 (T051-T056) - can start after foundational
   - Developer C: User Story 3 (T057-T064) - can start after foundational
3. User Story 4 waits for User Story 1 completion
4. Parallel Execution waits for User Stories 1 & 2
5. Stories complete and integrate independently

---

## Summary

**Total Tasks**: 95 tasks

**Task Count by Phase**:
- Phase 1 (Setup): 5 tasks
- Phase 2 (Foundational): 10 tasks
- Phase 3 (User Story 1): 35 tasks
- Phase 4 (User Story 2): 6 tasks
- Phase 5 (User Story 3): 8 tasks
- Phase 6 (User Story 4): 8 tasks
- Phase 7 (Parallel Execution): 7 tasks
- Phase 8 (Polish): 16 tasks

**Parallel Opportunities**: 50+ tasks marked [P] can run in parallel within their phase

**Independent Test Criteria**:
- **US1**: Place test files, run `pgcov run`, verify coverage report generated
- **US2**: Run tests in different orders, verify identical results
- **US3**: Configure via flags/env vars, verify successful execution
- **US4**: Run `pgcov report` with different formats, verify output correctness

**Suggested MVP Scope**: User Story 1 only (T001-T050) - provides complete core functionality

**Format Validation**: ✅ All tasks follow checklist format (checkbox, ID, [P] marker for parallelizable tasks, [Story] label for user story tasks, file paths in descriptions)

---

## Notes

- [P] tasks = different files, no dependencies within their layer
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
