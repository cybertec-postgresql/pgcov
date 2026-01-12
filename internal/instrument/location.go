package instrument

import "fmt"

// TrackPosition creates a new coverage point for a given file, position, and length
func TrackPosition(file string, startPos int, length int) CoveragePoint {
	return CoveragePoint{
		File:     file,
		StartPos: startPos,
		Length:   length,
		Branch:   "",
		SignalID: FormatSignalID(file, startPos, length, ""),
	}
}

// TrackBranchPosition creates a new coverage point for a branch
func TrackBranchPosition(file string, startPos int, length int, branch string) CoveragePoint {
	return CoveragePoint{
		File:     file,
		StartPos: startPos,
		Length:   length,
		Branch:   branch,
		SignalID: FormatSignalID(file, startPos, length, branch),
	}
}

// PositionKey returns a unique key for a coverage point (without branch)
func PositionKey(file string, startPos int, length int) string {
	return fmt.Sprintf("%s:%d:%d", file, startPos, length)
}

// BranchPositionKey returns a unique key for a branch coverage point
func BranchPositionKey(file string, startPos int, length int, branch string) string {
	if branch == "" {
		return PositionKey(file, startPos, length)
	}
	return fmt.Sprintf("%s:%d:%d:%s", file, startPos, length, branch)
}

// ParseSignalID parses a signal ID into file, startPos, length, and optional branch
func ParseSignalID(signalID string) (file string, startPos int, length int, branch string, err error) {
	// Signal format: file:startPos:length or file:startPos:length:branch
	// Note: file path may contain colons on Windows (C:\path\to\file.sql)

	// Find the last colons from the end
	// We need to find: lastColon (branch separator if present), secondLastColon (length), thirdLastColon (startPos)
	colons := []int{}
	for i := len(signalID) - 1; i >= 0; i-- {
		if signalID[i] == ':' {
			colons = append(colons, i)
			if len(colons) >= 3 {
				break
			}
		}
	}

	if len(colons) < 2 {
		return "", 0, 0, "", fmt.Errorf("invalid signal ID format (expected at least 3 parts): %s", signalID)
	}

	// Check if there's a branch (four parts)
	if len(colons) >= 3 {
		// Could be file:startPos:length:branch
		// colons[0] = lastColon, colons[1] = secondLast, colons[2] = thirdLast

		// Try to parse as file:startPos:length:branch first
		thirdLastColon := colons[2]
		secondLastColon := colons[1]
		lastColon := colons[0]

		file = signalID[:thirdLastColon]
		startPosStr := signalID[thirdLastColon+1 : secondLastColon]
		lengthStr := signalID[secondLastColon+1 : lastColon]
		possibleBranch := signalID[lastColon+1:]

		// Try to parse startPos and length
		var startPosVal, lengthVal int
		_, parseErr1 := fmt.Sscanf(startPosStr, "%d", &startPosVal)
		_, parseErr2 := fmt.Sscanf(lengthStr, "%d", &lengthVal)

		if parseErr1 == nil && parseErr2 == nil {
			// Successfully parsed as file:startPos:length:branch
			if startPosVal < 0 {
				return "", 0, 0, "", fmt.Errorf("start position must be non-negative, got %d", startPosVal)
			}
			if lengthVal < 0 {
				return "", 0, 0, "", fmt.Errorf("length must be non-negative, got %d", lengthVal)
			}
			return file, startPosVal, lengthVal, possibleBranch, nil
		}
	}

	// Format: file:startPos:length (no branch)
	// colons[0] = lastColon, colons[1] = secondLast
	secondLastColon := colons[1]
	lastColon := colons[0]

	file = signalID[:secondLastColon]
	startPosStr := signalID[secondLastColon+1 : lastColon]
	lengthStr := signalID[lastColon+1:]

	var parseErr error
	startPos, parseErr = parseNumber(startPosStr)
	if parseErr != nil {
		return "", 0, 0, "", fmt.Errorf("invalid start position in signal ID %s: %w", signalID, parseErr)
	}

	length, parseErr = parseNumber(lengthStr)
	if parseErr != nil {
		return "", 0, 0, "", fmt.Errorf("invalid length in signal ID %s: %w", signalID, parseErr)
	}

	if startPos < 0 {
		return "", 0, 0, "", fmt.Errorf("start position must be non-negative, got %d", startPos)
	}
	if length < 0 {
		return "", 0, 0, "", fmt.Errorf("length must be non-negative, got %d", length)
	}

	return file, startPos, length, "", nil
}

// parseNumber safely parses a number string
func parseNumber(s string) (int, error) {
	var num int
	_, err := fmt.Sscanf(s, "%d", &num)
	if err != nil {
		return 0, fmt.Errorf("failed to parse number: %w", err)
	}
	return num, nil
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

// ConvertPositionToLine converts a byte position in source text to a line number (1-indexed)
func ConvertPositionToLine(sourceText string, startPos int) int {
	if startPos < 0 || startPos > len(sourceText) {
		return 1
	}

	line := 1
	for i := 0; i < startPos && i < len(sourceText); i++ {
		if sourceText[i] == '\n' {
			line++
		}
	}
	return line
}
