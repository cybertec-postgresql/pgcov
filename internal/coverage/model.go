package coverage

import (
	"fmt"
	"time"
)

// Coverage represents aggregated coverage data across all tests
// Uses position-based coverage only (byte offsets)
type Coverage struct {
	Version   string                  `json:"version"`   // Schema version (e.g., "1.0")
	Timestamp time.Time               `json:"timestamp"` // When coverage collected
	Positions map[string]PositionHits `json:"positions"` // Key: relative file path, Value: map of position keys to hit counts
}

// PositionHits represents position hit counts for a single file
type PositionHits map[string]int // Key: "startPos:length", Value: hit count

// BranchCoverage represents coverage data for a single branch
type BranchCoverage struct {
	BranchID string // e.g., "44:if_true"
	HitCount int    // Number of times branch taken
	Covered  bool   // true if HitCount > 0
}

// NewCoverage creates a new Coverage instance
func NewCoverage() *Coverage {
	return &Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Positions: make(map[string]PositionHits),
	}
}

// AddPosition adds or updates position-based coverage data
func (c *Coverage) AddPosition(file string, startPos int, length int, hitCount int) {
	if c.Positions == nil {
		c.Positions = make(map[string]PositionHits)
	}
	if c.Positions[file] == nil {
		c.Positions[file] = make(PositionHits)
	}
	posKey := formatPositionKey(startPos, length)
	c.Positions[file][posKey] = hitCount
}

// AddBranch adds or updates branch coverage data (placeholder for future)
func (c *Coverage) AddBranch(_ string, branchID string, hitCount int) {
	// TODO: Implement branch coverage tracking
	_ = branchID
	_ = hitCount
}

// PositionCoveragePercent calculates position coverage percentage for a file
func (c *Coverage) PositionCoveragePercent(file string) float64 {
	posHits := c.Positions[file]
	if len(posHits) == 0 {
		return 0.0
	}

	covered := 0
	for _, count := range posHits {
		if count > 0 {
			covered++
		}
	}

	return float64(covered) / float64(len(posHits)) * 100.0
}

// TotalPositionCoveragePercent calculates overall position coverage percentage
func (c *Coverage) TotalPositionCoveragePercent() float64 {
	totalPositions := 0
	coveredPositions := 0

	for _, posHits := range c.Positions {
		for _, count := range posHits {
			totalPositions++
			if count > 0 {
				coveredPositions++
			}
		}
	}

	if totalPositions == 0 {
		return 0.0
	}

	return float64(coveredPositions) / float64(totalPositions) * 100.0
}

// formatPositionKey creates a string key from startPos and length
func formatPositionKey(startPos int, length int) string {
	return fmt.Sprintf("%d:%d", startPos, length)
}

// ParsePositionKey parses a position key back into startPos and length
func ParsePositionKey(posKey string) (startPos int, length int, err error) {
	_, err = fmt.Sscanf(posKey, "%d:%d", &startPos, &length)
	return
}

// GetFiles returns a list of all files with coverage data
func (c *Coverage) GetFiles() []string {
	var files []string
	for file := range c.Positions {
		files = append(files, file)
	}
	return files
}
