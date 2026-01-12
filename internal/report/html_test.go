package report

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/cybertec-postgresql/pgcov/internal/coverage"
)

func TestHTMLReporter_Format(t *testing.T) {
	// Create test coverage data with positions
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Positions: map[string]coverage.PositionHits{
			"test.sql": {
				"0:10":  5,
				"20:15": 3,
				"50:20": 0,
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
		Positions: map[string]coverage.PositionHits{
			"auth.sql": {
				"0:10":  2,
				"20:15": 0,
				"50:20": 1,
			},
			"user.sql": {
				"0:10":  5,
				"30:20": 3,
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
		Positions: map[string]coverage.PositionHits{},
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
		Positions: map[string]coverage.PositionHits{
			"high_coverage.sql": {
				"0:10":  10,
				"20:15": 5,
				"40:20": 1,
				"70:25": 0, // 75% coverage (3/4)
			},
			"low_coverage.sql": {
				"0:10":  0,
				"20:15": 0,
				"40:20": 1, // 33.33% coverage (1/3)
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

	// Verify percentage values are present (Go format uses .1f, so 75.0% and 33.3%)
	if !strings.Contains(output, "75.0%") && !strings.Contains(output, "75%") {
		t.Error("Missing high_coverage.sql percentage (75%)")
	}

	if !strings.Contains(output, "33.3%") && !strings.Contains(output, "33%") {
		t.Error("Missing low_coverage.sql percentage (33.33%)")
	}
}

func TestHTMLReporter_CSSPresent(t *testing.T) {
	// Verify CSS is included
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Positions: map[string]coverage.PositionHits{
			"test.sql": {"0:10": 1},
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

	// Verify key Go-style CSS classes
	cssClasses := []string{
		".cov0",
		".cov1",
		".cov8",
		"#topbar",
		"#nav",
	}

	for _, class := range cssClasses {
		if !strings.Contains(output, class) {
			t.Errorf("Missing CSS class: %s", class)
		}
	}
}

func TestHTMLReporter_PositionCoverage(t *testing.T) {
	// Test position coverage indicators with Go-style format
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Positions: map[string]coverage.PositionHits{
			"test.sql": {
				"0:10":  5,  // covered
				"20:15": 0,  // uncovered
				"50:20": 10, // covered
			},
		},
	}

	reporter := NewHTMLReporter()
	output, err := reporter.FormatString(cov)
	if err != nil {
		t.Fatalf("FormatString failed: %v", err)
	}

	// In Go format, we use <span> elements with coverage classes and title attributes
	// Check for coverage classes
	if !strings.Contains(output, "cov5") && !strings.Contains(output, "cov10") {
		t.Error("Missing coverage class for covered positions")
	}

	if !strings.Contains(output, "cov0") {
		t.Error("Missing cov0 class for uncovered position")
	}
}

func TestHTMLReporter_ValidHTML5(t *testing.T) {
	// Test HTML5 validity
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Positions: map[string]coverage.PositionHits{
			"test.sql": {"0:10": 1},
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

	// Go's format uses minimal HTML with charset in Content-Type meta tag
	if !strings.Contains(output, `charset=utf-8`) {
		t.Error("Missing UTF-8 charset declaration")
	}
}

func TestHTMLReporter_EscapeHTML(t *testing.T) {
	// Test HTML escaping for file names with special characters
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Positions: map[string]coverage.PositionHits{
			"test<script>.sql": {"0:10": 1},
			"file&name.sql":    {"0:10": 1},
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
		Positions: map[string]coverage.PositionHits{
			"test1.sql": {"0:10": 1, "20:15": 0},
			"test2.sql": {"0:10": 1, "20:15": 1},
		},
	}

	reporter := NewHTMLReporter()
	output, err := reporter.FormatString(cov)
	if err != nil {
		t.Fatalf("FormatString failed: %v", err)
	}

	// Go format uses #topbar with #legend instead of summary-bar
	if !strings.Contains(output, `id="topbar"`) {
		t.Error("Missing topbar")
	}

	// Legend shows coverage info
	if !strings.Contains(output, `id="legend"`) {
		t.Error("Missing legend")
	}
}

func TestHTMLReporter_CoverageClasses(t *testing.T) {
	// Test coverage class assignments (cov0-cov10)
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Positions: map[string]coverage.PositionHits{
			"high.sql": {
				"0:10":   1,
				"15:10":  1,
				"30:10":  1,
				"45:10":  1,
				"60:10":  1,
				"75:10":  1,
				"90:10":  1,
				"105:10": 1,
				"120:10": 1,
				"135:10": 0,
			}, // 90% coverage
			"medium.sql": {
				"0:10":  3,
				"15:10": 3,
				"30:10": 3,
				"45:10": 0,
				"60:10": 0,
			}, // 60% coverage
			"low.sql": {
				"0:10":  5,
				"15:10": 0,
				"30:10": 0,
				"45:10": 0,
				"60:10": 0,
			}, // 20% coverage
		},
	}

	reporter := NewHTMLReporter()
	output, err := reporter.FormatString(cov)
	if err != nil {
		t.Fatalf("FormatString failed: %v", err)
	}

	// Check that cov0 is present for uncovered positions
	if !strings.Contains(output, `cov0`) {
		t.Error("Missing cov0 coverage class for uncovered positions")
	}

	// Check for covered position classes
	if !strings.Contains(output, `cov1`) && !strings.Contains(output, `cov3`) && !strings.Contains(output, `cov5`) {
		t.Error("Missing covered position classes (cov1, cov3, cov5, etc.)")
	}
}

func TestHTMLReporter_Footer(t *testing.T) {
	// Test that HTML properly closes
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Positions: map[string]coverage.PositionHits{},
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
