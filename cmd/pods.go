package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"quickpod-cli/internal/app"
)

func newPodsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pods",
		Short: "Manage GPU and CPU pods",
	}

	cmd.AddCommand(newPodsListCmd())
	cmd.AddCommand(newPodsHistoryCmd())
	cmd.AddCommand(newPodsCreateCmd())
	cmd.AddCommand(newPodsResetCmd())
	cmd.AddCommand(newPodLifecycleCmd("start"))
	cmd.AddCommand(newPodLifecycleCmd("stop"))
	cmd.AddCommand(newPodLifecycleCmd("restart"))
	cmd.AddCommand(newPodLifecycleCmd("destroy"))
	cmd.AddCommand(newPodLifecycleCmd("logs"))
	cmd.AddCommand(newPodsRenameCmd())

	return cmd
}

func newPodsListCmd() *cobra.Command {
	var kind string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List your GPU or CPU pods",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			ctx := context.Background()
			endpoint := "/mypods"
			if kind == "cpu" {
				endpoint = "/mypods_cpu"
			}
			var items []map[string]any
			if err := getJSON(ctx, endpoint, true, &items); err != nil {
				return err
			}
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, []string{
					app.StringValue(item["id"]),
					app.Truncate(firstNonEmpty(app.StringValue(item["altname"]), app.StringValue(item["Names"])), 28),
					app.StringValue(item["State"]),
					app.StringValue(item["Status"]),
					app.StringValue(item["hourly_cost"]),
					app.StringValue(item["public_ipaddr"]),
					app.StringValue(item["open_port_end"]),
				})
			}
			return renderTableOrJSON(items, []string{"ID", "NAME", "STATE", "STATUS", "HOURLY", "IP", "PORT"}, rows)
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "gpu", "pod kind: gpu or cpu")
	return cmd
}

func newPodsHistoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show historical pod activity",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			ctx := context.Background()
			var items []map[string]any
			if err := getJSON(ctx, "/my_pods_history", true, &items); err != nil {
				return err
			}
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, []string{
					app.StringValue(item["id"]),
					app.Truncate(firstNonEmpty(app.StringValue(item["altname"]), app.StringValue(item["Names"])), 28),
					app.StringValue(item["created_at"]),
					app.StringValue(item["pod_cost"]),
					app.StringValue(item["intended_state"]),
					app.StringValue(item["public_ipaddr"]),
				})
			}
			return renderTableOrJSON(items, []string{"ID", "NAME", "CREATED", "COST", "INTENDED", "IP"}, rows)
		},
	}
	return cmd
}

func newPodsCreateCmd() *cobra.Command {
	var kind string
	var job bool
	var templateUUID string
	var offerID int
	var diskSize string
	var dockerOptions string
	var altName string
	var couponCode string
	var volumeID int

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a GPU or CPU pod or job",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if templateUUID == "" || diskSize == "" {
				return fmt.Errorf("--template and --disk are required")
			}
			ctx := context.Background()
			endpoint := "/update/createpod"
			if job {
				endpoint = "/update/createjob"
			}
			if kind == "cpu" {
				endpoint = "/update/createpod_cpu"
				if job {
					endpoint = "/update/createjob_cpu"
				}
			}

			requestBody := map[string]any{
				"template_uuid":  templateUUID,
				"offers_id":      offerID,
				"disk_size":      diskSize,
				"docker_options": dockerOptions,
				"altname":        altName,
				"coupon_code":    couponCode,
			}
			if volumeID > 0 {
				requestBody["volume_id"] = volumeID
			}

			var response map[string]any
			if err := postJSON(ctx, endpoint, requestBody, true, &response); err != nil {
				return err
			}
			rows := [][]string{
				{"status", app.StringValue(response["status"])},
				{"message", app.StringValue(response["message"])},
				{"pod_uuid", app.StringValue(response["pod_uuid"])},
				{"public_ipaddress", app.StringValue(response["public_ipaddress"])},
				{"open_port_start", app.StringValue(response["open_port_start"])},
				{"open_port_end", app.StringValue(response["open_port_end"])},
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, rows)
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "gpu", "pod kind: gpu or cpu")
	cmd.Flags().BoolVar(&job, "job", false, "create a job pod instead of an interactive pod")
	cmd.Flags().StringVar(&templateUUID, "template", "", "template UUID")
	cmd.Flags().IntVar(&offerID, "offer", 0, "offer ID")
	cmd.Flags().StringVar(&diskSize, "disk", "", "disk size in GB")
	cmd.Flags().StringVar(&dockerOptions, "docker-options", "", "extra docker options")
	cmd.Flags().StringVar(&altName, "name", "", "friendly pod name")
	cmd.Flags().StringVar(&couponCode, "coupon", "", "coupon code")
	cmd.Flags().IntVar(&volumeID, "volume-id", 0, "optional attached user volume ID")
	return cmd
}

