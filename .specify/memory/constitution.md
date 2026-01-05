<!--
Sync Impact Report - Constitution v1.0.0

Version Change: INITIAL → 1.0.0 (Initial ratification)
Rationale: First constitution establishment for pgcov project

Principles Established:
  1. Direct Protocol Access (NEW) - No psql/shell dependency, direct PostgreSQL protocol
  2. Test Isolation (NEW) - Order-independent, self-contained tests
  3. Instrumentation Transparency (NEW) - Transparent, removable source instrumentation
  4. CLI-First Design (NEW) - Standalone CLI tool for CI/CD and local development
  5. Coverage Accuracy Over Speed (NEW) - Deterministic, reproducible coverage
  6. Go Development Ergonomics (NEW) - Familiar to Go developers (like `go test`)

Added Sections:
  - Core Principles (6 principles)
  - Implementation Constraints
  - Non-Goals & Boundaries
  - Governance

Templates Requiring Updates:
  ✅ plan-template.md - Constitution Check section references this file
  ✅ spec-template.md - No specific pgcov references needed (generic template)
  ✅ tasks-template.md - No specific pgcov references needed (generic template)
  ✅ agent-file-template.md - No specific pgcov references needed (generic template)

Follow-up TODOs: None
-->

# pgcov Constitution

## Core Principles

### I. Direct Protocol Access

pgcov MUST connect directly to PostgreSQL using the native protocol without depending on psql, shell execution, or external command-line tools. This principle ensures:

- Consistent behavior across environments (CI/CD, local development, containers)
- Eliminates shell injection vulnerabilities and escaping complexity
- Enables programmatic control of connection lifecycle and transaction boundaries
- Simplifies deployment (no external tool dependencies)

**Rationale**: CLI tools introduce variability, security concerns, and deployment complexity that conflict with CI/CD reliability requirements.

### II. Test Isolation

Each test MUST be isolated, order-independent, and self-contained. Tests MUST handle their own setup and teardown. This principle requires:

- No shared state between tests
- No assumptions about execution order
- Each test creates and destroys its own fixtures
- Tests can run in parallel without coordination

**Rationale**: Order-dependent tests create brittle test suites that fail unpredictably and prevent parallel execution, undermining CI/CD reliability.

### III. Instrumentation Transparency

pgcov MUST instrument SQL and PL/pgSQL source code in a transparent, removable manner. Instrumentation MUST NOT:

- Require PostgreSQL extensions
- Modify PostgreSQL internals or system catalogs
- Leave permanent artifacts after coverage collection
- Change the semantic behavior of the tested code

**Rationale**: Transparent instrumentation ensures pgcov works with any PostgreSQL version and hosting environment without special permissions or configuration.

### IV. CLI-First Design

pgcov MUST be a standalone CLI tool optimized for both CI/CD pipelines and local development workflows. The tool MUST:

- Accept configuration via command-line flags, environment variables, or configuration files
- Provide clear, actionable output suitable for automated parsing
- Exit with appropriate status codes for pipeline integration
- Feel familiar to Go developers (similar to `go test`, `go build`)

**Rationale**: CI/CD integration is a primary use case; CLI-first design ensures scriptability and automation-friendly behavior.

### V. Coverage Accuracy Over Speed

Coverage accuracy MUST take precedence over execution speed. pgcov MUST produce:

- Deterministic results (same code → same coverage)
- Reproducible reports across runs
- Accurate line and branch coverage
- No false positives or false negatives

**Rationale**: Inaccurate coverage metrics undermine trust and testing discipline; speed optimizations are secondary to correctness.

### VI. Go Development Ergonomics

pgcov MUST provide a development experience familiar to Go developers. This includes:

- Command structure similar to Go toolchain (`pgcov test`, `pgcov coverage`)
- Output formats compatible with existing tools (LCOV, JSON)
- Testing patterns that align with Go conventions
- Clear, Go-idiomatic error messages

**Rationale**: Reduces learning curve and increases adoption by aligning with established Go ecosystem conventions.

## Implementation Constraints

**Language**: Go (latest stable version)

**Core Dependencies**:
- `pg_query_go` for SQL parsing (official PostgreSQL parser bindings)
- `pgx` for PostgreSQL protocol access (direct connection, no psql)

**Output Formats**:
- JSON (machine-readable, structured coverage data)
- LCOV (for integration with existing coverage tools and CI systems)

**Platform Support**: Cross-platform (Linux, macOS, Windows) as single static binary

**Testing Requirements**:
- Unit tests for parser and instrumentation logic
- Integration tests against real PostgreSQL instances (multiple versions)
- Contract tests for output format stability

## Non-Goals & Boundaries

pgcov explicitly does NOT:

- **Manage database migrations**: Use dedicated migration tools (e.g., golang-migrate, Flyway)
- **Manage schema lifecycle globally**: Each test manages its own schema state
- **Replace assertion libraries**: Works alongside pgTAP or other testing frameworks
- **Require PostgreSQL extensions**: Must work with vanilla PostgreSQL

These boundaries keep pgcov focused on its core mission: test execution and coverage reporting.

## Governance

This constitution supersedes all other development practices and guidelines for pgcov. All design decisions, feature implementations, and code reviews MUST verify compliance with these principles.

**Amendment Process**:
- Amendments require documented rationale and impact analysis
- Version increments follow semantic versioning (see Version History below)
- Breaking changes to principles require MAJOR version bump
- New principles or expanded guidance require MINOR version bump
- Clarifications and wording improvements require PATCH version bump

**Compliance Review**:
- All feature specifications MUST reference applicable principles
- Implementation plans MUST include "Constitution Check" section
- Code reviews MUST verify principle adherence
- Violations require explicit justification in complexity tracking

**Development Guidance**:
- Runtime development context maintained in auto-generated files
- Templates (plan, spec, tasks) reference this constitution for validation gates
- Agent workflows derive structure from these principles

**Version**: 1.0.0 | **Ratified**: 2026-01-05 | **Last Amended**: 2026-01-05
