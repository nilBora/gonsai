package git

import (
	"strconv"
	"strings"
	"time"
)

// Branch holds metadata about a single local git branch.
type Branch struct {
	Name          string
	IsCurrent     bool
	IsProtected   bool
	IsMerged      bool
	LastCommit    time.Time
	LastCommitRel string
	Ahead         int
	Behind        int
	ShortHash     string
	Subject       string
}

// forEachRefFormat produces pipe-separated fields: name|unixtime|reltime|shorthash|subject
const forEachRefFormat = "%(refname:short)|%(committerdate:unix)|%(committerdate:relative)|%(objectname:short)|%(subject)"

// ListBranches returns all local branches sorted oldest-first.
// defaultBranch is used to compute IsMerged and ahead/behind counts.
func ListBranches(defaultBranch string) ([]Branch, error) {
	currentBranch, _ := Run("symbolic-ref", "--short", "HEAD")
	currentBranch = strings.TrimSpace(currentBranch)

	lines, err := RunLines("for-each-ref", "refs/heads/",
		"--format="+forEachRefFormat, "--sort=committerdate")
	if err != nil {
		return nil, err
	}

	mergedSet := make(map[string]bool)
	if defaultBranch != "" {
		mergedLines, _ := RunLines("branch", "--merged", defaultBranch)
		for _, l := range mergedLines {
			name := strings.TrimSpace(strings.TrimPrefix(l, "*"))
			name = strings.TrimSpace(name)
			mergedSet[name] = true
		}
	}

	var branches []Branch
	for _, line := range lines {
		if line == "" {
			continue
		}
		b, ok := parseBranchLine(line, currentBranch, mergedSet, defaultBranch)
		if !ok {
			continue
		}
		branches = append(branches, b)
	}
	return branches, nil
}

// parseBranchLine parses one line of for-each-ref output into a Branch.
func parseBranchLine(line, currentBranch string, mergedSet map[string]bool, defaultBranch string) (Branch, bool) {
	// SplitN(5) so subjects containing "|" are preserved intact.
	parts := strings.SplitN(line, "|", 5)
	if len(parts) < 5 {
		return Branch{}, false
	}
	name := parts[0]
	unixStr := parts[1]
	relTime := parts[2]
	hash := parts[3]
	subject := parts[4]

	var lastCommit time.Time
	if ts, err := strconv.ParseInt(unixStr, 10, 64); err == nil && ts > 0 {
		lastCommit = time.Unix(ts, 0)
	}

	ahead, behind := computeAheadBehind(defaultBranch, name)

	return Branch{
		Name:          name,
		IsCurrent:     name == currentBranch,
		IsMerged:      mergedSet[name],
		LastCommit:    lastCommit,
		LastCommitRel: relTime,
		Ahead:         ahead,
		Behind:        behind,
		ShortHash:     hash,
		Subject:       subject,
	}, true
}

// computeAheadBehind runs rev-list to count commits ahead/behind base.
func computeAheadBehind(base, branch string) (ahead, behind int) {
	if base == "" || base == branch {
		return 0, 0
	}
	out, err := Run("rev-list", "--left-right", "--count", base+"..."+branch)
	if err != nil {
		return 0, 0
	}
	return parseAheadBehind(out)
}

// parseAheadBehind parses "{behind}\t{ahead}" output from rev-list --left-right --count.
func parseAheadBehind(out string) (ahead, behind int) {
	parts := strings.Fields(strings.TrimSpace(out))
	if len(parts) != 2 {
		return 0, 0
	}
	behind, _ = strconv.Atoi(parts[0])
	ahead, _ = strconv.Atoi(parts[1])
	return
}
