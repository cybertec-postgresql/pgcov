package discovery

import (
	"path/filepath"
	"strings"
)

// ClassifyFile determines if a file is a test or source file based on naming convention
func ClassifyFile(filename string) FileType {
	// Normalize to lowercase for case-insensitive comparison
	lower := strings.ToLower(filename)

	// Check if it's a SQL file
	if !strings.HasSuffix(lower, ".sql") {
		return FileTypeSource // Non-SQL files are treated as source (edge case)
	}

	// Test files match *_test.sql pattern
	if strings.HasSuffix(lower, "_test.sql") {
		return FileTypeTest
	}

	// Everything else is a source file
	return FileTypeSource
}

// ClassifyPath determines file type from a full path
func ClassifyPath(path string) FileType {
	return ClassifyFile(filepath.Base(path))
}

// IsTestFile returns true if the file is a test file
func IsTestFile(filename string) bool {
	return ClassifyFile(filename) == FileTypeTest
}

// IsSourceFile returns true if the file is a source file
func IsSourceFile(filename string) bool {
	return ClassifyFile(filename) == FileTypeSource
}
