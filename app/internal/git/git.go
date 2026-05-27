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
