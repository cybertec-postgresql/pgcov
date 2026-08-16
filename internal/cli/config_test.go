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
		SignalTimeout:    250 * time.Millisecond,
		Parallelism:      2,
		CoverageFile:     "original.json",
		Verbose:          false,
	}

	// Apply empty flags (should not change config)
	ApplyFlagsToConfig(cfg, "", 0, 0, 0, "", false, nil, 0)

	if cfg.ConnectionString != originalConnString {
		t.Errorf("empty flag should not change connection string")
	}
	if cfg.Timeout != 45*time.Second {
		t.Errorf("zero flag should not change timeout")
	}
	if cfg.Parallelism != 2 {
		t.Errorf("zero flag should not change parallelism")
	}
	if len(cfg.SetupFiles) != 0 {
		t.Errorf("nil setup flag should not set setup files")
	}
	if cfg.SignalTimeout != 250*time.Millisecond {
		t.Errorf("zero flag should not change signal timeout")
	}
}

func TestApplyFlagsToConfig_SetupFiles(t *testing.T) {
	cfg := &Config{}
	ApplyFlagsToConfig(cfg, "", 0, 0, 0, "", false, []string{"a.sql", "glob/*.sql"}, 0)
	if len(cfg.SetupFiles) != 2 {
		t.Fatalf("expected 2 setup files, got %d", len(cfg.SetupFiles))
	}
}

func TestApplyFlagsToConfig_SignalTimeout(t *testing.T) {
	cfg := &Config{SignalTimeout: 100 * time.Millisecond}
	ApplyFlagsToConfig(cfg, "", 0, 500*time.Millisecond, 0, "", false, nil, 0)
	if cfg.SignalTimeout != 500*time.Millisecond {
		t.Fatalf("expected signal timeout 500ms, got %v", cfg.SignalTimeout)
	}
}

func TestApplyFlagsToConfig_SignalTimeoutZeroPreservesDefault(t *testing.T) {
	cfg := &Config{SignalTimeout: 250 * time.Millisecond}
	ApplyFlagsToConfig(cfg, "", 0, 0, 0, "", false, nil, 0)
	if cfg.SignalTimeout != 250*time.Millisecond {
		t.Fatalf("zero flag must preserve signal timeout, got %v", cfg.SignalTimeout)
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

func TestConfigValidate_FailUnder(t *testing.T) {
	tests := []struct {
		name      string
		failUnder float64
		wantErr   bool
	}{
		{"zero disables", 0, false},
		{"valid boundary zero", 0.0, false},
		{"valid mid", 80.5, false},
		{"valid boundary hundred", 100, false},
		{"negative invalid", -1, true},
		{"too high invalid", 100.01, true},
		{"way too high invalid", 250, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ConnectionString: "host=localhost port=5432 dbname=postgres",
				Timeout:          30 * time.Second,
				Parallelism:      1,
				CoverageFile:     ".pgcov/coverage.json",
				FailUnder:        tt.failUnder,
			}

			err := cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected validation error for fail-under=%g", tt.failUnder)
				}
				configErr, ok := err.(*ConfigError)
				if !ok {
					t.Fatalf("expected ConfigError, got %T", err)
				}
				if configErr.Field != "fail-under" {
					t.Errorf("expected field 'fail-under', got %q", configErr.Field)
				}
			} else if err != nil {
				t.Errorf("unexpected error for fail-under=%g: %v", tt.failUnder, err)
			}
		})
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
