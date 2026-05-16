package ccnf

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"
)

func CanonicalizeFields(m map[string]any) map[string]any {
	m = normalizeStrings(m).(map[string]any)

	result := deepCopyMap(m)
	result = convertTimestamps(result)
	result = normalizeAbsentFields(result)

	return result
}

func deepCopyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case map[string]any:
			out[k] = deepCopyMap(val)
		case []any:
			out[k] = deepCopySlice(val)
		default:
			out[k] = v
		}
	}
	return out
}

func deepCopySlice(s []any) []any {
	out := make([]any, len(s))
	for i, v := range s {
		switch val := v.(type) {
		case map[string]any:
			out[i] = deepCopyMap(val)
		case []any:
			out[i] = deepCopySlice(val)
		default:
			out[i] = v
		}
	}
	return out
}

func convertTimestamps(m map[string]any) map[string]any {
	for k, v := range m {
		switch val := v.(type) {
		case string:
			if (k == "timestamp") {
				if t, err := time.Parse(time.RFC3339, val); err == nil {
					m[k] = t.Unix()
				}
			}
		case float64:
			if k == "timestamp" {
				m[k] = int64(val)
			}
		case map[string]any:
			convertTimestamps(val)
		case []any:
			for _, elem := range val {
				if em, ok := elem.(map[string]any); ok {
					convertTimestamps(em)
				}
			}
		}
	}
	return m
}

func normalizeAbsentFields(m map[string]any) map[string]any {
	for k, v := range m {
		switch val := v.(type) {
		case nil:
			if k == "external_id" || k == "collapse_key" || k == "signature" {
				continue
			}
		case float64:
			m[k] = normalizeNumber(val)
		case string:
			if val == "" && k == "collapse_key" {
				m[k] = nil
			}
		case map[string]any:
			normalizeAbsentFields(val)
		case []any:
			for _, elem := range val {
				if em, ok := elem.(map[string]any); ok {
					normalizeAbsentFields(em)
				}
			}
		}
	}
	return m
}

func normalizeNumber(f float64) any {
	if f == float64(int64(f)) {
		return int64(f)
	}
	return f
}

func parseTimestamp(v any) (int64, bool) {
	switch val := v.(type) {
	case float64:
		return int64(val), true
	case int64:
		return val, true
	case string:
		t, err := time.Parse(time.RFC3339, val)
		if err == nil {
			return t.Unix(), true
		}
		t, err = time.Parse("2006-01-02T15:04:05Z", val)
		if err == nil {
			return t.Unix(), true
		}
		return 0, false
	default:
		return 0, false
	}
}

type orderedMap struct {
	keys []string
	m    map[string]any
}

func sortMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func recursiveSortKeys(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			out[k] = recursiveSortKeys(vv)
		}
		return out
	case []any:
		res := make([]any, len(val))
		for i, vv := range val {
			res[i] = recursiveSortKeys(vv)
		}
		return res
	default:
		return v
	}
}

func toJSONString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func trimTrailing(s string) string {
	return strings.TrimRight(s, " \t\n\r")
}
