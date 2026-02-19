package category

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mailerlite/mailerlite-cli/internal/cmdutil"
	"github.com/mailerlite/mailerlite-cli/internal/ecommerce"
	"github.com/mailerlite/mailerlite-cli/internal/output"
	"github.com/mailerlite/mailerlite-cli/internal/prompt"
	"github.com/mailerlite/mailerlite-cli/internal/sdkclient"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "category",
	Short: "Manage e-commerce categories",
	Long:  "List, create, update, and delete categories within a shop.",
}

func init() {
	Cmd.PersistentFlags().String("shop", "", "shop ID (required)")

	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(countCmd)
	Cmd.AddCommand(productsCmd)
	Cmd.AddCommand(assignProductCmd)
	Cmd.AddCommand(unassignProductCmd)

	// list flags
	listCmd.Flags().Int("limit", 25, "maximum number of categories to return (0 = all)")

	// create flags
	createCmd.Flags().String("name", "", "category name (required)")

	// update flags
	updateCmd.Flags().String("name", "", "category name")

	// assign-product flags
	assignProductCmd.Flags().String("product", "", "product ID (required)")

	// unassign-product flags
	unassignProductCmd.Flags().String("product", "", "product ID (required)")
}

func shopFlag(cmd *cobra.Command) (string, error) {
	shop, _ := cmd.Flags().GetString("shop")
	return prompt.RequireArg(shop, "shop", "Shop ID")
}

// list
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List categories",
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		ctx := context.Background()

		categories, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]ecommerce.Category, bool, error) {
			path := fmt.Sprintf("/ecommerce/shops/%s/categories?page=%d&limit=%d", shopID, page, perPage)
			var result ecommerce.RootCategories
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
			return output.JSON(categories)
		}

		headers := []string{"ID", "NAME", "CREATED"}
		var rows [][]string
		for _, c := range categories {
			rows = append(rows, []string{c.ID, c.Name, c.CreatedAt})
		}

		output.Table(headers, rows)
		return nil
	},
}

// get
var getCmd = &cobra.Command{
	Use:   "get <category_id>",
	Short: "Get category details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		path := fmt.Sprintf("/ecommerce/shops/%s/categories/%s", shopID, args[0])
		var result ecommerce.RootCategory
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodGet, path, nil, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result.Data)
		}

		c := result.Data
		headers := []string{"FIELD", "VALUE"}
		rows := [][]string{
			{"ID", c.ID},
			{"Name", c.Name},
			{"Created", c.CreatedAt},
			{"Updated", c.UpdatedAt},
		}

		output.Table(headers, rows)
		return nil
	},
}

// create
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new category",
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		name, err = prompt.RequireArg(name, "name", "Category name")
		if err != nil {
			return err
		}

		body := map[string]string{"name": name}

		ctx := context.Background()
		path := fmt.Sprintf("/ecommerce/shops/%s/categories", shopID)
		var result ecommerce.RootCategory
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodPost, path, body, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result.Data)
		}

		output.Success(fmt.Sprintf("Category created: %s (ID: %s)", result.Data.Name, result.Data.ID))
		return nil
	},
}

// update
var updateCmd = &cobra.Command{
	Use:   "update <category_id>",
	Short: "Update a category",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		body := make(map[string]string)
		if cmd.Flags().Changed("name") {
			v, _ := cmd.Flags().GetString("name")
			body["name"] = v
		}
		if len(body) == 0 {
			return fmt.Errorf("no flags provided; use --help to see available options")
		}

		ctx := context.Background()
		path := fmt.Sprintf("/ecommerce/shops/%s/categories/%s", shopID, args[0])
		var result ecommerce.RootCategory
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodPut, path, body, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result.Data)
		}

		output.Success(fmt.Sprintf("Category updated: %s (ID: %s)", result.Data.Name, result.Data.ID))
		return nil
	},
}

// delete
var deleteCmd = &cobra.Command{
	Use:   "delete <category_id>",
	Short: "Delete a category",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		path := fmt.Sprintf("/ecommerce/shops/%s/categories/%s", shopID, args[0])
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodDelete, path, nil, nil)
		if err != nil {
			return err
		}

		output.Success(fmt.Sprintf("Category %s deleted.", args[0]))
		return nil
	},
}

// count
var countCmd = &cobra.Command{
	Use:   "count",
	Short: "Get total category count",
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		path := fmt.Sprintf("/ecommerce/shops/%s/categories?limit=0", shopID)
		var result ecommerce.RootCount
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodGet, path, nil, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result)
		}

		fmt.Printf("Total categories: %d\n", result.Total)
		return nil
	},
}

// products - list products in a category
var productsCmd = &cobra.Command{
	Use:   "products <category_id>",
	Short: "List products in a category",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		path := fmt.Sprintf("/ecommerce/shops/%s/categories/%s/products", shopID, args[0])
		var result ecommerce.RootProducts
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodGet, path, nil, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result.Data)
		}

		headers := []string{"ID", "NAME", "PRICE", "CREATED"}
		var rows [][]string
		for _, p := range result.Data {
			rows = append(rows, []string{
				p.ID,
				p.Name,
				fmt.Sprintf("%.2f", p.Price),
				p.CreatedAt,
			})
		}

		output.Table(headers, rows)
		return nil
	},
}

// assign-product
var assignProductCmd = &cobra.Command{
	Use:   "assign-product <category_id>",
	Short: "Assign a product to a category",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		productID, _ := cmd.Flags().GetString("product")
		productID, err = prompt.RequireArg(productID, "product", "Product ID")
		if err != nil {
			return err
		}

		body := map[string]string{"product_id": productID}

		ctx := context.Background()
		path := fmt.Sprintf("/ecommerce/shops/%s/categories/%s/products", shopID, args[0])
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodPost, path, body, nil)
		if err != nil {
			return err
		}

		output.Success(fmt.Sprintf("Product %s assigned to category %s.", productID, args[0]))
		return nil
	},
}

// unassign-product
var unassignProductCmd = &cobra.Command{
	Use:   "unassign-product <category_id>",
	Short: "Remove a product from a category",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		productID, _ := cmd.Flags().GetString("product")
		productID, err = prompt.RequireArg(productID, "product", "Product ID")
		if err != nil {
			return err
		}

		ctx := context.Background()
		path := fmt.Sprintf("/ecommerce/shops/%s/categories/%s/products/%s", shopID, args[0], productID)
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodDelete, path, nil, nil)
		if err != nil {
			return err
		}

		output.Success(fmt.Sprintf("Product %s removed from category %s.", productID, args[0]))
		return nil
	},
}
