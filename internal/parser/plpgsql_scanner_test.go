package parser

import (
	"fmt"
	"strings"
	"testing"
)

// ── helpers ──────────────────────────────────────────────────────────────────

// tokTypes returns just the TokenType values from ScanAll.
func tokTypes(src string) []TokenType {
	s := NewScanner(src)
	tokens := s.ScanAll()
	types := make([]TokenType, len(tokens))
	for i, t := range tokens {
		types[i] = t.Type
	}
	return types
}

// tokTexts returns just the Text values from ScanAll.
func tokTexts(src string) []string {
	s := NewScanner(src)
	tokens := s.ScanAll()
	texts := make([]string, len(tokens))
	for i, t := range tokens {
		texts[i] = t.Text
	}
	return texts
}

// first returns the first token from src.
func first(src string) Token {
	return NewScanner(src).Scan()
}

// assertTypes fails the test when the produced token type sequence does not
// match expected.
func assertTypes(t *testing.T, src string, want ...TokenType) {
	t.Helper()
	got := tokTypes(src)
	if len(got) != len(want) {
		t.Fatalf("src=%q\n  got  %v\n  want %v", src, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("src=%q token[%d]: got %v, want %v\n  full got:  %v\n  full want: %v",
				src, i, got[i], want[i], got, want)
		}
	}
}

// assertTexts fails the test when the produced token text sequence does not
// match expected.
func assertTexts(t *testing.T, src string, want ...string) {
	t.Helper()
	got := tokTexts(src)
	if len(got) != len(want) {
		t.Fatalf("src=%q\n  got  %v\n  want %v", src, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("src=%q token[%d]: got %q, want %q", src, i, got[i], want[i])
		}
	}
}

// ── EOF / empty input ─────────────────────────────────────────────────────────

func TestEmpty(t *testing.T) {
	tok := first("")
	if tok.Type != EOF {
		t.Fatalf("got %v, want EOF", tok.Type)
	}
}

func TestWhitespaceOnly(t *testing.T) {
	tok := first("   \t\n  ")
	if tok.Type != EOF {
		t.Fatalf("got %v, want EOF", tok.Type)
	}
}

// ── Comments ─────────────────────────────────────────────────────────────────

func TestLineComment(t *testing.T) {
	tok := first("-- this is a comment\n")
	if tok.Type != Comment {
		t.Fatalf("got %v, want Comment", tok.Type)
	}
	if tok.Text != "-- this is a comment" {
		t.Fatalf("got %q", tok.Text)
	}
}

func TestBlockComment(t *testing.T) {
	tok := first("/* hello */")
	if tok.Type != Comment || tok.Text != "/* hello */" {
		t.Fatalf("got %v %q", tok.Type, tok.Text)
	}
}

func TestNestedBlockComment(t *testing.T) {
	// PostgreSQL allows nested /* /* */ */ – must not terminate at inner */.
	tok := first("/* outer /* inner */ still outer */")
	if tok.Type != Comment {
		t.Fatalf("got %v", tok.Type)
	}
	if !strings.HasSuffix(tok.Text, "still outer */") {
		t.Fatalf("nested comment ended too early: %q", tok.Text)
	}
}

func TestSemicolonInsideComment(t *testing.T) {
	// The semicolon is inside a comment and must NOT produce a ';' token.
	types := tokTypes("/* ; */ x")
	for _, tt := range types {
		if tt == TokenType(';') {
			t.Fatal("semicolon inside comment was incorrectly tokenised as ';'")
		}
	}
}

// ── String literals ───────────────────────────────────────────────────────────

func TestSingleQuotedString(t *testing.T) {
	assertTypes(t, `'hello'`, SConst)
	assertTexts(t, `'hello'`, `'hello'`)
}

func TestSingleQuotedWithEscapedQuote(t *testing.T) {
	assertTexts(t, `'it''s'`, `'it''s'`)
	assertTypes(t, `'it''s'`, SConst)
}

func TestEmptyString(t *testing.T) {
	assertTypes(t, `''`, SConst)
	assertTexts(t, `''`, `''`)
}

func TestEscapeString(t *testing.T) {
	assertTypes(t, `E'line\nbreak'`, SConst)
	assertTexts(t, `E'line\nbreak'`, `E'line\nbreak'`)
}

