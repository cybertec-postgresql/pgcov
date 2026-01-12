package report

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/cybertec-postgresql/pgcov/internal/coverage"
)

func TestLCOVReporter_Format(t *testing.T) {
	// Create test coverage data with positions
	// LCOV reporter converts positions to lines by reading source files
	// For tests without real files, it falls back to outputting positions as line numbers
	timestamp, _ := time.Parse(time.RFC3339, "2026-01-05T10:00:00Z")
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: timestamp,
		Positions: map[string]coverage.PositionHits{
			"test.sql": {
				"1:10": 5,
				"2:15": 3,
				"3:20": 0,
			},
		},
	}

	// Create reporter
	reporter := NewLCOVReporter()

	// Test Format method
	t.Run("Format", func(t *testing.T) {
		var buf bytes.Buffer
		err := reporter.Format(cov, &buf)
		if err != nil {
			t.Fatalf("Format failed: %v", err)
		}

		output := buf.String()

		// Verify LCOV format structure
		if !strings.Contains(output, "SF:test.sql") {
			t.Error("Missing SF: (source file) line")
		}

		if !strings.Contains(output, "DA:") {
			t.Error("Missing DA: (data) lines")
		}

		if !strings.Contains(output, "LF:") {
			t.Error("Missing LF: (lines found) line")
		}

		if !strings.Contains(output, "LH:") {
			t.Error("Missing LH: (lines hit) line")
		}

		if !strings.Contains(output, "end_of_record") {
			t.Error("Missing end_of_record marker")
		}
	})

	// Test FormatString method
	t.Run("FormatString", func(t *testing.T) {
		output, err := reporter.FormatString(cov)
		if err != nil {
			t.Fatalf("FormatString failed: %v", err)
		}

		// Verify LCOV format structure
		if !strings.Contains(output, "SF:test.sql") {
			t.Error("Missing SF: (source file) line")
		}

		if !strings.Contains(output, "end_of_record") {
			t.Error("Missing end_of_record marker")
		}
	})

	// Test Name method
	t.Run("Name", func(t *testing.T) {
		name := reporter.Name()
		if name != "lcov" {
			t.Errorf("Name mismatch: got %s, want lcov", name)
		}
	})
}

func TestLCOVReporter_MultipleFiles(t *testing.T) {
	// Create coverage data with multiple files
	timestamp, _ := time.Parse(time.RFC3339, "2026-01-05T10:00:00Z")
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: timestamp,
		Positions: map[string]coverage.PositionHits{
			"auth.sql": {
				"10:20": 2,
				"11:15": 0,
				"12:25": 1,
			},
			"user.sql": {
				"1:10": 5,
				"2:15": 3,
			},
		},
	}

	reporter := NewLCOVReporter()
	var buf bytes.Buffer
	err := reporter.Format(cov, &buf)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Verify both files are present
	if !strings.Contains(output, "SF:auth.sql") {
		t.Error("Missing auth.sql in output")
	}

	if !strings.Contains(output, "SF:user.sql") {
		t.Error("Missing user.sql in output")
	}

	// Count end_of_record markers (should be 2)
	count := strings.Count(output, "end_of_record")
	if count != 2 {
		t.Errorf("Expected 2 end_of_record markers, got %d", count)
	}
}

func TestLCOVReporter_EmptyCoverage(t *testing.T) {
	// Create empty coverage data
	timestamp, _ := time.Parse(time.RFC3339, "2026-01-05T10:00:00Z")
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: timestamp,
		Positions: map[string]coverage.PositionHits{},
	}

	reporter := NewLCOVReporter()
	var buf bytes.Buffer
	err := reporter.Format(cov, &buf)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Empty coverage should produce empty output
	if output != "" {
		t.Errorf("Expected empty output, got: %s", output)
	}
}

func TestLCOVReporter_PositionCounts(t *testing.T) {
	// Create coverage data with specific position counts
	timestamp, _ := time.Parse(time.RFC3339, "2026-01-05T10:00:00Z")
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: timestamp,
		Positions: map[string]coverage.PositionHits{
			"test.sql": {
				"1:10":  10,
				"2:15":  5,
				"3:20":  0,
				"4:25":  0,
				"5:30":  1,
				"10:35": 20,
			},
		},
	}

	reporter := NewLCOVReporter()
	var buf bytes.Buffer
	err := reporter.Format(cov, &buf)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Verify LF (lines found) = 6 total positions
	if !strings.Contains(output, "LF:6") {
		t.Error("Expected LF:6 (6 total instrumented positions)")
	}

	// Verify LH (lines hit) = 4 covered positions
	if !strings.Contains(output, "LH:4") {
		t.Error("Expected LH:4 (4 covered positions)")
	}
}

