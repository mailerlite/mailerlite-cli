package webhook

import (
	"context"
	"fmt"
	"strings"

	"github.com/mailerlite/mailerlite-cli/internal/cmdutil"
	"github.com/mailerlite/mailerlite-cli/internal/output"
	"github.com/mailerlite/mailerlite-cli/internal/prompt"
	"github.com/mailerlite/mailerlite-cli/internal/sdkclient"
	"github.com/mailerlite/mailerlite-go"
	"github.com/spf13/cobra"
)

var webhookEvents = []string{
	"subscriber.created",
	"subscriber.updated",
	"subscriber.unsubscribed",
	"subscriber.added_to_group",
	"subscriber.removed_from_group",
	"subscriber.bounced",
	"subscriber.automation_triggered",
	"subscriber.automation_completed",
	"campaign.sent",
	"campaign.draft_created",
}

var Cmd = &cobra.Command{
	Use:   "webhook",
	Short: "Manage webhooks",
	Long:  "List, view, create, update, and delete webhooks.",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(deleteCmd)

	// list flags
	listCmd.Flags().Int("limit", 25, "maximum number of webhooks to return (0 = all)")
	listCmd.Flags().String("sort", "", "sort order")

	// create flags
	createCmd.Flags().String("name", "", "webhook name (required)")
	createCmd.Flags().String("url", "", "webhook URL (required)")
	createCmd.Flags().StringSlice("events", nil, "webhook events (required)")
	createCmd.Flags().Bool("enabled", true, "whether the webhook is enabled")

	// update flags
	updateCmd.Flags().String("name", "", "webhook name")
	updateCmd.Flags().String("url", "", "webhook URL")
	updateCmd.Flags().StringSlice("events", nil, "webhook events")
	updateCmd.Flags().Bool("enabled", true, "whether the webhook is enabled")
}

// --- list ---

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List webhooks",
	RunE:  runList,
}

func runList(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	limit, _ := c.Flags().GetInt("limit")
	sort, _ := c.Flags().GetString("sort")

	ctx := context.Background()

	allWebhooks, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]mailerlite.Webhook, bool, error) {
		opts := &mailerlite.ListWebhookOptions{
			Page:  page,
			Limit: perPage,
			Sort:  sort,
		}
		root, _, err := ml.Webhook.List(ctx, opts)
		if err != nil {
			return nil, false, sdkclient.WrapError(transport, err)
		}
		return root.Data, !root.Links.IsLastPage(), nil
	}, limit)
	if err != nil {
		return err
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(allWebhooks)
	}

	headers := []string{"ID", "NAME", "URL", "ENABLED", "CREATED AT"}
	var rows [][]string

	for _, w := range allWebhooks {
		enabled := "No"
		if w.Enabled {
			enabled = "Yes"
		}
		rows = append(rows, []string{
			w.Id,
			output.Truncate(w.Name, 40),
			output.Truncate(w.Url, 50),
			enabled,
			w.CreatedAt,
		})
	}

	output.Table(headers, rows)
	return nil
}

// --- get ---

var getCmd = &cobra.Command{
	Use:   "get <webhook_id>",
	Short: "Get webhook details",
	Args:  cobra.ExactArgs(1),
	RunE:  runGet,
}

func runGet(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	ctx := context.Background()
	result, _, err := ml.Webhook.Get(ctx, args[0])
	if err != nil {
		return sdkclient.WrapError(transport, err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	d := result.Data

	enabled := "No"
	if d.Enabled {
		enabled = "Yes"
	}

	fmt.Printf("ID:           %s\n", d.Id)
	fmt.Printf("Name:         %s\n", d.Name)
	fmt.Printf("URL:          %s\n", d.Url)
	fmt.Printf("Enabled:      %s\n", enabled)
	fmt.Printf("Created At:   %s\n", d.CreatedAt)
	fmt.Printf("Updated At:   %s\n", d.UpdatedAt)

	fmt.Println()
	fmt.Println("Events:")
	for _, e := range d.Events {
		fmt.Printf("  - %s\n", e)
	}

	return nil
}

// --- create ---

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a webhook",
	Long:  "Create a new webhook.\n\nValid events: " + strings.Join(webhookEvents, ", "),
	RunE:  runCreate,
}

func runCreate(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	name, _ := c.Flags().GetString("name")
	name, err = prompt.RequireArg(name, "name", "Webhook name")
	if err != nil {
		return err
	}

	url, _ := c.Flags().GetString("url")
	url, err = prompt.RequireArg(url, "url", "Webhook URL")
	if err != nil {
		return err
	}

	events, _ := c.Flags().GetStringSlice("events")
	events, err = prompt.RequireSliceArg(events, "events", "Webhook events")
	if err != nil {
		return err
	}

	ctx := context.Background()
	opts := &mailerlite.CreateWebhookOptions{
		Name:   name,
		Url:    url,
		Events: events,
	}

	result, _, err := ml.Webhook.Create(ctx, opts)
	if err != nil {
		return sdkclient.WrapError(transport, err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	output.Success("Webhook created successfully. ID: " + result.Data.Id)
	return nil
}

// --- update ---

var updateCmd = &cobra.Command{
	Use:   "update <webhook_id>",
	Short: "Update a webhook",
	Long:  "Update an existing webhook.\n\nValid events: " + strings.Join(webhookEvents, ", "),
	Args:  cobra.ExactArgs(1),
	RunE:  runUpdate,
}

func runUpdate(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	opts := &mailerlite.UpdateWebhookOptions{
		WebhookID: args[0],
	}

	if c.Flags().Changed("name") {
		name, _ := c.Flags().GetString("name")
		opts.Name = name
	}
	if c.Flags().Changed("url") {
		url, _ := c.Flags().GetString("url")
		opts.Url = url
	}
	if c.Flags().Changed("events") {
		events, _ := c.Flags().GetStringSlice("events")
		opts.Events = events
	}
	if c.Flags().Changed("enabled") {
		enabled, _ := c.Flags().GetBool("enabled")
		if enabled {
			opts.Enabled = "true"
		} else {
			opts.Enabled = "false"
		}
	}

	ctx := context.Background()
	result, _, err := ml.Webhook.Update(ctx, opts)
	if err != nil {
		return sdkclient.WrapError(transport, err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	output.Success("Webhook " + args[0] + " updated successfully.")
	return nil
}

// --- delete ---

var deleteCmd = &cobra.Command{
	Use:   "delete <webhook_id>",
	Short: "Delete a webhook",
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

func runDelete(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	if !cmdutil.YesFlag(c) && prompt.IsInteractive() {
		ok, err := prompt.Confirm(fmt.Sprintf("Delete webhook %s?", args[0]))
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	ctx := context.Background()
	_, err = ml.Webhook.Delete(ctx, args[0])
	if err != nil {
		return sdkclient.WrapError(transport, err)
	}

	output.Success("Webhook " + args[0] + " deleted successfully.")
	return nil
}
