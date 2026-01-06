package report

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/pashagolub/pgcov/internal/coverage"
)

func TestHTMLReporter_Format(t *testing.T) {
	// Create test coverage data
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Files: map[string]coverage.FileHits{
			"test.sql": {
				1: 5,
				2: 3,
				3: 0,
			},
		},
	}

	// Create reporter
	reporter := NewHTMLReporter()

	// Test Format method
	t.Run("Format", func(t *testing.T) {
		var buf bytes.Buffer
		err := reporter.Format(cov, &buf)
		if err != nil {
			t.Fatalf("Format failed: %v", err)
		}

		output := buf.String()

		// Verify HTML structure
		requiredElements := []string{
			"<!DOCTYPE html>",
			"<html",
			"<head>",
			"<body>",
			"</html>",
			"Coverage Report",
			"pgcov",
		}

		for _, elem := range requiredElements {
			if !strings.Contains(output, elem) {
				t.Errorf("Missing required HTML element: %s", elem)
			}
		}

		// Verify file is present
		if !strings.Contains(output, "test.sql") {
			t.Error("File test.sql not found in HTML output")
		}

		// Verify coverage indicators (cov0, cov1-cov8, or class names)
		hasCoverage := strings.Contains(output, "cov0") ||
			strings.Contains(output, "cov1") ||
			strings.Contains(output, "not-tracked")
		if !hasCoverage {
			t.Error("Missing coverage indicators (cov0, cov1, etc.)")
		}
	})

	// Test FormatString method
	t.Run("FormatString", func(t *testing.T) {
		output, err := reporter.FormatString(cov)
		if err != nil {
			t.Fatalf("FormatString failed: %v", err)
		}

		// Verify HTML structure
		if !strings.Contains(output, "<!DOCTYPE html>") {
			t.Error("Missing DOCTYPE declaration")
		}

		if !strings.Contains(output, "</html>") {
			t.Error("Missing closing html tag")
		}
	})

	// Test Name method
	t.Run("Name", func(t *testing.T) {
		name := reporter.Name()
		if name != "html" {
			t.Errorf("Name mismatch: got %s, want html", name)
		}
	})
}

func TestHTMLReporter_MultipleFiles(t *testing.T) {
	// Create coverage data with multiple files
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Files: map[string]coverage.FileHits{
			"auth.sql": {
				10: 2,
				11: 0,
				12: 1,
			},
			"user.sql": {
				1: 5,
				2: 3,
			},
		},
	}

	reporter := NewHTMLReporter()
	var buf bytes.Buffer
	err := reporter.Format(cov, &buf)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Verify both files are present
	if !strings.Contains(output, "auth.sql") {
		t.Error("Missing auth.sql in output")
	}

	if !strings.Contains(output, "user.sql") {
		t.Error("Missing user.sql in output")
	}
}

func TestHTMLReporter_EmptyCoverage(t *testing.T) {
	// Create empty coverage data
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Files:     map[string]coverage.FileHits{},
	}

	reporter := NewHTMLReporter()
	var buf bytes.Buffer
	err := reporter.Format(cov, &buf)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Should still produce valid HTML
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("Missing DOCTYPE declaration")
	}

	if !strings.Contains(output, "Coverage Report") {
		t.Error("Missing report title")
	}
}

func TestHTMLReporter_CoveragePercentages(t *testing.T) {
	// Create coverage data with known percentages
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Files: map[string]coverage.FileHits{
			"high_coverage.sql": {
				1: 10,
				2: 5,
				3: 1,
				4: 0, // 75% coverage (3/4)
			},
			"low_coverage.sql": {
				1: 0,
				2: 0,
				3: 1, // 33.33% coverage (1/3)
			},
		},
	}

	reporter := NewHTMLReporter()
	var buf bytes.Buffer
	err := reporter.Format(cov, &buf)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	output := buf.String()

	// Verify percentage values are present
	if !strings.Contains(output, "75.00%") {
		t.Error("Missing high_coverage.sql percentage (75%)")
	}

	if !strings.Contains(output, "33.33%") {
		t.Error("Missing low_coverage.sql percentage (33.33%)")
	}
}

func TestHTMLReporter_CSSPresent(t *testing.T) {
	// Verify CSS is included
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Files: map[string]coverage.FileHits{
			"test.sql": {1: 1},
		},
	}

	reporter := NewHTMLReporter()
	output, err := reporter.FormatString(cov)
	if err != nil {
		t.Fatalf("FormatString failed: %v", err)
	}

	// Verify CSS is present
	if !strings.Contains(output, "<style>") {
		t.Error("Missing <style> tag")
	}

	if !strings.Contains(output, "</style>") {
		t.Error("Missing </style> closing tag")
	}

	// Verify some key CSS classes
	cssClasses := []string{
		".cov0",
		".cov1",
		".source-line",
		".line-num",
	}

	for _, class := range cssClasses {
		if !strings.Contains(output, class) {
			t.Errorf("Missing CSS class: %s", class)
		}
	}
}

