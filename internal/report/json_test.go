package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/pashagolub/pgcov/internal/coverage"
)

func TestJSONReporter_Format(t *testing.T) {
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
			"auth.sql": {
				10: 2,
				11: 0,
				12: 1,
			},
		},
	}

	// Create reporter
	reporter := NewJSONReporter()

	// Test Format method
	t.Run("Format", func(t *testing.T) {
		var buf bytes.Buffer
		err := reporter.Format(cov, &buf)
		if err != nil {
			t.Fatalf("Format failed: %v", err)
		}

		// Verify JSON is valid
		var decoded coverage.Coverage
		err = json.Unmarshal(buf.Bytes(), &decoded)
		if err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}

		// Verify structure
		if decoded.Version != cov.Version {
			t.Errorf("Version mismatch: got %s, want %s", decoded.Version, cov.Version)
		}

		if len(decoded.Files) != len(cov.Files) {
			t.Errorf("Files count mismatch: got %d, want %d", len(decoded.Files), len(cov.Files))
		}

		// Verify file coverage
		for file, hits := range cov.Files {
			decodedHits, ok := decoded.Files[file]
			if !ok {
				t.Errorf("File %s not found in output", file)
				continue
			}

			if len(decodedHits) != len(hits) {
				t.Errorf("File %s: line count mismatch: got %d, want %d", file, len(decodedHits), len(hits))
			}

			for line, count := range hits {
				if decodedHits[line] != count {
					t.Errorf("File %s, line %d: hit count mismatch: got %d, want %d", file, line, decodedHits[line], count)
				}
			}
		}
	})

	// Test FormatString method
	t.Run("FormatString", func(t *testing.T) {
		output, err := reporter.FormatString(cov)
		if err != nil {
			t.Fatalf("FormatString failed: %v", err)
		}

		// Verify JSON is valid
		var decoded coverage.Coverage
		err = json.Unmarshal([]byte(output), &decoded)
		if err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}

		// Verify structure
		if decoded.Version != cov.Version {
			t.Errorf("Version mismatch: got %s, want %s", decoded.Version, cov.Version)
		}
	})

	// Test Name method
	t.Run("Name", func(t *testing.T) {
		name := reporter.Name()
		if name != "json" {
			t.Errorf("Name mismatch: got %s, want json", name)
		}
	})
}

func TestJSONReporter_EmptyCoverage(t *testing.T) {
	// Create empty coverage data
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Files:     map[string]coverage.FileHits{},
	}

	reporter := NewJSONReporter()
	var buf bytes.Buffer
	err := reporter.Format(cov, &buf)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Verify JSON is valid
	var decoded coverage.Coverage
	err = json.Unmarshal(buf.Bytes(), &decoded)
	if err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	if len(decoded.Files) != 0 {
		t.Errorf("Expected empty files map, got %d files", len(decoded.Files))
	}
}

func TestJSONReporter_FormatSummary(t *testing.T) {
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

	reporter := NewJSONReporter()
	output, err := reporter.FormatSummary(cov)
	if err != nil {
		t.Fatalf("FormatSummary failed: %v", err)
	}

	// Verify JSON is valid
	var summary map[string]interface{}
	err = json.Unmarshal([]byte(output), &summary)
	if err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Verify summary fields
	if summary["version"] != cov.Version {
		t.Errorf("Version mismatch: got %v, want %s", summary["version"], cov.Version)
	}

	if summary["total_coverage_percent"] == nil {
		t.Error("Missing total_coverage_percent field")
	}

	files, ok := summary["files"].(map[string]interface{})
	if !ok {
		t.Fatal("Files field is not a map")
	}

	if len(files) != len(cov.Files) {
		t.Errorf("Files count mismatch: got %d, want %d", len(files), len(cov.Files))
	}
}

func TestJSONReporter_SchemaCompliance(t *testing.T) {
	// Create comprehensive coverage data to test all fields
	timestamp, _ := time.Parse(time.RFC3339, "2026-01-05T10:00:00Z")
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: timestamp,
		Files: map[string]coverage.FileHits{
			"complex.sql": {
				1:  10,
				5:  0,
				10: 1,
				15: 100,
			},
		},
	}

	reporter := NewJSONReporter()
	output, err := reporter.FormatString(cov)
	if err != nil {
		t.Fatalf("FormatString failed: %v", err)
	}

	// Verify required fields are present
	requiredFields := []string{"version", "timestamp", "files"}
	for _, field := range requiredFields {
		if !strings.Contains(output, `"`+field+`"`) {
			t.Errorf("Missing required field: %s", field)
		}
	}

	// Verify JSON structure matches schema
	var decoded map[string]interface{}
	err = json.Unmarshal([]byte(output), &decoded)
	if err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Check top-level fields
	if _, ok := decoded["version"].(string); !ok {
		t.Error("version field should be a string")
	}

	if _, ok := decoded["timestamp"].(string); !ok {
		t.Error("timestamp field should be a string")
	}

	if _, ok := decoded["files"].(map[string]interface{}); !ok {
		t.Error("files field should be an object")
	}
}
