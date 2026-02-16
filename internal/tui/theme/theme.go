package theme

import "github.com/charmbracelet/lipgloss"

var (
	Primary    = lipgloss.AdaptiveColor{Light: "4", Dark: "12"}
	Accent     = lipgloss.AdaptiveColor{Light: "25", Dark: "24"} // 256-palette dark blue bg, not remapped by themes
	Muted      = lipgloss.AdaptiveColor{Light: "245", Dark: "245"}
	Text       = lipgloss.AdaptiveColor{Light: "0", Dark: "231"} // 256-palette pure white, not remapped
	TextSub    = lipgloss.AdaptiveColor{Light: "238", Dark: "250"}
	Success    = lipgloss.AdaptiveColor{Light: "2", Dark: "10"}
	Error      = lipgloss.AdaptiveColor{Light: "1", Dark: "9"}
	Key        = lipgloss.AdaptiveColor{Light: "6", Dark: "14"}
	BgSelected = lipgloss.AdaptiveColor{Light: "254", Dark: "238"}
	BgOverlay  = lipgloss.AdaptiveColor{Light: "255", Dark: "237"}
)
