package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"quickpod-cli/internal/app"
)

func newServerlessCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "serverless",
		Short:   "Manage serverless endpoints backed by cluster services",
		Example: "  quickpod serverless list\n  quickpod serverless get --id 9\n  quickpod serverless create --file ./endpoint.json\n  quickpod serverless logs --id 9 --limit 25",
	}
	cmd.AddCommand(newServerlessListCmd())
	cmd.AddCommand(newServerlessGetCmd())
	cmd.AddCommand(newServerlessCreateCmd())
	cmd.AddCommand(newServerlessUpdateCmd())
	cmd.AddCommand(newServerlessDeleteCmd())
	cmd.AddCommand(newServerlessLogsCmd())
	return cmd
}

func newServerlessListCmd() *cobra.Command {
	var wide bool
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List your serverless endpoints",
		Example: "  quickpod serverless list\n  quickpod serverless list --wide\n  quickpod --output json serverless list",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			ctx := context.Background()
			var items []map[string]any
			if err := getJSON(ctx, "/update/auth/serverless/endpoints", true, &items); err != nil {
				return err
			}
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				if wide {
					rows = append(rows, []string{
						app.StringValue(item["id"]),
						app.Truncate(app.StringValue(item["name"]), 22),
						app.StringValue(item["slug"]),
						app.StringValue(item["auth_mode"]),
						boolString(app.BoolValue(item["active"])),
						app.StringValue(item["hot_service_id"]),
						valueOrDash(app.StringValue(item["warm_service_id"])),
						app.StringValue(item["total_requests"]),
						app.StringValue(item["last_status_code"]),
						app.StringValue(item["request_timeout_seconds"]),
					})
					continue
				}
				rows = append(rows, []string{
					app.StringValue(item["id"]),
					app.Truncate(app.StringValue(item["name"]), 22),
					app.StringValue(item["slug"]),
					app.StringValue(item["auth_mode"]),
					boolString(app.BoolValue(item["active"])),
					app.StringValue(item["hot_service_id"]),
					app.Truncate(valueOrDash(app.StringValue(item["invoke_path"])), 34),
				})
			}
			if wide {
				return renderTableOrJSON(items, []string{"ID", "NAME", "SLUG", "AUTH", "ACTIVE", "HOT_SVC", "WARM_SVC", "REQUESTS", "LAST", "TIMEOUT"}, rows)
			}
			return renderTableOrJSON(items, []string{"ID", "NAME", "SLUG", "AUTH", "ACTIVE", "HOT_SVC", "INVOKE"}, rows)
		},
	}
	cmd.Flags().BoolVar(&wide, "wide", false, "show additional endpoint columns")
	return cmd
}

func newServerlessGetCmd() *cobra.Command {
	var endpointID int64
	cmd := &cobra.Command{
		Use:     "get",
		Short:   "Describe one serverless endpoint",
		Example: "  quickpod serverless get --id 9",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if endpointID <= 0 {
				return fmt.Errorf("--id is required")
			}
			ctx := context.Background()
			var response map[string]any
			if err := getJSON(ctx, fmt.Sprintf("/update/auth/serverless/endpoints/%d", endpointID), true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, orderedDisplayKeyValueRows(response, "id", "name", "slug", "description", "auth_mode", "active", "hot_service_id", "warm_service_id", "request_timeout_seconds", "base_price_per_call", "price_per_100ms", "allowed_host_ids", "invoke_path", "last_status_code", "total_requests", "total_billed_amount", "hot_service", "warm_service", "created_at", "updated_at"))
		},
	}
	cmd.Flags().Int64Var(&endpointID, "id", 0, "serverless endpoint ID")
	return cmd
}