func newPodsResetCmd() *cobra.Command {
	var kind string
	var podUUID string
	var templateUUID string
	var diskSize int
	var dockerOptions string
	var altName string
	var couponCode string

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset a GPU or CPU pod with a new template payload",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if podUUID == "" || templateUUID == "" || diskSize <= 0 {
				return fmt.Errorf("--pod, --template, and --disk are required")
			}
			ctx := context.Background()
			endpoint := "/update/resetpod"
			if kind == "cpu" {
				endpoint = "/update/resetpod_cpu"
			}
			requestBody := map[string]any{
				"pod_uuid":       podUUID,
				"template_uuid":  templateUUID,
				"disk_size":      diskSize,
				"docker_options": dockerOptions,
				"altname":        altName,
				"coupon_code":    couponCode,
			}
			var response map[string]any
			if err := postJSON(ctx, endpoint, requestBody, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"status", app.StringValue(response["status"])}, {"message", app.StringValue(response["message"])}, {"pod_uuid", app.StringValue(response["pod_uuid"])}})
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "gpu", "pod kind: gpu or cpu")
	cmd.Flags().StringVar(&podUUID, "pod", "", "pod UUID")
	cmd.Flags().StringVar(&templateUUID, "template", "", "replacement template UUID")
	cmd.Flags().IntVar(&diskSize, "disk", 0, "disk size in GB")
	cmd.Flags().StringVar(&dockerOptions, "docker-options", "", "extra docker options")
	cmd.Flags().StringVar(&altName, "name", "", "friendly pod name")
	cmd.Flags().StringVar(&couponCode, "coupon", "", "coupon code")
	return cmd
}

func newPodLifecycleCmd(action string) *cobra.Command {
	var kind string
	var podUUID string

	cmd := &cobra.Command{
		Use:   action,
		Short: fmt.Sprintf("%s a GPU or CPU pod", action),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if podUUID == "" {
				return fmt.Errorf("--pod is required")
			}

			base := map[string]string{
				"start":   "/update/startpod",
				"stop":    "/update/stoppod",
				"restart": "/update/restartpod",
				"destroy": "/update/destroypod",
				"logs":    "/update/podlogs",
			}[action]
			if kind == "cpu" {
				base += "_cpu"
			}

			query := url.Values{}
			query.Set("pod_uuid", podUUID)
			ctx := context.Background()
			var response map[string]any
			if err := getJSONQuery(ctx, base, query, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"message", app.StringValue(response["message"])}, {"pod_uuid", app.StringValue(response["pod_uuid"])}})
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "gpu", "pod kind: gpu or cpu")
	cmd.Flags().StringVar(&podUUID, "pod", "", "pod UUID")
	return cmd
}

func newPodsRenameCmd() *cobra.Command {
	var podID int
	var name string
	cmd := &cobra.Command{
		Use:   "rename",
		Short: "Rename a pod by internal pod ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if podID <= 0 || name == "" {
				return fmt.Errorf("--pod-id and --name are required")
			}
			ctx := context.Background()
			var response map[string]any
			if err := postJSON(ctx, "/update/renamepod", map[string]any{"podid": podID, "alt_name": name}, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"result", app.StringValue(response["result"])}})
		},
	}
	cmd.Flags().IntVar(&podID, "pod-id", 0, "numeric pod ID")
	cmd.Flags().StringVar(&name, "name", "", "new pod display name")
	return cmd
}