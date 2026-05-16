package ccnf

import (
	"crypto/sha256"
	"fmt"
)

func ComputeStateDeltas(m map[string]any, artifactRefs []string, artifacts []map[string]any) ([]StateDelta, error) {
	var deltas []StateDelta

	for i, ref := range artifactRefs {
		if i < len(artifacts) {
			patch := artifacts[i]["patch"].(map[string]any)

			beforeHash := computeBeforeHash(m)
			afterHash := computePatchHash(patch)

			delta := StateDelta{
				ArtifactID: ref,
				BeforeHash: beforeHash,
				AfterHash:  afterHash,
				Patch:      patch,
			}
			deltas = append(deltas, delta)
		}
	}

	return deltas, nil
}

func ValidateDeltaScope(deltas []StateDelta, m map[string]any) error {
	intent := getMap(m, "intent")
	targetID := getString(intent, "target_id")
	if targetID == "" {
		for _, d := range deltas {
			if !isValidArtifactID(d.ArtifactID) {
				return fmt.Errorf("%w: invalid artifact_id %q", ErrDeltaScopeViolation, d.ArtifactID)
			}
		}
		return nil
	}

	targetPrefix := extractArtifactPrefix(targetID)
	for _, d := range deltas {
		prefix := extractArtifactPrefix(d.ArtifactID)
		if prefix != targetPrefix {
			return fmt.Errorf("%w: artifact %q (prefix %q) outside scope of target %q (prefix %q)",
				ErrDeltaScopeViolation, d.ArtifactID, prefix, targetID, targetPrefix)
		}
	}
	return nil
}

func extractArtifactPrefix(id string) string {
	for i := 0; i < len(id); i++ {
		if id[i] == ':' {
			return id[:i]
		}
	}
	return id
}

func computeBeforeHash(m map[string]any) *string {
	return nil
}

func computePatchHash(patch map[string]any) string {
	h := sha256.New()
	h.Write(CanonicalJSON(patch))
	return fmt.Sprintf("%x", h.Sum(nil))
}
