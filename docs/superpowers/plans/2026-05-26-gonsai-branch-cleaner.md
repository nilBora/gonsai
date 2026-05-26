# gonsai — Interactive CLI Branch Cleaner Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a colorful, interactive Bubble Tea CLI that lists local git branches with metadata and lets the user multi-select and safely delete them.

**Architecture:** Three internal packages — `git` (exec layer), `protect` (detects protected branches via HEAD + remote default), and `tui` (Bubble Tea model with filtering, selection, and confirmation). The entrypoint (`main.go`) wires them together: check repo → detect protected branches → list branches → run TUI.

**Tech Stack:** Go 1.21+, `charmbracelet/bubbletea`, `charmbracelet/lipgloss`, `charmbracelet/bubbles` (textinput), `sahilm/fuzzy`

---

## File Map

| File | Responsibility |
|---|---|
| `main.go` | Bootstrap: repo check → detect protected → list branches → start TUI |
| `internal/git/git.go` | `Run`, `RunLines`, `IsGitRepo` — exec helpers |
| `internal/git/branch.go` | `Branch` struct, `ListBranches`, `parseBranchLine`, `parseAheadBehind` |
| `internal/git/delete.go` | `DeleteSafe` (`-d`), `DeleteForce` (`-D`) |
| `internal/protect/protect.go` | `DetectProtected` — current HEAD + remote default + fallback |
| `internal/tui/styles.go` | All Lipgloss styles and colors |
| `internal/tui/keys.go` | All key bindings via `bubbles/key` |
| `internal/tui/filter.go` | `filterBranches` — fuzzy + toggle filters (pure function) |
| `internal/tui/confirm.go` | `confirmState` — unmerged-delete confirmation dialog |
| `internal/tui/model.go` | Bubble Tea `Model`: `Init`, `Update`, `View`, `Run` |

---

## Task 1: Initialize Go Module and Dependencies

**Files:**
- Create: `go.mod`, `go.sum`

- [ ] **Step 1: Initialize the module**

```bash
cd /Users/nilborodulia/Sites/gonsai
go mod init gonsai
```

Expected output: `go: creating new go.mod: module gonsai`

- [ ] **Step 2: Install dependencies**

```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/sahilm/fuzzy@latest
```

Expected: each command prints `go: added github.com/charmbracelet/...`

- [ ] **Step 3: Commit**

```bash
git init
git add go.mod go.sum
git commit -m "chore: initialize Go module with Bubble Tea + Lipgloss + fuzzy deps"
```

---

## Task 2: Git Exec Layer (`internal/git/git.go`)

**Files:**
- Create: `internal/git/git.go`

- [ ] **Step 1: Create the file**

```go
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Run executes a git command and returns trimmed stdout.
// Returns a non-nil error (containing stderr) on failure.
func Run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// RunLines calls Run and splits the result on newlines.
// Returns nil on empty output.
func RunLines(args ...string) ([]string, error) {
	out, err := Run(args...)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// IsGitRepo reports whether the current directory is inside a git repository.
func IsGitRepo() bool {
	_, err := Run("rev-parse", "--is-inside-work-tree")
	return err == nil
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/git/...
```

Expected: no output (clean build)

- [ ] **Step 3: Commit**

```bash
git add internal/git/git.go
git commit -m "feat: add git exec helpers (Run, RunLines, IsGitRepo)"
```

---

## Task 3: Branch Model and Listing (`internal/git/branch.go`)

**Files:**
- Create: `internal/git/branch.go`
- Create: `internal/git/branch_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/git/branch_test.go
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
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/git/... -run TestParse -v
```

Expected: `cannot find package` or `undefined: parseBranchLine`

- [ ] **Step 3: Implement branch.go**

```go
// internal/git/branch.go
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

// parseBranchLine parses one line of for-each-ref output.
// Exported (capitalized first char) for table-driven tests.
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
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/git/... -run TestParse -v
```

Expected: all `TestParse*` tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/git/branch.go internal/git/branch_test.go
git commit -m "feat: add Branch model, ListBranches, and for-each-ref parsing"
```

---

## Task 4: Delete Helpers (`internal/git/delete.go`)

**Files:**
- Create: `internal/git/delete.go`

- [ ] **Step 1: Create the file**

```go
// internal/git/delete.go
package git

