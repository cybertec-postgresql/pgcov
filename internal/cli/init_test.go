package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const twoFunctionSQL = `-- Two CREATE [OR REPLACE] FUNCTION statements used by the init tests.
-- One schema-qualified, one with OR REPLACE and a default-value argument.

CREATE FUNCTION public.add_one(x integer) RETURNS integer AS $$
BEGIN
    RETURN x + 1;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION echo_message(
    msg text,
    shout boolean DEFAULT false,
    repeats integer DEFAULT 1
) RETURNS text AS $$
DECLARE
    out_text text := msg;
BEGIN
    IF shout THEN
        out_text := upper(out_text);
    END IF;
    RETURN out_text;
END;
$$ LANGUAGE plpgsql;
`

func TestExtractFunctions_TwoFunctions(t *testing.T) {
	got := extractFunctions(twoFunctionSQL)

	if len(got) != 2 {
		t.Fatalf("extractFunctions() got %d functions, want 2", len(got))
	}

	first := got[0]
	if first.Schema != "public" {
		t.Errorf("first.Schema = %q, want %q", first.Schema, "public")
	}
	if first.Name != "add_one" {
		t.Errorf("first.Name = %q, want %q", first.Name, "add_one")
	}
	if first.Args != "integer" {
		t.Errorf("first.Args = %q, want %q", first.Args, "integer")
	}
	if first.StartLine <= 0 {
		t.Errorf("first.StartLine = %d, want > 0", first.StartLine)
	}

	second := got[1]
	if second.Schema != "" {
		t.Errorf("second.Schema = %q, want empty", second.Schema)
	}
	if second.Name != "echo_message" {
		t.Errorf("second.Name = %q, want %q", second.Name, "echo_message")
	}
	if second.Args != "text, boolean, integer" {
		t.Errorf("second.Args = %q, want %q", second.Args, "text, boolean, integer")
	}
	if second.StartLine <= first.StartLine {
		t.Errorf("second.StartLine=%d should be > first.StartLine=%d", second.StartLine, first.StartLine)
	}
}

func TestExtractFunctions_NoFunctions(t *testing.T) {
	sql := "CREATE TABLE t (id int); SELECT 1;"
	if got := extractFunctions(sql); len(got) != 0 {
		t.Fatalf("extractFunctions() got %d, want 0", len(got))
	}
}

func TestExtractFunctions_QuotedIdentifiers(t *testing.T) {
	sql := `CREATE FUNCTION "My Schema"."my-func"(x int) RETURNS int AS $$ SELECT x; $$ LANGUAGE sql;`
	got := extractFunctions(sql)
	if len(got) != 1 {
		t.Fatalf("got %d functions, want 1", len(got))
	}
	if got[0].Schema != "My Schema" {
		t.Errorf("Schema = %q, want %q", got[0].Schema, "My Schema")
	}
	if got[0].Name != "my-func" {
		t.Errorf("Name = %q, want %q", got[0].Name, "my-func")
	}
	if got[0].Args != "int" {
		t.Errorf("Args = %q, want %q", got[0].Args, "int")
	}
}

func TestExtractFunctions_NoArgList(t *testing.T) {
	// SQL syntax requires parens but legacy code may omit them in some dialects;
	// we record zero-arg functions without panicking.
	sql := "CREATE FUNCTION zero() RETURNS void AS $$ BEGIN END; $$ LANGUAGE plpgsql;"
	got := extractFunctions(sql)
	if len(got) != 1 {
		t.Fatalf("got %d, want 1", len(got))
	}
	if got[0].Args != "" {
		t.Errorf("Args = %q, want empty", got[0].Args)
	}
}

func TestGenerateScaffold_ContentAndOrder(t *testing.T) {
	functions := []FunctionInfo{
		{Schema: "public", Name: "add_one", Args: "integer", StartLine: 5},
		{Schema: "", Name: "echo_message", Args: "text, boolean, integer", StartLine: 14},
	}

	scaffold := generateScaffold("/tmp/my.sql", functions)

	wantSubstrings := []string{
		"Auto-generated scaffold for /tmp/my.sql",
		"pgcov init",
		"public.add_one",
		"echo_message",
		"Function 1 of 2",
		"Function 2 of 2",
		"-- SELECT public.add_one(integer);",
		"-- SELECT echo_message(text, boolean, integer);",
		"-- DO $$ BEGIN",
		"--     PERFORM public.add_one(/* fill in literal arguments */);",
	}

	for _, sub := range wantSubstrings {
		if !strings.Contains(scaffold, sub) {
			t.Errorf("scaffold missing %q\n---\n%s", sub, scaffold)
		}
	}

	// Deterministic order: add_one must appear before echo_message.
	addIdx := strings.Index(scaffold, "public.add_one")
	echoIdx := strings.Index(scaffold, "echo_message")
	if addIdx < 0 || echoIdx < 0 || addIdx > echoIdx {
		t.Errorf("expected add_one before echo_message; addIdx=%d echoIdx=%d", addIdx, echoIdx)
	}
}

func TestGenerateScaffold_NoFunctions(t *testing.T) {
	got := generateScaffold("/tmp/empty.sql", nil)
	if !strings.Contains(got, "Auto-generated scaffold for /tmp/empty.sql") {
		t.Errorf("missing header:\n%s", got)
	}
	if !strings.Contains(got, "No CREATE [OR REPLACE] FUNCTION") {
		t.Errorf("missing empty-state note:\n%s", got)
	}
}

func TestDefaultScaffoldPath(t *testing.T) {
	got := defaultScaffoldPath("/a/b/foo.sql")
	want := filepath.Join("/a", "b", "foo_test.sql")
	if got != want {
		t.Errorf("defaultScaffoldPath = %q, want %q", got, want)
	}
}

func TestInit_WritesScaffoldAndRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "demo.sql")
	if err := os.WriteFile(source, []byte(twoFunctionSQL), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	out, err := Init(source, "", false)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if !strings.HasSuffix(out, "demo_test.sql") {
		t.Errorf("Init output = %q, want suffix demo_test.sql", out)
	}

	body, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read scaffold: %v", err)
	}
	text := string(body)

	for _, want := range []string{"public.add_one", "echo_message", "DO $$ BEGIN"} {
		if !strings.Contains(text, want) {
			t.Errorf("scaffold missing %q", want)
		}
	}

	// Refusing overwrite.
	if _, err := Init(source, "", false); err == nil {
		t.Error("Init() second call should fail without --force")
	}

	// --force overwrites.
	out2, err := Init(source, "", true)
	if err != nil {
		t.Fatalf("Init() with force error = %v", err)
	}
	if out2 != out {
		t.Errorf("force rewrite path changed: %q vs %q", out2, out)
	}
}

func TestInit_RejectsNonSQLSource(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.txt")
	if err := os.WriteFile(bad, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Init(bad, "", false); err == nil {
		t.Error("Init() should reject non-.sql source")
	}
}

func TestInit_RejectsMissingSource(t *testing.T) {
	if _, err := Init(filepath.Join(t.TempDir(), "nope.sql"), "", false); err == nil {
		t.Error("Init() should reject missing source")
	}
}

func TestInit_RespectsExplicitOutput(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "alpha.sql")
	if err := os.WriteFile(source, []byte(twoFunctionSQL), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	out := filepath.Join(dir, "custom_name.sql")
	got, err := Init(source, out, false)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if got != out {
		t.Errorf("Init returned %q, want %q", got, out)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("expected output at %q: %v", out, err)
	}
}