func TestEscapeStringUpperCase(t *testing.T) {
	assertTypes(t, `E'hello'`, SConst)
}

func TestBitString(t *testing.T) {
	assertTypes(t, `B'1010'`, BConst)
	assertTexts(t, `B'1010'`, `B'1010'`)
}

func TestHexString(t *testing.T) {
	assertTypes(t, `X'DEAD'`, XConst)
	assertTexts(t, `X'DEAD'`, `X'DEAD'`)
}

// N'...' must produce TWO tokens: Ident "N" + SConst '...'
// scan.l {xnstart} uses yyless(1) to push back the ' and returns n/N as IDENT.
func TestNationalStringIsTwoTokens(t *testing.T) {
	assertTypes(t, `N'text'`, Ident, SConst)
	assertTexts(t, `N'text'`, "N", "'text'")
}

func TestNationalStringLowercase(t *testing.T) {
	assertTypes(t, `n'text'`, Ident, SConst)
	assertTexts(t, `n'text'`, "n", "'text'")
}

func TestUnicodeString(t *testing.T) {
	assertTypes(t, `U&'caf\00E9'`, SConst)
}

func TestUnicodeStringFallback(t *testing.T) {
	// U& not followed by ' or " → U as Ident, & as Op.
	assertTypes(t, `U&x`, Ident, Op, Ident)
	assertTexts(t, `U&x`, "U", "&", "x")
}

func TestUnicodeQuotedIdent(t *testing.T) {
	assertTypes(t, `U&"mycol"`, Ident)
}

func TestSemicolonInsideString(t *testing.T) {
	// ';' inside a string must NOT produce a ';' token.
	types := tokTypes(`'a ; b'`)
	for _, tt := range types {
		if tt == TokenType(';') {
			t.Fatal("semicolon inside string tokenised as ';'")
		}
	}
}

// ── Dollar-quoted strings ─────────────────────────────────────────────────────

func TestDollarQuoteEmpty(t *testing.T) {
	assertTypes(t, `$$hello$$`, SConst)
	assertTexts(t, `$$hello$$`, `$$hello$$`)
}

func TestDollarQuoteTagged(t *testing.T) {
	assertTypes(t, `$body$x := 1;$body$`, SConst)
	// The semicolon inside must not produce a ';' token.
	types := tokTypes(`$body$x := 1;$body$`)
	for _, tt := range types {
		if tt == TokenType(';') {
			t.Fatal("semicolon inside dollar-quote tokenised as ';'")
		}
	}
}

func TestDollarQuotedNested(t *testing.T) {
	// Nested $...$ inside the body must not terminate the outer quote.
	src := `$outer$outer $inner$ still $inner$ outer$outer$`
	assertTypes(t, src, SConst)
	if !strings.HasSuffix(src, "outer$outer$") {
		t.Fatalf("dollar-quote ended too early: %q", src)
	}
}

func TestDollarQuoteUnicodeTag(t *testing.T) {
	// Tags with high bytes (≥ 0x80) are valid per scan.l dolq_start/dolq_cont.
	src := "$\xc3\xa9tag$content$\xc3\xa9tag$" // $étag$content$étag$
	assertTypes(t, src, SConst)
}

func TestDollarQuoteWithSemicolon(t *testing.T) {
	src := "$$SELECT 1; SELECT 2;$$ ;"
	stmts := SplitStatements(src)
	// There should be exactly one statement ending with ';' after the dollar-quoted block.
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts))
	}
}

// ── Double-quoted identifiers ─────────────────────────────────────────────────

func TestDoubleQuotedIdent(t *testing.T) {
	assertTypes(t, `"MyTable"`, Ident)
	assertTexts(t, `"MyTable"`, `"MyTable"`)
}

func TestDoubleQuotedIdentEscapedQuote(t *testing.T) {
	assertTexts(t, `"it""s"`, `"it""s"`)
}

// ── Positional parameters ─────────────────────────────────────────────────────

func TestParam(t *testing.T) {
	assertTypes(t, `$1`, Param)
	assertTexts(t, `$1`, `$1`)
}

func TestParamMultiDigit(t *testing.T) {
	assertTypes(t, `$42`, Param)
	assertTexts(t, `$42`, `$42`)
}

// ── Numeric literals ──────────────────────────────────────────────────────────

