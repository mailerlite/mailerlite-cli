package timezone

import (
	"context"
	"strconv"

	"github.com/mailerlite/mailerlite-cli/internal/cmdutil"
	"github.com/mailerlite/mailerlite-cli/internal/output"
	"github.com/mailerlite/mailerlite-cli/internal/sdkclient"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "timezone",
	Short: "Manage timezones",
	Long:  "List available timezones.",
}

func init() {
	Cmd.AddCommand(listCmd)
}

// --- list ---

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List timezones",
	RunE:  runList,
}

func runList(c *cobra.Command, args []string) error {
	ml, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	ctx := context.Background()
	result, _, err := ml.Timezone.List(ctx)
	if err != nil {
		return sdkclient.WrapError(err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result.Data)
	}

	headers := []string{"ID", "NAME", "OFFSET"}
	var rows [][]string

	for _, tz := range result.Data {
		rows = append(rows, []string{
			tz.Id,
			tz.Name,
			strconv.Itoa(tz.Offset),
		})
	}

	output.Table(headers, rows)
	return nil
}
