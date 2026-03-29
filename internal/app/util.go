package app

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

func StringValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		if math.Mod(typed, 1) == 0 {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func FloatValue(value any) float64 {
	switch typed := value.(type) {
	case nil:
		return 0
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		parsed, _ := typed.Float64()
		return parsed
	case string:
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed
	default:
		return 0
	}
}

func BoolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		parsed, _ := strconv.ParseBool(strings.TrimSpace(typed))
		return parsed
	default:
		return false
	}
}

func MapValue(item map[string]any, key string) map[string]any {
	value, ok := item[key]
	if !ok {
		return nil
	}
	result, ok := value.(map[string]any)
	if ok {
		return result
	}
	return nil
}

func SliceValue(item map[string]any, key string) []any {
	value, ok := item[key]
	if !ok {
		return nil
	}
	result, ok := value.([]any)
	if ok {
		return result
	}
	return nil
}

func ReadJSONFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func MergeMap(dst, src map[string]any) map[string]any {
	if dst == nil {
		dst = map[string]any{}
	}
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func Truncate(value string, length int) string {
	if length <= 0 || len(value) <= length {
		return value
	}
	if length <= 3 {
		return value[:length]
	}
	return value[:length-3] + "..."
}