package coverage

import "time"

// Coverage represents aggregated coverage data across all tests
type Coverage struct {
	Version   string              // Schema version (e.g., "1.0")
	Timestamp time.Time           // When coverage collected
	Files     map[string]FileHits // Key: relative file path, Value: map of line numbers to hit counts
}

// FileHits represents line hit counts for a single file
type FileHits map[int]int // Key: line number, Value: hit count

// FileCoverage represents coverage data for a single source file (for compatibility)
type FileCoverage struct {
	Path     string                     // Relative file path
	Lines    map[int]*LineCoverage      // Key: line number
	Branches map[string]*BranchCoverage // Key: branch identifier
}

// LineCoverage represents coverage data for a single line (for compatibility)
type LineCoverage struct {
	LineNumber int  // 1-indexed line number
	HitCount   int  // Number of times line executed
	Covered    bool // true if HitCount > 0
}

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
		Files:     make(map[string]FileHits),
	}
}

// NewFileCoverage creates a new FileCoverage instance (for compatibility)
func NewFileCoverage(path string) *FileCoverage {
	return &FileCoverage{
		Path:     path,
		Lines:    make(map[int]*LineCoverage),
		Branches: make(map[string]*BranchCoverage),
	}
}

// AddLine adds or updates line coverage data
func (c *Coverage) AddLine(file string, line int, hitCount int) {
	if c.Files[file] == nil {
		c.Files[file] = make(FileHits)
	}
	c.Files[file][line] = hitCount
}

// AddBranch adds or updates branch coverage data (placeholder for future)
func (c *Coverage) AddBranch(_ string, branchID string, hitCount int) {
	// TODO: Implement branch coverage tracking
	_ = branchID
	_ = hitCount
}

// LineCoveragePercent calculates the line coverage percentage for a file
func (c *Coverage) LineCoveragePercent(file string) float64 {
	hits := c.Files[file]
	if len(hits) == 0 {
		return 0.0
	}

	covered := 0
	for _, count := range hits {
		if count > 0 {
			covered++
		}
	}

	return float64(covered) / float64(len(hits)) * 100.0
}

// TotalLineCoveragePercent calculates overall line coverage percentage
func (c *Coverage) TotalLineCoveragePercent() float64 {
	totalLines := 0
	coveredLines := 0

	for _, hits := range c.Files {
		for _, count := range hits {
			totalLines++
			if count > 0 {
				coveredLines++
			}
		}
	}

	if totalLines == 0 {
		return 0.0
	}

	return float64(coveredLines) / float64(totalLines) * 100.0
}
