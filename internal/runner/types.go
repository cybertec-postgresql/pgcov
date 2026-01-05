package runner

import (
	"time"

	"github.com/pashagolub/pgcov/internal/discovery"
)

// TestRun represents a single test execution
type TestRun struct {
	Test         *discovery.DiscoveredFile
	Database     *TempDatabase
	StartTime    time.Time
	EndTime      time.Time
	Status       TestStatus
	Error        error            // Non-nil if test failed
	CoverageSigs []CoverageSignal // Signals collected during test
}

// TestStatus represents the current state of a test execution
type TestStatus int

const (
	TestPending TestStatus = iota
	TestRunning
	TestPassed
	TestFailed
	TestTimeout
)

// String returns a string representation of TestStatus
func (ts TestStatus) String() string {
	switch ts {
	case TestPending:
		return "pending"
	case TestRunning:
		return "running"
	case TestPassed:
		return "passed"
	case TestFailed:
		return "failed"
	case TestTimeout:
		return "timeout"
	default:
		return "unknown"
	}
}

// TempDatabase represents a temporary PostgreSQL database for test isolation
type TempDatabase struct {
	Name             string // e.g., "pgcov_test_20260105_a3f9c2b1"
	CreatedAt        time.Time
	ConnectionString string
}

// CoverageSignal represents a single coverage signal emitted via NOTIFY
type CoverageSignal struct {
	SignalID  string    // Matches CoveragePoint.SignalID
	Timestamp time.Time // When signal received
	TestRun   *TestRun  // Associated test (optional reference)
}

// Duration returns the test execution duration
func (tr *TestRun) Duration() time.Duration {
	if tr.EndTime.IsZero() {
		return time.Since(tr.StartTime)
	}
	return tr.EndTime.Sub(tr.StartTime)
}

// TestSummary summarizes all test executions
type TestSummary struct {
	TotalTests    int
	PassedTests   int
	FailedTests   int
	TimedOutTests int
	TotalDuration time.Duration
}

// AllPassed returns true if all tests passed
func (s *TestSummary) AllPassed() bool {
	return s.FailedTests == 0 && s.TimedOutTests == 0
}

// ExitCode returns the appropriate exit code based on test results
func (s *TestSummary) ExitCode() int {
	if s.AllPassed() {
		return 0
	}
	return 1
}
