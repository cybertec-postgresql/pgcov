package errors

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
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

// ConnectionError represents PostgreSQL connection failure
type ConnectionError struct {
	Host    string
	Port    int
	Message string
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("failed to connect to %s:%d: %s", e.Host, e.Port, e.Message)
}

// NewConnectionError creates a new ConnectionError
func NewConnectionError(host string, port int, message string) *ConnectionError {
	return &ConnectionError{
		Host:    host,
		Port:    port,
		Message: message,
	}
}

// TestFailureError represents test execution failure
type TestFailureError struct {
	Test     string
	SQLError *pgconn.PgError // PostgreSQL error details
}

func (e *TestFailureError) Error() string {
	if e.SQLError != nil {
		return fmt.Sprintf("test %s failed: [%s] %s", e.Test, e.SQLError.Code, e.SQLError.Message)
	}
	return fmt.Sprintf("test %s failed", e.Test)
}

// NewTestFailureError creates a new TestFailureError
func NewTestFailureError(test string, sqlError *pgconn.PgError) *TestFailureError {
	return &TestFailureError{
		Test:     test,
		SQLError: sqlError,
	}
}

// InstrumentationError represents SQL instrumentation failure
type InstrumentationError struct {
	File    string
	Message string
}

func (e *InstrumentationError) Error() string {
	return fmt.Sprintf("failed to instrument %s: %s", e.File, e.Message)
}

// NewInstrumentationError creates a new InstrumentationError
func NewInstrumentationError(file, message string) *InstrumentationError {
	return &InstrumentationError{
		File:    file,
		Message: message,
	}
}
