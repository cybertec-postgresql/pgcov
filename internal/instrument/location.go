package instrument

import "fmt"

// TrackLocation creates a new coverage point for a given file and line
func TrackLocation(file string, line int) CoveragePoint {
	return CoveragePoint{
		File:     file,
		Line:     line,
		Branch:   "",
		SignalID: FormatSignalID(file, line, ""),
	}
}

// TrackBranchLocation creates a new coverage point for a branch
func TrackBranchLocation(file string, line int, branch string) CoveragePoint {
	return CoveragePoint{
		File:     file,
		Line:     line,
		Branch:   branch,
		SignalID: FormatSignalID(file, line, branch),
	}
}

// LocationKey returns a unique key for a coverage point (without branch)
func LocationKey(file string, line int) string {
	return fmt.Sprintf("%s:%d", file, line)
}

// BranchLocationKey returns a unique key for a branch coverage point
func BranchLocationKey(file string, line int, branch string) string {
	if branch == "" {
		return LocationKey(file, line)
	}
	return fmt.Sprintf("%s:%d:%s", file, line, branch)
}

// ParseSignalID parses a signal ID into file, line, and optional branch
func ParseSignalID(signalID string) (file string, line int, branch string, err error) {
	// Signal format: file:line or file:line:branch
	// Note: file path may contain colons on Windows (C:\path\to\file.sql)

	// Find the last two colons
	lastColon := -1
	secondLastColon := -1

	for i := len(signalID) - 1; i >= 0; i-- {
		if signalID[i] == ':' {
			if lastColon == -1 {
				lastColon = i
			} else if secondLastColon == -1 {
				secondLastColon = i
				break
			}
		}
	}

	if lastColon == -1 {
		return "", 0, "", fmt.Errorf("invalid signal ID format: %s", signalID)
	}

	// Check if there's a branch (three parts)
	if secondLastColon != -1 {
		// Format: file:line:branch
		file = signalID[:secondLastColon]
		lineStr := signalID[secondLastColon+1 : lastColon]
		branch = signalID[lastColon+1:]

		var parseErr error
		line, parseErr = parseLineNumber(lineStr)
		if parseErr != nil {
			return "", 0, "", fmt.Errorf("invalid line number in signal ID %s: %w", signalID, parseErr)
		}
	} else {
		// Format: file:line
		file = signalID[:lastColon]
		lineStr := signalID[lastColon+1:]

		var parseErr error
		line, parseErr = parseLineNumber(lineStr)
		if parseErr != nil {
			return "", 0, "", fmt.Errorf("invalid line number in signal ID %s: %w", signalID, parseErr)
		}
	}

	return file, line, branch, nil
}

// parseLineNumber safely parses a line number string
func parseLineNumber(s string) (int, error) {
	var line int
	_, err := fmt.Sscanf(s, "%d", &line)
	if err != nil {
		return 0, fmt.Errorf("failed to parse line number: %w", err)
	}
	if line < 1 {
		return 0, fmt.Errorf("line number must be positive, got %d", line)
	}
	return line, nil
}

// GroupLocationsByFile groups coverage points by file
func GroupLocationsByFile(locations []CoveragePoint) map[string][]CoveragePoint {
	grouped := make(map[string][]CoveragePoint)
	for _, loc := range locations {
		grouped[loc.File] = append(grouped[loc.File], loc)
	}
	return grouped
}

// GetUniqueFiles returns a list of unique files from coverage points
func GetUniqueFiles(locations []CoveragePoint) []string {
	fileSet := make(map[string]bool)
	for _, loc := range locations {
		fileSet[loc.File] = true
	}

	var files []string
	for file := range fileSet {
		files = append(files, file)
	}
	return files
}
