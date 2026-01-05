package cli

import (
	"os"
	"strconv"
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

// DefaultConfig provides default configuration values
var DefaultConfig = Config{
	PGHost:       "localhost",
	PGPort:       5432,
	PGDatabase:   "postgres",
	Timeout:      30 * time.Second,
	Parallelism:  1,
	CoverageFile: ".pgcov/coverage.json",
	Verbose:      false,
}

// LoadConfig creates a configuration by layering flags → env vars → defaults
// Priority: flags override env vars override defaults
func LoadConfig() *Config {
	cfg := DefaultConfig

	// Load from environment variables
	if host := os.Getenv("PGHOST"); host != "" {
		cfg.PGHost = host
	}
	if portStr := os.Getenv("PGPORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			cfg.PGPort = port
		}
	}
	if user := os.Getenv("PGUSER"); user != "" {
		cfg.PGUser = user
	}
	if password := os.Getenv("PGPASSWORD"); password != "" {
		cfg.PGPassword = password
	}
	if database := os.Getenv("PGDATABASE"); database != "" {
		cfg.PGDatabase = database
	}

	return &cfg
}

// ApplyFlags applies command-line flag values to configuration
func (c *Config) ApplyFlags(host string, port int, user, password, database string,
	timeout time.Duration, parallel int, coverageFile string, verbose bool) {

	if host != "" {
		c.PGHost = host
	}
	if port != 0 {
		c.PGPort = port
	}
	if user != "" {
		c.PGUser = user
	}
	if password != "" {
		c.PGPassword = password
	}
	if database != "" {
		c.PGDatabase = database
	}
	if timeout != 0 {
		c.Timeout = timeout
	}
	if parallel != 0 {
		c.Parallelism = parallel
	}
	if coverageFile != "" {
		c.CoverageFile = coverageFile
	}
	c.Verbose = verbose
}
