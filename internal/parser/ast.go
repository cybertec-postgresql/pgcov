package parser

import (
	"strings"

	pgquery "github.com/pganalyze/pg_query_go/v6"
)

// ExtractStatements extracts individual statements from a parsed SQL file
func ExtractStatements(sql string, result *pgquery.ParseResult) ([]*Statement, error) {
	var statements []*Statement

	// Process each statement in the parse result
	for _, stmt := range result.Stmts {
		// Get statement location from pg_query_go
		stmtLocation := stmt.StmtLocation
		stmtLen := stmt.StmtLen

		// Calculate line numbers
		startLine := calculateLineNumber(sql, int(stmtLocation))
		endLine := calculateLineNumber(sql, int(stmtLocation+stmtLen))

		// Extract raw SQL text
		rawSQL := ""
		if stmtLocation >= 0 && stmtLocation+stmtLen <= int32(len(sql)) {
			rawSQL = sql[stmtLocation : stmtLocation+stmtLen]

			// pg_query_go doesn't include the trailing semicolon in stmtLen
			// Check if there's a semicolon right after and include it
			endPos := int(stmtLocation + stmtLen)
			if endPos < len(sql) && sql[endPos] == ';' {
				rawSQL += ";"
			}
		}

		// Classify statement type
		stmtType := ClassifyStatement(stmt.Stmt)

		statements = append(statements, &Statement{
			RawSQL:    rawSQL,
			StartLine: startLine,
			EndLine:   endLine,
			Type:      stmtType,
		})
	}

	// If no statements were extracted, return empty slice
	// Don't create a dummy statement for empty files or comment-only files
	return statements, nil
}

// calculateLineNumber converts a byte offset to a 1-indexed line number
func calculateLineNumber(sql string, offset int) int {
	if offset < 0 {
		return 1
	}
	if offset > len(sql) {
		offset = len(sql)
	}

	lineNum := 1
	for i := 0; i < offset && i < len(sql); i++ {
		if sql[i] == '\n' {
			lineNum++
		}
	}
	return lineNum
}

// ClassifyStatement determines the type of SQL statement
func ClassifyStatement(node *pgquery.Node) StatementType {
	if node == nil {
		return StmtUnknown
	}

	switch node.Node.(type) {
	case *pgquery.Node_CreateFunctionStmt:
		return StmtFunction
	case *pgquery.Node_CreateTrigStmt:
		return StmtTrigger
	case *pgquery.Node_ViewStmt:
		return StmtView
	// Note: pg_query_go doesn't have a separate procedure type in older versions
	// Procedures are represented as functions in PostgreSQL
	default:
		// Check if it's a procedure by looking at the SQL text
		// This is a heuristic since pg_query_go may not distinguish
		return StmtOther
	}
}

// GetStatementAtLine returns the statement that contains the given line number
func GetStatementAtLine(statements []*Statement, lineNum int) *Statement {
	for _, stmt := range statements {
		if lineNum >= stmt.StartLine && lineNum <= stmt.EndLine {
			return stmt
		}
	}
	return nil
}

// GetExecutableStatements returns statements that can be executed
// (filters out comments, empty statements, etc.)
func GetExecutableStatements(statements []*Statement) []*Statement {
	var executable []*Statement
	for _, stmt := range statements {
		// Filter out empty or whitespace-only statements
		if strings.TrimSpace(stmt.RawSQL) != "" {
			executable = append(executable, stmt)
		}
	}
	return executable
}

// GetStatementsByType returns all statements of a given type
func GetStatementsByType(statements []*Statement, stmtType StatementType) []*Statement {
	var filtered []*Statement
	for _, stmt := range statements {
		if stmt.Type == stmtType {
			filtered = append(filtered, stmt)
		}
	}
	return filtered
}
