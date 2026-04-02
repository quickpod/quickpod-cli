package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "search",
		Short:   "Search rentable or occupied GPU and CPU offers",
		Example: "  quickpod search gpu --type A100 --max-hourly 3 --verified-only\n  quickpod search cpu --min-count 8 --max-hourly 0.50 --sort cpus --desc\n  quickpod search gpu --occupied --location frankfurt --limit 15",
	}

	cmd.AddCommand(newSearchKindCmd("gpu"))
	cmd.AddCommand(newSearchKindCmd("cpu"))
	return cmd
}

func newSearchKindCmd(kind string) *cobra.Command {
	var occupied bool
	var typeFilter string
	var location string
	var minHourly float64
	var maxHourly float64
	var minCount int
	var maxCount int
	var minReliability float64
	var verifiedOnly bool
	var limit int
	var sortBy string
	var desc bool

	cmd := &cobra.Command{
		Use:   kind,
		Short: fmt.Sprintf("Search %s offers", kind),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			endpoint := "/rentable"
			if kind == "cpu" {
				endpoint = "/rentable_cpu"
			}
			if occupied {
				endpoint = "/notrentable"
				if kind == "cpu" {
					endpoint = "/notrentable_cpu"
				}
			}

			var items []map[string]any
			if err := getJSON(ctx, endpoint, false, &items); err != nil {
				return err
			}

			items = filterOffers(items, kind, typeFilter, location, minHourly, maxHourly, minCount, maxCount, minReliability, verifiedOnly)
			sortMaps(items, sortBy, desc)

			return renderTableOrJSON(items, []string{"ID", "OFFER", "TYPE", "COUNT", "HOURLY", "RELIABILITY", "VERIFIED", "MACHINE", "PORTS", "LOCATION"}, offerRows(items, kind, limit))
		},
	}

	cmd.Flags().BoolVar(&occupied, "occupied", false, "show occupied offers instead of rentable offers")
	cmd.Flags().StringVar(&typeFilter, "type", "", "filter by GPU or CPU type label")
	cmd.Flags().StringVar(&location, "location", "", "filter by location or geoinfo substring")
	cmd.Flags().Float64Var(&minHourly, "min-hourly", 0, "minimum hourly cost")
	cmd.Flags().Float64Var(&maxHourly, "max-hourly", 0, "maximum hourly cost")
	cmd.Flags().IntVar(&minCount, "min-count", 0, "minimum GPU or CPU count")
	cmd.Flags().IntVar(&maxCount, "max-count", 0, "maximum GPU or CPU count")
	cmd.Flags().Float64Var(&minReliability, "min-reliability", 0, "minimum reliability score")
	cmd.Flags().BoolVar(&verifiedOnly, "verified-only", false, "show only verified hosts")
	cmd.Flags().IntVar(&limit, "limit", 25, "maximum rows to print; 0 prints all rows")
	cmd.Flags().StringVar(&sortBy, "sort", "hourly_cost", "sort key such as hourly_cost, reliability, num_gpus, cpus, offer_name")
	cmd.Flags().BoolVar(&desc, "desc", false, "sort descending")

	return cmd
}
