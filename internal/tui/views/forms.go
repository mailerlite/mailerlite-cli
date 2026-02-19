package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mailerlite/mailerlite-cli/internal/sdkclient"
	"github.com/mailerlite/mailerlite-cli/internal/tui/components"
	"github.com/mailerlite/mailerlite-cli/internal/tui/theme"
	"github.com/mailerlite/mailerlite-cli/internal/tui/types"
	"github.com/mailerlite/mailerlite-go"
)

// FormType represents the type filter for forms.
type FormType int

const (
	FormTypePopup FormType = iota
	FormTypeEmbedded
	FormTypePromotion
)

func (f FormType) String() string {
	switch f {
	case FormTypePopup:
		return "Popup"
	case FormTypeEmbedded:
		return "Embedded"
	case FormTypePromotion:
		return "Promotion"
	default:
		return "Unknown"
	}
}

func (f FormType) APIValue() string {
	switch f {
	case FormTypePopup:
		return "popup"
	case FormTypeEmbedded:
		return "embedded"
	case FormTypePromotion:
		return "promotion"
	default:
		return "popup"
	}
}

var (
	formTabStyle = lipgloss.NewStyle().
			Padding(0, 2)

	activeFormTabStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(theme.Primary).
				Background(theme.BgSelected).
				Padding(0, 2)

	formTabBarStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(theme.Muted).
			MarginBottom(1)
)

// FormsView displays the list of forms.
type FormsView struct {
	client *mailerlite.Client

	table         components.Table
	detail        components.DetailPanel
	forms         []mailerlite.Form
	loading       bool
	err           error
	width         int
	height        int
	focused       bool
	activeTab     FormType
	showingDetail bool
}

// NewFormsView creates a new forms view.
func NewFormsView(client *mailerlite.Client) FormsView {
	columns := []components.Column{
		{Title: "NAME", Width: 28},
		{Title: "TYPE", Width: 12},
		{Title: "ACTIVE", Width: 8},
		{Title: "CONVERSIONS", Width: 12},
		{Title: "OPENS", Width: 8},
	}
	table := components.NewTable(columns)
	table.SetEmptyMessage("No forms found.")

	return FormsView{
		client: client,

		table:     table,
		loading:   true,
		activeTab: FormTypePopup,
	}
}

// SetSize sets the view dimensions.
func (v *FormsView) SetSize(width, height int) {
	v.width = width
	v.height = height
	// Reserve space for tab bar
	v.table.SetSize(width, height-4)
}

// SetFocused sets whether this view is focused.
func (v *FormsView) SetFocused(focused bool) {
	v.focused = focused
	v.table.SetFocused(focused)
}

// Loading returns whether the view is loading.
func (v FormsView) Loading() bool {
	return v.loading
}

// ItemCount returns the number of items.
func (v FormsView) ItemCount() int {
	return len(v.forms)
}

// SelectedForm returns the currently selected form.
func (v FormsView) SelectedForm() *mailerlite.Form {
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.forms) {
		return &v.forms[idx]
	}
	return nil
}

// Fetch returns a command to fetch forms.
func (v FormsView) Fetch() tea.Cmd {
	return func() tea.Msg {
		if v.client == nil {
			return types.FormsLoadedMsg{Err: nil}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		forms, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]mailerlite.Form, bool, error) {
			root, _, err := v.client.Form.List(ctx, &mailerlite.ListFormOptions{
				Type:  v.activeTab.APIValue(),
				Page:  page,
				Limit: perPage,
			})
			if err != nil {
				return nil, false, sdkclient.WrapError(err)
			}
			return root.Data, root.Links.Next != "", nil
		}, 100)

		return types.FormsLoadedMsg{
			Forms: forms,
			Err:   err,
		}
	}
}

