package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Version returns the version string for use by other packages (e.g. user-agent).
func Version() string {
	return version
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of mailerlite",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("mailerlite v%s (%s) built %s\n", version, commit, date)
	},
}
