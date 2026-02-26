/*
 * plpgsql_scanner.go
 *
 * Go implementation of the PL/pgSQL lexical scanner.
 *
 * This file is a faithful port of the two-layer scanner used in PostgreSQL:
 *
 *   Layer 1 – the core lexer:   src/backend/parser/scan.l
 *   Layer 2 – the PL/pgSQL wrapper: src/pl/plpgsql/src/pl_scanner.c
 *
 * The token stream produced here matches what the C code would produce, with
 * one deliberate simplification: we do not perform variable / datum lookup
 * (pl_scanner.c plpgsql_yylex(), lines 159–319) because that requires a live
 * catalog connection.  Tokens that would become T_DATUM or T_WORD remain Ident
 * here; that is sufficient for statement splitting and syntax highlighting.
 *
 * Porting notes
 * -------------
 * scan.l is a flex source file.  Flex performs longest-match, and when two
 * rules match the same length the earlier rule in the file wins.  In the Go
 * code the same effect is achieved by ordering the cases in Scan() to mirror
 * the rule ordering in scan.l (whitespace → comments → string prefixes →
 * quoted strings → dollar → numbers → identifiers → fixed two-char tokens →
 * operator / self).
 *
 * Usage – manual tokenisation:
 *
 *	s := plpgsqlscanner.NewScanner(src)
 *	for {
 *	    tok := s.Scan()
 *	    if tok.Type == plpgsqlscanner.EOF { break }
 *	    // use tok.Type, tok.Text, tok.Pos
 *	}
 *
 * Usage – statement splitting:
 *
 *	for _, stmt := range plpgsqlscanner.SplitStatements(src) { … }
 */
package parser

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

/*
 * TokenType is the lexical category of a token.
 *
 * Single-character punctuation tokens use their Unicode code point as the
 * TokenType value (always 1–999 for ordinary SQL punctuation); this mirrors
 * the way the core scanner returns single-char tokens directly as their ASCII
 * value (scan.l {self} rule, line 856, and {other} rule, line 1072).
 *
 * All named multi-character or keyword constants are ≥ 1000.
 */
type TokenType int

// EOF is returned when the input is fully consumed.
const EOF TokenType = 0

/*
 * Named token constants (≥ 1000).
 *
 * Token names mirror those defined in src/include/parser/scanner.h for the
 * core tokens (IDENT, FCONST, SCONST, BCONST, XCONST, Op, PARAM, ICONST /
 * integer) and in src/pl/plpgsql/src/pl_gram.h for the PL/pgSQL-specific
 * ones (LESS_LESS, GREATER_GREATER, COLON_EQUALS, DOT_DOT, TYPECAST,
 * EQUALS_GREATER, LESS_EQUALS, GREATER_EQUALS, NOT_EQUALS, K_*).
 *
 * The comparison operators EQUALS_GREATER, LESS_EQUALS, GREATER_EQUALS, and
 * NOT_EQUALS are also named in scanner.h and returned directly by the core
 * lexer from their dedicated rules (scan.l lines 829–854) as well as from
 * inside the {operator} block when stripping reduces the match to two chars
 * (scan.l lines 941–953).
 */
