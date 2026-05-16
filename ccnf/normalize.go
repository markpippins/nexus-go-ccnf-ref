package ccnf

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

func normalizeString(s string) string {
	s = strings.Map(func(r rune) rune {
		if r >= 0x200B && r <= 0x200D {
			return -1
		}
		if r == 0xFEFF {
			return -1
		}
		return r
	}, s)
	s = norm.NFC.String(s)
	return s
}

func normalizeStrings(v any) any {
	switch val := v.(type) {
	case string:
		return normalizeString(val)
	case map[string]any:
		m := make(map[string]any, len(val))
		for k, vv := range val {
			m[normalizeString(k)] = normalizeStrings(vv)
		}
		return m
	case []any:
		res := make([]any, len(val))
		for i, vv := range val {
			res[i] = normalizeStrings(vv)
		}
		return res
	default:
		return v
	}
}
