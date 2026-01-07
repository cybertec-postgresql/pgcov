# pgcov Examples

This directory contains example projects and configurations demonstrating various use cases for pgcov.

## Available Examples

### 1. Simple Example

**Directory**: `simple/`

A minimal example showing basic pgcov usage:
- Single source file with a PL/pgSQL function
- Single test file with test cases
- Demonstrates co-location strategy
- Shows basic assertions and error handling

**Use this to**:
- Learn pgcov basics
- Understand file organization
- See minimal working example

**Quick start**:
```bash
cd simple/
pgcov run .
```

### 2. CI Integration Examples

**Directory**: `ci-integration/`

Production-ready CI/CD configurations:
- GitHub Actions workflow
- GitLab CI pipeline
- Coverage threshold enforcement
- Report generation and upload
- PR comments with coverage stats

**Use this to**:
- Integrate pgcov into CI/CD
- Set up automated coverage checks
- Configure coverage reporting services
- Enforce coverage standards

**Quick start**:
```bash
# Copy GitHub Actions config
cp ci-integration/github-actions.yml .github/workflows/coverage.yml

# Or copy GitLab CI config
cp ci-integration/gitlab-ci.yml .gitlab-ci.yml
```

## Example Structure

```
examples/
├── README.md              # This file
├── simple/                # Basic usage example
│   ├── calculate.sql      # Source file
│   ├── calculate_test.sql # Test file
│   └── README.md          # Example documentation
└── ci-integration/        # CI/CD configurations
    ├── github-actions.yml # GitHub Actions workflow
    ├── gitlab-ci.yml      # GitLab CI pipeline
    └── README.md          # CI setup guide
```

## General Usage Pattern

All examples follow this pattern:

1. **Organize files**: Co-locate test files with source files
   ```
   myproject/
   ├── feature.sql       # Source
   └── feature_test.sql  # Test (in same directory)
   ```

2. **Write tests**: Use SQL assertions to verify behavior
   ```sql
   DO $$
   BEGIN
       ASSERT my_function(42) = expected_value;
   END $$;
   ```

3. **Run tests**: Execute with pgcov
   ```bash
   pgcov run .
   ```

4. **Generate reports**: Choose output format
   ```bash
   pgcov report --format=lcov -o coverage.lcov
   ```

## Common Patterns

### Testing Functions

```sql
-- Source: math.sql
CREATE FUNCTION add(a INT, b INT) RETURNS INT AS $$
BEGIN
    RETURN a + b;
END;
$$ LANGUAGE plpgsql;

-- Test: math_test.sql
DO $$
BEGIN
    ASSERT add(2, 3) = 5, 'Addition failed';
END $$;
```

### Testing Procedures

```sql
-- Source: log.sql
CREATE PROCEDURE log_event(msg TEXT) AS $$
BEGIN
    RAISE NOTICE '%', msg;
END;
$$ LANGUAGE plpgsql;

-- Test: log_test.sql
CALL log_event('Test message');
```

### Testing with Setup/Teardown

```sql
-- Test: user_test.sql
DO $$
BEGIN
    -- Setup
    CREATE TABLE users (id INT, name TEXT);
    INSERT INTO users VALUES (1, 'Alice');
    
    -- Test
    ASSERT (SELECT COUNT(*) FROM users) = 1;
    
    -- Teardown (automatic - temp database is destroyed)
END $$;
```

### Testing Error Conditions

```sql
DO $$
BEGIN
    -- Test that function raises exception
    BEGIN
        PERFORM divide_by_zero();
        RAISE EXCEPTION 'Should have raised error';
    EXCEPTION
        WHEN division_by_zero THEN
            -- Expected
    END;
END $$;
```

## Configuration Examples

### Minimal Configuration

```bash
# Use defaults
pgcov run .
```

### Full Configuration

```bash
# All options
pgcov run ./... \
  --host=localhost \
  --port=5432 \
  --user=postgres \
  --password=secret \
  --database=postgres \
  --timeout=60s \
  --parallel=4 \
  --coverage-file=.pgcov/coverage.json \
  --verbose
```

### Using Environment Variables

```bash
# Set connection
export PGHOST=localhost
export PGPORT=5432
export PGUSER=postgres
export PGPASSWORD=secret

# Run with defaults
pgcov run .
```

## Report Formats

### JSON (default)

```bash
pgcov report --format=json
```

Output: Machine-readable coverage data

### LCOV

```bash
pgcov report --format=lcov -o coverage.lcov
```

Output: LCOV tracefile for genhtml, Codecov, Coveralls

### HTML (via genhtml)

```bash
pgcov report --format=lcov -o coverage.lcov
genhtml coverage.lcov -o coverage_html/
open coverage_html/index.html
```

Output: HTML visualization of coverage

## Performance Tips

1. **Use parallel execution**: `--parallel=4` for faster tests
2. **Increase timeout for slow tests**: `--timeout=120s`
3. **Run specific directories**: `pgcov run ./critical/` instead of `./...`
4. **Use CI caching**: Cache Go modules and build artifacts
5. **Skip verbose locally**: Only use `--verbose` for debugging

## Best Practices

1. **Co-locate tests and source**: Same directory for discovery
2. **One test file per source file**: Clear mapping
3. **Use descriptive test names**: `feature_test.sql` not `test1.sql`
4. **Test edge cases**: Cover error conditions and boundaries
5. **Run tests often**: Integrate into development workflow
6. **Set coverage goals**: Aim for 80%+ coverage
7. **Review coverage reports**: Identify untested code paths

## Troubleshooting

### "No tests discovered"

- Check test files end with `_test.sql`
- Ensure tests are in specified path
- Use `--verbose` to see discovery process

### "Connection refused"

- Verify PostgreSQL is running
- Check connection parameters
- Test with `psql` first

### "Coverage is 0%"

- Ensure source files are in same directory as tests
- Check that tests actually execute (use RAISE NOTICE)
- Use `--verbose` to see instrumentation

### "Tests timeout"

- Increase `--timeout` value
- Check for infinite loops in code
- Verify database isn't overloaded

## Next Steps

1. Try the `simple/` example first
2. Adapt for your project structure
3. Set up CI/CD with `ci-integration/` examples
4. Explore coverage reports
5. Iterate to improve coverage

## Contributing Examples

Have a useful example? Contribute!

1. Create directory with descriptive name
2. Add source files, test files, README
3. Test the example works
4. Update this README with new example
5. Submit pull request

## Support

- Documentation: `docs/`
- Issues: GitHub Issues
- Discussions: GitHub Discussions
