package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/cybertec-postgresql/pgcov/internal/coverage"
)

func TestJSONReporter_Format(t *testing.T) {
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
			"auth.sql": {
				"0:10":  2,
				"30:20": 0,
				"60:15": 1,
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

		if len(decoded.Positions) != len(cov.Positions) {
			t.Errorf("Positions count mismatch: got %d, want %d", len(decoded.Positions), len(cov.Positions))
		}

		// Verify file coverage
		for file, posHits := range cov.Positions {
			decodedPosHits, ok := decoded.Positions[file]
			if !ok {
				t.Errorf("File %s not found in output", file)
				continue
			}

			if len(decodedPosHits) != len(posHits) {
				t.Errorf("File %s: position count mismatch: got %d, want %d", file, len(decodedPosHits), len(posHits))
			}

			for posKey, count := range posHits {
				if decodedPosHits[posKey] != count {
					t.Errorf("File %s, position %s: hit count mismatch: got %d, want %d", file, posKey, decodedPosHits[posKey], count)
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
		Positions: map[string]coverage.PositionHits{},
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

	if len(decoded.Positions) != 0 {
		t.Errorf("Expected empty positions map, got %d files", len(decoded.Positions))
	}
}

func TestJSONReporter_FormatSummary(t *testing.T) {
	// Create test coverage data
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

	reporter := NewJSONReporter()
	output, err := reporter.FormatSummary(cov)
	if err != nil {
		t.Fatalf("FormatSummary failed: %v", err)
	}

	// Verify JSON is valid
	var summary map[string]any
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

	files, ok := summary["files"].(map[string]any)
	if !ok {
		t.Fatal("Files field is not a map")
	}

	if len(files) != len(cov.Positions) {
		t.Errorf("Files count mismatch: got %d, want %d", len(files), len(cov.Positions))
	}
}

func TestJSONReporter_SchemaCompliance(t *testing.T) {
	// Create comprehensive coverage data to test all fields
	timestamp, _ := time.Parse(time.RFC3339, "2026-01-05T10:00:00Z")
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: timestamp,
		Positions: map[string]coverage.PositionHits{
			"complex.sql": {
				"0:10":   10,
				"50:20":  0,
				"100:30": 1,
				"150:40": 100,
			},
		},
	}

	reporter := NewJSONReporter()
	output, err := reporter.FormatString(cov)
	if err != nil {
		t.Fatalf("FormatString failed: %v", err)
	}

	// Verify required fields are present
	requiredFields := []string{"version", "timestamp", "positions"}
	for _, field := range requiredFields {
		if !strings.Contains(output, `"`+field+`"`) {
			t.Errorf("Missing required field: %s", field)
		}
	}

	// Verify JSON structure matches schema
	var decoded map[string]any
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

	if _, ok := decoded["positions"].(map[string]any); !ok {
		t.Error("positions field should be an object")
	}
}
