package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/pashagolub/pgcov/internal/coverage"
	"github.com/pashagolub/pgcov/internal/database"
	"github.com/pashagolub/pgcov/internal/discovery"
	"github.com/pashagolub/pgcov/internal/instrument"
	"github.com/pashagolub/pgcov/internal/parser"
	"github.com/pashagolub/pgcov/internal/runner"
)

// Run executes the test runner workflow
func Run(ctx context.Context, config *Config, searchPath string) (int, error) {
	startTime := time.Now()

	if config.Verbose {
		fmt.Printf("pgcov: discovering tests in %s\n", searchPath)
	}

	// Step 1: Discover test files
	testFiles, err := discovery.DiscoverTests(searchPath)
	if err != nil {
		return 1, fmt.Errorf("failed to discover tests: %w", err)
	}

	if len(testFiles) == 0 {
		fmt.Println("No test files found (*_test.sql)")
		return 0, nil
	}

	if config.Verbose {
		fmt.Printf("Found %d test file(s)\n", len(testFiles))
	}

	// Step 2: Discover source files (co-located with tests)
	sourceFiles, err := discovery.DiscoverCoLocatedSources(testFiles)
	if err != nil {
		return 1, fmt.Errorf("failed to discover source files: %w", err)
	}

	if config.Verbose {
		fmt.Printf("Found %d source file(s)\n", len(sourceFiles))
	}

	// Step 3: Parse source files
	var parsedSources []*parser.ParsedSQL
	for i := range sourceFiles {
		parsed, err := parser.Parse(&sourceFiles[i])
		if err != nil {
			return 1, fmt.Errorf("failed to parse %s: %w", sourceFiles[i].RelativePath, err)
		}
		parsedSources = append(parsedSources, parsed)
	}

	// Step 4: Instrument source files
	instrumentedSources, err := instrument.InstrumentBatch(parsedSources)
	if err != nil {
		return 1, fmt.Errorf("failed to instrument sources: %w", err)
	}

	// Step 5: Connect to PostgreSQL
	pool, err := database.NewPool(ctx, config)
	if err != nil {
		return 1, fmt.Errorf("database connection failed: %w", err)
	}
	defer pool.Close()

	if config.Verbose {
		fmt.Printf("Connected to PostgreSQL at %s:%d\n", config.PGHost, config.PGPort)
	}

	// Step 6: Execute tests (parallel or sequential based on config)
	executor := runner.NewExecutor(pool, config.Timeout, config.Verbose)

	var testRuns []*runner.TestRun
	if config.Parallelism > 1 {
		// Use parallel execution
		if config.Verbose {
			fmt.Printf("Executing tests in parallel (workers: %d)\n", config.Parallelism)
		}
		workerPool := runner.NewWorkerPool(executor, config.Parallelism, config.Verbose)
		testRuns, err = workerPool.ExecuteParallel(ctx, testFiles, instrumentedSources)
	} else {
		// Use sequential execution
		if config.Verbose {
			fmt.Println("Executing tests sequentially")
		}
		testRuns, err = executor.ExecuteBatch(ctx, testFiles, instrumentedSources)
	}

	if err != nil {
		return 1, fmt.Errorf("test execution failed: %w", err)
	}

	// Step 7: Collect coverage
	collector := coverage.NewCollector()
	if err := collector.CollectFromRuns(testRuns); err != nil {
		return 1, fmt.Errorf("coverage collection failed: %w", err)
	}

	// Step 8: Save coverage data
	store := coverage.NewStore(config.CoverageFile)
	if err := store.Save(collector.Coverage()); err != nil {
		return 1, fmt.Errorf("failed to save coverage: %w", err)
	}

	// Step 9: Display summary
	summary := runner.SummarizeRuns(testRuns)
	coveragePercent := collector.TotalCoveragePercent()

	fmt.Printf("\n")
	fmt.Printf("Tests:    %d passed, %d failed, %d total\n",
		summary.PassedTests, summary.FailedTests, summary.TotalTests)
	fmt.Printf("Coverage: %.2f%%\n", coveragePercent)
	fmt.Printf("Time:     %v\n", time.Since(startTime).Round(time.Millisecond))
	fmt.Printf("\n")
	fmt.Printf("Coverage data written to %s\n", config.CoverageFile)

	// Return appropriate exit code
	return summary.ExitCode(), nil
}

// PrintVerbose prints a message if verbose mode is enabled
func PrintVerbose(config *Config, format string, args ...interface{}) {
	if config.Verbose {
		fmt.Printf(format+"\n", args...)
	}
}
