package instrument

import (
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
			instrumented, locs := instrumentWithLexer(stmt, filePath)
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
		instrumented, locs := instrumentWithLexer(stmt, filePath)
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

	if createFunc := stmt.Node.GetCreateFunctionStmt(); createFunc != nil {
		// Look for the LANGUAGE option in the function definition
		for _, opt := range createFunc.Options {
			if opt.GetDefElem() != nil && opt.GetDefElem().Defname == "language" {
				if langNode := opt.GetDefElem().Arg; langNode != nil {
					if strNode := langNode.GetString_(); strNode != nil {
						return strings.ToLower(strNode.Sval)
					}
				}
			}
		}
	}

	return ""
}

// isDOBlock checks if the statement is a DO block using the AST node
func isDOBlock(stmt *parser.Statement) bool {
	return stmt.Node != nil && stmt.Node.GetDoStmt() != nil
}

// extractFunctionBody extracts the function/DO-block body text from the AST node.
// Returns the body text or "" if not found.
func extractFunctionBody(stmt *parser.Statement) string {
	if stmt.Node == nil {
		return ""
	}

	switch node := stmt.Node.Node; node.(type) {
	case *pgquery.Node_CreateFunctionStmt:
		createFunc := stmt.Node.GetCreateFunctionStmt()
		for _, opt := range createFunc.Options {
			if defElem := opt.GetDefElem(); defElem != nil && defElem.Defname == "as" {
				if defElem.Arg != nil {
					if strList := defElem.Arg.GetList(); strList != nil && len(strList.Items) > 0 {
						if strNode := strList.Items[0].GetString_(); strNode != nil {
							return strNode.Sval
						}
					} else if strNode := defElem.Arg.GetString_(); strNode != nil {
						return strNode.Sval
					}
				}
			}
		}

	case *pgquery.Node_DoStmt:
		if doStmt := stmt.Node.GetDoStmt(); len(doStmt.Args) > 0 {
			if strNode := doStmt.Args[0].GetString_(); strNode != nil {
				return strNode.Sval
			}
		}
	}

	return ""
}

func instrumentWithLexer(stmt *parser.Statement, filePath string) (string, []CoveragePoint) {
	functionBodyText := extractFunctionBody(stmt)
	if functionBodyText == "" {
		return stmt.RawSQL, nil
	}

	// Scan the function body content with our PL/pgSQL lexer
	scanner := parser.NewScanner(functionBodyText)
	tokens := scanner.ScanAll()
	if len(tokens) == 0 {
		return stmt.RawSQL, nil
	}

	// Find executable statement boundaries in the body content (skip to BEGIN for PL/pgSQL)
	executableSegments := findExecutableSegments(functionBodyText, tokens, true)
	if len(executableSegments) == 0 {
		return stmt.RawSQL, nil
	}

	// Create coverage points and inject PERFORM calls
	return instrumentFunctionBodyFromAST(stmt, filePath, functionBodyText, executableSegments, "PERFORM")
}

type executableSegment struct {
	startPos  int // Position in body content
	endPos    int // Position in body content
	lineStart int // Line number in body content (0-based)
	lineEnd   int // Line number in body content (0-based)
}

