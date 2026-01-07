package discovery

import "time"

// DiscoveredFile represents a SQL file discovered during filesystem traversal
type DiscoveredFile struct {
	Path         string    // Absolute path to file
	RelativePath string    // Path relative to search root
	Type         FileType  // Test or Source
	ModTime      time.Time // Last modification time
}

// FileType indicates whether a file is a test or source file
type FileType int

const (
	FileTypeTest   FileType = iota // Matches *_test.sql
	FileTypeSource                 // Does not match *_test.sql
)

// String returns a string representation of FileType
func (ft FileType) String() string {
	switch ft {
	case FileTypeTest:
		return "test"
	case FileTypeSource:
		return "source"
	default:
		return "unknown"
	}
}
