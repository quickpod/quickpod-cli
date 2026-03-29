package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"quickpod-cli/internal/app"
)

func requireAuth() error {
	if strings.TrimSpace(runtimeConfig.Token) == "" {
		return errors.New("no auth token configured; run 'quickpod auth login' or set QUICKPOD_TOKEN")
	}
	return nil
}

func renderTableOrJSON(data any, headers []string, rows [][]string) error {
	if runtimeConfig.Output == "json" {
		return app.PrintJSON(data)
	}
	app.PrintTable(headers, rows)
	return nil
}

func getJSON(ctx context.Context, path string, auth bool, out any) error {
	return apiClient.Get(ctx, path, nil, auth, out)
}

func getJSONQuery(ctx context.Context, path string, query url.Values, auth bool, out any) error {
	return apiClient.Get(ctx, path, query, auth, out)
}

func postJSON(ctx context.Context, path string, body any, auth bool, out any) error {
	return apiClient.Post(ctx, path, body, auth, out)
}

func putJSON(ctx context.Context, path string, body any, auth bool, out any) error {
	return apiClient.Put(ctx, path, body, auth, out)
}

func deleteJSON(ctx context.Context, path string, query url.Values, auth bool, out any) error {
	return apiClient.Delete(ctx, path, query, auth, out)
}

func parseBoolArg(name, value string) (bool, error) {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false, fmt.Errorf("invalid %s value %q; use true or false", name, value)
	}
	return parsed, nil
}

func boolString(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func flattenTypes(items []map[string]any) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{app.StringValue(item["gpu_type"])})
	}
	return rows
}

func sortMaps(items []map[string]any, key string, desc bool) {
	if strings.TrimSpace(key) == "" {
		return
	}
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i][key]
		right := items[j][key]

		leftString := strings.ToLower(app.StringValue(left))
		rightString := strings.ToLower(app.StringValue(right))
		leftFloat := app.FloatValue(left)
		rightFloat := app.FloatValue(right)

		useNumeric := leftFloat != 0 || rightFloat != 0 || leftString == "0" || rightString == "0"
		if useNumeric {
			if desc {
				return leftFloat > rightFloat
			}
			return leftFloat < rightFloat
		}

		if desc {
			return leftString > rightString
		}
		return leftString < rightString
	})
}

func filterOffers(items []map[string]any, kind string, typeFilter string, location string, minHourly, maxHourly float64, minCount, maxCount int, minReliability float64, verifiedOnly bool) []map[string]any {
	filtered := make([][]map[string]any, 0)
	_ = filtered
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		offerType := strings.ToLower(app.StringValue(item["gpu_type"]))
		if typeFilter != "" && !strings.Contains(offerType, strings.ToLower(typeFilter)) {
			continue
		}

		locationValue := strings.ToLower(app.StringValue(item["geolocation"]) + " " + app.StringValue(item["geoinfo"]))
		if location != "" && !strings.Contains(locationValue, strings.ToLower(location)) {
			continue
		}

		hourlyCost := app.FloatValue(item["hourly_cost"])
		if minHourly > 0 && hourlyCost < minHourly {
			continue
		}
		if maxHourly > 0 && hourlyCost > maxHourly {
			continue
		}

		countKey := "num_gpus"
		if kind == "cpu" {
			countKey = "cpus"
		}
		count := int(app.FloatValue(item[countKey]))
		if minCount > 0 && count < minCount {
			continue
		}
		if maxCount > 0 && count > maxCount {
			continue
		}

		reliability := app.FloatValue(item["reliability"])
		if minReliability > 0 && reliability < minReliability {
			continue
		}

		if verifiedOnly && !app.BoolValue(item["verification"]) {
			continue
		}

		result = append(result, item)
	}
	return result
}

func offerRows(items []map[string]any, kind string, limit int) [][]string {
	rows := make([][]string, 0, len(items))
	for index, item := range items {
		if limit > 0 && index >= limit {
			break
		}
		typeLabel := app.StringValue(item["gpu_type"])
		countLabel := app.StringValue(item["num_gpus"])
		if kind == "cpu" {
			if typeLabel == "" {
				typeLabel = app.StringValue(item["cpu_name"])
			}
			countLabel = app.StringValue(item["cpus"])
		}
		rows = append(rows, []string{
			app.StringValue(item["id"]),
			app.Truncate(app.StringValue(item["offer_name"]), 28),
			app.Truncate(typeLabel, 24),
			countLabel,
			app.StringValue(item["hourly_cost"]),
			app.StringValue(item["reliability"]),
			boolString(app.BoolValue(item["verification"])),
			app.Truncate(firstNonEmpty(app.StringValue(item["geolocation"]), app.StringValue(item["geoinfo"])), 22),
		})
	}
	return rows
}

func genericRows(items []map[string]any, columns ...string) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row := make([]string, 0, len(columns))
		for _, column := range columns {
			row = append(row, app.StringValue(item[column]))
		}
		rows = append(rows, row)
	}
	return rows
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}