// findExecutableSegments finds executable statement segments in function body.
// When skipToBegin is true (PL/pgSQL), tokens before the first BEGIN are skipped.
// When skipToBegin is false (SQL functions), all tokens are considered immediately.
func findExecutableSegments(bodyContent string, tokens []parser.Token, skipToBegin bool) []executableSegment {
	var segments []executableSegment

	hasExecutableContent := false
	firstExecutableTokenPos := -1

	if skipToBegin {
		for idx, token := range tokens {
			if token.Type == parser.KBegin { // Skip until BEGIN token
				tokens = tokens[idx+1:]
				break
			}
		}
	}

	for _, token := range tokens {
		// Skip comment tokens
		if token.Type == parser.Comment {
			continue
		}

		// Check if this is a semicolon (statement separator)
		if token.Type == parser.TokenType(';') {
			if hasExecutableContent && firstExecutableTokenPos >= 0 {
				// Check if this segment represents an executable statement
				segmentEnd := token.Pos
				segmentContent := bodyContent[firstExecutableTokenPos:segmentEnd]

				if isExecutableSegment(segmentContent) {
					segment := executableSegment{
						startPos:  firstExecutableTokenPos,
						endPos:    segmentEnd,
						lineStart: convertByteOffsetToLine(bodyContent, firstExecutableTokenPos),
						lineEnd:   convertByteOffsetToLine(bodyContent, segmentEnd),
					}
					segments = append(segments, segment)
				}
			}

			// Reset for next segment
			hasExecutableContent = false
			firstExecutableTokenPos = -1
		} else {
			// This is some non-comment token, so we have content
			if !hasExecutableContent {
				firstExecutableTokenPos = token.Pos
			}
			hasExecutableContent = true
		}
	}

	// Handle the last segment if there's remaining content
	if hasExecutableContent && firstExecutableTokenPos >= 0 && firstExecutableTokenPos < len(bodyContent) {
		segmentContent := bodyContent[firstExecutableTokenPos:]
		if isExecutableSegment(segmentContent) {
			segment := executableSegment{
				startPos:  firstExecutableTokenPos,
				endPos:    len(bodyContent),
				lineStart: convertByteOffsetToLine(bodyContent, firstExecutableTokenPos),
				lineEnd:   convertByteOffsetToLine(bodyContent, len(bodyContent)),
			}
			segments = append(segments, segment)
		}
	}

	return segments
}

// isExecutableSegment determines if a segment represents an executable statement
func isExecutableSegment(segmentContent string) bool {
	trimmedSegment := strings.TrimSpace(segmentContent)
	if trimmedSegment == "" {
		return false
	}

	// Convert to uppercase for easier matching
	upper := strings.ToUpper(trimmedSegment)

	// Skip pure structural statements (only if they don't contain other executable content)
	if upper == "BEGIN" || upper == "END" {
		return false
	}

	// Include assignment statements (contain :=)
	if strings.Contains(segmentContent, ":=") {
		return true
	}

	// Include RETURN statements
	if strings.HasPrefix(upper, "RETURN") || strings.Contains(upper, "\nRETURN ") || strings.Contains(upper, " RETURN ") {
		return true
	}

	// Include RAISE statements
	if strings.HasPrefix(upper, "RAISE") || strings.Contains(upper, "\nRAISE ") || strings.Contains(upper, " RAISE ") {
		return true
	}

	// Include PERFORM statements
	if strings.HasPrefix(upper, "PERFORM") || strings.Contains(upper, "\nPERFORM ") || strings.Contains(upper, " PERFORM ") {
		return true
	}

	// Include SQL statements (SELECT, INSERT, UPDATE, DELETE, etc.)
	if strings.HasPrefix(upper, "SELECT") || strings.Contains(upper, "\nSELECT ") || strings.Contains(upper, " SELECT ") ||
		strings.HasPrefix(upper, "INSERT") || strings.Contains(upper, "\nINSERT ") || strings.Contains(upper, " INSERT ") ||
		strings.HasPrefix(upper, "UPDATE") || strings.Contains(upper, "\nUPDATE ") || strings.Contains(upper, " UPDATE ") ||
		strings.HasPrefix(upper, "DELETE") || strings.Contains(upper, "\nDELETE ") || strings.Contains(upper, " DELETE ") {
		return true
	}

	return false
}

// convertByteOffsetToLine converts a byte offset to a 0-based line index
func convertByteOffsetToLine(sql string, byteOffset int) int {
	lineIdx := 0
	for i := 0; i < byteOffset && i < len(sql); i++ {
		if sql[i] == '\n' {
			lineIdx++
		}
	}
	return lineIdx
}

