package instrument

import "github.com/cybertec-postgresql/pgcov/internal/parser"

// InstrumentedSQL represents SQL code that has been instrumented for coverage tracking
type InstrumentedSQL struct {
	Original         *parser.ParsedSQL
	InstrumentedText string          // Rewritten SQL with NOTIFY calls
	Locations        []CoveragePoint // All instrumented locations
}

// CoveragePoint represents a single location in source code tracked for coverage
type CoveragePoint struct {
	File             string // Relative file path
	Line             int    // Line number (1-indexed)
	Branch           string // Branch identifier (optional, e.g., "if_true", "if_false")
	SignalID         string // Unique signal identifier sent via NOTIFY
	ImplicitCoverage bool   // True if covered by successful execution (DDL/DML), false if needs NOTIFY
}

// FormatSignalID generates a signal ID for a coverage point
// Format: {file}:{line} or {file}:{line}:{branch}
func (cp *CoveragePoint) FormatSignalID() string {
	if cp.Branch == "" {
		return cp.File + ":" + string(rune(cp.Line))
	}
	return cp.File + ":" + string(rune(cp.Line)) + ":" + cp.Branch
}
