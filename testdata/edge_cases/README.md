# Edge Case Test Fixtures

This directory contains SQL fixtures for testing edge cases and error handling in pgcov.

## Files

### syntax_error.sql
Contains intentional SQL syntax errors for testing error handling:
- Missing FROM clause
- Unclosed parenthesis
- Invalid CREATE statement
- Incomplete INSERT

**Purpose**: Verify that the parser correctly detects and reports syntax errors.

### empty.sql
An empty SQL file with no content.

**Purpose**: Verify that empty files are handled gracefully without errors.

### comments_only.sql
A file containing only SQL comments (both line comments and block comments).

**Purpose**: Verify that files with only comments produce no executable statements or coverage points.

### large.sql
A large SQL file with 50+ INSERT statements and various operations.

**Purpose**: Test performance with larger files and verify that the system can handle files with many statements.

### mixed_statements.sql
Contains a variety of SQL statement types:
- DDL (CREATE, ALTER, DROP)
- DML (INSERT, UPDATE, DELETE)
- Queries (SELECT)
- Indexes

**Purpose**: Verify that different statement types are all handled correctly by the parser and instrumenter.

## Usage

These fixtures are used by unit tests to verify edge case handling:

```go
func TestParser_EdgeCases(t *testing.T) {
    // Test syntax error handling
    _, err := parser.ParseFile("testdata/edge_cases/syntax_error.sql")
    if err == nil {
        t.Error("Expected error for syntax_error.sql")
    }
    
    // Test empty file
    parsed, err := parser.ParseFile("testdata/edge_cases/empty.sql")
    if err != nil || len(parsed.Statements) != 0 {
        t.Error("Empty file should parse with 0 statements")
    }
}
```

## Adding New Edge Cases

When adding new edge case fixtures:
1. Create the SQL file with descriptive name
2. Add comments explaining the edge case
3. Document the file in this README
4. Create corresponding unit tests