const (
	/*
	 * Ident – unquoted or double-quoted identifier (incl. U&"…").
	 *
	 * Produced by: scan.l {identifier} rule (line 1047), {xdstart}/{xdstop}
	 * double-quote state (lines 764, 774), {xuistart}/{dquote} U&" state
	 * (lines 769, 786), and {xufailed} (line 802) which does yyless(1) and
	 * returns the single letter as IDENT.
	 */
	Ident TokenType = 1000 + iota

	/*
	 * Param – positional parameter $1, $2, …
	 *
	 * Produced by: scan.l {param} rule (line 969).
	 * Pattern: param \${decdigit}+  (scan.l line 398).
	 * pl_scanner.c internal_yylex (line 371) re-stringifies the ival to str
	 * so it can be treated like an IDENT; we simply keep the raw text.
	 */
	Param

	/*
	 * IConst – integer literal.
	 *
	 * Produced by: scan.l {decinteger} (line 985), {hexinteger} (line 989),
	 * {octinteger} (line 993), {bininteger} (line 997).
	 *
	 * Patterns (scan.l lines 382–385):
	 *   decinteger  {decdigit}(_?{decdigit})*
	 *   hexinteger  0[xX](_?{hexdigit})+
	 *   octinteger  0[oO](_?{octdigit})+
	 *   bininteger  0[bB](_?{bindigit})+
	 *
	 * The underscore separator was introduced in PostgreSQL 14 (commit
	 * 1755e3f).  Exponent notation is not allowed in integer literals.
	 */
	IConst

	/*
	 * FConst – floating-point literal.
	 *
	 * Produced by: scan.l {numeric} rule (line 1013) and {real} rule
	 * (line 1024).
	 *
	 * Patterns (scan.l lines 391–394):
	 *   numeric    ({decinteger}\.{decinteger}?)|(\.{decinteger})
	 *   numericfail {decinteger}\.\.       ← not a float; see number()
	 *   real       ({decinteger}|{numeric})[Ee][-+]?{decinteger}
	 *
	 * Underscores are allowed in the mantissa digits but NOT in the exponent
	 * (the exponent uses bare {decinteger} without the _? pattern).
	 */
	FConst

	/*
	 * SConst – string literal in any quoting style.
	 *
	 * Covers: plain '' (scan.l {xqstart} rule, line 541), E'' escape strings
	 * ({xestart}, line 547), U&'…' Unicode strings ({xusstart}, line 553),
	 * dollar-quoted strings ({dolqdelim}, line 719), and B''/X'' after they
	 * are returned from the xqs lookahead state (scan.l lines 592–597).
	 *
	 * Adjacent quoted strings separated by whitespace containing at least one
	 * newline are implicitly concatenated into a single token.  This is
	 * implemented in scan.l via the intermediate xqs state and the
	 * {quotecontinue} pattern (lines 226, 571–578).
	 */
	SConst

	/*
	 * BConst – bit-string literal B'1010'.
	 *
	 * Produced by: scan.l {xbstart} rule (line 483), returned via the xqs
	 * lookahead state (line 592).
	 * Pattern: xbstart [bB]{quote}  (scan.l line 246).
	 */
	BConst

	/*
	 * XConst – hexadecimal string literal X'DEAD'.
	 *
	 * Produced by: scan.l {xhstart} rule (line 501), returned via the xqs
	 * lookahead state (line 595).
	 * Pattern: xhstart [xX]{quote}  (scan.l line 250).
	 */
	XConst

	/*
	 * Comment – a line comment (-- …) or a block comment (/* … * /).
	 *
	 * Block comments nest in PostgreSQL (unlike standard SQL).  The nesting
	 * is implemented in the <xc> state (scan.l lines 452–481) via the
	 * xcdepth counter: {xcstart} increments it (line 453–456) and {xcstop}
	 * decrements it (lines 459–464), returning to INITIAL only when the depth
	 * reaches zero.
	 *
	 * Line comments: pattern comment ("--"{non_newline}*)  (scan.l line 209).
	 * Block-comment start: xcstart \/\*{op_chars}*  (scan.l line 324).
	 */
	Comment

	/*
	 * Op – an operator string that has no dedicated token type.
	 *
	 * Produced by: scan.l {operator} rule (line 861), specifically the
	 * fall-through path at line 966 ("return Op") after the comment-boundary
	 * detection and the trailing-+/- stripping have been applied and the
	 * result does not match any of the special two-char tokens.
	 */
	Op

	/*
	 * LessLess – the << token (block-label open, bit-shift, etc.).
	 * GreaterGreater – the >> token (block-label close, bit-shift, etc.).
	 *
	 * The core lexer (scan.l) returns these as generic Op tokens because <<
	 * and >> match the {operator} rule and have no dedicated pattern.
	 * pl_scanner.c internal_yylex (lines 361–368) promotes them:
	 *
	 *   if (strcmp(auxdata->lval.str, "<<") == 0)  token = LESS_LESS;
	 *   else if (strcmp(auxdata->lval.str, ">>") == 0)  token = GREATER_GREATER;
	 *
	 * We replicate that promotion inside operator() rather than in a
	 * separate post-processing step.
	 */
	LessLess
	GreaterGreater

	/*
	 * ColonEquals – the := assignment operator.
	 *
	 * Dedicated pattern in scan.l: colon_equals ":=" (line 336).
	 * Returned by the {colon_equals} rule (line 824).
	 * Note: ':' and '=' are in the "self" set but NOT in op_chars, so ::
	 * and := are never consumed by the {operator} rule.
	 */
	ColonEquals

	/*
	 * DotDot – the .. range operator.
	 *
	 * Dedicated pattern: dot_dot \.\.  (scan.l line 335).
	 * Returned by the {dot_dot} rule (line 819).
	 * Note: '.' is not in op_chars, so .. is never consumed by {operator}.
	 * The {numericfail} rule (line 1018) uses yyless(yyleng-2) to prevent
	 * "1..2" from being scanned as a float; see number() below.
	 */
	DotDot

	/*
	 * Typecast – the :: type-cast operator.
	 *
	 * Dedicated pattern: typecast "::"  (scan.l line 334).
	 * Returned by the {typecast} rule (line 814).
	 */
	Typecast

	/*
	 * EqualsGreater – the => named-argument operator.
	 *
	 * Dedicated pattern: equals_greater "=>"  (scan.l line 346).
	 * Returned by the {equals_greater} rule (line 829) AND also by the
	 * {operator} block (line 943) when stripping reduces the match to "=>".
	 *
	 * scan.l comment (lines 338–344) warns that these two-char tokens also
	 * match {operator}, so the operator block must return the right token
	 * when stripping leaves exactly those two characters.
	 */
	EqualsGreater

	/*
	 * LessEquals – <=
	 * GreaterEquals – >=
	 * NotEquals – <> or !=
	 *
	 * Dedicated patterns (scan.l lines 347–350):
	 *   less_equals    "<="
	 *   greater_equals ">="
	 *   less_greater   "<>"
	 *   not_equals     "!="
	 *
	 * Returned by their own rules (lines 834–854) AND by the {operator}
	 * block (lines 945–952) when trailing-+/- stripping produces a 2-char
	 * result matching one of these patterns.
	 */
	LessEquals
	GreaterEquals
	NotEquals

	/*
	 * Reserved PL/pgSQL keywords.
	 *
	 * These are passed to the core scanner via the ReservedPLKeywords list
	 * (pl_scanner.c lines 66–68; built from pl_reserved_kwlist.h).  The core
	 * scanner recognises them before any identifier or variable name lookup,
	 * so they can never be used as variable names.
	 *
	 * Current reserved list (pl_reserved_kwlist.h): ALL BEGIN BY CASE
	 * DECLARE ELSE END FOR FOREACH FROM IF IN INTO LOOP NOT NULL OR THEN
	 * TO USING WHEN WHILE.
	 *
	 * See pl_scanner.c lines 28–57 for the design rationale on which words
	 * are reserved vs. unreserved.
	 */
	KAll
	KBegin
	KBy
	KCase
	KDeclare
	KElse
	KEnd
	KFor
	KForeach
	KFrom
	KIf
	KIn
	KInto
	KLoop
	KNot
	KNull
	KOr
	KThen
	KTo
	KUsing
	KWhen
	KWhile

	/*
	 * Unreserved PL/pgSQL keywords.
	 *
	 * These are NOT passed to the core scanner.  They are checked for in
	 * pl_scanner.c plpgsql_yylex() (lines 247–254) via ScanKeywordLookup
	 * against the UnreservedPLKeywords list (built from
	 * pl_unreserved_kwlist.h), but only after failing identifier/variable
	 * lookup and only for non-quoted words.
	 *
	 * We perform the same lookup unconditionally in ident(), which is safe
	 * for our read-only use-case (syntax highlighting, statement splitting).
	 *
	 * Note: both "elsif" and "elseif" map to K_ELSIF in pl_unreserved_kwlist.h;
	 * we map both spellings to KElsif here.
	 */
	KAbsolute
	KAlias
	KAnd
	KArray
	KAssert
	KBackward
	KCall
	KChain
	KClose
	KCollate
	KColumn
	KColumnName
	KCommit
	KConstant
	KConstraint
	KConstraintName
	KContinue
	KCurrent
	KCursor
	KDatatype
	KDebug
	KDefault
	KDetail
	KDiagnostics
	KDo
	KDump
	KElsif // also "elseif" – see pl_unreserved_kwlist.h
	KErrcode
	KError
	KException
	KExecute
	KExit
	KFetch
	KFirst
	KForward
	KGet
	KHint
	KImport
	KInfo
	KInsert
	KIs
	KLast
	KLog
	KMerge
	KMessage
	KMessageText
	KMove
	KNext
	KNo
	KNotice
	KOpen
	KOption
	KPerform
	KPgContext
	KPgDatatypeName
	KPgExceptionContext
	KPgExceptionDetail
	KPgExceptionHint
	KPgRoutineOid
	KPrintStrictParams
	KPrior
	KQuery
	KRaise
	KRelative
	KReturn
	KReturnedSqlstate
	KReverse
	KRollback
	KRowCount
	KRowtype
	KSchema
	KSchemaName
	KScroll
	KSlice
	KSqlstate
	KStacked
	KStrict
	KTable
	KTableName
	KType
	KUseColumn
	KUseVariable
	KVariableConflict
	KWarning
)

// lastReserved is the last TokenType in the reserved-keyword block.
// Used by IsReservedKeyword() to distinguish reserved from unreserved.
const lastReserved = KWhile

/*
 * keywords maps the lowercase canonical spelling of every PL/pgSQL keyword
 * to its TokenType.
 *
 * Reserved keywords are sourced from pl_reserved_kwlist.h; unreserved from
 * pl_unreserved_kwlist.h.  Both files use the PG_KEYWORD(kwname, value) macro
 * pattern; pl_scanner.c lines 59–74 show how they are consumed.
 *
 * Lookup is case-insensitive: ident() calls strings.ToLower before probing
 * this map, mirroring the core scanner's downcase_truncate_identifier()
 * (scan.l line 1067) and ScanKeywordLookup() which operates on the
 * already-lowercased word.
 */
