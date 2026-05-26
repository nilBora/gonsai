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
