package report

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/cybertec-postgresql/pgcov/internal/coverage"
)

// JSONReporter formats coverage data as JSON
type JSONReporter struct{}

// NewJSONReporter creates a new JSON reporter
func NewJSONReporter() *JSONReporter {
	return &JSONReporter{}
}

// Format formats coverage data as JSON and writes to the writer
func (r *JSONReporter) Format(cov *coverage.Coverage, writer io.Writer) error {
	// Convert coverage to JSON format
	data, err := json.MarshalIndent(cov, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal coverage to JSON: %w", err)
	}

	// Write JSON to writer
	_, err = writer.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write JSON output: %w", err)
	}

	// Add newline
	_, err = writer.Write([]byte("\n"))
	return err
}

// FormatString returns coverage data as a JSON string
func (r *JSONReporter) FormatString(cov *coverage.Coverage) (string, error) {
	data, err := json.MarshalIndent(cov, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal coverage to JSON: %w", err)
	}
	return string(data), nil
}

// FormatSummary formats a summary view of coverage as JSON
func (r *JSONReporter) FormatSummary(cov *coverage.Coverage) (string, error) {
	summary := make(map[string]interface{})
	summary["version"] = cov.Version
	summary["timestamp"] = cov.Timestamp
	summary["total_coverage_percent"] = cov.TotalLineCoveragePercent()

	files := make(map[string]interface{})
	for path, hits := range cov.Files {
		covered := 0
		for _, count := range hits {
			if count > 0 {
				covered++
			}
		}
		files[path] = map[string]interface{}{
			"lines_covered":    covered,
			"lines_total":      len(hits),
			"coverage_percent": cov.LineCoveragePercent(path),
		}
	}
	summary["files"] = files

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal summary to JSON: %w", err)
	}

	return string(data), nil
}

// Name returns the name of this reporter
func (r *JSONReporter) Name() string {
	return "json"
}
