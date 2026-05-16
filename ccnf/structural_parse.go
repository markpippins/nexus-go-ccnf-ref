package ccnf

import (
	"encoding/json"
	"fmt"
)

func StructuralParse(raw []byte, ccnfVersion int) (map[string]any, error) {
	if ccnfVersion != CurrentCCNFVersion {
		return nil, fmt.Errorf("%w: expected version %d, got %d", ErrCCNFVersionMismatch, CurrentCCNFVersion, ccnfVersion)
	}

	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON: %v", ErrValidation, err)
	}

	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: root must be a JSON object", ErrValidation)
	}

	required := []string{"actor", "intent", "domain", "event_id"}
	for _, field := range required {
		if _, exists := m[field]; !exists {
			return nil, fmt.Errorf("%w: missing required field %q", ErrValidation, field)
		}
	}

	return m, nil
}
