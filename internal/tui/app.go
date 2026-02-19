package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mailerlite/mailerlite-cli/internal/tui/components"
	"github.com/mailerlite/mailerlite-cli/internal/tui/theme"
	"github.com/mailerlite/mailerlite-cli/internal/tui/types"
	"github.com/mailerlite/mailerlite-cli/internal/tui/views"
	"github.com/mailerlite/mailerlite-go"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Primary).
			Padding(0, 1)

	headerBarStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(theme.Muted)

	contentStyle = lipgloss.NewStyle().
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(theme.Error)
)

// FocusArea represents which area of the UI is focused.
type FocusArea int

const (
	FocusSidebar FocusArea = iota
	FocusContent
)

// App is the main TUI application model.
type App struct {
	// SDK
	client *mailerlite.Client

	profile string

	// Components
	sidebar   components.Sidebar
	statusbar components.StatusBar
	spinner   components.Spinner
	help      components.Help
	keys      KeyMap

	// Views
	subscribers views.SubscribersView
	campaigns   views.CampaignsView
	automations views.AutomationsView
	groups      views.GroupsView
	forms       views.FormsView

	// State
	activeView  types.ViewType
	focus       FocusArea
	width       int
	height      int
	showHelp    bool
	err         error
	initialized bool
}

// NewApp creates a new TUI application.
func NewApp(client *mailerlite.Client, profile string) *App {
	keys := DefaultKeyMap()

	app := &App{
		client: client,

		profile:   profile,
		keys:      keys,
		sidebar:   components.NewSidebar(),
		statusbar: components.NewStatusBar(),
		spinner:   components.NewSpinner("Loading..."),
		help:      components.NewHelp(keys.HelpBindings()),
		focus:     FocusContent,
	}

	// Initialize views
	app.subscribers = views.NewSubscribersView(client)
	app.campaigns = views.NewCampaignsView(client)
	app.automations = views.NewAutomationsView(client)
	app.groups = views.NewGroupsView(client)
	app.forms = views.NewFormsView(client)

	// Set initial focus
	app.sidebar.SetFocused(false)
	app.subscribers.SetFocused(true)

	return app
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.spinner.Init(),
		a.fetchCurrentView(),
	)
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.updateLayout()
		if !a.initialized {
			a.initialized = true
		}

	case tea.KeyMsg:
		// Handle help overlay first
		if a.showHelp {
			if key.Matches(msg, a.keys.Help) || key.Matches(msg, a.keys.Back) {
				a.showHelp = false
				return a, nil
			}
			return a, nil
		}

		// Global keys
		switch {
		case key.Matches(msg, a.keys.Quit):
			return a, tea.Quit
		case key.Matches(msg, a.keys.Help):
			a.showHelp = true
			return a, nil
		case key.Matches(msg, a.keys.Tab):
			a.toggleFocus()
			return a, nil
		case key.Matches(msg, a.keys.View1):
			return a, a.switchView(types.ViewSubscribers)
		case key.Matches(msg, a.keys.View2):
			return a, a.switchView(types.ViewCampaigns)
		case key.Matches(msg, a.keys.View3):
			return a, a.switchView(types.ViewAutomations)
		case key.Matches(msg, a.keys.View4):
			return a, a.switchView(types.ViewGroups)
		case key.Matches(msg, a.keys.View5):
			return a, a.switchView(types.ViewForms)
		}

		// Focus-specific keys
		if a.focus == FocusSidebar {
			cmd := a.handleSidebarKey(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		} else {
			cmd := a.handleContentKey(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	// Handle data loaded messages
	case types.SubscribersLoadedMsg:
		a.subscribers, _ = a.subscribers.Update(msg)
		a.updateStatusBar()
	case types.CampaignsLoadedMsg:
		a.campaigns, _ = a.campaigns.Update(msg)
		a.updateStatusBar()
	case types.AutomationsLoadedMsg:
		a.automations, _ = a.automations.Update(msg)
		a.updateStatusBar()
	case types.GroupsLoadedMsg:
		a.groups, _ = a.groups.Update(msg)
		a.updateStatusBar()
	case types.FormsLoadedMsg:
		a.forms, _ = a.forms.Update(msg)
		a.updateStatusBar()

	case types.ErrorMsg:
		a.err = msg.Err
	}

	// Update spinner
	var spinnerCmd tea.Cmd
	a.spinner, spinnerCmd = a.spinner.Update(msg)
	if spinnerCmd != nil {
		cmds = append(cmds, spinnerCmd)
	}

	return a, tea.Batch(cmds...)
}

func (a *App) toggleFocus() {
	if a.focus == FocusSidebar {
		a.focus = FocusContent
		a.sidebar.SetFocused(false)
		a.setCurrentViewFocused(true)
	} else {
		a.focus = FocusSidebar
		a.sidebar.SetFocused(true)
		a.setCurrentViewFocused(false)
	}
}

func (a *App) handleSidebarKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, a.keys.Down):
		a.sidebar.Next()
		return a.switchView(a.sidebar.Active())
	case key.Matches(msg, a.keys.Up):
		a.sidebar.Prev()
		return a.switchView(a.sidebar.Active())
	case key.Matches(msg, a.keys.Enter), key.Matches(msg, a.keys.Right):
		a.focus = FocusContent
		a.sidebar.SetFocused(false)
		a.setCurrentViewFocused(true)
	}
	return nil
}

