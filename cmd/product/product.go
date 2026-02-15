package product

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
	Use:   "product",
	Short: "Manage e-commerce products",
	Long:  "List, create, update, and delete products within a shop.",
}

func init() {
	Cmd.PersistentFlags().String("shop", "", "shop ID (required)")

	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(countCmd)

	// list flags
	listCmd.Flags().Int("limit", 25, "maximum number of products to return (0 = all)")

	// create flags
	createCmd.Flags().String("name", "", "product name (required)")
	createCmd.Flags().Float64("price", 0, "product price (required)")
	createCmd.Flags().String("url", "", "product URL")
	createCmd.Flags().String("image-url", "", "product image URL")
	createCmd.Flags().String("description", "", "product description")
	createCmd.Flags().Int("quantity", 0, "product quantity")

	// update flags
	updateCmd.Flags().String("name", "", "product name")
	updateCmd.Flags().Float64("price", 0, "product price")
	updateCmd.Flags().String("url", "", "product URL")
	updateCmd.Flags().String("image-url", "", "product image URL")
	updateCmd.Flags().String("description", "", "product description")
	updateCmd.Flags().Int("quantity", 0, "product quantity")
}

func shopFlag(cmd *cobra.Command) (string, error) {
	shop, _ := cmd.Flags().GetString("shop")
	return prompt.RequireArg(shop, "shop", "Shop ID")
}

// list
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List products",
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, _, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		ctx := context.Background()

		products, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]ecommerce.Product, bool, error) {
			path := fmt.Sprintf("/shops/%s/products?page=%d&limit=%d", shopID, page, perPage)
			var result ecommerce.RootProducts
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
			return output.JSON(products)
		}

		headers := []string{"ID", "NAME", "PRICE", "QUANTITY", "CREATED"}
		var rows [][]string
		for _, p := range products {
			rows = append(rows, []string{
				p.ID,
				p.Name,
				strconv.FormatFloat(p.Price, 'f', 2, 64),
				strconv.Itoa(p.Quantity),
				p.CreatedAt,
			})
		}

		output.Table(headers, rows)
		return nil
	},
}

// get
var getCmd = &cobra.Command{
	Use:   "get <product_id>",
	Short: "Get product details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, _, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		path := fmt.Sprintf("/shops/%s/products/%s", shopID, args[0])
		var result ecommerce.RootProduct
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodGet, path, nil, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result.Data)
		}

		p := result.Data
		headers := []string{"FIELD", "VALUE"}
		rows := [][]string{
			{"ID", p.ID},
			{"Name", p.Name},
			{"Price", strconv.FormatFloat(p.Price, 'f', 2, 64)},
			{"URL", p.URL},
			{"Image URL", p.ImageURL},
			{"Description", output.Truncate(p.Description, 60)},
			{"Quantity", strconv.Itoa(p.Quantity)},
			{"Created", p.CreatedAt},
			{"Updated", p.UpdatedAt},
		}

		output.Table(headers, rows)
		return nil
	},
}

// create
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new product",
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, _, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		name, err = prompt.RequireArg(name, "name", "Product name")
		if err != nil {
			return err
		}

		price, _ := cmd.Flags().GetFloat64("price")

		body := map[string]interface{}{"name": name, "price": price}
		if v, _ := cmd.Flags().GetString("url"); v != "" {
			body["url"] = v
		}
		if v, _ := cmd.Flags().GetString("image-url"); v != "" {
			body["image_url"] = v
		}
		if v, _ := cmd.Flags().GetString("description"); v != "" {
			body["description"] = v
		}
		if cmd.Flags().Changed("quantity") {
			v, _ := cmd.Flags().GetInt("quantity")
			body["quantity"] = v
		}

		ctx := context.Background()
		path := fmt.Sprintf("/shops/%s/products", shopID)
		var result ecommerce.RootProduct
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodPost, path, body, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result.Data)
		}

		output.Success(fmt.Sprintf("Product created: %s (ID: %s)", result.Data.Name, result.Data.ID))
		return nil
	},
}

// update
var updateCmd = &cobra.Command{
	Use:   "update <product_id>",
	Short: "Update a product",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, _, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		body := make(map[string]interface{})
		if cmd.Flags().Changed("name") {
			v, _ := cmd.Flags().GetString("name")
			body["name"] = v
		}
		if cmd.Flags().Changed("price") {
			v, _ := cmd.Flags().GetFloat64("price")
			body["price"] = v
		}
		if cmd.Flags().Changed("url") {
			v, _ := cmd.Flags().GetString("url")
			body["url"] = v
		}
		if cmd.Flags().Changed("image-url") {
			v, _ := cmd.Flags().GetString("image-url")
			body["image_url"] = v
		}
		if cmd.Flags().Changed("description") {
			v, _ := cmd.Flags().GetString("description")
			body["description"] = v
		}
		if cmd.Flags().Changed("quantity") {
			v, _ := cmd.Flags().GetInt("quantity")
			body["quantity"] = v
		}
		if len(body) == 0 {
			return fmt.Errorf("no flags provided; use --help to see available options")
		}

		ctx := context.Background()
		path := fmt.Sprintf("/shops/%s/products/%s", shopID, args[0])
		var result ecommerce.RootProduct
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodPut, path, body, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result.Data)
		}

		output.Success(fmt.Sprintf("Product updated: %s (ID: %s)", result.Data.Name, result.Data.ID))
		return nil
	},
}

// delete
var deleteCmd = &cobra.Command{
	Use:   "delete <product_id>",
	Short: "Delete a product",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, _, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		path := fmt.Sprintf("/shops/%s/products/%s", shopID, args[0])
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodDelete, path, nil, nil)
		if err != nil {
			return err
		}

		output.Success(fmt.Sprintf("Product %s deleted.", args[0]))
		return nil
	},
}

// count
var countCmd = &cobra.Command{
	Use:   "count",
	Short: "Get total product count",
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, _, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		path := fmt.Sprintf("/shops/%s/products?limit=0", shopID)
		var result ecommerce.RootCount
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodGet, path, nil, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result)
		}

		fmt.Printf("Total products: %d\n", result.Total)
		return nil
	},
}
