package discovery

import (
	"testing"
)

func TestClassifyFile_SQLFiles(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     FileType
	}{
		{"test file lower-case", "foo_test.sql", FileTypeTest},
		{"test file upper-case", "FOO_TEST.SQL", FileTypeTest},
		{"test file mixed-case", "Foo_Test.Sql", FileTypeTest},
		{"test file nested name", "create_user_test.sql", FileTypeTest},

		{"source file simple", "foo.sql", FileTypeSource},
		{"source file upper-case", "FOO.SQL", FileTypeSource},
		{"source file mixed-case", "Foo.Sql", FileTypeSource},
		{"source file containing test", "testing.sql", FileTypeSource},
		{"source file ending with tests", "tests.sql", FileTypeSource},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyFile(tt.filename); got != tt.want {
				t.Errorf("ClassifyFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestClassifyFile_NonSQLFiles(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     FileType
	}{
		{"README.md", "README.md", FileTypeUnknown},
		{"upper-case README", "README.MD", FileTypeUnknown},
		{"plain text file", "notes.txt", FileTypeUnknown},
		{"yaml config", "config.yaml", FileTypeUnknown},
		{"empty string", "", FileTypeUnknown},
		{"no extension", "Makefile", FileTypeUnknown},
		{"sql-like but wrong extension", "foo.sq", FileTypeUnknown},
		{"sql embedded in name without suffix", "test.sql.bak", FileTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyFile(tt.filename); got != tt.want {
				t.Errorf("ClassifyFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestClassifyFile_DefaultReturnsUnknown(t *testing.T) {
	// Out-of-range FileType should map to "unknown" in String().
	ft := FileType(99)
	if got := ft.String(); got != "unknown" {
		t.Errorf("FileType(99).String() = %q, want %q", got, "unknown")
	}
}

func TestFileType_String(t *testing.T) {
	tests := []struct {
		name string
		ft   FileType
		want string
	}{
		{"test", FileTypeTest, "test"},
		{"source", FileTypeSource, "source"},
		{"unknown", FileTypeUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ft.String(); got != tt.want {
				t.Errorf("FileType(%d).String() = %q, want %q", tt.ft, got, tt.want)
			}
		})
	}
}

func TestClassifyPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want FileType
	}{
		{"test file absolute", "/tmp/foo_test.sql", FileTypeTest},
		{"source file absolute", "/tmp/foo.sql", FileTypeSource},
		{"non-sql file absolute", "/tmp/README.md", FileTypeUnknown},
		{"test file with leading dir", "sub/dir/foo_test.sql", FileTypeTest},
		{"non-sql with mixed path separators", `sub\dir\README.md`, FileTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyPath(tt.path); got != tt.want {
				t.Errorf("ClassifyPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
