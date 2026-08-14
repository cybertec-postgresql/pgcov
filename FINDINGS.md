## Inconsistencies

### I1 — `pgcov` NOTIFY channel name hardcoded in two separate places

> **Status: IMPLEMENTED** — stacked PR #16 (`feat(coverage): single-source, per-run unique NOTIFY channel name`, combined with E6). Verified 2026-08-14: hardcodes confirmed at instrumenter.go ×3 and executor.go ×1.

`internal/instrument/instrumenter.go` hardcodes `'pgcov'` inside the injected
`pg_notify(...)` call.  `internal/runner/executor.go` passes `"pgcov"` to
`database.NewListener`.  If either is changed independently, coverage collection breaks
silently.  A single exported constant (e.g. `instrument.CoverageChannel`) should be the
single source of truth.

---

### I2 — `Branch` field in `CoveragePoint` is always empty string

> **Status: IMPLEMENTED** — stacked PR #17 (`refactor(instrument): remove dead branch-coverage API surface`). Dead code removed: `CoveragePoint.Branch` field, `branch` param in `FormatSignalID`, 4-part branch path in `ParseSignalID`. Note: the `TrackBranchPosition` claim in this finding was inaccurate — that function does not exist; the rest was verified.

`types.go` defines `CoveragePoint.Branch`, `FormatSignalID` and `ParseSignalID` both
handle a `:branch` suffix, and `TrackBranchPosition` exists in `location.go` — but
`instrumentBody` always sets `Branch: ""` and the collector ignores branch data when
aggregating.  This is an unimplemented feature occupying API surface.  Either implement
it or remove the dead code.

---

### I3 — `PositionKey()` format differs from the key format used by the Collector

> **Status: IGNORED** — finding is factually wrong (verified 2026-08-14). No exported `PositionKey` function exists in `internal/instrument/location.go` (or anywhere in the tree); it contains only `FormatSignalID`, `ParseSignalID`, `parseNumber`. The collector's private `formatPositionKey` (`internal/coverage/model.go`) is the sole, consistent position-key format.

`location.go` exports:

```go
func PositionKey(file, startPos, length) string { return "file:startPos:length" }
```

The collector stores keys as:

```go
posKey := fmt.Sprintf("%d:%d", startPos, length)   // no file prefix
```

`PositionKey` is never called by the collector; the two formats are incompatible.  The
exported function is misleading and unused internally.

---

### I4 — `TrackPosition()` and `TrackBranchPosition()` in `location.go` are dead code

> **Status: IGNORED** — finding is factually wrong (verified 2026-08-14). Neither `TrackPosition` nor `TrackBranchPosition` exists anywhere in the source tree (`location.go` holds only `FormatSignalID`/`ParseSignalID`/`parseNumber`). The genuinely dead branch API was `Branch`/`FormatSignalID`, handled under I2.

Neither function is called anywhere in the production code.  They are part of the
unimplemented branch-tracking API (see I2).

---

### I5 — `IsolationValidator` lives in production code but is never wired up

> **Status: IGNORED** — finding is factually wrong (verified 2026-08-14). `internal/runner/isolation.go` does not exist and no `IsolationValidator`/`TrackDatabase`/`MarkCleaned`/`ValidateCleanup` symbol exists anywhere in the repository.

`internal/runner/isolation.go` provides a fully functional `IsolationValidator` with
`TrackDatabase` / `MarkCleaned` / `ValidateCleanup`, but `executor.go` never calls any
of these methods.  The struct is only used in tests.  It should either be wired in
(pre-flight check for uniqueness / post-run cleanup confirmation) or moved to a test
helper package.

---

### I6 — `cli-contract.md` describes a schema that was never implemented

> **Status: IMPLEMENTED** — stacked PR #24 (`docs(cli): rewrite cli-contract.md to match the implemented CLI and JSON schema`). Document rewritten to match reality: `--connection` flag set, exit codes 0/1/2, `{version, timestamp, positions}` JSON schema. Verified: fictitious `--host/--port/--user/--password/--database` flags, `lines`/`branches` schema, and exit code 3 confirmed absent from code.

`docs/cli-contract.md` (§ Coverage Data File Contract) documents a JSON schema with
`lines` (containing `LineCoverage` objects) and `branches` (containing `BranchCoverage`
objects).  The actual JSON output is:

```json
{"positions": {"file.sql": {"0:42": 3}}}
```

The CLI contract also lists individual flags (`--host`, `--port`, `--user`, `--password`,
`--database`) that do not exist; the real flag is a single `--connection` URI string.
The document should be updated or the flags added.

