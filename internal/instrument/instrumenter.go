package instrument

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cybertec-postgresql/pgcov/internal/parser"
	pgquery "github.com/pganalyze/pg_query_go/v6"
)

// GenerateCoverageInstruments instruments multiple parsed SQL files
func GenerateCoverageInstruments(parsedFiles []*parser.ParsedSQL) ([]*InstrumentedSQL, error) {
	var instrumented []*InstrumentedSQL

	for _, parsed := range parsedFiles {
		inst, err := GenerateCoverageInstrument(parsed)
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
func GenerateCoverageInstrument(parsed *parser.ParsedSQL) (*InstrumentedSQL, error) {
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

	// For functions, determine the language from AST
	if stmt.Type == parser.StmtFunction {
		funcLang := getFunctionLanguage(stmt)

		switch funcLang {
		case "plpgsql":
			instrumented, locs := instrumentPlpgsqlFunction(stmt, filePath)
			return instrumented, locs
		case "sql":
			instrumented, locs := instrumentSQLFunction(stmt, filePath)
			return instrumented, locs
		default:
			// Unknown language, mark as implicitly covered
			locations = markStatementLinesAsCovered(stmt, filePath)
			return stmt.RawSQL, locations
		}
	}

	// For DO blocks, check if they're PL/pgSQL
	if isDOBlock(stmt) {
		instrumented, locs := instrumentPlpgsqlFunction(stmt, filePath)
		return instrumented, locs
	}

	// For non-function statements (DDL, DML), mark all non-comment lines as covered
	// These will be automatically marked as covered if the file executes without errors
	locations = markStatementLinesAsCovered(stmt, filePath)

	// Return original SQL without instrumentation - DDL/DML are implicitly covered on success
	return stmt.RawSQL, locations
}

// getFunctionLanguage extracts the language from a CREATE FUNCTION statement using the AST node
func getFunctionLanguage(stmt *parser.Statement) string {
	if stmt.Node == nil {
		return ""
	}

	if createFunc, ok := stmt.Node.Node.(*pgquery.Node_CreateFunctionStmt); ok {
		if createFunc.CreateFunctionStmt != nil {
			// Look for the LANGUAGE option in the function definition
			for _, opt := range createFunc.CreateFunctionStmt.Options {
				if opt.GetDefElem() != nil && opt.GetDefElem().Defname == "language" {
					if langNode := opt.GetDefElem().Arg; langNode != nil {
						if strNode := langNode.GetString_(); strNode != nil {
							return strings.ToLower(strNode.Sval)
						}
					}
				}
			}
		}
	}

	return ""
}

// isDOBlock checks if the statement is a DO block using the AST node
func isDOBlock(stmt *parser.Statement) bool {
	if stmt.Node == nil {
		return false
	}

	if _, ok := stmt.Node.Node.(*pgquery.Node_DoStmt); ok {
		return true
	}

	return false
}

// instrumentPlpgsqlFunction instruments a PL/pgSQL function with line-by-line coverage
// Uses pg_query.ParsePlPgSqlToJSON to properly parse the PL/pgSQL AST
func instrumentPlpgsqlFunction(stmt *parser.Statement, filePath string) (string, []CoveragePoint) {
	// Trim leading empty lines from stmt.RawSQL to ensure PL/pgSQL parser line numbers match
	lines := strings.Split(stmt.RawSQL, "\n")
	firstNonEmptyIndex := 0
	for i, line := range lines {
		if len(strings.TrimSpace(line)) > 0 {
			firstNonEmptyIndex = i
			break
		}
	}

	trimmedSQL := strings.Join(lines[firstNonEmptyIndex:], "\n")
	lineOffset := firstNonEmptyIndex

	// Parse the PL/pgSQL function using the proper parser
	jsonResult, err := pgquery.ParsePlPgSqlToJSON(trimmedSQL)
	if err != nil {
		// Return empty instrumentation for malformed SQL
		return stmt.RawSQL, nil
	}

	// Parse the JSON result to extract statement line numbers
	var plpgsqlAST []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonResult), &plpgsqlAST); err != nil {
		// Return empty instrumentation for invalid AST
		return stmt.RawSQL, nil
	}

	// Extract executable line numbers from the AST
	executableLines := extractExecutableLines(plpgsqlAST)

	// If no executable lines found, return without instrumentation
	if len(executableLines) == 0 {
		return stmt.RawSQL, nil
	}

	// Find the function body offset using the AST structure (using trimmed SQL)
	trimmedStmt := &parser.Statement{
		RawSQL:    trimmedSQL,
		StartLine: stmt.StartLine + lineOffset,
		EndLine:   stmt.EndLine,
		Type:      stmt.Type,
		Node:      stmt.Node,
	}
	bodyOffset := findFunctionBodyOffset(trimmedStmt)
	if bodyOffset < 0 {
		// Could not determine body offset, return without instrumentation
		return stmt.RawSQL, nil
	}

	// Now instrument the function by injecting PERFORM pg_notify at each executable line
	instrumentedTrimmed, locations := injectNotifyAtLines(trimmedStmt, filePath, executableLines)

	// Reconstruct full SQL with leading lines
	if lineOffset > 0 {
		result := strings.Join(lines[:lineOffset], "\n") + "\n" + instrumentedTrimmed
		return result, locations
	}

	return instrumentedTrimmed, locations
}

