package ccnf

type CER struct {
	EventID      string         `json:"event_id"`
	EventVersion int            `json:"event_version"`
	CCNFVersion  int            `json:"ccnf_version"`
	System       string         `json:"system"`
	Domain       string         `json:"domain"`
	Timestamp    int64          `json:"timestamp"`
	Actor        map[string]any `json:"actor"`
	Intent       map[string]any `json:"intent"`
	Identity     Identity       `json:"identity"`
	Causality    map[string]any `json:"causality"`
	ArtifactRefs []string       `json:"artifact_refs"`
	StateDelta   []StateDelta   `json:"state_delta"`
	Payload      map[string]any `json:"payload"`
	Compression  map[string]any `json:"compression"`
	Signature    map[string]any `json:"signature"`
}

type Identity struct {
	EntityKey   string   `json:"entity_key"`
	Type        string   `json:"type"`
	Scope       string   `json:"scope"`
	CollapseKey *string  `json:"collapse_key"`
	AliasKeys   []string `json:"alias_keys"`
}

type StateDelta struct {
	ArtifactID string         `json:"artifact_id"`
	BeforeHash *string        `json:"before_hash"`
	AfterHash  string         `json:"after_hash"`
	Patch      map[string]any `json:"patch"`
}

type Vector struct {
	Name             string            `json:"name"`
	CCNFVersion      int               `json:"ccnf_version"`
	InvariantsTested []string          `json:"invariants_tested"`
	Description      string            `json:"description"`
	Input            map[string]any    `json:"input"`
	Expected         VectorExpected    `json:"expected"`
}

type VectorExpected struct {
	EntityKey     *string        `json:"entity_key,omitempty"`
	CanonicalHash *string        `json:"canonical_hash,omitempty"`
	CER           *CER           `json:"cer,omitempty"`
	Error         *string        `json:"error,omitempty"`
}

type CCNFError string

func (e CCNFError) Error() string { return string(e) }

const (
	ErrIntentNormalization  CCNFError = "INTENT_NORMALIZATION_FAILURE"
	ErrArtifactResolution   CCNFError = "ARTIFACT_RESOLUTION_FAILURE"
	ErrDeltaScopeViolation  CCNFError = "DELTA_SCOPE_VIOLATION"
	ErrCCNFVersionMismatch  CCNFError = "CCNF_VERSION_MISMATCH"
	ErrValidation           CCNFError = "STRUCTURAL_PARSE_FAILURE"
)

const CurrentCCNFVersion = 1
const CurrentEventVersion = 1
const SystemName = "nexus"