var keywords = map[string]TokenType{
	// ── Reserved (pl_reserved_kwlist.h) ──────────────────────────────────────
	"all": KAll, "begin": KBegin, "by": KBy, "case": KCase,
	"declare": KDeclare, "else": KElse, "end": KEnd,
	"for": KFor, "foreach": KForeach, "from": KFrom,
	"if": KIf, "in": KIn, "into": KInto, "loop": KLoop,
	"not": KNot, "null": KNull, "or": KOr, "then": KThen,
	"to": KTo, "using": KUsing, "when": KWhen, "while": KWhile,
	// ── Unreserved (pl_unreserved_kwlist.h) ──────────────────────────────────
	"absolute": KAbsolute, "alias": KAlias, "and": KAnd,
	"array": KArray, "assert": KAssert, "backward": KBackward,
	"call": KCall, "chain": KChain, "close": KClose,
	"collate": KCollate, "column": KColumn, "column_name": KColumnName,
	"commit": KCommit, "constant": KConstant, "constraint": KConstraint,
	"constraint_name": KConstraintName, "continue": KContinue,
	"current": KCurrent, "cursor": KCursor, "datatype": KDatatype,
	"debug": KDebug, "default": KDefault, "detail": KDetail,
	"diagnostics": KDiagnostics, "do": KDo, "dump": KDump,
	// Both spellings share one token (pl_unreserved_kwlist.h, K_ELSIF entry).
	"elseif": KElsif, "elsif": KElsif,
	"errcode": KErrcode, "error": KError, "exception": KException,
	"execute": KExecute, "exit": KExit, "fetch": KFetch,
	"first": KFirst, "forward": KForward, "get": KGet,
	"hint": KHint, "import": KImport, "info": KInfo,
	"insert": KInsert, "is": KIs, "last": KLast, "log": KLog,
	"merge": KMerge, "message": KMessage, "message_text": KMessageText,
	"move": KMove, "next": KNext, "no": KNo, "notice": KNotice,
	"open": KOpen, "option": KOption, "perform": KPerform,
	"pg_context": KPgContext, "pg_datatype_name": KPgDatatypeName,
	"pg_exception_context": KPgExceptionContext,
	"pg_exception_detail":  KPgExceptionDetail,
	"pg_exception_hint":    KPgExceptionHint,
	"pg_routine_oid":       KPgRoutineOid,
	"print_strict_params":  KPrintStrictParams,
	"prior":                KPrior, "query": KQuery, "raise": KRaise,
	"relative": KRelative, "return": KReturn,
	"returned_sqlstate": KReturnedSqlstate, "reverse": KReverse,
	"rollback": KRollback, "row_count": KRowCount, "rowtype": KRowtype,
	"schema": KSchema, "schema_name": KSchemaName, "scroll": KScroll,
	"slice": KSlice, "sqlstate": KSqlstate, "stacked": KStacked,
	"strict": KStrict, "table": KTable, "table_name": KTableName,
	"type": KType, "use_column": KUseColumn, "use_variable": KUseVariable,
	"variable_conflict": KVariableConflict, "warning": KWarning,
}

// Token is a single lexical token from PL/pgSQL source text.
type Token struct {
	Type TokenType // Lexical category.
	Text string    // Raw source bytes that form this token.
	Pos  int       // Byte offset of the first character (0-based).
}

// IsKeyword reports whether t is any PL/pgSQL keyword (reserved or unreserved).
func (t Token) IsKeyword() bool { return t.Type >= KAll }

// IsReservedKeyword reports whether t is a reserved PL/pgSQL keyword
// (one that the core scanner recognises before identifier lookup;
// see pl_scanner.c lines 33–36 and the comment at lines 28–57).
func (t Token) IsReservedKeyword() bool { return t.Type >= KAll && t.Type <= lastReserved }

/*
 * Scanner tokenizes PL/pgSQL source text one token at a time.
 * Whitespace between tokens is silently consumed (scan.l line 439).
 * All byte offsets (Pos) are 0-based indices into the original source string.
 */
type Scanner struct {
	src string
	pos int
}

// NewScanner returns a Scanner that reads from src.
func NewScanner(src string) *Scanner { return &Scanner{src: src} }

// Pos returns the byte offset of the next character to be read.
func (s *Scanner) Pos() int { return s.pos }

/*
 * Scan returns the next token, skipping any leading whitespace.
 * Returns Token{Type: EOF} when the input is exhausted.
 *
 * The case ordering below mirrors the flex rule ordering in scan.l.  Flex
 * uses longest-match with ties broken by rule order (earlier rule wins), so
 * the order here is significant for correctness:
 *
 *  1. Whitespace (scan.l line 439) – skipped silently before entering switch.
 *  2. Block-comment start /* (xcstart, scan.l line 443) – must precede the
 *     {operator} rule because xcstart is \/\*{op_chars}* and the leading /
 *     is in op_chars.  (scan.l comment at line 433.)
 *  3. String-literal prefixes E'  B'  X' (xestart/xbstart/xhstart).
 *  4. N' national-character prefix (xnstart, scan.l line 515) – NOTE: the
 *     core scanner does yyless(1) and returns n/N as IDENT (line 523).  We
 *     therefore do NOT handle N' here; instead n/N falls through to ident().
 *  5. U&' and U&" Unicode literals (xusstart/xuistart, lines 553/769).
 *     If U& is not followed by ' or ", the {xufailed} rule (line 802) fires
 *     yyless(1) and returns IDENT.
 *  6. Plain '…' string (xqstart, line 541).
 *  7. Double-quoted identifier "…" (xdstart, line 764).
 *  8. Dollar: positional parameter $N or dollar-quoted string $tag$…$tag$.
 *  9. Numeric literals (decinteger/numeric/real, lines 985–1032).
 * 10. Identifiers and keywords (identifier rule, line 1047).
 * 11. Fixed two-char tokens :: := .. whose constituents are NOT in op_chars
 *     (typecast/colon_equals/dot_dot, scan.l lines 814–822).
 * 12. All remaining characters: op_chars → operator(), everything else →
 *     single-char token (scan.l {self} line 856 / {other} line 1072).
 */
