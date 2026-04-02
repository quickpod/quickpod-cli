package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"quickpod-cli/internal/app"
)

func newClustersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clusters",
		Short:   "Manage pod clusters and stable cluster services",
		Example: "  quickpod clusters list\n  quickpod clusters get --id 12\n  quickpod clusters create --file ./cluster.json\n  quickpod clusters scale --id 12 --replicas 4 --offer-id 101 --offer-id 102",
	}

	cmd.AddCommand(newClustersListCmd())
	cmd.AddCommand(newClustersGetCmd())
	cmd.AddCommand(newClustersCreateCmd())
	cmd.AddCommand(newClustersScaleCmd())
	cmd.AddCommand(newClustersUpdateConfigCmd())
	cmd.AddCommand(newClustersStartStopCmd("start"))
	cmd.AddCommand(newClustersStartStopCmd("stop"))
	cmd.AddCommand(newClustersStartStopCmd("redeploy"))
	cmd.AddCommand(newClustersDeleteCmd())
	cmd.AddCommand(newClusterServicesCmd())
	cmd.AddCommand(newClusterReplicasCmd())

	return cmd
}

func newClustersListCmd() *cobra.Command {
	var wide bool
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List your pod clusters",
		Example: "  quickpod clusters list\n  quickpod clusters list --wide\n  quickpod --output json clusters list",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			ctx := context.Background()
			var items []map[string]any
			if err := getJSON(ctx, "/update/auth/pod_clusters", true, &items); err != nil {
				return err
			}
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				replicas := app.SliceValue(item, "replicas")
				services := app.SliceValue(item, "services")
				if wide {
					rows = append(rows, []string{
						app.StringValue(item["id"]),
						app.Truncate(app.StringValue(item["name"]), 24),
						app.StringValue(item["pod_type"]),
						app.StringValue(item["desired_replicas"]),
						app.StringValue(len(replicas)),
						app.StringValue(len(services)),
						valueOrDash(app.StringValue(item["storage_mode"])),
						valueOrDash(app.StringValue(item["shared_volume_id"])),
						app.Truncate(app.StringValue(item["template_uuid"]), 16),
						app.StringValue(item["updated_at"]),
					})
					continue
				}
				rows = append(rows, []string{
					app.StringValue(item["id"]),
					app.Truncate(app.StringValue(item["name"]), 24),
					app.StringValue(item["pod_type"]),
					app.StringValue(item["desired_replicas"]),
					app.StringValue(len(replicas)),
					app.StringValue(len(services)),
					valueOrDash(app.StringValue(item["storage_mode"])),
				})
			}
			if wide {
				return renderTableOrJSON(items, []string{"ID", "NAME", "TYPE", "DESIRED", "REPLICAS", "SERVICES", "STORAGE", "SHARED_VOL", "TEMPLATE", "UPDATED"}, rows)
			}
			return renderTableOrJSON(items, []string{"ID", "NAME", "TYPE", "DESIRED", "REPLICAS", "SERVICES", "STORAGE"}, rows)
		},
	}
	cmd.Flags().BoolVar(&wide, "wide", false, "show additional cluster columns")
	return cmd
}

func newClustersGetCmd() *cobra.Command {
	var clusterID int64
	cmd := &cobra.Command{
		Use:     "get",
		Short:   "Describe one pod cluster",
		Example: "  quickpod clusters get --id 12",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if clusterID <= 0 {
				return fmt.Errorf("--id is required")
			}
			ctx := context.Background()
			var response map[string]any
			if err := getJSON(ctx, fmt.Sprintf("/update/auth/pod_clusters/%d", clusterID), true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, orderedDisplayKeyValueRows(response, "id", "name", "pod_type", "desired_replicas", "storage_mode", "shared_volume_id", "template_uuid", "coupon_code", "disk_size", "docker_options", "altname_prefix", "secrets", "rollout", "autoscaling", "schedules", "placement", "replicas", "services", "created_at", "updated_at"))
		},
	}
	cmd.Flags().Int64Var(&clusterID, "id", 0, "cluster ID")
	return cmd
}

