package coverage

import (
	"fmt"
	"sync"

	"github.com/cybertec-postgresql/pgcov/internal/instrument"
	"github.com/cybertec-postgresql/pgcov/internal/runner"
)

// Collector aggregates coverage signals from test runs
type Collector struct {
	coverage *Coverage
	mu       sync.Mutex // Protects coverage for thread-safe parallel execution
}

// NewCollector creates a new coverage collector
func NewCollector() *Collector {
	return &Collector{
		coverage: NewCoverage(),
	}
}

// CollectFromRun processes coverage signals from a single test run
func (c *Collector) CollectFromRun(testRun *runner.TestRun) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, signal := range testRun.CoverageSigs {
		if err := c.addSignalUnsafe(signal); err != nil {
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
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.addSignalUnsafe(signal)
}

// addSignalUnsafe adds a signal without locking (internal use when lock is already held)
func (c *Collector) addSignalUnsafe(signal runner.CoverageSignal) error {
	// Parse signal ID to extract file, startPos, length, and branch
	file, startPos, length, branch, err := instrument.ParseSignalID(signal.SignalID)
	if err != nil {
		return fmt.Errorf("invalid signal ID: %w", err)
	}

	// Add position-based coverage only
	if branch == "" {
		// Position coverage - increment hit count
		posKey := fmt.Sprintf("%d:%d", startPos, length)
		if existingCount, exists := c.coverage.Positions[file][posKey]; exists {
			c.coverage.AddPosition(file, startPos, length, existingCount+1)
		} else {
			c.coverage.AddPosition(file, startPos, length, 1)
		}
	} else {
		// Branch coverage (placeholder for future)
		branchKey := fmt.Sprintf("%d:%d:%s", startPos, length, branch)
		c.coverage.AddBranch(file, branchKey, 1)
	}

	return nil
}

// Coverage returns the aggregated coverage data
func (c *Collector) Coverage() *Coverage {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.coverage
}

// Reset clears all collected coverage data
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.coverage = NewCoverage()
}

// Merge merges another coverage collector's data into this one
func (c *Collector) Merge(other *Collector) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	other.mu.Lock()
	defer other.mu.Unlock()

	// Merge position hit counts only
	for file, otherPosHits := range other.coverage.Positions {
		for posKey, count := range otherPosHits {
			// Parse position key to get startPos and length
			var startPos, length int
			_, err := fmt.Sscanf(posKey, "%d:%d", &startPos, &length)
			if err != nil {
				continue // Skip invalid keys
			}

			if existingCount, exists := c.coverage.Positions[file][posKey]; exists {
				c.coverage.AddPosition(file, startPos, length, existingCount+count)
			} else {
				c.coverage.AddPosition(file, startPos, length, count)
			}
		}
	}

	return nil
}

// GetFilePositionCoverage returns position coverage data for a specific file
func (c *Collector) GetFilePositionCoverage(filePath string) PositionHits {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.coverage.Positions[filePath]
}

// GetFileList returns a list of all files with coverage data
func (c *Collector) GetFileList() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	var files []string
	for file := range c.coverage.Positions {
		files = append(files, file)
	}
	return files
}

// TotalCoveragePercent returns the overall coverage percentage
func (c *Collector) TotalCoveragePercent() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.coverage.TotalPositionCoveragePercent()
}
