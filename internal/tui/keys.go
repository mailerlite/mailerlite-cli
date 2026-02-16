package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keyboard shortcuts for the dashboard.
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	Back     key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
	Refresh  key.Binding
	Profile  key.Binding
	Help     key.Binding
	Quit     key.Binding
	Search   key.Binding

	// View shortcuts
	View1 key.Binding
	View2 key.Binding
	View3 key.Binding
	View4 key.Binding
	View5 key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next panel"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev panel"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Profile: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "switch profile"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		View1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "subscribers"),
		),
		View2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "campaigns"),
		),
		View3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "automations"),
		),
		View4: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "groups"),
		),
		View5: key.NewBinding(
			key.WithKeys("5"),
			key.WithHelp("5", "forms"),
		),
	}
}

// HelpBindings returns key bindings formatted for the help overlay.
func (k KeyMap) HelpBindings() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Back},
		{k.Tab, k.Refresh, k.Profile, k.Help},
		{k.View1, k.View2, k.View3, k.View4, k.View5},
		{k.Quit},
	}
}
