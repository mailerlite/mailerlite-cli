package order

import (
	"context"
	"encoding/json"
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
	Use:   "order",
	Short: "Manage e-commerce orders",
	Long:  "List, create, update, and delete orders within a shop.",
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
	listCmd.Flags().Int("limit", 25, "maximum number of orders to return (0 = all)")

	// create flags
	createCmd.Flags().String("customer", "", "customer ID (required)")
	createCmd.Flags().String("status", "", "order status (required)")
	createCmd.Flags().Float64("total", 0, "order total (required)")
	createCmd.Flags().String("currency", "USD", "order currency")
	createCmd.Flags().String("items", "", "order items as JSON array")

	// update flags
	updateCmd.Flags().String("customer", "", "customer ID")
	updateCmd.Flags().String("status", "", "order status")
	updateCmd.Flags().Float64("total", 0, "order total")
	updateCmd.Flags().String("currency", "", "order currency")
	updateCmd.Flags().String("items", "", "order items as JSON array")
}

func shopFlag(cmd *cobra.Command) (string, error) {
	shop, _ := cmd.Flags().GetString("shop")
	return prompt.RequireArg(shop, "shop", "Shop ID")
}

// list
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List orders",
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

		orders, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]ecommerce.Order, bool, error) {
			path := fmt.Sprintf("/ecommerce/shops/%s/orders?page=%d&limit=%d", shopID, page, perPage)
			var result ecommerce.RootOrders
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
			return output.JSON(orders)
		}

		headers := []string{"ID", "CUSTOMER", "STATUS", "TOTAL", "CURRENCY", "CREATED"}
		var rows [][]string
		for _, o := range orders {
			rows = append(rows, []string{
				o.ID,
				o.CustomerID,
				o.Status,
				strconv.FormatFloat(o.Total, 'f', 2, 64),
				o.Currency,
				o.CreatedAt,
			})
		}

		output.Table(headers, rows)
		return nil
	},
}

// get
var getCmd = &cobra.Command{
	Use:   "get <order_id>",
	Short: "Get order details",
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
		path := fmt.Sprintf("/ecommerce/shops/%s/orders/%s", shopID, args[0])
		var result ecommerce.RootOrder
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodGet, path, nil, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result.Data)
		}

		o := result.Data
		headers := []string{"FIELD", "VALUE"}
		rows := [][]string{
			{"ID", o.ID},
			{"Customer ID", o.CustomerID},
			{"Status", o.Status},
			{"Total", strconv.FormatFloat(o.Total, 'f', 2, 64)},
			{"Currency", o.Currency},
			{"Items", strconv.Itoa(len(o.Items))},
			{"Created", o.CreatedAt},
			{"Updated", o.UpdatedAt},
		}

		output.Table(headers, rows)
		return nil
	},
}

// create
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new order",
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		customerID, _ := cmd.Flags().GetString("customer")
		customerID, err = prompt.RequireArg(customerID, "customer", "Customer ID")
		if err != nil {
			return err
		}

		status, _ := cmd.Flags().GetString("status")
		status, err = prompt.RequireArg(status, "status", "Order status")
		if err != nil {
			return err
		}

		total, _ := cmd.Flags().GetFloat64("total")
		currency, _ := cmd.Flags().GetString("currency")

		body := map[string]interface{}{
			"customer_id": customerID,
			"status":      status,
			"total":       total,
			"currency":    currency,
		}

		if itemsStr, _ := cmd.Flags().GetString("items"); itemsStr != "" {
			var items []ecommerce.OrderItem
			if err := json.Unmarshal([]byte(itemsStr), &items); err != nil {
				return fmt.Errorf("invalid --items JSON: %w", err)
			}
			body["items"] = items
		}

		ctx := context.Background()
		path := fmt.Sprintf("/ecommerce/shops/%s/orders", shopID)
		var result ecommerce.RootOrder
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodPost, path, body, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result.Data)
		}

		output.Success(fmt.Sprintf("Order created: %s (ID: %s)", result.Data.Status, result.Data.ID))
		return nil
	},
}

// update
var updateCmd = &cobra.Command{
	Use:   "update <order_id>",
	Short: "Update an order",
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

		body := make(map[string]interface{})
		if cmd.Flags().Changed("customer") {
			v, _ := cmd.Flags().GetString("customer")
			body["customer_id"] = v
		}
		if cmd.Flags().Changed("status") {
			v, _ := cmd.Flags().GetString("status")
			body["status"] = v
		}
		if cmd.Flags().Changed("total") {
			v, _ := cmd.Flags().GetFloat64("total")
			body["total"] = v
		}
		if cmd.Flags().Changed("currency") {
			v, _ := cmd.Flags().GetString("currency")
			body["currency"] = v
		}
		if cmd.Flags().Changed("items") {
			itemsStr, _ := cmd.Flags().GetString("items")
			var items []ecommerce.OrderItem
			if err := json.Unmarshal([]byte(itemsStr), &items); err != nil {
				return fmt.Errorf("invalid --items JSON: %w", err)
			}
			body["items"] = items
		}
		if len(body) == 0 {
			return fmt.Errorf("no flags provided; use --help to see available options")
		}

		ctx := context.Background()
		path := fmt.Sprintf("/ecommerce/shops/%s/orders/%s", shopID, args[0])
		var result ecommerce.RootOrder
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodPut, path, body, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result.Data)
		}

		output.Success(fmt.Sprintf("Order updated: %s (ID: %s)", result.Data.Status, result.Data.ID))
		return nil
	},
}

// delete
var deleteCmd = &cobra.Command{
	Use:   "delete <order_id>",
	Short: "Delete an order",
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
		path := fmt.Sprintf("/ecommerce/shops/%s/orders/%s", shopID, args[0])
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodDelete, path, nil, nil)
		if err != nil {
			return err
		}

		output.Success(fmt.Sprintf("Order %s deleted.", args[0]))
		return nil
	},
}

// count
var countCmd = &cobra.Command{
	Use:   "count",
	Short: "Get total order count",
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
		path := fmt.Sprintf("/ecommerce/shops/%s/orders?limit=0", shopID)
		var result ecommerce.RootCount
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodGet, path, nil, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result)
		}

		fmt.Printf("Total orders: %d\n", result.Total)
		return nil
	},
}