func TestHTMLReporter_LineCoverage(t *testing.T) {
	// Test line coverage indicators
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Files: map[string]coverage.FileHits{
			"test.sql": {
				1: 5,  // covered
				2: 0,  // uncovered
				3: 10, // covered
			},
		},
	}

	reporter := NewHTMLReporter()
	output, err := reporter.FormatString(cov)
	if err != nil {
		t.Fatalf("FormatString failed: %v", err)
	}

	// Verify line numbers are present (new format uses line-num class)
	if !strings.Contains(output, `class="line-num"`) {
		t.Error("Missing line-num class")
	}

	// Verify hit count display (new format: just numbers like "5", "0", "10")
	// Look for line-count spans with numbers
	if !strings.Contains(output, `class="line-count"`) {
		t.Error("Missing line-count class")
	}

	// Check for coverage classes
	if !strings.Contains(output, "cov5") && !strings.Contains(output, "cov8") {
		t.Error("Missing coverage class for line 1 (5 hits)")
	}

	if !strings.Contains(output, "cov0") {
		t.Error("Missing cov0 class for uncovered line")
	}
}

func TestHTMLReporter_ValidHTML5(t *testing.T) {
	// Test HTML5 validity
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Files: map[string]coverage.FileHits{
			"test.sql": {1: 1},
		},
	}

	reporter := NewHTMLReporter()
	output, err := reporter.FormatString(cov)
	if err != nil {
		t.Fatalf("FormatString failed: %v", err)
	}

	// Verify HTML5 doctype
	if !strings.HasPrefix(strings.TrimSpace(output), "<!DOCTYPE html>") {
		t.Error("HTML5 DOCTYPE not at beginning of document")
	}

	// Verify meta charset
	if !strings.Contains(output, `<meta charset="UTF-8">`) {
		t.Error("Missing UTF-8 charset declaration")
	}

	// Verify viewport meta tag
	if !strings.Contains(output, `<meta name="viewport"`) {
		t.Error("Missing viewport meta tag")
	}

	// Verify lang attribute
	if !strings.Contains(output, `<html lang="en">`) {
		t.Error("Missing lang attribute on html element")
	}
}

func TestHTMLReporter_EscapeHTML(t *testing.T) {
	// Test HTML escaping for file names with special characters
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Files: map[string]coverage.FileHits{
			"test<script>.sql": {1: 1},
			"file&name.sql":    {1: 1},
		},
	}

	reporter := NewHTMLReporter()
	output, err := reporter.FormatString(cov)
	if err != nil {
		t.Fatalf("FormatString failed: %v", err)
	}

	// Verify HTML entities are escaped
	if strings.Contains(output, "test<script>.sql") && !strings.Contains(output, "&lt;") {
		t.Error("HTML special characters not escaped properly (<)")
	}

	if strings.Contains(output, "file&name.sql") && !strings.Contains(output, "&amp;") {
		t.Error("HTML special characters not escaped properly (&)")
	}
}

func TestHTMLReporter_SummarySection(t *testing.T) {
	// Test summary section content
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Files: map[string]coverage.FileHits{
			"test1.sql": {1: 1, 2: 0},
			"test2.sql": {1: 1, 2: 1},
		},
	}

	reporter := NewHTMLReporter()
	output, err := reporter.FormatString(cov)
	if err != nil {
		t.Fatalf("FormatString failed: %v", err)
	}

	// Verify summary bar exists
	if !strings.Contains(output, `class="summary-bar"`) {
		t.Error("Missing summary bar")
	}

	// Verify summary stats
	if !strings.Contains(output, "Total Coverage") {
		t.Error("Missing total coverage stat")
	}
}

func TestHTMLReporter_CoverageClasses(t *testing.T) {
	// Test coverage class assignments (cov0-cov8)
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Files: map[string]coverage.FileHits{
			"high.sql": {
				1: 1, 2: 1, 3: 1, 4: 1, 5: 1,
				6: 1, 7: 1, 8: 1, 9: 1, 10: 0,
			}, // 90% coverage - should use cov1-cov8
			"medium.sql": {
				1: 3, 2: 3, 3: 3, 4: 0, 5: 0,
			}, // 60% coverage - should use cov3
			"low.sql": {
				1: 5, 2: 0, 3: 0, 4: 0, 5: 0,
			}, // 20% coverage - should use cov5 and cov0
		},
	}

	reporter := NewHTMLReporter()
	output, err := reporter.FormatString(cov)
	if err != nil {
		t.Fatalf("FormatString failed: %v", err)
	}

	// Count occurrences of coverage classes (Go-style)
	// Note: we can't be too specific as the order may vary
	if !strings.Contains(output, `class="source-line cov0"`) &&
		!strings.Contains(output, `cov0`) {
		t.Error("Missing cov0 coverage class for uncovered lines")
	}

	if !strings.Contains(output, `cov1`) && !strings.Contains(output, `cov3`) && !strings.Contains(output, `cov5`) {
		t.Error("Missing covered line classes (cov1, cov3, cov5, etc.)")
	}
}

func TestHTMLReporter_Footer(t *testing.T) {
	// Test that HTML properly closes
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Files:     map[string]coverage.FileHits{},
	}

	reporter := NewHTMLReporter()
	output, err := reporter.FormatString(cov)
	if err != nil {
		t.Fatalf("FormatString failed: %v", err)
	}

	// Verify proper HTML closing
	if !strings.Contains(output, "</body>") {
		t.Error("Missing closing body tag")
	}

	if !strings.Contains(output, "</html>") {
		t.Error("Missing closing html tag")
	}
}