func TestDecimalInt(t *testing.T) {
	assertTypes(t, `42`, IConst)
	assertTexts(t, `42`, `42`)
}

func TestDecimalIntWithUnderscores(t *testing.T) {
	assertTypes(t, `1_000_000`, IConst)
	assertTexts(t, `1_000_000`, `1_000_000`)
}

func TestHexInt(t *testing.T) {
	assertTypes(t, `0xDEAD`, IConst)
	assertTexts(t, `0xDEAD`, `0xDEAD`)
}

func TestHexIntWithUnderscores(t *testing.T) {
	assertTypes(t, `0xFF_FF`, IConst)
	assertTexts(t, `0xFF_FF`, `0xFF_FF`)
}

func TestOctalInt(t *testing.T) {
	assertTypes(t, `0o777`, IConst)
}

func TestOctalIntWithUnderscores(t *testing.T) {
	assertTypes(t, `0o7_7`, IConst)
}

func TestBinaryInt(t *testing.T) {
	assertTypes(t, `0b1010`, IConst)
}

func TestBinaryIntWithUnderscores(t *testing.T) {
	assertTypes(t, `0b1_010`, IConst)
}

func TestFloat(t *testing.T) {
	assertTypes(t, `3.14`, FConst)
	assertTexts(t, `3.14`, `3.14`)
}

func TestFloatWithExponent(t *testing.T) {
	assertTypes(t, `1e5`, FConst)
	assertTexts(t, `1e5`, `1e5`)
}

func TestFloatLeadingDot(t *testing.T) {
	// .5 is a float, not DotDot + integer.
	assertTypes(t, `.5`, FConst)
}

func TestDotDotNotFloat(t *testing.T) {
	// 1..2 is IConst DotDot IConst (range), not a float.
	assertTypes(t, `1..2`, IConst, DotDot, IConst)
	assertTexts(t, `1..2`, `1`, `..`, `2`)
}

func TestFloatWithUnderscoreDigits(t *testing.T) {
	assertTypes(t, `1_0.2_0`, FConst)
	assertTexts(t, `1_0.2_0`, `1_0.2_0`)
}

// ── Reserved keywords (case-insensitive) ─────────────────────────────────────

func TestReservedKeywords(t *testing.T) {
	cases := []struct {
		src  string
		want TokenType
	}{
		{"BEGIN", KBegin}, {"begin", KBegin}, {"Begin", KBegin},
		{"DECLARE", KDeclare}, {"END", KEnd}, {"IF", KIf}, {"THEN", KThen},
		{"ELSE", KElse}, {"ELSIF", KElsif}, {"ELSEIF", KElsif},
		{"LOOP", KLoop}, {"WHILE", KWhile}, {"FOR", KFor}, {"FOREACH", KForeach},
		{"RETURN", KReturn}, {"NULL", KNull}, {"NOT", KNot}, {"OR", KOr},
		{"ALL", KAll}, {"BY", KBy}, {"CASE", KCase}, {"FROM", KFrom},
		{"IN", KIn}, {"INTO", KInto}, {"TO", KTo}, {"USING", KUsing},
		{"WHEN", KWhen},
	}
	for _, c := range cases {
		tok := first(c.src)
		if tok.Type != c.want {
			t.Errorf("src=%q: got %v, want %v", c.src, tok.Type, c.want)
		}
	}
}

func TestUnreservedKeywords(t *testing.T) {
	cases := []struct {
		src  string
		want TokenType
	}{
		{"RAISE", KRaise}, {"OPEN", KOpen}, {"CLOSE", KClose},
		{"FETCH", KFetch}, {"MOVE", KMove}, {"EXECUTE", KExecute},
		{"PERFORM", KPerform}, {"GET", KGet}, {"DIAGNOSTICS", KDiagnostics},
		{"STRICT", KStrict}, {"CURSOR", KCursor}, {"EXIT", KExit},
		{"CONTINUE", KContinue}, {"EXCEPTION", KException},
		{"RETURN", KReturn}, {"ROWTYPE", KRowtype},
	}
	for _, c := range cases {
		tok := first(c.src)
		if tok.Type != c.want {
			t.Errorf("src=%q: got %v, want %v", c.src, tok.Type, c.want)
		}
	}
}

