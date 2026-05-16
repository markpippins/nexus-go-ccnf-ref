package ccnf

func BuildCERMap(cer *CER) map[string]any {
	m := map[string]any{
		"event_id":      cer.EventID,
		"event_version": cer.EventVersion,
		"ccnf_version":  cer.CCNFVersion,
		"system":        cer.System,
		"domain":        cer.Domain,
		"timestamp":     cer.Timestamp,
		"actor":         cer.Actor,
		"intent":        cer.Intent,
		"identity": map[string]any{
			"entity_key":   cer.Identity.EntityKey,
			"type":         cer.Identity.Type,
			"scope":        cer.Identity.Scope,
			"collapse_key": cer.Identity.CollapseKey,
			"alias_keys":   cer.Identity.AliasKeys,
		},
		"causality":     cer.Causality,
		"artifact_refs": cer.ArtifactRefs,
		"state_delta":   marshalStateDelta(cer.StateDelta),
		"payload":       cer.Payload,
		"compression":   cer.Compression,
		"signature":     cer.Signature,
	}

	return m
}

func marshalStateDelta(deltas []StateDelta) []map[string]any {
	result := make([]map[string]any, len(deltas))
	for i, d := range deltas {
		entry := map[string]any{
			"artifact_id": d.ArtifactID,
			"before_hash": d.BeforeHash,
			"after_hash":  d.AfterHash,
			"patch":       d.Patch,
		}
		result[i] = entry
	}
	return result
}
