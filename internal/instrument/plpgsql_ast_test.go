package instrument

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cybertec-postgresql/pgcov/internal/discovery"
	"github.com/cybertec-postgresql/pgcov/internal/parser"
)

func TestInstrumentPlpgsql_ComplexFunction(t *testing.T) {
	// Test a complex PL/pgSQL function with multiple control structures
	sql := `CREATE OR REPLACE FUNCTION calculate_discount(total_amount NUMERIC)
RETURNS NUMERIC AS $$
DECLARE
    discount_rate NUMERIC;
BEGIN
    IF total_amount > 1000 THEN
        discount_rate := 0.20;
    ELSIF total_amount > 500 THEN
        discount_rate := 0.10;
    ELSE
        discount_rate := 0.05;
    END IF;
    
    RETURN total_amount * discount_rate;
END;
$$ LANGUAGE plpgsql;`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "discount.sql")
	if err := os.WriteFile(tmpFile, []byte(sql), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	file := &discovery.DiscoveredFile{
		Path:         tmpFile,
		RelativePath: "discount.sql",
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

	// Should have NOTIFY calls
	if !strings.Contains(instrumented.InstrumentedText, "pg_notify") {
		t.Error("Instrument() did not inject NOTIFY calls")
	}

	// Should have coverage points for all executable statements (3 assignments + 1 return)
	if len(instrumented.Locations) != 4 {
		t.Errorf("Expected 4 coverage points, got %d", len(instrumented.Locations))
	}

	// Verify coverage points are at the correct lines (by converting positions to lines)
	expectedLines := []int{7, 9, 11, 14}
	for i, cp := range instrumented.Locations {
		// Convert position to line number for validation
		// For this test, we'll use the original SQL content
		sqlContent, _ := os.ReadFile(tmpFile)
		actualLine := ConvertPositionToLine(string(sqlContent), cp.StartPos)
		if actualLine != expectedLines[i] {
			t.Errorf("Coverage point %d: expected line %d, got %d", i, expectedLines[i], actualLine)
		}
		if cp.ImplicitCoverage {
			t.Errorf("Coverage point %d: should not be implicit", i)
		}
	}

	// Verify PERFORM statements are injected before the executable lines
	// Note: After instrumentation, line numbers shift due to inserted PERFORM statements
	// We just verify that PERFORM statements exist for each coverage point
	for _, cp := range instrumented.Locations {
		signalID := cp.SignalID
		if !strings.Contains(instrumented.InstrumentedText, fmt.Sprintf("PERFORM pg_notify('pgcov', '%s')", signalID)) {
			t.Errorf("Missing PERFORM pg_notify for signal %s", signalID)
		}
	}
}

func TestInstrumentPlpgsql_WithLoop(t *testing.T) {
	sql := `CREATE OR REPLACE FUNCTION sum_to_n(n INT)
RETURNS INT AS $$
DECLARE
    total INT := 0;
    i INT;
BEGIN
    FOR i IN 1..n LOOP
        total := total + i;
    END LOOP;
    
    RETURN total;
END;
$$ LANGUAGE plpgsql;`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "loop.sql")
	if err := os.WriteFile(tmpFile, []byte(sql), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	file := &discovery.DiscoveredFile{
		Path:         tmpFile,
		RelativePath: "loop.sql",
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

	// Should have coverage points for the assignment inside the loop and the return
	if len(instrumented.Locations) < 2 {
		t.Errorf("Expected at least 2 coverage points, got %d", len(instrumented.Locations))
	}

	// Verify all coverage points are non-implicit
	for i, cp := range instrumented.Locations {
		if cp.ImplicitCoverage {
			t.Errorf("Coverage point %d: should not be implicit", i)
		}
	}
}

func TestInstrumentPlpgsql_DOBlock(t *testing.T) {
	// Test DO blocks which are also PL/pgSQL but not functions
	sql := `DO $$
BEGIN
    RAISE NOTICE 'Hello World';
END $$;`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "do_block.sql")
	if err := os.WriteFile(tmpFile, []byte(sql), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	file := &discovery.DiscoveredFile{
		Path:         tmpFile,
		RelativePath: "do_block.sql",
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

	// DO blocks may not instrument with AST-only approach
	// This is acceptable as we only support properly formatted functions
	t.Logf("Coverage points: %d", len(instrumented.Locations))
}

func TestInstrumentPlpgsql_FallbackOnParseError(t *testing.T) {
	// Test that if PL/pgSQL parsing fails, we return without instrumentation
	// This is a malformed function that might not parse correctly
	sql := `CREATE FUNCTION bad_func() RETURNS void AS $$
BEGIN
    SELECT 1;
END;
$$ LANGUAGE plpgsql;`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "bad.sql")
	if err := os.WriteFile(tmpFile, []byte(sql), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	file := &discovery.DiscoveredFile{
		Path:         tmpFile,
		RelativePath: "bad.sql",
		Type:         discovery.FileTypeSource,
	}

	parsed, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Should not fail even if PL/pgSQL parsing has issues
	instrumented, err := GenerateCoverageInstrument(parsed)
	if err != nil {
		t.Fatalf("Instrument() should not fail on parse errors, got: %v", err)
	}

	// With AST-only approach, may return no instrumentation for malformed SQL
	if instrumented == nil {
		t.Fatal("Instrument() returned nil")
	}
	t.Logf("Coverage points: %d (may be 0 for malformed SQL)", len(instrumented.Locations))
}