func newClustersCreateCmd() *cobra.Command {
	var filePath string
	var name string
	var podType string
	var templateUUID string
	var diskSize int
	var couponCode string
	var dockerOptions string
	var altNamePrefix string
	var offerIDs []string
	var storageMode string
	var sharedVolumeID int64
	var shardVolumeIDs []string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a pod cluster from JSON or flags",
		Example: "  quickpod clusters create --file ./cluster.json\n  quickpod clusters create --name trainers --kind gpu --template TEMPLATE_UUID --disk 50 --offer-id 101 --offer-id 102 --storage-mode shared --shared-volume-id 44",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			payload := map[string]any{}
			if filePath != "" {
				filePayload, err := app.ReadJSONFile(filePath)
				if err != nil {
					return err
				}
				payload = app.MergeMap(payload, filePayload)
			}
			setIfChanged := func(flagName string, key string, value any) {
				if cmd.Flags().Changed(flagName) {
					payload[key] = value
				}
			}
			setIfChanged("name", "name", name)
			setIfChanged("kind", "pod_type", strings.ToUpper(strings.TrimSpace(podType)))
			setIfChanged("template", "template_uuid", templateUUID)
			setIfChanged("disk", "disk_size", diskSize)
			setIfChanged("coupon", "coupon_code", couponCode)
			setIfChanged("docker-options", "docker_options", dockerOptions)
			setIfChanged("altname-prefix", "altname_prefix", altNamePrefix)
			if cmd.Flags().Changed("offer-id") {
				parsed, err := parseIntSlice(offerIDs)
				if err != nil {
					return err
				}
				payload["offer_ids"] = parsed
			}
			if cmd.Flags().Changed("storage-mode") || cmd.Flags().Changed("shared-volume-id") || cmd.Flags().Changed("shard-volume-id") {
				storage := app.MapValue(payload, "storage")
				if storage == nil {
					storage = map[string]any{}
				}
				if cmd.Flags().Changed("storage-mode") {
					storage["mode"] = storageMode
				}
				if cmd.Flags().Changed("shared-volume-id") {
					storage["shared_volume_id"] = sharedVolumeID
				}
				if cmd.Flags().Changed("shard-volume-id") {
					parsed, err := parseInt64Slice(shardVolumeIDs)
					if err != nil {
						return err
					}
					storage["shard_volume_ids"] = parsed
				}
				payload["storage"] = storage
			}
			if len(payload) == 0 {
				return fmt.Errorf("no cluster fields were provided")
			}
			ctx := context.Background()
			var response map[string]any
			if err := postJSON(ctx, "/update/auth/pod_clusters", payload, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, orderedDisplayKeyValueRows(response, "warning", "id", "name", "pod_type", "desired_replicas", "replicas", "services", "cluster"))
		},
	}

	cmd.Flags().StringVar(&filePath, "file", "", "path to a JSON cluster payload")
	cmd.Flags().StringVar(&name, "name", "", "cluster name")
	cmd.Flags().StringVar(&podType, "kind", "gpu", "cluster type: gpu or cpu")
	cmd.Flags().StringVar(&templateUUID, "template", "", "template UUID")
	cmd.Flags().IntVar(&diskSize, "disk", 0, "disk size in GB")
	cmd.Flags().StringVar(&couponCode, "coupon", "", "coupon code")
	cmd.Flags().StringVar(&dockerOptions, "docker-options", "", "extra docker options")
	cmd.Flags().StringVar(&altNamePrefix, "altname-prefix", "", "prefix for replica display names")
	cmd.Flags().StringSliceVar(&offerIDs, "offer-id", nil, "repeatable offer IDs used for initial replicas")
	cmd.Flags().StringVar(&storageMode, "storage-mode", "", "storage mode: none, shared, or sharded")
	cmd.Flags().Int64Var(&sharedVolumeID, "shared-volume-id", 0, "shared storage volume ID")
	cmd.Flags().StringSliceVar(&shardVolumeIDs, "shard-volume-id", nil, "repeatable shard volume IDs")
	return cmd
}

func newClustersScaleCmd() *cobra.Command {
	var clusterID int64
	var replicas int
	var offerIDs []string
	var shardVolumeIDs []string
	cmd := &cobra.Command{
		Use:     "scale",
		Short:   "Scale a pod cluster to a desired replica count",
		Example: "  quickpod clusters scale --id 12 --replicas 5 --offer-id 101 --offer-id 102",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if clusterID <= 0 || replicas < 0 {
				return fmt.Errorf("--id and a non-negative --replicas are required")
			}
			payload := map[string]any{"desired_replicas": replicas}
			if cmd.Flags().Changed("offer-id") {
				parsed, err := parseIntSlice(offerIDs)
				if err != nil {
					return err
				}
				payload["offer_ids"] = parsed
			}
			if cmd.Flags().Changed("shard-volume-id") {
				parsed, err := parseInt64Slice(shardVolumeIDs)
				if err != nil {
					return err
				}
				payload["shard_volume_ids"] = parsed
			}
			ctx := context.Background()
			var response map[string]any
			if err := postJSON(ctx, fmt.Sprintf("/update/auth/pod_clusters/%d/scale", clusterID), payload, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, orderedDisplayKeyValueRows(response, "id", "name", "desired_replicas", "replicas", "services", "updated_at"))
		},
	}
	cmd.Flags().Int64Var(&clusterID, "id", 0, "cluster ID")
	cmd.Flags().IntVar(&replicas, "replicas", -1, "desired replica count")
	cmd.Flags().StringSliceVar(&offerIDs, "offer-id", nil, "repeatable offer IDs for scale-up placement")
	cmd.Flags().StringSliceVar(&shardVolumeIDs, "shard-volume-id", nil, "repeatable shard volume IDs")
	return cmd
}