func newServerlessCreateCmd() *cobra.Command {
	var filePath string
	var name string
	var slug string
	var description string
	var hotServiceID int64
	var warmServiceID int64
	var authMode string
	var authToken string
	var requestTimeout int
	var basePrice float64
	var pricePer100ms float64
	var active string
	var allowedHostIDs []string
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a serverless endpoint from JSON or flags",
		Example: "  quickpod serverless create --file ./endpoint.json\n  quickpod serverless create --name inference --slug inference --hot-service-id 77 --auth-mode public --request-timeout 60",
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
			setIfChanged("slug", "slug", slug)
			setIfChanged("description", "description", description)
			setIfChanged("hot-service-id", "hot_service_id", hotServiceID)
			setIfChanged("warm-service-id", "warm_service_id", warmServiceID)
			setIfChanged("auth-mode", "auth_mode", authMode)
			setIfChanged("auth-token", "auth_token", authToken)
			setIfChanged("request-timeout", "request_timeout_seconds", requestTimeout)
			setIfChanged("base-price", "base_price_per_call", basePrice)
			setIfChanged("price-per-100ms", "price_per_100ms", pricePer100ms)
			if cmd.Flags().Changed("active") {
				parsed, err := parseBoolArg("active", active)
				if err != nil {
					return err
				}
				payload["active"] = parsed
			}
			if cmd.Flags().Changed("allowed-host-id") {
				parsed, err := parseInt64Slice(allowedHostIDs)
				if err != nil {
					return err
				}
				payload["allowed_host_ids"] = parsed
			}
			if len(payload) == 0 {
				return fmt.Errorf("no endpoint fields were provided")
			}
			ctx := context.Background()
			var response map[string]any
			if err := postJSON(ctx, "/update/auth/serverless/endpoints", payload, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, orderedDisplayKeyValueRows(response, "id", "name", "slug", "auth_mode", "active", "hot_service_id", "warm_service_id", "invoke_path"))
		},
	}
	cmd.Flags().StringVar(&filePath, "file", "", "path to a JSON endpoint payload")
	cmd.Flags().StringVar(&name, "name", "", "endpoint name")
	cmd.Flags().StringVar(&slug, "slug", "", "public endpoint slug")
	cmd.Flags().StringVar(&description, "description", "", "endpoint description")
	cmd.Flags().Int64Var(&hotServiceID, "hot-service-id", 0, "cluster service ID used for hot path routing")
	cmd.Flags().Int64Var(&warmServiceID, "warm-service-id", 0, "optional warm service ID")
	cmd.Flags().StringVar(&authMode, "auth-mode", "", "auth mode: public or token")
	cmd.Flags().StringVar(&authToken, "auth-token", "", "token required when auth mode is token")
	cmd.Flags().IntVar(&requestTimeout, "request-timeout", 0, "request timeout in seconds")
	cmd.Flags().Float64Var(&basePrice, "base-price", 0, "base billed amount per call")
	cmd.Flags().Float64Var(&pricePer100ms, "price-per-100ms", 0, "billed amount per 100ms")
	cmd.Flags().StringVar(&active, "active", "", "set endpoint active state: true or false")
	cmd.Flags().StringSliceVar(&allowedHostIDs, "allowed-host-id", nil, "repeatable allowed host machine IDs")
	return cmd
}

