package field

import (
	"context"
	"fmt"

	"github.com/mailerlite/mailerlite-cli/internal/cmdutil"
	"github.com/mailerlite/mailerlite-cli/internal/output"
	"github.com/mailerlite/mailerlite-cli/internal/prompt"
	"github.com/mailerlite/mailerlite-cli/internal/sdkclient"
	"github.com/mailerlite/mailerlite-go"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "field",
	Short: "Manage fields",
	Long:  "List, create, update, and delete subscriber fields.",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(deleteCmd)

	// list flags
	listCmd.Flags().Int("limit", 25, "maximum number of fields to return (0 = all)")
	listCmd.Flags().String("sort", "", "sort order (e.g. name)")

	// create flags
	createCmd.Flags().String("name", "", "field name (required)")
	createCmd.Flags().String("type", "", "field type: text, number, or date (required)")

	// update flags
	updateCmd.Flags().String("name", "", "new field name (required)")
}

// --- list ---

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List fields",
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

	allFields, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]mailerlite.Field, bool, error) {
		opts := &mailerlite.ListFieldOptions{
			Page:  page,
			Limit: perPage,
			Sort:  sort,
		}
		root, _, err := ml.Field.List(ctx, opts)
		if err != nil {
			return nil, false, sdkclient.WrapError(transport, err)
		}
		return root.Data, !root.Links.IsLastPage(), nil
	}, limit)
	if err != nil {
		return err
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(allFields)
	}

	headers := []string{"ID", "NAME", "KEY", "TYPE"}
	var rows [][]string

	for _, f := range allFields {
		rows = append(rows, []string{
			f.Id,
			f.Name,
			f.Key,
			f.Type,
		})
	}

	output.Table(headers, rows)
	return nil
}

// --- create ---

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a field",
	RunE:  runCreate,
}

func runCreate(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	name, _ := c.Flags().GetString("name")
	name, err = prompt.RequireArg(name, "name", "Field name")
	if err != nil {
		return err
	}

	fieldType, _ := c.Flags().GetString("type")
	fieldType, err = prompt.RequireArg(fieldType, "type", "Field type (text, number, date)")
	if err != nil {
		return err
	}

	ctx := context.Background()
	result, _, err := ml.Field.Create(ctx, name, fieldType)
	if err != nil {
		return sdkclient.WrapError(transport, err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	output.Success("Field created successfully. ID: " + result.Data.Id)
	return nil
}

// --- update ---

var updateCmd = &cobra.Command{
	Use:   "update <field_id>",
	Short: "Update a field",
	Args:  cobra.ExactArgs(1),
	RunE:  runUpdate,
}

func runUpdate(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	name, _ := c.Flags().GetString("name")
	name, err = prompt.RequireArg(name, "name", "New field name")
	if err != nil {
		return err
	}

	ctx := context.Background()
	result, _, err := ml.Field.Update(ctx, args[0], name)
	if err != nil {
		return sdkclient.WrapError(transport, err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	output.Success("Field " + args[0] + " updated successfully.")
	return nil
}

// --- delete ---

var deleteCmd = &cobra.Command{
	Use:   "delete <field_id>",
	Short: "Delete a field",
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

func runDelete(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	if !cmdutil.YesFlag(c) && prompt.IsInteractive() {
		ok, err := prompt.Confirm(fmt.Sprintf("Delete field %s?", args[0]))
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	ctx := context.Background()
	_, err = ml.Field.Delete(ctx, args[0])
	if err != nil {
		return sdkclient.WrapError(transport, err)
	}

	output.Success("Field " + args[0] + " deleted successfully.")
	return nil
}