func TestKeywordPreservesCase(t *testing.T) {
	tok := first("BEGIN")
	if tok.Text != "BEGIN" {
		t.Fatalf("got %q, want %q", tok.Text, "BEGIN")
	}
}

// ── Multi-character operator tokens ──────────────────────────────────────────

func TestColonEquals(t *testing.T) {
	assertTypes(t, `:=`, ColonEquals)
}

func TestTypecast(t *testing.T) {
	assertTypes(t, `::`, Typecast)
}

func TestDotDot(t *testing.T) {
	assertTypes(t, `..`, DotDot)
}

func TestLessLess(t *testing.T) {
	assertTypes(t, `<<`, LessLess)
	assertTexts(t, `<<`, `<<`)
}

func TestGreaterGreater(t *testing.T) {
	assertTypes(t, `>>`, GreaterGreater)
}

func TestLessEquals(t *testing.T) {
	assertTypes(t, `<=`, LessEquals)
}

func TestGreaterEquals(t *testing.T) {
	assertTypes(t, `>=`, GreaterEquals)
}

func TestEqualsGreater(t *testing.T) {
	assertTypes(t, `=>`, EqualsGreater)
}

func TestNotEqualsAngle(t *testing.T) {
	assertTypes(t, `<>`, NotEquals)
}

func TestNotEqualsBang(t *testing.T) {
	assertTypes(t, `!=`, NotEquals)
}

// ── Single-character punctuation from the self set ───────────────────────────

func TestSemicolon(t *testing.T) {
	assertTypes(t, `;`, TokenType(';'))
}

func TestComma(t *testing.T) {
	assertTypes(t, `,`, TokenType(','))
}

func TestOpenParen(t *testing.T) {
	assertTypes(t, `(`, TokenType('('))
}

func TestCloseParen(t *testing.T) {
	assertTypes(t, `)`, TokenType(')'))
}

func TestDot(t *testing.T) {
	assertTypes(t, `.`, TokenType('.'))
}

func TestColon(t *testing.T) {
	assertTypes(t, `:`, TokenType(':'))
}

func TestPlus(t *testing.T) {
	assertTypes(t, `+`, TokenType('+'))
}

func TestMinus(t *testing.T) {
	assertTypes(t, `-`, TokenType('-'))
}

func TestStar(t *testing.T) {
	assertTypes(t, `*`, TokenType('*'))
}

func TestSlash(t *testing.T) {
	assertTypes(t, `/`, TokenType('/'))
}

func TestPercent(t *testing.T) {
	assertTypes(t, `%`, TokenType('%'))
}

func TestCaret(t *testing.T) {
	assertTypes(t, `^`, TokenType('^'))
}

func TestLessThan(t *testing.T) {
	assertTypes(t, `<`, TokenType('<'))
}

func TestGreaterThan(t *testing.T) {
	assertTypes(t, `>`, TokenType('>'))
}

func TestEquals(t *testing.T) {
	assertTypes(t, `=`, TokenType('='))
}

func TestHash(t *testing.T) {
	// # is an op_char; pl_scanner.c converts Op "#" → '#'.
	assertTypes(t, `#`, TokenType('#'))
}

// ── Trailing +/- stripping rule ───────────────────────────────────────────────

func TestTrailingPlusStripped(t *testing.T) {
	// scan.l: "+-" matches {operator} (2 chars), trailing '-' is stripped
	// ('+' is not qualifying), yyless(1) pushes '-' back.
	// Result: two tokens — '+' from the operator block, then '-' from the next scan.
	assertTypes(t, `+-`, TokenType('+'), TokenType('-'))
}

func TestTrailingMinusStripped(t *testing.T) {
	// "-+" → same logic, two tokens.
	assertTypes(t, `-+`, TokenType('-'), TokenType('+'))
}

func TestTrailingMinusNotStrippedWithQualifying(t *testing.T) {
	// "~-" ends with -, '~' is qualifying → NOT stripped, result is Op "~-".
	assertTypes(t, `~-`, Op)
	assertTexts(t, `~-`, `~-`)
}

func TestTrailingPlusNotStrippedWithQualifying(t *testing.T) {
	// "+|+" ends with +, has '|' → not stripped.
	assertTypes(t, `+|+`, Op)
}

