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
