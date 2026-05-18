package runtime

import "fmt"

var (
	ErrEmptyRequestID         = fmt.Errorf("INVALID_REQUEST: request_id must be non-empty")
	ErrZeroTimestamp          = fmt.Errorf("INVALID_REQUEST: timestamp must be > 0")
	ErrInvalidVersion         = fmt.Errorf("INVALID_REQUEST: version triple components must all be > 0")
	ErrEmptyCCNFHash          = fmt.Errorf("INVALID_RECEIPT: ccnf_hash must be non-empty for status %s/%s", StatusSuccess, StatusPartial)
	ErrNilFailureNode         = fmt.Errorf("INVALID_RECEIPT: failure node must be non-nil for status %s/%s", StatusFailure, StatusPartial)
	ErrNonNilFailureNode      = fmt.Errorf("INVALID_RECEIPT: failure node must be nil for status %s", StatusSuccess)
	ErrNegativeDuration       = fmt.Errorf("INVALID_RECEIPT: duration_ms must be >= 0")
	ErrEmptyTraceRootHash     = fmt.Errorf("INVALID_RECEIPT: trace_root_hash must be non-empty for status %s/%s", StatusSuccess, StatusPartial)
	ErrZeroTraceEventCount    = fmt.Errorf("INVALID_RECEIPT: trace_event_count must be > 0 for status %s/%s", StatusSuccess, StatusPartial)
	ErrNonZeroTraceEventCount = fmt.Errorf("INVALID_RECEIPT: trace_event_count must be 0 for status %s", StatusFailure)
	ErrEmptyReplayBindingHash = fmt.Errorf("INVALID_RECEIPT: replay_binding_hash must be non-empty for status %s/%s", StatusSuccess, StatusPartial)
)

func ValidateRequest(req *ExecutionRequest) error {
	if req.RequestID == "" {
		return ErrEmptyRequestID
	}
	if req.Timestamp <= 0 {
		return ErrZeroTimestamp
	}
	if req.Version.CCNF <= 0 || req.Version.CollapseEngine <= 0 || req.Version.Rehydration <= 0 {
		return ErrInvalidVersion
	}
	return nil
}

func BuildReceipt(
	req *ExecutionRequest,
	status ExecutionStatus,
	ccnfHash string,
	trace *TraceBuilder,
	failure *FailureNode,
	timing Timing,
	ccnfVersion int,
) *ExecutionReceipt {
	traceRootHash := trace.RootHash()

	var replayBindingHash string
	if status == StatusSuccess || status == StatusPartial {
		replayBindingHash = ComputeReplayBinding(traceRootHash, traceRootHash, CurrentRuntimeSemanticsVersion, ccnfVersion)
	}

	return &ExecutionReceipt{
		RequestID:           req.RequestID,
		CCNFHash:            ccnfHash,
		CERRootHash:         traceRootHash,
		TraceRootHash:       traceRootHash,
		TraceEventCount:     trace.EventCount(),
		ReplayBindingHash:   replayBindingHash,
		Status:              status,
		Failure:             failure,
		Timing:              timing,
		CCNFVersion:         ccnfVersion,
	}
}

func ValidateReceipt(rec *ExecutionReceipt) error {
	if rec.RequestID == "" {
		return ErrEmptyRequestID
	}
	if rec.Timing.DurationMs < 0 {
		return ErrNegativeDuration
	}
	switch rec.Status {
	case StatusSuccess:
		if rec.Failure != nil {
			return ErrNonNilFailureNode
		}
		if rec.CCNFHash == "" {
			return ErrEmptyCCNFHash
		}
		if rec.TraceRootHash == "" {
			return ErrEmptyTraceRootHash
		}
		if rec.TraceEventCount == 0 {
			return ErrZeroTraceEventCount
		}
		if rec.ReplayBindingHash == "" {
			return ErrEmptyReplayBindingHash
		}
	case StatusFailure:
		if rec.Failure == nil {
			return ErrNilFailureNode
		}
		if rec.TraceEventCount != 0 {
			return ErrNonZeroTraceEventCount
		}
	case StatusPartial:
		if rec.Failure == nil {
			return ErrNilFailureNode
		}
		if rec.CCNFHash == "" {
			return ErrEmptyCCNFHash
		}
		if rec.TraceRootHash == "" {
			return ErrEmptyTraceRootHash
		}
		if rec.TraceEventCount == 0 {
			return ErrZeroTraceEventCount
		}
		if rec.ReplayBindingHash == "" {
			return ErrEmptyReplayBindingHash
		}
	}
	return nil
}