---

### I7 — `ClassifyFile` returns `FileTypeSource` for non-SQL files

> **Status: IMPLEMENTED** — stacked PR #14 (`fix(discovery): return FileTypeUnknown for non-SQL files in ClassifyFile`). Note: the claimed risk was partially inaccurate — `Discover` already filters non-`.sql` files before calling `ClassifyFile`, so the parser could never receive arbitrary files; the fix is API hardening only.

If `filepath.Walk` delivers a non-`.sql` file, `ClassifyFile` returns `FileTypeSource`
(documented as an "edge case").  This could cause the parser to attempt reading arbitrary
files.  An explicit `FileTypeUnknown` or early-return guard in `Discover` would be safer.

---

### I8 — `GetFiles()` / `GetFileList()` return unsorted slices

> **Status: IMPLEMENTED** — stacked PR #15 (`refactor(coverage): return sorted file lists from GetFiles and GetFileList`). Verified: both getters iterated maps unsorted and every reporter caller immediately re-sorted; sorting moved into the getters, redundant caller sorts removed.

Both `Coverage.GetFiles()` (`model.go`) and `Collector.GetFileList()` (`collector.go`)
return files in random map-iteration order.  Every caller (all three reporters) performs
its own `sort.Strings()` immediately after.  Sorting once in these methods would
eliminate the repeated pattern.

---

### I9 — `isExecutableSegment` does not recognise `RETURN` as a segment boundary marker

> **Status: IGNORED** — not a defect (verified 2026-08-14). The finding itself concedes treating `RETURN` as executable "is correct". The claimed follow-on risk (signals after `RETURN` being unreachable) is already handled: `findTerminalPos` places the coverage signal BEFORE terminal `RETURN`/`RAISE` statements, covered by `TestInstrumentBody_ReturnInBranches` and related B2 tests.

`isExecutableSegment` excludes `BEGIN`, `END`, `LOOP`, `DECLARE`, and `EXCEPTION` from
instrumentation.  `RETURN` is treated as an ordinary executable statement.  While this
is correct — a `RETURN` is executable — it means a bare `RETURN;` at the end of a
complex function body generates its own signal, inflating coverage point counts.  More
importantly, any signal injected immediately *after* a `RETURN` statement becomes
unreachable (see B2), but the exclusion list does not account for this.

---

## Improvements / Enhancements

### E1 — No coverage merge / accumulation across runs

> **Status: IMPLEMENTED** — stacked PR #21 (`feat(cli): add merge command to accumulate coverage across runs`). New `pgcov merge` subcommand + `coverage.Merge()` summing per-position hit counts.

`pgcov run` always overwrites `.pgcov/coverage.json`.  There is no `pgcov merge` command
or `--append` flag for accumulating coverage across CI matrix shards or multiple test
directories.  The CI example configs work around this with external `jq` post-processing.

---

### E2 — No `--fail-under` coverage threshold

> **Status: IMPLEMENTED** — stacked PR #20 (`feat(cli): add --fail-under coverage threshold to run command`). `--fail-under` flag; exit 1 when total coverage is below the threshold, with test-failure exit codes taking precedence.

There is no built-in mechanism to exit non-zero when coverage falls below a configured
threshold.  This is a standard CI requirement and is simulated in the example CI configs
using shell arithmetic on the JSON output.

---

### E3 — Source file deployment order is undefined

> **Status: IMPLEMENTED (docs)** — stacked PR #25 (`docs: document lexical source deployment order and numeric-prefix convention`). Note: the premise was partially wrong — `filepath.Walk` guarantees lexical order within each directory, so deployment order is deterministic; the gap was documentation of that guarantee plus the numeric-prefix convention and `--setup` escape hatch.

`filepath.Walk` delivers files in lexicographic order within each directory, but the
behaviour is OS-specific.  If `02_functions.sql` depends on objects created by
`01_schema.sql`, naming convention is the only ordering guarantee.  A numeric-prefix
convention should be documented, or an explicit `order` annotation / `pgcov.toml`
manifest should be supported.

---

### E4 — No `--base-dir` flag for source resolution in reporters

> **Status: IMPLEMENTED** — stacked PR #22 (`feat(report): add --base-dir flag for source resolution at report time`). HTML/LCOV reporters gain `SetBaseDir` (Formatter interface unchanged); verified sources resolve against base-dir from a different CWD.

The HTML and LCOV reporters resolve source file paths relative to the current working
directory at report time.  If `pgcov report` is run from a different directory than
`pgcov run`, source content cannot be found and the HTML report shows an error comment
in place of annotated code.  A `--base-dir` flag would decouple the report generation
location from the run location.