// extractExecutableLines walks the PL/pgSQL AST and extracts line numbers of executable statements
func extractExecutableLines(ast []map[string]interface{}) []int {
	var lines []int

	for _, node := range ast {
		lines = append(lines, walkPlpgsqlNode(node)...)
	}

	return lines
}

// walkPlpgsqlNode recursively walks a PL/pgSQL AST node and extracts executable line numbers
func walkPlpgsqlNode(node map[string]interface{}) []int {
	var lines []int

	for key, value := range node {
		switch key {
		case "PLpgSQL_function":
			// Walk the function action (body)
			if funcMap, ok := value.(map[string]interface{}); ok {
				if action, ok := funcMap["action"].(map[string]interface{}); ok {
					lines = append(lines, walkPlpgsqlNode(action)...)
				}
			}

		case "PLpgSQL_stmt_block":
			// Walk block body
			if blockMap, ok := value.(map[string]interface{}); ok {
				if body, ok := blockMap["body"].([]interface{}); ok {
					for _, stmt := range body {
						if stmtMap, ok := stmt.(map[string]interface{}); ok {
							lines = append(lines, walkPlpgsqlNode(stmtMap)...)
						}
					}
				}
			}

		case "PLpgSQL_stmt_assign":
			// Assignment statement - executable
			if stmtMap, ok := value.(map[string]interface{}); ok {
				if lineno, ok := stmtMap["lineno"].(float64); ok {
					lines = append(lines, int(lineno))
				}
			}

		case "PLpgSQL_stmt_return":
			// Return statement - executable
			if stmtMap, ok := value.(map[string]interface{}); ok {
				if lineno, ok := stmtMap["lineno"].(float64); ok {
					lines = append(lines, int(lineno))
				}
			}

		case "PLpgSQL_stmt_if":
			// IF statement - walk branches
			if stmtMap, ok := value.(map[string]interface{}); ok {
				// Walk then_body
				if thenBody, ok := stmtMap["then_body"].([]interface{}); ok {
					for _, stmt := range thenBody {
						if s, ok := stmt.(map[string]interface{}); ok {
							lines = append(lines, walkPlpgsqlNode(s)...)
						}
					}
				}
				// Walk elsif_list
				if elsifList, ok := stmtMap["elsif_list"].([]interface{}); ok {
					for _, elsif := range elsifList {
						if elsifMap, ok := elsif.(map[string]interface{}); ok {
							if elsifStmt, ok := elsifMap["PLpgSQL_if_elsif"].(map[string]interface{}); ok {
								if stmts, ok := elsifStmt["stmts"].([]interface{}); ok {
									for _, stmt := range stmts {
										if s, ok := stmt.(map[string]interface{}); ok {
											lines = append(lines, walkPlpgsqlNode(s)...)
										}
									}
								}
							}
						}
					}
				}
				// Walk else_body
				if elseBody, ok := stmtMap["else_body"].([]interface{}); ok {
					for _, stmt := range elseBody {
						if s, ok := stmt.(map[string]interface{}); ok {
							lines = append(lines, walkPlpgsqlNode(s)...)
						}
					}
				}
			}

		case "PLpgSQL_stmt_loop":
			// Loop statement
			if stmtMap, ok := value.(map[string]interface{}); ok {
				if body, ok := stmtMap["body"].([]interface{}); ok {
					for _, stmt := range body {
						if s, ok := stmt.(map[string]interface{}); ok {
							lines = append(lines, walkPlpgsqlNode(s)...)
						}
					}
				}
			}

		case "PLpgSQL_stmt_while":
			// While loop
			if stmtMap, ok := value.(map[string]interface{}); ok {
				if body, ok := stmtMap["body"].([]interface{}); ok {
					for _, stmt := range body {
						if s, ok := stmt.(map[string]interface{}); ok {
							lines = append(lines, walkPlpgsqlNode(s)...)
						}
					}
				}
			}

		case "PLpgSQL_stmt_fori", "PLpgSQL_stmt_fors", "PLpgSQL_stmt_forc", "PLpgSQL_stmt_dynfors":
			// FOR loops
			if stmtMap, ok := value.(map[string]interface{}); ok {
				if body, ok := stmtMap["body"].([]interface{}); ok {
					for _, stmt := range body {
						if s, ok := stmt.(map[string]interface{}); ok {
							lines = append(lines, walkPlpgsqlNode(s)...)
						}
					}
				}
			}

		case "PLpgSQL_stmt_exit":
			// EXIT statement
			if stmtMap, ok := value.(map[string]interface{}); ok {
				if lineno, ok := stmtMap["lineno"].(float64); ok {
					lines = append(lines, int(lineno))
				}
			}

		case "PLpgSQL_stmt_raise":
			// RAISE statement
			if stmtMap, ok := value.(map[string]interface{}); ok {
				if lineno, ok := stmtMap["lineno"].(float64); ok {
					lines = append(lines, int(lineno))
				}
			}

		case "PLpgSQL_stmt_execsql":
			// Execute SQL statement
			if stmtMap, ok := value.(map[string]interface{}); ok {
				if lineno, ok := stmtMap["lineno"].(float64); ok {
					lines = append(lines, int(lineno))
				}
			}

		case "PLpgSQL_stmt_perform":
			// PERFORM statement
			if stmtMap, ok := value.(map[string]interface{}); ok {
				if lineno, ok := stmtMap["lineno"].(float64); ok {
					lines = append(lines, int(lineno))
				}
			}
		}
	}

	return lines
}