func TestStrippingMultipleTrailing(t *testing.T) {
	// "<+-": no qualifying chars → strip all trailing +/- → "<" (nchars=1).
	// yyless(1) pushes "+-" back; those then scan as '+' and '-' separately.
	assertTypes(t, `<+-`, TokenType('<'), TokenType('+'), TokenType('-'))
}

func TestOperatorNotStrippedMidway(t *testing.T) {
	// "!=-" ends with -, check '!' and '=': '!' is qualifying → NOT stripped.
	assertTypes(t, `!=-`, Op)
	assertTexts(t, `!=-`, `!=-`)
}

// ── Operator stops at comment start ──────────────────────────────────────────

func TestOperatorStopsAtLineComment(t *testing.T) {
	// "+" then "-- comment" must not be consumed into the operator.
	assertTypes(t, `+-- comment`, TokenType('+'), Comment)
}

func TestOperatorStopsAtBlockComment(t *testing.T) {
	assertTypes(t, `+/* comment */`, TokenType('+'), Comment)
}

// ── Token positions ───────────────────────────────────────────────────────────

func TestTokenPosition(t *testing.T) {
	tokens := NewScanner("  hello  world").ScanAll()
	if tokens[0].Pos != 2 {
		t.Errorf("pos[0] = %d, want 2", tokens[0].Pos)
	}
	if tokens[1].Pos != 9 {
		t.Errorf("pos[1] = %d, want 9", tokens[1].Pos)
	}
}

// ── Statement splitting ───────────────────────────────────────────────────────

func TestSplitEmpty(t *testing.T) {
	stmts := SplitStatements("")
	if len(stmts) != 0 {
		t.Fatalf("got %d statements, want 0", len(stmts))
	}
}

func TestSplitSingleStatement(t *testing.T) {
	stmts := SplitStatements("SELECT 1;")
	if len(stmts) != 1 {
		t.Fatalf("got %d statements, want 1", len(stmts))
	}
	last := stmts[0][len(stmts[0])-1]
	if last.Type != TokenType(';') {
		t.Fatalf("last token: got %v, want ';'", last.Type)
	}
}

func TestSplitMultipleStatements(t *testing.T) {
	stmts := SplitStatements("SELECT 1; SELECT 2; SELECT 3;")
	if len(stmts) != 3 {
		t.Fatalf("got %d statements, want 3", len(stmts))
	}
}

func TestSplitSemiInStringNoSplit(t *testing.T) {
	// The ';' inside the string literal must not cause a split.
	stmts := SplitStatements(`INSERT INTO t VALUES('a;b');`)
	if len(stmts) != 1 {
		t.Fatalf("got %d statements, want 1", len(stmts))
	}
}

func TestSplitSemiInDollarQuoteNoSplit(t *testing.T) {
	stmts := SplitStatements("DO $$BEGIN x:=1; END;$$;")
	if len(stmts) != 1 {
		t.Fatalf("got %d statements, want 1", len(stmts))
	}
}

func TestSplitSemiInCommentNoSplit(t *testing.T) {
	stmts := SplitStatements("SELECT 1 -- ;\n;")
	if len(stmts) != 1 {
		t.Fatalf("got %d statements, want 1", len(stmts))
	}
}

func TestSplitNoTerminatingSemicolon(t *testing.T) {
	// Trailing tokens without a ';' are returned as the last "statement".
	stmts := SplitStatements("SELECT 1; SELECT 2")
	if len(stmts) != 2 {
		t.Fatalf("got %d statements, want 2", len(stmts))
	}
}

// ── String continuation across newlines ──────────────────────────────────────

func TestStringContinuationSimple(t *testing.T) {
	// Two adjacent string literals with a newline in between → one token.
	assertTypes(t, "'hello'\n'world'", SConst)
	assertTexts(t, "'hello'\n'world'", "'hello'\n'world'")
}

func TestStringContinuationWithComment(t *testing.T) {
	// Comment in the whitespace gap is OK (scan.l {quotecontinue}).
	src := "'hello' -- trailing comment\n'world'"
	assertTypes(t, src, SConst)
}

func TestStringContinuationRequiresNewline(t *testing.T) {
	// Two strings on the SAME line → two separate tokens.
	assertTypes(t, "'hello' 'world'", SConst, SConst)
}

// ── PL/pgSQL code fragments ───────────────────────────────────────────────────