func newClustersUpdateConfigCmd() *cobra.Command {
	var clusterID int64
	var filePath string
	cmd := &cobra.Command{
		Use:     "update-config",
		Short:   "Update rollout, autoscaling, schedules, or placement from a JSON file",
		Example: "  quickpod clusters update-config --id 12 --file ./cluster-config.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if clusterID <= 0 || strings.TrimSpace(filePath) == "" {
				return fmt.Errorf("--id and --file are required")
			}
			payload, err := app.ReadJSONFile(filePath)
			if err != nil {
				return err
			}
			ctx := context.Background()
			var response map[string]any
			if err := putJSON(ctx, fmt.Sprintf("/update/auth/pod_clusters/%d/config", clusterID), payload, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, orderedDisplayKeyValueRows(response, "result", "message", "cluster"))
		},
	}
	cmd.Flags().Int64Var(&clusterID, "id", 0, "cluster ID")
	cmd.Flags().StringVar(&filePath, "file", "", "path to a JSON cluster config payload")
	return cmd
}

func newClustersStartStopCmd(action string) *cobra.Command {
	var clusterID int64
	cmd := &cobra.Command{
		Use:     action,
		Short:   fmt.Sprintf("%s a pod cluster", action),
		Example: fmt.Sprintf("  quickpod clusters %s --id 12", action),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if clusterID <= 0 {
				return fmt.Errorf("--id is required")
			}
			ctx := context.Background()
			var response map[string]any
			if err := postJSON(ctx, fmt.Sprintf("/update/auth/pod_clusters/%d/%s", clusterID, action), map[string]any{}, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, orderedDisplayKeyValueRows(response, "result", "message", "cluster", "id", "name", "desired_replicas", "replicas"))
		},
	}
	cmd.Flags().Int64Var(&clusterID, "id", 0, "cluster ID")
	return cmd
}

func newClustersDeleteCmd() *cobra.Command {
	var clusterID int64
	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "Delete a pod cluster",
		Example: "  quickpod clusters delete --id 12",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if clusterID <= 0 {
				return fmt.Errorf("--id is required")
			}
			ctx := context.Background()
			var response map[string]any
			if err := deleteJSON(ctx, fmt.Sprintf("/update/auth/pod_clusters/%d", clusterID), url.Values{}, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, orderedDisplayKeyValueRows(response, "result", "message", "cluster_id", "cluster"))
		},
	}
	cmd.Flags().Int64Var(&clusterID, "id", 0, "cluster ID")
	return cmd
}

func newClusterServicesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "services",
		Short: "Manage stable service endpoints for a pod cluster",
	}
	cmd.AddCommand(newClusterServicesListCmd())
	cmd.AddCommand(newClusterServicesCreateCmd())
	cmd.AddCommand(newClusterServicesUpdateCmd())
	cmd.AddCommand(newClusterServicesDeleteCmd())
	return cmd
}

func newClusterServicesListCmd() *cobra.Command {
	var clusterID int64
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List stable service endpoints for one cluster",
		Example: "  quickpod clusters services list --cluster-id 12",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if clusterID <= 0 {
				return fmt.Errorf("--cluster-id is required")
			}
			ctx := context.Background()
			var items []map[string]any
			if err := getJSON(ctx, fmt.Sprintf("/update/auth/pod_clusters/%d/services", clusterID), true, &items); err != nil {
				return err
			}
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, []string{
					app.StringValue(item["id"]),
					app.Truncate(app.StringValue(item["name"]), 22),
					app.StringValue(item["target_port"]),
					valueOrDash(app.StringValue(item["protocol"])),
					app.Truncate(valueOrDash(app.StringValue(item["endpoint_url"])), 34),
					app.StringValue(item["healthy_replica_count"]),
				})
			}
			return renderTableOrJSON(items, []string{"ID", "NAME", "TARGET", "PROTO", "ENDPOINT", "HEALTHY"}, rows)
		},
	}
	cmd.Flags().Int64Var(&clusterID, "cluster-id", 0, "cluster ID")
	return cmd
}

