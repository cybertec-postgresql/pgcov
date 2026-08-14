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
		return FileTypeUnknown
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


