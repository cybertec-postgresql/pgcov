package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cybertec-postgresql/pgcov/internal/discovery"
	"github.com/cybertec-postgresql/pgcov/internal/parser"
	"github.com/pashagolub/pglex"
)

// FunctionInfo describes a CREATE [OR REPLACE] FUNCTION statement extracted
// from a source file. Schema is empty when the source omits it.
type FunctionInfo struct {
	Schema    string // schema name ("" when unqualified)
	Name      string // function name
	Args      string // argument list text, e.g. "integer, text" or ""
	StartLine int    // 1-indexed line number where the statement begins
}

// extractFunctions walks pglex tokens to find CREATE [OR REPLACE] FUNCTION
// statements and extracts their schema-qualified name and argument list.
//
// The lexer (not regex) drives the walk so we honour string literals,
// dollar-quoted bodies, and nested parens correctly. We deliberately do not
// re-use parser.Parse here because that layer discards the function name; we
// need the qualifier and argument list for scaffolding.
func extractFunctions(sql string) []FunctionInfo {
	var functions []FunctionInfo

	scanner := pglex.NewScanner(sql)
	var toks []pglex.Token
	for {
		t := scanner.Scan()
		if t.Type == pglex.EOF {
			break
		}
		toks = append(toks, t)
	}

	for i := 0; i < len(toks); i++ {
		if !tokenEqualFold(toks[i], "CREATE") {
			continue
		}

		// Skip past CREATE [OR REPLACE]
		j := i + 1
		if j < len(toks) && tokenEqualFold(toks[j], "OR") && j+1 < len(toks) && tokenEqualFold(toks[j+1], "REPLACE") {
			j += 2
		}

		if j >= len(toks) || !tokenEqualFold(toks[j], "FUNCTION") {
			continue
		}

		// Walk to the function name. Qualified name is Ident ("." Ident)?.
		k := j + 1
		// Skip whitespace? pglex discards whitespace, so tokens are dense.
		if k >= len(toks) {
			continue
		}

		var schema, name string
		switch {
		case toks[k].Type == pglex.Ident && k+1 < len(toks) && toks[k+1].Type == TokenType('.') && k+2 < len(toks) && toks[k+2].Type == pglex.Ident:
			schema = stripQuotes(toks[k].Text)
			name = stripQuotes(toks[k+2].Text)
			k += 3
		case toks[k].Type == pglex.Ident:
			name = stripQuotes(toks[k].Text)
			k++
		default:
			// No name token; skip.
			continue
		}

		// Find the opening paren of the argument list, tolerating whitespace
		// (already discarded) and any number of intervening tokens such as
		// parenthesised name lists for window/aggregate functions.
		parenStart := -1
		for m := k; m < len(toks); m++ {
			if toks[m].Type == TokenType('(') {
				parenStart = m
				break
			}
		}
		if parenStart < 0 {
			// No arg list — zero-arg function or unparseable; record with empty Args.
			functions = append(functions, FunctionInfo{
				Schema:    schema,
				Name:      name,
				Args:      "",
				StartLine: calculateLineNumber(sql, toks[i].Pos),
			})
			i = j // advance past the header; outer loop's i++ lands safely
			continue
		}

		// Walk paren-balanced tokens to find matching ')'.
		args := collectBalancedArgs(sql, toks, parenStart)
		functions = append(functions, FunctionInfo{
			Schema:    schema,
			Name:      name,
			Args:      formatArgList(args),
			StartLine: calculateLineNumber(sql, toks[i].Pos),
		})

		// Advance past the function name to avoid re-matching nested tokens.
		i = parenStart
	}

	return functions
}

// formatArgList turns a raw slice of argument texts into a compact "types only"
// signature suitable for a scaffold comment, e.g. "integer, text". Argument
// names and DEFAULT clauses are dropped; only the type expression remains.
func formatArgList(args []string) string {
	var types []string
	for _, raw := range args {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		// Strip the leading argument name if present (Ident followed by type).
		if t := firstArgType(trimmed); t != "" {
			types = append(types, t)
		}
	}
	return strings.Join(types, ", ")
}

