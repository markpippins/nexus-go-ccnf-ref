package ccnf

import (
	"fmt"
	"time"
)

func Run(raw []byte, ccnfVersion int) (*CER, error) {
	if ccnfVersion != CurrentCCNFVersion {
		return nil, fmt.Errorf("%w: expected version %d, got %d", ErrCCNFVersionMismatch, CurrentCCNFVersion, ccnfVersion)
	}

	m, err := StructuralParse(raw, ccnfVersion)
	if err != nil {
		return nil, err
	}

	if err := checkEmbeddedVersion(m, ccnfVersion); err != nil {
		return nil, err
	}

	m = CanonicalizeFields(m)

	normalizedIntent, err := NormalizeIntent(m)
	if err != nil {
		return nil, err
	}
	m["intent"] = normalizedIntent

	if err := checkTargetID(m); err != nil {
		return nil, err
	}

	artifactRefs, artifacts, err := ResolveArtifacts(m)
	if err != nil {
		return nil, err
	}

	entityKey, identityType, scope, err := DeriveIdentity(m)
	if err != nil {
		return nil, err
	}

	stateDeltas, err := ComputeStateDeltas(m, artifactRefs, artifacts)
	if err != nil {
		return nil, err
	}

	collapseKey := DeriveCollapseKey(m)
	aliasKeys := DeriveAliasKeys(m)

	domain := getString(m, "domain")
	eventID := getString(m, "event_id")

	timestamp, _ := parseTimestamp(m["timestamp"])
	if timestamp == 0 {
		timestamp = time.Now().Unix()
	}

	rawCausality := getMap(m, "causality")
	causality := normalizeCausality(rawCausality)

	payload := buildPayload(m)

	compression := map[string]any{
		"strategy":           "full",
		"lossless":           true,
		"compression_version": 1,
	}

	cer := &CER{
		EventID:      eventID,
		EventVersion: CurrentEventVersion,
		CCNFVersion:  ccnfVersion,
		System:       SystemName,
		Domain:       domain,
		Timestamp:    timestamp,
		Actor:        getMap(m, "actor"),
		Intent:       normalizedIntent,
		Identity: Identity{
			EntityKey:   entityKey,
			Type:        identityType,
			Scope:       scope,
			CollapseKey: collapseKey,
			AliasKeys:   aliasKeys,
		},
		Causality:    causality,
		ArtifactRefs: artifactRefs,
		StateDelta:   stateDeltas,
		Payload:      payload,
		Compression:  compression,
	}

	cer.Signature = BuildSignature(cer)

	return cer, nil
}

func normalizeCausality(c map[string]any) map[string]any {
	if c == nil {
		return map[string]any{
			"parent_event_ids": []any{},
			"causal_chain_id":  "",
			"trace_depth":      0,
			"ordered":          true,
		}
	}
	if _, ok := c["ordered"]; !ok {
		c["ordered"] = true
	}
	if _, ok := c["parent_event_ids"]; !ok {
		c["parent_event_ids"] = []any{}
	}
	return c
}

func checkEmbeddedVersion(m map[string]any, expectedVersion int) error {
	if v, ok := m["ccnf_version"].(float64); ok {
		if int(v) != expectedVersion {
			return fmt.Errorf("%w: input declares ccnf_version %d, engine is %d", ErrCCNFVersionMismatch, int(v), expectedVersion)
		}
	}
	return nil
}

func checkTargetID(m map[string]any) error {
	intent := getMap(m, "intent")
	if intent == nil {
		return nil
	}
	targetID := getString(intent, "target_id")
	if targetID == "" {
		return nil
	}
	if !isValidArtifactID(targetID) {
		return fmt.Errorf("%w: target_id %q is not a valid type:id reference", ErrArtifactResolution, targetID)
	}
	return nil
}

func buildPayload(m map[string]any) map[string]any {
	payload := map[string]any{
		"type": "structured",
		"data": map[string]any{},
	}

	rawPayload := getMap(m, "payload")
	if rawPayload != nil {
		if d, ok := rawPayload["data"]; ok {
			switch dd := d.(type) {
			case map[string]any:
				filteredData := make(map[string]any)
				for k, v := range dd {
					if !isValidArtifactID(k) {
						filteredData[k] = v
					}
				}
				payload["data"] = filteredData
			}
		}
	}

	return payload
}
