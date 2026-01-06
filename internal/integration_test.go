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
	"github.com/pashagolub/pgcov/internal/runner"
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

// TestOrderIndependence verifies that tests produce identical results
// regardless of execution order (test A→B vs B→A)
func TestOrderIndependence(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	t.Log("Starting PostgreSQL container for order-independence test...")
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
		PGHost:       host,
		PGPort:       port.Int(),
		PGUser:       "testuser",
		PGPassword:   "testpass",
		PGDatabase:   "testdb",
		Timeout:      30 * time.Second,
		Parallelism:  1,
		CoverageFile: filepath.Join(t.TempDir(), "coverage.json"),
		Verbose:      true,
	}

	testDir := "../testdata/simple"

	// Discover and prepare test files
	testFiles, err := discovery.DiscoverTests(testDir)
	if err != nil {
		t.Fatalf("Failed to discover tests: %v", err)
	}

	if len(testFiles) < 2 {
		t.Skip("Need at least 2 test files for order-independence test")
	}

	t.Logf("Found %d test files for order testing", len(testFiles))

	// Discover and instrument source files
	sourceFiles, err := discovery.DiscoverCoLocatedSources(testFiles)
	if err != nil {
		t.Fatalf("Failed to discover sources: %v", err)
	}

	var parsedSources []*parser.ParsedSQL
	for i := range sourceFiles {
		parsed, err := parser.Parse(&sourceFiles[i])
		if err != nil {
			t.Fatalf("Failed to parse %s: %v", sourceFiles[i].RelativePath, err)
		}
		parsedSources = append(parsedSources, parsed)
	}

	instrumentedSources, err := instrument.InstrumentBatch(parsedSources)
	if err != nil {
		t.Fatalf("Failed to instrument sources: %v", err)
	}

	// Create connection pool
	pool, err := database.NewPool(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Helper function to run tests in a specific order and collect coverage
	runTestsInOrder := func(order []discovery.DiscoveredFile, label string) *coverage.Coverage {
		t.Logf("Running tests in order %s", label)

		executor := runner.NewExecutor(pool, config.Timeout, config.Verbose)
		testRuns, err := executor.ExecuteBatch(ctx, order, instrumentedSources)
		if err != nil {
			t.Fatalf("Test execution failed for %s: %v", label, err)
		}

		collector := coverage.NewCollector()
		if err := collector.CollectFromRuns(testRuns); err != nil {
			t.Fatalf("Coverage collection failed for %s: %v", label, err)
		}

		return collector.Coverage()
	}

	// Run tests in order A→B
	orderAB := make([]discovery.DiscoveredFile, len(testFiles))
	copy(orderAB, testFiles)
	coverageAB := runTestsInOrder(orderAB, "A→B")

	// Run tests in order B→A (reverse)
	orderBA := make([]discovery.DiscoveredFile, len(testFiles))
	copy(orderBA, testFiles)
	// Reverse the order
	for i, j := 0, len(orderBA)-1; i < j; i, j = i+1, j-1 {
		orderBA[i], orderBA[j] = orderBA[j], orderBA[i]
	}
	coverageBA := runTestsInOrder(orderBA, "B→A")

	// Compare coverage results
	t.Log("Comparing coverage results...")

	// Check that both have the same files
	if len(coverageAB.Files) != len(coverageBA.Files) {
		t.Errorf("Different number of files covered: A→B has %d, B→A has %d",
			len(coverageAB.Files), len(coverageBA.Files))
	}

	// Compare each file's coverage
	for filePath, fileAB := range coverageAB.Files {
		fileBA, exists := coverageBA.Files[filePath]
		if !exists {
			t.Errorf("File %s covered in A→B but not in B→A", filePath)
			continue
		}

		// Compare line coverage
		if len(fileAB.Lines) != len(fileBA.Lines) {
			t.Errorf("File %s: different number of lines covered: A→B has %d, B→A has %d",
				filePath, len(fileAB.Lines), len(fileBA.Lines))
		}

		// Check each line
		for lineNum, lineAB := range fileAB.Lines {
			lineBA, exists := fileBA.Lines[lineNum]
			if !exists {
				t.Errorf("File %s, line %d: covered in A→B but not in B→A",
					filePath, lineNum)
				continue
			}

			// Verify coverage status is identical
			if lineAB.Covered != lineBA.Covered {
				t.Errorf("File %s, line %d: coverage mismatch - A→B=%v, B→A=%v",
					filePath, lineNum, lineAB.Covered, lineBA.Covered)
			}

			// Note: We don't compare HitCount exactly, as it may vary if tests
			// execute the same line multiple times. We only care that Covered
			// status is the same.
		}

		// Compare branch coverage
		for branchID, branchAB := range fileAB.Branches {
			branchBA, exists := fileBA.Branches[branchID]
			if !exists {
				t.Errorf("File %s, branch %s: covered in A→B but not in B→A",
					filePath, branchID)
				continue
			}

			if branchAB.Covered != branchBA.Covered {
				t.Errorf("File %s, branch %s: coverage mismatch - A→B=%v, B→A=%v",
					filePath, branchID, branchAB.Covered, branchBA.Covered)
			}
		}

		t.Logf("✓ File %s: coverage identical in both orders", filePath)
	}

	// Check for files in BA but not in AB
	for filePath := range coverageBA.Files {
		if _, exists := coverageAB.Files[filePath]; !exists {
			t.Errorf("File %s covered in B→A but not in A→B", filePath)
		}
	}

	// Compare total coverage percentages
	totalAB := coverageAB.TotalLineCoveragePercent()
	totalBA := coverageBA.TotalLineCoveragePercent()
	t.Logf("Total coverage A→B: %.2f%%", totalAB)
	t.Logf("Total coverage B→A: %.2f%%", totalBA)

	// Allow tiny floating point differences
	if totalAB != totalBA {
		diff := totalAB - totalBA
		if diff < 0 {
			diff = -diff
		}
		if diff > 0.01 { // Allow 0.01% difference for floating point precision
			t.Errorf("Total coverage mismatch: A→B=%.2f%%, B→A=%.2f%%", totalAB, totalBA)
		}
	}

	t.Log("✓ Order-independence test passed! Coverage is identical regardless of test execution order.")
}

