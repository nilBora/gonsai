package tui

import (
	"fmt"
	"strings"
	"time"

	"gonsai/app/internal/git"

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
	offset        int // first visible index into m.visible (viewport scroll)
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

// refilter recomputes the visible slice, clamps cursor, and adjusts viewport.
func (m Model) refilter() Model {
	m.visible = filterBranches(m.branches, m.filter, m.onlyMerged, m.onlyOlderDays)
	if len(m.visible) == 0 {
		m.cursor = 0
	} else if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
	return m.adjustViewport()
}

const chromeRows = 12

func (m Model) listRows() int {
	if m.height <= 0 {
		return len(m.visible)
	}
	r := m.height - chromeRows
	if r < 3 {
		r = 3
	}
	return r
}

func (m Model) adjustViewport() Model {
	rows := m.listRows()
	if len(m.visible) == 0 {
		m.offset = 0
		return m
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+rows {
		m.offset = m.cursor - rows + 1
	}
	maxOffset := len(m.visible) - rows
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.offset > maxOffset {
		m.offset = maxOffset
	}
	if m.offset < 0 {
		m.offset = 0
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
		m = m.adjustViewport()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
				m = m.adjustViewport()
			}
		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.visible)-1 {
				m.cursor++
				m = m.adjustViewport()
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

	// Branch rows (windowed viewport)
	rows := m.listRows()
	end := m.offset + rows
	if end > len(m.visible) {
		end = len(m.visible)
	}
	for j := m.offset; j < end; j++ {
		i := m.visible[j]
		sb.WriteString(m.renderRow(m.branches[i], j == m.cursor))
		sb.WriteString("\n")
	}
	if len(m.visible) == 0 {
		sb.WriteString(styleHelp.Render("  (no branches match filters)"))
		sb.WriteString("\n")
	}
	if len(m.visible) > 0 && (m.offset > 0 || end < len(m.visible)) {
		sb.WriteString(styleHelp.Render(fmt.Sprintf(
			"  showing %d–%d of %d", m.offset+1, end, len(m.visible),
		)))
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
