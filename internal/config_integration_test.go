package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pashagolub/pgcov/internal/cli"
	"github.com/pashagolub/pgcov/pkg/types"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestConfigurationWithTestcontainers verifies configuration loading and validation
// with a real PostgreSQL instance using testcontainers
func TestConfigurationWithTestcontainers(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	t.Log("Starting PostgreSQL container for configuration tests...")
	pgContainer, err := postgres.Run(ctx,
		"docker.io/postgres:16-alpine",
		postgres.WithDatabase("config_test"),
		postgres.WithUsername("configuser"),
		postgres.WithPassword("configpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Get connection details
	host, err := pgContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}

	port, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

	t.Logf("PostgreSQL running at %s:%s", host, port.Port())

	// Test 1: Configuration from flags
	t.Run("ConfigurationFromFlags", func(t *testing.T) {
		config := &types.Config{
			PGHost:       host,
			PGPort:       port.Int(),
			PGUser:       "configuser",
			PGPassword:   "configpass",
			PGDatabase:   "config_test",
			Timeout:      45 * time.Second,
			Parallelism:  2,
			CoverageFile: ".pgcov/custom-coverage.json",
			Verbose:      true,
		}

		// Validate configuration
		if err := config.Validate(); err != nil {
			t.Errorf("Valid configuration failed validation: %v", err)
		}

		// Verify values were set correctly
		if config.PGHost != host {
			t.Errorf("Expected host %s, got %s", host, config.PGHost)
		}
		if config.Timeout != 45*time.Second {
			t.Errorf("Expected timeout 45s, got %v", config.Timeout)
		}
		if config.Parallelism != 2 {
			t.Errorf("Expected parallelism 2, got %d", config.Parallelism)
		}
	})

	// Test 2: Configuration from environment variables
	t.Run("ConfigurationFromEnvVars", func(t *testing.T) {
		// Set environment variables
		os.Setenv("PGHOST", host)
		os.Setenv("PGPORT", port.Port())
		os.Setenv("PGUSER", "configuser")
		os.Setenv("PGPASSWORD", "configpass")
		os.Setenv("PGDATABASE", "config_test")
		defer func() {
			os.Unsetenv("PGHOST")
			os.Unsetenv("PGPORT")
			os.Unsetenv("PGUSER")
			os.Unsetenv("PGPASSWORD")
			os.Unsetenv("PGDATABASE")
		}()

		// Load configuration (should pick up env vars)
		config := cli.LoadConfig()

		// Validate
		if err := config.Validate(); err != nil {
			t.Errorf("Configuration from env vars failed validation: %v", err)
		}

		// Verify values
		if config.PGHost != host {
			t.Errorf("Expected host from env %s, got %s", host, config.PGHost)
		}
		if config.PGUser != "configuser" {
			t.Errorf("Expected user from env 'configuser', got %s", config.PGUser)
		}
	})

	// Test 3: Configuration priority (flags override env vars)
	t.Run("ConfigurationPriority", func(t *testing.T) {
		// Set environment variables
		os.Setenv("PGHOST", "envhost")
		os.Setenv("PGPORT", "9999")
		defer func() {
			os.Unsetenv("PGHOST")
			os.Unsetenv("PGPORT")
		}()

		// Load config from env
		config := cli.LoadConfig()

		// Apply flags (should override env)
		cli.ApplyFlagsToConfig(config, host, port.Int(), "flaguser", "flagpass", "config_test",
			60*time.Second, 3, "flag-coverage.json", true)

		// Verify flags took precedence
		if config.PGHost != host {
			t.Errorf("Expected flag host %s to override env, got %s", host, config.PGHost)
		}
		if config.PGPort != port.Int() {
			t.Errorf("Expected flag port %d to override env, got %d", port.Int(), config.PGPort)
		}
		if config.PGUser != "flaguser" {
			t.Errorf("Expected flag user 'flaguser', got %s", config.PGUser)
		}
		if config.Timeout != 60*time.Second {
			t.Errorf("Expected flag timeout 60s, got %v", config.Timeout)
		}
		if config.Parallelism != 3 {
			t.Errorf("Expected flag parallelism 3, got %d", config.Parallelism)
		}
	})

	// Test 4: Configuration validation errors
	t.Run("ConfigurationValidation", func(t *testing.T) {
		tests := []struct {
			name       string
			modifyFunc func(*types.Config)
			expectErr  bool
		}{
			{
				name: "valid config",
				modifyFunc: func(c *types.Config) {
					// No modifications - should be valid
				},
				expectErr: false,
			},
			{
				name: "invalid port - zero",
				modifyFunc: func(c *types.Config) {
					c.PGPort = 0
				},
				expectErr: true,
			},
			{
				name: "invalid port - too high",
				modifyFunc: func(c *types.Config) {
					c.PGPort = 99999
				},
				expectErr: true,
			},
			{
				name: "invalid timeout",
				modifyFunc: func(c *types.Config) {
					c.Timeout = -1 * time.Second
				},
				expectErr: true,
			},
			{
				name: "invalid parallelism - zero",
				modifyFunc: func(c *types.Config) {
					c.Parallelism = 0
				},
				expectErr: true,
			},
			{
				name: "invalid parallelism - too high",
				modifyFunc: func(c *types.Config) {
					c.Parallelism = 101
				},
				expectErr: true,
			},
			{
				name: "empty host",
				modifyFunc: func(c *types.Config) {
					c.PGHost = ""
				},
				expectErr: true,
			},
			{
				name: "empty database",
				modifyFunc: func(c *types.Config) {
					c.PGDatabase = ""
				},
				expectErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				config := &types.Config{
					PGHost:       host,
					PGPort:       port.Int(),
					PGUser:       "configuser",
					PGPassword:   "configpass",
					PGDatabase:   "config_test",
					Timeout:      30 * time.Second,
					Parallelism:  1,
					CoverageFile: ".pgcov/coverage.json",
				}

				tt.modifyFunc(config)

				err := config.Validate()
				if tt.expectErr && err == nil {
					t.Errorf("Expected validation error but got none")
				}
				if !tt.expectErr && err != nil {
					t.Errorf("Expected no validation error but got: %v", err)
				}
			})
		}
	})

	// Test 5: ConfigError provides helpful suggestions
	t.Run("ConfigErrorSuggestions", func(t *testing.T) {
		config := &types.Config{
			PGHost:       host,
			PGPort:       99999, // Invalid port
			PGUser:       "configuser",
			PGPassword:   "configpass",
			PGDatabase:   "config_test",
			Timeout:      30 * time.Second,
			Parallelism:  1,
			CoverageFile: ".pgcov/coverage.json",
		}

		err := config.Validate()
		if err == nil {
			t.Fatal("Expected validation error for invalid port")
		}

		// Check that error message contains helpful information
		errStr := err.Error()
		if !contains(errStr, "port") {
			t.Errorf("Error message should mention 'port', got: %s", errStr)
		}
		if !contains(errStr, "Suggestion") {
			t.Errorf("Error message should contain suggestion, got: %s", errStr)
		}
		if !contains(errStr, "65535") {
			t.Errorf("Error message should mention valid port range, got: %s", errStr)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