func (s *Scanner) Scan() Token {
	s.skipWS()
	if s.pos >= len(s.src) {
		return Token{Type: EOF, Pos: s.pos}
	}
	start := s.pos
	ch := s.src[s.pos]

	switch {
	/*
	 * Line comment: -- to end of line.
	 * Pattern: comment ("--"{non_newline}*)  scan.l line 209.
	 * The {whitespace} rule (line 439) keeps these from reaching {operator}.
	 */
	case ch == '-' && s.peek(1) == '-':
		return s.lineComment(start)

	/*
	 * Block comment: /* … * / with nesting.
	 * Pattern: xcstart \/\*{op_chars}*  scan.l line 324.
	 * scan.l note (line 433): "xcstart must appear before operator".
	 */
	case ch == '/' && s.peek(1) == '*':
		return s.blockComment(start)

	/*
	 * E'…' escape string.  Pattern: xestart [eE]{quote}  scan.l line 257.
	 * Rule: {xestart} at line 547.
	 */
	case (ch == 'e' || ch == 'E') && s.peek(1) == '\'':
		return s.escapeString(start)

	/*
	 * B'…' bit-string.  Pattern: xbstart [bB]{quote}  scan.l line 246.
	 * Rule: {xbstart} at line 483.
	 */
	case (ch == 'b' || ch == 'B') && s.peek(1) == '\'':
		return s.prefixString(start, BConst)

	/*
	 * X'…' hex-string.  Pattern: xhstart [xX]{quote}  scan.l line 250.
	 * Rule: {xhstart} at line 501.
	 */
	case (ch == 'x' || ch == 'X') && s.peek(1) == '\'':
		return s.prefixString(start, XConst)

	/*
	 * U&'…' Unicode string.
	 * Pattern: xusstart [uU]&{quote}  scan.l line 300.  Rule: line 553.
	 */
	case (ch == 'u' || ch == 'U') && s.peek(1) == '&' && s.peek(2) == '\'':
		return s.unicodeString(start)

	/*
	 * U&"…" Unicode quoted identifier.
	 * Pattern: xuistart [uU]&{dquote}  scan.l line 297.  Rule: line 769.
	 */
	case (ch == 'u' || ch == 'U') && s.peek(1) == '&' && s.peek(2) == '"':
		return s.unicodeQuotedIdent(start)

	/*
	 * NOTE: N'…' (national character, xnstart [nN]{quote} scan.l line 254)
	 * is intentionally NOT handled here.  The core scanner's {xnstart} rule
	 * (line 515) calls yyless(1) to consume only the 'n'/'N', then looks up
	 * "nchar" in the keyword list.  Since "nchar" is not a PL/pgSQL keyword
	 * it falls through to returning IDENT for the single letter 'n'/'N'.
	 * The following quote is then a plain xqstart '…' token.
	 * Result: N'text' → two tokens: Ident("N") + SConst("'text'").
	 * We replicate this by letting n/N reach the ident() path below.
	 */

	/*
	 * Plain '…' string.
	 * Pattern: xqstart {quote}  scan.l line 541 ({xqstart} → xq state).
	 */
	case ch == '\'':
		return s.quotedString(start)

	/*
	 * Double-quoted identifier "…".
	 * Pattern: xdstart {dquote}  scan.l line 291.  Rule: line 764.
	 */
	case ch == '"':
		return s.quotedIdent(start)

	/*
	 * Dollar sign: positional parameter $N or dollar-quoted string $tag$…$tag$.
	 * See dollar() for the dispatch logic (scan.l {param} line 969,
	 * {dolqdelim} line 719, {dolqfailed} line 725).
	 */
	case ch == '$':
		return s.dollar(start)

	/*
	 * Decimal integer or floating-point number.
	 * Patterns: {decinteger} (line 985), {numeric} (line 1013), {real} (line 1024).
	 */
	case ch >= '0' && ch <= '9':
		return s.number(start)

	/*
	 * A leading dot followed by a digit starts a fractional float (.5, .25).
	 * This corresponds to the ({numeric}) alternative "\.{decinteger}"
	 * in scan.l line 391.
	 */
	case ch == '.' && s.peek(1) >= '0' && s.peek(1) <= '9':
		return s.number(start)

	/*
	 * Unquoted identifier or keyword.
	 * Pattern: identifier {ident_start}{ident_cont}*  scan.l line 331.
	 * ident_start [A-Za-z\200-\377_]  (scan.l line 328).
	 * The ch >= 0x80 branch catches the high (non-ASCII) byte range.
	 */
	case isIdentStart(ch) || ch >= 0x80:
		return s.ident(start)

	/*
	 * Fixed two-char tokens whose characters are NOT in op_chars, so they
	 * are safe to dispatch before the operator() catch-all.
	 *
	 * Patterns and rules:
	 *   typecast     "::"  scan.l line 334, rule line 814
	 *   colon_equals ":="  scan.l line 336, rule line 824
	 *   dot_dot      "\.\."  scan.l line 335, rule line 819
	 *
	 * ':' and '.' are in the "self" set (scan.l line 362) but NOT in
	 * op_chars (line 363), so they are never consumed by {operator}.
	 */
	case ch == ':' && s.peek(1) == ':':
		s.pos += 2
		return Token{Type: Typecast, Text: "::", Pos: start}
	case ch == ':' && s.peek(1) == '=':
		s.pos += 2
		return Token{Type: ColonEquals, Text: ":=", Pos: start}
	case ch == '.' && s.peek(1) == '.':
		s.pos += 2
		return Token{Type: DotDot, Text: "..", Pos: start}

	/*
	 * Operator characters and single-char self tokens.
	 *
	 * op_chars [\~\!\@\#\^\&\|\`\?\+\-\*\/\%\<\>\=]  scan.l line 363.
	 * self     [,()\[\].;\:\+\-\*\/\%\^\<\>\=]        scan.l line 362.
	 *
	 * op_chars chars go through operator(), which implements the full
	 * {operator} rule logic (scan.l line 861).
	 * Everything else is returned as a single-char token, mirroring the
	 * {self} rule (line 856) and the {other} rule (line 1072).
	 */
	default:
		if isOpChar(ch) {
			return s.operator(start)
		}
		/* High byte that wasn't caught by the ident_start branch (rare). */
		if ch >= 0x80 {
			_, size := utf8.DecodeRuneInString(s.src[s.pos:])
			if size < 1 {
				size = 1
			}
			s.pos += size
			return Token{Type: TokenType(s.src[start]), Text: s.src[start:s.pos], Pos: start}
		}
		/* Single-character token: {self} or {other} (scan.l lines 856/1072). */
		s.pos++
		return Token{Type: TokenType(ch), Text: string(ch), Pos: start}
	}
}

// ScanAll tokenises the entire source and returns every token (no EOF entry).
func (s *Scanner) ScanAll() []Token {
	var toks []Token
	for {
		t := s.Scan()
		if t.Type == EOF {
			break
		}
		toks = append(toks, t)
	}
	return toks
}

/*
 * SplitStatements tokenises src and groups tokens into per-statement slices.
 * Each slice ends with the ';' token that terminates it.
 * Trailing tokens that have no closing ';' are collected into a final slice.
 *
 * This is the primary use-case for the scanner: splitting a PL/pgSQL function
 * body into individual statements without misidentifying ';' characters that
 * appear inside string literals, dollar-quoted blocks, or comments.
 */
func SplitStatements(src string) [][]Token {
	s := NewScanner(src)
	var stmts [][]Token
	var cur []Token
	for {
		tok := s.Scan()
		if tok.Type == EOF {
			if len(cur) > 0 {
				stmts = append(stmts, cur)
			}
			break
		}
		cur = append(cur, tok)
		if tok.Type == TokenType(';') {
			stmts = append(stmts, cur)
			cur = nil
		}
	}
	return stmts
}

// ---------------------------------------------------------------------------
// Internal scanner methods
// ---------------------------------------------------------------------------

// peek returns the byte at position s.pos+offset, or 0 if out of bounds.
func (s *Scanner) peek(offset int) byte {
	if i := s.pos + offset; i < len(s.src) {
		return s.src[i]
	}
	return 0
}

/*
 * skipWS skips whitespace characters.
 *
 * Corresponds to the {whitespace} rule (scan.l line 439):
 *   whitespace ({space}+|{comment}|{xwhitespace})
 * where space is [ \t\n\r\f\v].  We do not "return" from whitespace;
 * scan.l also discards it silently.
 */
func (s *Scanner) skipWS() {
	for s.pos < len(s.src) {
		switch s.src[s.pos] {
		case ' ', '\t', '\n', '\r', '\f', '\v':
			s.pos++
		default:
			return
		}
	}
}

/*
 * lineComment consumes from "--" to end of line (newline not consumed).
 *
 * Pattern: comment ("--"{non_newline}*)  scan.l line 209.
 * non_newline [^\n\r]  (scan.l line 207).
 * The newline itself is left in the stream so that the quotecontinue
 * check in continueString() can see it.
 */
func (s *Scanner) lineComment(start int) Token {
	for s.pos < len(s.src) && s.src[s.pos] != '\n' && s.src[s.pos] != '\r' {
		s.pos++
	}
	return Token{Type: Comment, Text: s.src[start:s.pos], Pos: start}
}

/*
 * blockComment consumes / * … * / with support for PostgreSQL nested nesting.
 *
 * scan.l implements nesting via the <xc> exclusive state (lines 452–481):
 *   {xcstart}   increments xcdepth counter (line 453–456)
 *   {xcstop}    decrements xcdepth; returns to INITIAL when depth == 0
 *               (lines 459–464)
 *   {xcinside}  ignored content (line 466)
 *   {op_chars}  also ignored (line 470) — required because xcstart contains
 *               op_chars after the initial "/*" (xcstart pattern line 324)
 *
 * Standard SQL does not support nested block comments; this is a PostgreSQL
 * extension (see PostgreSQL docs §4.1.2).
 */
