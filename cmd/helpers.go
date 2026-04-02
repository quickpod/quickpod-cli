package cmd

import (
	"context"
	"encoding/json"
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
		return errors.New("no auth credential configured; run 'quickpod auth login', use 'quickpod auth set-token', or set QUICKPOD_TOKEN/QUICKPOD_API_KEY")
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

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return strings.TrimSpace(value)
}

func formatLocation(item map[string]any) string {
	return valueOrDash(firstNonEmpty(app.StringValue(item["geolocation"]), app.StringValue(item["geoinfo"]), app.StringValue(item["location"])))
}

func formatPortRange(item map[string]any) string {
	start := app.StringValue(item["open_port_start"])
	end := app.StringValue(item["open_port_end"])
	if start == "" && end == "" {
		return "-"
	}
	if start == end || end == "" {
		return start
	}
	if start == "" {
		return end
	}
	return start + "-" + end
}

func formatAccess(item map[string]any) string {
	ip := app.StringValue(item["public_ipaddr"])
	ports := formatPortRange(item)
	if ip == "" {
		return ports
	}
	if ports == "-" || ports == "" {
		return ip
	}
	return ip + ":" + ports
}

func orderedKeyValueRows(values map[string]any, preferredKeys ...string) [][]string {
	rows := make([][]string, 0, len(values))
	seen := make(map[string]struct{}, len(preferredKeys))
	for _, key := range preferredKeys {
		if _, ok := values[key]; !ok {
			continue
		}
		rows = append(rows, []string{key, valueOrDash(app.StringValue(values[key]))})
		seen[key] = struct{}{}
	}
	remaining := make([]string, 0, len(values))
	for key := range values {
		if _, ok := seen[key]; ok {
			continue
		}
		remaining = append(remaining, key)
	}
	sort.Strings(remaining)
	for _, key := range remaining {
		rows = append(rows, []string{key, valueOrDash(app.StringValue(values[key]))})
	}
	return rows
}

func displayValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case map[string]any, []any, []map[string]any:
		payload, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%v", typed)
		}
		return string(payload)
	default:
		return app.StringValue(value)
	}
}

func orderedDisplayKeyValueRows(values map[string]any, preferredKeys ...string) [][]string {
	rows := make([][]string, 0, len(values))
	seen := make(map[string]struct{}, len(preferredKeys))
	for _, key := range preferredKeys {
		if _, ok := values[key]; !ok {
			continue
		}
		rows = append(rows, []string{key, valueOrDash(displayValue(values[key]))})
		seen[key] = struct{}{}
	}
	remaining := make([]string, 0, len(values))
	for key := range values {
		if _, ok := seen[key]; ok {
			continue
		}
		remaining = append(remaining, key)
	}
	sort.Strings(remaining)
	for _, key := range remaining {
		rows = append(rows, []string{key, valueOrDash(displayValue(values[key]))})
	}
	return rows
}

func findMapByAny(items []map[string]any, target string, keys ...string) (map[string]any, bool) {
	wanted := strings.TrimSpace(target)
	if wanted == "" {
		return nil, false
	}
	for _, item := range items {
		for _, key := range keys {
			if strings.EqualFold(strings.TrimSpace(app.StringValue(item[key])), wanted) {
				return item, true
			}
		}
	}
	return nil, false
}

func parseIntSlice(values []string) ([]int, error) {
	result := make([]int, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return nil, fmt.Errorf("invalid integer value %q", value)
		}
		result = append(result, parsed)
	}
	return result, nil
}

func parseInt64Slice(values []string) ([]int64, error) {
	result := make([]int64, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		parsed, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid integer value %q", value)
		}
		result = append(result, parsed)
	}
	return result, nil
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
			app.StringValue(item["machines_id"]),
			formatPortRange(item),
			app.Truncate(formatLocation(item), 22),
		})
	}
	return rows
}

func genericRows(items []map[string]any, columns ...string) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row := make([]string, 0, len(columns))
		for _, column := range columns {
			row = append(row, displayValue(item[column]))
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
