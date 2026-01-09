package instrument

import (
	"fmt"
	"strings"

	"github.com/cybertec-postgresql/pgcov/internal/parser"
)

// Instrument takes parsed SQL and injects coverage tracking calls
func Instrument(parsed *parser.ParsedSQL) (*InstrumentedSQL, error) {
	// Use full instrumentation with NOTIFY injection
	return InstrumentWithNotify(parsed)
}

// InstrumentBatch instruments multiple parsed SQL files
func InstrumentBatch(parsedFiles []*parser.ParsedSQL) ([]*InstrumentedSQL, error) {
	var instrumented []*InstrumentedSQL

	for _, parsed := range parsedFiles {
		inst, err := Instrument(parsed)
		if err != nil {
			return nil, fmt.Errorf("failed to instrument %s: %w", parsed.File.Path, err)
		}
		instrumented = append(instrumented, inst)
	}

	return instrumented, nil
}

// GetCoveragePointBySignal finds a coverage point by its signal ID
func GetCoveragePointBySignal(instrumented *InstrumentedSQL, signalID string) *CoveragePoint {
	for _, cp := range instrumented.Locations {
		if cp.SignalID == signalID {
			return &cp
		}
	}
	return nil
}

// InstrumentWithNotify instruments SQL by injecting NOTIFY calls for coverage tracking
func InstrumentWithNotify(parsed *parser.ParsedSQL) (*InstrumentedSQL, error) {
	if parsed == nil || parsed.File == nil {
		return nil, fmt.Errorf("parsed SQL or file is nil")
	}

	var locations []CoveragePoint
	var instrumentedStatements []string

	// Process each statement
	for _, stmt := range parsed.Statements {
		relPath := parsed.File.RelativePath
		if relPath == "" {
			relPath = parsed.File.Path
		}

		// Instrument the statement and collect coverage points
		instrumentedSQL, stmtLocations := instrumentStatement(stmt, relPath)
		locations = append(locations, stmtLocations...)
		instrumentedStatements = append(instrumentedStatements, instrumentedSQL)
	}

	// Join all instrumented statements with proper separators
	instrumentedText := strings.Join(instrumentedStatements, "\n\n")

	return &InstrumentedSQL{
		Original:         parsed,
		InstrumentedText: instrumentedText,
		Locations:        locations,
	}, nil
}

// instrumentStatement instruments a single statement with line-by-line coverage
func instrumentStatement(stmt *parser.Statement, filePath string) (string, []CoveragePoint) {
	var locations []CoveragePoint

	// For PL/pgSQL functions, instrument each line
	if stmt.Type == parser.StmtFunction && strings.Contains(strings.ToUpper(stmt.RawSQL), "LANGUAGE PLPGSQL") {
		instrumented, locs := instrumentPlpgsqlFunction(stmt, filePath)
		return instrumented, locs
	}

	// For SQL functions (LANGUAGE SQL), instrument the statement
	if stmt.Type == parser.StmtFunction && strings.Contains(strings.ToUpper(stmt.RawSQL), "LANGUAGE SQL") {
		instrumented, locs := instrumentSQLFunction(stmt, filePath)
		return instrumented, locs
	}

	// For DO blocks with PL/pgSQL, instrument like a function
	if strings.HasPrefix(strings.TrimSpace(strings.ToUpper(stmt.RawSQL)), "DO") {
		instrumented, locs := instrumentPlpgsqlFunction(stmt, filePath)
		return instrumented, locs
	}

	// For non-function statements (DDL, DML), mark all non-comment lines as covered
	// These will be automatically marked as covered if the file executes without errors
	locations = markStatementLinesAsCovered(stmt, filePath)

	// Return original SQL without instrumentation - DDL/DML are implicitly covered on success
	return stmt.RawSQL, locations
}

