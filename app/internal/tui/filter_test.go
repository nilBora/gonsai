package tui

import (
	"testing"
	"time"

	"gonsai/app/internal/git"
)

func testBranches() []git.Branch {
	now := time.Now()
	return []git.Branch{
		{Name: "feature/login", IsMerged: true, LastCommit: now.Add(-45 * 24 * time.Hour)},
		{Name: "feature/logout", IsMerged: false, LastCommit: now.Add(-60 * 24 * time.Hour)},
		{Name: "fix/typo", IsMerged: true, LastCommit: now.Add(-5 * 24 * time.Hour)},
		{Name: "experiment/wip", IsMerged: false, LastCommit: now.Add(-10 * 24 * time.Hour)},
	}
}

func TestFilterBranchesNoFilter(t *testing.T) {
	result := filterBranches(testBranches(), "", false, 0)
	if len(result) != 4 {
		t.Errorf("expected 4 results, got %d", len(result))
	}
}

func TestFilterBranchesOnlyMerged(t *testing.T) {
	branches := testBranches()
	result := filterBranches(branches, "", true, 0)
	if len(result) != 2 {
		t.Fatalf("expected 2 merged branches, got %d", len(result))
	}
	for _, i := range result {
		if !branches[i].IsMerged {
			t.Errorf("branch %q should be merged", branches[i].Name)
		}
	}
}

func TestFilterBranchesOlderThan30Days(t *testing.T) {
	// feature/login (45d) and feature/logout (60d) are older than 30d
	branches := testBranches()
	result := filterBranches(branches, "", false, 30)
	if len(result) != 2 {
		t.Fatalf("expected 2 branches older than 30d, got %d", len(result))
	}
}

func TestFilterBranchesFuzzyLogin(t *testing.T) {
	branches := testBranches()
	result := filterBranches(branches, "login", false, 0)
	if len(result) != 1 {
		t.Fatalf("expected 1 fuzzy match for 'login', got %d", len(result))
	}
	if branches[result[0]].Name != "feature/login" {
		t.Errorf("expected feature/login, got %q", branches[result[0]].Name)
	}
}

func TestFilterBranchesCombinedMergedAndOld(t *testing.T) {
	// merged + older than 30d → only feature/login (45d, merged)
	branches := testBranches()
	result := filterBranches(branches, "", true, 30)
	if len(result) != 1 {
		t.Fatalf("expected 1 branch, got %d", len(result))
	}
	if branches[result[0]].Name != "feature/login" {
		t.Errorf("expected feature/login, got %q", branches[result[0]].Name)
	}
}

func TestFilterBranchesEmptyQuery(t *testing.T) {
	branches := testBranches()
	result := filterBranches(branches, "zzznomatch", false, 0)
	if len(result) != 0 {
		t.Errorf("expected 0 results for unmatched query, got %d", len(result))
	}
}
