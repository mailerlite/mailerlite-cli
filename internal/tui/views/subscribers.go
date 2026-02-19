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

var (
	checkStyle = lipgloss.NewStyle().Foreground(theme.Success)
	crossStyle = lipgloss.NewStyle().Foreground(theme.Error)
)

func statusBadge(status string) string {
	switch status {
	case "active":
		return checkStyle.Render("active")
	case "unsubscribed":
		return crossStyle.Render("unsub")
	case "unconfirmed":
		return lipgloss.NewStyle().Foreground(theme.Muted).Render("unconf")
	case "bounced":
		return crossStyle.Render("bounced")
	case "junk":
		return crossStyle.Render("junk")
	default:
		return status
	}
}

// SubscribersView displays the list of subscribers.
type SubscribersView struct {
	client *mailerlite.Client

	table         components.Table
	detail        components.DetailPanel
	subscribers   []mailerlite.Subscriber
	loading       bool
	err           error
	width         int
	height        int
	focused       bool
	showingDetail bool
}

// NewSubscribersView creates a new subscribers view.
func NewSubscribersView(client *mailerlite.Client) SubscribersView {
	columns := []components.Column{
		{Title: "EMAIL", Width: 30},
		{Title: "STATUS", Width: 10},
		{Title: "SOURCE", Width: 12},
		{Title: "OPENS", Width: 8},
		{Title: "CLICKS", Width: 8},
		{Title: "SUBSCRIBED", Width: 12},
	}
	table := components.NewTable(columns)
	table.SetEmptyMessage("No subscribers found.")

	return SubscribersView{
		client: client,

		table:   table,
		loading: true,
	}
}

// SetSize sets the view dimensions.
func (v *SubscribersView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.table.SetSize(width, height)
}

// SetFocused sets whether this view is focused.
func (v *SubscribersView) SetFocused(focused bool) {
	v.focused = focused
	v.table.SetFocused(focused)
}

// Loading returns whether the view is loading.
func (v SubscribersView) Loading() bool {
	return v.loading
}

// ItemCount returns the number of items.
func (v SubscribersView) ItemCount() int {
	return len(v.subscribers)
}

// SelectedSubscriber returns the currently selected subscriber.
func (v SubscribersView) SelectedSubscriber() *mailerlite.Subscriber {
	idx := v.table.Cursor()
	if idx >= 0 && idx < len(v.subscribers) {
		return &v.subscribers[idx]
	}
	return nil
}

// Fetch returns a command to fetch subscribers.
func (v SubscribersView) Fetch() tea.Cmd {
	return func() tea.Msg {
		if v.client == nil {
			return types.SubscribersLoadedMsg{Err: nil}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		subscribers, err := sdkclient.FetchAllStringCursor(ctx, func(ctx context.Context, cursor string, perPage int) ([]mailerlite.Subscriber, string, error) {
			root, _, err := v.client.Subscriber.List(ctx, &mailerlite.ListSubscriberOptions{
				Cursor: cursor,
				Limit:  perPage,
			})
			if err != nil {
				return nil, "", sdkclient.WrapError(err)
			}
			return root.Data, root.Meta.NextCursor, nil
		}, 100)

		return types.SubscribersLoadedMsg{
			Subscribers: subscribers,
			Err:         err,
		}
	}
}

// Update handles messages for this view.
func (v SubscribersView) Update(msg tea.Msg) (SubscribersView, tea.Cmd) {
	switch msg := msg.(type) {
	case types.SubscribersLoadedMsg:
		v.loading = false
		v.err = msg.Err
		if msg.Err == nil {
			v.subscribers = msg.Subscribers
			v.updateTable()
		}
	}
	return v, nil
}

// HandleKey handles key events when this view is active.
func (v *SubscribersView) HandleKey(msg tea.KeyMsg) tea.Cmd {
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

func (v *SubscribersView) showDetail() {
	sub := v.SelectedSubscriber()
	if sub == nil {
		return
	}

	v.detail.SetTitle("Subscriber: " + sub.Email)

	subscribed := sub.SubscribedAt
	if t, err := time.Parse("2006-01-02 15:04:05", sub.SubscribedAt); err == nil {
		subscribed = t.Format("2006-01-02 15:04:05")
	}

	v.detail.SetRows([]components.DetailRow{
		{Label: "ID", Value: sub.ID},
		{Label: "Email", Value: sub.Email},
		{Label: "Status", Value: sub.Status},
		{Label: "Source", Value: sub.Source},
		{Label: "Opens", Value: fmt.Sprintf("%d", sub.OpensCount)},
		{Label: "Clicks", Value: fmt.Sprintf("%d", sub.ClicksCount)},
		{Label: "Open Rate", Value: fmt.Sprintf("%.1f%%", sub.OpenRate)},
		{Label: "Click Rate", Value: fmt.Sprintf("%.1f%%", sub.ClickRate)},
		{Label: "Subscribed", Value: subscribed},
	})
	v.detail.SetSize(v.width, v.height)
	v.showingDetail = true
}

func (v *SubscribersView) updateTable() {
	var rows [][]string
	for _, s := range v.subscribers {
		subscribed := ""
		if t, err := time.Parse("2006-01-02 15:04:05", s.SubscribedAt); err == nil {
			subscribed = t.Format("2006-01-02")
		}

		rows = append(rows, []string{
			s.Email,
			statusBadge(s.Status),
			s.Source,
			fmt.Sprintf("%d", s.OpensCount),
			fmt.Sprintf("%d", s.ClicksCount),
			subscribed,
		})
	}
	v.table.SetRows(rows)
	v.table.SetLoading(false)
}

// View renders the subscribers view.
func (v SubscribersView) View() string {
	if v.showingDetail {
		return v.detail.View()
	}
	return v.table.View()
}

// ShowingDetail returns whether the detail view is active.
func (v SubscribersView) ShowingDetail() bool {
	return v.showingDetail
}
