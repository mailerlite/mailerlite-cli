package segment

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mailerlite/mailerlite-cli/internal/cmdutil"
	"github.com/mailerlite/mailerlite-cli/internal/output"
	"github.com/mailerlite/mailerlite-cli/internal/prompt"
	"github.com/mailerlite/mailerlite-cli/internal/sdkclient"
	"github.com/mailerlite/mailerlite-go"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "segment",
	Short: "Manage segments",
	Long:  "List, update, delete segments and view segment subscribers.",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(deleteCmd)
	Cmd.AddCommand(subscribersCmd)

	// list flags
	listCmd.Flags().Int("limit", 25, "maximum number of segments to return (0 = all)")

	// update flags
	updateCmd.Flags().String("name", "", "new segment name (required)")

	// subscribers flags
	subscribersCmd.Flags().Int("limit", 25, "maximum number of subscribers to return (0 = all)")
}

// --- list ---

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List segments",
	RunE:  runList,
}

func runList(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	limit, _ := c.Flags().GetInt("limit")

	ctx := context.Background()

	allSegments, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]mailerlite.Segment, bool, error) {
		opts := &mailerlite.ListSegmentOptions{
			Page:  page,
			Limit: perPage,
		}
		root, _, err := ml.Segment.List(ctx, opts)
		if err != nil {
			return nil, false, sdkclient.WrapError(transport, err)
		}
		return root.Data, !root.Links.IsLastPage(), nil
	}, limit)
	if err != nil {
		return err
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(allSegments)
	}

	headers := []string{"ID", "NAME", "TOTAL", "OPEN RATE", "CLICK RATE", "CREATED AT"}
	var rows [][]string

	for _, s := range allSegments {
		rows = append(rows, []string{
			s.ID,
			s.Name,
			strconv.Itoa(s.Total),
			s.OpenRate.String,
			s.ClickRate.String,
			s.CreatedAt,
		})
	}

	output.Table(headers, rows)
	return nil
}

// --- update ---

var updateCmd = &cobra.Command{
	Use:   "update <segment_id>",
	Short: "Update a segment",
	Args:  cobra.ExactArgs(1),
	RunE:  runUpdate,
}

func runUpdate(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	name, _ := c.Flags().GetString("name")
	name, err = prompt.RequireArg(name, "name", "New segment name")
	if err != nil {
		return err
	}

	ctx := context.Background()
	result, _, err := ml.Segment.Update(ctx, args[0], name)
	if err != nil {
		return sdkclient.WrapError(transport, err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	output.Success("Segment " + args[0] + " updated successfully.")
	return nil
}

// --- delete ---

var deleteCmd = &cobra.Command{
	Use:   "delete <segment_id>",
	Short: "Delete a segment",
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

func runDelete(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	if !cmdutil.YesFlag(c) && prompt.IsInteractive() {
		ok, err := prompt.Confirm(fmt.Sprintf("Delete segment %s?", args[0]))
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	ctx := context.Background()
	_, err = ml.Segment.Delete(ctx, args[0])
	if err != nil {
		return sdkclient.WrapError(transport, err)
	}

	output.Success("Segment " + args[0] + " deleted successfully.")
	return nil
}

// --- subscribers ---

var subscribersCmd = &cobra.Command{
	Use:   "subscribers <segment_id>",
	Short: "List subscribers in a segment",
	Args:  cobra.ExactArgs(1),
	RunE:  runSubscribers,
}

func runSubscribers(c *cobra.Command, args []string) error {
	ml, transport, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	limit, _ := c.Flags().GetInt("limit")
	segmentID := args[0]

	ctx := context.Background()

	allSubscribers, err := sdkclient.FetchAllCursor(ctx, func(ctx context.Context, after, perPage int) ([]mailerlite.Subscriber, int, error) {
		opts := &mailerlite.ListSegmentSubscriberOptions{
			SegmentID: segmentID,
			Limit:     perPage,
			After:     after,
		}
		root, _, err := ml.Segment.Subscribers(ctx, opts)
		if err != nil {
			return nil, 0, sdkclient.WrapError(transport, err)
		}
		nextAfter := 0
		if root.Meta.Last > 0 {
			nextAfter = root.Meta.Last
		}
		return root.Data, nextAfter, nil
	}, limit)
	if err != nil {
		return err
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(allSubscribers)
	}

	headers := []string{"ID", "EMAIL", "STATUS", "SUBSCRIBED AT", "CREATED AT"}
	var rows [][]string

	for _, s := range allSubscribers {
		rows = append(rows, []string{
			s.ID,
			s.Email,
			s.Status,
			s.SubscribedAt,
			s.CreatedAt,
		})
	}

	output.Table(headers, rows)
	return nil
}
