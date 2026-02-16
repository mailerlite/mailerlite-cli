package types

import "github.com/mailerlite/mailerlite-go"

// ViewType represents the different views in the dashboard.
type ViewType int

const (
	ViewSubscribers ViewType = iota
	ViewCampaigns
	ViewAutomations
	ViewGroups
	ViewForms
)

func (v ViewType) String() string {
	switch v {
	case ViewSubscribers:
		return "Subscribers"
	case ViewCampaigns:
		return "Campaigns"
	case ViewAutomations:
		return "Automations"
	case ViewGroups:
		return "Groups"
	case ViewForms:
		return "Forms"
	default:
		return "Unknown"
	}
}

// ViewInfo contains display information for a view.
type ViewInfo struct {
	Type  ViewType
	Label string
	Icon  string
}

// AllViews returns all available views.
func AllViews() []ViewInfo {
	return []ViewInfo{
		{ViewSubscribers, "Subscribers", "◉"},
		{ViewCampaigns, "Campaigns", "◈"},
		{ViewAutomations, "Automations", "◆"},
		{ViewGroups, "Groups", "◇"},
		{ViewForms, "Forms", "◌"},
	}
}

// Data loading messages

// SubscribersLoadedMsg is sent when subscribers are fetched.
type SubscribersLoadedMsg struct {
	Subscribers []mailerlite.Subscriber
	Err         error
}

// CampaignsLoadedMsg is sent when campaigns are fetched.
type CampaignsLoadedMsg struct {
	Campaigns []mailerlite.Campaign
	Err       error
}

// AutomationsLoadedMsg is sent when automations are fetched.
type AutomationsLoadedMsg struct {
	Automations []mailerlite.Automation
	Err         error
}

// GroupsLoadedMsg is sent when groups are fetched.
type GroupsLoadedMsg struct {
	Groups []mailerlite.Group
	Err    error
}

// FormsLoadedMsg is sent when forms are fetched.
type FormsLoadedMsg struct {
	Forms []mailerlite.Form
	Err   error
}

// Control messages

// RefreshMsg triggers a refresh of the current view.
type RefreshMsg struct{}

// ProfileChangedMsg is sent when the profile changes.
type ProfileChangedMsg struct {
	Profile string
}

// ErrorMsg wraps an error for display.
type ErrorMsg struct {
	Err error
}

func (e ErrorMsg) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return ""
}
