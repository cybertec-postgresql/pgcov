## Inconsistencies

### I1 — `pgcov` NOTIFY channel name hardcoded in two separate places

`internal/instrument/instrumenter.go` hardcodes `'pgcov'` inside the injected
`pg_notify(...)` call.  `internal/runner/executor.go` passes `"pgcov"` to
`database.NewListener`.  If either is changed independently, coverage collection breaks
silently.  A single exported constant (e.g. `instrument.CoverageChannel`) should be the
single source of truth.

---

### I2 — `Branch` field in `CoveragePoint` is always empty string

`types.go` defines `CoveragePoint.Branch`, `FormatSignalID` and `ParseSignalID` both
handle a `:branch` suffix, and `TrackBranchPosition` exists in `location.go` — but
`instrumentBody` always sets `Branch: ""` and the collector ignores branch data when
aggregating.  This is an unimplemented feature occupying API surface.  Either implement
it or remove the dead code.

---

### I3 — `PositionKey()` format differs from the key format used by the Collector

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

Neither function is called anywhere in the production code.  They are part of the
unimplemented branch-tracking API (see I2).

---

### I5 — `IsolationValidator` lives in production code but is never wired up

`internal/runner/isolation.go` provides a fully functional `IsolationValidator` with
`TrackDatabase` / `MarkCleaned` / `ValidateCleanup`, but `executor.go` never calls any
of these methods.  The struct is only used in tests.  It should either be wired in
(pre-flight check for uniqueness / post-run cleanup confirmation) or moved to a test
helper package.

---

### I6 — `cli-contract.md` describes a schema that was never implemented

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

If `filepath.Walk` delivers a non-`.sql` file, `ClassifyFile` returns `FileTypeSource`
(documented as an "edge case").  This could cause the parser to attempt reading arbitrary
files.  An explicit `FileTypeUnknown` or early-return guard in `Discover` would be safer.

---

### I8 — `GetFiles()` / `GetFileList()` return unsorted slices

Both `Coverage.GetFiles()` (`model.go`) and `Collector.GetFileList()` (`collector.go`)
return files in random map-iteration order.  Every caller (all three reporters) performs
its own `sort.Strings()` immediately after.  Sorting once in these methods would
eliminate the repeated pattern.

---

### I9 — `isExecutableSegment` does not recognise `RETURN` as a segment boundary marker

`isExecutableSegment` excludes `BEGIN`, `END`, `LOOP`, `DECLARE`, and `EXCEPTION` from
instrumentation.  `RETURN` is treated as an ordinary executable statement.  While this
is correct — a `RETURN` is executable — it means a bare `RETURN;` at the end of a
complex function body generates its own signal, inflating coverage point counts.  More
importantly, any signal injected immediately *after* a `RETURN` statement becomes
unreachable (see B2), but the exclusion list does not account for this.

---

## Improvements / Enhancements

### E1 — No coverage merge / accumulation across runs

`pgcov run` always overwrites `.pgcov/coverage.json`.  There is no `pgcov merge` command
or `--append` flag for accumulating coverage across CI matrix shards or multiple test
directories.  The CI example configs work around this with external `jq` post-processing.

---

### E2 — No `--fail-under` coverage threshold

There is no built-in mechanism to exit non-zero when coverage falls below a configured
threshold.  This is a standard CI requirement and is simulated in the example CI configs
using shell arithmetic on the JSON output.

---

### E3 — Source file deployment order is undefined

`filepath.Walk` delivers files in lexicographic order within each directory, but the
behaviour is OS-specific.  If `02_functions.sql` depends on objects created by
`01_schema.sql`, naming convention is the only ordering guarantee.  A numeric-prefix
convention should be documented, or an explicit `order` annotation / `pgcov.toml`
manifest should be supported.

---

### E4 — No `--base-dir` flag for source resolution in reporters

The HTML and LCOV reporters resolve source file paths relative to the current working
directory at report time.  If `pgcov report` is run from a different directory than
`pgcov run`, source content cannot be found and the HTML report shows an error comment
in place of annotated code.  A `--base-dir` flag would decouple the report generation
location from the run location.

---

### E5 — Hardcoded 100 ms grace period in `CollectSignals`

After test SQL executes, `CollectSignals` waits 100 ms for in-flight notifications.  On
slow systems or under heavy parallel load this window may be too short, causing
legitimate signals to be lost.  The value should be exposed as a configuration option
(e.g. `--signal-timeout`).

---

### E6 — Listener channel is not per-run unique

The `pgcov` NOTIFY channel is a well-known name.  If the user's application already
publishes notifications on a channel named `pgcov`, signals will be delivered to the
listener in addition to (or instead of) the coverage ones.  Making the channel name
incorporate the temp-database name or a random UUID would eliminate interference.

---

### E7 — `TestRun.Error` is set but not surfaced in the summary output

When a test fails, `testRun.Error` holds the Go-level error.  The CLI summary prints
`X failed` but does not print the error message.  Users must re-run with `--verbose` to
see why a test failed.

---

### E8 — No `pgcov init` / scaffold command

There is no helper to generate a skeleton `_test.sql` file for an existing source.  A
`pgcov init` command that reads function signatures from a source file and emits a
commented test scaffold would lower the barrier to adoption.

---

### E9 — Pool `MaxConns` formula may be insufficient

`pool.go` sets `MaxConns = parallelism * 2` under the assumption that each test uses
two connections (one for execution, one for LISTEN).  Each test actually acquires up to
three connections simultaneously (pool acquire for sources, listener connection, pool
acquire for test SQL).  Under high parallelism this can cause connection pool exhaustion
and `pgx` acquire timeouts.

---

### E10 — `CREATE DATABASE` uses an unquoted identifier

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