// DeleteSafe attempts to delete branch with git branch -d.
// Returns non-nil error if the branch is unmerged.
func DeleteSafe(name string) error {
	_, err := Run("branch", "-d", name)
	return err
}

// DeleteForce deletes branch with git branch -D, even if unmerged.
func DeleteForce(name string) error {
	_, err := Run("branch", "-D", name)
	return err
}
```

- [ ] **Step 2: Build to confirm compilation**

```bash
go build ./internal/git/...
```

Expected: no output

- [ ] **Step 3: Commit**

```bash
git add internal/git/delete.go
git commit -m "feat: add DeleteSafe and DeleteForce git helpers"
```

---

## Task 5: Protected Branch Detection (`internal/protect/protect.go`)

**Files:**
- Create: `internal/protect/protect.go`
- Create: `internal/protect/protect_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/protect/protect_test.go
package protect

import "testing"

func TestStripRemotePrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"origin/main", "main"},
		{"origin/develop", "develop"},
		{"main", "main"},
		{"", ""},
		{"  origin/master  ", "master"},
		{"upstream/main", "upstream/main"}, // only strips "origin/" prefix
	}
	for _, tc := range tests {
		got := stripRemotePrefix(tc.input)
		if got != tc.want {
			t.Errorf("stripRemotePrefix(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFallbackDefaultsContainsMainAndMaster(t *testing.T) {
	found := make(map[string]bool)
	for _, n := range fallbackDefaults {
		found[n] = true
	}
	for _, required := range []string{"main", "master"} {
		if !found[required] {
			t.Errorf("fallbackDefaults missing %q", required)
		}
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/protect/... -v
```

Expected: compilation error — `undefined: stripRemotePrefix`, `undefined: fallbackDefaults`

- [ ] **Step 3: Implement protect.go**

```go
// internal/protect/protect.go
package protect

import (
	"strings"

	"gonsai/internal/git"
)

var fallbackDefaults = []string{"main", "master", "develop"}

// DetectProtected returns the current branch name, the default branch name,
// and a set of all protected branch names (current + default).
func DetectProtected() (current, defaultBranch string, protected map[string]bool) {
	protected = make(map[string]bool)

	current = strings.TrimSpace(func() string {
		s, _ := git.Run("symbolic-ref", "--short", "HEAD")
		return s
	}())
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
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/protect/... -v
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/protect/protect.go internal/protect/protect_test.go
git commit -m "feat: add protected branch detection with remote+fallback logic"
```

---

## Task 6: TUI Styles (`internal/tui/styles.go`)

**Files:**
- Create: `internal/tui/styles.go`

- [ ] **Step 1: Create the file**

```go
// internal/tui/styles.go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorGreen  = lipgloss.AdaptiveColor{Light: "#16a34a", Dark: "#4ade80"}
	colorYellow = lipgloss.AdaptiveColor{Light: "#d97706", Dark: "#fbbf24"}
	colorRed    = lipgloss.AdaptiveColor{Light: "#dc2626", Dark: "#f87171"}
	colorCyan   = lipgloss.AdaptiveColor{Light: "#0891b2", Dark: "#22d3ee"}
	colorGray   = lipgloss.AdaptiveColor{Light: "#6b7280", Dark: "#9ca3af"}

	styleSelected  = lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	styleProtected = lipgloss.NewStyle().Foreground(colorGray)
	styleMerged    = lipgloss.NewStyle().Foreground(colorGreen)
	styleUnmerged  = lipgloss.NewStyle().Foreground(colorYellow)
	styleHelp      = lipgloss.NewStyle().Foreground(colorGray).Faint(true)
	styleCursor    = lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
	styleStatus    = lipgloss.NewStyle().Foreground(colorCyan)
	styleHeader    = lipgloss.NewStyle().Bold(true).Foreground(colorCyan)
	styleCounter   = lipgloss.NewStyle().Foreground(colorGray)
	styleError     = lipgloss.NewStyle().Foreground(colorRed)
	styleBorder    = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorRed).
		Padding(1, 3)
)
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/tui/...
```

Expected: may fail with "no Go files" — that is fine, will be resolved in later tasks.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/styles.go
git commit -m "feat: add Lipgloss TUI color palette and styles"
```

---

## Task 7: Key Bindings (`internal/tui/keys.go`)

**Files:**
- Create: `internal/tui/keys.go`

- [ ] **Step 1: Create the file**

```go
// internal/tui/keys.go
package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up           key.Binding
	Down         key.Binding
	Toggle       key.Binding
	SelectAll    key.Binding
	DeselectAll  key.Binding
	Filter       key.Binding
	ToggleMerged key.Binding
	CycleOlder   key.Binding
	Delete       key.Binding
	Quit         key.Binding
	Back         key.Binding
}

var keys = keyMap{
	Up:           key.NewBinding(key.WithKeys("up", "k")),
	Down:         key.NewBinding(key.WithKeys("down", "j")),
	Toggle:       key.NewBinding(key.WithKeys(" ")),
	SelectAll:    key.NewBinding(key.WithKeys("a")),
	DeselectAll:  key.NewBinding(key.WithKeys("n")),
	Filter:       key.NewBinding(key.WithKeys("/")),
	ToggleMerged: key.NewBinding(key.WithKeys("m")),
	CycleOlder:   key.NewBinding(key.WithKeys("o")),
	Delete:       key.NewBinding(key.WithKeys("enter")),
	Quit:         key.NewBinding(key.WithKeys("q", "ctrl+c")),
	Back:         key.NewBinding(key.WithKeys("esc")),
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/tui/keys.go
git commit -m "feat: add TUI key bindings"
```

---

## Task 8: Fuzzy Filter + Toggles (`internal/tui/filter.go`)

**Files:**
- Create: `internal/tui/filter.go`
- Create: `internal/tui/filter_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/tui/filter_test.go
package tui

import (
	"testing"
	"time"

	"gonsai/internal/git"
)

func testBranches() []git.Branch {
	now := time.Now()
	return []git.Branch{
		{Name: "feature/login",  IsMerged: true,  LastCommit: now.Add(-45 * 24 * time.Hour)},
		{Name: "feature/logout", IsMerged: false, LastCommit: now.Add(-60 * 24 * time.Hour)},
		{Name: "fix/typo",       IsMerged: true,  LastCommit: now.Add(-5 * 24 * time.Hour)},
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
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/tui/... -run TestFilter -v
```

Expected: compilation error — `undefined: filterBranches`

- [ ] **Step 3: Implement filter.go**

```go
// internal/tui/filter.go
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
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/tui/... -run TestFilter -v
```

Expected: all `TestFilter*` tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/filter.go internal/tui/filter_test.go
git commit -m "feat: add fuzzy branch filter with merged/older-than toggles"
```

---

## Task 9: Confirmation Dialog (`internal/tui/confirm.go`)

**Files:**
- Create: `internal/tui/confirm.go`

- [ ] **Step 1: Create the file**

```go
// internal/tui/confirm.go
package tui

import (
	"fmt"
	"strings"

	"gonsai/internal/git"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// confirmState holds the state for the unmerged-delete confirmation dialog.
type confirmState struct {
	safe     []git.Branch
	unmerged []git.Branch
	input    textinput.Model
}

// newConfirmState creates a focused confirmation dialog for the given branch sets.
func newConfirmState(safe, unmerged []git.Branch) confirmState {
	ti := textinput.New()
	ti.Placeholder = "yes"
	ti.CharLimit = 8
	ti.Focus()
	return confirmState{safe: safe, unmerged: unmerged, input: ti}
}

// update forwards a Bubble Tea message to the textinput.
func (c confirmState) update(msg tea.Msg) (confirmState, tea.Cmd) {
	var cmd tea.Cmd
	c.input, cmd = c.input.Update(msg)
	return c, cmd
}

// confirmed returns true only when the user typed exactly "yes".
func (c confirmState) confirmed() bool {
	return strings.TrimSpace(c.input.Value()) == "yes"
}

// view renders the confirmation dialog box.
func (c confirmState) view() string {
	names := make([]string, len(c.unmerged))
	for i, b := range c.unmerged {
		names[i] = "  • " + b.Name
	}
	body := fmt.Sprintf(
		"%s\n\n%s\n\nType %s to confirm, or press Esc to cancel:\n\n%s",
		styleUnmerged.Render(fmt.Sprintf(
			"⚠  %d unmerged branch(es) cannot be safely deleted:", len(c.unmerged),
		)),
		styleError.Render(strings.Join(names, "\n")),
		styleSelected.Render("'yes'"),
		c.input.View(),
	)
	return styleBorder.Render(body)
}
```

- [ ] **Step 2: Build to confirm compilation**

```bash
go build ./internal/tui/...
```

Expected: may still fail with missing `Model` — that is fine.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/confirm.go
git commit -m "feat: add unmerged-delete confirmation dialog component"
```

---

## Task 10: Bubble Tea Model (`internal/tui/model.go`)

**Files:**
- Create: `internal/tui/model.go`

- [ ] **Step 1: Create the file**

```go
// internal/tui/model.go
package tui

import (
	"fmt"
	"strings"
	"time"

	"gonsai/internal/git"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model is the top-level Bubble Tea model for gonsai.
type Model struct {
	branches      []git.Branch
	visible       []int
	selected      map[string]bool
	cursor        int
	filter        string
	filterMode    bool
	filterInput   textinput.Model
	onlyMerged    bool
	onlyOlderDays int // 0 = off, else days threshold
	defaultBranch string
	confirm       *confirmState
	status        string
	width         int
	height        int
}

// NewModel initialises a Model with the given branches and default branch name.
// Branches must already have IsProtected set by the caller.
func NewModel(branches []git.Branch, defaultBranch string) Model {
	fi := textinput.New()
	fi.Placeholder = "filter..."
	m := Model{
		branches:      branches,
		selected:      make(map[string]bool),
		defaultBranch: defaultBranch,
		filterInput:   fi,
	}
	m.visible = filterBranches(m.branches, "", false, 0)
	return m
}

// Run starts the Bubble Tea program in alt-screen mode.
func Run(branches []git.Branch, defaultBranch string) error {
	p := tea.NewProgram(NewModel(branches, defaultBranch), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// refilter recomputes the visible slice and clamps cursor.
func (m Model) refilter() Model {
	m.visible = filterBranches(m.branches, m.filter, m.onlyMerged, m.onlyOlderDays)
	if len(m.visible) == 0 {
		m.cursor = 0
	} else if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
	return m
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.confirm != nil {
		return m.updateConfirm(msg)
	}
	if m.filterMode {
		return m.updateFilterMode(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.visible)-1 {
				m.cursor++
			}
		case key.Matches(msg, keys.Toggle):
			m = m.toggleCurrent()
		case key.Matches(msg, keys.SelectAll):
			m = m.selectAllVisible()
		case key.Matches(msg, keys.DeselectAll):
			m.selected = make(map[string]bool)
		case key.Matches(msg, keys.Filter):
			m.filterMode = true
			m.filterInput.SetValue("")
			cmd := m.filterInput.Focus()
			return m, cmd
		case key.Matches(msg, keys.ToggleMerged):
			m.onlyMerged = !m.onlyMerged
			m = m.refilter()
		case key.Matches(msg, keys.CycleOlder):
			switch m.onlyOlderDays {
			case 0:
				m.onlyOlderDays = 30
			case 30:
				m.onlyOlderDays = 90
			case 90:
				m.onlyOlderDays = 180
			default:
				m.onlyOlderDays = 0
			}
			m = m.refilter()
		case key.Matches(msg, keys.Delete):
			return m.handleDelete()
		}
	}
	return m, nil
}

func (m Model) updateFilterMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(kMsg, keys.Back):
			m.filterMode = false
			m.filterInput.Blur()
			m.filterInput.SetValue("")
			m.filter = ""
			m = m.refilter()
			return m, nil
		case key.Matches(kMsg, keys.Delete): // enter commits the filter
			m.filterMode = false
			m.filterInput.Blur()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.filter = m.filterInput.Value()
	m = m.refilter()
	return m, cmd
}

func (m Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(kMsg, keys.Quit):
			return m, tea.Quit
		case key.Matches(kMsg, keys.Back):
			m.confirm = nil
			m.status = styleHelp.Render("Delete cancelled.")
			return m, nil
		case key.Matches(kMsg, keys.Delete): // enter
			if m.confirm.confirmed() {
				return m.executeDelete(m.confirm.safe, m.confirm.unmerged, true)
			}
			m.confirm = nil
			m.status = styleHelp.Render("Cancelled — type 'yes' to confirm force-delete.")
			return m, nil
		}
	}
	updated, cmd := m.confirm.update(msg)
	m.confirm = &updated
	return m, cmd
}

func (m Model) toggleCurrent() Model {
	if len(m.visible) == 0 {
		return m
	}
	b := m.branches[m.visible[m.cursor]]
	if b.IsProtected {
		return m
	}
	if m.selected[b.Name] {
		delete(m.selected, b.Name)
	} else {
		m.selected[b.Name] = true
	}
	return m
}

func (m Model) selectAllVisible() Model {
	for _, i := range m.visible {
		b := m.branches[i]
		if !b.IsProtected {
			m.selected[b.Name] = true
		}
	}
	return m
}

func (m Model) handleDelete() (tea.Model, tea.Cmd) {
	var safe, unmerged []git.Branch
	for i := range m.branches {
		b := m.branches[i]
		if !m.selected[b.Name] || b.IsProtected {
			continue
		}
		if b.IsMerged {
			safe = append(safe, b)
		} else {
			unmerged = append(unmerged, b)
		}
	}

	if len(safe)+len(unmerged) == 0 {
		m.status = styleHelp.Render("No branches selected. Press [space] to select.")
		return m, nil
	}
	if len(unmerged) > 0 {
		cs := newConfirmState(safe, unmerged)
		m.confirm = &cs
		return m, textinput.Blink
	}
	return m.executeDelete(safe, nil, false)
}

func (m Model) executeDelete(safe, unmerged []git.Branch, forceUnmerged bool) (tea.Model, tea.Cmd) {
	m.confirm = nil
	var deleted, failed []string

	for _, b := range safe {
		if err := git.DeleteSafe(b.Name); err != nil {
			failed = append(failed, b.Name)
		} else {
			deleted = append(deleted, b.Name)
		}
	}
	if forceUnmerged {
		for _, b := range unmerged {
			if err := git.DeleteForce(b.Name); err != nil {
				failed = append(failed, b.Name)
			} else {
				deleted = append(deleted, b.Name)
			}
		}
	}

	deletedSet := make(map[string]bool, len(deleted))
	for _, n := range deleted {
		deletedSet[n] = true
		delete(m.selected, n)
	}
	remaining := make([]git.Branch, 0, len(m.branches))
	for _, b := range m.branches {
		if !deletedSet[b.Name] {
			remaining = append(remaining, b)
		}
	}
	m.branches = remaining
	m = m.refilter()

	if len(failed) > 0 {
		m.status = styleError.Render(
			fmt.Sprintf("Deleted %d, failed: %s", len(deleted), strings.Join(failed, ", ")),
		)
	} else {
		m.status = styleStatus.Render(fmt.Sprintf("✓ Deleted %d branch(es).", len(deleted)))
	}
	return m, nil
}

func (m Model) View() string {
	if m.confirm != nil {
		if m.width > 0 && m.height > 0 {
			return lipgloss.Place(m.width, m.height,
				lipgloss.Center, lipgloss.Center,
				m.confirm.view())
		}
		return m.confirm.view()
	}

	var sb strings.Builder

	// Header
	repoDir, _ := git.Run("rev-parse", "--show-toplevel")
	sb.WriteString(styleHeader.Render(
		fmt.Sprintf(" gonsai  %s  default: %s", repoDir, m.defaultBranch),
	))
	sb.WriteString("\n\n")

	// Stats bar
	total := len(m.branches)
	selCount := len(m.selected)
	var mergedCount, staleCount int
	now := time.Now()
	for _, b := range m.branches {
		if b.IsMerged {
			mergedCount++
		}
		if now.Sub(b.LastCommit) > 30*24*time.Hour {
			staleCount++
		}
	}
	var filters []string
	if m.onlyMerged {
		filters = append(filters, "merged")
	}
	if m.onlyOlderDays > 0 {
		filters = append(filters, fmt.Sprintf(">%dd", m.onlyOlderDays))
	}
	filterTag := ""
	if len(filters) > 0 {
		filterTag = "  [" + strings.Join(filters, "+") + "]"
	}
	sb.WriteString(styleCounter.Render(fmt.Sprintf(
		"  %d branches · %d selected · %d merged · %d stale (>30d)%s",
		total, selCount, mergedCount, staleCount, filterTag,
	)))
	sb.WriteString("\n\n")

	// Branch rows
	for j, i := range m.visible {
		sb.WriteString(m.renderRow(m.branches[i], j == m.cursor))
		sb.WriteString("\n")
	}
	if len(m.visible) == 0 {
		sb.WriteString(styleHelp.Render("  (no branches match filters)"))
		sb.WriteString("\n")
	}

	// Filter bar
	sb.WriteString("\n")
	if m.filterMode {
		sb.WriteString(styleStatus.Render("  /") + m.filterInput.View() + "\n")
	} else if m.filter != "" {
		sb.WriteString(styleHelp.Render(
			fmt.Sprintf("  filter: %q  (press / to edit, esc to clear)", m.filter),
		))
		sb.WriteString("\n")
	}

	// Status line
	if m.status != "" {
		sb.WriteString("\n")
		sb.WriteString("  " + m.status)
		sb.WriteString("\n")
	}

	// Help bar
	sb.WriteString("\n")
	sb.WriteString(styleHelp.Render(
		"  [↑/k ↓/j] move  [space] toggle  [a] all  [n] none  [/] filter  [m] merged  [o] older  [enter] delete  [q] quit",
	))
	return sb.String()
}

func (m Model) renderRow(b git.Branch, cursor bool) string {
	cursorMark := "  "
	if cursor {
		cursorMark = styleCursor.Render("▶ ")
	}

	lockIcon := "   "
	if b.IsProtected {
		lockIcon = "🔒 "
	}

	checkbox := "[ ]"
	if b.IsProtected {
		checkbox = styleProtected.Render("[·]")
	} else if m.selected[b.Name] {
		checkbox = styleSelected.Render("[✓]")
	}

	// Truncate long names
	name := b.Name
	if len(name) > 28 {
		name = name[:25] + "..."
	}
	nameCol := fmt.Sprintf("%-28s", name)
	if b.IsProtected {
		nameCol = styleProtected.Render(nameCol)
	}

	ageCol := styleHelp.Render(fmt.Sprintf("%-18s", truncate(b.LastCommitRel, 18)))
	abCol := styleHelp.Render(fmt.Sprintf("↑%-2d ↓%-2d", b.Ahead, b.Behind))

	var statusCol string
	switch {
	case b.IsCurrent:
		statusCol = styleStatus.Render("  HEAD    ")
	case b.IsMerged:
		statusCol = styleMerged.Render("  merged  ")
	default:
		statusCol = styleUnmerged.Render(" unmerged ")
	}

	return fmt.Sprintf("  %s%s%s  %s  %s  %s  %s",
		cursorMark, lockIcon, checkbox, nameCol, ageCol, abCol, statusCol)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
```

- [ ] **Step 2: Build the tui package to confirm compilation**

```bash
go build ./internal/tui/...
```

Expected: clean build, no errors

- [ ] **Step 3: Run all tests**

```bash
go test ./...
```

Expected: all tests PASS

- [ ] **Step 4: Commit**

```bash
git add internal/tui/model.go
git commit -m "feat: implement Bubble Tea model with filtering, selection, and delete flow"
```

---

## Task 11: Entry Point (`main.go`)

**Files:**
- Create: `main.go`

- [ ] **Step 1: Create the file**

```go
// main.go
package main

import (
	"fmt"
	"os"

	"gonsai/internal/git"
	"gonsai/internal/protect"
	"gonsai/internal/tui"

	"github.com/charmbracelet/lipgloss"
)

var errStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#f87171")).Bold(true)

func main() {
	if !git.IsGitRepo() {
		fmt.Fprintln(os.Stderr, errStyle.Render("✗ Not inside a git repository."))
		os.Exit(1)
	}

	_, defaultBranch, protectedSet := protect.DetectProtected()

	branches, err := git.ListBranches(defaultBranch)
	if err != nil {
		fmt.Fprintln(os.Stderr, errStyle.Render("✗ Failed to list branches: "+err.Error()))
		os.Exit(1)
	}

	for i := range branches {
		if protectedSet[branches[i].Name] {
			branches[i].IsProtected = true
		}
	}

	if err := tui.Run(branches, defaultBranch); err != nil {
		fmt.Fprintln(os.Stderr, errStyle.Render("✗ "+err.Error()))
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Build the binary**

```bash
go build -o gonsai ./
```

Expected: produces a `gonsai` binary in the current directory, no errors

- [ ] **Step 3: Run all tests one final time**

```bash
go test ./...
```

Expected: all tests PASS

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "feat: add main entrypoint — gonsai binary complete"
```

---

## Task 12: Smoke Test

**Files:**
- Create: `scripts/test-repo.sh`

- [ ] **Step 1: Create smoke-test script**

```bash
#!/usr/bin/env bash
# Creates a throwaway git repo in /tmp and launches gonsai against it.
set -e

REPO=/tmp/gonsai-smoke-$$
BINARY="$(cd "$(dirname "$0")/.." && pwd)/gonsai"

mkdir -p "$REPO"
cd "$REPO"
git init -q
git config user.email "test@test.com"
git config user.name "Test"
git commit --allow-empty -q -m "init"

# Create 8 merged feature branches
for i in $(seq 1 8); do
  git branch "feature/old-$i"
done

# Create 1 unmerged branch with a commit
git checkout -q -b experiment/wip
echo "wip" > wip.txt
git add wip.txt
git commit -q -m "work in progress"
git checkout -q main 2>/dev/null || git checkout -q master

echo "Smoke-test repo created at $REPO"
echo "Launching gonsai..."
"$BINARY"

rm -rf "$REPO"
```

Save to `scripts/test-repo.sh`, then:

```bash
chmod +x scripts/test-repo.sh
```

- [ ] **Step 2: Build the binary if not already built**

```bash
go build -o gonsai ./
```

- [ ] **Step 3: Run the smoke test**

```bash
./scripts/test-repo.sh
```

Expected behavior:
- TUI opens showing 9 branches
- `main`/`master` shows `🔒[·]` (protected, cannot be selected)
- `feature/old-1` through `feature/old-8` show `merged` in green
- `experiment/wip` shows `unmerged` in yellow
- Pressing `space` on a merged branch checks it `[✓]`
- Pressing `a` selects all non-protected branches
- Pressing `enter` with unmerged selected shows red confirmation dialog
- Typing `yes` + enter deletes; Esc cancels
- Pressing `/` + typing `old` filters to feature branches
- Pressing `m` shows only merged branches
- Pressing `o` cycles older-than filter (off → 30d → 90d → 180d)
- Pressing `q` quits

- [ ] **Step 4: Commit**

```bash
git add scripts/test-repo.sh
git commit -m "test: add smoke-test script for manual TUI verification"
```

---

## Verification Checklist

- [ ] `go build -o gonsai ./` — binary compiles
- [ ] `go test ./...` — all unit tests pass
- [ ] `go vet ./...` — no vet warnings
- [ ] Smoke test: protected branches show `🔒` and cannot be selected
- [ ] Smoke test: unmerged delete triggers `'yes'` confirmation, Esc cancels
- [ ] Smoke test: `/` filter narrows list in real-time, `esc` clears
- [ ] Smoke test: `m` toggle shows only merged; `o` cycles age filter
- [ ] Edge case: run outside git repo → red error message + exit 1
- [ ] Edge case: repo with no remote → default branch detected from local `main`/`master`
