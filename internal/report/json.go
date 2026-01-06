package report

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/pashagolub/pgcov/internal/coverage"
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
	for path, fileCov := range cov.Files {
		files[path] = map[string]interface{}{
			"lines_covered":   countCovered(fileCov),
			"lines_total":     len(fileCov.Lines),
			"coverage_percent": fileCov.LineCoveragePercent(),
		}
	}
	summary["files"] = files

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal summary to JSON: %w", err)
	}

	return string(data), nil
}

// countCovered counts the number of covered lines in a file
func countCovered(fileCov *coverage.FileCoverage) int {
	count := 0
	for _, line := range fileCov.Lines {
		if line.Covered {
			count++
		}
	}
	return count
}

// Name returns the name of this reporter
func (r *JSONReporter) Name() string {
	return "json"
}
