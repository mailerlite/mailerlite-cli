package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mailerlite/mailerlite-cli/internal/tui/theme"
)

var (
	statusStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(theme.Muted)

	statusTextStyle = lipgloss.NewStyle().
			Foreground(theme.Muted)

	statusKeyStyle = lipgloss.NewStyle().
			Foreground(theme.Key)

	statusValueStyle = lipgloss.NewStyle().
				Foreground(theme.TextSub)
)

// StatusBar represents the bottom status bar.
type StatusBar struct {
	width    int
	left     string
	center   string
	right    string
	profile  string
	loading  bool
	loadText string
}

// NewStatusBar creates a new status bar.
func NewStatusBar() StatusBar {
	return StatusBar{}
}

// SetWidth sets the status bar width.
func (s *StatusBar) SetWidth(w int) {
	s.width = w
}

// SetLeft sets the left section text.
func (s *StatusBar) SetLeft(text string) {
	s.left = text
}

// SetCenter sets the center section text.
func (s *StatusBar) SetCenter(text string) {
	s.center = text
}

// SetRight sets the right section text.
func (s *StatusBar) SetRight(text string) {
	s.right = text
}

// SetProfile sets the current profile name.
func (s *StatusBar) SetProfile(name string) {
	s.profile = name
}

// SetLoading sets the loading state and text.
func (s *StatusBar) SetLoading(loading bool, text string) {
	s.loading = loading
	s.loadText = text
}

// View renders the status bar.
func (s StatusBar) View() string {
	if s.width == 0 {
		return ""
	}

	// Build left section: view name + count
	left := statusTextStyle.Render(s.left)

	// Build center section: loading or custom text
	center := ""
	if s.loading {
		center = statusTextStyle.Render(s.loadText)
	} else if s.center != "" {
		center = statusTextStyle.Render(s.center)
	}

	// Build right section: key hints + profile
	hints := []string{
		fmt.Sprintf("%s %s", statusKeyStyle.Render("↑↓"), statusValueStyle.Render("navigate")),
		fmt.Sprintf("%s %s", statusKeyStyle.Render("enter"), statusValueStyle.Render("select")),
		fmt.Sprintf("%s %s", statusKeyStyle.Render("?"), statusValueStyle.Render("help")),
	}
	if s.profile != "" {
		hints = append(hints, fmt.Sprintf("%s %s", statusKeyStyle.Render("profile:"), statusValueStyle.Render(s.profile)))
	}
	right := strings.Join(hints, "  ")

	// Calculate spacing
	leftLen := lipgloss.Width(left)
	centerLen := lipgloss.Width(center)
	rightLen := lipgloss.Width(right)

	// Available space for padding
	contentWidth := s.width - 4 // account for padding
	totalContentLen := leftLen + centerLen + rightLen

	if totalContentLen >= contentWidth {
		// Not enough space, just show left and right
		gap := contentWidth - leftLen - rightLen
		if gap < 1 {
			gap = 1
		}
		return statusStyle.Width(s.width).Render(left + strings.Repeat(" ", gap) + right)
	}

	// Calculate gaps
	remainingSpace := contentWidth - totalContentLen
	leftGap := remainingSpace / 2
	rightGap := remainingSpace - leftGap

	content := left + strings.Repeat(" ", leftGap) + center + strings.Repeat(" ", rightGap) + right

	return statusStyle.Width(s.width).Render(content)
}
