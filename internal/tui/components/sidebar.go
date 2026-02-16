package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mailerlite/mailerlite-cli/internal/tui/theme"
	"github.com/mailerlite/mailerlite-cli/internal/tui/types"
)

const SidebarWidth = 20

var (
	sidebarStyle = lipgloss.NewStyle().
			Width(SidebarWidth).
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(theme.Muted).
			Padding(1, 1)

	itemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	activeItemStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Primary).
			Background(theme.BgSelected).
			Padding(0, 1)

	focusedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(theme.Text).
				Background(theme.Accent).
				Padding(0, 1)
)

// Sidebar is the navigation sidebar component.
type Sidebar struct {
	views   []types.ViewInfo
	active  types.ViewType
	focused bool
	height  int
}

// NewSidebar creates a new sidebar.
func NewSidebar() Sidebar {
	return Sidebar{
		views:  types.AllViews(),
		active: types.ViewSubscribers,
	}
}

// SetActive sets the active view.
func (s *Sidebar) SetActive(v types.ViewType) {
	s.active = v
}

// Active returns the currently active view.
func (s Sidebar) Active() types.ViewType {
	return s.active
}

// SetFocused sets whether the sidebar is focused.
func (s *Sidebar) SetFocused(focused bool) {
	s.focused = focused
}

// Focused returns whether the sidebar is focused.
func (s Sidebar) Focused() bool {
	return s.focused
}

// SetHeight sets the sidebar height.
func (s *Sidebar) SetHeight(h int) {
	s.height = h
}

// Next moves to the next view.
func (s *Sidebar) Next() {
	idx := int(s.active)
	idx = (idx + 1) % len(s.views)
	s.active = s.views[idx].Type
}

// Prev moves to the previous view.
func (s *Sidebar) Prev() {
	idx := int(s.active)
	idx--
	if idx < 0 {
		idx = len(s.views) - 1
	}
	s.active = s.views[idx].Type
}

// SetView sets the view directly.
func (s *Sidebar) SetView(v types.ViewType) {
	if int(v) >= 0 && int(v) < len(s.views) {
		s.active = v
	}
}

// Width returns the sidebar width.
func (s Sidebar) Width() int {
	return SidebarWidth + 2 // account for border
}

// View renders the sidebar.
func (s Sidebar) View() string {
	var b strings.Builder

	for _, view := range s.views {
		style := itemStyle
		prefix := "  "

		if view.Type == s.active {
			if s.focused {
				style = focusedItemStyle
			} else {
				style = activeItemStyle
			}
			prefix = "▸ "
		}

		line := prefix + view.Icon + " " + view.Label
		// Pad to full width
		line = padRight(line, SidebarWidth-4)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	content := strings.TrimRight(b.String(), "\n")
	return sidebarStyle.Height(s.height - 2).Render(content)
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
