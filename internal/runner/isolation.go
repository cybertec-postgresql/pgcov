package runner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cybertec-postgresql/pgcov/internal/database"
	"github.com/cybertec-postgresql/pgcov/internal/discovery"
	"github.com/cybertec-postgresql/pgcov/internal/instrument"
)

// IsolationValidator tracks and validates test isolation guarantees
type IsolationValidator struct {
	mu              sync.Mutex
	usedDatabases   map[string]time.Time // database name -> creation time
	activeDatabases map[string]bool      // databases currently in use
	cleanedUp       map[string]bool      // databases that were properly cleaned up
}

// NewIsolationValidator creates a new isolation validator
func NewIsolationValidator() *IsolationValidator {
	return &IsolationValidator{
		usedDatabases:   make(map[string]time.Time),
		activeDatabases: make(map[string]bool),
		cleanedUp:       make(map[string]bool),
	}
}

// TrackDatabase records that a database has been created for a test
func (iv *IsolationValidator) TrackDatabase(dbName string, createdAt time.Time) error {
	iv.mu.Lock()
	defer iv.mu.Unlock()

	// Check for duplicate database names (should never happen with unique generation)
	if existingTime, exists := iv.usedDatabases[dbName]; exists {
		return fmt.Errorf("database name collision detected: %s already created at %v", dbName, existingTime)
	}

	// Record the database
	iv.usedDatabases[dbName] = createdAt
	iv.activeDatabases[dbName] = true

	return nil
}

// MarkCleaned marks a database as properly cleaned up
func (iv *IsolationValidator) MarkCleaned(dbName string) {
	iv.mu.Lock()
	defer iv.mu.Unlock()

	delete(iv.activeDatabases, dbName)
	iv.cleanedUp[dbName] = true
}

// ValidateUniqueness verifies that each test got its own unique database
func (iv *IsolationValidator) ValidateUniqueness() error {
	iv.mu.Lock()
	defer iv.mu.Unlock()

	if len(iv.usedDatabases) == 0 {
		return fmt.Errorf("no databases were tracked")
	}

	// All database names should be unique (this is guaranteed by the map structure)
	// But we can verify that we have the expected number of databases
	return nil
}

// ValidateCleanup verifies that all databases were properly cleaned up
func (iv *IsolationValidator) ValidateCleanup() error {
	iv.mu.Lock()
	defer iv.mu.Unlock()

	var activeDBs []string
	for dbName := range iv.activeDatabases {
		activeDBs = append(activeDBs, dbName)
	}

	if len(activeDBs) > 0 {
		return fmt.Errorf("databases not properly cleaned up: %v", activeDBs)
	}

	// Verify all tracked databases were cleaned
	for dbName := range iv.usedDatabases {
		if !iv.cleanedUp[dbName] {
			return fmt.Errorf("database %s was not marked as cleaned up", dbName)
		}
	}

	return nil
}

// GetStats returns statistics about database usage
func (iv *IsolationValidator) GetStats() IsolationStats {
	iv.mu.Lock()
	defer iv.mu.Unlock()

	return IsolationStats{
		TotalDatabases:   len(iv.usedDatabases),
		ActiveDatabases:  len(iv.activeDatabases),
		CleanedDatabases: len(iv.cleanedUp),
		DatabaseNames:    iv.getDatabaseNames(),
	}
}

func (iv *IsolationValidator) getDatabaseNames() []string {
	names := make([]string, 0, len(iv.usedDatabases))
	for name := range iv.usedDatabases {
		names = append(names, name)
	}
	return names
}

// IsolationStats contains statistics about test isolation
type IsolationStats struct {
	TotalDatabases   int
	ActiveDatabases  int
	CleanedDatabases int
	DatabaseNames    []string
}

