package report

import (
	"fmt"
	"io"
	"sort"

	"github.com/pashagolub/pgcov/internal/coverage"
)

// LCOVReporter formats coverage data in LCOV format
// LCOV format specification: https://github.com/linux-test-project/lcov
type LCOVReporter struct{}

// NewLCOVReporter creates a new LCOV reporter
func NewLCOVReporter() *LCOVReporter {
	return &LCOVReporter{}
}

// Format formats coverage data as LCOV and writes to the writer
func (r *LCOVReporter) Format(cov *coverage.Coverage, writer io.Writer) error {
	// Sort files for deterministic output
	var files []string
	for file := range cov.Files {
		files = append(files, file)
	}
	sort.Strings(files)

	// Write LCOV format for each file
	for _, file := range files {
		fileCov := cov.Files[file]
		if err := r.formatFile(file, fileCov, writer); err != nil {
			return err
		}
	}

	return nil
}

// formatFile formats a single file's coverage in LCOV format
func (r *LCOVReporter) formatFile(path string, hits coverage.FileHits, writer io.Writer) error {
	// SF:<source file path>
	if _, err := fmt.Fprintf(writer, "SF:%s\n", path); err != nil {
		return err
	}

	// Sort line numbers for deterministic output
	var lines []int
	for line := range hits {
		lines = append(lines, line)
	}
	sort.Ints(lines)

	// DA:<line number>,<hit count>
	for _, line := range lines {
		hitCount := hits[line]
		if _, err := fmt.Fprintf(writer, "DA:%d,%d\n", line, hitCount); err != nil {
			return err
		}
	}

	// LF:<number of instrumented lines>
	linesFound := len(hits)
	if _, err := fmt.Fprintf(writer, "LF:%d\n", linesFound); err != nil {
		return err
	}

	// LH:<number of lines with non-zero execution count>
	linesHit := 0
	for _, count := range hits {
		if count > 0 {
			linesHit++
		}
	}
	if _, err := fmt.Fprintf(writer, "LH:%d\n", linesHit); err != nil {
		return err
	}

	// end_of_record
	if _, err := fmt.Fprintf(writer, "end_of_record\n"); err != nil {
		return err
	}

	return nil
}

// FormatString returns coverage data as an LCOV-formatted string
func (r *LCOVReporter) FormatString(cov *coverage.Coverage) (string, error) {
	var buf []byte
	writer := &byteWriter{data: &buf}
	if err := r.Format(cov, writer); err != nil {
		return "", err
	}
	return string(buf), nil
}

// Name returns the name of this reporter
func (r *LCOVReporter) Name() string {
	return "lcov"
}

// byteWriter is a simple io.Writer that writes to a byte slice
type byteWriter struct {
	data *[]byte
}

func (w *byteWriter) Write(p []byte) (n int, err error) {
	*w.data = append(*w.data, p...)
	return len(p), nil
}
