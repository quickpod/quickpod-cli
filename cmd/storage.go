package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"quickpod-cli/internal/app"
)

func newStorageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "storage",
		Short: "List storage servers and manage user volumes",
	}

	cmd.AddCommand(newStorageServersCmd())
	cmd.AddCommand(newStorageVolumesCmd())
	return cmd
}

func newStorageServersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "servers",
		Short: "List active storage servers available to authenticated users",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			ctx := context.Background()
			var items []map[string]any
			if err := getJSON(ctx, "/update/storage_servers", true, &items); err != nil {
				return err
			}
			return renderTableOrJSON(items, []string{"ID", "HOSTNAME", "LOCATION", "STATUS", "AVAILABLE", "RATE/TB/H"}, genericRows(items, "id", "hostname", "location", "status", "available_storage", "per_tb_hourly_rate"))
		},
	}
	return cmd
}

func newStorageVolumesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volumes",
		Short: "Manage user NFS volumes",
	}
	cmd.AddCommand(newStorageVolumesListCmd())
	cmd.AddCommand(newStorageVolumesGetCmd())
	cmd.AddCommand(newStorageVolumesCreateCmd())
	cmd.AddCommand(newStorageVolumesDeleteCmd())
	return cmd
}

func newStorageVolumesListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List your volumes",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			ctx := context.Background()
			var items []map[string]any
			if err := getJSON(ctx, "/update/user_volumes", true, &items); err != nil {
				return err
			}
			return renderTableOrJSON(items, []string{"ID", "NAME", "SIZE_GB", "STATUS", "SERVER", "MOUNT", "RATE/H"}, genericRows(items, "id", "volume_name", "allocated_size_gb", "status", "storage_server_id", "mount_server", "hourly_rate"))
		},
	}
	return cmd
}

func newStorageVolumesGetCmd() *cobra.Command {
	var volumeID int
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get one volume by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if volumeID <= 0 {
				return fmt.Errorf("--id is required")
			}
			ctx := context.Background()
			var response map[string]any
			if err := getJSON(ctx, fmt.Sprintf("/update/user_volumes/%d", volumeID), true, &response); err != nil {
				return err
			}
			rows := [][]string{{"id", app.StringValue(response["id"])}, {"volume_name", app.StringValue(response["volume_name"])}, {"status", app.StringValue(response["status"])}, {"mount_server", app.StringValue(response["mount_server"])}, {"docker_volume", app.StringValue(response["docker_volume"])}}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, rows)
		},
	}
	cmd.Flags().IntVar(&volumeID, "id", 0, "volume ID")
	return cmd
}

func newStorageVolumesCreateCmd() *cobra.Command {
	var storageServerID int
	var name string
	var sizeGB int
	var allowedHosts []string
	var nfsOptions string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Provision a new user volume",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if storageServerID <= 0 || name == "" || sizeGB <= 0 {
				return fmt.Errorf("--server-id, --name, and --size-gb are required")
			}
			ctx := context.Background()
			requestBody := map[string]any{
				"storage_server_id": storageServerID,
				"volume_name":       name,
				"volume_size_gb":    sizeGB,
				"allowed_hosts":     allowedHosts,
				"nfs_options":       nfsOptions,
			}
			var response map[string]any
			if err := postJSON(ctx, "/update/user_volumes", requestBody, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"id", app.StringValue(response["id"])}, {"volume_name", app.StringValue(response["volume_name"])}, {"mount_server", app.StringValue(response["mount_server"])}, {"docker_volume", app.StringValue(response["docker_volume"])}})
		},
	}
	cmd.Flags().IntVar(&storageServerID, "server-id", 0, "storage server ID")
	cmd.Flags().StringVar(&name, "name", "", "volume name")
	cmd.Flags().IntVar(&sizeGB, "size-gb", 0, "volume size in GB")
	cmd.Flags().StringSliceVar(&allowedHosts, "allowed-host", nil, "repeatable NFS allowed host entries")
	cmd.Flags().StringVar(&nfsOptions, "nfs-options", "", "custom NFS options")
	return cmd
}

func newStorageVolumesDeleteCmd() *cobra.Command {
	var volumeID int
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a user volume",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if volumeID <= 0 {
				return fmt.Errorf("--id is required")
			}
			ctx := context.Background()
			var response map[string]any
			if err := deleteJSON(ctx, fmt.Sprintf("/update/user_volumes/%d", volumeID), url.Values{}, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"id", app.StringValue(response["id"])}, {"status", app.StringValue(response["status"])}, {"volume_name", app.StringValue(response["volume_name"])}})
		},
	}
	cmd.Flags().IntVar(&volumeID, "id", 0, "volume ID")
	return cmd
}