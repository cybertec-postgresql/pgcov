package report

import (
	"fmt"
	"html"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/pashagolub/pgcov/internal/coverage"
)

// HTMLReporter formats coverage data as HTML
type HTMLReporter struct{}

// NewHTMLReporter creates a new HTML reporter
func NewHTMLReporter() *HTMLReporter {
	return &HTMLReporter{}
}

// Format formats coverage data as HTML and writes to the writer
func (r *HTMLReporter) Format(cov *coverage.Coverage, writer io.Writer) error {
	// Sort files for deterministic output
	var files []string
	for file := range cov.Files {
		files = append(files, file)
	}
	sort.Strings(files)

	// Write HTML header
	if err := r.writeHeader(cov, writer); err != nil {
		return err
	}

	// Write summary section
	if err := r.writeSummary(cov, files, writer); err != nil {
		return err
	}

	// Write file details
	for _, file := range files {
		if err := r.writeFileDetail(file, cov, writer); err != nil {
			return err
		}
	}

	// Write HTML footer
	if err := r.writeFooter(writer); err != nil {
		return err
	}

	return nil
}

// writeHeader writes the HTML document header with CSS
func (r *HTMLReporter) writeHeader(cov *coverage.Coverage, writer io.Writer) error {
	timestamp := time.Now().Format(time.RFC1123)
	if !cov.Timestamp.IsZero() {
		timestamp = cov.Timestamp.Format(time.RFC1123)
	}

	_, err := fmt.Fprintf(writer, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>pgcov Coverage Report</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; background: #f5f5f5; color: #333; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        header { background: #2c3e50; color: white; padding: 30px 0; margin-bottom: 30px; }
        header h1 { font-size: 2.5em; margin-bottom: 10px; }
        header .meta { opacity: 0.8; font-size: 0.9em; }
        .summary { background: white; border-radius: 8px; padding: 25px; margin-bottom: 30px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .summary h2 { margin-bottom: 20px; color: #2c3e50; }
        .summary-stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; }
        .stat-card { background: #f8f9fa; padding: 20px; border-radius: 6px; border-left: 4px solid #3498db; }
        .stat-card .label { font-size: 0.85em; color: #7f8c8d; text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 8px; }
        .stat-card .value { font-size: 2em; font-weight: bold; color: #2c3e50; }
        .coverage-bar { width: 100%%; height: 24px; background: #ecf0f1; border-radius: 4px; overflow: hidden; margin-top: 10px; }
        .coverage-fill { height: 100%%; background: linear-gradient(90deg, #e74c3c 0%%, #f39c12 50%%, #2ecc71 100%%); transition: width 0.3s ease; }
        .file-list { background: white; border-radius: 8px; padding: 25px; margin-bottom: 30px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .file-list h2 { margin-bottom: 20px; color: #2c3e50; }
        .file-item { padding: 15px; border-bottom: 1px solid #ecf0f1; display: flex; justify-content: space-between; align-items: center; }
        .file-item:last-child { border-bottom: none; }
        .file-item:hover { background: #f8f9fa; }
        .file-name { font-family: 'Courier New', monospace; font-size: 0.95em; }
        .file-coverage { font-weight: bold; padding: 4px 12px; border-radius: 4px; }
        .file-coverage.high { background: #d4edda; color: #155724; }
        .file-coverage.medium { background: #fff3cd; color: #856404; }
        .file-coverage.low { background: #f8d7da; color: #721c24; }
        .file-detail { background: white; border-radius: 8px; padding: 25px; margin-bottom: 30px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .file-detail h3 { margin-bottom: 15px; color: #2c3e50; font-family: 'Courier New', monospace; }
        .source-code { background: #282c34; color: #abb2bf; font-family: 'Courier New', monospace; font-size: 0.9em; line-height: 1.6; border-radius: 6px; overflow-x: auto; }
        .source-line { display: flex; padding: 2px 0; }
        .source-line:hover { background: rgba(255,255,255,0.05); }
        .line-number { padding: 0 15px; text-align: right; user-select: none; color: #5c6370; min-width: 60px; }
        .line-hits { padding: 0 10px; text-align: right; user-select: none; min-width: 50px; font-size: 0.85em; }
        .line-content { padding: 0 15px; flex: 1; white-space: pre; }
        .covered { background: rgba(46, 204, 113, 0.15); }
        .covered .line-hits { color: #2ecc71; font-weight: bold; }
        .uncovered { background: rgba(231, 76, 60, 0.15); }
        .uncovered .line-hits { color: #e74c3c; font-weight: bold; }
        .not-instrumented { opacity: 0.6; }
        footer { text-align: center; padding: 30px 0; color: #7f8c8d; font-size: 0.9em; }
        .keyword { color: #c678dd; }
        .string { color: #98c379; }
        .comment { color: #5c6370; font-style: italic; }
    </style>
</head>
<body>
    <header>
        <div class="container">
            <h1>📊 pgcov Coverage Report</h1>
            <div class="meta">Generated: %s | Version: %s</div>
        </div>
    </header>
    <div class="container">
`, timestamp, html.EscapeString(cov.Version))
	return err
}

// writeSummary writes the coverage summary section
func (r *HTMLReporter) writeSummary(cov *coverage.Coverage, files []string, writer io.Writer) error {
	totalLines := 0
	coveredLines := 0

	for _, file := range files {
		hits := cov.Files[file]
		totalLines += len(hits)
		for _, count := range hits {
			if count > 0 {
				coveredLines++
			}
		}
	}

	totalPercent := cov.TotalLineCoveragePercent()

	_, err := fmt.Fprintf(writer, `        <section class="summary">
            <h2>Overall Coverage</h2>
            <div class="summary-stats">
                <div class="stat-card">
                    <div class="label">Total Coverage</div>
                    <div class="value">%.2f%%</div>
                    <div class="coverage-bar">
                        <div class="coverage-fill" style="width: %.2f%%;"></div>
                    </div>
                </div>
                <div class="stat-card">
                    <div class="label">Lines Covered</div>
                    <div class="value">%d / %d</div>
                </div>
                <div class="stat-card">
                    <div class="label">Files</div>
                    <div class="value">%d</div>
                </div>
            </div>
        </section>

`, totalPercent, totalPercent, coveredLines, totalLines, len(files))
	return err
}

// writeFileDetail writes detailed coverage for a single file
func (r *HTMLReporter) writeFileDetail(file string, cov *coverage.Coverage, writer io.Writer) error {
	hits := cov.Files[file]
	percent := cov.LineCoveragePercent(file)

	coverageClass := "high"
	if percent < 80 {
		coverageClass = "medium"
	}
	if percent < 50 {
		coverageClass = "low"
	}

	// Write file header
	_, err := fmt.Fprintf(writer, `        <section class="file-detail">
            <h3>%s <span class="file-coverage %s">%.2f%%</span></h3>
            <div class="source-code">
`, html.EscapeString(file), coverageClass, percent)
	if err != nil {
		return err
	}

	// Get sorted line numbers
	var lines []int
	for line := range hits {
		lines = append(lines, line)
	}
	sort.Ints(lines)

	// Write source lines
	for _, lineNum := range lines {
		hitCount := hits[lineNum]
		lineClass := ""
		hitsDisplay := ""

		if hitCount > 0 {
			lineClass = "covered"
			hitsDisplay = fmt.Sprintf("%d×", hitCount)
		} else {
			lineClass = "uncovered"
			hitsDisplay = "0×"
		}

		// For now, we don't have the actual source code, so we show line numbers only
		_, err := fmt.Fprintf(writer, `                <div class="source-line %s">
                    <div class="line-number">%d</div>
                    <div class="line-hits">%s</div>
                    <div class="line-content">&nbsp;</div>
                </div>
`, lineClass, lineNum, hitsDisplay)
		if err != nil {
			return err
		}
	}

	// Close source-code div and file-detail section
	_, err = writer.Write([]byte(`            </div>
        </section>

`))
	return err
}

// writeFooter writes the HTML document footer
func (r *HTMLReporter) writeFooter(writer io.Writer) error {
	_, err := fmt.Fprintf(writer, `        <footer>
            Generated by <strong>pgcov</strong> - PostgreSQL Test Coverage Tool
        </footer>
    </div>
</body>
</html>
`)
	return err
}

// FormatString returns coverage data as an HTML string
func (r *HTMLReporter) FormatString(cov *coverage.Coverage) (string, error) {
	var buf strings.Builder
	if err := r.Format(cov, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Name returns the name of this reporter
func (r *HTMLReporter) Name() string {
	return "html"
}
