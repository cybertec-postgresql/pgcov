package instrument

import (
	"fmt"

	"github.com/cybertec-postgresql/pgcov/internal/parser"
)

// InstrumentedSQL represents SQL code that has been instrumented for coverage tracking
type InstrumentedSQL struct {
	Original         *parser.ParsedSQL
	InstrumentedText string          // Rewritten SQL with NOTIFY calls
	Locations        []CoveragePoint // All instrumented locations
}

// CoveragePoint represents a single location in source code tracked for coverage
type CoveragePoint struct {
	File             string // Relative file path
	StartPos         int    // Start position (byte offset, 0-indexed)
	Length           int    // Length of the covered code segment in bytes
	Branch           string // Branch identifier (optional, e.g., "if_true", "if_false")
	SignalID         string // Unique signal identifier sent via NOTIFY
	ImplicitCoverage bool   // True if covered by successful execution (DDL/DML), false if needs NOTIFY
}

// FormatSignalID generates a signal ID for a coverage point
// Format: {file}:{startPos}:{length} or {file}:{startPos}:{length}:{branch}
func (cp *CoveragePoint) FormatSignalID() string {
	if cp.Branch == "" {
		return fmt.Sprintf("%s:%d:%d", cp.File, cp.StartPos, cp.Length)
	}
	return fmt.Sprintf("%s:%d:%d:%s", cp.File, cp.StartPos, cp.Length, cp.Branch)
}
