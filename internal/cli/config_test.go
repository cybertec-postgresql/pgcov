package cli

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Clear environment variables
	clearPGEnvVars(t)

	cfg := LoadConfig()

	if cfg.PGHost != "localhost" {
		t.Errorf("expected default host 'localhost', got '%s'", cfg.PGHost)
	}
	if cfg.PGPort != 5432 {
		t.Errorf("expected default port 5432, got %d", cfg.PGPort)
	}
	if cfg.PGDatabase != "postgres" {
		t.Errorf("expected default database 'postgres', got '%s'", cfg.PGDatabase)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", cfg.Timeout)
	}
	if cfg.Parallelism != 1 {
		t.Errorf("expected default parallelism 1, got %d", cfg.Parallelism)
	}
	if cfg.CoverageFile != ".pgcov/coverage.json" {
		t.Errorf("expected default coverage file '.pgcov/coverage.json', got '%s'", cfg.CoverageFile)
	}
	if cfg.Verbose != false {
		t.Errorf("expected default verbose false, got %v", cfg.Verbose)
	}
}

func TestLoadConfig_EnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("PGHOST", "testhost")
	os.Setenv("PGPORT", "5433")
	os.Setenv("PGUSER", "testuser")
	os.Setenv("PGPASSWORD", "testpass")
	os.Setenv("PGDATABASE", "testdb")
	defer clearPGEnvVars(t)

	cfg := LoadConfig()

	if cfg.PGHost != "testhost" {
		t.Errorf("expected host from env 'testhost', got '%s'", cfg.PGHost)
	}
	if cfg.PGPort != 5433 {
		t.Errorf("expected port from env 5433, got %d", cfg.PGPort)
	}
	if cfg.PGUser != "testuser" {
		t.Errorf("expected user from env 'testuser', got '%s'", cfg.PGUser)
	}
	if cfg.PGPassword != "testpass" {
		t.Errorf("expected password from env 'testpass', got '%s'", cfg.PGPassword)
	}
	if cfg.PGDatabase != "testdb" {
		t.Errorf("expected database from env 'testdb', got '%s'", cfg.PGDatabase)
	}
}

func TestApplyFlagsToConfig_OverridesEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("PGHOST", "envhost")
	os.Setenv("PGPORT", "5433")
	defer clearPGEnvVars(t)

	cfg := LoadConfig()

	// Apply flags (should override env vars)
	ApplyFlagsToConfig(cfg, "flaghost", 5434, "flaguser", "flagpass", "flagdb",
		60*time.Second, 4, "custom.json", true)

	if cfg.PGHost != "flaghost" {
		t.Errorf("expected host from flag 'flaghost', got '%s'", cfg.PGHost)
	}
	if cfg.PGPort != 5434 {
		t.Errorf("expected port from flag 5434, got %d", cfg.PGPort)
	}
	if cfg.PGUser != "flaguser" {
		t.Errorf("expected user from flag 'flaguser', got '%s'", cfg.PGUser)
	}
	if cfg.PGPassword != "flagpass" {
		t.Errorf("expected password from flag 'flagpass', got '%s'", cfg.PGPassword)
	}
	if cfg.PGDatabase != "flagdb" {
		t.Errorf("expected database from flag 'flagdb', got '%s'", cfg.PGDatabase)
	}
	if cfg.Timeout != 60*time.Second {
		t.Errorf("expected timeout from flag 60s, got %v", cfg.Timeout)
	}
	if cfg.Parallelism != 4 {
		t.Errorf("expected parallelism from flag 4, got %d", cfg.Parallelism)
	}
	if cfg.CoverageFile != "custom.json" {
		t.Errorf("expected coverage file from flag 'custom.json', got '%s'", cfg.CoverageFile)
	}
	if cfg.Verbose != true {
		t.Errorf("expected verbose from flag true, got %v", cfg.Verbose)
	}
}

func TestApplyFlagsToConfig_EmptyFlagsPreserveConfig(t *testing.T) {
	cfg := &Config{
		PGHost:       "originalhost",
		PGPort:       5433,
		PGUser:       "originaluser",
		PGPassword:   "originalpass",
		PGDatabase:   "originaldb",
		Timeout:      45 * time.Second,
		Parallelism:  2,
		CoverageFile: "original.json",
		Verbose:      false,
	}

	// Apply empty flags (should not change config)
	ApplyFlagsToConfig(cfg, "", 0, "", "", "", 0, 0, "", false)

	if cfg.PGHost != "originalhost" {
		t.Errorf("empty flag should not change host")
	}
	if cfg.PGPort != 5433 {
		t.Errorf("zero flag should not change port")
	}
	if cfg.PGUser != "originaluser" {
		t.Errorf("empty flag should not change user")
	}
	if cfg.Timeout != 45*time.Second {
		t.Errorf("zero flag should not change timeout")
	}
}

func TestConfigValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		PGHost:       "localhost",
		PGPort:       5432,
		PGDatabase:   "postgres",
		Timeout:      30 * time.Second,
		Parallelism:  1,
		CoverageFile: ".pgcov/coverage.json",
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("valid config should not return error: %v", err)
	}
}

func TestConfigValidate_InvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"zero port", 0},
		{"negative port", -1},
		{"port too high", 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				PGHost:       "localhost",
				PGPort:       tt.port,
				PGDatabase:   "postgres",
				Timeout:      30 * time.Second,
				Parallelism:  1,
				CoverageFile: ".pgcov/coverage.json",
			}

			err := cfg.Validate()
			if err == nil {
				t.Errorf("expected validation error for port %d", tt.port)
			}

			configErr, ok := err.(*ConfigError)
			if !ok {
				t.Errorf("expected ConfigError, got %T", err)
			}
			if configErr.Field != "port" {
				t.Errorf("expected error field 'port', got '%s'", configErr.Field)
			}
		})
	}
}

func TestConfigValidate_InvalidTimeout(t *testing.T) {
	cfg := &Config{
		PGHost:       "localhost",
		PGPort:       5432,
		PGDatabase:   "postgres",
		Timeout:      -1 * time.Second,
		Parallelism:  1,
		CoverageFile: ".pgcov/coverage.json",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for negative timeout")
	}

	configErr, ok := err.(*ConfigError)
	if !ok {
		t.Errorf("expected ConfigError, got %T", err)
	}
	if configErr.Field != "timeout" {
		t.Errorf("expected error field 'timeout', got '%s'", configErr.Field)
	}
}

func TestConfigValidate_InvalidParallelism(t *testing.T) {
	tests := []struct {
		name        string
		parallelism int
	}{
		{"zero parallelism", 0},
		{"negative parallelism", -1},
		{"too high parallelism", 101},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				PGHost:       "localhost",
				PGPort:       5432,
				PGDatabase:   "postgres",
				Timeout:      30 * time.Second,
				Parallelism:  tt.parallelism,
				CoverageFile: ".pgcov/coverage.json",
			}

			err := cfg.Validate()
			if err == nil {
				t.Errorf("expected validation error for parallelism %d", tt.parallelism)
			}

			configErr, ok := err.(*ConfigError)
			if !ok {
				t.Errorf("expected ConfigError, got %T", err)
			}
			if configErr.Field != "parallel" {
				t.Errorf("expected error field 'parallel', got '%s'", configErr.Field)
			}
		})
	}
}

func TestConfigValidate_EmptyRequiredFields(t *testing.T) {
	tests := []struct {
		name          string
		modifyConfig  func(*Config)
		expectedField string
	}{
		{
			name: "empty host",
			modifyConfig: func(c *Config) {
				c.PGHost = ""
			},
			expectedField: "host",
		},
		{
			name: "empty database",
			modifyConfig: func(c *Config) {
				c.PGDatabase = ""
			},
			expectedField: "database",
		},
		{
			name: "empty coverage file",
			modifyConfig: func(c *Config) {
				c.CoverageFile = ""
			},
			expectedField: "coverage-file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				PGHost:       "localhost",
				PGPort:       5432,
				PGDatabase:   "postgres",
				Timeout:      30 * time.Second,
				Parallelism:  1,
				CoverageFile: ".pgcov/coverage.json",
			}

			tt.modifyConfig(cfg)

			err := cfg.Validate()
			if err == nil {
				t.Errorf("expected validation error for empty %s", tt.expectedField)
			}

			configErr, ok := err.(*ConfigError)
			if !ok {
				t.Errorf("expected ConfigError, got %T", err)
			}
			if configErr.Field != tt.expectedField {
				t.Errorf("expected error field '%s', got '%s'", tt.expectedField, configErr.Field)
			}
			if configErr.Suggestion == "" {
				t.Error("expected suggestion to be provided")
			}
		})
	}
}

func TestConfigError_Error(t *testing.T) {
	err := &ConfigError{
		Field:      "port",
		Value:      99999,
		Message:    "invalid port number: 99999",
		Suggestion: "Port must be between 1 and 65535.",
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("error string should not be empty")
	}

	// Check that error contains field, message, and suggestion
	expectedSubstrings := []string{"port", "invalid port number", "Suggestion"}
	for _, substr := range expectedSubstrings {
		if !contains(errStr, substr) {
			t.Errorf("error string should contain '%s', got: %s", substr, errStr)
		}
	}
}

// Helper functions

func clearPGEnvVars(t *testing.T) {
	t.Helper()
	os.Unsetenv("PGHOST")
	os.Unsetenv("PGPORT")
	os.Unsetenv("PGUSER")
	os.Unsetenv("PGPASSWORD")
	os.Unsetenv("PGDATABASE")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
