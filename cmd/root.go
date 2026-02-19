package cmd

import (
	"github.com/mailerlite/mailerlite-cli/cmd/account"
	"github.com/mailerlite/mailerlite-cli/cmd/auth"
	"github.com/mailerlite/mailerlite-cli/cmd/automation"
	"github.com/mailerlite/mailerlite-cli/cmd/campaign"
	"github.com/mailerlite/mailerlite-cli/cmd/cart"
	"github.com/mailerlite/mailerlite-cli/cmd/cartitem"
	"github.com/mailerlite/mailerlite-cli/cmd/category"
	"github.com/mailerlite/mailerlite-cli/cmd/completion"
	"github.com/mailerlite/mailerlite-cli/cmd/customer"
	"github.com/mailerlite/mailerlite-cli/cmd/dashboard"
	"github.com/mailerlite/mailerlite-cli/cmd/field"
	"github.com/mailerlite/mailerlite-cli/cmd/form"
	"github.com/mailerlite/mailerlite-cli/cmd/group"
	importcmd "github.com/mailerlite/mailerlite-cli/cmd/import"
	"github.com/mailerlite/mailerlite-cli/cmd/order"
	"github.com/mailerlite/mailerlite-cli/cmd/product"
	"github.com/mailerlite/mailerlite-cli/cmd/profile"
	"github.com/mailerlite/mailerlite-cli/cmd/segment"
	"github.com/mailerlite/mailerlite-cli/cmd/shop"
	"github.com/mailerlite/mailerlite-cli/cmd/subscriber"
	"github.com/mailerlite/mailerlite-cli/cmd/timezone"
	"github.com/mailerlite/mailerlite-cli/cmd/webhook"
	"github.com/mailerlite/mailerlite-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "mailerlite",
	Short:         "MailerLite CLI — manage your email marketing from the terminal",
	Long:          "A command-line interface for the MailerLite API. Manage subscribers, campaigns, automations, groups, forms, and more.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.Version = version
	cmdutil.SetVersion(version)
	rootCmd.PersistentFlags().String("profile", "", "config profile to use")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "show HTTP request/response details")
	rootCmd.PersistentFlags().Bool("json", false, "output as JSON")
	rootCmd.PersistentFlags().BoolP("yes", "y", false, "skip confirmation prompts")

	rootCmd.AddCommand(dashboard.Cmd)
	rootCmd.AddCommand(subscriber.Cmd)
	rootCmd.AddCommand(group.Cmd)
	rootCmd.AddCommand(campaign.Cmd)
	rootCmd.AddCommand(automation.Cmd)
	rootCmd.AddCommand(form.Cmd)
	rootCmd.AddCommand(field.Cmd)
	rootCmd.AddCommand(segment.Cmd)
	rootCmd.AddCommand(webhook.Cmd)
	rootCmd.AddCommand(timezone.Cmd)
	rootCmd.AddCommand(shop.Cmd)
	rootCmd.AddCommand(product.Cmd)
	rootCmd.AddCommand(category.Cmd)
	rootCmd.AddCommand(customer.Cmd)
	rootCmd.AddCommand(order.Cmd)
	rootCmd.AddCommand(cart.Cmd)
	rootCmd.AddCommand(cartitem.Cmd)
	rootCmd.AddCommand(importcmd.Cmd)
	rootCmd.AddCommand(account.Cmd)
	rootCmd.AddCommand(auth.Cmd)
	rootCmd.AddCommand(profile.Cmd)
	rootCmd.AddCommand(completion.Cmd)
	rootCmd.AddCommand(versionCmd)
}

func Execute() error {
	return rootCmd.Execute()
}

func IsJSON() bool {
	return cmdutil.JSONFlag(rootCmd)
}
