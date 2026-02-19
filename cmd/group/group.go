package group

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mailerlite/mailerlite-cli/internal/cmdutil"
	"github.com/mailerlite/mailerlite-cli/internal/output"
	"github.com/mailerlite/mailerlite-cli/internal/prompt"
	"github.com/mailerlite/mailerlite-cli/internal/sdkclient"
	"github.com/mailerlite/mailerlite-go"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "group",
	Short: "Manage groups",
	Long:  "List, create, update, and delete groups. Manage group subscriber assignments.",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(subscribersCmd)
	Cmd.AddCommand(assignCmd)
	Cmd.AddCommand(unassignCmd)

	// list flags
	listCmd.Flags().Int("limit", 25, "maximum number of groups to return (0 = all)")
	listCmd.Flags().String("sort", "", "sort field (e.g. name, created_at)")

	// create flags
	createCmd.Flags().String("name", "", "group name (required)")

	// update flags
	updateCmd.Flags().String("name", "", "group name (required)")

	// subscribers flags
	subscribersCmd.Flags().Int("limit", 25, "maximum number of subscribers to return (0 = all)")
}

// --- list ---

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List groups",
	RunE:  runList,
}

func runList(c *cobra.Command, args []string) error {
	ml, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	limit, _ := c.Flags().GetInt("limit")
	sort, _ := c.Flags().GetString("sort")

	ctx := context.Background()

	groups, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]mailerlite.Group, bool, error) {
		opts := &mailerlite.ListGroupOptions{
			Page:  page,
			Limit: perPage,
			Sort:  sort,
		}

		root, _, err := ml.Group.List(ctx, opts)
		if err != nil {
			return nil, false, sdkclient.WrapError(err)
		}

		hasNext := !root.Links.IsLastPage()
		return root.Data, hasNext, nil
	}, limit)
	if err != nil {
		return err
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(groups)
	}

	headers := []string{"ID", "NAME", "ACTIVE", "SENT", "OPENS", "CLICK RATE", "CREATED AT"}
	var rows [][]string

	for _, g := range groups {
		rows = append(rows, []string{
			g.ID,
			output.Truncate(g.Name, 40),
			strconv.Itoa(g.ActiveCount),
			strconv.Itoa(g.SentCount),
			strconv.Itoa(g.OpensCount),
			g.ClickRate.String,
			g.CreatedAt,
		})
	}

	output.Table(headers, rows)
	return nil
}

// --- create ---

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a group",
	RunE:  runCreate,
}

func runCreate(c *cobra.Command, args []string) error {
	ml, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	name, _ := c.Flags().GetString("name")
	name, err = prompt.RequireArg(name, "name", "Group name")
	if err != nil {
		return err
	}

	ctx := context.Background()
	result, _, err := ml.Group.Create(ctx, name)
	if err != nil {
		return sdkclient.WrapError(err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	output.Success("Group created successfully. ID: " + result.Data.ID)
	return nil
}

// --- update ---

var updateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a group",
	Args:  cobra.ExactArgs(1),
	RunE:  runUpdate,
}

func runUpdate(c *cobra.Command, args []string) error {
	ml, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	name, _ := c.Flags().GetString("name")
	name, err = prompt.RequireArg(name, "name", "Group name")
	if err != nil {
		return err
	}

	ctx := context.Background()
	result, _, err := ml.Group.Update(ctx, args[0], name)
	if err != nil {
		return sdkclient.WrapError(err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	output.Success("Group " + args[0] + " updated successfully.")
	return nil
}

// --- delete ---

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a group",
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

func runDelete(c *cobra.Command, args []string) error {
	ml, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	if !cmdutil.YesFlag(c) && prompt.IsInteractive() {
		ok, err := prompt.Confirm("Delete group " + args[0] + "?")
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	ctx := context.Background()
	_, err = ml.Group.Delete(ctx, args[0])
	if err != nil {
		return sdkclient.WrapError(err)
	}

	output.Success("Group " + args[0] + " deleted successfully.")
	return nil
}

// --- subscribers ---

var subscribersCmd = &cobra.Command{
	Use:   "subscribers <group_id>",
	Short: "List subscribers in a group",
	Args:  cobra.ExactArgs(1),
	RunE:  runSubscribers,
}

func runSubscribers(c *cobra.Command, args []string) error {
	ml, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	limit, _ := c.Flags().GetInt("limit")
	groupID := args[0]

	ctx := context.Background()

	subscribers, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]mailerlite.Subscriber, bool, error) {
		opts := &mailerlite.ListGroupSubscriberOptions{
			GroupID: groupID,
			Page:    page,
			Limit:   perPage,
		}

		root, _, err := ml.Group.Subscribers(ctx, opts)
		if err != nil {
			return nil, false, sdkclient.WrapError(err)
		}

		hasNext := !root.Links.IsLastPage()
		return root.Data, hasNext, nil
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

// --- assign ---

var assignCmd = &cobra.Command{
	Use:   "assign <group_id> <subscriber_id>",
	Short: "Assign a subscriber to a group",
	Args:  cobra.ExactArgs(2),
	RunE:  runAssign,
}

func runAssign(c *cobra.Command, args []string) error {
	ml, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	ctx := context.Background()
	_, _, err = ml.Group.Assign(ctx, args[0], args[1])
	if err != nil {
		return sdkclient.WrapError(err)
	}

	output.Success(fmt.Sprintf("Subscriber %s assigned to group %s successfully.", args[1], args[0]))
	return nil
}

// --- unassign ---

var unassignCmd = &cobra.Command{
	Use:   "unassign <group_id> <subscriber_id>",
	Short: "Unassign a subscriber from a group",
	Args:  cobra.ExactArgs(2),
	RunE:  runUnassign,
}

func runUnassign(c *cobra.Command, args []string) error {
	ml, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	ctx := context.Background()
	_, err = ml.Group.UnAssign(ctx, args[0], args[1])
	if err != nil {
		return sdkclient.WrapError(err)
	}

	output.Success(fmt.Sprintf("Subscriber %s unassigned from group %s successfully.", args[1], args[0]))
	return nil
}
