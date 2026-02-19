package campaign

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mailerlite/mailerlite-cli/internal/cmdutil"
	"github.com/mailerlite/mailerlite-cli/internal/output"
	"github.com/mailerlite/mailerlite-cli/internal/prompt"
	"github.com/mailerlite/mailerlite-cli/internal/sdkclient"
	"github.com/mailerlite/mailerlite-go"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "campaign",
	Short: "Manage campaigns",
	Long:  "List, view, create, update, schedule, cancel, and delete campaigns.",
}

func init() {
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(scheduleCmd)
	Cmd.AddCommand(cancelCmd)
	Cmd.AddCommand(subscribersCmd)
	Cmd.AddCommand(languagesCmd)
	Cmd.AddCommand(deleteCmd)

	// list flags
	listCmd.Flags().Int("limit", 25, "maximum number of campaigns to return (0 = all)")
	listCmd.Flags().String("status", "", "filter by status (sent, draft, ready)")
	listCmd.Flags().String("type", "", "filter by type (regular, ab, resend)")

	// create flags
	createCmd.Flags().String("name", "", "campaign name (required)")
	createCmd.Flags().String("type", "regular", "campaign type (regular, ab, resend)")
	createCmd.Flags().String("subject", "", "email subject (required)")
	createCmd.Flags().String("from", "", "sender email address (required)")
	createCmd.Flags().String("from-name", "", "sender name (required)")
	createCmd.Flags().String("content", "", "email HTML content")
	createCmd.Flags().StringSlice("groups", nil, "group IDs")
	createCmd.Flags().StringSlice("segments", nil, "segment IDs")

	// update flags
	updateCmd.Flags().String("name", "", "campaign name")
	updateCmd.Flags().String("type", "", "campaign type (regular, ab, resend)")
	updateCmd.Flags().String("subject", "", "email subject")
	updateCmd.Flags().String("from", "", "sender email address")
	updateCmd.Flags().String("from-name", "", "sender name")
	updateCmd.Flags().String("content", "", "email HTML content")
	updateCmd.Flags().StringSlice("groups", nil, "group IDs")
	updateCmd.Flags().StringSlice("segments", nil, "segment IDs")

	// schedule flags
	scheduleCmd.Flags().String("delivery", "instant", "delivery type (instant, scheduled)")
	scheduleCmd.Flags().String("date", "", "schedule date (YYYY-MM-DD)")
	scheduleCmd.Flags().String("hours", "", "schedule hours (00-23)")
	scheduleCmd.Flags().String("minutes", "", "schedule minutes (00-59)")
	scheduleCmd.Flags().Int("timezone-id", 0, "timezone ID")

	// subscribers flags
	subscribersCmd.Flags().Int("limit", 25, "maximum number of subscribers to return (0 = all)")
}

// --- list ---

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List campaigns",
	RunE:  runList,
}

func runList(c *cobra.Command, _ []string) error {
	ml, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	limit, _ := c.Flags().GetInt("limit")
	status, _ := c.Flags().GetString("status")
	campaignType, _ := c.Flags().GetString("type")

	ctx := context.Background()

	campaigns, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]mailerlite.Campaign, bool, error) {
		var filters []mailerlite.Filter
		if status != "" {
			filters = append(filters, mailerlite.Filter{Name: "status", Value: status})
		}
		if campaignType != "" {
			filters = append(filters, mailerlite.Filter{Name: "type", Value: campaignType})
		}

		opts := &mailerlite.ListCampaignOptions{
			Page:  page,
			Limit: perPage,
		}
		if len(filters) > 0 {
			opts.Filters = &filters
		}

		root, _, err := ml.Campaign.List(ctx, opts)
		if err != nil {
			return nil, false, sdkclient.WrapError(err)
		}

		return root.Data, !root.Links.IsLastPage(), nil
	}, limit)
	if err != nil {
		return err
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(campaigns)
	}

	headers := []string{"ID", "NAME", "TYPE", "STATUS", "SENT", "OPENS", "CLICKS"}
	var rows [][]string

	for _, camp := range campaigns {
		rows = append(rows, []string{
			camp.ID,
			output.Truncate(camp.Name, 40),
			camp.Type,
			camp.Status,
			strconv.Itoa(camp.Stats.Sent),
			strconv.Itoa(camp.Stats.OpensCount),
			strconv.Itoa(camp.Stats.ClicksCount),
		})
	}

	output.Table(headers, rows)
	return nil
}

// --- get ---

var getCmd = &cobra.Command{
	Use:   "get <campaign_id>",
	Short: "Get campaign details",
	Args:  cobra.ExactArgs(1),
	RunE:  runGet,
}