// findFunctionBodyOffset determines the line offset where the function body starts
// by using AST location information from CreateFunctionStmt
// Returns the 0-based line index within stmt.RawSQL where the function body begins
func findFunctionBodyOffset(stmt *parser.Statement) int {
	if stmt.Node == nil {
		return -1
	}

	// Extract location from CreateFunctionStmt
	if createFunc, ok := stmt.Node.Node.(*pgquery.Node_CreateFunctionStmt); ok {
		if createFunc.CreateFunctionStmt != nil {
			// Find the "as" option which contains the function body location
			for _, opt := range createFunc.CreateFunctionStmt.Options {
				if defElem := opt.GetDefElem(); defElem != nil && defElem.Defname == "as" {
					// The defElem.Location points to the start of "AS" keyword
					// The body starts after AS and the delimiter ($$, $function$, etc.)
					if defElem.Location > 0 {
						// Convert byte offset to line number (relative to statement start)
						lineOffset := calculateLineFromByteOffset(stmt.RawSQL, int(defElem.Location))
						// The body typically starts on the next line after AS $$
						// But we need to be more precise - the body starts after the newline following the delimiter
						return lineOffset + 1
					}
				}
			}
		}
	}

	return -1
}

// calculateLineFromByteOffset converts a byte offset to a 0-based line index
func calculateLineFromByteOffset(sql string, byteOffset int) int {
	lineIdx := 0
	for i := 0; i < byteOffset && i < len(sql); i++ {
		if sql[i] == '\n' {
			lineIdx++
		}
	}
	return lineIdx
}

// injectNotifyAtLines injects PERFORM pg_notify calls at specified line numbers
func injectNotifyAtLines(stmt *parser.Statement, filePath string, executableLines []int) (string, []CoveragePoint) {
	var locations []CoveragePoint

	// Split into lines
	lines := strings.Split(stmt.RawSQL, "\n")
	result := strings.Builder{}

	// The line numbers from ParsePlPgSqlToJSON map directly to 0-based indices in stmt.RawSQL
	// (i.e., PL/pgSQL line N → index N in lines array)
	// To get file line numbers: PL/pgSQL line N → stmt.RawSQL index N → file line (stmt.StartLine + N)

	absoluteLines := make(map[int]bool)
	for _, plpgsqlLine := range executableLines {
		// plpgsqlLine maps to index plpgsqlLine in the lines array (0-based)
		lineIndex := plpgsqlLine
		if lineIndex >= 0 && lineIndex < len(lines) {
			// Convert to absolute file line number (1-based)
			absoluteLine := stmt.StartLine + lineIndex
			absoluteLines[absoluteLine] = true
		}
	}

	currentLine := stmt.StartLine
	for i, line := range lines {
		// Check if this line should be instrumented
		if absoluteLines[currentLine] {
			// Create coverage point
			cp := CoveragePoint{
				File:             filePath,
				Line:             currentLine,
				Branch:           "",
				ImplicitCoverage: false,
			}
			cp.SignalID = FormatSignalID(cp.File, cp.Line, cp.Branch)
			locations = append(locations, cp)

			// Inject NOTIFY before the line
			indent := getIndentation(line)
			notifyCall := fmt.Sprintf("%sPERFORM pg_notify('pgcov', '%s');\n",
				indent, strings.ReplaceAll(cp.SignalID, "'", "''"))
			result.WriteString(notifyCall)
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

// getIndentation returns the leading whitespace of a line
func getIndentation(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
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
// Uses AST node location to determine the statement boundaries rather than string operations
func markStatementLinesAsCovered(stmt *parser.Statement, filePath string) []CoveragePoint {
	var locations []CoveragePoint

	// For DDL/DML statements, mark the primary line(s) as implicitly covered
	// We use the statement's line range from the AST
	for line := stmt.StartLine; line <= stmt.EndLine; line++ {
		cp := CoveragePoint{
			File:             filePath,
			Line:             line,
			Branch:           "",
			ImplicitCoverage: true, // DDL/DML are implicitly covered on successful execution
		}
		cp.SignalID = FormatSignalID(cp.File, cp.Line, cp.Branch)
		locations = append(locations, cp)
	}

	return locations
}