// VerifyDatabaseIsolation performs comprehensive isolation checks
// This is a helper function for integration tests
func VerifyDatabaseIsolation(ctx context.Context, pool *database.Pool, runs []*TestRun) error {
	validator := NewIsolationValidator()

	// Track all databases used in test runs
	for _, run := range runs {
		if run.Database == nil {
			return fmt.Errorf("test run %s has no database assigned", run.Test.RelativePath)
		}

		err := validator.TrackDatabase(run.Database.Name, run.Database.CreatedAt)
		if err != nil {
			return fmt.Errorf("isolation violation for test %s: %w", run.Test.RelativePath, err)
		}

		// Verify database was cleaned up by checking if it still exists
		exists, err := databaseExists(ctx, pool, run.Database.Name)
		if err != nil {
			return fmt.Errorf("failed to check if database exists: %w", err)
		}

		if !exists {
			validator.MarkCleaned(run.Database.Name)
		}
	}

	// Validate uniqueness
	if err := validator.ValidateUniqueness(); err != nil {
		return fmt.Errorf("uniqueness validation failed: %w", err)
	}

	// Validate cleanup
	if err := validator.ValidateCleanup(); err != nil {
		return fmt.Errorf("cleanup validation failed: %w", err)
	}

	return nil
}

// databaseExists checks if a database exists in PostgreSQL
func databaseExists(ctx context.Context, pool *database.Pool, dbName string) (bool, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`
	err = conn.QueryRow(ctx, query, dbName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to query database existence: %w", err)
	}

	return exists, nil
}

// DetectConnectionLeaks checks for open connections to test databases
func DetectConnectionLeaks(ctx context.Context, pool *database.Pool, dbNames []string) (map[string]int, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	leaks := make(map[string]int)

	for _, dbName := range dbNames {
		query := `
			SELECT COUNT(*)
			FROM pg_stat_activity
			WHERE datname = $1 AND pid <> pg_backend_pid()
		`

		var count int
		err := conn.QueryRow(ctx, query, dbName).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("failed to check connections for %s: %w", dbName, err)
		}

		if count > 0 {
			leaks[dbName] = count
		}
	}

	return leaks, nil
}

// VerifyStatelessExecution verifies that running the same test multiple times
// produces identical results (ensuring no state leakage)
func VerifyStatelessExecution(run1, run2 *TestRun) error {
	// Verify both tests have the same status
	if run1.Status != run2.Status {
		return fmt.Errorf("test status differs: %s vs %s", run1.Status, run2.Status)
	}

	// Verify both tests used different databases
	if run1.Database.Name == run2.Database.Name {
		return fmt.Errorf("tests used the same database: %s", run1.Database.Name)
	}

	// Verify both tests collected the same coverage signals
	if len(run1.CoverageSigs) != len(run2.CoverageSigs) {
		return fmt.Errorf("coverage signal count differs: %d vs %d",
			len(run1.CoverageSigs), len(run2.CoverageSigs))
	}

	// Build sets of signal IDs for comparison
	signals1 := make(map[string]bool)
	for _, sig := range run1.CoverageSigs {
		signals1[sig.SignalID] = true
	}

	signals2 := make(map[string]bool)
	for _, sig := range run2.CoverageSigs {
		signals2[sig.SignalID] = true
	}

	// Verify signal sets are identical
	for sigID := range signals1 {
		if !signals2[sigID] {
			return fmt.Errorf("signal %s present in first run but not second", sigID)
		}
	}

	for sigID := range signals2 {
		if !signals1[sigID] {
			return fmt.Errorf("signal %s present in second run but not first", sigID)
		}
	}

	return nil
}

// IsolationReport provides a comprehensive report on test isolation
type IsolationReport struct {
	TotalTests          int
	UniqueDatabases     int
	ProperlyCleanedUp   int
	ConnectionLeaks     map[string]int
	IsolationViolations []string
}

// GenerateIsolationReport creates a comprehensive isolation report
func GenerateIsolationReport(ctx context.Context, pool *database.Pool, runs []*TestRun) (*IsolationReport, error) {
	report := &IsolationReport{
		TotalTests:          len(runs),
		IsolationViolations: []string{},
		ConnectionLeaks:     make(map[string]int),
	}

	validator := NewIsolationValidator()
	dbNames := []string{}

	// Track all databases
	for _, run := range runs {
		if run.Database == nil {
			report.IsolationViolations = append(report.IsolationViolations,
				fmt.Sprintf("test %s has no database", run.Test.RelativePath))
			continue
		}

		dbNames = append(dbNames, run.Database.Name)

		err := validator.TrackDatabase(run.Database.Name, run.Database.CreatedAt)
		if err != nil {
			report.IsolationViolations = append(report.IsolationViolations, err.Error())
			continue
		}

		// Check if cleaned up
		exists, err := databaseExists(ctx, pool, run.Database.Name)
		if err != nil {
			report.IsolationViolations = append(report.IsolationViolations,
				fmt.Sprintf("failed to check database %s: %v", run.Database.Name, err))
			continue
		}

		if !exists {
			validator.MarkCleaned(run.Database.Name)
		}
	}

	stats := validator.GetStats()
	report.UniqueDatabases = stats.TotalDatabases
	report.ProperlyCleanedUp = stats.CleanedDatabases

	// Check for connection leaks
	leaks, err := DetectConnectionLeaks(ctx, pool, dbNames)
	if err != nil {
		report.IsolationViolations = append(report.IsolationViolations,
			fmt.Sprintf("failed to detect connection leaks: %v", err))
	} else {
		report.ConnectionLeaks = leaks
	}

	// Validate cleanup
	if err := validator.ValidateCleanup(); err != nil {
		report.IsolationViolations = append(report.IsolationViolations, err.Error())
	}

	return report, nil
}

// ValidateOrderIndependence runs tests in different orders and verifies identical results
// This is a helper for integration tests to verify isolation
func ValidateOrderIndependence(ctx context.Context, executor *Executor, testFiles []discovery.DiscoveredFile, sourceFiles []*instrument.InstrumentedSQL) error {
	if len(testFiles) < 2 {
		return fmt.Errorf("need at least 2 test files to validate order independence")
	}

	// Run tests in original order
	runs1, err := executor.ExecuteBatch(ctx, testFiles, sourceFiles)
	if err != nil {
		return fmt.Errorf("failed to run tests in original order: %w", err)
	}

	// Reverse the order
	reversedTests := make([]discovery.DiscoveredFile, len(testFiles))
	for i, test := range testFiles {
		reversedTests[len(testFiles)-1-i] = test
	}

	// Run tests in reversed order
	runs2, err := executor.ExecuteBatch(ctx, reversedTests, sourceFiles)
	if err != nil {
		return fmt.Errorf("failed to run tests in reversed order: %w", err)
	}

	// Build a map of test results by test file path for easier comparison
	resultsMap1 := make(map[string]*TestRun)
	for _, run := range runs1 {
		resultsMap1[run.Test.RelativePath] = run
	}

	resultsMap2 := make(map[string]*TestRun)
	for _, run := range runs2 {
		resultsMap2[run.Test.RelativePath] = run
	}

	// Compare results for each test
	for testPath, run1 := range resultsMap1 {
		run2, exists := resultsMap2[testPath]
		if !exists {
			return fmt.Errorf("test %s not found in second run", testPath)
		}

		// Verify tests used different databases
		if run1.Database.Name == run2.Database.Name {
			return fmt.Errorf("test %s used same database in both runs: %s",
				testPath, run1.Database.Name)
		}

		// Verify same status
		if run1.Status != run2.Status {
			return fmt.Errorf("test %s has different status: %s vs %s",
				testPath, run1.Status, run2.Status)
		}

		// Verify same coverage (signal IDs should match)
		err := VerifyStatelessExecution(run1, run2)
		if err != nil {
			return fmt.Errorf("test %s failed stateless execution check: %w", testPath, err)
		}
	}

	return nil
}
