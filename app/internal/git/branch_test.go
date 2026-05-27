package git

import (
	"testing"
	"time"
)

func TestParseBranchLine(t *testing.T) {
	mergedSet := map[string]bool{"feature/login": true}
	line := "feature/login|1716768000|3 weeks ago|abc1234|Add login form"
	b, ok := parseBranchLine(line, "main", mergedSet, "main")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if b.Name != "feature/login" {
		t.Errorf("Name: got %q, want %q", b.Name, "feature/login")
	}
	want := time.Unix(1716768000, 0)
	if !b.LastCommit.Equal(want) {
		t.Errorf("LastCommit: got %v, want %v", b.LastCommit, want)
	}
	if b.LastCommitRel != "3 weeks ago" {
		t.Errorf("LastCommitRel: got %q", b.LastCommitRel)
	}
	if b.ShortHash != "abc1234" {
		t.Errorf("ShortHash: got %q", b.ShortHash)
	}
	if b.Subject != "Add login form" {
		t.Errorf("Subject: got %q", b.Subject)
	}
	if !b.IsMerged {
		t.Error("expected IsMerged=true")
	}
	if b.IsCurrent {
		t.Error("expected IsCurrent=false (currentBranch=main, not feature/login)")
	}
}

func TestParseBranchLineCurrentBranch(t *testing.T) {
	line := "main|1716768000|2 hours ago|abc1234|Initial commit"
	b, ok := parseBranchLine(line, "main", nil, "main")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !b.IsCurrent {
		t.Error("expected IsCurrent=true when name matches currentBranch")
	}
}

func TestParseBranchLineMalformed(t *testing.T) {
	_, ok := parseBranchLine("only-one-field", "main", nil, "main")
	if ok {
		t.Error("expected ok=false for line with fewer than 5 fields")
	}
}

func TestParseBranchLineSubjectWithPipes(t *testing.T) {
	// Subject may contain "|" — SplitN(5) ensures only 4 splits.
	line := "feat/x|1716768000|1 day ago|abc1234|Merge: a|b into main"
	b, ok := parseBranchLine(line, "", nil, "")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if b.Subject != "Merge: a|b into main" {
		t.Errorf("Subject: got %q, want %q", b.Subject, "Merge: a|b into main")
	}
}

func TestParseAheadBehind(t *testing.T) {
	tests := []struct {
		input      string
		wantAhead  int
		wantBehind int
	}{
		{"3\t1", 1, 3},
		{"0\t0", 0, 0},
		{"10\t5", 5, 10},
		{"", 0, 0},
		{"bad", 0, 0},
	}
	for _, tc := range tests {
		ahead, behind := parseAheadBehind(tc.input)
		if ahead != tc.wantAhead || behind != tc.wantBehind {
			t.Errorf("input=%q: got ahead=%d behind=%d, want ahead=%d behind=%d",
				tc.input, ahead, behind, tc.wantAhead, tc.wantBehind)
		}
	}
}
