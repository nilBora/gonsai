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
