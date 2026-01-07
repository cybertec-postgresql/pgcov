# CI Integration Examples

This directory contains example CI/CD configuration files for integrating pgcov into your continuous integration pipeline.

## Available Examples

### GitHub Actions

**File**: `github-actions.yml`

Copy this file to `.github/workflows/coverage.yml` in your repository.

**Features**:
- Runs PostgreSQL 16 as a service
- Builds pgcov from source
- Runs all tests with coverage
- Generates LCOV report
- Uploads coverage to Codecov
- Comments coverage on pull requests
- Uploads coverage artifacts

**Setup**:
1. Copy file to `.github/workflows/coverage.yml`
2. Add `CODECOV_TOKEN` to repository secrets (if using Codecov)
3. Push to trigger workflow

### GitLab CI

**File**: `gitlab-ci.yml`

Copy this file to `.gitlab-ci.yml` in your repository.

**Features**:
- Uses PostgreSQL 16 service
- Three-stage pipeline (build, test, coverage)
- Generates LCOV and HTML reports
- Coverage threshold enforcement (80%)
- Coverage badge integration
- Artifacts retention

**Setup**:
1. Copy file to `.gitlab-ci.yml` in repository root
2. Adjust `THRESHOLD` if needed (default: 80%)
3. Commit and push to trigger pipeline

## Configuration Options

### Environment Variables

All CI examples use these PostgreSQL environment variables:

```yaml
PGHOST: localhost  # or 'postgres' for GitLab
PGPORT: 5432
PGUSER: postgres
PGPASSWORD: postgres
PGDATABASE: postgres
```

### pgcov Flags

Common flags used in CI:

- `--coverage-file`: Where to write coverage data (default: `.pgcov/coverage.json`)
- `--verbose`: Enable detailed logging for debugging
- `--parallel=4`: Run tests in parallel (faster CI)
- `--timeout=60s`: Increase timeout for slow tests

### Coverage Thresholds

Both examples support coverage thresholds:

**GitHub Actions** (using jq):
```yaml
- name: Check coverage threshold
  run: |
    PERCENT=$(jq -r '.coverage.total.percent' coverage.json)
    if (( $(echo "$PERCENT < 80" | bc -l) )); then
      echo "Coverage $PERCENT% below 80%"
      exit 1
    fi
```

**GitLab CI** (built-in):
```yaml
coverage: '/SQL Coverage: (\d+\.\d+)%/'
```

## Coverage Report Integration

### Codecov

GitHub Actions example includes Codecov integration:

```yaml
- uses: codecov/codecov-action@v4
  with:
    files: ./coverage.lcov
    flags: sql-tests
```

### Coveralls

Add Coveralls support:

```yaml
- name: Upload to Coveralls
  uses: coverallsapp/github-action@v2
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    path-to-lcov: coverage.lcov
```

### SonarQube

For SonarQube integration, use the generic coverage format:

```bash
pgcov report --format=generic -o sonar-coverage.xml
```

Then configure in `sonar-project.properties`:
```properties
sonar.coverageReportPaths=sonar-coverage.xml
```

## HTML Reports

Generate HTML coverage reports using `genhtml`:

```bash
# Generate LCOV format
pgcov report --format=lcov -o coverage.lcov

# Convert to HTML
genhtml coverage.lcov -o coverage_html/

# View in browser
open coverage_html/index.html
```

## Parallel Execution in CI

Speed up CI by running tests in parallel:

```bash
# Run 4 tests concurrently
pgcov run ./... --parallel=4
```

**Note**: Ensure your PostgreSQL service has sufficient connection limits.

## Caching

### GitHub Actions

Cache Go modules and build artifacts:

```yaml
- name: Cache Go modules
  uses: actions/cache@v4
  with:
    path: |
      ~/.cache/go-build
      ~/go/pkg/mod
    key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
```

### GitLab CI

Cache Go modules:

```yaml
cache:
  key: ${CI_COMMIT_REF_SLUG}
  paths:
    - .cache/go-build
    - .cache/go-mod
```

## Troubleshooting

### PostgreSQL Connection Issues

If tests fail to connect:

1. Check service health:
   ```yaml
   options: >-
     --health-cmd pg_isready
     --health-interval 10s
   ```

2. Add connection wait:
   ```bash
   until pg_isready -h postgres; do sleep 1; done
   ```

3. Verify environment variables are set correctly

### Coverage File Not Found

If `pgcov report` fails:

1. Check that `pgcov run` completed successfully
2. Verify `--coverage-file` path matches in both commands
3. Add artifacts between jobs:
   ```yaml
   artifacts:
     paths:
       - coverage.json
   ```

### Low Coverage in CI

If coverage is lower than expected:

1. Use `--verbose` to see which tests run
2. Check test discovery with `pgcov run --dry-run`
3. Verify source files are co-located with tests

## Best Practices

1. **Run on Every PR**: Catch coverage regressions early
2. **Set Thresholds**: Enforce minimum coverage (e.g., 80%)
3. **Comment on PRs**: Show coverage changes in pull requests
4. **Cache Dependencies**: Speed up builds with caching
5. **Parallel Execution**: Use `--parallel` for faster tests
6. **Fail Fast**: Use timeouts to prevent hanging tests
7. **Archive Reports**: Save coverage artifacts for history

## Example Workflow

1. Developer pushes code
2. CI builds pgcov
3. CI runs tests with coverage
4. CI generates LCOV report
5. CI uploads to coverage service (Codecov, Coveralls)
6. CI comments coverage on PR
7. CI fails if coverage below threshold
8. CI uploads artifacts for download

## Additional Resources

- [GitHub Actions Documentation](https://docs.github.com/actions)
- [GitLab CI Documentation](https://docs.gitlab.com/ee/ci/)
- [Codecov Documentation](https://docs.codecov.com/)
- [LCOV Documentation](https://ltp.sourceforge.net/coverage/lcov.php)
