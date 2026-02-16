package shop

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/mailerlite/mailerlite-cli/internal/cmdutil"
	"github.com/mailerlite/mailerlite-cli/internal/ecommerce"
	"github.com/mailerlite/mailerlite-cli/internal/output"
	"github.com/mailerlite/mailerlite-cli/internal/prompt"
	"github.com/mailerlite/mailerlite-cli/internal/sdkclient"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "shop",
	Short: "Manage e-commerce shops",
	Long:  "List, create, update, and delete e-commerce shops.",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(countCmd)

	// list flags
	listCmd.Flags().Int("limit", 25, "maximum number of shops to return (0 = all)")

	// create flags
	createCmd.Flags().String("name", "", "shop name (required)")
	createCmd.Flags().String("url", "", "shop URL (required)")

	// update flags
	updateCmd.Flags().String("name", "", "shop name")
	updateCmd.Flags().String("url", "", "shop URL")
}

// list
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List shops",
	RunE: func(cmd *cobra.Command, args []string) error {
		httpClient, apiKey, _, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		ctx := context.Background()

		shops, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]ecommerce.Shop, bool, error) {
			path := "/ecommerce/shops?page=" + strconv.Itoa(page) + "&limit=" + strconv.Itoa(perPage)
			var result ecommerce.RootShops
			_, err := sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodGet, path, nil, &result)
			if err != nil {
				return nil, false, err
			}
			return result.Data, !result.Links.IsLastPage(), nil
		}, limit)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(shops)
		}

		headers := []string{"ID", "NAME", "URL", "CREATED"}
		var rows [][]string
		for _, s := range shops {
			rows = append(rows, []string{s.ID, s.Name, s.URL, s.CreatedAt})
		}

		output.Table(headers, rows)
		return nil
	},
}

// get
var getCmd = &cobra.Command{
	Use:   "get <shop_id>",
	Short: "Get shop details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		httpClient, apiKey, _, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		path := fmt.Sprintf("/ecommerce/shops/%s", args[0])
		var result ecommerce.RootShop
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodGet, path, nil, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result.Data)
		}

		s := result.Data
		headers := []string{"FIELD", "VALUE"}
		rows := [][]string{
			{"ID", s.ID},
			{"Name", s.Name},
			{"URL", s.URL},
			{"Created", s.CreatedAt},
			{"Updated", s.UpdatedAt},
		}

		output.Table(headers, rows)
		return nil
	},
}

// create
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new shop",
	RunE: func(cmd *cobra.Command, args []string) error {
		httpClient, apiKey, _, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		name, err = prompt.RequireArg(name, "name", "Shop name")
		if err != nil {
			return err
		}

		shopURL, _ := cmd.Flags().GetString("url")
		shopURL, err = prompt.RequireArg(shopURL, "url", "Shop URL")
		if err != nil {
			return err
		}

		body := map[string]string{"name": name, "url": shopURL}

		ctx := context.Background()
		var result ecommerce.RootShop
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodPost, "/ecommerce/shops", body, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result.Data)
		}

		output.Success(fmt.Sprintf("Shop created: %s (ID: %s)", result.Data.Name, result.Data.ID))
		return nil
	},
}

// update
var updateCmd = &cobra.Command{
	Use:   "update <shop_id>",
	Short: "Update a shop",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		httpClient, apiKey, _, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		body := make(map[string]string)
		if cmd.Flags().Changed("name") {
			v, _ := cmd.Flags().GetString("name")
			body["name"] = v
		}
		if cmd.Flags().Changed("url") {
			v, _ := cmd.Flags().GetString("url")
			body["url"] = v
		}
		if len(body) == 0 {
			return fmt.Errorf("no flags provided; use --help to see available options")
		}

		ctx := context.Background()
		path := fmt.Sprintf("/ecommerce/shops/%s", args[0])
		var result ecommerce.RootShop
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodPut, path, body, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result.Data)
		}

		output.Success(fmt.Sprintf("Shop updated: %s (ID: %s)", result.Data.Name, result.Data.ID))
		return nil
	},
}

// delete
var deleteCmd = &cobra.Command{
	Use:   "delete <shop_id>",
	Short: "Delete a shop",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		httpClient, apiKey, _, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		path := fmt.Sprintf("/ecommerce/shops/%s", args[0])
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodDelete, path, nil, nil)
		if err != nil {
			return err
		}

		output.Success(fmt.Sprintf("Shop %s deleted.", args[0]))
		return nil
	},
}

// count
var countCmd = &cobra.Command{
	Use:   "count",
	Short: "Get total shop count",
	RunE: func(cmd *cobra.Command, args []string) error {
		httpClient, apiKey, _, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		var result ecommerce.RootCount
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodGet, "/ecommerce/shops?limit=0", nil, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result)
		}

		fmt.Printf("Total shops: %d\n", result.Total)
		return nil
	},
}
