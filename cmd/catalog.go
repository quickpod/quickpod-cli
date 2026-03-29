package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

func newCatalogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "Browse public QuickPod catalog data",
	}

	cmd.AddCommand(newSimpleListCmd("gpu-types", "/availablegputypes", []string{"TYPE"}, func(items []map[string]any) [][]string {
		return flattenTypes(items)
	}))
	cmd.AddCommand(newSimpleListCmd("cpu-types", "/availablecputypes", []string{"TYPE"}, func(items []map[string]any) [][]string {
		return flattenTypes(items)
	}))
	cmd.AddCommand(newSimpleListCmd("gpu-pricing", "/gpupricing", []string{"GPU", "MIN", "AVG", "MAX"}, func(items []map[string]any) [][]string {
		return genericRows(items, "gpu_type", "min_hourly_cost", "avg_hourly_cost", "max_hourly_cost")
	}))
	cmd.AddCommand(newSimpleListCmd("gpu-distribution", "/gpu_distribution", []string{"GPU", "COUNT"}, func(items []map[string]any) [][]string {
		return genericRows(items, "gpu_type", "count")
	}))
	cmd.AddCommand(newSimpleListCmd("machine-locations", "/machine_location", []string{"ID", "LATITUDE", "LONGITUDE"}, func(items []map[string]any) [][]string {
		return genericRows(items, "id", "latitude", "longitude")
	}))
	cmd.AddCommand(newSimpleListCmd("host-stores", "/update/host_stores", []string{"ID", "STORE", "SLUG", "USER"}, func(items []map[string]any) [][]string {
		return genericRows(items, "id", "store_name", "slug", "user_id")
	}))

	return cmd
}

func newSimpleListCmd(use string, endpoint string, headers []string, buildRows func([]map[string]any) [][]string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: "Fetch and print catalog data",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			var items []map[string]any
			if err := getJSON(ctx, endpoint, false, &items); err != nil {
				return err
			}
			return renderTableOrJSON(items, headers, buildRows(items))
		},
	}
	return cmd
}