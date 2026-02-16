package automation

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mailerlite/mailerlite-cli/internal/cmdutil"
	"github.com/mailerlite/mailerlite-cli/internal/output"
	"github.com/mailerlite/mailerlite-cli/internal/sdkclient"
	"github.com/mailerlite/mailerlite-go"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "automation",
	Short: "Manage automations",
	Long:  "List and view automations.",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(subscribersCmd)

	// list flags
	listCmd.Flags().Int("limit", 25, "maximum number of automations to return (0 = all)")
	listCmd.Flags().String("enabled", "", "filter by enabled status (true, false)")

	// subscribers flags
	subscribersCmd.Flags().Int("limit", 25, "maximum number of subscribers to return (0 = all)")
}

// --- list ---

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List automations",
	RunE:  runList,
}

func runList(c *cobra.Command, _ []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	limit, _ := c.Flags().GetInt("limit")
	enabled, _ := c.Flags().GetString("enabled")

	ctx := context.Background()

	automations, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]mailerlite.Automation, bool, error) {
		var filters []mailerlite.Filter
		if enabled != "" {
			filters = append(filters, mailerlite.Filter{Name: "enabled", Value: enabled})
		}

		opts := &mailerlite.ListAutomationOptions{
			Page:  page,
			Limit: perPage,
		}
		if len(filters) > 0 {
			opts.Filters = &filters
		}

		root, _, err := ml.Automation.List(ctx, opts)
		if err != nil {
			return nil, false, sdkclient.WrapError(transport, err)
		}

		return root.Data, !root.Links.IsLastPage(), nil
	}, limit)
	if err != nil {
		return err
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(automations)
	}

	headers := []string{"ID", "NAME", "ENABLED", "EMAILS", "COMPLETED", "IN QUEUE"}
	var rows [][]string

	for _, a := range automations {
		enabledStr := "No"
		if a.Enabled {
			enabledStr = "Yes"
		}
		rows = append(rows, []string{
			a.ID,
			output.Truncate(a.Name, 40),
			enabledStr,
			strconv.Itoa(a.EmailsCount),
			strconv.Itoa(a.Stats.CompletedSubscribersCount),
			strconv.Itoa(a.Stats.SubscribersInQueueCount),
		})
	}

	output.Table(headers, rows)
	return nil
}

// --- get ---

var getCmd = &cobra.Command{
	Use:   "get <automation_id>",
	Short: "Get automation details",
	Args:  cobra.ExactArgs(1),
	RunE:  runGet,
}

func runGet(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	ctx := context.Background()
	result, _, err := ml.Automation.Get(ctx, args[0])
	if err != nil {
		return sdkclient.WrapError(transport, err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	d := result.Data

	enabledStr := "No"
	if d.Enabled {
		enabledStr = "Yes"
	}

	fmt.Printf("ID:           %s\n", d.ID)
	fmt.Printf("Name:         %s\n", d.Name)
	fmt.Printf("Enabled:      %s\n", enabledStr)
	fmt.Printf("Emails:       %d\n", d.EmailsCount)
	fmt.Printf("Created At:   %s\n", d.CreatedAt)

	fmt.Println()
	fmt.Println("Stats:")
	fmt.Printf("  Completed:  %d\n", d.Stats.CompletedSubscribersCount)
	fmt.Printf("  In Queue:   %d\n", d.Stats.SubscribersInQueueCount)
	fmt.Printf("  Sent:       %d\n", d.Stats.Sent)
	fmt.Printf("  Opens:      %d\n", d.Stats.OpensCount)
	fmt.Printf("  Clicks:     %d\n", d.Stats.ClicksCount)

	if len(d.Steps) > 0 {
		fmt.Println()
		fmt.Println("Steps:")
		for _, s := range d.Steps {
			fmt.Printf("  - %s (%s)\n", s.Description, s.Type)
		}
	}

	return nil
}

// --- subscribers ---

var subscribersCmd = &cobra.Command{
	Use:   "subscribers <automation_id>",
	Short: "List automation subscriber activity",
	Args:  cobra.ExactArgs(1),
	RunE:  runSubscribers,
}

func runSubscribers(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	limit, _ := c.Flags().GetInt("limit")

	ctx := context.Background()

	subscribers, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]mailerlite.AutomationSubscriber, bool, error) {
		opts := &mailerlite.ListAutomationSubscriberOptions{
			AutomationID: args[0],
			Page:         page,
			Limit:        perPage,
		}

		root, _, err := ml.Automation.Subscribers(ctx, opts)
		if err != nil {
			return nil, false, sdkclient.WrapError(transport, err)
		}

		return root.Data, !root.Links.IsLastPage(), nil
	}, limit)
	if err != nil {
		return err
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(subscribers)
	}

	headers := []string{"ID", "EMAIL", "STATUS", "DATE"}
	var rows [][]string

	for _, s := range subscribers {
		rows = append(rows, []string{
			s.ID,
			s.Subscriber.Email,
			s.Status,
			s.Date,
		})
	}

	output.Table(headers, rows)
	return nil
}
