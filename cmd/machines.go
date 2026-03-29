package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"quickpod-cli/internal/app"
)

func newMachinesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "machines",
		Short: "Inspect and manage your machines and contracts",
	}

	cmd.AddCommand(newMachinesListCmd())
	cmd.AddCommand(newMachinesContractsCmd())
	cmd.AddCommand(newMachinesUpdateGPUCmd())
	cmd.AddCommand(newMachinesUpdateCPUCmd())
	cmd.AddCommand(newMachinesPrivilegedCmd())

	return cmd
}

func newMachinesListCmd() *cobra.Command {
	var kind string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List your GPU or CPU machines",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			ctx := context.Background()
			endpoint := "/mymachines"
			if kind == "cpu" {
				endpoint = "/mymachines_cpu"
			}
			var items []map[string]any
			if err := getJSON(ctx, endpoint, true, &items); err != nil {
				return err
			}
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, []string{
					app.StringValue(item["id"]),
					app.Truncate(app.StringValue(item["hostname"]), 24),
					app.StringValue(item["machine_type"]),
					app.StringValue(item["num_gpus"]),
					app.StringValue(item["cpu_cores"]),
					boolString(app.BoolValue(item["listed"])),
					boolString(app.BoolValue(item["online"])),
					app.StringValue(item["public_ipaddr"]),
				})
			}
			return renderTableOrJSON(items, []string{"ID", "HOSTNAME", "TYPE", "GPUS", "CPUS", "LISTED", "ONLINE", "IP"}, rows)
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "gpu", "machine kind: gpu or cpu")
	return cmd
}

func newMachinesContractsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contracts",
		Short: "List your machine contracts",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			ctx := context.Background()
			var items []map[string]any
			if err := getJSON(ctx, "/mymachine_contracts", true, &items); err != nil {
				return err
			}
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, []string{
					app.StringValue(item["id"]),
					app.Truncate(app.StringValue(item["hostname"]), 24),
					app.StringValue(item["machine_type"]),
					app.StringValue(item["earn_hour"]),
					app.StringValue(item["earn_day"]),
					app.StringValue(item["current_rentals_resident"]),
				})
			}
			return renderTableOrJSON(items, []string{"ID", "HOSTNAME", "TYPE", "EARN/HOUR", "EARN/DAY", "ACTIVE"}, rows)
		},
	}
	return cmd
}

func newMachinesUpdateGPUCmd() *cobra.Command {
	var machineID int
	var listed string
	var minGPU string
	var maxDuration string
	var storageCost string
	var inetDownCost string
	var gpuPrices []string

	cmd := &cobra.Command{
		Use:   "update-gpu",
		Short: "Update a GPU machine listing",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if machineID <= 0 {
				return fmt.Errorf("--machine-id is required")
			}
			listedValue, err := parseBoolArg("listed", listed)
			if err != nil {
				return err
			}
			gpusListing, err := parseGPUPriceFlags(gpuPrices)
			if err != nil {
				return err
			}

			payload := map[string]any{
				"listed":                 listedValue,
				"min_gpu":                minGPU,
				"max_duration":           maxDuration,
				"listed_storage_cost":    storageCost,
				"listed_inet_down_cost":  inetDownCost,
				"gpus_listing":           gpusListing,
			}
			ctx := context.Background()
			var response map[string]any
			if err := putJSON(ctx, fmt.Sprintf("/update/listmachines/%d", machineID), payload, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"result", app.StringValue(response["result"])}})
		},
	}
	cmd.Flags().IntVar(&machineID, "machine-id", 0, "machine ID")
	cmd.Flags().StringVar(&listed, "listed", "", "set listing state: true or false")
	cmd.Flags().StringVar(&minGPU, "min-gpu", "0", "minimum GPU count for the listing")
	cmd.Flags().StringVar(&maxDuration, "max-duration", "0", "maximum duration in hours")
	cmd.Flags().StringVar(&storageCost, "storage-cost", "0", "listed storage cost")
	cmd.Flags().StringVar(&inetDownCost, "inet-down-cost", "0", "listed internet download cost")
	cmd.Flags().StringSliceVar(&gpuPrices, "gpu-price", nil, "repeatable gpu pricing entries in the form gpuID=price")
	return cmd
}

func newMachinesUpdateCPUCmd() *cobra.Command {
	var machineID int
	var listed string
	var maxDuration string
	var cpuCost string
	var storageCost string

	cmd := &cobra.Command{
		Use:   "update-cpu",
		Short: "Update a CPU machine listing",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if machineID <= 0 {
				return fmt.Errorf("--machine-id is required")
			}
			listedValue, err := parseBoolArg("listed", listed)
			if err != nil {
				return err
			}
			payload := map[string]any{
				"listed":               listedValue,
				"max_duration":         maxDuration,
				"listed_cpu_cost":      cpuCost,
				"listed_storage_cost":  storageCost,
				"gpus_listing":         []string{},
			}
			ctx := context.Background()
			var response map[string]any
			if err := putJSON(ctx, fmt.Sprintf("/update/listmachines_cpu/%d", machineID), payload, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"result", app.StringValue(response["result"])}})
		},
	}
	cmd.Flags().IntVar(&machineID, "machine-id", 0, "machine ID")
	cmd.Flags().StringVar(&listed, "listed", "", "set listing state: true or false")
	cmd.Flags().StringVar(&maxDuration, "max-duration", "0", "maximum duration in hours")
	cmd.Flags().StringVar(&cpuCost, "cpu-cost", "0", "listed CPU cost")
	cmd.Flags().StringVar(&storageCost, "storage-cost", "0", "listed storage cost")
	return cmd
}

func newMachinesPrivilegedCmd() *cobra.Command {
	var machineID int
	var allow string
	cmd := &cobra.Command{
		Use:   "privileged",
		Short: "Allow or block privileged pod access on a machine",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireAuth(); err != nil {
				return err
			}
			if machineID <= 0 {
				return fmt.Errorf("--machine-id is required")
			}
			allowValue, err := parseBoolArg("allow", allow)
			if err != nil {
				return err
			}
			ctx := context.Background()
			var response map[string]any
			if err := postJSON(ctx, "/update/machine_allow_privileged_access", map[string]any{"machine_id": machineID, "allow_priveleged": allowValue}, true, &response); err != nil {
				return err
			}
			return renderTableOrJSON(response, []string{"KEY", "VALUE"}, [][]string{{"result", firstNonEmpty(app.StringValue(response["result"]), "updated")}})
		},
	}
	cmd.Flags().IntVar(&machineID, "machine-id", 0, "machine ID")
	cmd.Flags().StringVar(&allow, "allow", "", "set privileged access: true or false")
	return cmd
}

func parseGPUPriceFlags(entries []string) ([]map[string]any, error) {
	list := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid gpu price entry %q; use gpuID=price", entry)
		}
		gpuID, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid gpu id in %q", entry)
		}
		list = append(list, map[string]any{
			"id":               gpuID,
			"listed_gpu_cost":  strings.TrimSpace(parts[1]),
		})
	}
	return list, nil
}