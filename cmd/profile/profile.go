package profile

import (
	"fmt"
	"sort"

	"github.com/mailerlite/mailerlite-cli/internal/cmdutil"
	"github.com/mailerlite/mailerlite-cli/internal/config"
	"github.com/mailerlite/mailerlite-cli/internal/output"
	"github.com/mailerlite/mailerlite-cli/internal/prompt"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage authentication profiles",
}

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runAdd,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	RunE:  runList,
}

var switchCmd = &cobra.Command{
	Use:   "switch <name>",
	Short: "Switch active profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runSwitch,
}

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemove,
}

func init() {
	addCmd.Flags().String("token", "", "API token for this profile")
	Cmd.AddCommand(addCmd, listCmd, switchCmd, removeCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	token, _ := cmd.Flags().GetString("token")

	if token == "" && prompt.IsInteractive() {
		var err error
		token, err = prompt.Input("API Token", "")
		if err != nil {
			return err
		}
	}
	if token == "" {
		return fmt.Errorf("--token is required")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if _, exists := cfg.Profiles[name]; exists {
		if !cmdutil.YesFlag(cmd) && prompt.IsInteractive() {
			ok, err := prompt.Confirm(fmt.Sprintf("Profile %q already exists. Overwrite?", name))
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
		}
	}

	cfg.Profiles[name] = config.Profile{APIToken: token}
	if cfg.ActiveProfile == "" {
		cfg.ActiveProfile = name
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	output.Success(fmt.Sprintf("Profile %q added.", name))
	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	jsonFlag, _ := cmd.Root().PersistentFlags().GetBool("json")
	if jsonFlag {
		profiles := make([]map[string]interface{}, 0, len(cfg.Profiles))
		for name, p := range cfg.Profiles {
			profiles = append(profiles, map[string]interface{}{
				"name":      name,
				"active":    name == cfg.ActiveProfile,
				"has_token": p.APIToken != "",
			})
		}
		return output.JSON(profiles)
	}

	if len(cfg.Profiles) == 0 {
		fmt.Println("No profiles configured. Run 'mailerlite profile add <name>' to create one.")
		return nil
	}

	names := make([]string, 0, len(cfg.Profiles))
	for n := range cfg.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)

	var rows [][]string
	for _, name := range names {
		active := ""
		if name == cfg.ActiveProfile {
			active = "*"
		}
		rows = append(rows, []string{active, name, "token"})
	}

	output.Table([]string{"", "NAME", "METHOD"}, rows)
	return nil
}

func runSwitch(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}

	cfg.ActiveProfile = name
	if err := config.Save(cfg); err != nil {
		return err
	}

	output.Success(fmt.Sprintf("Switched to profile: %s", name))
	return nil
}

func runRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}

	if !cmdutil.YesFlag(cmd) && prompt.IsInteractive() {
		ok, err := prompt.Confirm(fmt.Sprintf("Remove profile %q?", name))
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	delete(cfg.Profiles, name)
	if cfg.ActiveProfile == name {
		cfg.ActiveProfile = ""
		for n := range cfg.Profiles {
			cfg.ActiveProfile = n
			break
		}
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	output.Success(fmt.Sprintf("Profile %q removed.", name))
	return nil
}
