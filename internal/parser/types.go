package parser

import (
	"fmt"

	"github.com/cybertec-postgresql/pgcov/internal/discovery"
	pgquery "github.com/pganalyze/pg_query_go/v6"
)

// ParseError represents SQL parsing failure
type ParseError struct {
	File    string
	Line    int
	Column  int
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.File, e.Line, e.Column, e.Message)
}

// NewParseError creates a new ParseError
func NewParseError(file string, line, column int, message string) *ParseError {
	return &ParseError{
		File:    file,
		Line:    line,
		Column:  column,
		Message: message,
	}
}

// ParsedSQL represents a successfully parsed SQL file with AST
type ParsedSQL struct {
	File       *discovery.DiscoveredFile
	AST        *pgquery.ParseResult // From pg_query_go
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