func (s *Scanner) blockComment(start int) Token {
	s.pos += 2 /* consume opening "/*" */
	for depth := 1; s.pos < len(s.src) && depth > 0; {
		switch {
		case s.src[s.pos] == '/' && s.peek(1) == '*':
			depth++ /* xcstart inside <xc>: scan.l line 453 */
			s.pos += 2
		case s.src[s.pos] == '*' && s.peek(1) == '/':
			depth-- /* xcstop: scan.l line 459 */
			s.pos += 2
		default:
			s.pos++
		}
	}
	return Token{Type: Comment, Text: s.src[start:s.pos], Pos: start}
}

/*
 * quotedString consumes a '…' single-quoted string literal.
 *
 * Corresponds to the <xq> state in scan.l (entered by {xqstart} at line 541).
 * Key rules within <xq>:
 *   - {xqinside}  regular characters accumulate (line ~620+).
 *   - {quote}     ends the xq state; transitions to xqs for lookahead
 *                 (lines 559–570).
 *
 * A doubled quote '' is the standard SQL escape for a literal single-quote
 * (ISO SQL §5.3).  We detect this by checking whether the character after the
 * closing quote is another quote, and if so continuing to scan.
 *
 * Implicit concatenation: adjacent string literals separated only by
 * whitespace that contains at least one newline are joined into a single token.
 * This is scan.l's {quotecontinue} mechanism: after the end-quote the scanner
 * enters the xqs lookahead state (line 568), and if {quotecontinue} matches
 * (line 571) it returns to the in-string state.  We implement the same logic
 * in continueString() below.
 */
func (s *Scanner) quotedString(start int) Token {
	s.pos++ /* consume opening ' */
	for s.pos < len(s.src) {
		if s.src[s.pos] != '\'' {
			s.pos++
			continue
		}
		s.pos++ /* consume closing quote or first of '' */
		if s.pos < len(s.src) && s.src[s.pos] == '\'' {
			s.pos++ /* '' inside xq: escaped single-quote; keep scanning */
			continue
		}
		/* xqs lookahead: try {quotecontinue} (scan.l line 571) */
		if s.continueString() {
			continue
		}
		break
	}
	return Token{Type: SConst, Text: s.src[start:s.pos], Pos: start}
}

/*
 * escapeString consumes an E'…' escape string literal.
 *
 * Corresponds to the <xe> state in scan.l (entered by {xestart} at line 547).
 * Pattern: xestart [eE]{quote}  (scan.l line 257).
 *
 * Inside <xe>, backslash sequences are recognised: a backslash followed by
 * any character is a two-byte escape sequence.  The scan.l <xe> state has
 * individual rules for each escape form, but for our purposes (we return the
 * raw source text) it is sufficient to skip two bytes on any backslash.
 */
func (s *Scanner) escapeString(start int) Token {
	s.pos += 2 /* consume E' */
	for s.pos < len(s.src) {
		ch := s.src[s.pos]
		if ch == '\\' {
			s.pos += 2 /* skip backslash + escaped char (scan.l <xe> backslash rules) */
			continue
		}
		if ch != '\'' {
			s.pos++
			continue
		}
		s.pos++ /* consume closing ' */
		if s.pos < len(s.src) && s.src[s.pos] == '\'' {
			s.pos++ /* '' inside xe: escaped quote */
			continue
		}
		/* xqs lookahead */
		if s.continueString() {
			continue
		}
		break
	}
	return Token{Type: SConst, Text: s.src[start:s.pos], Pos: start}
}

/*
 * prefixString consumes a B'…' bit-string or X'…' hex-string literal.
 *
 * Corresponds to the <xb> and <xh> states in scan.l.
 * Patterns: xbstart [bB]{quote} (line 246), xhstart [xX]{quote} (line 250).
 * Rules: {xbstart} line 483, {xhstart} line 501.
 *
 * The scan.l xqs lookahead state also handles B''/X'' continuation
 * (cases xb and xh in the switch at lines 592–597 of scan.l).
 */
func (s *Scanner) prefixString(start int, typ TokenType) Token {
	s.pos += 2 /* consume prefix letter + opening ' */
	for s.pos < len(s.src) {
		if s.src[s.pos] == '\'' {
			s.pos++
			if s.pos < len(s.src) && s.src[s.pos] == '\'' {
				s.pos++ /* '' → escaped quote */
				continue
			}
			/* xqs lookahead (scan.l lines 559–578) */
			if s.continueString() {
				continue
			}
			break
		}
		s.pos++
	}
	return Token{Type: typ, Text: s.src[start:s.pos], Pos: start}
}

/*
 * unicodeString consumes a U&'…' Unicode string literal.
 *
 * Corresponds to the <xus> state in scan.l.
 * Pattern: xusstart [uU]&{quote}  (scan.l line 300).  Rule: line 553.
 *
 * Inside <xus>, backslash introduces a Unicode escape sequence (\XXXX or
 * \+XXXXXX); we skip two bytes on any backslash, which is sufficient for
 * preserving the raw source.
 *
 * Note: U& not followed by ' or " is handled by the {xufailed} rule
 * (scan.l line 802) which calls yyless(1) and returns IDENT for the 'u'/'U'.
 * We replicate that in Scan(): the U&' case above only fires when peek(2)=='\''.
 */
func (s *Scanner) unicodeString(start int) Token {
	s.pos += 3 /* consume U&' */
	for s.pos < len(s.src) {
		ch := s.src[s.pos]
		if ch == '\\' {
			s.pos += 2 /* Unicode escape or backslash-backslash */
			continue
		}
		if ch == '\'' {
			s.pos++
			if s.pos < len(s.src) && s.src[s.pos] == '\'' {
				s.pos++ /* '' inside xus */
				continue
			}
			break
		}
		s.pos++
	}
	return Token{Type: SConst, Text: s.src[start:s.pos], Pos: start}
}

/*
 * unicodeQuotedIdent consumes a U&"…" Unicode quoted identifier.
 *
 * Corresponds to the <xui> state in scan.l.
 * Pattern: xuistart [uU]&{dquote}  (scan.l line 297).  Rule: line 769.
 * Return token: UIDENT (line 792) – we map this to Ident.
 *
 * "" inside the identifier is an escaped double-quote ({xddouble}, line 294).
 * Backslash introduces a Unicode escape (\XXXX or \+XXXXXX).
 */
func (s *Scanner) unicodeQuotedIdent(start int) Token {
	s.pos += 3 /* consume U&" */
	for s.pos < len(s.src) {
		ch := s.src[s.pos]
		if ch == '\\' {
			s.pos += 2
			continue
		}
		if ch == '"' {
			s.pos++
			if s.pos < len(s.src) && s.src[s.pos] == '"' {
				s.pos++ /* "" → escaped double-quote (xddouble) */
				continue
			}
			break
		}
		s.pos++
	}
	return Token{Type: Ident, Text: s.src[start:s.pos], Pos: start}
}

/*
 * quotedIdent consumes a "…" double-quoted identifier.
 *
 * Corresponds to the <xd> state in scan.l.
 * Pattern: xdstart {dquote}  (scan.l line 291).  Rule: line 764.
 * Return token: IDENT (line 784).
 *
 * "" (xddouble, pattern line 293) inside the identifier is an escaped
 * double-quote.  scan.l truncates identifiers ≥ NAMEDATALEN; we preserve
 * the raw text without truncation.
 */
