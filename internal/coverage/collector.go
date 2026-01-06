package coverage

import (
	"fmt"

	"github.com/pashagolub/pgcov/internal/instrument"
	"github.com/pashagolub/pgcov/internal/runner"
)

// Collector aggregates coverage signals from test runs
type Collector struct {
	coverage *Coverage
}

// NewCollector creates a new coverage collector
func NewCollector() *Collector {
	return &Collector{
		coverage: NewCoverage(),
	}
}

// CollectFromRun processes coverage signals from a single test run
func (c *Collector) CollectFromRun(testRun *runner.TestRun) error {
	for _, signal := range testRun.CoverageSigs {
		if err := c.AddSignal(signal); err != nil {
			return fmt.Errorf("failed to process signal %s: %w", signal.SignalID, err)
		}
	}
	return nil
}

// CollectFromRuns processes coverage signals from multiple test runs
func (c *Collector) CollectFromRuns(testRuns []*runner.TestRun) error {
	for _, run := range testRuns {
		if err := c.CollectFromRun(run); err != nil {
			return err
		}
	}
	return nil
}

// AddSignal adds a single coverage signal to the aggregated coverage
func (c *Collector) AddSignal(signal runner.CoverageSignal) error {
	// Parse signal ID to extract file, line, and branch
	file, line, branch, err := instrument.ParseSignalID(signal.SignalID)
	if err != nil {
		return fmt.Errorf("invalid signal ID: %w", err)
	}

	// Add line or branch coverage
	if branch == "" {
		// Line coverage - increment hit count
		if existingCount, exists := c.coverage.Files[file][line]; exists {
			c.coverage.AddLine(file, line, existingCount+1)
		} else {
			c.coverage.AddLine(file, line, 1)
		}
	} else {
		// Branch coverage (placeholder for future)
		branchKey := fmt.Sprintf("%d:%s", line, branch)
		c.coverage.AddBranch(file, branchKey, 1)
	}

	return nil
}

// Coverage returns the aggregated coverage data
func (c *Collector) Coverage() *Coverage {
	return c.coverage
}

// Reset clears all collected coverage data
func (c *Collector) Reset() {
	c.coverage = NewCoverage()
}

// Merge merges another coverage collector's data into this one
func (c *Collector) Merge(other *Collector) error {
	for file, otherHits := range other.coverage.Files {
		// Merge line hit counts
		for line, count := range otherHits {
			if existingCount, exists := c.coverage.Files[file][line]; exists {
				c.coverage.AddLine(file, line, existingCount+count)
			} else {
				c.coverage.AddLine(file, line, count)
			}
		}
	}
	return nil
}

// GetFileCoverage returns coverage data for a specific file (simplified)
func (c *Collector) GetFileCoverage(filePath string) FileHits {
	return c.coverage.Files[filePath]
}

// GetFileList returns a list of all files with coverage data
func (c *Collector) GetFileList() []string {
	var files []string
	for file := range c.coverage.Files {
		files = append(files, file)
	}
	return files
}

// TotalCoveragePercent returns the overall coverage percentage
func (c *Collector) TotalCoveragePercent() float64 {
	return c.coverage.TotalLineCoveragePercent()
}
