package types

import (
	"fmt"
	"time"
)

// Config holds runtime configuration combining flags, environment variables, and defaults
type Config struct {
	// PostgreSQL connection
	PGHost     string
	PGPort     int
	PGUser     string
	PGPassword string
	PGDatabase string // Template database for creating temp DBs

	// Execution
	SearchPath  string        // Root path for test/source discovery
	Timeout     time.Duration // Per-test timeout
	Parallelism int           // Max concurrent tests (1 = sequential)

	// Output
	CoverageFile string // Coverage data output path
	Verbose      bool   // Enable debug logging
}

// ConfigError represents a configuration validation error
type ConfigError struct {
	Field      string
	Value      interface{}
	Message    string
	Suggestion string
}

func (e *ConfigError) Error() string {
	msg := fmt.Sprintf("configuration error: %s", e.Message)
	if e.Field != "" {
		msg = fmt.Sprintf("configuration error for %s: %s", e.Field, e.Message)
	}
	if e.Suggestion != "" {
		msg += fmt.Sprintf("\n\nSuggestion: %s", e.Suggestion)
	}
	return msg
}

// Validate checks configuration for errors and returns helpful error messages
func (c *Config) Validate() error {
	// Validate PostgreSQL port
	if c.PGPort <= 0 || c.PGPort > 65535 {
		return &ConfigError{
			Field:      "port",
			Value:      c.PGPort,
			Message:    fmt.Sprintf("invalid port number: %d", c.PGPort),
			Suggestion: "Port must be between 1 and 65535. Default PostgreSQL port is 5432. Set via --port flag or PGPORT environment variable.",
		}
	}

	// Validate timeout
	if c.Timeout <= 0 {
		return &ConfigError{
			Field:      "timeout",
			Value:      c.Timeout,
			Message:    "timeout must be positive",
			Suggestion: "Use --timeout flag with format like '30s', '1m', '90s'. Default is 30s.",
		}
	}

	// Validate parallelism
	if c.Parallelism < 1 {
		return &ConfigError{
			Field:      "parallel",
			Value:      c.Parallelism,
			Message:    fmt.Sprintf("parallelism must be at least 1, got: %d", c.Parallelism),
			Suggestion: "Use --parallel=N where N is number of tests to run concurrently. Use 1 for sequential execution.",
		}
	}

	if c.Parallelism > 100 {
		return &ConfigError{
			Field:      "parallel",
			Value:      c.Parallelism,
			Message:    fmt.Sprintf("parallelism too high: %d", c.Parallelism),
			Suggestion: "Consider a lower value to avoid overwhelming PostgreSQL connection limits. Recommended maximum: 100.",
		}
	}

	// Validate required fields
	if c.PGHost == "" {
		return &ConfigError{
			Field:      "host",
			Message:    "PostgreSQL host is required",
			Suggestion: "Set via --host flag or PGHOST environment variable. Default is 'localhost'.",
		}
	}

	if c.PGDatabase == "" {
		return &ConfigError{
			Field:      "database",
			Message:    "template database is required",
			Suggestion: "Set via --database flag or PGDATABASE environment variable. Default is 'postgres'.",
		}
	}

	if c.CoverageFile == "" {
		return &ConfigError{
			Field:      "coverage-file",
			Message:    "coverage file path is required",
			Suggestion: "Set via --coverage-file flag. Default is '.pgcov/coverage.json'.",
		}
	}

	return nil
}

// CoverageSignal represents a single coverage signal emitted via NOTIFY
type CoverageSignal struct {
	SignalID  string    // Matches CoveragePoint.SignalID
	Timestamp time.Time // When signal received
}

// TempDatabase represents a temporary PostgreSQL database for test isolation
type TempDatabase struct {
	Name             string // e.g., "pgcov_test_20260105_a3f9c2b1"
	CreatedAt        time.Time
	ConnectionString string
}

