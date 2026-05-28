package tui

import (
	"fmt"
	"strings"

	"gonsai/app/internal/git"

	tea "github.com/charmbracelet/bubbletea"
)

// confirmState holds the state for the unmerged-delete confirmation dialog.
type confirmState struct {
	safe     []git.Branch
	unmerged []git.Branch
	choice   int // 0 = Yes (force-delete), 1 = No (cancel)
}

// newConfirmState creates a confirmation dialog defaulting to Yes.
func newConfirmState(safe, unmerged []git.Branch) confirmState {
	return confirmState{safe: safe, unmerged: unmerged, choice: 0}
}

// update handles left/right/y/n keys to move the selection.
func (c confirmState) update(msg tea.Msg) (confirmState, tea.Cmd) {
	kMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return c, nil
	}
	switch kMsg.String() {
	case "left", "h", "y", "Y":
		c.choice = 0
	case "right", "l", "n", "N", "tab":
		c.choice = 1
	}
	return c, nil
}

// confirmed returns true when the Yes button is selected.
func (c confirmState) confirmed() bool { return c.choice == 0 }

// view renders the confirmation dialog box.
func (c confirmState) view() string {
	names := make([]string, len(c.unmerged))
	for i, b := range c.unmerged {
		names[i] = "  • " + b.Name
	}

	var yesBtn, noBtn string
	if c.choice == 0 {
		yesBtn = styleSelected.Render("▶ Yes ◀")
		noBtn = styleHelp.Render("  No  ")
	} else {
		yesBtn = styleHelp.Render("  Yes  ")
		noBtn = styleError.Render("▶ No ◀")
	}

	body := fmt.Sprintf(
		"%s\n\n%s\n\nForce-delete %d unmerged branch(es)?\n\n  %s    %s\n\n%s",
		styleUnmerged.Render(fmt.Sprintf(
			"⚠  %d unmerged branch(es) cannot be safely deleted:", len(c.unmerged),
		)),
		styleError.Render(strings.Join(names, "\n")),
		len(c.unmerged),
		yesBtn,
		noBtn,
		styleHelp.Render("←/→ choose · enter confirm · esc cancel · y/n shortcut"),
	)
	return styleBorder.Render(body)
}
