package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cybertec-postgresql/pgcov/internal/database"
	"github.com/cybertec-postgresql/pgcov/internal/discovery"
	"github.com/cybertec-postgresql/pgcov/internal/instrument"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Executor orchestrates test execution with coverage tracking
type Executor struct {
	pool    *database.Pool
	timeout time.Duration
	verbose bool
}

// NewExecutor creates a new test executor
func NewExecutor(pool *database.Pool, timeout time.Duration, verbose bool) *Executor {
	return &Executor{
		pool:    pool,
		timeout: timeout,
		verbose: verbose,
	}
}

// Execute runs a single test file and collects coverage
func (e *Executor) Execute(ctx context.Context, testFile *discovery.DiscoveredFile, sourceFiles []*instrument.InstrumentedSQL) (*TestRun, error) {
	testRun := &TestRun{
		Test:      testFile,
		StartTime: time.Now(),
		Status:    TestPending,
	}

	// Create context with timeout
	testCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Execute the per-test workflow
	err := e.executeTestWorkflow(testCtx, testRun, sourceFiles)
	if err != nil {
		testRun.Status = TestFailed
		testRun.Error = err
		if e.verbose {
			fmt.Printf("[ERROR] Test failed: %v\n", err)
		}
	} else {
		testRun.Status = TestPassed
	}

	testRun.EndTime = time.Now()

	return testRun, nil
}

// ExecuteBatch runs multiple tests sequentially
func (e *Executor) ExecuteBatch(ctx context.Context, testFiles []discovery.DiscoveredFile, sourceFiles []*instrument.InstrumentedSQL) ([]*TestRun, error) {
	var runs []*TestRun

	for i := range testFiles {
		if e.verbose {
			fmt.Printf("Running test: %s\n", testFiles[i].RelativePath)
		}

		// Filter source files to only include those from the same directory as the test
		testDir := filepath.Dir(testFiles[i].Path)
		filteredSources := filterSourcesByDirectory(sourceFiles, testDir)

		run, err := e.Execute(ctx, &testFiles[i], filteredSources)
		if err != nil {
			// Continue with other tests even if one fails
			if e.verbose {
				fmt.Printf("Test failed: %s: %v\n", testFiles[i].RelativePath, err)
			}
		}

		runs = append(runs, run)

		// Check if context was cancelled
		if ctx.Err() != nil {
			break
		}
	}

	return runs, nil
}

// filterSourcesByDirectory returns only source files from the specified directory
func filterSourcesByDirectory(sources []*instrument.InstrumentedSQL, testDir string) []*instrument.InstrumentedSQL {
	var filtered []*instrument.InstrumentedSQL
	for _, src := range sources {
		sourceDir := filepath.Dir(src.Original.File.Path)
		if sourceDir == testDir {
			filtered = append(filtered, src)
		}
	}
	return filtered
}

// SummarizeRuns creates a summary of test execution results
func SummarizeRuns(runs []*TestRun) *TestSummary {
	summary := &TestSummary{
		TotalTests: len(runs),
	}

	var totalDuration time.Duration

	for _, run := range runs {
		totalDuration += run.Duration()

		switch run.Status {
		case TestPassed:
			summary.PassedTests++
		case TestFailed:
			summary.FailedTests++
		case TestTimeout:
			summary.TimedOutTests++
		}
	}

	summary.TotalDuration = totalDuration

	return summary
}

