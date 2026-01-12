package instrument

import (
	"fmt"
)

// FormatSignalID generates a signal ID for a coverage point
// Format: {file}:{startPos}:{length} or {file}:{startPos}:{length}:{branch}
func FormatSignalID(file string, startPos int, length int, branch string) string {
	if branch == "" {
		return fmt.Sprintf("%s:%d:%d", file, startPos, length)
	}
	return fmt.Sprintf("%s:%d:%d:%s", file, startPos, length, branch)
}

// FormatSignalIDFromPoint generates a signal ID from a CoveragePoint
func FormatSignalIDFromPoint(cp CoveragePoint) string {
	return FormatSignalID(cp.File, cp.StartPos, cp.Length, cp.Branch)
}

// ValidateSignalID checks if a signal ID is properly formatted
func ValidateSignalID(signalID string) error {
	_, startPos, length, _, err := ParseSignalID(signalID)
	if err != nil {
		return fmt.Errorf("invalid signal ID: %w", err)
	}

	if startPos < 0 {
		return fmt.Errorf("signal ID has invalid start position: %d", startPos)
	}

	if length < 0 {
		return fmt.Errorf("signal ID has invalid length: %d", length)
	}

	return nil
}

// GenerateUniqueSignalID creates a unique signal ID with optional suffix
func GenerateUniqueSignalID(file string, startPos int, length int, branch string, suffix string) string {
	baseID := FormatSignalID(file, startPos, length, branch)
	if suffix != "" {
		return baseID + "_" + suffix
	}
	return baseID
}

// CompareSignalIDs compares two signal IDs for equality
func CompareSignalIDs(id1, id2 string) bool {
	return id1 == id2
}

// ExtractPositionFromSignalID extracts the start position from a signal ID
func ExtractPositionFromSignalID(signalID string) (int, error) {
	_, startPos, _, _, err := ParseSignalID(signalID)
	return startPos, err
}

// ExtractFileFromSignalID extracts just the file path from a signal ID
func ExtractFileFromSignalID(signalID string) (string, error) {
	file, _, _, _, err := ParseSignalID(signalID)
	return file, err
}

// SignalIDToKey converts a signal ID to a simple string key for map lookups
func SignalIDToKey(signalID string) string {
	// Signal ID is already unique, so it can be used directly as a key
	return signalID
}