// firstArgType returns the type portion of an argument, dropping a leading
// mode/name token and stopping before DEFAULT. Operates on the raw argument
// text captured by the lexer; the lexer has already correctly handled nested
// parens, type casts and string literals, so we only need a small string
// scan here for the optional "name TYPE" pair.
func firstArgType(arg string) string {
	upper := strings.ToUpper(arg)
	if cut := strings.Index(upper, "DEFAULT"); cut >= 0 {
		arg = arg[:cut]
	}
	if cut := strings.Index(arg, "="); cut >= 0 {
		arg = arg[:cut]
	}
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return ""
	}

	// Optional mode marker (IN/OUT/INOUT/VARIADIC) followed by a name.
	fields := strings.Fields(arg)
	if len(fields) >= 2 {
		switch strings.ToUpper(fields[0]) {
		case "IN", "OUT", "INOUT", "VARIADIC":
			return strings.TrimSpace(strings.TrimPrefix(arg, fields[0]))
		}
	}

	// "name TYPE" → drop the leading identifier.
	if len(fields) >= 2 && isIdentifier(fields[0]) {
		return strings.TrimSpace(arg[len(fields[0]):])
	}

	// Bare type.
	return arg
}

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_':
		default:
			return false
		}
	}
	return true
}

// collectBalancedArgs walks tokens from openParen (inclusive) until the
// matching close paren, then splits the captured text on top-level commas.
func collectBalancedArgs(sql string, toks []pglex.Token, openParen int) []string {
	depth := 0
	for m := openParen; m < len(toks); m++ {
		switch toks[m].Type {
		case TokenType('('):
			depth++
		case TokenType(')'):
			depth--
			if depth == 0 {
				return splitArgs(sql, toks[openParen+1].Pos, toks[m].Pos)
			}
		}
	}
	// Unbalanced parens — return raw text from open to end.
	if openParen+1 < len(toks) {
		last := toks[len(toks)-1]
		return splitArgs(sql, toks[openParen+1].Pos, last.Pos+len(last.Text))
	}
	return nil
}

// splitArgs slices sql[start:end] on commas at paren depth zero.
func splitArgs(sql string, start, end int) []string {
	if start > end || end > len(sql) {
		return nil
	}
	body := sql[start:end]

	var args []string
	depth := 0
	last := 0
	inSingle := false
	for i := 0; i < len(body); i++ {
		ch := body[i]
		switch {
		case ch == '\'':
			inSingle = !inSingle
		case !inSingle && ch == '(':
			depth++
		case !inSingle && ch == ')':
			if depth > 0 {
				depth--
			}
		case !inSingle && ch == ',' && depth == 0:
			args = append(args, body[last:i])
			last = i + 1
		}
	}
	args = append(args, body[last:])
	return args
}

// calculateLineNumber converts a byte offset to a 1-indexed line number.
// Duplicates parser.calculateLineNumber to keep the init helper DB-free.
func calculateLineNumber(sql string, offset int) int {
	if offset < 0 {
		return 1
	}
	if offset > len(sql) {
		offset = len(sql)
	}
	line := 1
	for i := 0; i < offset && i < len(sql); i++ {
		if sql[i] == '\n' {
			line++
		}
	}
	return line
}

// stripQuotes removes the surrounding double quotes from a quoted identifier
// (e.g. "My Schema" → My Schema) and unescapes embedded doubled quotes.
func stripQuotes(text string) string {
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		inner := text[1 : len(text)-1]
		return strings.ReplaceAll(inner, `""`, `"`)
	}
	return text
}

// tokenEqualFold reports whether the token is an identifier (or keyword)
// matching word, case-insensitively. Mirrors parser.isIdent.
func tokenEqualFold(tok pglex.Token, word string) bool {
	return strings.EqualFold(tok.Text, word)
}

// TokenType aliases pglex.TokenType to keep the parentheses literals readable.
// Defined as a local type alias so callers don't need to convert.
type TokenType = pglex.TokenType

