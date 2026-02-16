package subscriber

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mailerlite/mailerlite-cli/internal/cmdutil"
	"github.com/mailerlite/mailerlite-cli/internal/output"
	"github.com/mailerlite/mailerlite-cli/internal/prompt"
	"github.com/mailerlite/mailerlite-cli/internal/sdkclient"
	"github.com/mailerlite/mailerlite-go"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "subscriber",
	Short: "Manage subscribers",
	Long:  "List, view, create, update, and delete subscribers.",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(countCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(upsertCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(forgetCmd)

	// list flags
	listCmd.Flags().Int("limit", 25, "maximum number of subscribers to return (0 = all)")
	listCmd.Flags().String("status", "", "filter by status (active, unsubscribed, unconfirmed, bounced, junk)")
	listCmd.Flags().String("email", "", "filter by email address")

	// upsert flags
	upsertCmd.Flags().String("email", "", "subscriber email (required)")
	upsertCmd.Flags().String("status", "", "subscriber status")
	upsertCmd.Flags().StringSlice("groups", nil, "group IDs to assign")
	upsertCmd.Flags().StringSlice("fields", nil, "custom fields as key=value pairs")

	// update flags
	updateCmd.Flags().String("email", "", "subscriber email")
	updateCmd.Flags().String("status", "", "subscriber status")
	updateCmd.Flags().StringSlice("fields", nil, "custom fields as key=value pairs")
}

// --- list ---

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List subscribers",
	RunE:  runList,
}

func runList(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	limit, _ := c.Flags().GetInt("limit")
	status, _ := c.Flags().GetString("status")
	email, _ := c.Flags().GetString("email")

	ctx := context.Background()

	var filters []mailerlite.Filter
	if status != "" {
		filters = append(filters, *mailerlite.NewFilter("status", status))
	}
	if email != "" {
		filters = append(filters, *mailerlite.NewFilter("email", email))
	}

	subscribers, err := sdkclient.FetchAllStringCursor(ctx, func(ctx context.Context, cursor string, perPage int) ([]mailerlite.Subscriber, string, error) {
		opts := &mailerlite.ListSubscriberOptions{
			Cursor: cursor,
			Limit:  perPage,
		}
		if len(filters) > 0 {
			opts.Filters = &filters
		}

		root, _, err := ml.Subscriber.List(ctx, opts)
		if err != nil {
			return nil, "", sdkclient.WrapError(transport, err)
		}

		return root.Data, root.Meta.NextCursor, nil
	}, limit)
	if err != nil {
		return err
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(subscribers)
	}

	headers := []string{"EMAIL", "STATUS", "SOURCE", "OPENS", "CLICKS", "SUBSCRIBED AT"}
	var rows [][]string

	for _, s := range subscribers {
		rows = append(rows, []string{
			output.Truncate(s.Email, 40),
			s.Status,
			s.Source,
			strconv.Itoa(s.OpensCount),
			strconv.Itoa(s.ClicksCount),
			s.SubscribedAt,
		})
	}

	output.Table(headers, rows)
	return nil
}

// --- count ---

var countCmd = &cobra.Command{
	Use:   "count",
	Short: "Get total subscriber count",
	RunE:  runCount,
}

