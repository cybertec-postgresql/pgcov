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
			"pgcov Coverage Report",
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

		// Verify coverage data
		if !strings.Contains(output, "covered") || !strings.Contains(output, "uncovered") {
			t.Error("Missing coverage indicators (covered/uncovered)")
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

	if !strings.Contains(output, "pgcov Coverage Report") {
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
		".covered",
		".uncovered",
		".file-detail",
		".source-code",
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

	// Verify line numbers are present
	for i := 1; i <= 3; i++ {
		// Look for line number divs
		if !strings.Contains(output, `<div class="line-number">`) {
			t.Error("Missing line-number divs")
			break
		}
	}

	// Verify hit count display
	if !strings.Contains(output, "5×") {
		t.Error("Missing hit count display for line 1 (5×)")
	}

	if !strings.Contains(output, "0×") {
		t.Error("Missing hit count display for uncovered line (0×)")
	}

	if !strings.Contains(output, "10×") {
		t.Error("Missing hit count display for line 3 (10×)")
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

	// Verify summary section exists
	if !strings.Contains(output, `class="summary"`) {
		t.Error("Missing summary section")
	}

	// Verify summary stats
	if !strings.Contains(output, "Overall Coverage") {
		t.Error("Missing overall coverage heading")
	}

	if !strings.Contains(output, "Lines Covered") {
		t.Error("Missing lines covered stat")
	}

	if !strings.Contains(output, "Files") {
		t.Error("Missing files count stat")
	}
}

func TestHTMLReporter_CoverageClasses(t *testing.T) {
	// Test coverage class assignments (high/medium/low)
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Files: map[string]coverage.FileHits{
			"high.sql": {
				1: 1, 2: 1, 3: 1, 4: 1, 5: 1,
				6: 1, 7: 1, 8: 1, 9: 1, 10: 0,
			}, // 90% coverage - should be "high"
			"medium.sql": {
				1: 1, 2: 1, 3: 1, 4: 0, 5: 0,
			}, // 60% coverage - should be "medium"
			"low.sql": {
				1: 1, 2: 0, 3: 0, 4: 0, 5: 0,
			}, // 20% coverage - should be "low"
		},
	}

	reporter := NewHTMLReporter()
	output, err := reporter.FormatString(cov)
	if err != nil {
		t.Fatalf("FormatString failed: %v", err)
	}

	// Count occurrences of coverage classes
	// Note: we can't be too specific as the order may vary
	if !strings.Contains(output, `class="file-coverage high"`) {
		t.Error("Missing high coverage class")
	}

	if !strings.Contains(output, `class="file-coverage medium"`) {
		t.Error("Missing medium coverage class")
	}

	if !strings.Contains(output, `class="file-coverage low"`) {
		t.Error("Missing low coverage class")
	}
}

func TestHTMLReporter_Footer(t *testing.T) {
	// Test footer content
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

	// Verify footer exists
	if !strings.Contains(output, "<footer>") {
		t.Error("Missing footer element")
	}

	if !strings.Contains(output, "pgcov") {
		t.Error("Missing pgcov branding in footer")
	}
}
