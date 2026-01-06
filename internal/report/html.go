package report

import (
	"bufio"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pashagolub/pgcov/internal/coverage"
)

// HTMLReporter formats coverage data as HTML
type HTMLReporter struct{}

// NewHTMLReporter creates a new HTML reporter
func  NewHTMLReporter() *HTMLReporter {
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
	if err := r.writeHeader(cov, files, writer); err != nil {
		return err
	}

	// Write file details with source code
	for _, file := range files {
		if err := r.writeFileDetailWithSource(file, cov, writer); err != nil {
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
func (r *HTMLReporter) writeHeader(cov *coverage.Coverage, files []string, writer io.Writer) error {
	timestamp := time.Now().Format(time.RFC1123)
	if !cov.Timestamp.IsZero() {
		timestamp = cov.Timestamp.Format(time.RFC1123)
	}

	totalPercent := cov.TotalLineCoveragePercent()

	_, err := fmt.Fprintf(writer, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Coverage Report</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; background: #f5f5f5; color: #333; margin: 0; }
        .topbar { background: #000; color: white; padding: 10px 20px; display: flex; justify-content: space-between; align-items: center; }
        .topbar select { background: #333; color: white; border: 1px solid #555; padding: 5px 10px; border-radius: 3px; }
        .summary-bar { background: #eee; padding: 10px 20px; border-bottom: 1px solid #ccc; }
        .summary-stats { display: inline-block; margin-right: 20px; }
        .summary-stats .label { font-weight: bold; }
        .file-selector { background: white; border-bottom: 1px solid #ccc; padding: 0; }
        .file-selector a { display: block; padding: 10px 20px; text-decoration: none; color: #00f; border-bottom: 1px solid #eee; }
        .file-selector a:hover { background: #f5f5f5; }
        .file-content { background: white; }
        .source-line { display: block; font-family: 'Courier New', Consolas, monospace; font-size: 13px; line-height: 1.5; white-space: pre; padding: 0; border: none; }
        .source-line:hover { background: #f0f0f0; }
        .line-num { display: inline-block; width: 60px; text-align: right; padding-right: 10px; color: #666; user-select: none; background: #f5f5f5; border-right: 1px solid #ddd; }
        .line-count { display: inline-block; width: 80px; text-align: right; padding: 0 10px; user-select: none; font-weight: bold; }
        .line-code { display: inline-block; padding-left: 10px; }
        .cov0 { background: #ffcccc; }
        .cov0 .line-count { color: #c00; }
        .cov1, .cov2, .cov3, .cov4, .cov5, .cov6, .cov7, .cov8 { background: #c0ffc0; }
        .cov1 .line-count, .cov2 .line-count, .cov3 .line-count, .cov4 .line-count,
        .cov5 .line-count, .cov6 .line-count, .cov7 .line-count, .cov8 .line-count { color: #080; }
        .not-tracked { background: #f5f5f5; }
        .not-tracked .line-count { color: #999; }
        /* SQL Syntax highlighting */
        .sql-keyword { color: #0000ff; font-weight: bold; }
        .sql-string { color: #a31515; }
        .sql-comment { color: #008000; font-style: italic; }
        .sql-number { color: #098658; }
        .sql-operator { color: #000; }
        .sql-function { color: #795e26; }
    </style>
</head>
<body>
    <div class="topbar">
        <span>pgcov</span>
        <select id="fileSelector" onchange="location.href='#'+this.value">
            <option value="">-- Select file --</option>
`)
	if err != nil {
		return err
	}

	// Write file options
	for _, file := range files {
		_, err = fmt.Fprintf(writer, `            <option value="%s">%s</option>
`, html.EscapeString(file), html.EscapeString(file))
		if err != nil {
			return err
		}
	}

	// Write summary bar
	_, err = fmt.Fprintf(writer, `        </select>
    </div>
    <div class="summary-bar">
        <span class="summary-stats"><span class="label">Total Coverage:</span> %.2f%%%%</span>
        <span class="summary-stats"><span class="label">Generated:</span> %s</span>
    </div>
    <div class="file-selector">
`, totalPercent, html.EscapeString(timestamp))
	if err != nil {
		return err
	}

	// Write file links
	for _, file := range files {
		percent := cov.LineCoveragePercent(file)
		_, err = fmt.Fprintf(writer, `        <a href="#%s">%s (%.2f%%%%)</a>
`, html.EscapeString(file), html.EscapeString(file), percent)
		if err != nil {
			return err
		}
	}

	_, err = writer.Write([]byte(`    </div>
`))
	return err
}

// writeFileDetailWithSource writes detailed coverage for a single file with actual source code
func (r *HTMLReporter) writeFileDetailWithSource(file string, cov *coverage.Coverage, writer io.Writer) error {
	hits := cov.Files[file]
	percent := cov.LineCoveragePercent(file)

	// Write file header
	_, err := fmt.Fprintf(writer, `    <div class="file-content" id="%s">
        <h2 style="padding: 20px; background: #f0f0f0; border-bottom: 2px solid #ccc; font-family: 'Courier New', monospace;">%s (%.2f%%%%)</h2>
`, html.EscapeString(file), html.EscapeString(file), percent)
	if err != nil {
		return err
	}

	// Read the source file
	sourceLines, err := r.readSourceFile(file)
	if err != nil {
		// If we can't read the file, show line numbers only
		_, err = fmt.Fprintf(writer, `        <div style="padding: 20px; color: #c00;">Error reading source file: %s</div>
`, html.EscapeString(err.Error()))
		if err != nil {
			return err
		}
		
		// Still show hit counts for lines we have
		var lines []int
		for line := range hits {
			lines = append(lines, line)
		}
		sort.Ints(lines)

		for _, lineNum := range lines {
			hitCount := hits[lineNum]
			covClass := r.getCoverageClass(hitCount)
			countDisplay := r.getCountDisplay(hitCount)

			_, err = fmt.Fprintf(writer, `        <div class="source-line %s">
            <span class="line-num">%d</span>
            <span class="line-count">%s</span>
            <span class="line-code">(source not available)</span>
        </div>
`, covClass, lineNum, countDisplay)
			if err != nil {
				return err
			}
		}
	} else {
		// Show actual source code with coverage
		for lineNum, lineContent := range sourceLines {
			hitCount, hasCoverage := hits[lineNum]
			covClass := "not-tracked"
			countDisplay := ""

			if hasCoverage {
				covClass = r.getCoverageClass(hitCount)
				countDisplay = r.getCountDisplay(hitCount)
			}

			// Apply SQL syntax highlighting
			highlightedCode := r.highlightSQL(lineContent)

			_, err = fmt.Fprintf(writer, `        <div class="source-line %s">
            <span class="line-num">%d</span>
            <span class="line-count">%s</span>
            <span class="line-code">%s</span>
        </div>
`, covClass, lineNum, countDisplay, highlightedCode)
			if err != nil {
				return err
			}
		}
	}

	// Close file-content div
	_, err = writer.Write([]byte(`    </div>
`))
	return err
}

// readSourceFile reads a source file and returns a map of line numbers to content
func (r *HTMLReporter) readSourceFile(filePath string) (map[int]string, error) {
	// Try to open the file - handle both absolute and relative paths
	file, err := os.Open(filePath)
	if err != nil {
		// Try with current working directory
		cwd, _ := os.Getwd()
		altPath := filepath.Join(cwd, filePath)
		file, err = os.Open(altPath)
		if err != nil {
			return nil, fmt.Errorf("cannot open file: %w", err)
		}
	}
	defer file.Close()

	lines := make(map[int]string)
	scanner := bufio.NewScanner(file)
	lineNum := 1

	for scanner.Scan() {
		lines[lineNum] = scanner.Text()
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return lines, nil
}

// getCoverageClass returns the CSS class for coverage styling
func (r *HTMLReporter) getCoverageClass(hitCount int) string {
	if hitCount == 0 {
		return "cov0"
	}
	// Use cov1-cov8 for different hit counts (like go tool cover)
	if hitCount > 8 {
		return "cov8"
	}
	return fmt.Sprintf("cov%d", hitCount)
}

// getCountDisplay returns the display string for hit count
func (r *HTMLReporter) getCountDisplay(hitCount int) string {
	if hitCount == 0 {
		return "0"
	}
	if hitCount == 1 {
		return "1"
	}
	return fmt.Sprintf("%d", hitCount)
}

// highlightSQL applies basic SQL syntax highlighting
func (r *HTMLReporter) highlightSQL(line string) string {
	if line == "" {
		return ""
	}

	// Escape HTML first
	line = html.EscapeString(line)

	// SQL keywords (case-insensitive)
	keywords := []string{
		"SELECT", "FROM", "WHERE", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER",
		"TABLE", "INDEX", "VIEW", "FUNCTION", "PROCEDURE", "TRIGGER", "BEGIN", "END",
		"IF", "THEN", "ELSE", "ELSIF", "LOOP", "WHILE", "FOR", "RETURN", "RETURNS",
		"AS", "IS", "IN", "NOT", "NULL", "AND", "OR", "ON", "JOIN", "LEFT", "RIGHT",
		"INNER", "OUTER", "CROSS", "USING", "GROUP", "BY", "ORDER", "HAVING", "LIMIT",
		"OFFSET", "UNION", "INTERSECT", "EXCEPT", "CASE", "WHEN", "EXISTS", "ANY", "ALL",
		"DECLARE", "SET", "INTO", "VALUES", "DEFAULT", "PRIMARY", "KEY", "FOREIGN",
		"REFERENCES", "UNIQUE", "CHECK", "CONSTRAINT", "CASCADE", "SERIAL", "BOOLEAN",
		"INTEGER", "BIGINT", "TEXT", "VARCHAR", "CHAR", "DATE", "TIME", "TIMESTAMP",
		"NUMERIC", "DECIMAL", "REAL", "DOUBLE", "PRECISION", "ARRAY", "JSON", "JSONB",
		"GRANT", "REVOKE", "TO", "WITH", "RECURSIVE", "DISTINCT", "ASC", "DESC",
	}

	// Common SQL functions
	functions := []string{
		"COUNT", "SUM", "AVG", "MIN", "MAX", "CONCAT", "SUBSTRING", "LENGTH",
		"UPPER", "LOWER", "TRIM", "COALESCE", "NULLIF", "NOW", "CURRENT_TIMESTAMP",
		"CURRENT_DATE", "EXTRACT", "DATE_PART", "AGE", "ARRAY_AGG", "STRING_AGG",
	}

	// Highlight keywords
	for _, kw := range keywords {
		// Match whole words only (case-insensitive)
		re := regexp.MustCompile(`(?i)\b` + kw + `\b`)
		line = re.ReplaceAllStringFunc(line, func(match string) string {
			return `<span class="sql-keyword">` + match + `</span>`
		})
	}

	// Highlight functions
	for _, fn := range functions {
		re := regexp.MustCompile(`(?i)\b` + fn + `\s*\(`)
		line = re.ReplaceAllStringFunc(line, func(match string) string {
			return `<span class="sql-function">` + match[:len(match)-1] + `</span>(`
		})
	}

	// Highlight strings (single quotes)
	stringRe := regexp.MustCompile(`'[^']*'`)
	line = stringRe.ReplaceAllStringFunc(line, func(match string) string {
		return `<span class="sql-string">` + match + `</span>`
	})

	// Highlight comments (-- style)
	commentRe := regexp.MustCompile(`--.*$`)
	line = commentRe.ReplaceAllStringFunc(line, func(match string) string {
		return `<span class="sql-comment">` + match + `</span>`
	})

	// Highlight numbers
	numberRe := regexp.MustCompile(`\b\d+(\.\d+)?\b`)
	line = numberRe.ReplaceAllStringFunc(line, func(match string) string {
		// Skip if already inside a span (e.g., from previous highlighting)
		return `<span class="sql-number">` + match + `</span>`
	})

	return line
}

// writeFooter writes the HTML document footer
func (r *HTMLReporter) writeFooter(writer io.Writer) error {
	_, err := writer.Write([]byte(`</body>
</html>
`))
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
