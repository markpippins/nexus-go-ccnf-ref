package ccnf

import "fmt"

var controlledVocab = map[string]bool{
	"create": true,
	"update": true,
	"delete": true,
	"execute": true,
	"validate": true,
	"emit": true,
}

func NormalizeIntent(m map[string]any) (map[string]any, error) {
	rawIntent, exists := m["intent"]
	if !exists {
		return nil, fmt.Errorf("%w: no intent field", ErrIntentNormalization)
	}

	switch v := rawIntent.(type) {
	case string:
		return nil, fmt.Errorf("%w: free-text intent %q cannot be mapped", ErrIntentNormalization, v)
	case map[string]any:
		action := getString(v, "action")
		if action == "" {
			return nil, fmt.Errorf("%w: empty action in intent", ErrIntentNormalization)
		}
		if !controlledVocab[action] {
			return nil, fmt.Errorf("%w: unknown action %q", ErrIntentNormalization, action)
		}

		normalized := map[string]any{
			"type":        "normalized_verb",
			"action":      action,
			"target_type": getString(v, "target_type"),
			"target_id":   getString(v, "target_id"),
		}
		return normalized, nil
	default:
		return nil, fmt.Errorf("%w: unexpected intent type %T", ErrIntentNormalization, rawIntent)
	}
}
