package runner_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/cybertec-postgresql/pgcov/internal/discovery"
	"github.com/cybertec-postgresql/pgcov/internal/runner"
)

func TestFormatFailedTests(t *testing.T) {
	tests := []struct {
		name    string
		runs    []*runner.TestRun
		want    []string
		wantNil bool
	}{
		{
			name:    "no runs",
			runs:    nil,
			wantNil: true,
		},
		{
			name: "all passed",
			runs: []*runner.TestRun{
				{Test: &discovery.DiscoveredFile{RelativePath: "a_test.sql"}, Status: runner.TestPassed},
			},
			wantNil: true,
		},
		{
			name: "failed with error",
			runs: []*runner.TestRun{
				{
					Test:   &discovery.DiscoveredFile{RelativePath: "dir/broken_test.sql"},
					Status: runner.TestFailed,
					Error:  errors.New("test execution failed: syntax error at or near \"FORM\""),
				},
			},
			want: []string{
				"FAILED dir/broken_test.sql: test execution failed: syntax error at or near \"FORM\"",
			},
		},
		{
			name: "mixed statuses only failed surfaces",
			runs: []*runner.TestRun{
				{Test: &discovery.DiscoveredFile{RelativePath: "a_test.sql"}, Status: runner.TestPassed},
				{
					Test:   &discovery.DiscoveredFile{RelativePath: "b_test.sql"},
					Status: runner.TestFailed,
					Error:  errors.New("boom"),
				},
				{
					Test:   &discovery.DiscoveredFile{RelativePath: "c_test.sql"},
					Status: runner.TestTimeout,
					Error:  errors.New("context deadline exceeded"),
				},
				{
					Test:   &discovery.DiscoveredFile{RelativePath: "d_test.sql"},
					Status: runner.TestFailed,
					// No error attached — runner normally populates this, but
					// the helper must not panic if a caller hands in a run
					// with Status=TestFailed and Error=nil.
				},
			},
			want: []string{
				"FAILED b_test.sql: boom",
			},
		},
		{
			name: "nil entry safely skipped",
			runs: []*runner.TestRun{
				nil,
				{
					Test:   &discovery.DiscoveredFile{RelativePath: "x_test.sql"},
					Status: runner.TestFailed,
					Error:  errors.New("kaboom"),
				},
			},
			want: []string{
				"FAILED x_test.sql: kaboom",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := runner.FormatFailedTests(tc.runs)
			if tc.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}
			if len(got) != len(tc.want) {
				t.Fatalf("expected %d line(s), got %d: %v", len(tc.want), len(got), got)
			}
			for i, line := range got {
				if line != tc.want[i] {
					t.Errorf("line %d: want %q, got %q", i, tc.want[i], line)
				}
				// Sanity: every emitted line carries the FAILED prefix so the
				// summary block stays scannable next to the "Tests:" line.
				if !strings.HasPrefix(line, "FAILED ") {
					t.Errorf("line %d missing FAILED prefix: %q", i, line)
				}
			}
		})
	}
}
