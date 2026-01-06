package types

import "time"

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