func newClusterServicesCreateCmd() *cobra.Command {
	var clusterID int64
	var filePath string
	var name string
	var targetPort int
	var protocol string
	var contextPath string
	var healthProtocol string
	var healthPath string
	var healthPort int
	var expectedStatus int
	var checkInterval int
	var readinessTimeout int
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a stable service endpoint for a cluster",
		Example: "  quickpod clusters services create --cluster-id 12 --name api --target-port 8000 --protocol http --context-path /",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if clusterID <= 0 {
				return fmt.Errorf("--cluster-id is required")
			}
			payload := map[string]any{}
			if filePath != "" {
				filePayload, err := app.ReadJSONFile(filePath)
				if err != nil {
					return err
				}
				payload = app.MergeMap(payload, filePayload)
			}
			setIfChanged := func(flagName string, key string, value any) {
				if cmd.Flags().Changed(flagName) {
					payload[key] = value
				}
			}
			setIfChanged("name", "name", name)
			setIfChanged("target-port", "target_port", targetPort)
			setIfChanged("protocol", "protocol", protocol)
			setIfChanged("context-path", "context_path", contextPath)
			setIfChanged("health-protocol", "health_check_protocol", healthProtocol)
			setIfChanged("health-path", "health_check_path", healthPath)
			setIfChanged("health-port", "health_check_port", healthPort)
			setIfChanged("expected-status", "expected_status", expectedStatus)
			setIfChanged("check-interval", "check_interval_seconds", checkInterval)
			setIfChanged("readiness-timeout", "readiness_timeout_seconds", readinessTimeout)
			if len(payload) == 0 {
				return fmt.Errorf("no service fields were provided")
			}
			ctx := context.Background()
			var response map[string]any
			if err := postJSON(ctx, fmt.Sprintf("/update/auth/pod_clusters/%d/services", clusterID), payload, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, orderedDisplayKeyValueRows(response, "id", "name", "target_port", "protocol", "endpoint_slug", "endpoint_url", "healthy_replica_count", "backends"))
		},
	}
	cmd.Flags().Int64Var(&clusterID, "cluster-id", 0, "cluster ID")
	cmd.Flags().StringVar(&filePath, "file", "", "path to a JSON service payload")
	cmd.Flags().StringVar(&name, "name", "", "service name")
	cmd.Flags().IntVar(&targetPort, "target-port", 0, "replica target port")
	cmd.Flags().StringVar(&protocol, "protocol", "", "service protocol, for example http or tcp")
	cmd.Flags().StringVar(&contextPath, "context-path", "", "context path for HTTP routing")
	cmd.Flags().StringVar(&healthProtocol, "health-protocol", "", "health check protocol")
	cmd.Flags().StringVar(&healthPath, "health-path", "", "health check path")
	cmd.Flags().IntVar(&healthPort, "health-port", 0, "health check port")
	cmd.Flags().IntVar(&expectedStatus, "expected-status", 0, "expected health check HTTP status")
	cmd.Flags().IntVar(&checkInterval, "check-interval", 0, "health check interval in seconds")
	cmd.Flags().IntVar(&readinessTimeout, "readiness-timeout", 0, "readiness timeout in seconds")
	return cmd
}

