package customer

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
	Use:   "customer",
	Short: "Manage e-commerce customers",
	Long:  "List, create, update, and delete customers within a shop.",
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
	listCmd.Flags().Int("limit", 25, "maximum number of customers to return (0 = all)")

	// create flags
	createCmd.Flags().String("email", "", "customer email (required)")
	createCmd.Flags().String("first-name", "", "customer first name")
	createCmd.Flags().String("last-name", "", "customer last name")

	// update flags
	updateCmd.Flags().String("email", "", "customer email")
	updateCmd.Flags().String("first-name", "", "customer first name")
	updateCmd.Flags().String("last-name", "", "customer last name")
}

func shopFlag(cmd *cobra.Command) (string, error) {
	shop, _ := cmd.Flags().GetString("shop")
	return prompt.RequireArg(shop, "shop", "Shop ID")
}

// list
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List customers",
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

		customers, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]ecommerce.Customer, bool, error) {
			path := fmt.Sprintf("/shops/%s/customers?page=%d&limit=%d", shopID, page, perPage)
			var result ecommerce.RootCustomers
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
			return output.JSON(customers)
		}

		headers := []string{"ID", "EMAIL", "FIRST NAME", "LAST NAME", "CREATED"}
		var rows [][]string
		for _, c := range customers {
			rows = append(rows, []string{c.ID, c.Email, c.FirstName, c.LastName, c.CreatedAt})
		}

		output.Table(headers, rows)
		return nil
	},
}

// get
var getCmd = &cobra.Command{
	Use:   "get <customer_id>",
	Short: "Get customer details",
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
		path := fmt.Sprintf("/shops/%s/customers/%s", shopID, args[0])
		var result ecommerce.RootCustomer
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
			{"Email", c.Email},
			{"First Name", c.FirstName},
			{"Last Name", c.LastName},
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
	Short: "Create a new customer",
	RunE: func(cmd *cobra.Command, args []string) error {
		shopID, err := shopFlag(cmd)
		if err != nil {
			return err
		}

		httpClient, apiKey, _, err := cmdutil.RawHTTPClient(cmd)
		if err != nil {
			return err
		}

		email, _ := cmd.Flags().GetString("email")
		email, err = prompt.RequireArg(email, "email", "Customer email")
		if err != nil {
			return err
		}

		body := map[string]string{"email": email}
		if v, _ := cmd.Flags().GetString("first-name"); v != "" {
			body["first_name"] = v
		}
		if v, _ := cmd.Flags().GetString("last-name"); v != "" {
			body["last_name"] = v
		}

		ctx := context.Background()
		path := fmt.Sprintf("/shops/%s/customers", shopID)
		var result ecommerce.RootCustomer
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodPost, path, body, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result.Data)
		}

		output.Success(fmt.Sprintf("Customer created: %s (ID: %s)", result.Data.Email, result.Data.ID))
		return nil
	},
}

// update
var updateCmd = &cobra.Command{
	Use:   "update <customer_id>",
	Short: "Update a customer",
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
		if cmd.Flags().Changed("email") {
			v, _ := cmd.Flags().GetString("email")
			body["email"] = v
		}
		if cmd.Flags().Changed("first-name") {
			v, _ := cmd.Flags().GetString("first-name")
			body["first_name"] = v
		}
		if cmd.Flags().Changed("last-name") {
			v, _ := cmd.Flags().GetString("last-name")
			body["last_name"] = v
		}
		if len(body) == 0 {
			return fmt.Errorf("no flags provided; use --help to see available options")
		}

		ctx := context.Background()
		path := fmt.Sprintf("/shops/%s/customers/%s", shopID, args[0])
		var result ecommerce.RootCustomer
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodPut, path, body, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result.Data)
		}

		output.Success(fmt.Sprintf("Customer updated: %s (ID: %s)", result.Data.Email, result.Data.ID))
		return nil
	},
}

// delete
var deleteCmd = &cobra.Command{
	Use:   "delete <customer_id>",
	Short: "Delete a customer",
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
		path := fmt.Sprintf("/shops/%s/customers/%s", shopID, args[0])
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodDelete, path, nil, nil)
		if err != nil {
			return err
		}

		output.Success(fmt.Sprintf("Customer %s deleted.", args[0]))
		return nil
	},
}

// count
var countCmd = &cobra.Command{
	Use:   "count",
	Short: "Get total customer count",
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
		path := fmt.Sprintf("/shops/%s/customers?limit=0", shopID)
		var result ecommerce.RootCount
		_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodGet, path, nil, &result)
		if err != nil {
			return err
		}

		if cmdutil.JSONFlag(cmd) {
			return output.JSON(result)
		}

		fmt.Printf("Total customers: %d\n", result.Total)
		return nil
	},
}
