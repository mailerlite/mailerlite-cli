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

// CampaignsView displays the list of campaigns.
type CampaignsView struct {
	client *mailerlite.Client

	table         components.Table
	detail        components.DetailPanel
	campaigns     []mailerlite.Campaign
	loading       bool
	err           error
	width         int
	height        int
	focused       bool
	showingDetail bool
}

// NewCampaignsView creates a new campaigns view.
func NewCampaignsView(client *mailerlite.Client) CampaignsView {
	columns := []components.Column{
		{Title: "NAME", Width: 28},
		{Title: "TYPE", Width: 10},
		{Title: "STATUS", Width: 12},
		{Title: "SENT", Width: 8},
		{Title: "OPENS", Width: 8},
		{Title: "CLICKS", Width: 8},
	}
	table := components.NewTable(columns)
	table.SetEmptyMessage("No campaigns found.")

	return CampaignsView{
		client: client,

		table:   table,
		loading: true,
	}
}

// SetSize sets the view dimensions.
func (v *CampaignsView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.table.SetSize(width, height)
}

// SetFocused sets whether this view is focused.
func (v *CampaignsView) SetFocused(focused bool) {
	v.focused = focused
	v.table.SetFocused(focused)
}

// Loading returns whether the view is loading.
func (v CampaignsView) Loading() bool {
	return v.loading
}

// ItemCount returns the number of items.
func (v CampaignsView) ItemCount() int {
	return len(v.campaigns)
}

// SelectedCampaign returns the currently selected campaign.
func (v CampaignsView) SelectedCampaign() *mailerlite.Campaign {
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.campaigns) {
		return &v.campaigns[idx]
	}
	return nil
}

// Fetch returns a command to fetch campaigns.
func (v CampaignsView) Fetch() tea.Cmd {
	return func() tea.Msg {
		if v.client == nil {
			return types.CampaignsLoadedMsg{Err: nil}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		campaigns, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]mailerlite.Campaign, bool, error) {
			root, _, err := v.client.Campaign.List(ctx, &mailerlite.ListCampaignOptions{
				Page:  page,
				Limit: perPage,
			})
			if err != nil {
				return nil, false, sdkclient.WrapError(err)
			}
			return root.Data, root.Links.Next != "", nil
		}, 100)

		return types.CampaignsLoadedMsg{
			Campaigns: campaigns,
			Err:       err,
		}
	}
}

// Update handles messages for this view.
func (v CampaignsView) Update(msg tea.Msg) (CampaignsView, tea.Cmd) {
	switch msg := msg.(type) {
	case types.CampaignsLoadedMsg:
		v.loading = false
		v.err = msg.Err
		if msg.Err == nil {
			v.campaigns = msg.Campaigns
			v.updateTable()
		}
	}
	return v, nil
}

// HandleKey handles key events when this view is active.
func (v *CampaignsView) HandleKey(msg tea.KeyMsg) tea.Cmd {
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

func (v *CampaignsView) showDetail() {
	c := v.SelectedCampaign()
	if c == nil {
		return
	}

	v.detail.SetTitle("Campaign: " + c.Name)

	created := c.CreatedAt
	if t, err := time.Parse("2006-01-02 15:04:05", c.CreatedAt); err == nil {
		created = t.Format("2006-01-02 15:04:05")
	}

	rows := []components.DetailRow{
		{Label: "ID", Value: c.ID},
		{Label: "Name", Value: c.Name},
		{Label: "Type", Value: c.TypeForHumans},
		{Label: "Status", Value: c.Status},
		{Label: "Sent", Value: fmt.Sprintf("%d", c.Stats.Sent)},
		{Label: "Opens", Value: fmt.Sprintf("%d", c.Stats.OpensCount)},
		{Label: "Clicks", Value: fmt.Sprintf("%d", c.Stats.ClicksCount)},
		{Label: "Open Rate", Value: c.Stats.OpenRate.String},
		{Label: "Click Rate", Value: c.Stats.ClickRate.String},
		{Label: "Created", Value: created},
	}

	if c.ScheduledFor != "" {
		rows = append(rows, components.DetailRow{Label: "Scheduled For", Value: c.ScheduledFor})
	}

	v.detail.SetRows(rows)
	v.detail.SetSize(v.width, v.height)
	v.showingDetail = true
}

func (v *CampaignsView) updateTable() {
	var rows [][]string
	for _, c := range v.campaigns {
		rows = append(rows, []string{
			c.Name,
			c.TypeForHumans,
			c.Status,
			fmt.Sprintf("%d", c.Stats.Sent),
			fmt.Sprintf("%d", c.Stats.OpensCount),
			fmt.Sprintf("%d", c.Stats.ClicksCount),
		})
	}
	v.table.SetRows(rows)
	v.table.SetLoading(false)
}

// View renders the campaigns view.
func (v CampaignsView) View() string {
	if v.showingDetail {
		return v.detail.View()
	}
	return v.table.View()
}

// ShowingDetail returns whether the detail view is active.
func (v CampaignsView) ShowingDetail() bool {
	return v.showingDetail
}
