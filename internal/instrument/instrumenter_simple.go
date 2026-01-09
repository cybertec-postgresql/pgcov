package instrument

import (
	"fmt"
	"os"

	"github.com/cybertec-postgresql/pgcov/internal/parser"
)

// InstrumentSimple is a Phase 3 implementation that doesn't inject NOTIFY yet
// It just tracks coverage points and returns the original SQL
func InstrumentSimple(parsed *parser.ParsedSQL) (*InstrumentedSQL, error) {
	if parsed == nil || parsed.File == nil {
		return nil, fmt.Errorf("parsed SQL or file is nil")
	}

	// Read original SQL
	originalSQLBytes, err := os.ReadFile(parsed.File.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read source file: %w", err)
	}
	originalSQL := string(originalSQLBytes)

	var locations []CoveragePoint

	// Track coverage points for each statement
	for _, stmt := range parsed.Statements {
		relPath := parsed.File.RelativePath
		if relPath == "" {
			relPath = parsed.File.Path
		}

		cp := CoveragePoint{
			File:   relPath,
			Line:   stmt.StartLine,
			Branch: "",
		}
		cp.SignalID = FormatSignalID(cp.File, cp.Line, cp.Branch)
		locations = append(locations, cp)
	}

	// Phase 3: Return original SQL without instrumentation
	// Phase 4 will add proper NOTIFY injection inside function bodies
	return &InstrumentedSQL{
		Original:         parsed,
		InstrumentedText: originalSQL,
		Locations:        locations,
	}, nil
}