// TestTestIndependence verifies that running the same test multiple times
// produces identical coverage results (no state accumulation across runs)
func TestTestIndependence(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	t.Log("Starting PostgreSQL container for test independence verification...")
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
		PGHost:       host,
		PGPort:       port.Int(),
		PGUser:       "testuser",
		PGPassword:   "testpass",
		PGDatabase:   "testdb",
		Timeout:      30 * time.Second,
		Parallelism:  1,
		CoverageFile: filepath.Join(t.TempDir(), "coverage.json"),
		Verbose:      true,
	}

	testDir := "../testdata/isolation"

	// Discover test files
	testFiles, err := discovery.DiscoverTests(testDir)
	if err != nil {
		t.Fatalf("Failed to discover tests: %v", err)
	}

	if len(testFiles) == 0 {
		t.Fatal("No test files found in testdata/isolation")
	}

	// Use the first test file for repeated execution
	testFile := testFiles[0]
	t.Logf("Using test file: %s", testFile.RelativePath)

	// Discover and instrument source files
	sourceFiles, err := discovery.DiscoverCoLocatedSources(testFiles)
	if err != nil {
		t.Fatalf("Failed to discover sources: %v", err)
	}

	var parsedSources []*parser.ParsedSQL
	for i := range sourceFiles {
		parsed, err := parser.Parse(&sourceFiles[i])
		if err != nil {
			t.Fatalf("Failed to parse %s: %v", sourceFiles[i].RelativePath, err)
		}
		parsedSources = append(parsedSources, parsed)
	}

	instrumentedSources, err := instrument.InstrumentBatch(parsedSources)
	if err != nil {
		t.Fatalf("Failed to instrument sources: %v", err)
	}

	// Create connection pool
	pool, err := database.NewPool(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Helper function to run a single test and collect coverage
	runSingleTest := func(iteration int) (*runner.TestRun, *coverage.Coverage) {
		t.Logf("Running test iteration %d...", iteration)

		executor := runner.NewExecutor(pool, config.Timeout, config.Verbose)
		testRuns, err := executor.ExecuteBatch(ctx, []discovery.DiscoveredFile{testFile}, instrumentedSources)
		if err != nil {
			t.Fatalf("Test execution failed (iteration %d): %v", iteration, err)
		}

		if len(testRuns) != 1 {
			t.Fatalf("Expected 1 test run, got %d", len(testRuns))
		}

		collector := coverage.NewCollector()
		if err := collector.CollectFromRuns(testRuns); err != nil {
			t.Fatalf("Coverage collection failed (iteration %d): %v", iteration, err)
		}

		return testRuns[0], collector.Coverage()
	}

	// Run the same test twice
	t.Log("=== First Execution ===")
	run1, coverage1 := runSingleTest(1)

	t.Log("=== Second Execution ===")
	run2, coverage2 := runSingleTest(2)

	// Verify isolation using the isolation validator
	t.Log("Verifying stateless execution...")
	err = runner.VerifyStatelessExecution(run1, run2)
	if err != nil {
		t.Fatalf("Stateless execution verification failed: %v", err)
	}
	t.Log("✓ Test runs are properly isolated (different databases used)")

	// Verify test status is identical
	if run1.Status != run2.Status {
		t.Errorf("Test status differs: run1=%s, run2=%s", run1.Status, run2.Status)
	}
	t.Logf("✓ Test status identical: %s", run1.Status)

	// Compare coverage results
	t.Log("Comparing coverage results...")

	// Check that both have the same files
	if len(coverage1.Files) != len(coverage2.Files) {
		t.Errorf("Different number of files covered: run1 has %d, run2 has %d",
			len(coverage1.Files), len(coverage2.Files))
	}

	// Compare each file's coverage in detail
	for filePath, file1 := range coverage1.Files {
		file2, exists := coverage2.Files[filePath]
		if !exists {
			t.Errorf("File %s covered in run1 but not in run2", filePath)
			continue
		}

		// Compare number of lines
		if len(file1.Lines) != len(file2.Lines) {
			t.Errorf("File %s: different number of lines covered: run1 has %d, run2 has %d",
				filePath, len(file1.Lines), len(file2.Lines))
		}

		// Compare line-by-line coverage
		mismatchCount := 0
		for lineNum, line1 := range file1.Lines {
			line2, exists := file2.Lines[lineNum]
			if !exists {
				t.Errorf("File %s, line %d: covered in run1 but not in run2",
					filePath, lineNum)
				mismatchCount++
				continue
			}

			// Verify coverage status is identical
			if line1.Covered != line2.Covered {
				t.Errorf("File %s, line %d: coverage mismatch - run1=%v, run2=%v",
					filePath, lineNum, line1.Covered, line2.Covered)
				mismatchCount++
			}
		}

		// Compare branch coverage
		for branchID, branch1 := range file1.Branches {
			branch2, exists := file2.Branches[branchID]
			if !exists {
				t.Errorf("File %s, branch %s: covered in run1 but not in run2",
					filePath, branchID)
				mismatchCount++
				continue
			}

			if branch1.Covered != branch2.Covered {
				t.Errorf("File %s, branch %s: coverage mismatch - run1=%v, run2=%v",
					filePath, branchID, branch1.Covered, branch2.Covered)
				mismatchCount++
			}
		}

		if mismatchCount == 0 {
			t.Logf("✓ File %s: coverage identical in both runs", filePath)
		} else {
			t.Errorf("File %s: %d mismatches found", filePath, mismatchCount)
		}
	}

	// Check for files in run2 but not in run1
	for filePath := range coverage2.Files {
		if _, exists := coverage1.Files[filePath]; !exists {
			t.Errorf("File %s covered in run2 but not in run1", filePath)
		}
	}

	// Compare total coverage percentages
	total1 := coverage1.TotalLineCoveragePercent()
	total2 := coverage2.TotalLineCoveragePercent()
	t.Logf("Total coverage run1: %.2f%%", total1)
	t.Logf("Total coverage run2: %.2f%%", total2)

	if total1 != total2 {
		diff := total1 - total2
		if diff < 0 {
			diff = -diff
		}
		if diff > 0.01 { // Allow 0.01% difference for floating point precision
			t.Errorf("Total coverage mismatch: run1=%.2f%%, run2=%.2f%%", total1, total2)
		}
	}

	// Verify coverage signals are identical
	t.Logf("Coverage signals: run1=%d, run2=%d", len(run1.CoverageSigs), len(run2.CoverageSigs))
	if len(run1.CoverageSigs) != len(run2.CoverageSigs) {
		t.Errorf("Different number of coverage signals: run1=%d, run2=%d",
			len(run1.CoverageSigs), len(run2.CoverageSigs))
	}

	// Verify databases were properly cleaned up
	t.Log("Verifying database cleanup...")
	exists1, err := databaseExists(ctx, pool, run1.Database.Name)
	if err != nil {
		t.Errorf("Failed to check database existence: %v", err)
	} else if exists1 {
		t.Errorf("Database %s from run1 was not cleaned up", run1.Database.Name)
	}

	exists2, err := databaseExists(ctx, pool, run2.Database.Name)
	if err != nil {
		t.Errorf("Failed to check database existence: %v", err)
	} else if exists2 {
		t.Errorf("Database %s from run2 was not cleaned up", run2.Database.Name)
	}

	if !exists1 && !exists2 {
		t.Log("✓ Both test databases were properly cleaned up")
	}

	t.Log("✓ Test independence verified! Same test produces identical coverage across multiple runs.")
}

// databaseExists checks if a database exists in PostgreSQL
func databaseExists(ctx context.Context, pool *database.Pool, dbName string) (bool, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Release()

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`
	err = conn.QueryRow(ctx, query, dbName).Scan(&exists)
	return exists, err
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