func newClusterServicesUpdateCmd() *cobra.Command {
	var clusterID int64
	var serviceID int64
	var filePath string
	var name string
	var targetPort int
	var protocol string
	var contextPath string
	var healthProtocol string
	var healthPath string
	var healthPort int
	var expectedStatus int
	var checkInterval int
	var readinessTimeout int
	cmd := &cobra.Command{
		Use:     "update",
		Short:   "Update a stable service endpoint for a cluster",
		Example: "  quickpod clusters services update --cluster-id 12 --service-id 7 --file ./service-update.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if clusterID <= 0 || serviceID <= 0 {
				return fmt.Errorf("--cluster-id and --service-id are required")
			}
			payload := map[string]any{}
			if filePath != "" {
				filePayload, err := app.ReadJSONFile(filePath)
				if err != nil {
					return err
				}
				payload = app.MergeMap(payload, filePayload)
			}
			setIfChanged := func(flagName string, key string, value any) {
				if cmd.Flags().Changed(flagName) {
					payload[key] = value
				}
			}
			setIfChanged("name", "name", name)
			setIfChanged("target-port", "target_port", targetPort)
			setIfChanged("protocol", "protocol", protocol)
			setIfChanged("context-path", "context_path", contextPath)
			setIfChanged("health-protocol", "health_check_protocol", healthProtocol)
			setIfChanged("health-path", "health_check_path", healthPath)
			setIfChanged("health-port", "health_check_port", healthPort)
			setIfChanged("expected-status", "expected_status", expectedStatus)
			setIfChanged("check-interval", "check_interval_seconds", checkInterval)
			setIfChanged("readiness-timeout", "readiness_timeout_seconds", readinessTimeout)
			if len(payload) == 0 {
				return fmt.Errorf("no service fields were provided")
			}
			ctx := context.Background()
			var response map[string]any
			if err := putJSON(ctx, fmt.Sprintf("/update/auth/pod_clusters/%d/services/%d", clusterID, serviceID), payload, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, orderedDisplayKeyValueRows(response, "id", "name", "target_port", "protocol", "endpoint_slug", "endpoint_url", "healthy_replica_count", "backends"))
		},
	}
	cmd.Flags().Int64Var(&clusterID, "cluster-id", 0, "cluster ID")
	cmd.Flags().Int64Var(&serviceID, "service-id", 0, "service ID")
	cmd.Flags().StringVar(&filePath, "file", "", "path to a JSON service update payload")
	cmd.Flags().StringVar(&name, "name", "", "service name")
	cmd.Flags().IntVar(&targetPort, "target-port", 0, "replica target port")
	cmd.Flags().StringVar(&protocol, "protocol", "", "service protocol, for example http or tcp")
	cmd.Flags().StringVar(&contextPath, "context-path", "", "context path for HTTP routing")
	cmd.Flags().StringVar(&healthProtocol, "health-protocol", "", "health check protocol")
	cmd.Flags().StringVar(&healthPath, "health-path", "", "health check path")
	cmd.Flags().IntVar(&healthPort, "health-port", 0, "health check port")
	cmd.Flags().IntVar(&expectedStatus, "expected-status", 0, "expected health check HTTP status")
	cmd.Flags().IntVar(&checkInterval, "check-interval", 0, "health check interval in seconds")
	cmd.Flags().IntVar(&readinessTimeout, "readiness-timeout", 0, "readiness timeout in seconds")
	return cmd
}

func newClusterServicesDeleteCmd() *cobra.Command {
	var clusterID int64
	var serviceID int64
	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "Delete a stable service endpoint from a cluster",
		Example: "  quickpod clusters services delete --cluster-id 12 --service-id 7",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if clusterID <= 0 || serviceID <= 0 {
				return fmt.Errorf("--cluster-id and --service-id are required")
			}
			ctx := context.Background()
			var response map[string]any
			if err := deleteJSON(ctx, fmt.Sprintf("/update/auth/pod_clusters/%d/services/%d", clusterID, serviceID), url.Values{}, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, orderedDisplayKeyValueRows(response, "result", "message", "cluster", "cluster_id"))
		},
	}
	cmd.Flags().Int64Var(&clusterID, "cluster-id", 0, "cluster ID")
	cmd.Flags().Int64Var(&serviceID, "service-id", 0, "service ID")
	return cmd
}

func newClusterReplicasCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "replicas",
		Short: "Manage individual cluster replicas",
	}
	cmd.AddCommand(newClusterReplicasDeleteCmd())
	return cmd
}

func newClusterReplicasDeleteCmd() *cobra.Command {
	var clusterID int64
	var replicaID int64
	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "Delete one replica from a cluster",
		Example: "  quickpod clusters replicas delete --cluster-id 12 --replica-id 44",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if clusterID <= 0 || replicaID <= 0 {
				return fmt.Errorf("--cluster-id and --replica-id are required")
			}
			ctx := context.Background()
			var response map[string]any
			if err := deleteJSON(ctx, fmt.Sprintf("/update/auth/pod_clusters/%d/replicas/%d", clusterID, replicaID), url.Values{}, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, orderedDisplayKeyValueRows(response, "result", "message", "cluster", "cluster_id"))
		},
	}
	cmd.Flags().Int64Var(&clusterID, "cluster-id", 0, "cluster ID")
	cmd.Flags().Int64Var(&replicaID, "replica-id", 0, "replica ID")
	return cmd
}
