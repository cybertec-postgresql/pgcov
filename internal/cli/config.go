package cli

import (
	"time"

	"github.com/cybertec-postgresql/pgcov/pkg/types"
)

// Config is an alias for the shared Config type
type Config = types.Config

// ConfigError is an alias for the shared ConfigError type
type ConfigError = types.ConfigError

// DefaultConfig provides default configuration values
var DefaultConfig = Config{
	ConnectionString: "",
	Timeout:          30 * time.Second,
	SignalTimeout:    100 * time.Millisecond,
	Parallelism:      1,
	CoverageFile:     ".pgcov/coverage.json",
	Verbose:          false,
}

// ApplyFlagsToConfig applies command-line flag values to configuration
func ApplyFlagsToConfig(c *Config, connection string, timeout time.Duration,
	signalTimeout time.Duration, parallel int, coverageFile string, verbose bool, setupFiles []string) {
	if connection != "" {
		c.ConnectionString = connection
	}
	if timeout != 0 {
		c.Timeout = timeout
	}
	if signalTimeout != 0 {
		c.SignalTimeout = signalTimeout
	}
	if parallel != 0 {
		c.Parallelism = parallel
	}
	if coverageFile != "" {
		c.CoverageFile = coverageFile
	}
	if len(setupFiles) > 0 {
		c.SetupFiles = setupFiles
	}
	c.Verbose = verbose
}
