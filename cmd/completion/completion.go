package completion

import (
	"os"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for mailerlite.

To load completions:

Bash:
  $ source <(mailerlite completion bash)
  # Or for permanent:
  $ mailerlite completion bash > /etc/bash_completion.d/mailerlite

Zsh:
  $ mailerlite completion zsh > "${fpath[1]}/_mailerlite"

Fish:
  $ mailerlite completion fish | source
  # Or for permanent:
  $ mailerlite completion fish > ~/.config/fish/completions/mailerlite.fish

PowerShell:
  PS> mailerlite completion powershell | Out-String | Invoke-Expression
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}
