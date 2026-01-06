package parser

import (
	"fmt"
	"os"

	"github.com/pashagolub/pgcov/internal/discovery"
	"github.com/pashagolub/pgcov/internal/errors"
	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// Parse parses a SQL file and returns ParsedSQL with AST and statements
func Parse(file *discovery.DiscoveredFile) (*ParsedSQL, error) {
	// Read file content
	content, err := os.ReadFile(file.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse SQL using pg_query_go
	result, err := pg_query.Parse(string(content))
	if err != nil {
		// Extract location information if available
		return nil, &errors.ParseError{
			File:    file.Path,
			Line:    0, // pg_query_go doesn't provide line info in error
			Column:  0,
			Message: err.Error(),
		}
	}

	// Extract statements from AST
	statements, err := ExtractStatements(string(content), result)
	if err != nil {
		return nil, fmt.Errorf("failed to extract statements: %w", err)
	}

	return &ParsedSQL{
		File:       file,
		AST:        result,
		Statements: statements,
	}, nil
}

// ParseFile is a convenience function that parses a file path directly
func ParseFile(filePath string) (*ParsedSQL, error) {
	file := &discovery.DiscoveredFile{
		Path: filePath,
		Type: discovery.ClassifyPath(filePath),
	}
	return Parse(file)
}

// ParseSQL parses SQL text directly without a file
func ParseSQL(sql string) (*pg_query.ParseResult, error) {
	result, err := pg_query.Parse(sql)
	if err != nil {
		return nil, &errors.ParseError{
			File:    "<inline>",
			Message: err.Error(),
		}
	}
	return result, nil
}
