package ccnf

import (
	"fmt"
	"strings"
)

func ResolveArtifacts(m map[string]any) ([]string, []map[string]any, error) {
	payload := getMap(m, "payload")
	data := getMap(payload, "data")

	var refs []string
	var artifacts []map[string]any

	if data == nil || len(data) == 0 {
		return refs, artifacts, nil
	}

	for k, v := range data {
		if !isValidArtifactID(k) {
			return nil, nil, fmt.Errorf("%w: invalid artifact id %q", ErrArtifactResolution, k)
		}
		refs = append(refs, k)

		patch, ok := v.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("%w: artifact %q value must be an object", ErrArtifactResolution, k)
		}

		artifacts = append(artifacts, map[string]any{
			"artifact_id": k,
			"patch":       patch,
		})
	}

	return refs, artifacts, nil
}

func isValidArtifactID(id string) bool {
	parts := strings.SplitN(id, ":", 3)
	if len(parts) < 2 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
	}
	return true
}
