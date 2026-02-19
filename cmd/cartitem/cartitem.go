package cartitem

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
	Use:   "cart-item",
	Short: "Manage cart items",
	Long:  "List, create, update, and delete items within a cart.",
}

func init() {
	Cmd.PersistentFlags().String("shop", "", "shop ID (required)")
	Cmd.PersistentFlags().String("cart", "", "cart ID (required)")

	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(countCmd)

	// list flags
	listCmd.Flags().Int("limit", 25, "maximum number of items to return (0 = all)")

	// create flags
	createCmd.Flags().String("product", "", "product ID (required)")
	createCmd.Flags().Int("quantity", 1, "item quantity")
	createCmd.Flags().Float64("price", 0, "item price")

	// update flags
	updateCmd.Flags().Int("quantity", 0, "item quantity")
	updateCmd.Flags().Float64("price", 0, "item price")
}

func shopFlag(cmd *cobra.Command) (string, error) {
	shop, _ := cmd.Flags().GetString("shop")
	return prompt.RequireArg(shop, "shop", "Shop ID")
}

func cartFlag(cmd *cobra.Command) (string, error) {
	cart, _ := cmd.Flags().GetString("cart")
	return prompt.RequireArg(cart, "cart", "Cart ID")
}

func basePath(shopID, cartID string) string {
	return fmt.Sprintf("/ecommerce/shops/%s/carts/%s/items", shopID, cartID)
}

// list
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List cart items",
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}
		cartID, err := cartFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		ctx := context.Background()

		items, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]ecommerce.CartItem, bool, error) {
			path := fmt.Sprintf("%s?page=%d&limit=%d", basePath(shopID, cartID), page, perPage)
			var result ecommerce.RootCartItems
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
			return output.JSON(items)
		}

		headers := []string{"ID", "PRODUCT", "QUANTITY", "PRICE", "CREATED"}
		var rows [][]string
		for _, i := range items {
			rows = append(rows, []string{
				i.ID,
				i.ProductID,
				strconv.Itoa(i.Quantity),
				strconv.FormatFloat(i.Price, 'f', 2, 64),
				i.CreatedAt,
			})
		}

		output.Table(headers, rows)
		return nil
	},
}

// get
var getCmd = &cobra.Command{
	Use:   "get <item_id>",
	Short: "Get cart item details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}
		cartID, err := cartFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		path := fmt.Sprintf("%s/%s", basePath(shopID, cartID), args[0])
		var result ecommerce.RootCartItem
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodGet, path, nil, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result.Data)
		}

		i := result.Data
		headers := []string{"FIELD", "VALUE"}
		rows := [][]string{
			{"ID", i.ID},
			{"Product ID", i.ProductID},
			{"Quantity", strconv.Itoa(i.Quantity)},
			{"Price", strconv.FormatFloat(i.Price, 'f', 2, 64)},
			{"Created", i.CreatedAt},
			{"Updated", i.UpdatedAt},
		}

		output.Table(headers, rows)
		return nil
	},
}

// create
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Add an item to a cart",
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}
		cartID, err := cartFlag(cmd)
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

		quantity, _ := cmd.Flags().GetInt("quantity")
		price, _ := cmd.Flags().GetFloat64("price")

		body := map[string]interface{}{
			"product_id": productID,
			"quantity":   quantity,
			"price":      price,
		}

		ctx := context.Background()
		var result ecommerce.RootCartItem
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodPost, basePath(shopID, cartID), body, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result.Data)
		}

		output.Success(fmt.Sprintf("Cart item created: %s (ID: %s)", result.Data.ProductID, result.Data.ID))
		return nil
	},
}

// update
var updateCmd = &cobra.Command{
	Use:   "update <item_id>",
	Short: "Update a cart item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}
		cartID, err := cartFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		body := make(map[string]interface{})
		if cmd.Flags().Changed("quantity") {
			v, _ := cmd.Flags().GetInt("quantity")
			body["quantity"] = v
		}
		if cmd.Flags().Changed("price") {
			v, _ := cmd.Flags().GetFloat64("price")
			body["price"] = v
		}
		if len(body) == 0 {
			return fmt.Errorf("no flags provided; use --help to see available options")
		}

		ctx := context.Background()
		path := fmt.Sprintf("%s/%s", basePath(shopID, cartID), args[0])
		var result ecommerce.RootCartItem
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodPut, path, body, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result.Data)
		}

		output.Success(fmt.Sprintf("Cart item updated: %s", result.Data.ID))
		return nil
	},
}

// delete
var deleteCmd = &cobra.Command{
	Use:   "delete <item_id>",
	Short: "Delete a cart item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}
		cartID, err := cartFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		path := fmt.Sprintf("%s/%s", basePath(shopID, cartID), args[0])
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodDelete, path, nil, nil)
		if err != nil {
			return err
		}

		output.Success(fmt.Sprintf("Cart item %s deleted.", args[0]))
		return nil
	},
}

// count
var countCmd = &cobra.Command{
	Use:   "count",
	Short: "Get total cart item count",
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}
		cartID, err := cartFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		path := fmt.Sprintf("%s?limit=0", basePath(shopID, cartID))
		var result ecommerce.RootCount
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodGet, path, nil, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result)
		}

		fmt.Printf("Total cart items: %d\n", result.Total)
		return nil
	},
}
