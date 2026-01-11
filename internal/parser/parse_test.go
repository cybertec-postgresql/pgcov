package parser

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cybertec-postgresql/pgcov/internal/discovery"
)

func TestParse_ValidSQL(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		wantStmt int
	}{
		{
			name:     "single SELECT",
			sql:      "SELECT 1;",
			wantStmt: 1,
		},
		{
			name:     "multi statement",
			sql:      "SELECT 1; SELECT 2;",
			wantStmt: 2,
		},
		{
			name:     "CREATE TABLE",
			sql:      "CREATE TABLE users (id INT, name TEXT);",
			wantStmt: 1,
		},
		{
			name:     "INSERT",
			sql:      "INSERT INTO users VALUES (1, 'Alice');",
			wantStmt: 1,
		},
		{
			name:     "UPDATE",
			sql:      "UPDATE users SET name = 'Bob' WHERE id = 1;",
			wantStmt: 1,
		},
		{
			name:     "DELETE",
			sql:      "DELETE FROM users WHERE id = 1;",
			wantStmt: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.sql")
			if err := os.WriteFile(tmpFile, []byte(tt.sql), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			file := &discovery.DiscoveredFile{
				Path: tmpFile,
				Type: discovery.FileTypeSource,
			}

			parsed, err := Parse(file)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if parsed == nil {
				t.Fatal("Parse() returned nil")
			}

			if len(parsed.Statements) != tt.wantStmt {
				t.Errorf("Parse() got %d statements, want %d", len(parsed.Statements), tt.wantStmt)
			}
		})
	}
}

func TestParse_SyntaxError(t *testing.T) {
	tests := []struct {
		name string
		sql  string
	}{
		{
			name: "invalid SELECT",
			sql:  "SELECT FROM;",
		},
		{
			name: "unclosed parenthesis",
			sql:  "SELECT * FROM users WHERE (id = 1;",
		},
		{
			name: "invalid CREATE",
			sql:  "CREATE TABLE;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.sql")
			if err := os.WriteFile(tmpFile, []byte(tt.sql), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			file := &discovery.DiscoveredFile{
				Path: tmpFile,
				Type: discovery.FileTypeSource,
			}

			parsed, err := Parse(file)
			if err == nil {
				t.Errorf("Parse() expected error, got nil")
			}

			if parsed != nil {
				t.Errorf("Parse() expected nil parsed result on error, got %v", parsed)
			}

			// Check error type
			var parseErr *ParseError
			if !errors.As(err, &parseErr) {
				t.Errorf("Parse() error type = %T, want *errors.ParseError", err)
			}
		})
	}
}

func TestParse_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty.sql")
	if err := os.WriteFile(tmpFile, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	file := &discovery.DiscoveredFile{
		Path: tmpFile,
		Type: discovery.FileTypeSource,
	}

	parsed, err := Parse(file)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	if len(parsed.Statements) != 0 {
		t.Errorf("Parse() got %d statements, want 0", len(parsed.Statements))
	}
}

func TestParse_CommentsOnly(t *testing.T) {
	sql := `-- This is a comment
-- Another comment
/* Block comment */`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "comments.sql")
	if err := os.WriteFile(tmpFile, []byte(sql), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	file := &discovery.DiscoveredFile{
		Path: tmpFile,
		Type: discovery.FileTypeSource,
	}

	parsed, err := Parse(file)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	// Comments should not produce statements
	if len(parsed.Statements) != 0 {
		t.Errorf("Parse() got %d statements, want 0", len(parsed.Statements))
	}
}

func TestParse_MixedStatements(t *testing.T) {
	sql := `
CREATE TABLE users (id INT PRIMARY KEY, name TEXT);

INSERT INTO users VALUES (1, 'Alice'), (2, 'Bob');

SELECT * FROM users;

UPDATE users SET name = 'Charlie' WHERE id = 1;

DELETE FROM users WHERE id = 2;
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "mixed.sql")
	if err := os.WriteFile(tmpFile, []byte(sql), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	file := &discovery.DiscoveredFile{
		Path: tmpFile,
		Type: discovery.FileTypeSource,
	}

	parsed, err := Parse(file)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	expectedStmts := 5
	if len(parsed.Statements) != expectedStmts {
		t.Errorf("Parse() got %d statements, want %d", len(parsed.Statements), expectedStmts)
	}
}

func TestParseFile(t *testing.T) {
	sql := "SELECT 42;"
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.sql")
	if err := os.WriteFile(tmpFile, []byte(sql), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	parsed, err := ParseFile(tmpFile)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	if parsed == nil {
		t.Fatal("ParseFile() returned nil")
	}

	if len(parsed.Statements) != 1 {
		t.Errorf("ParseFile() got %d statements, want 1", len(parsed.Statements))
	}
}

func TestParseSQL(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{
			name:    "valid SQL",
			sql:     "SELECT 1;",
			wantErr: false,
		},
		{
			name:    "invalid SQL",
			sql:     "SELECT FROM;",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSQL(tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSQL() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && result == nil {
				t.Error("ParseSQL() returned nil result for valid SQL")
			}

			if tt.wantErr {
				var parseErr *ParseError
				if !errors.As(err, &parseErr) {
					t.Errorf("ParseSQL() error type = %T, want *errors.ParseError", err)
				}
			}
		})
	}
}

func TestParse_PLpgSQL(t *testing.T) {
	sql := `
CREATE OR REPLACE FUNCTION add_numbers(a INT, b INT)
RETURNS INT AS $$
BEGIN
    RETURN a + b;
END;
$$ LANGUAGE plpgsql;
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "func.sql")
	if err := os.WriteFile(tmpFile, []byte(sql), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	file := &discovery.DiscoveredFile{
		Path: tmpFile,
		Type: discovery.FileTypeSource,
	}

	parsed, err := Parse(file)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	if len(parsed.Statements) != 1 {
		t.Errorf("Parse() got %d statements, want 1", len(parsed.Statements))
	}

	// Check that the statement contains LANGUAGE plpgsql
	if !strings.Contains(parsed.Statements[0].RawSQL, "plpgsql") {
		t.Error("Parse() did not preserve PL/pgSQL function definition")
	}
}

func TestParse_FileNotFound(t *testing.T) {
	file := &discovery.DiscoveredFile{
		Path: "/nonexistent/file.sql",
		Type: discovery.FileTypeSource,
	}

	parsed, err := Parse(file)
	if err == nil {
		t.Errorf("Parse() expected error for nonexistent file, got nil")
	}

	if parsed != nil {
		t.Errorf("Parse() expected nil result, got %v", parsed)
	}
}
