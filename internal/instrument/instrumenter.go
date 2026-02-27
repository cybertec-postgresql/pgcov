package instrument

import (
	"fmt"
	"strings"

	"github.com/cybertec-postgresql/pgcov/internal/parser"
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

	// For functions/procedures, determine the language from the parsed statement
	if stmt.Type == parser.StmtFunction || stmt.Type == parser.StmtProcedure {
		switch stmt.Language {
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

	// For DO blocks, instrument the body
	if stmt.Type == parser.StmtDO {
		instrumented, locs := instrumentWithLexer(stmt, filePath)
		return instrumented, locs
	}

	// For non-function statements (DDL, DML), mark all non-comment lines as covered
	// These will be automatically marked as covered if the file executes without errors
	locations = markStatementLinesAsCovered(stmt, filePath)

	// Return original SQL without instrumentation - DDL/DML are implicitly covered on success
	return stmt.RawSQL, locations
}

// extractFunctionBody extracts the function/DO-block body text from the Statement.
// Returns the body text or "" if not found.
func extractFunctionBody(stmt *parser.Statement) string {
	return stmt.Body
}

func instrumentWithLexer(stmt *parser.Statement, filePath string) (string, []CoveragePoint) {
	return instrumentBody(stmt, filePath, true, "PERFORM")
}

// instrumentSQLFunction instruments a SQL-language function.
// SQL functions have no DECLARE/BEGIN block, so we scan the body immediately.
// Since PERFORM is not valid in SQL functions, we use SELECT pg_notify(...) instead.
func instrumentSQLFunction(stmt *parser.Statement, filePath string) (string, []CoveragePoint) {
	return instrumentBody(stmt, filePath, false, "SELECT")
}

// instrumentBody scans the function body token-by-token using the streaming
// Scan() method and injects coverage-tracking calls at each executable
// statement boundary.  This single-pass approach mirrors SplitStatements and
// avoids materializing the full token slice, which saves memory on large bodies.
//
// For PL/pgSQL (skipToBegin=true), tokens before the first BEGIN are skipped.
// For SQL functions (skipToBegin=false), instrumentation starts immediately.
// notifyCmd is "PERFORM" for PL/pgSQL or "SELECT" for SQL functions.
func instrumentBody(stmt *parser.Statement, filePath string, skipToBegin bool, notifyCmd string) (string, []CoveragePoint) {
	bodyContent := extractFunctionBody(stmt)
	if bodyContent == "" {
		return stmt.RawSQL, nil
	}

	// Use the pre-computed body offset within the statement text.
	bodyIndexInOriginal := stmt.BodyStart
	if bodyIndexInOriginal < 0 || bodyIndexInOriginal > len(stmt.RawSQL) {
		return stmt.RawSQL, nil
	}

	sc := parser.NewScanner(bodyContent)

	var locations []CoveragePoint
	var instrumentedBody strings.Builder
	lastWrittenPos := 0
	pastBegin := !skipToBegin

	// Current-segment tracking (same state as the old findExecutableSegments).
	hasContent := false
	segStart := -1

	// emitSegment checks the segment between segStart..segEnd for
	// executability and, if it qualifies, writes the gap + notify + segment
	// into instrumentedBody.
	emitSegment := func(segEnd int) {
		segText := bodyContent[segStart:segEnd]
		if !isExecutableSegment(segText) {
			return
		}

		// Write any unwritten content preceding this segment.
		if segStart > lastWrittenPos {
			instrumentedBody.WriteString(bodyContent[lastWrittenPos:segStart])
		}

		// Build coverage point.
		absoluteStartPos := stmt.StartPos + bodyIndexInOriginal + segStart
		cp := CoveragePoint{
			File:             filePath,
			StartPos:         absoluteStartPos,
			Length:           len(segText),
			Branch:           "",
			ImplicitCoverage: false,
		}
		cp.SignalID = FormatSignalID(cp.File, cp.StartPos, cp.Length, cp.Branch)
		locations = append(locations, cp)

		// Determine indentation from the first non-empty line.
		indent := ""
		for _, line := range strings.Split(segText, "\n") {
			if strings.TrimSpace(line) != "" {
				indent = getIndentation(line)
				break
			}
		}

		// Write notify call, then the original segment text.
		fmt.Fprintf(&instrumentedBody, "%s%s pg_notify('pgcov', '%s');\n",
			indent, notifyCmd, strings.ReplaceAll(cp.SignalID, "'", "''"))
		instrumentedBody.WriteString(segText)
		lastWrittenPos = segEnd
	}

	// Stream tokens one at a time – mirrors SplitStatements style.
	for {
		tok := sc.Scan()
		if tok.Type == parser.EOF {
			break
		}

		// Skip everything before the first BEGIN in PL/pgSQL bodies.
		if !pastBegin {
			if tok.Type == parser.KBegin {
				pastBegin = true
			}
			continue
		}

		// Comments are not executable content.
		if tok.Type == parser.Comment {
			continue
		}

		if tok.Type == parser.TokenType(';') {
			if hasContent && segStart >= 0 {
				emitSegment(tok.Pos)
			}
			hasContent = false
			segStart = -1
		} else {
			if !hasContent {
				segStart = tok.Pos
			}
			hasContent = true
		}
	}

	// Handle a trailing segment that has no closing semicolon.
	if hasContent && segStart >= 0 && segStart < len(bodyContent) {
		emitSegment(len(bodyContent))
	}

	if len(locations) == 0 {
		return stmt.RawSQL, nil
	}

	// Flush any remaining body content after the last instrumented segment.
	if lastWrittenPos < len(bodyContent) {
		instrumentedBody.WriteString(bodyContent[lastWrittenPos:])
	}

	result := stmt.RawSQL[:bodyIndexInOriginal] + instrumentedBody.String() + stmt.RawSQL[bodyIndexInOriginal+len(bodyContent):]
	return result, locations
}

// isExecutableSegment determines whether a ;-terminated segment from a function
// body represents executable code.  It scans the first token using the PL/pgSQL
// lexer instead of relying on string-prefix matching.
//
// The logic is an exclusion list: everything is considered executable except
// known structural markers (BEGIN, END, LOOP, DECLARE, EXCEPTION).
func isExecutableSegment(segmentContent string) bool {
	sc := parser.NewScanner(segmentContent)

	// Find the first non-comment token.
	var first parser.Token
	for {
		first = sc.Scan()
		if first.Type == parser.EOF {
			return false // empty or comments-only
		}
		if first.Type != parser.Comment {
			break
		}
	}

	switch first.Type {
	case parser.KBegin, parser.KEnd, parser.KLoop, parser.KDeclare, parser.KException:
		// Pure block openers/closers and declaration sections — not useful code.
		return false
	}

	// Any other leading token (identifier, keyword, operator, etc.)
	// indicates an executable statement.
	return true
}

// getIndentation returns the leading whitespace of a line.
func getIndentation(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
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