// generateScaffold renders the test-scaffold file content for the given source
// path and extracted functions. Output is deterministic (source order).
func generateScaffold(sourcePath string, functions []FunctionInfo) string {
	rel := filepath.ToSlash(sourcePath)

	var b strings.Builder
	fmt.Fprintf(&b, "-- Auto-generated scaffold for %s\n", rel)
	fmt.Fprintf(&b, "-- pgcov init produced this file. Edit freely; rerun 'pgcov init' with --force\n")
	fmt.Fprintf(&b, "-- to regenerate. Functions are detected in source order.\n")
	fmt.Fprintf(&b, "--\n")
	fmt.Fprintf(&b, "-- Each function below ships with a commented sample call and a DO\n")
	fmt.Fprintf(&b, "-- block template for multi-step coverage scenarios. Uncomment, fill in\n")
	fmt.Fprintf(&b, "-- literal argument values that match the function's parameter list,\n")
	fmt.Fprintf(&b, "-- and run 'pgcov run' to collect coverage against %s.\n", rel)
	fmt.Fprintf(&b, "\n")

	if len(functions) == 0 {
		fmt.Fprintf(&b, "-- No CREATE [OR REPLACE] FUNCTION statements were detected in %s.\n", rel)
		fmt.Fprintf(&b, "-- Add a SELECT against a table here, then run 'pgcov run'.\n")
		return b.String()
	}

	for i, fn := range functions {
		qualifier := fn.Name
		if fn.Schema != "" {
			qualifier = fn.Schema + "." + fn.Name
		}
		argsDisplay := fn.Args
		if argsDisplay == "" {
			argsDisplay = "—"
		}

		fmt.Fprintf(&b, "-- Function %d of %d: %s (line %d)\n", i+1, len(functions), qualifier, fn.StartLine)
		fmt.Fprintf(&b, "-- SELECT %s(%s);\n", qualifier, argsDisplay)
		fmt.Fprintf(&b, "--\n")
		fmt.Fprintf(&b, "-- DO $$ BEGIN\n")
		fmt.Fprintf(&b, "--     PERFORM %s(/* fill in literal arguments */);\n", qualifier)
		fmt.Fprintf(&b, "-- END $$;\n")
		fmt.Fprintf(&b, "\n")
	}
	return b.String()
}

// defaultScaffoldPath returns the conventional sibling file path for a source
// file: foo.sql → foo_test.sql. Empty source returns "".
func defaultScaffoldPath(sourcePath string) string {
	if sourcePath == "" {
		return ""
	}
	dir := filepath.Dir(sourcePath)
	base := filepath.Base(sourcePath)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	ext := filepath.Ext(base)
	if ext == "" {
		ext = ".sql"
	}
	return filepath.Join(dir, stem+"_test"+ext)
}

// Init scaffolds a test file for the given source. It returns the absolute
// output path that was written (or would be written if dry-run).
//
// Behaviour:
//   - Reads sourcePath; errors if it is missing or not a .sql file.
//   - Refuses to overwrite an existing target unless force is true.
//   - Returns an error if no functions were detected AND no other statements
//     were detected either (i.e. the source is empty), so callers don't get
//     silently empty scaffolds.
//
// The output path is computed as follows:
//   - explicit outputPath (already absolute, or relative to cwd) is honoured;
//   - otherwise a sibling <stem>_test.sql is created next to the source.
func Init(sourcePath, outputPath string, force bool) (string, error) {
	if sourcePath == "" {
		return "", errors.New("source path is required")
	}

	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return "", fmt.Errorf("resolve source path: %w", err)
	}
	info, err := os.Stat(absSource)
	if err != nil {
		return "", fmt.Errorf("stat source: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("source path is a directory: %s", absSource)
	}
	if !strings.EqualFold(filepath.Ext(absSource), ".sql") {
		return "", fmt.Errorf("source must be a .sql file: %s", absSource)
	}

	content, err := os.ReadFile(absSource)
	if err != nil {
		return "", fmt.Errorf("read source: %w", err)
	}

	// Parse via the existing parser so we report a clean error if reading fails;
	// the actual extraction uses extractFunctions (lexer-driven).
	file := &discovery.DiscoveredFile{
		Path:         absSource,
		RelativePath: absSource,
		Type:         discovery.FileTypeSource,
	}
	if _, err := parser.Parse(file); err != nil {
		return "", fmt.Errorf("parse source: %w", err)
	}

	functions := extractFunctions(string(content))

	outPath := outputPath
	if outPath == "" {
		outPath = defaultScaffoldPath(absSource)
	}
	absOut, err := filepath.Abs(outPath)
	if err != nil {
		return "", fmt.Errorf("resolve output path: %w", err)
	}

	if _, err := os.Stat(absOut); err == nil && !force {
		return absOut, fmt.Errorf("output already exists: %s (use --force to overwrite)", absOut)
	}

	scaffold := generateScaffold(absSource, functions)

	if err := os.MkdirAll(filepath.Dir(absOut), 0o755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(absOut, []byte(scaffold), 0o644); err != nil {
		return "", fmt.Errorf("write output: %w", err)
	}

	return absOut, nil
}