// instrumentFunctionBodyFromAST injects coverage-tracking calls using AST-extracted function body.
// notifyCmd is the SQL command used for the pg_notify call: "PERFORM" for PL/pgSQL, "SELECT" for SQL functions.
func instrumentFunctionBodyFromAST(stmt *parser.Statement, filePath string, bodyContent string, segments []executableSegment, notifyCmd string) (string, []CoveragePoint) {
	var locations []CoveragePoint

	// Find where the function body content actually starts in the original SQL
	// Use the AST location if available, otherwise search for the body content
	bodyIndexInOriginal := strings.Index(stmt.RawSQL, bodyContent)
	if bodyIndexInOriginal == -1 {
		// If we can't find the exact body content, try to find it with different whitespace
		normalizedBody := strings.TrimSpace(bodyContent)
		for i := 0; i < len(stmt.RawSQL)-len(normalizedBody); i++ {
			if strings.TrimSpace(stmt.RawSQL[i:i+len(normalizedBody)]) == normalizedBody {
				bodyIndexInOriginal = i
				break
			}
		}
	}

	if bodyIndexInOriginal == -1 {
		// Fallback: return original SQL if we can't find the body
		return stmt.RawSQL, nil
	}

	// Build instrumented function body
	instrumentedBody := strings.Builder{}
	lastProcessedPos := 0

	for _, segment := range segments {
		// Write any content before this segment
		if segment.startPos > lastProcessedPos {
			instrumentedBody.WriteString(bodyContent[lastProcessedPos:segment.startPos])
		}

		// Get the segment content
		segmentContent := bodyContent[segment.startPos:segment.endPos]
		segmentLines := strings.Split(segmentContent, "\n")

		// Create coverage point for this segment
		// Convert body positions to absolute file positions
		absoluteStartPos := stmt.StartPos + bodyIndexInOriginal + segment.startPos
		cp := CoveragePoint{
			File:             filePath,
			StartPos:         absoluteStartPos,
			Length:           len(segmentContent),
			Branch:           "",
			ImplicitCoverage: false,
		}
		cp.SignalID = FormatSignalID(cp.File, cp.StartPos, cp.Length, cp.Branch)
		locations = append(locations, cp)

		// Find the first non-empty line in segment to get proper indentation
		indent := ""
		for _, line := range segmentLines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				indent = getIndentation(line)
				break
			}
		}

		// Inject coverage-tracking pg_notify call before the segment
		notifyCall := fmt.Sprintf("%s%s pg_notify('pgcov', '%s');\n",
			indent, notifyCmd, strings.ReplaceAll(cp.SignalID, "'", "''"))
		instrumentedBody.WriteString(notifyCall)

		// Write the original segment content
		instrumentedBody.WriteString(segmentContent)

		lastProcessedPos = segment.endPos
	}

	// Write any remaining body content after the last segment
	if lastProcessedPos < len(bodyContent) {
		instrumentedBody.WriteString(bodyContent[lastProcessedPos:])
	}

	// Replace the function body in the original SQL
	result := stmt.RawSQL[:bodyIndexInOriginal] + instrumentedBody.String() + stmt.RawSQL[bodyIndexInOriginal+len(bodyContent):]

	return result, locations
}

// getIndentation returns the leading whitespace of a line
func getIndentation(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

// instrumentSQLFunction instruments a SQL-language function.
// SQL functions have no DECLARE/BEGIN block, so we scan the body immediately.
// Since PERFORM is not valid in SQL functions, we use SELECT pg_notify(...) instead.
func instrumentSQLFunction(stmt *parser.Statement, filePath string) (string, []CoveragePoint) {
	functionBodyText := extractFunctionBody(stmt)
	if functionBodyText == "" {
		return stmt.RawSQL, nil
	}

	// Scan the function body with our PL/pgSQL lexer (works for plain SQL too)
	scanner := parser.NewScanner(functionBodyText)
	tokens := scanner.ScanAll()
	if len(tokens) == 0 {
		return stmt.RawSQL, nil
	}

	// Find executable segments without skipping to BEGIN (SQL functions have no BEGIN)
	executableSegments := findExecutableSegments(functionBodyText, tokens, false)
	if len(executableSegments) == 0 {
		return stmt.RawSQL, nil
	}

	// Inject SELECT pg_notify calls
	return instrumentFunctionBodyFromAST(stmt, filePath, functionBodyText, executableSegments, "SELECT")
}

// markStatementLinesAsCovered creates coverage points for all non-comment lines
// Uses AST node location to determine the statement boundaries rather than string operations
func markStatementLinesAsCovered(stmt *parser.Statement, filePath string) []CoveragePoint {
	var locations []CoveragePoint

	// For DDL/DML statements, mark the entire statement as implicitly covered
	// Use the byte position from the parsed statement
	bytePos := stmt.StartPos
	stmtLength := len(stmt.RawSQL)

	cp := CoveragePoint{
		File:             filePath,
		StartPos:         bytePos,
		Length:           stmtLength,
		Branch:           "",
		ImplicitCoverage: true, // DDL/DML are implicitly covered on successful execution
	}
	cp.SignalID = FormatSignalID(cp.File, cp.StartPos, cp.Length, cp.Branch)
	locations = append(locations, cp)

	return locations
}
