package instrument

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cybertec-postgresql/pgcov/internal/discovery"
	"github.com/cybertec-postgresql/pgcov/internal/parser"
)

func TestInstrumentWithLexer(t *testing.T) {
	sql := `CREATE OR REPLACE FUNCTION get_grade(score INT)
RETURNS TEXT AS $$
BEGIN
    IF score >= 90 THEN
        RETURN 'A';
    ELSIF score >= 80 THEN
        RETURN 'B';
    ELSIF score >= 70 THEN
        RETURN 'C';
    ELSE
        RETURN 'F';
    END IF;
END;
$$ LANGUAGE plpgsql;`

	res, err := parser.ParseSQL(sql)
	if err != nil {
		t.Fatalf("ParseSQL() error = %v", err)
	}
	if len(res.Stmts) == 0 {
		t.Fatal("ParseSQL() returned no statements")
	}
	stmt := &parser.Statement{
		RawSQL: sql,
		Node:   res.Stmts[0].GetStmt(),
	}

	instrumentedSQL, coveragePoints := instrumentWithLexer(stmt, "test.sql")
	if instrumentedSQL == "" {
		t.Error("instrumentWithLexer() returned empty instrumented SQL")
	}
	if len(coveragePoints) == 0 {
		t.Error("instrumentWithLexer() returned no coverage points")
	}

	// Should have NOTIFY calls injected
	if !strings.Contains(instrumentedSQL, "pg_notify") {
		t.Error("instrumentWithLexer() did not inject NOTIFY calls")
	}
	t.Log(instrumentedSQL)
}

