package ccnf

import (
	"crypto/sha256"
	"fmt"
	"sort"
)

func DeriveIdentity(m map[string]any) (string, string, string, error) {
	domain := getString(m, "domain")
	scope := domainToScope(domain)

	intent := getMap(m, "intent")

	if getString(intent, "action") == "" {
		return "", "", "", fmt.Errorf("cannot derive identity: no action in intent")
	}

	fields := map[string]any{
		"domain":      domain,
		"intent":      intent,
		"actor":       getMap(m, "actor"),
		"scope":       scope,
	}

	entityKey := hashEntitySignature(fields)

	return entityKey, "event", scope, nil
}

func DeriveCollapseKey(m map[string]any) *string {
	intent := getMap(m, "intent")
	targetType := getString(intent, "target_type")
	targetID := getString(intent, "target_id")

	if targetType == "" || targetID == "" {
		return nil
	}

	ck := fmt.Sprintf("%s:%s", targetType, targetID)
	return &ck
}

func DeriveAliasKeys(m map[string]any) []string {
	return nil
}

func hashEntitySignature(fields map[string]any) string {
	h := sha256.New()

	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte{0})
		h.Write(CanonicalJSON(fields[k]))
		h.Write([]byte{0})
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

func domainToScope(domain string) string {
	switch domain {
	case "execution":
		return "executiongraph.v2"
	case "specification":
		return "specification.v1"
	case "system":
		return "system.v1"
	default:
		return domain + ".v1"
	}
}

func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func getMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok {
		return nil
	}
	mm, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return mm
}

func getFloat(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok {
		return 0
	}
	f, ok := v.(float64)
	if !ok {
		return 0
	}
	return f
}