func (a *App) handleContentKey(msg tea.KeyMsg) tea.Cmd {
	switch a.activeView {
	case types.ViewSubscribers:
		return a.subscribers.HandleKey(msg)
	case types.ViewCampaigns:
		return a.campaigns.HandleKey(msg)
	case types.ViewAutomations:
		return a.automations.HandleKey(msg)
	case types.ViewGroups:
		return a.groups.HandleKey(msg)
	case types.ViewForms:
		return a.forms.HandleKey(msg)
	}
	return nil
}

func (a *App) switchView(v types.ViewType) tea.Cmd {
	if a.activeView == v {
		return nil
	}

	a.setCurrentViewFocused(false)
	a.activeView = v
	a.sidebar.SetActive(v)
	a.setCurrentViewFocused(a.focus == FocusContent)
	a.updateStatusBar()

	return a.fetchCurrentView()
}

func (a *App) setCurrentViewFocused(focused bool) {
	switch a.activeView {
	case types.ViewSubscribers:
		a.subscribers.SetFocused(focused)
	case types.ViewCampaigns:
		a.campaigns.SetFocused(focused)
	case types.ViewAutomations:
		a.automations.SetFocused(focused)
	case types.ViewGroups:
		a.groups.SetFocused(focused)
	case types.ViewForms:
		a.forms.SetFocused(focused)
	}
}

func (a *App) fetchCurrentView() tea.Cmd {
	a.spinner.Start()
	a.spinner.SetLabel("Loading " + a.activeView.String() + "...")

	switch a.activeView {
	case types.ViewSubscribers:
		return a.subscribers.Fetch()
	case types.ViewCampaigns:
		return a.campaigns.Fetch()
	case types.ViewAutomations:
		return a.automations.Fetch()
	case types.ViewGroups:
		return a.groups.Fetch()
	case types.ViewForms:
		return a.forms.Fetch()
	}
	return nil
}

func (a *App) updateLayout() {
	// Header takes 2 lines, status bar takes 2 lines
	contentHeight := a.height - 4

	a.sidebar.SetHeight(contentHeight)
	a.statusbar.SetWidth(a.width)
	a.help.SetSize(a.width, a.height)

	// Content width is total minus sidebar
	contentWidth := a.width - a.sidebar.Width() - 2

	a.subscribers.SetSize(contentWidth, contentHeight)
	a.campaigns.SetSize(contentWidth, contentHeight)
	a.automations.SetSize(contentWidth, contentHeight)
	a.groups.SetSize(contentWidth, contentHeight)
	a.forms.SetSize(contentWidth, contentHeight)

	a.updateStatusBar()
}

func (a *App) updateStatusBar() {
	a.statusbar.SetProfile(a.profile)

	// Get current view info
	viewName := a.activeView.String()
	itemCount := 0
	loading := false

	switch a.activeView {
	case types.ViewSubscribers:
		itemCount = a.subscribers.ItemCount()
		loading = a.subscribers.Loading()
	case types.ViewCampaigns:
		itemCount = a.campaigns.ItemCount()
		loading = a.campaigns.Loading()
	case types.ViewAutomations:
		itemCount = a.automations.ItemCount()
		loading = a.automations.Loading()
	case types.ViewGroups:
		itemCount = a.groups.ItemCount()
		loading = a.groups.Loading()
	case types.ViewForms:
		itemCount = a.forms.ItemCount()
		loading = a.forms.Loading()
	}

	if loading {
		a.statusbar.SetLeft(viewName)
		a.statusbar.SetLoading(true, "Loading...")
		a.spinner.Start()
	} else {
		a.statusbar.SetLeft(fmt.Sprintf("%s (%d)", viewName, itemCount))
		a.statusbar.SetLoading(false, "")
		a.spinner.Stop()
	}
}

// View implements tea.Model.
func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Initializing..."
	}

	var b strings.Builder

	// Header
	header := a.renderHeader()
	b.WriteString(header)
	b.WriteString("\n")

	// Main content area
	mainContent := a.renderMainContent()
	b.WriteString(mainContent)

	// Status bar
	b.WriteString("\n")
	b.WriteString(a.statusbar.View())

	// Help overlay (rendered on top)
	if a.showHelp {
		// Clear and render help overlay
		return a.help.View()
	}

	return b.String()
}

func (a *App) renderHeader() string {
	title := headerStyle.Render("MailerLite Dashboard")
	profile := lipgloss.NewStyle().Foreground(theme.Muted).Render("profile: " + a.profile)

	// Calculate spacing
	gap := a.width - lipgloss.Width(title) - lipgloss.Width(profile) - 4
	if gap < 1 {
		gap = 1
	}

	content := title + strings.Repeat(" ", gap) + profile
	return headerBarStyle.Width(a.width).Render(content)
}

func (a *App) renderMainContent() string {
	sidebar := a.sidebar.View()

	// Render active view
	var content string
	switch a.activeView {
	case types.ViewSubscribers:
		content = a.subscribers.View()
	case types.ViewCampaigns:
		content = a.campaigns.View()
	case types.ViewAutomations:
		content = a.automations.View()
	case types.ViewGroups:
		content = a.groups.View()
	case types.ViewForms:
		content = a.forms.View()
	}

	// Add error display if present
	if a.err != nil {
		content = errorStyle.Render("Error: "+a.err.Error()) + "\n\n" + content
	}

	contentWidth := a.width - a.sidebar.Width() - 4
	styledContent := contentStyle.Width(contentWidth).Render(content)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, styledContent)
}
