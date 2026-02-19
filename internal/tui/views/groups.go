package views

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mailerlite/mailerlite-cli/internal/sdkclient"
	"github.com/mailerlite/mailerlite-cli/internal/tui/components"
	"github.com/mailerlite/mailerlite-cli/internal/tui/types"
	"github.com/mailerlite/mailerlite-go"
)

// GroupsView displays the list of groups.
type GroupsView struct {
	client *mailerlite.Client

	table         components.Table
	detail        components.DetailPanel
	groups        []mailerlite.Group
	loading       bool
	err           error
	width         int
	height        int
	focused       bool
	showingDetail bool
}

// NewGroupsView creates a new groups view.
func NewGroupsView(client *mailerlite.Client) GroupsView {
	columns := []components.Column{
		{Title: "NAME", Width: 28},
		{Title: "ACTIVE", Width: 8},
		{Title: "SENT", Width: 8},
		{Title: "OPENS", Width: 8},
		{Title: "CLICK RATE", Width: 12},
		{Title: "CREATED", Width: 12},
	}
	table := components.NewTable(columns)
	table.SetEmptyMessage("No groups found.")

	return GroupsView{
		client: client,

		table:   table,
		loading: true,
	}
}

// SetSize sets the view dimensions.
func (v *GroupsView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.table.SetSize(width, height)
}

// SetFocused sets whether this view is focused.
func (v *GroupsView) SetFocused(focused bool) {
	v.focused = focused
	v.table.SetFocused(focused)
}

// Loading returns whether the view is loading.
func (v GroupsView) Loading() bool {
	return v.loading
}

// ItemCount returns the number of items.
func (v GroupsView) ItemCount() int {
	return len(v.groups)
}

// SelectedGroup returns the currently selected group.
func (v GroupsView) SelectedGroup() *mailerlite.Group {
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.groups) {
		return &v.groups[idx]
	}
	return nil
}

// Fetch returns a command to fetch groups.
func (v GroupsView) Fetch() tea.Cmd {
	return func() tea.Msg {
		if v.client == nil {
			return types.GroupsLoadedMsg{Err: nil}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		groups, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]mailerlite.Group, bool, error) {
			root, _, err := v.client.Group.List(ctx, &mailerlite.ListGroupOptions{
				Page:  page,
				Limit: perPage,
			})
			if err != nil {
				return nil, false, sdkclient.WrapError(err)
			}
			return root.Data, root.Links.Next != "", nil
		}, 100)

		return types.GroupsLoadedMsg{
			Groups: groups,
			Err:    err,
		}
	}
}

// Update handles messages for this view.
func (v GroupsView) Update(msg tea.Msg) (GroupsView, tea.Cmd) {
	switch msg := msg.(type) {
	case types.GroupsLoadedMsg:
		v.loading = false
		v.err = msg.Err
		if msg.Err == nil {
			v.groups = msg.Groups
			v.updateTable()
		}
	}
	return v, nil
}

// HandleKey handles key events when this view is active.
func (v *GroupsView) HandleKey(msg tea.KeyMsg) tea.Cmd {
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

func (v *GroupsView) showDetail() {
	g := v.SelectedGroup()
	if g == nil {
		return
	}

	v.detail.SetTitle("Group: " + g.Name)

	created := g.CreatedAt
	if t, err := time.Parse("2006-01-02 15:04:05", g.CreatedAt); err == nil {
		created = t.Format("2006-01-02 15:04:05")
	}

	v.detail.SetRows([]components.DetailRow{
		{Label: "ID", Value: g.ID},
		{Label: "Name", Value: g.Name},
		{Label: "Active", Value: fmt.Sprintf("%d", g.ActiveCount)},
		{Label: "Sent", Value: fmt.Sprintf("%d", g.SentCount)},
		{Label: "Opens", Value: fmt.Sprintf("%d", g.OpensCount)},
		{Label: "Open Rate", Value: g.OpenRate.String},
		{Label: "Clicks", Value: fmt.Sprintf("%d", g.ClicksCount)},
		{Label: "Click Rate", Value: g.ClickRate.String},
		{Label: "Unsubscribed", Value: fmt.Sprintf("%d", g.UnsubscribedCount)},
		{Label: "Bounced", Value: fmt.Sprintf("%d", g.BouncedCount)},
		{Label: "Created", Value: created},
	})
	v.detail.SetSize(v.width, v.height)
	v.showingDetail = true
}

func (v *GroupsView) updateTable() {
	var rows [][]string
	for _, g := range v.groups {
		created := ""
		if t, err := time.Parse("2006-01-02 15:04:05", g.CreatedAt); err == nil {
			created = t.Format("2006-01-02")
		}

		rows = append(rows, []string{
			g.Name,
			fmt.Sprintf("%d", g.ActiveCount),
			fmt.Sprintf("%d", g.SentCount),
			fmt.Sprintf("%d", g.OpensCount),
			g.ClickRate.String,
			created,
		})
	}
	v.table.SetRows(rows)
	v.table.SetLoading(false)
}

// View renders the groups view.
func (v GroupsView) View() string {
	if v.showingDetail {
		return v.detail.View()
	}
	return v.table.View()
}

// ShowingDetail returns whether the detail view is active.
func (v GroupsView) ShowingDetail() bool {
	return v.showingDetail
}