// Update handles messages for this view.
func (v FormsView) Update(msg tea.Msg) (FormsView, tea.Cmd) {
	switch msg := msg.(type) {
	case types.FormsLoadedMsg:
		v.loading = false
		v.err = msg.Err
		if msg.Err == nil {
			v.forms = msg.Forms
			v.updateTable()
		}
	}
	return v, nil
}

// HandleKey handles key events when this view is active.
func (v *FormsView) HandleKey(msg tea.KeyMsg) tea.Cmd {
	if v.showingDetail {
		switch msg.String() {
		case "esc", "backspace", "q":
			v.showingDetail = false
		}
		return nil
	}

	switch msg.String() {
	case "j", "down":
		v.table.MoveDown()
	case "k", "up":
		v.table.MoveUp()
	case "g":
		v.table.GotoTop()
	case "G":
		v.table.GotoBottom()
	case "h", "left":
		v.prevTab()
		v.loading = true
		return v.Fetch()
	case "l", "right":
		v.nextTab()
		v.loading = true
		return v.Fetch()
	case "enter":
		v.showDetail()
	case "r":
		v.loading = true
		v.table.SetLoading(true)
		return v.Fetch()
	}
	return nil
}

func (v *FormsView) nextTab() {
	v.activeTab = (v.activeTab + 1) % 3
}

func (v *FormsView) prevTab() {
	v.activeTab--
	if v.activeTab < 0 {
		v.activeTab = 2
	}
}

func (v *FormsView) showDetail() {
	f := v.SelectedForm()
	if f == nil {
		return
	}

	v.detail.SetTitle("Form: " + f.Name)

	active := "No"
	if f.Active {
		active = "Yes"
	}

	created := f.CreatedAt
	if t, err := time.Parse("2006-01-02 15:04:05", f.CreatedAt); err == nil {
		created = t.Format("2006-01-02 15:04:05")
	}

	v.detail.SetRows([]components.DetailRow{
		{Label: "ID", Value: f.Id},
		{Label: "Name", Value: f.Name},
		{Label: "Type", Value: f.Type},
		{Label: "Active", Value: active},
		{Label: "Conversions", Value: fmt.Sprintf("%d", f.ConversionsCount)},
		{Label: "Conversion Rate", Value: f.ConversionsRate.String},
		{Label: "Opens", Value: fmt.Sprintf("%d", f.OpensCount)},
		{Label: "Created", Value: created},
	})
	v.detail.SetSize(v.width, v.height-4)
	v.showingDetail = true
}

func (v *FormsView) updateTable() {
	var rows [][]string
	for _, f := range v.forms {
		active := checkStyle.Render("yes")
		if !f.Active {
			active = crossStyle.Render("no")
		}

		rows = append(rows, []string{
			f.Name,
			f.Type,
			active,
			fmt.Sprintf("%d", f.ConversionsCount),
			fmt.Sprintf("%d", f.OpensCount),
		})
	}
	v.table.SetRows(rows)
	v.table.SetLoading(false)
}

// View renders the forms view.
func (v FormsView) View() string {
	if v.showingDetail {
		return v.detail.View()
	}

	var b strings.Builder

	// Tab bar
	formTypes := []FormType{
		FormTypePopup,
		FormTypeEmbedded,
		FormTypePromotion,
	}

	var tabViews []string
	for _, ft := range formTypes {
		style := formTabStyle
		if ft == v.activeTab {
			style = activeFormTabStyle
		}
		tabViews = append(tabViews, style.Render(ft.String()))
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabViews...)
	b.WriteString(formTabBarStyle.Render(tabBar))
	b.WriteString("\n")

	// Hint for tab navigation
	hint := fmt.Sprintf("← → to switch types | %d forms", len(v.forms))
	b.WriteString(lipgloss.NewStyle().Foreground(theme.Muted).Render(hint))
	b.WriteString("\n\n")

	// Table
	b.WriteString(v.table.View())

	return b.String()
}

// ShowingDetail returns whether the detail view is active.
func (v FormsView) ShowingDetail() bool {
	return v.showingDetail
}
