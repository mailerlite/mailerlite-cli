package account

import (
	"context"
	"fmt"

	"github.com/mailerlite/mailerlite-cli/internal/cmdutil"
	"github.com/mailerlite/mailerlite-cli/internal/config"
	"github.com/mailerlite/mailerlite-cli/internal/output"
	"github.com/mailerlite/mailerlite-cli/internal/prompt"
	"github.com/mailerlite/mailerlite-cli/internal/sdkclient"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "account",
	Short: "Manage MailerLite accounts",
	Long:  "List and switch between MailerLite accounts (requires OAuth authentication).",
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List accounts you have access to",
	RunE:  runList,
}

var switchCmd = &cobra.Command{
	Use:   "switch [account-id]",
	Short: "Switch active account",
	Long:  "Switch to a different MailerLite account. If no account ID is provided, shows an interactive picker.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSwitch,
}

func init() {
	Cmd.AddCommand(listCmd, switchCmd)
}

type accountEntry struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type accountsResponse struct {
	Data []accountEntry `json:"data"`
}

func fetchAccounts(cmd *cobra.Command) ([]accountEntry, error) {
	httpClient, token, _, err := cmdutil.RawHTTPClient(cmd)
	if err != nil {
		return nil, err
	}

	var resp accountsResponse
	_, err = sdkclient.DoRaw(context.Background(), httpClient, token, "GET", "/accounts", nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch accounts: %w", err)
	}
	return resp.Data, nil
}

func runList(cmd *cobra.Command, args []string) error {
	accounts, err := fetchAccounts(cmd)
	if err != nil {
		return err
	}

	activeAccountID := config.GetAccountID(cmdutil.ProfileFlag(cmd))

	jsonFlag := cmdutil.JSONFlag(cmd)
	if jsonFlag {
		return output.JSON(accounts)
	}

	var rows [][]string
	for _, a := range accounts {
		name := a.Name
		if a.ID == activeAccountID {
			name += " (active)"
		}
		rows = append(rows, []string{a.ID, name, a.Status})
	}

	output.Table([]string{"ID", "Name", "Status"}, rows)
	return nil
}

func runSwitch(cmd *cobra.Command, args []string) error {
	var accountID string

	if len(args) > 0 {
		accountID = args[0]
	} else {
		accounts, err := fetchAccounts(cmd)
		if err != nil {
			return err
		}
		if len(accounts) == 0 {
			return fmt.Errorf("no accounts found")
		}
		if len(accounts) == 1 {
			accountID = accounts[0].ID
			output.Success(fmt.Sprintf("Only one account available: %s (%s)", accounts[0].Name, accounts[0].ID))
		} else {
			if !prompt.IsInteractive() {
				return fmt.Errorf("account ID argument is required in non-interactive mode")
			}
			labels := make([]string, len(accounts))
			values := make([]string, len(accounts))
			activeAccountID := config.GetAccountID(cmdutil.ProfileFlag(cmd))
			for i, a := range accounts {
				label := fmt.Sprintf("%s (%s)", a.Name, a.ID)
				if a.ID == activeAccountID {
					label += " [active]"
				}
				labels[i] = label
				values[i] = a.ID
			}
			accountID, err = prompt.SelectLabeled("Select account", labels, values)
			if err != nil {
				return err
			}
		}
	}

	profName := cmdutil.ProfileFlag(cmd)
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if profName == "" {
		profName, _, err = config.ActiveProfile(cfg)
		if err != nil {
			return err
		}
	}

	prof, ok := cfg.Profiles[profName]
	if !ok {
		return fmt.Errorf("profile %q not found", profName)
	}

	prof.AccountID = accountID
	cfg.Profiles[profName] = prof
	if err := config.Save(cfg); err != nil {
		return err
	}

	output.Success(fmt.Sprintf("Switched to account: %s", accountID))
	return nil
}