// executeTestWorkflow implements the per-test workflow:
// 1. Create temp database
// 2. Load instrumented source code
// 3. Start LISTEN for coverage signals
// 4. Run test
// 5. Collect coverage signals
// 6. Destroy temp database
func (e *Executor) executeTestWorkflow(ctx context.Context, testRun *TestRun, sourceFiles []*instrument.InstrumentedSQL) error {
	if e.verbose {
		fmt.Println("[DEBUG] Step 1: Creating temp database...")
	}
	// Step 1: Create temporary database
	tempDB, err := database.CreateTempDatabase(ctx, e.pool)
	if err != nil {
		return fmt.Errorf("failed to create temp database: %w", err)
	}
	testRun.Database = tempDB
	if e.verbose {
		fmt.Printf("[DEBUG] Created temp database: %s\n", tempDB.Name)
	}

	// Ensure cleanup
	defer func() {
		if e.verbose {
			fmt.Println("[DEBUG] Cleaning up temp database...")
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = database.DestroyTempDatabase(cleanupCtx, e.pool, tempDB)
	}()

	if e.verbose {
		fmt.Println("[DEBUG] Step 2: Connecting to temp database...")
	}
	// Step 2: Connect to temp database
	tempPool, err := pgxpool.New(ctx, tempDB.ConnectionString)
	if err != nil {
		return fmt.Errorf("failed to connect to temp database: %w", err)
	}
	defer tempPool.Close()
	if e.verbose {
		fmt.Println("[DEBUG] Connected to temp database")
	}

	if e.verbose {
		fmt.Println("[DEBUG] Step 3: Starting LISTEN for coverage signals...")
	}
	// Step 3: Start LISTEN for coverage signals
	listener, err := database.NewListener(ctx, tempDB.ConnectionString, "pgcov")
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	defer listener.Close(ctx)
	if e.verbose {
		fmt.Println("[DEBUG] Listener started")
	}

	if e.verbose {
		fmt.Println("[DEBUG] Step 4: Loading instrumented source code...")
	}
	// Step 4: Load instrumented source code
	conn, err := tempPool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}

	for _, source := range sourceFiles {
		if e.verbose {
			fmt.Printf("[DEBUG] Loading source: %s\n", source.Original.File.RelativePath)
		}
		_, err := conn.Exec(ctx, source.InstrumentedText)
		if err != nil {
			conn.Release()
			if e.verbose {
				fmt.Printf("[DEBUG] Failed to load source: %v\n", err)
				fmt.Printf("[DEBUG] Instrumented SQL was:\n%s\n", source.InstrumentedText)
			}
			return fmt.Errorf("failed to load source %s: %w", source.Original.File.RelativePath, err)
		}

		// For successfully loaded source files, mark DDL/DML locations as implicitly covered
		// (PL/pgSQL code coverage is tracked via NOTIFY signals during execution)
		for _, loc := range source.Locations {
			if loc.ImplicitCoverage {
				testRun.CoverageSigs = append(testRun.CoverageSigs, CoverageSignal{
					SignalID:  loc.SignalID,
					Timestamp: time.Now(),
				})
			}
		}
	}
	conn.Release()
	if e.verbose {
		fmt.Println("[DEBUG] All sources loaded")
		fmt.Printf("[DEBUG] Added %d implicit coverage signals from DDL/DML\n", len(testRun.CoverageSigs))
	}

	if e.verbose {
		fmt.Println("[DEBUG] Step 5: Reading test file...")
	}
	// Step 5: Run test file
	testContent, err := os.ReadFile(testRun.Test.Path)
	if err != nil {
		return fmt.Errorf("failed to read test file: %w", err)
	}
	if e.verbose {
		fmt.Printf("[DEBUG] Test file read: %d bytes\n", len(testContent))
	}

	testRun.Status = TestRunning

	if e.verbose {
		fmt.Println("[DEBUG] Step 6: Executing test SQL...")
	}
	conn, err = tempPool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection for test: %w", err)
	}
	defer conn.Release()

	// Execute test SQL
	_, err = conn.Exec(ctx, string(testContent))
	if err != nil {
		return fmt.Errorf("test execution failed: %w", err)
	}
	if e.verbose {
		fmt.Println("[DEBUG] Test SQL executed successfully")
	}

	if e.verbose {
		fmt.Println("[DEBUG] Step 7: Collecting coverage signals...")
	}
	// Step 6: Collect coverage signals
	// Give a short time for any remaining signals to arrive
	signals, err := listener.CollectSignals(ctx, 100*time.Millisecond)
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		return fmt.Errorf("failed to collect signals: %w", err)
	}
	if e.verbose {
		fmt.Printf("[DEBUG] Collected %d signals\n", len(signals))
	}

	// Append NOTIFY signals to the implicit coverage signals
	testRun.CoverageSigs = append(testRun.CoverageSigs, signals...)

	return nil
}