func runGet(c *cobra.Command, args []string) error {
	ml, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	ctx := context.Background()
	result, _, err := ml.Campaign.Get(ctx, args[0])
	if err != nil {
		return sdkclient.WrapError(err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	d := result.Data

	fmt.Printf("ID:           %s\n", d.ID)
	fmt.Printf("Name:         %s\n", d.Name)
	fmt.Printf("Type:         %s\n", d.Type)
	fmt.Printf("Status:       %s\n", d.Status)
	fmt.Printf("Created At:   %s\n", d.CreatedAt)
	fmt.Printf("Updated At:   %s\n", d.UpdatedAt)
	if d.ScheduledFor != "" {
		fmt.Printf("Scheduled:    %s\n", d.ScheduledFor)
	}

	fmt.Println()
	fmt.Println("Stats:")
	fmt.Printf("  Sent:       %d\n", d.Stats.Sent)
	fmt.Printf("  Opens:      %d\n", d.Stats.OpensCount)
	fmt.Printf("  Clicks:     %d\n", d.Stats.ClicksCount)

	if len(d.Emails) > 0 {
		fmt.Println()
		fmt.Println("Emails:")
		for _, e := range d.Emails {
			fmt.Printf("  - %s (from: %s <%s>)\n", e.Subject, e.FromName, e.From)
		}
	}

	return nil
}

// --- create ---

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a campaign",
	RunE:  runCreate,
}

func runCreate(c *cobra.Command, _ []string) error {
	ml, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	name, _ := c.Flags().GetString("name")
	name, err = prompt.RequireArg(name, "name", "Campaign name")
	if err != nil {
		return err
	}

	campaignType, _ := c.Flags().GetString("type")
	subject, _ := c.Flags().GetString("subject")
	subject, err = prompt.RequireArg(subject, "subject", "Email subject")
	if err != nil {
		return err
	}

	from, _ := c.Flags().GetString("from")
	from, err = prompt.RequireArg(from, "from", "Sender email address")
	if err != nil {
		return err
	}

	fromName, _ := c.Flags().GetString("from-name")
	fromName, err = prompt.RequireArg(fromName, "from-name", "Sender name")
	if err != nil {
		return err
	}

	content, _ := c.Flags().GetString("content")
	groups, _ := c.Flags().GetStringSlice("groups")
	segments, _ := c.Flags().GetStringSlice("segments")

	ctx := context.Background()
	opts := &mailerlite.CreateCampaign{
		Name: name,
		Type: campaignType,
		Emails: []mailerlite.Emails{
			{
				Subject:  subject,
				From:     from,
				FromName: fromName,
				Content:  content,
			},
		},
		Groups:   groups,
		Segments: segments,
	}

	result, _, err := ml.Campaign.Create(ctx, opts)
	if err != nil {
		return sdkclient.WrapError(err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	output.Success("Campaign created successfully. ID: " + result.Data.ID)
	return nil
}

// --- update ---

var updateCmd = &cobra.Command{
	Use:   "update <campaign_id>",
	Short: "Update a campaign",
	Args:  cobra.ExactArgs(1),
	RunE:  runUpdate,
}

func runUpdate(c *cobra.Command, args []string) error {
	ml, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	// First get the existing campaign to preserve unchanged fields.
	ctx := context.Background()
	existing, _, err := ml.Campaign.Get(ctx, args[0])
	if err != nil {
		return sdkclient.WrapError(err)
	}

	name := existing.Data.Name
	if c.Flags().Changed("name") {
		name, _ = c.Flags().GetString("name")
	}

	campaignType := existing.Data.Type
	if c.Flags().Changed("type") {
		campaignType, _ = c.Flags().GetString("type")
	}

	// Build email from existing or flags.
	var existingEmail mailerlite.Emails
	if len(existing.Data.Emails) > 0 {
		e := existing.Data.Emails[0]
		existingEmail = mailerlite.Emails{
			Subject:  e.Subject,
			From:     e.From,
			FromName: e.FromName,
		}
	}

	if c.Flags().Changed("subject") {
		existingEmail.Subject, _ = c.Flags().GetString("subject")
	}
	if c.Flags().Changed("from") {
		existingEmail.From, _ = c.Flags().GetString("from")
	}
	if c.Flags().Changed("from-name") {
		existingEmail.FromName, _ = c.Flags().GetString("from-name")
	}
	if c.Flags().Changed("content") {
		existingEmail.Content, _ = c.Flags().GetString("content")
	}

	opts := &mailerlite.UpdateCampaign{
		Name:   name,
		Type:   campaignType,
		Emails: []mailerlite.Emails{existingEmail},
	}

	if c.Flags().Changed("groups") {
		opts.Groups, _ = c.Flags().GetStringSlice("groups")
	}
	if c.Flags().Changed("segments") {
		opts.Segments, _ = c.Flags().GetStringSlice("segments")
	}

	result, _, err := ml.Campaign.Update(ctx, args[0], opts)
	if err != nil {
		return sdkclient.WrapError(err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	output.Success("Campaign " + args[0] + " updated successfully.")
	return nil
}

// --- schedule ---

var scheduleCmd = &cobra.Command{
	Use:   "schedule <campaign_id>",
	Short: "Schedule a campaign",
	Args:  cobra.ExactArgs(1),
	RunE:  runSchedule,
}

func runSchedule(c *cobra.Command, args []string) error {
	ml, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	delivery, _ := c.Flags().GetString("delivery")

	opts := &mailerlite.ScheduleCampaign{
		Delivery: delivery,
	}

	if strings.EqualFold(delivery, "scheduled") {
		date, _ := c.Flags().GetString("date")
		hours, _ := c.Flags().GetString("hours")
		minutes, _ := c.Flags().GetString("minutes")
		timezoneID, _ := c.Flags().GetInt("timezone-id")

		opts.Schedule = &mailerlite.Schedule{
			Date:       date,
			Hours:      hours,
			Minutes:    minutes,
			TimezoneID: timezoneID,
		}
	}

	ctx := context.Background()
	result, _, err := ml.Campaign.Schedule(ctx, args[0], opts)
	if err != nil {
		return sdkclient.WrapError(err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	output.Success("Campaign " + args[0] + " scheduled successfully.")
	return nil
}

// --- cancel ---

var cancelCmd = &cobra.Command{
	Use:   "cancel <campaign_id>",
	Short: "Cancel a campaign",
	Args:  cobra.ExactArgs(1),
	RunE:  runCancel,
}

func runCancel(c *cobra.Command, args []string) error {
	ml, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	ctx := context.Background()
	result, _, err := ml.Campaign.Cancel(ctx, args[0])
	if err != nil {
		return sdkclient.WrapError(err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result)
	}

	output.Success("Campaign " + args[0] + " cancelled successfully.")
	return nil
}

// --- subscribers ---

var subscribersCmd = &cobra.Command{
	Use:   "subscribers <campaign_id>",
	Short: "List campaign subscriber activity",
	Args:  cobra.ExactArgs(1),
	RunE:  runSubscribers,
}

func runSubscribers(c *cobra.Command, args []string) error {
	ml, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	limit, _ := c.Flags().GetInt("limit")

	ctx := context.Background()

	subscribers, err := sdkclient.FetchAll(ctx, func(ctx context.Context, page, perPage int) ([]mailerlite.CampaignSubscriber, bool, error) {
		opts := &mailerlite.ListCampaignSubscriberOptions{
			CampaignID: args[0],
			Page:       page,
			Limit:      perPage,
		}

		root, _, err := ml.Campaign.Subscribers(ctx, opts)
		if err != nil {
			return nil, false, sdkclient.WrapError(err)
		}

		return root.Data, !root.Links.IsLastPage(), nil
	}, limit)
	if err != nil {
		return err
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(subscribers)
	}

	headers := []string{"ID", "EMAIL", "OPENS", "CLICKS"}
	var rows [][]string

	for _, s := range subscribers {
		rows = append(rows, []string{
			s.ID,
			s.Subscriber.Email,
			strconv.Itoa(s.OpensCount),
			strconv.Itoa(s.ClicksCount),
		})
	}

	output.Table(headers, rows)
	return nil
}

// --- languages ---

var languagesCmd = &cobra.Command{
	Use:   "languages",
	Short: "List campaign languages",
	RunE:  runLanguages,
}

func runLanguages(c *cobra.Command, _ []string) error {
	ml, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	ctx := context.Background()
	result, _, err := ml.Campaign.Languages(ctx)
	if err != nil {
		return sdkclient.WrapError(err)
	}

	if cmdutil.JSONFlag(c) {
		return output.JSON(result.Data)
	}

	headers := []string{"ID", "NAME", "SHORTCODE"}
	var rows [][]string

	for _, lang := range result.Data {
		rows = append(rows, []string{
			lang.Id,
			lang.Name,
			lang.Shortcode,
		})
	}

	output.Table(headers, rows)
	return nil
}

// --- delete ---

var deleteCmd = &cobra.Command{
	Use:   "delete <campaign_id>",
	Short: "Delete a campaign",
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

func runDelete(c *cobra.Command, args []string) error {
	ml, err := cmdutil.NewSDKClient(c)
	if err != nil {
		return err
	}

	if !cmdutil.YesFlag(c) && prompt.IsInteractive() {
		ok, err := prompt.Confirm("Are you sure you want to delete campaign " + args[0] + "?")
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	ctx := context.Background()
	_, err = ml.Campaign.Delete(ctx, args[0])
	if err != nil {
		return sdkclient.WrapError(err)
	}

	output.Success("Campaign " + args[0] + " deleted successfully.")
	return nil
}