func (s *Scanner) quotedIdent(start int) Token {
	s.pos++ /* consume opening " */
	for s.pos < len(s.src) {
		if s.src[s.pos] != '"' {
			s.pos++
			continue
		}
		s.pos++
		if s.pos < len(s.src) && s.src[s.pos] == '"' {
			s.pos++ /* "" → xddouble (scan.l line 294) */
			continue
		}
		break
	}
	return Token{Type: Ident, Text: s.src[start:s.pos], Pos: start}
}

/*
 * dollar dispatches between a positional parameter ($1) and a dollar-quoted
 * string ($tag$…$tag$, including the empty-tag $$…$$ form).
 *
 * scan.l handles this via three patterns (lines 283–284, 398):
 *
 *   param       \${decdigit}+                        → PARAM (line 969)
 *   dolqdelim   \$({dolq_start}{dolq_cont}*)?\$      → enter xdolq state (line 719)
 *   dolqfailed  \${dolq_start}{dolq_cont}*           → yyless(1), return '$' (line 725)
 *
 * If none of those patterns match (bare '$' not followed by digits or a
 * valid tag start), scan.l's {other} rule (line 1072) returns the '$'.
 * We replicate that by returning TokenType('$').
 */
func (s *Scanner) dollar(start int) Token {
	s.pos++ /* consume leading $ */

	/*
	 * $N positional parameter.
	 * Pattern: param \${decdigit}+  (scan.l line 398).
	 * Note: no underscore separators in parameter numbers; the {param}
	 * pattern uses bare {decdigit}+, not {decinteger}.
	 */
	if s.pos < len(s.src) && s.src[s.pos] >= '0' && s.src[s.pos] <= '9' {
		for s.pos < len(s.src) && s.src[s.pos] >= '0' && s.src[s.pos] <= '9' {
			s.pos++
		}
		return Token{Type: Param, Text: s.src[start:s.pos], Pos: start}
	}

	/*
	 * Dollar-quoted string $tag$…$tag$ or $$…$$.
	 *
	 * dolq_start [A-Za-z\200-\377_]    scan.l line 281 (no digits, no '$')
	 * dolq_cont  [A-Za-z\200-\377_0-9] scan.l line 282
	 * dolqdelim  \$({dolq_start}{dolq_cont}*)?\$  line 283
	 *
	 * The high-byte range (\200-\377) in dolq_start/dolq_cont allows UTF-8
	 * multi-byte sequences in tag names.  isDolqStart/isDolqCont implement
	 * this by testing ch >= 0x80.
	 *
	 * If the opening $…$ delimiter cannot be fully formed (dolqfailed rule,
	 * line 725), scan.l does yyless(1) and returns '$'.  We replicate this
	 * by resetting s.pos to start+1 before returning.
	 */
	if s.pos < len(s.src) && (s.src[s.pos] == '$' || isDolqStart(s.src[s.pos])) {
		tagStart := s.pos
		if s.src[s.pos] != '$' {
			s.pos++
			for s.pos < len(s.src) && isDolqCont(s.src[s.pos]) {
				s.pos++
			}
		}
		if s.pos >= len(s.src) || s.src[s.pos] != '$' {
			/* dolqfailed: scan.l line 725 – yyless(1), return '$' */
			s.pos = start + 1
			return Token{Type: TokenType('$'), Text: "$", Pos: start}
		}
		s.pos++ /* consume closing $ of opening delimiter */
		tag := s.src[tagStart : s.pos-1]
		closing := "$" + tag + "$"

		idx := strings.Index(s.src[s.pos:], closing)
		if idx < 0 {
			s.pos = len(s.src) /* unterminated: consume to end of input */
		} else {
			s.pos += idx + len(closing)
		}
		return Token{Type: SConst, Text: s.src[start:s.pos], Pos: start}
	}

	/* Bare '$' not matched by any of the above (scan.l {other} line 1072). */
	return Token{Type: TokenType('$'), Text: "$", Pos: start}
}

/*
 * number scans an integer or floating-point numeric literal.
 *
 * scan.l patterns (lines 377–394):
 *
 *   decdigit    [0-9]                                        line 377
 *   decinteger  {decdigit}(_?{decdigit})*                    line 382
 *   hexinteger  0[xX](_?{hexdigit})+                         line 383
 *   octinteger  0[oO](_?{octdigit})+                         line 384
 *   bininteger  0[bB](_?{bindigit})+                         line 385
 *   numeric     ({decinteger}\.{decinteger}?)|(\.{decinteger}) line 391
 *   numericfail {decinteger}\.\.                              line 392
 *   real        ({decinteger}|{numeric})[Ee][-+]?{decinteger} line 394
 *
 * Underscore digit-group separators were introduced in PostgreSQL 14
 * (commit 1755e3f).  They are allowed in all integer forms and in the
 * mantissa of a float, but NOT in the exponent (the exponent part of
 * {real} uses bare {decinteger}, not the _?-decorated form).
 *
 * {numericfail} handles the case "1..2" which would otherwise be ambiguous
 * between a float "1." and the ".." (dot_dot) range operator.  scan.l's
 * {numericfail} action does yyless(yyleng-2) to push back the ".." (line 1020).
 * We replicate this by checking peek(1) != '.' before consuming a '.'.
 */
func (s *Scanner) number(start int) Token {
	/* Consume leading decimal digits (may be zero for the ".nnn" form). */
	for s.pos < len(s.src) && (isDecDigit(s.src[s.pos]) || s.src[s.pos] == '_') {
		s.pos++
	}

	/*
	 * Prefixed integer: 0x…, 0o…, 0b…
	 * Only valid when exactly one leading '0' was consumed (no underscores
	 * before the prefix letter, matching the scan.l patterns literally).
	 */
	if s.pos == start+1 && s.src[start] == '0' && s.pos < len(s.src) {
		switch s.src[s.pos] {
		case 'x', 'X': /* hexinteger: 0[xX](_?{hexdigit})+ (scan.l line 383) */
			s.pos++
			for s.pos < len(s.src) && (isHexDigit(s.src[s.pos]) || s.src[s.pos] == '_') {
				s.pos++
			}
			return Token{Type: IConst, Text: s.src[start:s.pos], Pos: start}
		case 'o', 'O': /* octinteger: 0[oO](_?{octdigit})+ (scan.l line 384) */
			s.pos++
			for s.pos < len(s.src) && (isOctDigit(s.src[s.pos]) || s.src[s.pos] == '_') {
				s.pos++
			}
			return Token{Type: IConst, Text: s.src[start:s.pos], Pos: start}
		case 'b', 'B': /* bininteger: 0[bB](_?{bindigit})+ (scan.l line 385) */
			s.pos++
			for s.pos < len(s.src) && (s.src[s.pos] == '0' || s.src[s.pos] == '1' || s.src[s.pos] == '_') {
				s.pos++
			}
			return Token{Type: IConst, Text: s.src[start:s.pos], Pos: start}
		}
	}

	typ := IConst

	/*
	 * Fractional part.
	 * Guard: s.peek(1) != '.' prevents consuming the first '.' of ".."
	 * (dot_dot), replicating {numericfail} yyless(yyleng-2) (scan.l line 1020).
	 */
	if s.pos < len(s.src) && s.src[s.pos] == '.' && s.peek(1) != '.' {
		typ = FConst
		s.pos++ /* consume '.' */
		for s.pos < len(s.src) && (isDecDigit(s.src[s.pos]) || s.src[s.pos] == '_') {
			s.pos++
		}
	}

	/*
	 * Exponent part: [Ee][+-]?{decdigit}+
	 * scan.l {real} pattern (line 394) and {realfail} (line 395).
	 * No underscore separator in the exponent (plain {decinteger} used there).
	 */
	if s.pos < len(s.src) && (s.src[s.pos] == 'e' || s.src[s.pos] == 'E') {
		next := s.peek(1)
		if isDecDigit(next) || next == '+' || next == '-' {
			typ = FConst
			s.pos++ /* consume 'e'/'E' */
			if s.src[s.pos] == '+' || s.src[s.pos] == '-' {
				s.pos++
			}
			for s.pos < len(s.src) && isDecDigit(s.src[s.pos]) {
				s.pos++
			}
		}
	}

	return Token{Type: typ, Text: s.src[start:s.pos], Pos: start}
}