// instrumentPlpgsqlFunction instruments a PL/pgSQL function with line-by-line coverage
func instrumentPlpgsqlFunction(stmt *parser.Statement, filePath string) (string, []CoveragePoint) {
	var locations []CoveragePoint

	// Split the function into lines
	lines := strings.Split(stmt.RawSQL, "\n")
	result := strings.Builder{}

	currentLine := stmt.StartLine
	inFunctionBody := false
	inMultiLineStatement := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		trimmedUpper := strings.ToUpper(trimmed)

		// Check if we're entering the function body
		if !inFunctionBody && trimmedUpper == "BEGIN" {
			inFunctionBody = true
			result.WriteString(line)
			result.WriteString("\n")
			currentLine++
			continue
		}

		// Check if we're exiting the function body (END followed by semicolon, not END IF/LOOP/CASE)
		if inFunctionBody &&
			(trimmedUpper == "END;" || trimmedUpper == "END$$;" || trimmedUpper == "END") &&
			!strings.HasPrefix(trimmedUpper, "END IF") &&
			!strings.HasPrefix(trimmedUpper, "END LOOP") &&
			!strings.HasPrefix(trimmedUpper, "END CASE") {
			result.WriteString(line)
			result.WriteString("\n")
			inFunctionBody = false
			currentLine++
			continue
		}

		// Inside function body: instrument executable lines
		if inFunctionBody && trimmed != "" && !strings.HasPrefix(trimmed, "--") {
			// Skip control flow keywords that aren't executable statements
			isControlFlow := isControlFlowKeyword(trimmedUpper)

			if !isControlFlow {
				// Only instrument the start of a new statement (not continuation lines)
				if !inMultiLineStatement {
					// Create coverage point for this line
					cp := CoveragePoint{
						File:             filePath,
						Line:             currentLine,
						Branch:           "",
						ImplicitCoverage: false, // Explicit coverage via NOTIFY
					}
					cp.SignalID = FormatSignalID(cp.File, cp.Line, cp.Branch)
					locations = append(locations, cp)

					// Inject NOTIFY before the line
					indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					notifyCall := fmt.Sprintf("%sPERFORM pg_notify('pgcov', '%s');\n",
						indent, strings.ReplaceAll(cp.SignalID, "'", "''"))
					result.WriteString(notifyCall)
				}
			}

			// Track if we're in a multi-line statement
			// Control flow keywords don't count as multi-line statement starters
			if !isControlFlow {
				if strings.HasSuffix(trimmed, ";") {
					inMultiLineStatement = false
				} else {
					inMultiLineStatement = true
				}
			}
		}

		// Write original line
		result.WriteString(line)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
		currentLine++
	}

	return result.String(), locations
}

// isControlFlowKeyword checks if a line is a control flow keyword that shouldn't be instrumented
func isControlFlowKeyword(upperTrimmed string) bool {
	// Skip lines that are ONLY control flow keywords (no executable code)
	// For IF/ELSIF with conditions, we still skip them
	// For ELSE without code, we skip it
	// But assignments like "discount_rate := 0.20;" should NOT be skipped

	// Keywords that are complete statements by themselves
	keywordsExact := []string{
		"ELSE", "ELSIF", "END IF", "END IF;", "LOOP", "END LOOP", "END LOOP;",
		"END CASE", "END CASE;", "DECLARE", "BEGIN",
	}
	for _, kw := range keywordsExact {
		if upperTrimmed == kw {
			return true
		}
	}

	// Keywords that start a line but may have conditions
	keywordsPrefix := []string{
		"IF ", "ELSIF ", "WHEN ", "FOR ", "WHILE ", "CASE ",
	}
	for _, kw := range keywordsPrefix {
		if strings.HasPrefix(upperTrimmed, kw) {
			return true
		}
	}

	// Also skip transaction control statements (these are complete statements)
	if upperTrimmed == "COMMIT;" || upperTrimmed == "ROLLBACK;" ||
		upperTrimmed == "COMMIT" || upperTrimmed == "ROLLBACK" {
		return true
	}

	return false
}

// instrumentSQLFunction instruments a SQL function
func instrumentSQLFunction(stmt *parser.Statement, filePath string) (string, []CoveragePoint) {
	// For SQL functions, we can't inject PERFORM, so we mark the function definition line
	cp := CoveragePoint{
		File:   filePath,
		Line:   stmt.StartLine,
		Branch: "",
	}
	cp.SignalID = FormatSignalID(cp.File, cp.Line, cp.Branch)

	// SQL functions are harder to instrument - for now, just track the function call
	// This would require wrapping the SQL expression which is complex
	return stmt.RawSQL, []CoveragePoint{cp}
}

// markStatementLinesAsCovered creates coverage points for all non-comment lines
func markStatementLinesAsCovered(stmt *parser.Statement, filePath string) []CoveragePoint {
	var locations []CoveragePoint

	lines := strings.Split(stmt.RawSQL, "\n")
	currentLine := stmt.StartLine

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip empty lines and comments
		if trimmed != "" && !strings.HasPrefix(trimmed, "--") {
			cp := CoveragePoint{
				File:             filePath,
				Line:             currentLine,
				Branch:           "",
				ImplicitCoverage: true, // DDL/DML are implicitly covered on successful execution
			}
			cp.SignalID = FormatSignalID(cp.File, cp.Line, cp.Branch)
			locations = append(locations, cp)
		}
		currentLine++
	}

	return locations
}

func readOriginalSQL(parsed *parser.ParsedSQL) string {
	if len(parsed.Statements) == 0 {
		return ""
	}
	var parts []string
	for _, stmt := range parsed.Statements {
		parts = append(parts, stmt.RawSQL)
	}
	return strings.Join(parts, "\n")
}

func findStatementPosition(sql string, lineNum int) int {
	if lineNum <= 1 {
		return 0
	}
	pos := 0
	currentLine := 1
	for i := 0; i < len(sql) && currentLine < lineNum; i++ {
		if sql[i] == '\n' {
			currentLine++
			pos = i + 1
		}
	}
	return pos
}