---

### E5 — Hardcoded 100 ms grace period in `CollectSignals`

> **Status: IMPLEMENTED** — stacked PR #18 (`feat(cli): add --signal-timeout flag for coverage signal grace period`). Verified hardcode at executor.go (`100*time.Millisecond`); now `Config.SignalTimeout` + `--signal-timeout` flag, with `runner.DefaultSignalTimeout` fallback for non-positive values (bare `Config{}` constructions previously got a 0 ms grace and lost signals nondeterministically — caught by `TestTestIndependence` during stack verification).

After test SQL executes, `CollectSignals` waits 100 ms for in-flight notifications.  On
slow systems or under heavy parallel load this window may be too short, causing
legitimate signals to be lost.  The value should be exposed as a configuration option
(e.g. `--signal-timeout`).

---

### E6 — Listener channel is not per-run unique

> **Status: IMPLEMENTED** — stacked PR #16 (combined with I1: `feat(coverage): single-source, per-run unique NOTIFY channel name`). `Config.CoverageChannel` is the single source of truth; `pgcov run` generates `pgcov_<8 hex>` per invocation and threads it to both the injected `pg_notify` calls and the listener. Note: NOTIFY is database-scoped and each test runs in its own temp DB, so interference required user code inside the temp DB NOTIFYing on `pgcov` — now structurally impossible.

The `pgcov` NOTIFY channel is a well-known name.  If the user's application already
publishes notifications on a channel named `pgcov`, signals will be delivered to the
listener in addition to (or instead of) the coverage ones.  Making the channel name
incorporate the temp-database name or a random UUID would eliminate interference.

---

### E7 — `TestRun.Error` is set but not surfaced in the summary output

> **Status: IMPLEMENTED** — stacked PR #19 (`feat(cli): print failed test errors in run summary`). Verified: summary printed only counts; now `runner.FormatFailedTests` prints `FAILED <path>: <error>` per failed run (unit-tested helper, no DB needed).

When a test fails, `testRun.Error` holds the Go-level error.  The CLI summary prints
`X failed` but does not print the error message.  Users must re-run with `--verbose` to
see why a test failed.

---

### E8 — No `pgcov init` / scaffold command

> **Status: IMPLEMENTED** — stacked PR #23 (`feat(cli): add init command to scaffold test files from sources`). `pgcov init <source.sql>` extracts `CREATE [OR REPLACE] FUNCTION` signatures via the existing pglex tokenizer (no regex) and writes a commented `_test.sql` scaffold, with `--output`/`--force` guards.

There is no helper to generate a skeleton `_test.sql` file for an existing source.  A
`pgcov init` command that reads function signatures from a source file and emits a
commented test scaffold would lower the barrier to adoption.

---

### E9 — Pool `MaxConns` formula may be insufficient

> **Status: IGNORED** — premise is factually wrong (verified 2026-08-14). The LISTEN connection is a dedicated `pgx.ConnectConfig` connection in `NewListener` — it does NOT come from the pool. Source deployment and test SQL acquire pool connections sequentially (acquire → exec → release), never 3 simultaneously. `MaxConns = parallelism*2` applies to the admin pool, which only serves short `CREATE/DROP DATABASE` calls. No acquire-timeout path exists as described; bumping the multiplier would address nothing.

`pool.go` sets `MaxConns = parallelism * 2` under the assumption that each test uses
two connections (one for execution, one for LISTEN).  Each test actually acquires up to
three connections simultaneously (pool acquire for sources, listener connection, pool
acquire for test SQL).  Under high parallelism this can cause connection pool exhaustion
and `pgx` acquire timeouts.

---

### E10 — `CREATE DATABASE` uses an unquoted identifier

> **Status: IMPLEMENTED** — stacked PR #13 (`fix(database): quote temp database identifiers with pgx.Identifier.Sanitize`). Verified: both `CREATE DATABASE` and `DROP DATABASE ... WITH (FORCE)` (plus the rollback DROP) now sanitize the identifier.

`tempdb.go`:

```go
adminPool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName))
```

The generated name (`pgcov_test_20260318_140534_ab12cd34`) uses only lowercase, digits
and underscores, so it is safe today.  However, the pattern should use `pgx`'s
`pgx.Identifier.Sanitize()` (quote the identifier) to be robust against any future
change to the name format and to prevent a theoretical injection if the name-generation
logic ever incorporates external input.

---

*File generated 2026-03-18 based on a code review of the full pgcov source tree and
hands-on testing with the `examples/demo` walkthrough.*