/*
 * ident scans an unquoted identifier and resolves it against the keyword table.
 *
 * Corresponds to the {identifier} rule in scan.l (line 1047):
 *   identifier {ident_start}{ident_cont}*  (line 331)
 *   ident_start [A-Za-z\200-\377_]          (line 328)
 *   ident_cont  [A-Za-z\200-\377_0-9\$]     (line 329)
 *
 * The high-byte range \200-\377 is the octal representation of bytes 0x80–0xFF,
 * which are the leading bytes of multi-byte UTF-8 sequences for non-ASCII
 * Unicode characters.  We decode them with utf8.DecodeRune and accept any
 * Unicode letter or digit, which is a superset of what PostgreSQL requires but
 * harmless for our read-only purpose.
 *
 * The core scanner calls downcase_truncate_identifier() (line 1067) to
 * lowercase and truncate; we lowercase only (no NAMEDATALEN truncation) for
 * the keyword lookup, but preserve the original mixed-case text in Token.Text.
 *
 * Reserved keyword lookup uses the ReservedPLKeywords list passed to
 * scanner_init() (pl_scanner.c line 628); unreserved keywords are checked in
 * plpgsql_yylex() via ScanKeywordLookup against UnreservedPLKeywords (line 248).
 * We perform both lookups here in a single map probe.
 */
func (s *Scanner) ident(start int) Token {
	for s.pos < len(s.src) {
		ch := s.src[s.pos]
		if ch < 0x80 {
			if !isIdentCont(ch) {
				break
			}
			s.pos++
		} else {
			/* Multi-byte UTF-8: scan.l ident_cont includes \200-\377 */
			r, size := utf8.DecodeRuneInString(s.src[s.pos:])
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				s.pos += size
			} else {
				break
			}
		}
	}
	text := s.src[start:s.pos]
	if kw, ok := keywords[strings.ToLower(text)]; ok {
		return Token{Type: kw, Text: text, Pos: start}
	}
	return Token{Type: Ident, Text: text, Pos: start}
}

/*
 * operator implements the full {operator} rule from scan.l (lines 861–967).
 *
 * The rule has four phases:
 *
 * Phase 1 – Greedy consumption (scan.l line 861: operator {op_chars}+).
 *   We consume all consecutive op_chars characters.
 *
 * Phase 2 – Embedded comment truncation (scan.l lines 862–881).
 *   If the consumed sequence contains "--" or "/*", the scan.l code uses
 *   strstr() to find the first such occurrence and sets nchars accordingly.
 *   We replicate this by stopping the greedy loop at the first "--" or "/*".
 *   Note: scan.l comment at line 433 says "xcstart must appear before
 *   operator" in the rule list; our Scan() handles that by dispatching to
 *   blockComment() before isOpChar().
 *
 * Phase 3 – Trailing +/- stripping (scan.l lines 883–918).
 *   For SQL compatibility, '+' and '-' cannot be the last character of a
 *   multi-character operator unless the operator contains at least one of the
 *   "qualifying" characters ~!@#^&|`?%  (scan.l comment lines 883–889).
 *   If the last char is '+' or '-' and no qualifying char is found, ALL
 *   trailing '+' and '-' characters are stripped (down to a minimum of one
 *   character).  The stripped characters are put back via yyless(nchars)
 *   (line 925) so they can be rescanned as separate tokens.
 *   We replicate the put-back by setting s.pos = start + nchars.
 *
 * Phase 4 – Single-char and two-char special returns (scan.l lines 922–953).
 *   After stripping (when nchars < original yyleng):
 *   - If nchars == 1 and the char is in the "self" set [,()\[\].;:+- * /%^<>=],
 *     return it as a character token rather than Op (line 932–934).
 *     In practice only [+-* /%^<>=] are reachable here (the other self chars
 *     are not in op_chars so they never enter operator()).
 *   - If nchars == 2, check for >=, <=, =>, <>, !=, <<, >> and return the
 *     dedicated token type (lines 941–953 plus pl_scanner.c promotions).
 *
 * Additionally, pl_scanner.c internal_yylex (lines 360–368) promotes three
 * Op tokens that the core scanner would return as generic Op:
 *   Op("<<")  → LESS_LESS
 *   Op(">>")  → GREATER_GREATER
 *   Op("#")   → TokenType('#')  (character token)
 * We handle all three in the nchars==1 / nchars==2 branches below.
 */
func (s *Scanner) operator(start int) Token {
	/*
	 * Phase 1: greedy consumption of op_chars.
	 * Phase 2: stop before "--" or "/" + "*" (embedded comment detection,
	 *          scan.l lines 869–881).
	 */
	for s.pos < len(s.src) && isOpChar(s.src[s.pos]) {
		if s.src[s.pos] == '-' && s.peek(1) == '-' {
			break
		}
		if s.src[s.pos] == '/' && s.peek(1) == '*' {
			break
		}
		s.pos++
	}
	nchars := s.pos - start

	/*
	 * Phase 3: trailing +/- rule (scan.l lines 891–918).
	 *
	 * The C code (reproduced for reference):
	 *
	 *   if (nchars > 1 &&
	 *       (yytext[nchars-1] == '+' || yytext[nchars-1] == '-'))
	 *   {
	 *       int ic;
	 *       for (ic = nchars - 2; ic >= 0; ic--)
	 *       {
	 *           char c = yytext[ic];
	 *           if (c == '~' || c == '!' || c == '@' ||    // line 900
	 *               c == '#' || c == '^' || c == '&' ||
	 *               c == '|' || c == '`' || c == '?' ||
	 *               c == '%')
	 *               break;
	 *       }
	 *       if (ic < 0)           // no qualifying char found
	 *       {
	 *           do { nchars--; }  // strip trailing +/-
	 *           while (nchars > 1 &&
	 *                  (yytext[nchars-1] == '+' || yytext[nchars-1] == '-'));
	 *       }
	 *   }
	 *   if (nchars < yyleng)
	 *       yyless(nchars);       // put back stripped chars (line 925)
	 */
	if nchars > 1 {
		last := s.src[start+nchars-1]
		if last == '+' || last == '-' {
			hasQualifying := false
			for i := 0; i < nchars-1; i++ {
				c := s.src[start+i]
				if c == '~' || c == '!' || c == '@' || c == '#' || c == '^' ||
					c == '&' || c == '|' || c == '`' || c == '?' || c == '%' {
					hasQualifying = true
					break
				}
			}
			if !hasQualifying {
				for nchars > 1 {
					if t := s.src[start+nchars-1]; t != '+' && t != '-' {
						break
					}
					nchars--
				}
			}
		}
	}

	/* yyless(nchars): put back any stripped characters (scan.l line 925). */
	s.pos = start + nchars
	text := s.src[start:s.pos]

	/*
	 * Phase 4a: single-character result.
	 *
	 * scan.l lines 932-934: "if nchars == 1 and it's one of the characters
	 * matching {self}, then return it as a character token."
	 *
	 * The full self set is [,()\[\].;:+- * /%^<>=] (scan.l line 362).
	 * The subset reachable here (i.e., in the intersection with op_chars)
	 * is [+- * /%^<>=].
	 *
	 * '#' is not in either set but pl_scanner.c (line 367) converts
	 * Op("#") to character token '#', so we do it here.
	 */
	if nchars == 1 {
		ch := s.src[start]
		switch ch {
		case '+', '-', '*', '/', '%', '^', '<', '>', '=':
			return Token{Type: TokenType(ch), Text: text, Pos: start}
		case '#':
			/* pl_scanner.c internal_yylex line 367: Op "#" → '#' */
			return Token{Type: TokenType('#'), Text: text, Pos: start}
		}
		return Token{Type: Op, Text: text, Pos: start}
	}

	/*
	 * Phase 4b: two-character result.
	 *
	 * scan.l lines 941–953: after stripping, check for the dedicated
	 * two-char token types.  Note that "<<" and ">>" are NOT in this list
	 * in scan.l (they fall through to "return Op"), but pl_scanner.c
	 * internal_yylex (lines 363–366) promotes them.  We include all six
	 * here for clarity.
	 */
	if nchars == 2 {
		switch text {
		case "<<": /* pl_scanner.c line 363 */
			return Token{Type: LessLess, Text: text, Pos: start}
		case ">>": /* pl_scanner.c line 365 */
			return Token{Type: GreaterGreater, Text: text, Pos: start}
		case ">=": /* scan.l line 945 */
			return Token{Type: GreaterEquals, Text: text, Pos: start}
		case "<=": /* scan.l line 947 */
			return Token{Type: LessEquals, Text: text, Pos: start}
		case "=>": /* scan.l line 943 */
			return Token{Type: EqualsGreater, Text: text, Pos: start}
		case "<>": /* scan.l line 949 */
			return Token{Type: NotEquals, Text: text, Pos: start}
		case "!=": /* scan.l line 951 */
			return Token{Type: NotEquals, Text: text, Pos: start}
		}
	}

	return Token{Type: Op, Text: text, Pos: start}
}

