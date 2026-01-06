package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Discover recursively finds all SQL files in the given directory
func Discover(rootPath string) ([]DiscoveredFile, error) {
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if directory exists
	info, err := os.Stat(absRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory not found: %s", absRoot)
		}
		return nil, fmt.Errorf("failed to access directory: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", absRoot)
	}

	var files []DiscoveredFile

	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories we can't access
			if os.IsPermission(err) {
				return nil
			}
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process .sql files
		if !strings.HasSuffix(strings.ToLower(path), ".sql") {
			return nil
		}

		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Classify the file
		fileType := ClassifyFile(filepath.Base(path))

		files = append(files, DiscoveredFile{
			Path:         path,
			RelativePath: relPath,
			Type:         fileType,
			ModTime:      info.ModTime(),
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return files, nil
}

// DiscoverTests finds only test files (*_test.sql) in the given directory
func DiscoverTests(rootPath string) ([]DiscoveredFile, error) {
	allFiles, err := Discover(rootPath)
	if err != nil {
		return nil, err
	}

	var testFiles []DiscoveredFile
	for _, file := range allFiles {
		if file.Type == FileTypeTest {
			testFiles = append(testFiles, file)
		}
	}

	return testFiles, nil
}

// DiscoverSources finds only source files (*.sql but not *_test.sql) in the given directory
func DiscoverSources(rootPath string) ([]DiscoveredFile, error) {
	allFiles, err := Discover(rootPath)
	if err != nil {
		return nil, err
	}

	var sourceFiles []DiscoveredFile
	for _, file := range allFiles {
		if file.Type == FileTypeSource {
			sourceFiles = append(sourceFiles, file)
		}
	}

	return sourceFiles, nil
}

// DiscoverCoLocatedSources finds source files in the same directories as test files
// This implements the co-location strategy where tests and source code are kept together
func DiscoverCoLocatedSources(testFiles []DiscoveredFile) ([]DiscoveredFile, error) {
	// Collect unique directories containing test files
	testDirs := make(map[string]bool)
	for _, test := range testFiles {
		testDirs[filepath.Dir(test.Path)] = true
	}

	// Discover all source files in those directories
	var sourceFiles []DiscoveredFile
	seenFiles := make(map[string]bool) // Avoid duplicates

	for testDir := range testDirs {
		files, err := DiscoverSources(testDir)
		if err != nil {
			return nil, fmt.Errorf("failed to discover sources in %s: %w", testDir, err)
		}

		for _, file := range files {
			if !seenFiles[file.Path] {
				sourceFiles = append(sourceFiles, file)
				seenFiles[file.Path] = true
			}
		}
	}

	return sourceFiles, nil
}
