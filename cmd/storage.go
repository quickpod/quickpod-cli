package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

func newStorageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "storage",
		Short:   "List storage servers and manage user volumes",
		Example: "  quickpod storage servers\n  quickpod storage volumes list\n  quickpod storage volumes create --server-id 4 --name datasets --size-gb 250",
	}

	cmd.AddCommand(newStorageServersCmd())
	cmd.AddCommand(newStorageVolumesCmd())
	return cmd
}

func newStorageServersCmd() *cobra.Command {
	var wide bool
	cmd := &cobra.Command{
		Use:     "servers",
		Short:   "List active storage servers available to authenticated users",
		Example: "  quickpod storage servers\n  quickpod storage servers get --id 4\n  quickpod --output json storage servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStorageServersList(wide)
		},
	}
	cmd.Flags().BoolVar(&wide, "wide", false, "show additional storage server columns")
	cmd.AddCommand(newStorageServersListCmd())
	cmd.AddCommand(newStorageServersGetCmd())
	return cmd
}

func newStorageServersListCmd() *cobra.Command {
	var wide bool
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List active storage servers available to authenticated users",
		Example: "  quickpod storage servers list\n  quickpod storage servers list --wide",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStorageServersList(wide)
		},
	}
	cmd.Flags().BoolVar(&wide, "wide", false, "show additional storage server columns")
	return cmd
}

func runStorageServersList(wide bool) error {
	if err := requireAuth(); err != nil {
		return err
	}
	ctx := context.Background()
	var items []map[string]any
	if err := getJSON(ctx, "/update/storage_servers", true, &items); err != nil {
		return err
	}
	if wide {
		return renderTableOrJSON(items, []string{"ID", "HOSTNAME", "LOCATION", "STATUS", "AVAILABLE", "TOTAL", "NFS", "RATE/TB/H", "MOUNT_ROOT", "PUBLIC_IP"}, genericRows(items, "id", "hostname", "location", "status", "available_storage", "total_storage_size", "nfs_port", "per_tb_hourly_rate", "mount_root", "public_ipaddr"))
	}
	return renderTableOrJSON(items, []string{"ID", "HOSTNAME", "LOCATION", "STATUS", "AVAILABLE", "TOTAL", "NFS", "RATE/TB/H"}, genericRows(items, "id", "hostname", "location", "status", "available_storage", "total_storage_size", "nfs_port", "per_tb_hourly_rate"))
}

func newStorageServersGetCmd() *cobra.Command {
	var serverID int
	cmd := &cobra.Command{
		Use:     "get",
		Short:   "Get one storage server by ID",
		Example: "  quickpod storage servers get --id 4",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if serverID <= 0 {
				return fmt.Errorf("--id is required")
			}
			ctx := context.Background()
			var response map[string]any
			if err := getJSON(ctx, fmt.Sprintf("/update/storage_servers/%d", serverID), true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, orderedDisplayKeyValueRows(response, "id", "hostname", "location", "status", "available_storage", "total_storage_size", "nfs_port", "per_tb_hourly_rate", "mount_root", "public_ipaddr"))
		},
	}
	cmd.Flags().IntVar(&serverID, "id", 0, "storage server ID")
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
	var wide bool
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List your volumes",
		Example: "  quickpod storage volumes list\n  quickpod storage volumes list --wide\n  quickpod --output json storage volumes list",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			ctx := context.Background()
			var items []map[string]any
			if err := getJSON(ctx, "/update/user_volumes", true, &items); err != nil {
				return err
			}
			if wide {
				return renderTableOrJSON(items, []string{"ID", "NAME", "SIZE_GB", "STATUS", "SERVER", "MOUNT", "DOCKER_VOLUME", "RATE/H", "NFS_OPTIONS", "ALLOWED_HOSTS"}, genericRows(items, "id", "volume_name", "allocated_size_gb", "status", "storage_server_id", "mount_server", "docker_volume", "hourly_rate", "nfs_options", "allowed_hosts"))
			}
			return renderTableOrJSON(items, []string{"ID", "NAME", "SIZE_GB", "STATUS", "SERVER", "MOUNT", "DOCKER_VOLUME", "RATE/H"}, genericRows(items, "id", "volume_name", "allocated_size_gb", "status", "storage_server_id", "mount_server", "docker_volume", "hourly_rate"))
		},
	}
	cmd.Flags().BoolVar(&wide, "wide", false, "show additional volume columns")
	return cmd
}

func newStorageVolumesGetCmd() *cobra.Command {
	var volumeID int
	cmd := &cobra.Command{
		Use:     "get",
		Short:   "Get one volume by ID",
		Example: "  quickpod storage volumes get --id 42",
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
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, orderedDisplayKeyValueRows(response, "id", "volume_name", "allocated_size_gb", "status", "storage_server_id", "mount_server", "docker_volume", "hourly_rate", "nfs_options", "allowed_hosts"))
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
		Use:     "create",
		Short:   "Provision a new user volume",
		Example: "  quickpod storage volumes create --server-id 4 --name datasets --size-gb 250 --allowed-host 10.0.0.10",
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
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, orderedKeyValueRows(response, "id", "volume_name", "allocated_size_gb", "status", "mount_server", "docker_volume", "hourly_rate"))
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
		Use:     "delete",
		Short:   "Delete a user volume",
		Example: "  quickpod storage volumes delete --id 42",
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
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, orderedKeyValueRows(response, "id", "status", "volume_name"))
		},
	}
	cmd.Flags().IntVar(&volumeID, "id", 0, "volume ID")
	return cmd
}