/*
 * continueString checks whether a just-closed string literal is immediately
 * continued on the next line (PostgreSQL implicit string concatenation).
 *
 * This corresponds to the xqs intermediate state in scan.l (lines 559–610)
 * and the {quotecontinue} pattern (line 226):
 *
 *   quotecontinue  {whitespace_with_newline}{quote}    line 226
 *   whitespace_with_newline
 *     {non_newline_whitespace}*{newline}{special_whitespace}*  line 222
 *   non_newline_whitespace
 *     ({non_newline_space}|{comment})                  line 221
 *   special_whitespace
 *     ({space}+|{comment}{newline})                    line 220
 *   comment  ("--"{non_newline}*)                      line 209
 *
 * In plain English: after a closing quote, if we find only horizontal
 * whitespace and/or line comments, then a newline, then more horizontal
 * whitespace and/or line comments, and then another quote, the two string
 * literals are joined into one token.
 *
 * If continuation is found, s.pos is advanced past the opening ' of the
 * continued fragment and true is returned.  Otherwise s.pos is restored and
 * false is returned, matching the xqs {quotecontinuefail} path (line 579).
 */
func (s *Scanner) continueString() bool {
	save := s.pos
	hasNewline := false

	for s.pos < len(s.src) {
		ch := s.src[s.pos]
		switch {
		case ch == '\n' || ch == '\r':
			hasNewline = true /* {newline} in whitespace_with_newline */
			s.pos++
		case ch == ' ' || ch == '\t' || ch == '\f' || ch == '\v':
			s.pos++ /* non_newline_space or space in special_whitespace */
		case ch == '-' && s.peek(1) == '-':
			/*
			 * Line comment inside the gap.
			 * non_newline_whitespace = {comment} = "--"{non_newline}*
			 * Consume up to but NOT including the newline, so the loop
			 * processes the newline on the next iteration.
			 */
			for s.pos < len(s.src) && s.src[s.pos] != '\n' && s.src[s.pos] != '\r' {
				s.pos++
			}
		default:
			goto done
		}
	}
done:
	if hasNewline && s.pos < len(s.src) && s.src[s.pos] == '\'' {
		s.pos++ /* consume the opening quote of the continuation fragment */
		return true
	}
	/* {quotecontinuefail}: yyless(0) → restore position (scan.l line 587) */
	s.pos = save
	return false
}

// ---------------------------------------------------------------------------
// Character-class predicates
// ---------------------------------------------------------------------------

/*
 * isIdentStart reports whether the ASCII byte ch can open an unquoted
 * identifier.
 *
 * scan.l: ident_start [A-Za-z\200-\377_]  (line 328).
 *
 * The \200-\377 (0x80–0xFF) range is handled separately in ident() by
 * calling utf8.DecodeRune on multi-byte sequences.
 */
func isIdentStart(ch byte) bool {
	return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

/*
 * isIdentCont reports whether the ASCII byte ch can continue an identifier.
 *
 * scan.l: ident_cont [A-Za-z\200-\377_0-9\$]  (line 329).
 *
 * '$' is included (it appears in internal PostgreSQL names like pg_catalog
 * identifiers) but '$' in user identifiers is a PostgreSQL extension.
 */
func isIdentCont(ch byte) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9') || ch == '$'
}

/*
 * isDolqStart reports whether ch can be the first byte of a dollar-quote tag.
 *
 * scan.l: dolq_start [A-Za-z\200-\377_]  (line 281).
 * No digits and no '$' are allowed at position 0 of a tag.
 * Bytes >= 0x80 are valid (UTF-8 multi-byte sequences).
 */
func isDolqStart(ch byte) bool {
	return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch >= 0x80
}

/*
 * isDolqCont reports whether ch can continue a dollar-quote tag.
 *
 * scan.l: dolq_cont [A-Za-z\200-\377_0-9]  (line 282).
 * Digits are allowed after position 0.
 */
func isDolqCont(ch byte) bool {
	return isDolqStart(ch) || (ch >= '0' && ch <= '9')
}

// isDecDigit reports whether ch is a decimal digit.
func isDecDigit(ch byte) bool { return ch >= '0' && ch <= '9' }

/*
 * isHexDigit reports whether ch is a hexadecimal digit.
 * scan.l: hexdigit [0-9A-Fa-f]  (line 378).
 */
func isHexDigit(ch byte) bool {
	return isDecDigit(ch) || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

/*
 * isOctDigit reports whether ch is an octal digit.
 * scan.l: octdigit [0-7]  (line 379).
 */
func isOctDigit(ch byte) bool { return ch >= '0' && ch <= '7' }

/*
 * isOpChar reports whether ch belongs to PostgreSQL's op_chars set.
 *
 * scan.l: op_chars [\~\!\@\#\^\&\|\`\?\+\-\*\/\%\<\>\=]  (line 363).
 *
 * Note the overlap with the self set (scan.l comment lines 352–360): both
 * sets contain +, -, *, /, %, ^, <, >, =.  For a single character in both
 * sets, flex's {self} rule fires before {operator} because it appears first
 * in scan.l (line 856 before line 861).  In our code, single-char op_chars
 * are handled inside operator() by the nchars==1 branch (phase 4a).
 */
func isOpChar(ch byte) bool {
	switch ch {
	case '~', '!', '@', '#', '^', '&', '|', '`', '?',
		'+', '-', '*', '/', '%', '<', '>', '=':
		return true
	}
	return false
}
