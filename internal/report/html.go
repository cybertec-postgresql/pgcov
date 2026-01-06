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
	for i, file := range files {
		if err := r.writeFileDetailWithSource(file, cov, writer, i); err != nil {
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
	_, err := fmt.Fprintf(writer, `
<!DOCTYPE html>
<html>
	<head>
		<meta http-equiv="Content-Type" content="text/html; charset=utf-8">
		<title>pgcov: Coverage Report</title>
		<style>
			body {
				background: black;
				color: rgb(80, 80, 80);
			}
			body, pre, #legend span {
				font-family: Menlo, monospace;
				font-weight: bold;
			}
			#topbar {
				background: black;
				position: fixed;
				top: 0; left: 0; right: 0;
				height: 42px;
				border-bottom: 1px solid rgb(80, 80, 80);
			}
			#content {
				margin-top: 50px;
			}
			#nav, #legend {
				float: left;
				margin-left: 10px;
			}
			#legend {
				margin-top: 12px;
			}
			#nav {
				margin-top: 10px;
			}
			#legend span {
				margin: 0 5px;
			}
			.cov0 { color: rgb(192, 0, 0) }
.cov1 { color: rgb(128, 128, 128) }
.cov2 { color: rgb(116, 140, 131) }
.cov3 { color: rgb(104, 152, 134) }
.cov4 { color: rgb(92, 164, 137) }
.cov5 { color: rgb(80, 176, 140) }
.cov6 { color: rgb(68, 188, 143) }
.cov7 { color: rgb(56, 200, 146) }
.cov8 { color: rgb(44, 212, 149) }
.cov9 { color: rgb(32, 224, 152) }
.cov10 { color: rgb(20, 236, 155) }

		</style>
	</head>
	<body>
		<div id="topbar">
			<div id="nav">
				<select id="files">
`)
	if err != nil {
		return err
	}

	// Write file options with coverage percentages
	for i, file := range files {
		percent := cov.LineCoveragePercent(file)
		_, err = fmt.Fprintf(writer, `				<option value="file%d">%s (%.1f%%%%)</option>
`, i, html.EscapeString(file), percent)
		if err != nil {
			return err
		}
	}

	// Write legend
	_, err = writer.Write([]byte(`				</select>
			</div>
			<div id="legend">
				<span>not tracked</span>
			
				<span class="cov0">not covered</span>
				<span class="cov8">covered</span>
			
			</div>
		</div>
		<div id="content">
`))
	return err
}

// writeFileDetailWithSource writes detailed coverage for a single file with actual source code
func (r *HTMLReporter) writeFileDetailWithSource(file string, cov *coverage.Coverage, writer io.Writer, fileIndex int) error {
	hits := cov.Files[file]

	// Write file pre tag with ID and hidden by default
	displayStyle := "display: none"
	if fileIndex == 0 {
		displayStyle = "" // Show first file by default
	}

	_, err := fmt.Fprintf(writer, `		<pre class="file" id="file%d" style="%s">`, fileIndex, displayStyle)
	if err != nil {
		return err
	}

	// Read the source file
	sourceLines, err := r.readSourceFile(file)
	if err != nil {
		// If we can't read the file, show error
		_, err = fmt.Fprintf(writer, `package main

// Error reading source file: %s
`, html.EscapeString(err.Error()))
		if err != nil {
			return err
		}
	} else {
		// Show actual source code with coverage coloring
		for lineNum := 1; lineNum <= len(sourceLines); lineNum++ {
			lineContent := sourceLines[lineNum]
			hitCount, hasCoverage := hits[lineNum]
			
			if hasCoverage {
				// Escape HTML and apply coverage coloring
				escapedContent := html.EscapeString(lineContent)
				covClass := r.getCoverageClass(hitCount)
				
				// Write the span with newline at end
				_, err = fmt.Fprintf(writer, `<span class="%s" title="%d">%s</span>
`, covClass, hitCount, escapedContent)
				if err != nil {
					return err
				}
			} else {
				// No coverage for this line - just output as is with newline
				escapedContent := html.EscapeString(lineContent)
				_, err = fmt.Fprintf(writer, "%s\n", escapedContent)
				if err != nil {
					return err
				}
			}
		}
	}

	// Close pre tag
	_, err = writer.Write([]byte("</pre>\n\t\t\n\t\t"))
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
	// Use cov1-cov10 for different hit counts (like go tool cover)
	if hitCount > 10 {
		return "cov10"
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

// writeFooter writes the HTML document footer with JavaScript
func (r *HTMLReporter) writeFooter(writer io.Writer) error {
	_, err := writer.Write([]byte(`	</div>
	</body>
	<script>
	(function() {
		var files = document.getElementById('files');
		var visible;
		files.addEventListener('change', onChange, false);
		function select(part) {
			if (visible)
				visible.style.display = 'none';
			visible = document.getElementById(part);
			if (!visible)
				return;
			files.value = part;
			visible.style.display = 'block';
			location.hash = part;
		}
		function onChange() {
			select(files.value);
			window.scrollTo(0, 0);
		}
		if (location.hash != "") {
			select(location.hash.substr(1));
		}
		if (!visible) {
			select("file0");
		}
	})();
	</script>
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