func newServerlessUpdateCmd() *cobra.Command {
	var endpointID int64
	var filePath string
	var name string
	var slug string
	var description string
	var hotServiceID int64
	var warmServiceID int64
	var clearWarm bool
	var authMode string
	var authToken string
	var requestTimeout int
	var basePrice float64
	var pricePer100ms float64
	var active string
	var allowedHostIDs []string
	cmd := &cobra.Command{
		Use:     "update",
		Short:   "Update a serverless endpoint from JSON or flags",
		Example: "  quickpod serverless update --id 9 --file ./endpoint-update.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if endpointID <= 0 {
				return fmt.Errorf("--id is required")
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
			setIfChanged("slug", "slug", slug)
			setIfChanged("description", "description", description)
			setIfChanged("hot-service-id", "hot_service_id", hotServiceID)
			setIfChanged("warm-service-id", "warm_service_id", warmServiceID)
			setIfChanged("clear-warm-service", "clear_warm_service", clearWarm)
			setIfChanged("auth-mode", "auth_mode", authMode)
			setIfChanged("auth-token", "auth_token", authToken)
			setIfChanged("request-timeout", "request_timeout_seconds", requestTimeout)
			setIfChanged("base-price", "base_price_per_call", basePrice)
			setIfChanged("price-per-100ms", "price_per_100ms", pricePer100ms)
			if cmd.Flags().Changed("active") {
				parsed, err := parseBoolArg("active", active)
				if err != nil {
					return err
				}
				payload["active"] = parsed
			}
			if cmd.Flags().Changed("allowed-host-id") {
				parsed, err := parseInt64Slice(allowedHostIDs)
				if err != nil {
					return err
				}
				payload["allowed_host_ids"] = parsed
			}
			if len(payload) == 0 {
				return fmt.Errorf("no endpoint fields were provided")
			}
			ctx := context.Background()
			var response map[string]any
			if err := putJSON(ctx, fmt.Sprintf("/update/auth/serverless/endpoints/%d", endpointID), payload, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, orderedDisplayKeyValueRows(response, "id", "name", "slug", "auth_mode", "active", "hot_service_id", "warm_service_id", "invoke_path"))
		},
	}
	cmd.Flags().Int64Var(&endpointID, "id", 0, "serverless endpoint ID")
	cmd.Flags().StringVar(&filePath, "file", "", "path to a JSON endpoint update payload")
	cmd.Flags().StringVar(&name, "name", "", "endpoint name")
	cmd.Flags().StringVar(&slug, "slug", "", "public endpoint slug")
	cmd.Flags().StringVar(&description, "description", "", "endpoint description")
	cmd.Flags().Int64Var(&hotServiceID, "hot-service-id", 0, "cluster service ID used for hot path routing")
	cmd.Flags().Int64Var(&warmServiceID, "warm-service-id", 0, "optional warm service ID")
	cmd.Flags().BoolVar(&clearWarm, "clear-warm-service", false, "clear the warm service binding")
	cmd.Flags().StringVar(&authMode, "auth-mode", "", "auth mode: public or token")
	cmd.Flags().StringVar(&authToken, "auth-token", "", "token required when auth mode is token")
	cmd.Flags().IntVar(&requestTimeout, "request-timeout", 0, "request timeout in seconds")
	cmd.Flags().Float64Var(&basePrice, "base-price", 0, "base billed amount per call")
	cmd.Flags().Float64Var(&pricePer100ms, "price-per-100ms", 0, "billed amount per 100ms")
	cmd.Flags().StringVar(&active, "active", "", "set endpoint active state: true or false")
	cmd.Flags().StringSliceVar(&allowedHostIDs, "allowed-host-id", nil, "repeatable allowed host machine IDs")
	return cmd
}

func newServerlessDeleteCmd() *cobra.Command {
	var endpointID int64
	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "Delete a serverless endpoint",
		Example: "  quickpod serverless delete --id 9",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if endpointID <= 0 {
				return fmt.Errorf("--id is required")
			}
			ctx := context.Background()
			var response map[string]any
			if err := deleteJSON(ctx, fmt.Sprintf("/update/auth/serverless/endpoints/%d", endpointID), url.Values{}, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, orderedDisplayKeyValueRows(response, "result", "endpoint_id"))
		},
	}
	cmd.Flags().Int64Var(&endpointID, "id", 0, "serverless endpoint ID")
	return cmd
}

func newServerlessLogsCmd() *cobra.Command {
	var endpointID int64
	var limit int
	cmd := &cobra.Command{
		Use:     "logs",
		Short:   "List recent request logs for one serverless endpoint",
		Example: "  quickpod serverless logs --id 9 --limit 25",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if endpointID <= 0 {
				return fmt.Errorf("--id is required")
			}
			ctx := context.Background()
			query := url.Values{}
			if limit > 0 {
				query.Set("limit", fmt.Sprintf("%d", limit))
			}
			var items []map[string]any
			if err := getJSONQuery(ctx, fmt.Sprintf("/update/auth/serverless/endpoints/%d/logs", endpointID), query, true, &items); err != nil {
				return err
			}
			return renderTableOrJSON(items, []string{"ID", "METHOD", "PATH", "STATUS", "DURATION_MS", "BILLED", "CREATED"}, genericRows(items, "id", "method", "request_path", "status_code", "duration_ms", "billed_amount", "created_at"))
		},
	}
	cmd.Flags().Int64Var(&endpointID, "id", 0, "serverless endpoint ID")
	cmd.Flags().IntVar(&limit, "limit", 50, "maximum number of logs to fetch")
	return cmd
}
