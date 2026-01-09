package cli

import (
	"fmt"
	"os"

	"github.com/cybertec-postgresql/pgcov/internal/coverage"
	"github.com/cybertec-postgresql/pgcov/internal/report"
)

// Report generates a coverage report from saved coverage data
func Report(coverageFile string, format string, outputPath string) error {
	// Step 1: Load coverage data
	store := coverage.NewStore(coverageFile)
	if !store.Exists() {
		return fmt.Errorf("coverage file not found: %s (run 'pgcov run' first)", coverageFile)
	}

	cov, err := store.Load()
	if err != nil {
		return fmt.Errorf("failed to load coverage data: %w", err)
	}

	// Step 2: Validate format
	if !report.ValidFormat(format) {
		return fmt.Errorf("unsupported format: %s (supported: %v)", format, report.SupportedFormats())
	}

	// Step 3: Get formatter
	formatter, err := report.GetFormatter(report.FormatType(format))
	if err != nil {
		return err
	}

	// Step 4: Format and output
	var writer *os.File
	if outputPath == "-" || outputPath == "" {
		// Write to stdout
		writer = os.Stdout
	} else {
		// Write to file
		writer, err = os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer writer.Close()
	}

	// Format coverage data
	if err := formatter.Format(cov, writer); err != nil {
		return fmt.Errorf("failed to format coverage data: %w", err)
	}

	// Print success message to stderr (so it doesn't interfere with stdout output)
	if outputPath != "-" && outputPath != "" {
		fmt.Fprintf(os.Stderr, "Report written to %s\n", outputPath)
	}

	return nil
}

// ReportSummary prints a human-readable summary of coverage
func ReportSummary(coverageFile string) error {
	store := coverage.NewStore(coverageFile)
	if !store.Exists() {
		return fmt.Errorf("coverage file not found: %s", coverageFile)
	}

	cov, err := store.Load()
	if err != nil {
		return fmt.Errorf("failed to load coverage data: %w", err)
	}

	// Print overall coverage
	fmt.Printf("Overall Coverage: %.2f%%\n\n", cov.TotalLineCoveragePercent())

	// Print per-file coverage
	fmt.Println("File Coverage:")
	for file, hits := range cov.Files {
		covered := 0
		for _, count := range hits {
			if count > 0 {
				covered++
			}
		}
		total := len(hits)
		percent := cov.LineCoveragePercent(file)

		fmt.Printf("  %s: %.2f%% (%d/%d lines)\n", file, percent, covered, total)
	}

	return nil
}