func TestInstrument_BasicSQL(t *testing.T) {
	sql := "SELECT 1;"

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.sql")
	if err := os.WriteFile(tmpFile, []byte(sql), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	file := &discovery.DiscoveredFile{
		Path:         tmpFile,
		RelativePath: "test.sql",
		Type:         discovery.FileTypeSource,
	}

	parsed, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	instrumented, err := GenerateCoverageInstrument(parsed)
	if err != nil {
		t.Fatalf("Instrument() error = %v", err)
	}

	if instrumented == nil {
		t.Fatal("Instrument() returned nil")
	}

	if len(instrumented.Locations) == 0 {
		t.Error("Instrument() produced no coverage points")
	}

	// Verify original is preserved
	if instrumented.Original != parsed {
		t.Error("Instrument() did not preserve original parsed SQL")
	}
}

func TestInstrument_PLpgSQLFunction(t *testing.T) {
	sql := `CREATE OR REPLACE FUNCTION add_numbers(a INT, b INT)
RETURNS INT AS $$
BEGIN
    RETURN a + b;
END;
$$ LANGUAGE plpgsql;`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "func.sql")
	if err := os.WriteFile(tmpFile, []byte(sql), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	file := &discovery.DiscoveredFile{
		Path:         tmpFile,
		RelativePath: "func.sql",
		Type:         discovery.FileTypeSource,
	}

	parsed, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	instrumented, err := GenerateCoverageInstrument(parsed)
	if err != nil {
		t.Fatalf("Instrument() error = %v", err)
	}

	// Should have NOTIFY calls injected
	if !strings.Contains(instrumented.InstrumentedText, "pg_notify") {
		t.Error("Instrument() did not inject NOTIFY calls for PL/pgSQL function")
	}

	// Should have coverage points
	if len(instrumented.Locations) == 0 {
		t.Error("Instrument() produced no coverage points for PL/pgSQL function")
	}
}

func TestInstrument_MultipleStatements(t *testing.T) {
	sql := `SELECT 1;
SELECT 2;
SELECT 3;`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "multi.sql")
	if err := os.WriteFile(tmpFile, []byte(sql), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	file := &discovery.DiscoveredFile{
		Path:         tmpFile,
		RelativePath: "multi.sql",
		Type:         discovery.FileTypeSource,
	}

	parsed, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	instrumented, err := GenerateCoverageInstrument(parsed)
	if err != nil {
		t.Fatalf("Instrument() error = %v", err)
	}

	// Should have multiple coverage points (one per statement line)
	if len(instrumented.Locations) < 3 {
		t.Errorf("Instrument() got %d coverage points, want at least 3", len(instrumented.Locations))
	}
}

func TestInstrument_EmptyFile(t *testing.T) {
	sql := ""

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty.sql")
	if err := os.WriteFile(tmpFile, []byte(sql), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	file := &discovery.DiscoveredFile{
		Path:         tmpFile,
		RelativePath: "empty.sql",
		Type:         discovery.FileTypeSource,
	}

	parsed, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	instrumented, err := GenerateCoverageInstrument(parsed)
	if err != nil {
		t.Fatalf("Instrument() error = %v", err)
	}

	if len(instrumented.Locations) != 0 {
		t.Errorf("Instrument() got %d coverage points for empty file, want 0", len(instrumented.Locations))
	}
}

func TestInstrument_CommentsOnly(t *testing.T) {
	sql := `-- Comment line 1
-- Comment line 2`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "comments.sql")
	if err := os.WriteFile(tmpFile, []byte(sql), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	file := &discovery.DiscoveredFile{
		Path:         tmpFile,
		RelativePath: "comments.sql",
		Type:         discovery.FileTypeSource,
	}

	parsed, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	instrumented, err := GenerateCoverageInstrument(parsed)
	if err != nil {
		t.Fatalf("Instrument() error = %v", err)
	}

	// Comments should not produce coverage points
	if len(instrumented.Locations) != 0 {
		t.Errorf("Instrument() got %d coverage points for comments, want 0", len(instrumented.Locations))
	}
}

func TestInstrument_SignalIDFormat(t *testing.T) {
	sql := "SELECT 1;"

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.sql")
	if err := os.WriteFile(tmpFile, []byte(sql), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	file := &discovery.DiscoveredFile{
		Path:         tmpFile,
		RelativePath: "test.sql",
		Type:         discovery.FileTypeSource,
	}

	parsed, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	instrumented, err := GenerateCoverageInstrument(parsed)
	if err != nil {
		t.Fatalf("Instrument() error = %v", err)
	}

	// Verify signal ID format (should be file:line or file:line:branch)
	for _, loc := range instrumented.Locations {
		if loc.SignalID == "" {
			t.Error("Instrument() produced empty SignalID")
		}

		// Signal ID should contain file and line
		if !strings.Contains(loc.SignalID, ":") {
			t.Errorf("Instrument() SignalID %q doesn't contain separator", loc.SignalID)
		}
	}
}

func TestInstrumentBatch(t *testing.T) {
	files := []string{
		"SELECT 1;",
		"SELECT 2;",
		"SELECT 3;",
	}

	tmpDir := t.TempDir()
	var parsedFiles []*parser.ParsedSQL

	for i, content := range files {
		tmpFile := filepath.Join(tmpDir, string(rune('a'+i))+".sql")
		if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}

		file := &discovery.DiscoveredFile{
			Path:         tmpFile,
			RelativePath: string(rune('a'+i)) + ".sql",
			Type:         discovery.FileTypeSource,
		}

		parsed, err := parser.Parse(file)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		parsedFiles = append(parsedFiles, parsed)
	}

	instrumented, err := GenerateCoverageInstruments(parsedFiles)
	if err != nil {
		t.Fatalf("InstrumentBatch() error = %v", err)
	}

	if len(instrumented) != len(files) {
		t.Errorf("InstrumentBatch() got %d results, want %d", len(instrumented), len(files))
	}
}

func TestGetCoveragePointBySignal(t *testing.T) {
	sql := "SELECT 1;"

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.sql")
	if err := os.WriteFile(tmpFile, []byte(sql), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	file := &discovery.DiscoveredFile{
		Path:         tmpFile,
		RelativePath: "test.sql",
		Type:         discovery.FileTypeSource,
	}

	parsed, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	instrumented, err := GenerateCoverageInstrument(parsed)
	if err != nil {
		t.Fatalf("Instrument() error = %v", err)
	}

	if len(instrumented.Locations) == 0 {
		t.Fatal("No coverage points generated")
	}

	// Test finding a coverage point by signal
	signal := instrumented.Locations[0].SignalID
	cp := GetCoveragePointBySignal(instrumented, signal)
	if cp == nil {
		t.Errorf("GetCoveragePointBySignal() returned nil for signal %q", signal)
	} else if cp.SignalID != signal {
		t.Errorf("GetCoveragePointBySignal() got signal %q, want %q", cp.SignalID, signal)
	}

	// Test non-existent signal
	cp = GetCoveragePointBySignal(instrumented, "nonexistent:signal")
	if cp != nil {
		t.Errorf("GetCoveragePointBySignal() expected nil for nonexistent signal, got %v", cp)
	}
}

func TestInstrument_NilInput(t *testing.T) {
	_, err := GenerateCoverageInstrument(nil)
	if err == nil {
		t.Error("Instrument() expected error for nil input, got nil")
	}
}

func TestInstrument_DifferentStatementTypes(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		wantLoc bool
	}{
		{
			name:    "SELECT",
			sql:     "SELECT * FROM users;",
			wantLoc: true,
		},
		{
			name:    "INSERT",
			sql:     "INSERT INTO users VALUES (1, 'Alice');",
			wantLoc: true,
		},
		{
			name:    "UPDATE",
			sql:     "UPDATE users SET name = 'Bob';",
			wantLoc: true,
		},
		{
			name:    "DELETE",
			sql:     "DELETE FROM users WHERE id = 1;",
			wantLoc: true,
		},
		{
			name:    "CREATE TABLE",
			sql:     "CREATE TABLE users (id INT);",
			wantLoc: true,
		},
		{
			name:    "DROP TABLE",
			sql:     "DROP TABLE users;",
			wantLoc: true,
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
				Path:         tmpFile,
				RelativePath: "test.sql",
				Type:         discovery.FileTypeSource,
			}

			parsed, err := parser.Parse(file)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			instrumented, err := GenerateCoverageInstrument(parsed)
			if err != nil {
				t.Fatalf("Instrument() error = %v", err)
			}

			hasLocations := len(instrumented.Locations) > 0
			if hasLocations != tt.wantLoc {
				t.Errorf("Instrument() hasLocations = %v, want %v", hasLocations, tt.wantLoc)
			}
		})
	}
}
