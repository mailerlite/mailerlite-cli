package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mailerlite/mailerlite-cli/internal/tui/theme"
)

var (
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(theme.Primary).
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(theme.Muted)

	tableRowStyle = lipgloss.NewStyle()

	tableSelectedStyle = lipgloss.NewStyle().
				Background(theme.BgSelected).
				Foreground(theme.Text)

	tableFocusedSelectedStyle = lipgloss.NewStyle().
					Background(theme.Accent).
					Foreground(theme.Text)

	emptyStyle = lipgloss.NewStyle().
			Foreground(theme.Muted).
			Italic(true)
)

// Column defines a table column.
type Column struct {
	Title string
	Width int
}

// Table is an interactive table component.
type Table struct {
	columns  []Column
	rows     [][]string
	cursor   int
	offset   int
	width    int
	height   int
	focused  bool
	loading  bool
	emptyMsg string
}

// NewTable creates a new table with the given columns.
func NewTable(columns []Column) Table {
	return Table{
		columns:  columns,
		emptyMsg: "No data",
	}
}

// SetColumns updates the table columns.
func (t *Table) SetColumns(columns []Column) {
	t.columns = columns
}

// SetRows sets the table data.
func (t *Table) SetRows(rows [][]string) {
	t.rows = rows
	t.cursor = 0
	t.offset = 0
}

// SetSize sets the table dimensions.
func (t *Table) SetSize(width, height int) {
	t.width = width
	t.height = height
}

// SetFocused sets whether the table is focused.
func (t *Table) SetFocused(focused bool) {
	t.focused = focused
}

// SetLoading sets the loading state.
func (t *Table) SetLoading(loading bool) {
	t.loading = loading
}

// SetEmptyMessage sets the message shown when there are no rows.
func (t *Table) SetEmptyMessage(msg string) {
	t.emptyMsg = msg
}

// Cursor returns the current cursor position.
func (t Table) Cursor() int {
	return t.cursor
}

// SelectedRow returns the currently selected row data.
func (t Table) SelectedRow() []string {
	if t.cursor >= 0 && t.cursor < len(t.rows) {
		return t.rows[t.cursor]
	}
	return nil
}

// RowCount returns the number of rows.
func (t Table) RowCount() int {
	return len(t.rows)
}

// MoveUp moves the cursor up.
func (t *Table) MoveUp() {
	if t.cursor > 0 {
		t.cursor--
		t.updateOffset()
	}
}

// MoveDown moves the cursor down.
func (t *Table) MoveDown() {
	if t.cursor < len(t.rows)-1 {
		t.cursor++
		t.updateOffset()
	}
}

// GotoTop moves to the first row.
func (t *Table) GotoTop() {
	t.cursor = 0
	t.offset = 0
}

// GotoBottom moves to the last row.
func (t *Table) GotoBottom() {
	t.cursor = len(t.rows) - 1
	t.updateOffset()
}

func (t *Table) updateOffset() {
	visibleRows := t.visibleRowCount()
	if visibleRows <= 0 {
		return
	}

	// Scroll down if cursor is below visible area
	if t.cursor >= t.offset+visibleRows {
		t.offset = t.cursor - visibleRows + 1
	}
	// Scroll up if cursor is above visible area
	if t.cursor < t.offset {
		t.offset = t.cursor
	}
}

func (t Table) visibleRowCount() int {
	// Account for header and borders
	return t.height - 3
}

// View renders the table.
func (t Table) View() string {
	if t.loading {
		return emptyStyle.Render("Loading...")
	}

	if len(t.rows) == 0 {
		return emptyStyle.Render(t.emptyMsg)
	}

	var b strings.Builder

	// Render header
	header := t.renderRow(t.columnTitles(), tableHeaderStyle, false)
	b.WriteString(header)
	b.WriteString("\n")

	// Calculate visible range
	visibleRows := t.visibleRowCount()
	if visibleRows < 1 {
		visibleRows = 1
	}

	start := t.offset
	end := start + visibleRows
	if end > len(t.rows) {
		end = len(t.rows)
	}

	// Render visible rows
	for i := start; i < end; i++ {
		row := t.rows[i]
		style := tableRowStyle
		isSelected := i == t.cursor

		if isSelected {
			if t.focused {
				style = tableFocusedSelectedStyle
			} else {
				style = tableSelectedStyle
			}
		}

		b.WriteString(t.renderRow(row, style, isSelected))
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (t Table) columnTitles() []string {
	titles := make([]string, len(t.columns))
	for i, col := range t.columns {
		titles[i] = col.Title
	}
	return titles
}

func (t Table) renderRow(cells []string, style lipgloss.Style, pad bool) string {
	var parts []string

	for i, col := range t.columns {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}

		// Truncate if too long
		if len(cell) > col.Width {
			if col.Width > 3 {
				cell = cell[:col.Width-3] + "..."
			} else {
				cell = cell[:col.Width]
			}
		}

		// Pad to column width
		cell = padRight(cell, col.Width)
		parts = append(parts, cell)
	}

	content := strings.Join(parts, "  ")

	// Apply style and pad to full width if selected
	if pad && t.width > 0 {
		content = padRight(content, t.width-4)
	}

	return style.Render(content)
}
