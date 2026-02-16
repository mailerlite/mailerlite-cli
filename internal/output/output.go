package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var (
	noColor = os.Getenv("NO_COLOR") != ""

	HeaderStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	ErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	DimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func style(s lipgloss.Style, text string) string {
	if noColor {
		return text
	}
	return s.Render(text)
}

func Success(msg string) {
	fmt.Println(style(SuccessStyle, msg))
}

func Error(msg string) {
	fmt.Fprintln(os.Stderr, style(ErrorStyle, msg))
}

func Errorf(format string, args ...interface{}) {
	Error(fmt.Sprintf(format, args...))
}

func JSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func Table(headers []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Println(style(DimStyle, "No results found."))
		return
	}

	if noColor {
		printPlainTable(headers, rows)
		return
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("8"))).
		Headers(headers...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Padding(0, 1)
			}
			return lipgloss.NewStyle().Padding(0, 1)
		})

	for _, row := range rows {
		t.Row(row...)
	}

	fmt.Println(t)
}

func printPlainTable(headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Header
	for i, h := range headers {
		fmt.Printf("%-*s", widths[i]+2, strings.ToUpper(h))
	}
	fmt.Println()

	// Rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				fmt.Printf("%-*s", widths[i]+2, cell)
			}
		}
		fmt.Println()
	}
}

func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
