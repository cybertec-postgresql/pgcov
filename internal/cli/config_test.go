package cli

import (
	"testing"
	"time"
)

func TestApplyFlagsToConfig_EmptyFlagsPreserveConfig(t *testing.T) {
	originalConnString := "host=originalhost port=5433 user=originaluser dbname=originaldb"
	cfg := &Config{
		ConnectionString: originalConnString,
		Timeout:          45 * time.Second,
		Parallelism:      2,
		CoverageFile:     "original.json",
		Verbose:          false,
	}

	// Apply empty flags (should not change config)
	ApplyFlagsToConfig(cfg, "", 0, 0, "", false)

	if cfg.ConnectionString != originalConnString {
		t.Errorf("empty flag should not change connection string")
	}
	if cfg.Timeout != 45*time.Second {
		t.Errorf("zero flag should not change timeout")
	}
	if cfg.Parallelism != 2 {
		t.Errorf("zero flag should not change parallelism")
	}
}

func TestConfigValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		ConnectionString: "host=localhost port=5432 dbname=postgres",
		Timeout:          30 * time.Second,
		Parallelism:      1,
		CoverageFile:     ".pgcov/coverage.json",
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("valid config should not return error: %v", err)
	}
}

func TestConfigValidate_EmptyConnectionString(t *testing.T) {
	cfg := &Config{
		ConnectionString: "",
		Timeout:          30 * time.Second,
		Parallelism:      1,
		CoverageFile:     ".pgcov/coverage.json",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for empty connection string")
	}

	configErr, ok := err.(*ConfigError)
	if !ok {
		t.Errorf("expected ConfigError, got %T", err)
	}
	if configErr.Field != "connection" {
		t.Errorf("expected error field 'connection', got '%s'", configErr.Field)
	}
}

func TestConfigValidate_InvalidTimeout(t *testing.T) {
	cfg := &Config{
		ConnectionString: "host=localhost port=5432 dbname=postgres",
		Timeout:          -1 * time.Second,
		Parallelism:      1,
		CoverageFile:     ".pgcov/coverage.json",
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
				ConnectionString: "host=localhost port=5432 dbname=postgres",
				Timeout:          30 * time.Second,
				Parallelism:      tt.parallelism,
				CoverageFile:     ".pgcov/coverage.json",
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

func TestConfigValidate_EmptyCoverageFile(t *testing.T) {
	cfg := &Config{
		ConnectionString: "host=localhost port=5432 dbname=postgres",
		Timeout:          30 * time.Second,
		Parallelism:      1,
		CoverageFile:     "",
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error for empty coverage file")
	}

	configErr, ok := err.(*ConfigError)
	if !ok {
		t.Errorf("expected ConfigError, got %T", err)
	}
	if configErr.Field != "coverage-file" {
		t.Errorf("expected error field 'coverage-file', got '%s'", configErr.Field)
	}
	if configErr.Suggestion == "" {
		t.Error("expected suggestion to be provided")
	}
}

func TestConfigError_Error(t *testing.T) {
	err := &ConfigError{
		Field:      "connection",
		Value:      "",
		Message:    "PostgreSQL connection string is required",
		Suggestion: "Set via --connection flag or standard PG* environment variables.",
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("error string should not be empty")
	}

	// Check that error contains field, message, and suggestion
	expectedSubstrings := []string{"connection", "PostgreSQL connection string is required", "Suggestion"}
	for _, substr := range expectedSubstrings {
		if !contains(errStr, substr) {
			t.Errorf("error string should contain '%s', got: %s", substr, errStr)
		}
	}
}

// Helper functions

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