// TestAssignment verifies that a := assignment is tokenised correctly.
func TestAssignment(t *testing.T) {
	assertTypes(t, `x := 42;`, Ident, ColonEquals, IConst, TokenType(';'))
	assertTexts(t, `x := 42;`, "x", ":=", "42", ";")
}

// TestIfStatement checks a minimal IF … END IF block.
func TestIfStatement(t *testing.T) {
	src := `IF x > 0 THEN y := 1; END IF;`
	types := tokTypes(src)
	// KIf, Ident, >, IConst, KThen, Ident, ColonEquals, IConst, ';', KEnd, KIf, ';'
	want := []TokenType{
		KIf, Ident, TokenType('>'), IConst, KThen,
		Ident, ColonEquals, IConst, TokenType(';'),
		KEnd, KIf, TokenType(';'),
	}
	if len(types) != len(want) {
		t.Fatalf("src=%q\n  got  %v\n  want %v", src, types, want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Errorf("token[%d]: got %v, want %v", i, types[i], want[i])
		}
	}
}

// TestBlockLabel verifies << >> around a label name.
func TestBlockLabel(t *testing.T) {
	assertTypes(t, `<<lbl>>`, LessLess, Ident, GreaterGreater)
	assertTexts(t, `<<lbl>>`, `<<`, `lbl`, `>>`)
}

// TestTypeCast verifies :: is tokenised as Typecast.
func TestTypeCast(t *testing.T) {
	assertTypes(t, `x::int`, Ident, Typecast, Ident)
}

// TestNamedArgument verifies => is tokenised as EqualsGreater.
func TestNamedArgument(t *testing.T) {
	assertTypes(t, `foo => bar`, Ident, EqualsGreater, Ident)
}

// TestRaiseStatement verifies a typical RAISE NOTICE statement.
func TestRaiseStatement(t *testing.T) {
	src := `RAISE NOTICE 'value = %', v;`
	types := tokTypes(src)
	want := []TokenType{KRaise, KNotice, SConst, TokenType(','), Ident, TokenType(';')}
	if len(types) != len(want) {
		t.Fatalf("src=%q\n  got  %v\n  want %v", src, types, want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Errorf("token[%d]: got %v, want %v", i, types[i], want[i])
		}
	}
}

// TestFullProcedure runs a realistic PL/pgSQL fragment through the scanner and
// checks that statement splitting produces the expected number of statements.
func TestFullProcedure(t *testing.T) {
	src := `
CREATE OR REPLACE FUNCTION greet(name TEXT) RETURNS TEXT AS $$
DECLARE
    msg TEXT;
BEGIN
    msg := 'Hello, ' || name || '!';
    IF name = '' THEN
        msg := 'Hello, world!';
    END IF;
    RETURN msg;
END;
$$ LANGUAGE plpgsql;
`
	stmts := SplitStatements(src)
	if len(stmts) != 1 {
		t.Fatalf("expected 1 top-level statement, got %d", len(stmts))
	}
	// The ';' tokens inside the dollar-quoted body must NOT cause splits.
}

// TestIsKeyword exercises the Token.IsKeyword and Token.IsReservedKeyword helpers.
func TestIsKeyword(t *testing.T) {
	reserved := first("BEGIN")
	if !reserved.IsKeyword() {
		t.Error("BEGIN should be a keyword")
	}
	if !reserved.IsReservedKeyword() {
		t.Error("BEGIN should be a reserved keyword")
	}

	unreserved := first("RAISE")
	if !unreserved.IsKeyword() {
		t.Error("RAISE should be a keyword")
	}
	if unreserved.IsReservedKeyword() {
		t.Error("RAISE should NOT be a reserved keyword")
	}

	ident := first("myvar")
	if ident.IsKeyword() {
		t.Error("myvar should not be a keyword")
	}
}

// TestScanAllReturnsNoEOF confirms ScanAll never includes an EOF token.
func TestScanAllReturnsNoEOF(t *testing.T) {
	for _, tok := range NewScanner("x = 1").ScanAll() {
		if tok.Type == EOF {
			t.Fatal("ScanAll included an EOF token")
		}
	}
}

// TestTokenTypeString is a smoke test that at least demonstrates formatting.
func TestTokenTypeFormatting(t *testing.T) {
	s := fmt.Sprintf("%v", KBegin)
	if s == "" {
		t.Fatal("empty string")
	}
}
