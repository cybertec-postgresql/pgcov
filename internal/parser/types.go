package parser

import (
	"github.com/pganalyze/pg_query_go/v6"
	"github.com/pashagolub/pgcov/internal/discovery"
)

// ParsedSQL represents a successfully parsed SQL file with AST
type ParsedSQL struct {
	File       *discovery.DiscoveredFile
	AST        *pg_query.ParseResult // From pg_query_go
	Statements []*Statement
}

// Statement represents a single SQL statement with location information
type Statement struct {
	RawSQL    string        // Original SQL text
	StartLine int           // 1-indexed line number
	EndLine   int           // 1-indexed line number
	Type      StatementType // Statement classification
}

// StatementType classifies SQL statements
type StatementType int

const (
	StmtUnknown   StatementType = iota
	StmtFunction                // CREATE FUNCTION
	StmtProcedure               // CREATE PROCEDURE
	StmtTrigger                 // CREATE TRIGGER
	StmtView                    // CREATE VIEW
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
	case StmtOther:
		return "other"
	default:
		return "unknown"
	}
}
