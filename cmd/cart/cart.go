package cart

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
	Use:   "cart",
	Short: "Manage e-commerce carts",
	Long:  "List, view, and update carts within a shop.",
}

func init() {
	Cmd.PersistentFlags().String("shop", "", "shop ID (required)")

	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(countCmd)

	// list flags
	listCmd.Flags().Int("limit", 25, "maximum number of carts to return (0 = all)")

	// update flags
	updateCmd.Flags().String("customer", "", "customer ID")
	updateCmd.Flags().String("currency", "", "cart currency")
}

func shopFlag(cmd *cobra.Command) (string, error) {
	shop, _ := cmd.Flags().GetString("shop")
	return prompt.RequireArg(shop, "shop", "Shop ID")
}

// list
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List carts",
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

		carts, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]ecommerce.Cart, bool, error) {
			path := fmt.Sprintf("/shops/%s/carts?page=%d&limit=%d", shopID, page, perPage)
			var result ecommerce.RootCarts
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
			return output.JSON(carts)
		}

		headers := []string{"ID", "CUSTOMER", "CURRENCY", "TOTAL", "CREATED"}
		var rows [][]string
		for _, c := range carts {
			rows = append(rows, []string{
				c.ID,
				c.CustomerID,
				c.Currency,
				strconv.FormatFloat(c.Total, 'f', 2, 64),
				c.CreatedAt,
			})
		}

		output.Table(headers, rows)
		return nil
	},
}

// get
var getCmd = &cobra.Command{
	Use:   "get <cart_id>",
	Short: "Get cart details",
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
		path := fmt.Sprintf("/shops/%s/carts/%s", shopID, args[0])
		var result ecommerce.RootCart
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
			{"Customer ID", c.CustomerID},
			{"Currency", c.Currency},
			{"Total", strconv.FormatFloat(c.Total, 'f', 2, 64)},
			{"Created", c.CreatedAt},
			{"Updated", c.UpdatedAt},
		}

		output.Table(headers, rows)
		return nil
	},
}

// update
var updateCmd = &cobra.Command{
	Use:   "update <cart_id>",
	Short: "Update a cart",
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

		body := make(map[string]string)
		if cmd.Flags().Changed("customer") {
			v, _ := cmd.Flags().GetString("customer")
			body["customer_id"] = v
		}
		if cmd.Flags().Changed("currency") {
			v, _ := cmd.Flags().GetString("currency")
			body["currency"] = v
		}
		if len(body) == 0 {
			return fmt.Errorf("no flags provided; use --help to see available options")
		}

		ctx := context.Background()
		path := fmt.Sprintf("/shops/%s/carts/%s", shopID, args[0])
		var result ecommerce.RootCart
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodPut, path, body, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result.Data)
		}

		output.Success(fmt.Sprintf("Cart updated: %s", result.Data.ID))
		return nil
	},
}

// count
var countCmd = &cobra.Command{
	Use:   "count",
	Short: "Get total cart count",
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
		path := fmt.Sprintf("/shops/%s/carts?limit=0", shopID)
		var result ecommerce.RootCount
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodGet, path, nil, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result)
		}

		fmt.Printf("Total carts: %d\n", result.Total)
		return nil
	},
}
