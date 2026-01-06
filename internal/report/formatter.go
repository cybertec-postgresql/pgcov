package report

import (
	"fmt"
	"io"

	"github.com/pashagolub/pgcov/internal/coverage"
)

// Formatter is an interface for coverage report formatters
type Formatter interface {
	// Format formats coverage data and writes to the writer
	Format(cov *coverage.Coverage, writer io.Writer) error

	// FormatString returns coverage data as a string
	FormatString(cov *coverage.Coverage) (string, error)

	// Name returns the name of this formatter
	Name() string
}

// FormatType represents supported report formats
type FormatType string

const (
	FormatJSON FormatType = "json"
	FormatLCOV FormatType = "lcov"
	FormatHTML FormatType = "html"
)

// GetFormatter returns a formatter for the specified format type
func GetFormatter(format FormatType) (Formatter, error) {
	switch format {
	case FormatJSON:
		return NewJSONReporter(), nil
	case FormatLCOV:
		return NewLCOVReporter(), nil
	case FormatHTML:
		return NewHTMLReporter(), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s (supported: json, lcov, html)", format)
	}
}

// FormatToWriter formats coverage data to a writer using the specified format
func FormatToWriter(cov *coverage.Coverage, format FormatType, writer io.Writer) error {
	formatter, err := GetFormatter(format)
	if err != nil {
		return err
	}
	return formatter.Format(cov, writer)
}

// FormatToString formats coverage data to a string using the specified format
func FormatToString(cov *coverage.Coverage, format FormatType) (string, error) {
	formatter, err := GetFormatter(format)
	if err != nil {
		return "", err
	}
	return formatter.FormatString(cov)
}

// ValidFormat checks if a format string is valid
func ValidFormat(format string) bool {
	switch FormatType(format) {
	case FormatJSON, FormatLCOV, FormatHTML:
		return true
	default:
		return false
	}
}

// SupportedFormats returns a list of supported format names
func SupportedFormats() []string {
	return []string{string(FormatJSON), string(FormatLCOV), string(FormatHTML)}
}
