package ccnf

import (
	"encoding/json"
	"fmt"
)

func SerializeCER(cer *CER) []byte {
	return CanonicalJSON(BuildCERMap(cer))
}

func ParseCER(raw []byte) (*CER, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("cer parse: invalid json: %w", err)
	}

	eventID, _ := m["event_id"].(string)
	eventVersion := toInt(m["event_version"])
	ccnfVersion := toInt(m["ccnf_version"])
	system, _ := m["system"].(string)
	domain, _ := m["domain"].(string)
	timestamp := toInt64(m["timestamp"])

	actor, _ := m["actor"].(map[string]any)
	intent, _ := m["intent"].(map[string]any)
	causality, _ := m["causality"].(map[string]any)
	payload, _ := m["payload"].(map[string]any)
	compression, _ := m["compression"].(map[string]any)
	signature, _ := m["signature"].(map[string]any)

	artifactRefs := toStringSlice(m["artifact_refs"])

	identity, err := parseIdentity(m["identity"])
	if err != nil {
		return nil, fmt.Errorf("cer parse: identity: %w", err)
	}

	stateDelta, err := parseStateDelta(m["state_delta"])
	if err != nil {
		return nil, fmt.Errorf("cer parse: state_delta: %w", err)
	}

	return &CER{
		EventID:      eventID,
		EventVersion: eventVersion,
		CCNFVersion:  ccnfVersion,
		System:       system,
		Domain:       domain,
		Timestamp:    timestamp,
		Actor:        actor,
		Intent:       intent,
		Identity:     identity,
		Causality:    causality,
		ArtifactRefs: artifactRefs,
		StateDelta:   stateDelta,
		Payload:      payload,
		Compression:  compression,
		Signature:    signature,
	}, nil
}

func parseIdentity(v any) (Identity, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return Identity{}, fmt.Errorf("expected object, got %T", v)
	}

	entityKey, _ := m["entity_key"].(string)
	typ, _ := m["type"].(string)
	scope, _ := m["scope"].(string)

	var collapseKey *string
	if ck, ok := m["collapse_key"].(string); ok {
		collapseKey = &ck
	}

	aliasKeys := toStringSlice(m["alias_keys"])

	return Identity{
		EntityKey:   entityKey,
		Type:        typ,
		Scope:       scope,
		CollapseKey: collapseKey,
		AliasKeys:   aliasKeys,
	}, nil
}

func parseStateDelta(v any) ([]StateDelta, error) {
	list, ok := v.([]any)
	if !ok {
		if v == nil {
			return nil, nil
		}
		return nil, fmt.Errorf("expected array, got %T", v)
	}

	deltas := make([]StateDelta, 0, len(list))
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("state_delta[%d]: expected object, got %T", i, item)
		}

		artifactID, _ := m["artifact_id"].(string)
		afterHash, _ := m["after_hash"].(string)

		var beforeHash *string
		if bh, ok := m["before_hash"].(string); ok {
			beforeHash = &bh
		}

		patch, _ := m["patch"].(map[string]any)

		deltas = append(deltas, StateDelta{
			ArtifactID: artifactID,
			BeforeHash: beforeHash,
			AfterHash:  afterHash,
			Patch:      patch,
		})
	}

	return deltas, nil
}

func toStringSlice(v any) []string {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int:
		return int64(n)
	case int64:
		return n
	case json.Number:
		i, _ := n.Int64()
		return i
	default:
		return 0
	}
}
