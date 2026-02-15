package components

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mailerlite/mailerlite-cli/internal/tui/theme"
)

// Spinner wraps the bubbles spinner with consistent styling.
type Spinner struct {
	spinner spinner.Model
	label   string
	visible bool
}

// NewSpinner creates a new spinner with the given label.
func NewSpinner(label string) Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.Primary)
	return Spinner{
		spinner: s,
		label:   label,
		visible: false,
	}
}

// Start makes the spinner visible.
func (s *Spinner) Start() {
	s.visible = true
}

// Stop hides the spinner.
func (s *Spinner) Stop() {
	s.visible = false
}

// SetLabel updates the spinner label.
func (s *Spinner) SetLabel(label string) {
	s.label = label
}

// Visible returns whether the spinner is showing.
func (s Spinner) Visible() bool {
	return s.visible
}

// Init initializes the spinner.
func (s Spinner) Init() tea.Cmd {
	return s.spinner.Tick
}

// Update handles spinner tick messages.
func (s Spinner) Update(msg tea.Msg) (Spinner, tea.Cmd) {
	if !s.visible {
		return s, nil
	}
	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return s, cmd
}

// View renders the spinner.
func (s Spinner) View() string {
	if !s.visible {
		return ""
	}
	return s.spinner.View() + " " + s.label
}

// Tick returns the spinner tick command.
func (s Spinner) Tick() tea.Cmd {
	return s.spinner.Tick
}
