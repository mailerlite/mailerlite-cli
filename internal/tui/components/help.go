package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/mailerlite/mailerlite-cli/internal/tui/theme"
)

var (
	helpOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(theme.Primary).
				Padding(1, 2).
				Background(theme.BgOverlay)

	helpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Primary).
			MarginBottom(1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(theme.Key).
			Width(14)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(theme.TextSub)

	helpSectionStyle = lipgloss.NewStyle().
				MarginTop(1).
				MarginBottom(0)
)

// Help is the help overlay component.
type Help struct {
	bindings [][]key.Binding
	visible  bool
	width    int
	height   int
}

// NewHelp creates a new help overlay.
func NewHelp(bindings [][]key.Binding) Help {
	return Help{
		bindings: bindings,
	}
}

// SetVisible sets whether the help overlay is shown.
func (h *Help) SetVisible(visible bool) {
	h.visible = visible
}

// Toggle toggles the help overlay visibility.
func (h *Help) Toggle() {
	h.visible = !h.visible
}

// Visible returns whether the help is showing.
func (h Help) Visible() bool {
	return h.visible
}

// SetSize sets the overlay dimensions.
func (h *Help) SetSize(width, height int) {
	h.width = width
	h.height = height
}

// View renders the help overlay.
func (h Help) View() string {
	if !h.visible {
		return ""
	}

	var b strings.Builder

	b.WriteString(helpTitleStyle.Render("Keyboard Shortcuts"))
	b.WriteString("\n\n")

	sections := []struct {
		title    string
		bindings []key.Binding
	}{
		{"Navigation", h.bindings[0]},
		{"Actions", h.bindings[1]},
		{"Views", h.bindings[2]},
		{"General", h.bindings[3]},
	}

	for i, section := range sections {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(helpSectionStyle.Render(lipgloss.NewStyle().Bold(true).Render(section.title)))
		b.WriteString("\n")

		for _, binding := range section.bindings {
			help := binding.Help()
			line := helpKeyStyle.Render(help.Key) + helpDescStyle.Render(help.Desc)
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(theme.Muted).Render("Press ? or Esc to close"))

	content := b.String()

	// Calculate overlay size
	overlayWidth := 40
	overlayHeight := strings.Count(content, "\n") + 4

	// Center the overlay
	x := (h.width - overlayWidth) / 2
	y := (h.height - overlayHeight) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	styled := helpOverlayStyle.Width(overlayWidth).Render(content)

	// Add positioning
	var result strings.Builder
	for i := 0; i < y; i++ {
		result.WriteString("\n")
	}
	lines := strings.Split(styled, "\n")
	for _, line := range lines {
		result.WriteString(strings.Repeat(" ", x))
		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
}
