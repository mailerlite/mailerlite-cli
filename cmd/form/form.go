package form

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
	Use:   "form",
	Short: "Manage forms",
	Long:  "List, view, update, and delete forms.",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(subscribersCmd)

	// list flags
	listCmd.Flags().Int("limit", 25, "maximum number of forms to return (0 = all)")
	listCmd.Flags().String("type", "popup", "form type (popup, embedded, promotion)")
	listCmd.Flags().String("sort", "", "sort field")

	// update flags
	updateCmd.Flags().String("name", "", "form name (required)")

	// subscribers flags
	subscribersCmd.Flags().Int("limit", 25, "maximum number of subscribers to return (0 = all)")
}

// --- list ---

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List forms",
	RunE:  runList,
}

func runList(c *cobra.Command, _ []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	limit, _ := c.Flags().GetInt("limit")
	formType, _ := c.Flags().GetString("type")
	sort, _ := c.Flags().GetString("sort")

	ctx := context.Background()

	forms, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]mailerlite.Form, bool, error) {
		opts := &mailerlite.ListFormOptions{
			Type:  formType,
			Page:  page,
			Limit: perPage,
			Sort:  sort,
		}

		root, _, err := ml.Form.List(ctx, opts)
		if err != nil {
			return nil, false, sdkclient.WrapError(transport, err)
		}

		return root.Data, !root.Links.IsLastPage(), nil
	}, limit)
	if err != nil {
		return err
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(forms)
	}

	headers := []string{"ID", "NAME", "TYPE", "ACTIVE", "CONVERSIONS", "OPENS"}
	var rows [][]string

	for _, f := range forms {
		active := "No"
		if f.Active {
			active = "Yes"
		}
		rows = append(rows, []string{
			f.Id,
			output.Truncate(f.Name, 40),
			f.Type,
			active,
			strconv.Itoa(f.ConversionsCount),
			strconv.Itoa(f.OpensCount),
		})
	}

	output.Table(headers, rows)
	return nil
}

// --- get ---

var getCmd = &cobra.Command{
	Use:   "get <form_id>",
	Short: "Get form details",
	Args:  cobra.ExactArgs(1),
	RunE:  runGet,
}

func runGet(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	ctx := context.Background()
	result, _, err := ml.Form.Get(ctx, args[0])
	if err != nil {
		return sdkclient.WrapError(transport, err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	d := result.Data

	active := "No"
	if d.Active {
		active = "Yes"
	}

	fmt.Printf("ID:           %s\n", d.Id)
	fmt.Printf("Name:         %s\n", d.Name)
	fmt.Printf("Type:         %s\n", d.Type)
	fmt.Printf("Active:       %s\n", active)
	fmt.Printf("Conversions:  %d\n", d.ConversionsCount)
	fmt.Printf("Opens:        %d\n", d.OpensCount)
	fmt.Printf("Created At:   %s\n", d.CreatedAt)

	return nil
}

// --- update ---

var updateCmd = &cobra.Command{
	Use:   "update <form_id>",
	Short: "Update a form",
	Args:  cobra.ExactArgs(1),
	RunE:  runUpdate,
}

func runUpdate(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	name, _ := c.Flags().GetString("name")
	name, err = prompt.RequireArg(name, "name", "Form name")
	if err != nil {
		return err
	}

	ctx := context.Background()
	result, _, err := ml.Form.Update(ctx, args[0], name)
	if err != nil {
		return sdkclient.WrapError(transport, err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	output.Success("Form " + args[0] + " updated successfully.")
	return nil
}

// --- delete ---

var deleteCmd = &cobra.Command{
	Use:   "delete <form_id>",
	Short: "Delete a form",
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

func runDelete(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	if !cmdutil.YesFlag(c) && prompt.IsInteractive() {
		ok, err := prompt.Confirm("Are you sure you want to delete form " + args[0] + "?")
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	ctx := context.Background()
	_, err = ml.Form.Delete(ctx, args[0])
	if err != nil {
		return sdkclient.WrapError(transport, err)
	}

	output.Success("Form " + args[0] + " deleted successfully.")
	return nil
}

// --- subscribers ---

var subscribersCmd = &cobra.Command{
	Use:   "subscribers <form_id>",
	Short: "List form subscribers",
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

	subscribers, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]mailerlite.Subscriber, bool, error) {
		opts := &mailerlite.ListFormSubscriberOptions{
			FormID: args[0],
			Page:   page,
			Limit:  perPage,
		}

		root, _, err := ml.Form.Subscribers(ctx, opts)
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

	headers := []string{"ID", "EMAIL", "STATUS", "CREATED AT"}
	var rows [][]string

	for _, s := range subscribers {
		rows = append(rows, []string{
			s.ID,
			s.Email,
			s.Status,
			s.CreatedAt,
		})
	}

	output.Table(headers, rows)
	return nil
}
