package instrument

import (
	"fmt"
	"strconv"
)

// FormatSignalID generates a signal ID for a coverage point
// Format: {file}:{line} or {file}:{line}:{branch}
func FormatSignalID(file string, line int, branch string) string {
	if branch == "" {
		return fmt.Sprintf("%s:%d", file, line)
	}
	return fmt.Sprintf("%s:%d:%s", file, line, branch)
}

// FormatSignalIDFromPoint generates a signal ID from a CoveragePoint
func FormatSignalIDFromPoint(cp CoveragePoint) string {
	return FormatSignalID(cp.File, cp.Line, cp.Branch)
}

// ValidateSignalID checks if a signal ID is properly formatted
func ValidateSignalID(signalID string) error {
	_, line, _, err := ParseSignalID(signalID)
	if err != nil {
		return fmt.Errorf("invalid signal ID: %w", err)
	}
	
	if line < 1 {
		return fmt.Errorf("signal ID has invalid line number: %d", line)
	}
	
	return nil
}

// GenerateUniqueSignalID creates a unique signal ID with optional suffix
func GenerateUniqueSignalID(file string, line int, branch string, suffix string) string {
	baseID := FormatSignalID(file, line, branch)
	if suffix != "" {
		return baseID + "_" + suffix
	}
	return baseID
}

// CompareSignalIDs compares two signal IDs for equality
func CompareSignalIDs(id1, id2 string) bool {
	return id1 == id2
}

// ExtractLineFromSignalID extracts just the line number from a signal ID
func ExtractLineFromSignalID(signalID string) (int, error) {
	_, line, _, err := ParseSignalID(signalID)
	return line, err
}

// ExtractFileFromSignalID extracts just the file path from a signal ID
func ExtractFileFromSignalID(signalID string) (string, error) {
	file, _, _, err := ParseSignalID(signalID)
	return file, err
}

// SignalIDToKey converts a signal ID to a simple string key for map lookups
func SignalIDToKey(signalID string) string {
	// Signal ID is already unique, so it can be used directly as a key
	return signalID
}

// LineToString converts an integer line number to string
func LineToString(line int) string {
	return strconv.Itoa(line)
}

// StringToLine converts a string line number to integer
func StringToLine(s string) (int, error) {
	return strconv.Atoi(s)
}
