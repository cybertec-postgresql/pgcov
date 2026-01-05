package coverage

import "time"

// Coverage represents aggregated coverage data across all tests
type Coverage struct {
	Version   string                    // Schema version (e.g., "1.0")
	Timestamp time.Time                 // When coverage collected
	Files     map[string]*FileCoverage  // Key: relative file path
}

// FileCoverage represents coverage data for a single source file
type FileCoverage struct {
	Path     string                      // Relative file path
	Lines    map[int]*LineCoverage       // Key: line number
	Branches map[string]*BranchCoverage  // Key: branch identifier
}

// LineCoverage represents coverage data for a single line
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
		Files:     make(map[string]*FileCoverage),
	}
}

// NewFileCoverage creates a new FileCoverage instance
func NewFileCoverage(path string) *FileCoverage {
	return &FileCoverage{
		Path:     path,
		Lines:    make(map[int]*LineCoverage),
		Branches: make(map[string]*BranchCoverage),
	}
}

// AddLine adds or updates line coverage data
func (fc *FileCoverage) AddLine(line int, hitCount int) {
	fc.Lines[line] = &LineCoverage{
		LineNumber: line,
		HitCount:   hitCount,
		Covered:    hitCount > 0,
	}
}

// AddBranch adds or updates branch coverage data
func (fc *FileCoverage) AddBranch(branchID string, hitCount int) {
	fc.Branches[branchID] = &BranchCoverage{
		BranchID: branchID,
		HitCount: hitCount,
		Covered:  hitCount > 0,
	}
}

// LineCoveragePercent calculates the line coverage percentage
func (fc *FileCoverage) LineCoveragePercent() float64 {
	if len(fc.Lines) == 0 {
		return 0.0
	}
	
	covered := 0
	for _, line := range fc.Lines {
		if line.Covered {
			covered++
		}
	}
	
	return float64(covered) / float64(len(fc.Lines)) * 100.0
}

// TotalLineCoveragePercent calculates overall line coverage percentage
func (c *Coverage) TotalLineCoveragePercent() float64 {
	totalLines := 0
	coveredLines := 0
	
	for _, file := range c.Files {
		for _, line := range file.Lines {
			totalLines++
			if line.Covered {
				coveredLines++
			}
		}
	}
	
	if totalLines == 0 {
		return 0.0
	}
	
	return float64(coveredLines) / float64(totalLines) * 100.0
}
