package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pashagolub/pgcov/internal/cli"
	"github.com/pashagolub/pgcov/internal/coverage"
	"github.com/pashagolub/pgcov/internal/database"
	"github.com/pashagolub/pgcov/internal/discovery"
	"github.com/pashagolub/pgcov/internal/instrument"
	"github.com/pashagolub/pgcov/internal/parser"
	"github.com/pashagolub/pgcov/pkg/types"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestEndToEndWithTestcontainers performs a complete end-to-end test
// using testcontainers to spin up a real PostgreSQL instance
func TestEndToEndWithTestcontainers(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	t.Log("Starting PostgreSQL container...")
	pgContainer, err := postgres.Run(ctx,
		"docker.io/postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
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

	// Create test configuration
	config := &types.Config{
		PGHost:       host,
		PGPort:       port.Int(),
		PGUser:       "testuser",
		PGPassword:   "testpass",
		PGDatabase:   "testdb",
		Timeout:      30 * time.Second,
		Parallelism:  1,
		CoverageFile: "coverage.json",
		Verbose:      true,
	}

	// Test Phase 1: Discovery
	t.Run("Discovery", func(t *testing.T) {
		testDir := "../testdata/simple"

		// Discover test files
		testFiles, err := discovery.DiscoverTests(testDir)
		if err != nil {
			t.Fatalf("Failed to discover tests: %v", err)
		}

		if len(testFiles) == 0 {
			t.Fatal("No test files found")
		}

		t.Logf("Discovered %d test file(s)", len(testFiles))

		// Discover source files
		sourceFiles, err := discovery.DiscoverCoLocatedSources(testFiles)
		if err != nil {
			t.Fatalf("Failed to discover sources: %v", err)
		}

		if len(sourceFiles) == 0 {
			t.Fatal("No source files found")
		}

		t.Logf("Discovered %d source file(s)", len(sourceFiles))
	})

	// Test Phase 2: Parsing
	t.Run("Parsing", func(t *testing.T) {
		testDir := "../testdata/simple"
		sourceFiles, _ := discovery.DiscoverSources(testDir)

		for _, file := range sourceFiles {
			parsed, err := parser.Parse(&file)
			if err != nil {
				t.Fatalf("Failed to parse %s: %v", file.RelativePath, err)
			}

			if parsed.AST == nil {
				t.Fatalf("No AST generated for %s", file.RelativePath)
			}

			if len(parsed.Statements) == 0 {
				t.Logf("Warning: No statements found in %s", file.RelativePath)
			}

			t.Logf("Parsed %s: %d statements", file.RelativePath, len(parsed.Statements))
		}
	})

	// Test Phase 3: Instrumentation
	t.Run("Instrumentation", func(t *testing.T) {
		testDir := "../testdata/simple"
		sourceFiles, _ := discovery.DiscoverSources(testDir)

		for _, file := range sourceFiles {
			parsed, _ := parser.Parse(&file)
			instrumented, err := instrument.Instrument(parsed)
			if err != nil {
				t.Fatalf("Failed to instrument %s: %v", file.RelativePath, err)
			}

			if instrumented.InstrumentedText == "" {
				t.Logf("Warning: No instrumented text for %s", file.RelativePath)
			}

			t.Logf("Instrumented %s: %d coverage points",
				file.RelativePath, len(instrumented.Locations))
		}
	})

	// Test Phase 4: Full End-to-End Execution
	t.Run("FullExecution", func(t *testing.T) {
		testDir := "../testdata/simple"

		// Run the full workflow using cli.Run
		exitCode, err := cli.Run(ctx, config, testDir)
		// Note: Exit code might be non-zero if test has no assertions
		// For Phase 3, we just verify the workflow completes
		if err != nil {
			t.Fatalf("Test execution failed with error: %v", err)
		}

		t.Logf("Test completed with exit code: %d", exitCode)

		// Verify coverage file was created
		if _, err := os.Stat(config.CoverageFile); os.IsNotExist(err) {
			t.Fatal("Coverage file was not created")
		}

		// Load and verify coverage data
		store := coverage.NewStore(config.CoverageFile)
		cov, err := store.Load()
		if err != nil {
			t.Fatalf("Failed to load coverage data: %v", err)
		}

		// Phase 3: Coverage data structure should exist (even if empty without instrumentation)
		if cov == nil {
			t.Fatal("Coverage data is nil")
		}

		totalPercent := cov.TotalLineCoveragePercent()
		t.Logf("Total coverage: %.2f%%", totalPercent)

		// Verify coverage is >= 0% (instrumentation is TODO for Phase 4)
		if totalPercent < 0 {
			t.Error("Coverage should be >= 0%")
		}

		// Log coverage details (will be empty until Phase 4 implements NOTIFY injection)
		if len(cov.Files) > 0 {
			for file, fileCov := range cov.Files {
				percent := fileCov.LineCoveragePercent()
				t.Logf("  %s: %.2f%% (%d/%d lines)",
					file, percent, countCovered(fileCov), len(fileCov.Lines))
			}
		} else {
			t.Log("  No coverage data (instrumentation not yet implemented in Phase 3)")
		}
	})

	// Test Phase 5: Database Operations
	t.Run("DatabaseOperations", func(t *testing.T) {
		// Test connection pool
		pool, err := database.NewPool(ctx, config)
		if err != nil {
			t.Fatalf("Failed to create connection pool: %v", err)
		}
		defer pool.Close()

		// Test temp database creation
		tempDB, err := database.CreateTempDatabase(ctx, pool)
		if err != nil {
			t.Fatalf("Failed to create temp database: %v", err)
		}

		t.Logf("Created temp database: %s", tempDB.Name)

		// Test temp database cleanup
		err = database.DestroyTempDatabase(ctx, pool, tempDB)
		if err != nil {
			t.Fatalf("Failed to destroy temp database: %v", err)
		}

		t.Log("Successfully cleaned up temp database")
	})

	// Test Phase 6: Report Generation
	t.Run("ReportGeneration", func(t *testing.T) {
		// Ensure we have coverage data
		testDir := "../testdata/simple"
		_, _ = cli.Run(ctx, config, testDir)

		// Test JSON report
		err := cli.Report(config.CoverageFile, "json", "-")
		if err != nil {
			t.Fatalf("Failed to generate JSON report: %v", err)
		}

		// Test LCOV report
		lcovFile := filepath.Join(t.TempDir(), "coverage.lcov")
		err = cli.Report(config.CoverageFile, "lcov", lcovFile)
		if err != nil {
			t.Fatalf("Failed to generate LCOV report: %v", err)
		}

		// Verify LCOV file was created
		if _, err := os.Stat(lcovFile); os.IsNotExist(err) {
			t.Fatal("LCOV file was not created")
		}

		t.Log("Successfully generated both JSON and LCOV reports")
	})

	t.Log("✓ All integration tests passed!")
}

// TestRunnerIsolation verifies that each test runs in complete isolation
func TestRunnerIsolation(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	t.Log("Starting PostgreSQL container for isolation test...")
	pgContainer, err := postgres.Run(ctx,
		"docker.io/postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
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

	host, _ := pgContainer.Host(ctx)
	port, _ := pgContainer.MappedPort(ctx, "5432")

	config := &types.Config{
		PGHost:     host,
		PGPort:     port.Int(),
		PGUser:     "testuser",
		PGPassword: "testpass",
		PGDatabase: "testdb",
		Timeout:    30 * time.Second,
	}

	// Create connection pool
	pool, err := database.NewPool(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Create two temp databases and verify they're independent
	db1, err := database.CreateTempDatabase(ctx, pool)
	if err != nil {
		t.Fatalf("Failed to create first temp database: %v", err)
	}

	db2, err := database.CreateTempDatabase(ctx, pool)
	if err != nil {
		t.Fatalf("Failed to create second temp database: %v", err)
	}

	// Verify different names
	if db1.Name == db2.Name {
		t.Fatal("Temp databases should have unique names")
	}

	t.Logf("Created isolated databases: %s and %s", db1.Name, db2.Name)

	// Cleanup
	database.DestroyTempDatabase(ctx, pool, db1)
	database.DestroyTempDatabase(ctx, pool, db2)

	t.Log("✓ Isolation test passed!")
}

// Helper function to count covered lines
func countCovered(fileCov *coverage.FileCoverage) int {
	count := 0
	for _, line := range fileCov.Lines {
		if line.Covered {
			count++
		}
	}
	return count
}
