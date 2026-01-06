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

	// Get or create file coverage
	fileCov, exists := c.coverage.Files[file]
	if !exists {
		fileCov = NewFileCoverage(file)
		c.coverage.Files[file] = fileCov
	}

	// Add line or branch coverage
	if branch == "" {
		// Line coverage
		if existingLine, exists := fileCov.Lines[line]; exists {
			existingLine.HitCount++
			existingLine.Covered = true
		} else {
			fileCov.AddLine(line, 1)
		}
	} else {
		// Branch coverage
		branchKey := fmt.Sprintf("%d:%s", line, branch)
		if existingBranch, exists := fileCov.Branches[branchKey]; exists {
			existingBranch.HitCount++
			existingBranch.Covered = true
		} else {
			fileCov.AddBranch(branchKey, 1)
		}
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
	for file, otherFileCov := range other.coverage.Files {
		fileCov, exists := c.coverage.Files[file]
		if !exists {
			// Deep copy the file coverage
			fileCov = NewFileCoverage(file)
			c.coverage.Files[file] = fileCov
		}

		// Merge lines
		for line, otherLine := range otherFileCov.Lines {
			if existingLine, exists := fileCov.Lines[line]; exists {
				existingLine.HitCount += otherLine.HitCount
				existingLine.Covered = existingLine.HitCount > 0
			} else {
				fileCov.AddLine(line, otherLine.HitCount)
			}
		}

		// Merge branches
		for branchID, otherBranch := range otherFileCov.Branches {
			if existingBranch, exists := fileCov.Branches[branchID]; exists {
				existingBranch.HitCount += otherBranch.HitCount
				existingBranch.Covered = existingBranch.HitCount > 0
			} else {
				fileCov.AddBranch(branchID, otherBranch.HitCount)
			}
		}
	}

	return nil
}

// GetFileCoverage returns coverage data for a specific file
func (c *Collector) GetFileCoverage(filePath string) *FileCoverage {
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
