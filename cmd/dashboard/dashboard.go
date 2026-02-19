package dashboard

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mailerlite/mailerlite-cli/internal/cmdutil"
	"github.com/mailerlite/mailerlite-cli/internal/tui"
	"github.com/spf13/cobra"
)

// Cmd is the dashboard command.
var Cmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Interactive dashboard for managing MailerLite",
	Long: `Launch a terminal UI for browsing subscribers, campaigns, automations, and more.

The dashboard provides a lazygit-style interface with:
  - Sidebar navigation between views
  - Vim-style keybindings (j/k to navigate, Enter to select)
  - Real-time data from your MailerLite account

Press ? for help or q to quit.`,
	RunE: runDashboard,
}

func runDashboard(cmd *cobra.Command, _ []string) error {
	client, err := cmdutil.NewSDKClient(cmd)
	if err != nil {
		return err
	}

	profile := cmdutil.ProfileFlag(cmd)
	if profile == "" {
		profile = "default"
	}

	app := tui.NewApp(client, profile)

	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
