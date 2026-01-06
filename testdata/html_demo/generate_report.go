package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pashagolub/pgcov/internal/coverage"
	"github.com/pashagolub/pgcov/internal/report"
)

func main() {
	// Create sample coverage data
	cov := &coverage.Coverage{
		Version:   "1.0",
		Timestamp: time.Now(),
		Files: map[string]coverage.FileHits{
			"testdata/html_demo/sample.sql": {
				2:  1, // CREATE TABLE
				3:  1, // id SERIAL
				4:  1, // username VARCHAR
				5:  1, // email VARCHAR
				6:  1, // created_at TIMESTAMP
				7:  1, // closing paren
				10: 5, // INSERT INTO (executed 5 times)
				11: 5, // values line 1
				12: 5, // values line 2
				15: 3, // SELECT * (executed 3 times)
				18: 2, // UPDATE users (executed 2 times)
				21: 1, // DO block
				22: 1, // BEGIN
				23: 1, // IF EXISTS
				24: 0, // RAISE NOTICE (not executed)
				25: 0, // ELSE branch (not executed)
				26: 1, // INSERT admin (executed once)
				27: 1, // END IF
				28: 1, // END block
			},
		},
	}

	// Generate HTML report
	reporter := report.NewHTMLReporter()
	file, err := os.Create("testdata/html_demo/report.html")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating report file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	err = reporter.Format(cov, file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ HTML report generated: testdata/html_demo/report.html")
	fmt.Printf("  Total coverage: %.2f%%\n", cov.TotalLineCoveragePercent())
}
