package importcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/mailerlite/mailerlite-cli/internal/cmdutil"
	"github.com/mailerlite/mailerlite-cli/internal/output"
	"github.com/mailerlite/mailerlite-cli/internal/prompt"
	"github.com/mailerlite/mailerlite-cli/internal/sdkclient"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "import",
	Short: "Bulk import e-commerce data",
	Long:  "Import categories, products, or orders in bulk from a JSON file.",
}

func init() {
	Cmd.AddCommand(categoriesCmd)
	Cmd.AddCommand(productsCmd)
	Cmd.AddCommand(ordersCmd)

	// categories flags
	categoriesCmd.Flags().String("shop", "", "shop ID (required)")
	categoriesCmd.Flags().String("file", "", "path to JSON file (required)")

	// products flags
	productsCmd.Flags().String("shop", "", "shop ID (required)")
	productsCmd.Flags().String("file", "", "path to JSON file (required)")

	// orders flags
	ordersCmd.Flags().String("shop", "", "shop ID (required)")
	ordersCmd.Flags().String("file", "", "path to JSON file (required)")
}

func readJSONFile(path string) (json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	if !json.Valid(data) {
		return nil, fmt.Errorf("file %s does not contain valid JSON", path)
	}

	return json.RawMessage(data), nil
}

func doImport(cmd *cobra.Command, resource string) error {
	shopID, _ := cmd.Flags().GetString("shop")
	var err error
	shopID, err = prompt.RequireArg(shopID, "shop", "Shop ID")
	if err != nil {
		return err
	}

	filePath, _ := cmd.Flags().GetString("file")
	filePath, err = prompt.RequireArg(filePath, "file", "Path to JSON file")
	if err != nil {
		return err
	}

	data, err := readJSONFile(filePath)
	if err != nil {
		return err
	}

	httpClient, apiKey, _, err := cmdutil.RawHTTPClient(cmd)
	if err != nil {
		return err
	}

	ctx := context.Background()
	path := fmt.Sprintf("/ecommerce/shops/%s/%s/import", shopID, resource)

	var result json.RawMessage
	_, err = sdkclient.DoRaw(ctx, httpClient, apiKey, http.MethodPost, path, data, &result)
	if err != nil {
		return err
	}

	if cmdutil.JSONFlag(cmd) {
		return output.JSON(result)
	}

	output.Success(fmt.Sprintf("Import of %s completed successfully.", resource))
	return nil
}

// categories
var categoriesCmd = &cobra.Command{
	Use:   "categories",
	Short: "Bulk import categories",
	RunE: func(cmd *cobra.Command, args []string) error {
		return doImport(cmd, "categories")
	},
}

// products
var productsCmd = &cobra.Command{
	Use:   "products",
	Short: "Bulk import products",
	RunE: func(cmd *cobra.Command, args []string) error {
		return doImport(cmd, "products")
	},
}

// orders
var ordersCmd = &cobra.Command{
	Use:   "orders",
	Short: "Bulk import orders",
	RunE: func(cmd *cobra.Command, args []string) error {
		return doImport(cmd, "orders")
	},
}
