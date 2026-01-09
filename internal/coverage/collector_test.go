package coverage

import (
	"sync"
	"testing"
	"time"

	"github.com/cybertec-postgresql/pgcov/internal/runner"
)

func TestNewCollector(t *testing.T) {
	c := NewCollector()
	if c == nil {
		t.Fatal("NewCollector() returned nil")
	}

	if c.coverage == nil {
		t.Error("NewCollector() coverage is nil")
	}
}

func TestCollector_AddSignal(t *testing.T) {
	c := NewCollector()

	signal := runner.CoverageSignal{
		SignalID:  "test.sql:10",
		Timestamp: time.Now(),
	}

	err := c.AddSignal(signal)
	if err != nil {
		t.Fatalf("AddSignal() error = %v", err)
	}

	// Verify signal was recorded
	hits := c.GetFileCoverage("test.sql")
	if hits[10] != 1 {
		t.Errorf("AddSignal() line 10 hit count = %d, want 1", hits[10])
	}
}

func TestCollector_AddSignal_Multiple(t *testing.T) {
	c := NewCollector()

	// Add same signal multiple times
	signal := runner.CoverageSignal{
		SignalID:  "test.sql:10",
		Timestamp: time.Now(),
	}

	for i := 0; i < 5; i++ {
		err := c.AddSignal(signal)
		if err != nil {
			t.Fatalf("AddSignal() error = %v", err)
		}
	}

	// Verify hit count is accumulated
	hits := c.GetFileCoverage("test.sql")
	if hits[10] != 5 {
		t.Errorf("AddSignal() line 10 hit count = %d, want 5", hits[10])
	}
}

func TestCollector_AddSignal_InvalidSignalID(t *testing.T) {
	c := NewCollector()

	signal := runner.CoverageSignal{
		SignalID:  "invalid-signal",
		Timestamp: time.Now(),
	}

	err := c.AddSignal(signal)
	if err == nil {
		t.Error("AddSignal() expected error for invalid signal ID, got nil")
	}
}

func TestCollector_CollectFromRun(t *testing.T) {
	c := NewCollector()

	now := time.Now()
	testRun := &runner.TestRun{
		CoverageSigs: []runner.CoverageSignal{
			{SignalID: "test.sql:10", Timestamp: now},
			{SignalID: "test.sql:20", Timestamp: now.Add(time.Second)},
			{SignalID: "test.sql:30", Timestamp: now.Add(2 * time.Second)},
		},
	}

	err := c.CollectFromRun(testRun)
	if err != nil {
		t.Fatalf("CollectFromRun() error = %v", err)
	}

	// Verify all signals were recorded
	hits := c.GetFileCoverage("test.sql")
	if hits[10] != 1 {
		t.Errorf("CollectFromRun() line 10 hit count = %d, want 1", hits[10])
	}
	if hits[20] != 1 {
		t.Errorf("CollectFromRun() line 20 hit count = %d, want 1", hits[20])
	}
	if hits[30] != 1 {
		t.Errorf("CollectFromRun() line 30 hit count = %d, want 1", hits[30])
	}
}

func TestCollector_CollectFromRuns(t *testing.T) {
	c := NewCollector()

	now := time.Now()
	testRuns := []*runner.TestRun{
		{
			CoverageSigs: []runner.CoverageSignal{
				{SignalID: "test.sql:10", Timestamp: now},
			},
		},
		{
			CoverageSigs: []runner.CoverageSignal{
				{SignalID: "test.sql:10", Timestamp: now.Add(time.Second)},
				{SignalID: "test.sql:20", Timestamp: now.Add(2 * time.Second)},
			},
		},
	}

	err := c.CollectFromRuns(testRuns)
	if err != nil {
		t.Fatalf("CollectFromRuns() error = %v", err)
	}

	// Verify hit counts are aggregated
	hits := c.GetFileCoverage("test.sql")
	if hits[10] != 2 {
		t.Errorf("CollectFromRuns() line 10 hit count = %d, want 2", hits[10])
	}
	if hits[20] != 1 {
		t.Errorf("CollectFromRuns() line 20 hit count = %d, want 1", hits[20])
	}
}

func TestCollector_ThreadSafe(t *testing.T) {
	c := NewCollector()

	// Add signals concurrently
	var wg sync.WaitGroup
	numGoroutines := 10
	signalsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < signalsPerGoroutine; j++ {
				signal := runner.CoverageSignal{
					SignalID:  "test.sql:10",
					Timestamp: time.Now(),
				}
				_ = c.AddSignal(signal)
			}
		}()
	}

	wg.Wait()

	// Verify hit count is correct (should be numGoroutines * signalsPerGoroutine)
	expectedHits := numGoroutines * signalsPerGoroutine
	hits := c.GetFileCoverage("test.sql")
	if hits[10] != expectedHits {
		t.Errorf("Thread-safe AddSignal() line 10 hit count = %d, want %d", hits[10], expectedHits)
	}
}

