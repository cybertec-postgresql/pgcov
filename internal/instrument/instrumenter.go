package instrument

import (
	"fmt"
	"strings"

	"github.com/pashagolub/pgcov/internal/parser"
)

// Instrument takes parsed SQL and injects coverage tracking calls
func Instrument(parsed *parser.ParsedSQL) (*InstrumentedSQL, error) {
	// Phase 3: Use simple instrumentation (no NOTIFY injection)
	return InstrumentSimple(parsed)
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

// Unused helper functions - kept for future Phase 4 implementation

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

func generateNotifyCall(signalID string) string {
	escapedID := strings.ReplaceAll(signalID, "'", "''")
	return fmt.Sprintf("NOTIFY pgcov, '%s';\n", escapedID)
}

