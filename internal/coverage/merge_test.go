package coverage

import (
	"testing"
)

func TestMerge_OverlappingAndDisjoint(t *testing.T) {
	a := NewCoverage()
	a.AddPosition("a.sql", 100, 50, 2)
	a.AddPosition("a.sql", 200, 60, 1)
	a.AddPosition("shared.sql", 10, 5, 3)

	b := NewCoverage()
	b.AddPosition("a.sql", 100, 50, 5)    // overlaps a.sql:100:50 -> 2+5 = 7
	b.AddPosition("b.sql", 300, 70, 4)    // disjoint
	b.AddPosition("shared.sql", 10, 5, 1) // overlaps shared.sql:10:5 -> 3+1 = 4

	merged := Merge(a, b)

	if merged.Positions["a.sql"]["100:50"] != 7 {
		t.Errorf("a.sql:100:50 = %d, want 7", merged.Positions["a.sql"]["100:50"])
	}
	if merged.Positions["a.sql"]["200:60"] != 1 {
		t.Errorf("a.sql:200:60 = %d, want 1", merged.Positions["a.sql"]["200:60"])
	}
	if merged.Positions["b.sql"]["300:70"] != 4 {
		t.Errorf("b.sql:300:70 = %d, want 4", merged.Positions["b.sql"]["300:70"])
	}
	if merged.Positions["shared.sql"]["10:5"] != 4 {
		t.Errorf("shared.sql:10:5 = %d, want 4", merged.Positions["shared.sql"]["10:5"])
	}
}

func TestMerge_SumsHitCounts(t *testing.T) {
	a := NewCoverage()
	a.AddPosition("f.sql", 0, 10, 1)

	b := NewCoverage()
	b.AddPosition("f.sql", 0, 10, 1)

	c := NewCoverage()
	c.AddPosition("f.sql", 0, 10, 1)

	merged := Merge(a, b, c)

	got := merged.Positions["f.sql"]["0:10"]
	if got != 3 {
		t.Errorf("f.sql:0:10 = %d, want 3", got)
	}
}

func TestMerge_Empty(t *testing.T) {
	merged := Merge()
	if merged == nil {
		t.Fatal("Merge() returned nil")
	}
	if len(merged.Positions) != 0 {
		t.Errorf("empty merge produced positions: %v", merged.Positions)
	}
	if merged.Version != NewCoverage().Version {
		t.Errorf("empty merge Version = %q, want %q", merged.Version, NewCoverage().Version)
	}
}

func TestMerge_NilInputs(t *testing.T) {
	a := NewCoverage()
	a.AddPosition("f.sql", 0, 1, 1)

	merged := Merge(nil, a, nil)
	if merged.Positions["f.sql"]["0:1"] != 1 {
		t.Errorf("nil-aware merge lost a's hit count: got %d, want 1",
			merged.Positions["f.sql"]["0:1"])
	}
}

func TestMerge_DoesNotMutateInputs(t *testing.T) {
	a := NewCoverage()
	a.AddPosition("f.sql", 0, 1, 2)
	b := NewCoverage()
	b.AddPosition("f.sql", 0, 1, 3)

	_ = Merge(a, b)

	if a.Positions["f.sql"]["0:1"] != 2 {
		t.Errorf("Merge mutated a: got %d, want 2", a.Positions["f.sql"]["0:1"])
	}
	if b.Positions["f.sql"]["0:1"] != 3 {
		t.Errorf("Merge mutated b: got %d, want 3", b.Positions["f.sql"]["0:1"])
	}
}

func TestMerge_VersionAndTimestamp(t *testing.T) {
	c := NewCoverage()
	c.Version = "legacy"
	c.AddPosition("f.sql", 0, 1, 1)

	merged := Merge(c)
	if merged.Version != "1.0" {
		t.Errorf("merged.Version = %q, want %q", merged.Version, "1.0")
	}
	if merged.Timestamp.IsZero() {
		t.Errorf("merged.Timestamp is zero; expected time.Now()")
	}
}