func TestCollector_Reset(t *testing.T) {
	c := NewCollector()

	// Add some signals
	signal := runner.CoverageSignal{
		SignalID:  "test.sql:10",
		Timestamp: time.Now(),
	}
	_ = c.AddSignal(signal)

	// Reset
	c.Reset()

	// Verify coverage is cleared
	hits := c.GetFileCoverage("test.sql")
	if len(hits) != 0 {
		t.Errorf("Reset() coverage not cleared, got %d lines", len(hits))
	}
}

func TestCollector_Merge(t *testing.T) {
	c1 := NewCollector()
	c2 := NewCollector()

	now := time.Now()
	// Add signals to first collector
	_ = c1.AddSignal(runner.CoverageSignal{SignalID: "test.sql:10", Timestamp: now})
	_ = c1.AddSignal(runner.CoverageSignal{SignalID: "test.sql:20", Timestamp: now.Add(time.Second)})

	// Add signals to second collector
	_ = c2.AddSignal(runner.CoverageSignal{SignalID: "test.sql:10", Timestamp: now.Add(2 * time.Second)})
	_ = c2.AddSignal(runner.CoverageSignal{SignalID: "test.sql:30", Timestamp: now.Add(3 * time.Second)})

	// Merge c2 into c1
	err := c1.Merge(c2)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// Verify merged results
	hits := c1.GetFileCoverage("test.sql")
	if hits[10] != 2 {
		t.Errorf("Merge() line 10 hit count = %d, want 2", hits[10])
	}
	if hits[20] != 1 {
		t.Errorf("Merge() line 20 hit count = %d, want 1", hits[20])
	}
	if hits[30] != 1 {
		t.Errorf("Merge() line 30 hit count = %d, want 1", hits[30])
	}
}

func TestCollector_GetFileList(t *testing.T) {
	c := NewCollector()

	now := time.Now()
	// Add signals for multiple files
	_ = c.AddSignal(runner.CoverageSignal{SignalID: "file1.sql:10", Timestamp: now})
	_ = c.AddSignal(runner.CoverageSignal{SignalID: "file2.sql:10", Timestamp: now.Add(time.Second)})
	_ = c.AddSignal(runner.CoverageSignal{SignalID: "file3.sql:10", Timestamp: now.Add(2 * time.Second)})

	files := c.GetFileList()
	if len(files) != 3 {
		t.Errorf("GetFileList() got %d files, want 3", len(files))
	}

	// Verify all files are present
	fileMap := make(map[string]bool)
	for _, file := range files {
		fileMap[file] = true
	}

	if !fileMap["file1.sql"] {
		t.Error("GetFileList() missing file1.sql")
	}
	if !fileMap["file2.sql"] {
		t.Error("GetFileList() missing file2.sql")
	}
	if !fileMap["file3.sql"] {
		t.Error("GetFileList() missing file3.sql")
	}
}

func TestCollector_Coverage(t *testing.T) {
	c := NewCollector()

	// Add some signals
	_ = c.AddSignal(runner.CoverageSignal{SignalID: "test.sql:10", Timestamp: time.Now()})

	coverage := c.Coverage()
	if coverage == nil {
		t.Fatal("Coverage() returned nil")
	}

	// Verify coverage contains data
	if len(coverage.Files) == 0 {
		t.Error("Coverage() returned empty Files map")
	}
}

func TestCollector_TotalCoveragePercent(t *testing.T) {
	c := NewCollector()

	// Initially should be 0 or NaN (no lines tracked)
	percent := c.TotalCoveragePercent()
	if percent < 0 || percent > 100 {
		t.Logf("TotalCoveragePercent() = %f (expected 0-100 range)", percent)
	}

	// Add some signals
	_ = c.AddSignal(runner.CoverageSignal{SignalID: "test.sql:10", Timestamp: time.Now()})

	percent = c.TotalCoveragePercent()
	if percent < 0 || percent > 100 {
		t.Errorf("TotalCoveragePercent() = %f, want 0-100 range", percent)
	}
}

func TestCollector_MultipleFiles(t *testing.T) {
	c := NewCollector()

	now := time.Now()
	// Add signals for multiple files
	_ = c.AddSignal(runner.CoverageSignal{SignalID: "file1.sql:10", Timestamp: now})
	_ = c.AddSignal(runner.CoverageSignal{SignalID: "file1.sql:20", Timestamp: now.Add(time.Second)})
	_ = c.AddSignal(runner.CoverageSignal{SignalID: "file2.sql:15", Timestamp: now.Add(2 * time.Second)})

	// Verify file1 coverage
	hits1 := c.GetFileCoverage("file1.sql")
	if hits1[10] != 1 {
		t.Errorf("file1.sql line 10 hit count = %d, want 1", hits1[10])
	}
	if hits1[20] != 1 {
		t.Errorf("file1.sql line 20 hit count = %d, want 1", hits1[20])
	}

	// Verify file2 coverage
	hits2 := c.GetFileCoverage("file2.sql")
	if hits2[15] != 1 {
		t.Errorf("file2.sql line 15 hit count = %d, want 1", hits2[15])
	}
}