func TestLCOVReporter_DeterministicOutput(t *testing.T) {
	// Create coverage data
	timestamp, _ := time.Parse(time.RFC3339, "2026-01-05T10:00:00Z")
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: timestamp,
		Positions: map[string]coverage.PositionHits{
			"b.sql": {"3:10": 1, "1:15": 2, "2:20": 0},
			"a.sql": {"5:10": 3, "2:15": 1, "8:20": 0},
		},
	}

	reporter := NewLCOVReporter()

	// Format twice
	var buf1, buf2 bytes.Buffer
	err1 := reporter.Format(cov, &buf1)
	err2 := reporter.Format(cov, &buf2)

	if err1 != nil || err2 != nil {
		t.Fatalf("Format failed: %v, %v", err1, err2)
	}

	// Verify outputs are identical (deterministic)
	if buf1.String() != buf2.String() {
		t.Error("LCOV output is not deterministic")
	}

	// Verify files are sorted alphabetically
	output := buf1.String()
	aIndex := strings.Index(output, "SF:a.sql")
	bIndex := strings.Index(output, "SF:b.sql")

	if aIndex == -1 || bIndex == -1 {
		t.Fatal("Files not found in output")
	}

	if aIndex > bIndex {
		t.Error("Files not sorted alphabetically (expected a.sql before b.sql)")
	}
}

func TestLCOVReporter_FormatCompliance(t *testing.T) {
	// Test LCOV format specification compliance
	timestamp, _ := time.Parse(time.RFC3339, "2026-01-05T10:00:00Z")
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: timestamp,
		Positions: map[string]coverage.PositionHits{
			"spec_test.sql": {
				"1:10": 1,
			},
		},
	}

	reporter := NewLCOVReporter()
	output, err := reporter.FormatString(cov)
	if err != nil {
		t.Fatalf("FormatString failed: %v", err)
	}

	// Verify LCOV format structure according to spec
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Expected order: SF, DA lines, LF, LH, end_of_record
	if len(lines) < 5 {
		t.Fatalf("Expected at least 5 lines, got %d", len(lines))
	}

	// First line should be SF:
	if !strings.HasPrefix(lines[0], "SF:") {
		t.Errorf("First line should start with SF:, got: %s", lines[0])
	}

	// DA lines should come after SF
	foundDA := false
	for i := 1; i < len(lines)-3; i++ {
		if strings.HasPrefix(lines[i], "DA:") {
			foundDA = true
			break
		}
	}
	if !foundDA {
		t.Error("No DA: (data) lines found")
	}

	// LF should come before LH
	lfIndex := -1
	lhIndex := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "LF:") {
			lfIndex = i
		}
		if strings.HasPrefix(line, "LH:") {
			lhIndex = i
		}
	}

	if lfIndex == -1 || lhIndex == -1 {
		t.Error("Missing LF or LH line")
	}

	if lfIndex >= lhIndex {
		t.Error("LF should come before LH")
	}

	// Last line should be end_of_record
	if lines[len(lines)-1] != "end_of_record" {
		t.Errorf("Last line should be end_of_record, got: %s", lines[len(lines)-1])
	}
}

func TestLCOVReporter_HitCountFormat(t *testing.T) {
	// Test that hit counts are formatted correctly
	timestamp, _ := time.Parse(time.RFC3339, "2026-01-05T10:00:00Z")
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: timestamp,
		Positions: map[string]coverage.PositionHits{
			"test.sql": {
				"1:10": 0,
				"2:15": 1,
				"3:20": 100,
				"4:25": 9999,
			},
		},
	}

	reporter := NewLCOVReporter()
	output, err := reporter.FormatString(cov)
	if err != nil {
		t.Fatalf("FormatString failed: %v", err)
	}

	// Verify basic structure is present
	if !strings.Contains(output, "DA:") {
		t.Error("Missing DA: lines")
	}

	// Verify hit count values appear in output
	if !strings.Contains(output, ",0") {
		t.Error("Missing zero hit count")
	}
	if !strings.Contains(output, ",1") {
		t.Error("Missing 1 hit count")
	}
	if !strings.Contains(output, ",100") {
		t.Error("Missing 100 hit count")
	}
	if !strings.Contains(output, ",9999") {
		t.Error("Missing 9999 hit count")
	}
}
