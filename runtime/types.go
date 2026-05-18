package runtime

type ExecutionRequest struct {
	RequestID string        `json:"request_id"`
	Version   VersionTriple `json:"version"`
	Timestamp int64         `json:"timestamp"`
	Source    string        `json:"source"`
	Payload   map[string]any `json:"payload"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type VersionTriple struct {
	CCNF           int `json:"ccnf"`
	CollapseEngine int `json:"collapse_engine"`
	Rehydration    int `json:"rehydration"`
}

type ExecutionStatus string

const (
	StatusSuccess ExecutionStatus = "SUCCESS"
	StatusFailure ExecutionStatus = "FAILURE"
	StatusPartial ExecutionStatus = "PARTIAL"
)

type FailureNode struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Cause   *FailureNode `json:"cause,omitempty"`
}

type Timing struct {
	StartedAt   int64 `json:"started_at"`
	CompletedAt int64 `json:"completed_at"`
	DurationMs  int64 `json:"duration_ms"`
}

const CurrentRuntimeSemanticsVersion = 1

type ExecutionReceipt struct {
	RequestID           string          `json:"request_id"`
	CCNFHash            string          `json:"ccnf_hash"`
	CERRootHash         string          `json:"cer_root_hash"`
	TraceRootHash       string          `json:"trace_root_hash"`
	TraceEventCount     uint64          `json:"trace_event_count"`
	ReplayBindingHash   string          `json:"replay_binding_hash"`
	Status              ExecutionStatus `json:"status"`
	Failure             *FailureNode    `json:"failure,omitempty"`
	Timing              Timing          `json:"timing"`
	CCNFVersion         int             `json:"ccnf_version"`
}