func runCount(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	ctx := context.Background()
	result, _, err := ml.Subscriber.Count(ctx)
	if err != nil {
		return sdkclient.WrapError(transport, err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	fmt.Printf("Total subscribers: %d\n", result.Total)
	return nil
}

// --- get ---

var getCmd = &cobra.Command{
	Use:   "get <id_or_email>",
	Short: "Get subscriber details",
	Args:  cobra.ExactArgs(1),
	RunE:  runGet,
}

func runGet(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	opts := &mailerlite.GetSubscriberOptions{}
	if strings.Contains(args[0], "@") {
		opts.Email = args[0]
	} else {
		opts.SubscriberID = args[0]
	}

	ctx := context.Background()
	result, _, err := ml.Subscriber.Get(ctx, opts)
	if err != nil {
		return sdkclient.WrapError(transport, err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	s := result.Data
	fmt.Printf("ID:            %s\n", s.ID)
	fmt.Printf("Email:         %s\n", s.Email)
	fmt.Printf("Status:        %s\n", s.Status)
	fmt.Printf("Source:        %s\n", s.Source)
	fmt.Printf("Opens:         %d\n", s.OpensCount)
	fmt.Printf("Clicks:        %d\n", s.ClicksCount)
	fmt.Printf("Open Rate:     %.2f%%\n", s.OpenRate)
	fmt.Printf("Click Rate:    %.2f%%\n", s.ClickRate)
	fmt.Printf("Subscribed At: %s\n", s.SubscribedAt)
	fmt.Printf("Created At:    %s\n", s.CreatedAt)
	fmt.Printf("Updated At:    %s\n", s.UpdatedAt)

	if len(s.Fields) > 0 {
		fmt.Println()
		fmt.Println("Fields:")
		for k, v := range s.Fields {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}

	if len(s.Groups) > 0 {
		fmt.Println()
		fmt.Println("Groups:")
		for _, g := range s.Groups {
			fmt.Printf("  - %s (%s)\n", g.Name, g.ID)
		}
	}

	return nil
}

// --- upsert ---

var upsertCmd = &cobra.Command{
	Use:   "upsert",
	Short: "Create or update a subscriber",
	RunE:  runUpsert,
}

func runUpsert(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	email, _ := c.Flags().GetString("email")
	email, err = prompt.RequireArg(email, "email", "Subscriber email")
	if err != nil {
		return err
	}

	subscriber := &mailerlite.UpsertSubscriber{
		Email: email,
	}

	if c.Flags().Changed("status") {
		status, _ := c.Flags().GetString("status")
		subscriber.Status = status
	}
	if c.Flags().Changed("groups") {
		groups, _ := c.Flags().GetStringSlice("groups")
		subscriber.Groups = groups
	}
	if c.Flags().Changed("fields") {
		fieldPairs, _ := c.Flags().GetStringSlice("fields")
		fields, err := parseFields(fieldPairs)
		if err != nil {
			return err
		}
		subscriber.Fields = fields
	}

	ctx := context.Background()
	result, _, err := ml.Subscriber.Upsert(ctx, subscriber)
	if err != nil {
		return sdkclient.WrapError(transport, err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	output.Success("Subscriber upserted successfully. ID: " + result.Data.ID)
	return nil
}

// --- update ---

var updateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a subscriber",
	Args:  cobra.ExactArgs(1),
	RunE:  runUpdate,
}

func runUpdate(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	subscriber := &mailerlite.UpdateSubscriber{
		ID: args[0],
	}

	if c.Flags().Changed("email") {
		email, _ := c.Flags().GetString("email")
		subscriber.Email = email
	}
	if c.Flags().Changed("status") {
		status, _ := c.Flags().GetString("status")
		subscriber.Status = status
	}
	if c.Flags().Changed("fields") {
		fieldPairs, _ := c.Flags().GetStringSlice("fields")
		fields, err := parseFields(fieldPairs)
		if err != nil {
			return err
		}
		subscriber.Fields = fields
	}

	ctx := context.Background()
	result, _, err := ml.Subscriber.Update(ctx, subscriber)
	if err != nil {
		return sdkclient.WrapError(transport, err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	output.Success("Subscriber " + args[0] + " updated successfully.")
	return nil
}

// --- delete ---

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a subscriber",
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

func runDelete(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	if !cmdutil.YesFlag(c) && prompt.IsInteractive() {
		ok, err := prompt.Confirm("Delete subscriber " + args[0] + "?")
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	ctx := context.Background()
	_, err = ml.Subscriber.Delete(ctx, args[0])
	if err != nil {
		return sdkclient.WrapError(transport, err)
	}

	output.Success("Subscriber " + args[0] + " deleted successfully.")
	return nil
}

// --- forget ---

var forgetCmd = &cobra.Command{
	Use:   "forget <id>",
	Short: "Forget a subscriber (GDPR)",
	Long:  "Permanently forget a subscriber and all their data. This action cannot be undone.",
	Args:  cobra.ExactArgs(1),
	RunE:  runForget,
}

func runForget(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	if !cmdutil.YesFlag(c) && prompt.IsInteractive() {
		ok, err := prompt.Confirm("Permanently forget subscriber " + args[0] + "? This cannot be undone.")
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	ctx := context.Background()
	_, _, err = ml.Subscriber.Forget(ctx, args[0])
	if err != nil {
		return sdkclient.WrapError(transport, err)
	}

	output.Success("Subscriber " + args[0] + " forgotten successfully.")
	return nil
}

// parseFields converts "key=value" pairs into a map for subscriber fields.
func parseFields(pairs []string) (map[string]interface{}, error) {
	fields := make(map[string]interface{}, len(pairs))
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid field format %q: expected key=value", pair)
		}
		fields[parts[0]] = parts[1]
	}
	return fields, nil
}
