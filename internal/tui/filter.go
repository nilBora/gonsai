package tui

import (
	"time"

	"gonsai/internal/git"

	"github.com/sahilm/fuzzy"
)

// filterBranches returns indices into branches that match all active filters.
// Results preserve the original (oldest-first) ordering of branches.
func filterBranches(branches []git.Branch, query string, onlyMerged bool, olderDays int) []int {
	now := time.Now()
	var candidates []int
	for i, b := range branches {
		if onlyMerged && !b.IsMerged {
			continue
		}
		if olderDays > 0 && now.Sub(b.LastCommit) < time.Duration(olderDays)*24*time.Hour {
			continue
		}
		candidates = append(candidates, i)
	}

	if query == "" {
		return candidates
	}

	// Build a parallel name list so fuzzy indices map back to candidates.
	names := make([]string, len(candidates))
	for j, i := range candidates {
		names[j] = branches[i].Name
	}

	matches := fuzzy.Find(query, names)
	result := make([]int, 0, len(matches))
	for _, m := range matches {
		result = append(result, candidates[m.Index])
	}
	return result
}
