package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"quickpod-cli/internal/app"
)

func newStoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "store",
		Short: "Browse host stores and manage your own host store profile",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List public host stores",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			var items []map[string]any
			if err := getJSON(ctx, "/update/host_stores", false, &items); err != nil {
				return err
			}
			return renderTableOrJSON(items, []string{"ID", "STORE", "SLUG", "USER", "UPDATED"}, genericRows(items, "id", "store_name", "slug", "user_id", "updated_at"))
		},
	})
	cmd.AddCommand(newStoreUpsertCmd())
	return cmd
}

func newStoreUpsertCmd() *cobra.Command {
	var name string
	var slug string
	var bannerURL string
	var avatarURL string

	cmd := &cobra.Command{
		Use:   "upsert",
		Short: "Create or update your host store",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			ctx := context.Background()
			payload := map[string]any{
				"store_name": name,
				"slug":       slug,
			}
			if cmd.Flags().Changed("banner-url") {
				payload["banner_url"] = bannerURL
			}
			if cmd.Flags().Changed("avatar-url") {
				payload["avatar_url"] = avatarURL
			}
			var response map[string]any
			if err := postJSON(ctx, "/update/host_store", payload, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"id", app.StringValue(response["id"])}, {"store_name", app.StringValue(response["store_name"])}, {"slug", app.StringValue(response["slug"])}})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "store name")
	cmd.Flags().StringVar(&slug, "slug", "", "custom slug; if omitted the API generates one")
	cmd.Flags().StringVar(&bannerURL, "banner-url", "", "banner image URL")
	cmd.Flags().StringVar(&avatarURL, "avatar-url", "", "avatar image URL")
	return cmd
}