package ccnf

import (
	"sort"
	"strconv"
	"strings"
)

func CanonicalJSON(v any) []byte {
	return encode(v)
}

func encode(v any) []byte {
	switch val := v.(type) {
	case nil:
		return []byte("null")
	case bool:
		if val {
			return []byte("true")
		}
		return []byte("false")
	case int:
		return []byte(strconv.FormatInt(int64(val), 10))
	case int64:
		return []byte(strconv.FormatInt(val, 10))
	case int32:
		return []byte(strconv.FormatInt(int64(val), 10))
	case float64:
		return encodeFloat(val)
	case string:
		return encodeString(val)
	case []any:
		return encodeArray(val)
	case map[string]any:
		return encodeMap(val)
	default:
		return []byte("null")
	}
}

func encodeFloat(f float64) []byte {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if strings.ContainsAny(s, "eE") {
		return []byte("null")
	}
	if !strings.ContainsRune(s, '.') {
		return []byte(s)
	}
	return []byte(s)
}

func encodeString(s string) []byte {
	var b []byte
	b = append(b, '"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			b = append(b, '\\', '"')
		case c == '\\':
			b = append(b, '\\', '\\')
		case c == '\n':
			b = append(b, '\\', 'n')
		case c == '\r':
			b = append(b, '\\', 'r')
		case c == '\t':
			b = append(b, '\\', 't')
		case c < 0x20:
			b = append(b, []byte(`\u00`)...)
			b = append(b, hexChars[c>>4])
			b = append(b, hexChars[c&0x0f])
		default:
			b = append(b, c)
		}
	}
	b = append(b, '"')
	return b
}

var hexChars = []byte("0123456789abcdef")

func encodeArray(arr []any) []byte {
	if arr == nil {
		return []byte("null")
	}
	var b []byte
	b = append(b, '[')
	for i, elem := range arr {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, encode(elem)...)
	}
	b = append(b, ']')
	return b
}

func encodeMap(m map[string]any) []byte {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b []byte
	b = append(b, '{')
	for i, k := range keys {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, encodeString(k)...)
		b = append(b, ':')
		b = append(b, encode(m[k])...)
	}
	b = append(b, '}')
	return b
}
