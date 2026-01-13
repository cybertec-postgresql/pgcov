package instrument

import (
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
