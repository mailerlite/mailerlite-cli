package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mailerlite/mailerlite-cli/internal/tui/theme"
)

var (
	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(theme.Primary).
				MarginBottom(1)

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(theme.Muted).
				Width(20)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(theme.Text)

	detailHintStyle = lipgloss.NewStyle().
			Foreground(theme.Muted).
			Italic(true).
			MarginTop(1)
)

// DetailRow represents a key-value pair for display.
type DetailRow struct {
	Label string
	Value string
}

// DetailPanel displays detailed information about an item.
type DetailPanel struct {
	title  string
	rows   []DetailRow
	width  int
	height int
}

// NewDetailPanel creates a new detail panel.
func NewDetailPanel() DetailPanel {
	return DetailPanel{}
}

// SetTitle sets the panel title.
func (d *DetailPanel) SetTitle(title string) {
	d.title = title
}

// SetRows sets the detail rows.
func (d *DetailPanel) SetRows(rows []DetailRow) {
	d.rows = rows
}

// SetSize sets the panel dimensions.
func (d *DetailPanel) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// View renders the detail panel.
func (d DetailPanel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(detailTitleStyle.Render(d.title))
	b.WriteString("\n\n")

	// Rows
	for _, row := range d.rows {
		if row.Label == "" && row.Value == "" {
			b.WriteString("\n")
			continue
		}
		if row.Label == "" {
			value := detailValueStyle.Render(row.Value)
			b.WriteString(detailLabelStyle.Render("") + " " + value + "\n")
			continue
		}
		label := detailLabelStyle.Render(row.Label + ":")
		value := detailValueStyle.Render(row.Value)
		b.WriteString(label + " " + value + "\n")
	}

	// Hint
	b.WriteString("\n")
	b.WriteString(detailHintStyle.Render("Press Esc or Backspace to go back"))

	return b.String()
}
