package parser

import (
	"github.com/cybertec-postgresql/pgcov/internal/discovery"
)

// ParsedSQL represents a successfully parsed SQL file
type ParsedSQL struct {
	File       *discovery.DiscoveredFile
	Statements []*Statement
}

// Statement represents a single SQL statement with location information
type Statement struct {
	RawSQL    string        // Original SQL text
	StartPos  int           // Byte offset in source file
	StartLine int           // 1-indexed line number
	EndLine   int           // 1-indexed line number
	Type      StatementType // Statement classification
	Language  string        // Language for function/procedure statements (e.g. "plpgsql", "sql")
	Body      string        // Function/DO-block body text (unquoted)
	BodyStart int           // Byte offset of body within RawSQL
}

// StatementType classifies SQL statements
type StatementType int

const (
	StmtUnknown   StatementType = iota
	StmtFunction                // CREATE FUNCTION
	StmtProcedure               // CREATE PROCEDURE
	StmtTrigger                 // CREATE TRIGGER
	StmtView                    // CREATE VIEW
	StmtDO                      // DO block
	StmtOther                   // Any other statement
)

// String returns a string representation of StatementType
func (st StatementType) String() string {
	switch st {
	case StmtFunction:
		return "function"
	case StmtProcedure:
		return "procedure"
	case StmtTrigger:
		return "trigger"
	case StmtView:
		return "view"
	case StmtDO:
		return "do"
	case StmtOther:
		return "other"
	default:
		return "unknown"
	}
}
