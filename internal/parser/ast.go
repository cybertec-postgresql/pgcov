package parser

import (
	"strings"
)

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
