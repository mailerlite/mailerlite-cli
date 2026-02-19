package views

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mailerlite/mailerlite-cli/internal/sdkclient"
	"github.com/mailerlite/mailerlite-cli/internal/tui/components"
	"github.com/mailerlite/mailerlite-cli/internal/tui/theme"
	"github.com/mailerlite/mailerlite-cli/internal/tui/types"
	"github.com/mailerlite/mailerlite-go"
)

func enabledBadge(enabled bool) string {
	if enabled {
		return lipgloss.NewStyle().Foreground(theme.Success).Render("yes")
	}
	return lipgloss.NewStyle().Foreground(theme.Muted).Render("no")
}

// AutomationsView displays the list of automations.
type AutomationsView struct {
	client *mailerlite.Client

	table         components.Table
	detail        components.DetailPanel
	automations   []mailerlite.Automation
	loading       bool
	err           error
	width         int
	height        int
	focused       bool
	showingDetail bool
}

// NewAutomationsView creates a new automations view.
func NewAutomationsView(client *mailerlite.Client) AutomationsView {
	columns := []components.Column{
		{Title: "NAME", Width: 30},
		{Title: "ENABLED", Width: 9},
		{Title: "EMAILS", Width: 8},
		{Title: "COMPLETED", Width: 10},
		{Title: "IN QUEUE", Width: 10},
	}
	table := components.NewTable(columns)
	table.SetEmptyMessage("No automations found.")

	return AutomationsView{
		client: client,

		table:   table,
		loading: true,
	}
}

// SetSize sets the view dimensions.
func (v *AutomationsView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.table.SetSize(width, height)
}

// SetFocused sets whether this view is focused.
func (v *AutomationsView) SetFocused(focused bool) {
	v.focused = focused
	v.table.SetFocused(focused)
}

// Loading returns whether the view is loading.
func (v AutomationsView) Loading() bool {
	return v.loading
}

// ItemCount returns the number of items.
func (v AutomationsView) ItemCount() int {
	return len(v.automations)
}

// SelectedAutomation returns the currently selected automation.
func (v AutomationsView) SelectedAutomation() *mailerlite.Automation {
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.automations) {
		return &v.automations[idx]
	}
	return nil
}

// Fetch returns a command to fetch automations.
func (v AutomationsView) Fetch() tea.Cmd {
	return func() tea.Msg {
		if v.client == nil {
			return types.AutomationsLoadedMsg{Err: nil}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		automations, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]mailerlite.Automation, bool, error) {
			root, _, err := v.client.Automation.List(ctx, &mailerlite.ListAutomationOptions{
				Page:  page,
				Limit: perPage,
			})
			if err != nil {
				return nil, false, sdkclient.WrapError(err)
			}
			return root.Data, root.Links.Next != "", nil
		}, 100)

		return types.AutomationsLoadedMsg{
			Automations: automations,
			Err:         err,
		}
	}
}

// Update handles messages for this view.
func (v AutomationsView) Update(msg tea.Msg) (AutomationsView, tea.Cmd) {
	switch msg := msg.(type) {
	case types.AutomationsLoadedMsg:
		v.loading = false
		v.err = msg.Err
		if msg.Err == nil {
			v.automations = msg.Automations
			v.updateTable()
		}
	}
	return v, nil
}

// HandleKey handles key events when this view is active.
func (v *AutomationsView) HandleKey(msg tea.KeyMsg) tea.Cmd {
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
	case "enter":
		v.showDetail()
	case "r":
		v.loading = true
		v.table.SetLoading(true)
		return v.Fetch()
	}
	return nil
}

func (v *AutomationsView) showDetail() {
	a := v.SelectedAutomation()
	if a == nil {
		return
	}

	v.detail.SetTitle("Automation: " + a.Name)

	enabled := "No"
	if a.Enabled {
		enabled = "Yes"
	}

	created := a.CreatedAt
	if t, err := time.Parse("2006-01-02 15:04:05", a.CreatedAt); err == nil {
		created = t.Format("2006-01-02 15:04:05")
	}

	v.detail.SetRows([]components.DetailRow{
		{Label: "ID", Value: a.ID},
		{Label: "Name", Value: a.Name},
		{Label: "Enabled", Value: enabled},
		{Label: "Emails", Value: fmt.Sprintf("%d", a.EmailsCount)},
		{Label: "Completed", Value: fmt.Sprintf("%d", a.Stats.CompletedSubscribersCount)},
		{Label: "In Queue", Value: fmt.Sprintf("%d", a.Stats.SubscribersInQueueCount)},
		{Label: "Sent", Value: fmt.Sprintf("%d", a.Stats.Sent)},
		{Label: "Opens", Value: fmt.Sprintf("%d", a.Stats.OpensCount)},
		{Label: "Clicks", Value: fmt.Sprintf("%d", a.Stats.ClicksCount)},
		{Label: "Created", Value: created},
	})
	v.detail.SetSize(v.width, v.height)
	v.showingDetail = true
}

func (v *AutomationsView) updateTable() {
	var rows [][]string
	for _, a := range v.automations {
		rows = append(rows, []string{
			a.Name,
			enabledBadge(a.Enabled),
			fmt.Sprintf("%d", a.EmailsCount),
			fmt.Sprintf("%d", a.Stats.CompletedSubscribersCount),
			fmt.Sprintf("%d", a.Stats.SubscribersInQueueCount),
		})
	}
	v.table.SetRows(rows)
	v.table.SetLoading(false)
}

// View renders the automations view.
func (v AutomationsView) View() string {
	if v.showingDetail {
		return v.detail.View()
	}
	return v.table.View()
}

// ShowingDetail returns whether the detail view is active.
func (v AutomationsView) ShowingDetail() bool {
	return v.showingDetail
}
