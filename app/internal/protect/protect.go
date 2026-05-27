package protect

import (
	"strings"

	"gonsai/app/internal/git"
)

var fallbackDefaults = []string{"main", "master", "develop"}

// DetectProtected returns the current branch name, the default branch name,
// and a set of all protected branch names (current + default).
func DetectProtected() (current, defaultBranch string, protected map[string]bool) {
	protected = make(map[string]bool)

	cur, _ := git.Run("symbolic-ref", "--short", "HEAD")
	current = strings.TrimSpace(cur)
	if current != "" {
		protected[current] = true
	}

	// Prefer remote-tracked default branch
	remote, err := git.Run("symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	if err == nil && remote != "" {
		defaultBranch = stripRemotePrefix(remote)
		if defaultBranch != "" {
			protected[defaultBranch] = true
			return
		}
	}

	// Fallback: first well-known name that exists locally
	localLines, _ := git.RunLines("branch", "--format=%(refname:short)")
	localSet := make(map[string]bool)
	for _, b := range localLines {
		localSet[strings.TrimSpace(b)] = true
	}
	for _, name := range fallbackDefaults {
		if localSet[name] {
			defaultBranch = name
			protected[name] = true
			return
		}
	}

	// Last resort: default = current
	defaultBranch = current
	return
}

// stripRemotePrefix removes the "origin/" prefix from a remote symbolic-ref shorthand.
func stripRemotePrefix(ref string) string {
	return strings.TrimPrefix(strings.TrimSpace(ref), "origin/")
}
