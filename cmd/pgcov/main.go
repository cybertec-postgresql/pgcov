package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cybertec-postgresql/pgcov/internal/cli"
	urfavecli "github.com/urfave/cli/v3"
)

const version = "1.0.0"

func main() {
	app := &urfavecli.Command{
		Name:    "pgcov",
		Usage:   "PostgreSQL test runner and coverage tool",
		Version: version,
		Commands: []*urfavecli.Command{
			{
				Name:   "run",
				Usage:  "Run tests and collect coverage",
				Action: runCommand,
				Flags: []urfavecli.Flag{
					&urfavecli.StringFlag{
						Name:    "connection",
						Aliases: []string{"c"},
						Usage:   "PostgreSQL connection string (URI or key=value format). Supports standard PG* environment variables.",
					},
					&urfavecli.DurationFlag{
						Name:  "timeout",
						Usage: "Per-test timeout",
					},
					&urfavecli.IntFlag{
						Name:  "parallel",
						Usage: "Maximum concurrent tests (1 = sequential)",
					},
					&urfavecli.StringFlag{
						Name:  "coverage-file",
						Usage: "Coverage data output path",
					},
					&urfavecli.BoolFlag{
						Name:  "verbose",
						Usage: "Enable debug output",
					},
				},
			},
			{
				Name:   "report",
				Usage:  "Generate coverage report",
				Action: reportCommand,
				Flags: []urfavecli.Flag{
					&urfavecli.StringFlag{
						Name:  "format",
						Usage: "Output format (json, lcov, or html)",
						Value: "json",
					},
					&urfavecli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output file path (use - for stdout)",
						Value:   "-",
					},
					&urfavecli.StringFlag{
						Name:  "coverage-file",
						Usage: "Coverage data input path",
						Value: ".pgcov/coverage.json",
					},
				},
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runCommand handles the 'pgcov run' command
func runCommand(ctx context.Context, cmd *urfavecli.Command) error {
	// Load configuration
	config := &cli.DefaultConfig

	// Apply flags
	connection := cmd.String("connection")
	timeout := cmd.Duration("timeout")
	parallel := cmd.Int("parallel")
	coverageFile := cmd.String("coverage-file")
	verbose := cmd.Bool("verbose")

	cli.ApplyFlagsToConfig(config, connection, timeout, parallel, coverageFile, verbose)

	// Validate configuration
	if err := config.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	// Get search path (first non-flag argument, default to current directory)
	searchPath := cmd.Args().First()
	if searchPath == "" {
		searchPath = "."
	}

	// Run tests
	exitCode, err := cli.Run(ctx, config, searchPath)
	if err != nil {
		return err
	}

	// Exit with appropriate code
	if exitCode != 0 {
		os.Exit(exitCode)
	}

	return nil
}

// reportCommand handles the 'pgcov report' command
func reportCommand(ctx context.Context, cmd *urfavecli.Command) error {
	format := cmd.String("format")
	output := cmd.String("output")
	coverageFile := cmd.String("coverage-file")

	return cli.Report(ctx, coverageFile, format, output)
}
