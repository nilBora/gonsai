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
