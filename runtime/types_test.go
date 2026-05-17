package runtime

import (
	"testing"
)

func TestValidateRequest_EmptyRequestID(t *testing.T) {
	req := &ExecutionRequest{
		RequestID: "",
		Timestamp: 1000,
		Version:   VersionTriple{CCNF: 1, CollapseEngine: 1, Rehydration: 1},
	}
	if err := ValidateRequest(req); err != ErrEmptyRequestID {
		t.Fatalf("expected ErrEmptyRequestID, got %v", err)
	}
}

func TestValidateRequest_ZeroTimestamp(t *testing.T) {
	req := &ExecutionRequest{
		RequestID: "test-id",
		Timestamp: 0,
		Version:   VersionTriple{CCNF: 1, CollapseEngine: 1, Rehydration: 1},
	}
	if err := ValidateRequest(req); err != ErrZeroTimestamp {
		t.Fatalf("expected ErrZeroTimestamp, got %v", err)
	}
}

func TestValidateRequest_InvalidVersion(t *testing.T) {
	tests := []struct {
		name string
		v    VersionTriple
	}{
		{"zero ccnf", VersionTriple{CCNF: 0, CollapseEngine: 1, Rehydration: 1}},
		{"zero collapse", VersionTriple{CCNF: 1, CollapseEngine: 0, Rehydration: 1}},
		{"zero rehydration", VersionTriple{CCNF: 1, CollapseEngine: 1, Rehydration: 0}},
		{"all zero", VersionTriple{CCNF: 0, CollapseEngine: 0, Rehydration: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ExecutionRequest{
				RequestID: "test-id",
				Timestamp: 1000,
				Version:   tt.v,
			}
			if err := ValidateRequest(req); err != ErrInvalidVersion {
				t.Fatalf("expected ErrInvalidVersion, got %v", err)
			}
		})
	}
}

func TestValidateRequest_Valid(t *testing.T) {
	req := &ExecutionRequest{
		RequestID: "test-id",
		Timestamp: 1000,
		Version:   VersionTriple{CCNF: 1, CollapseEngine: 1, Rehydration: 1},
		Source:    "test",
	}
	if err := ValidateRequest(req); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestBuildReceipt_Success(t *testing.T) {
	req := &ExecutionRequest{
		RequestID: "req-1",
		Timestamp: 1000,
		Version:   VersionTriple{CCNF: 1, CollapseEngine: 1, Rehydration: 1},
	}
	timing := Timing{StartedAt: 1000, CompletedAt: 1010, DurationMs: 10}
	rec := BuildReceipt(req, StatusSuccess, "abc123", "abc123", nil, timing, 1)

	if rec.RequestID != "req-1" {
		t.Fatalf("expected req-1, got %s", rec.RequestID)
	}
	if rec.Status != StatusSuccess {
		t.Fatalf("expected SUCCESS, got %s", rec.Status)
	}
	if rec.Failure != nil {
		t.Fatalf("expected nil failure, got %v", rec.Failure)
	}
	if rec.CCNFHash != "abc123" {
		t.Fatalf("expected abc123, got %s", rec.CCNFHash)
	}
	if rec.CCNFVersion != 1 {
		t.Fatalf("expected version 1, got %d", rec.CCNFVersion)
	}
}

func TestBuildReceipt_Failure(t *testing.T) {
	req := &ExecutionRequest{
		RequestID: "req-2",
		Timestamp: 2000,
		Version:   VersionTriple{CCNF: 1, CollapseEngine: 1, Rehydration: 1},
	}
	timing := Timing{StartedAt: 2000, CompletedAt: 2005, DurationMs: 5}
	failure := &FailureNode{
		Code:    "INTENT_NORMALIZATION_FAILURE",
		Message: "unknown intent action",
	}
	rec := BuildReceipt(req, StatusFailure, "", "", failure, timing, 1)

	if rec.Status != StatusFailure {
		t.Fatalf("expected FAILURE, got %s", rec.Status)
	}
	if rec.Failure == nil {
		t.Fatal("expected non-nil failure")
	}
	if rec.Failure.Code != "INTENT_NORMALIZATION_FAILURE" {
		t.Fatalf("expected INTENT_NORMALIZATION_FAILURE, got %s", rec.Failure.Code)
	}
	if rec.Failure.Cause != nil {
		t.Fatal("expected nil cause")
	}
}

func TestBuildReceipt_Partial(t *testing.T) {
	req := &ExecutionRequest{
		RequestID: "req-3",
		Timestamp: 3000,
		Version:   VersionTriple{CCNF: 1, CollapseEngine: 1, Rehydration: 1},
	}
	timing := Timing{StartedAt: 3000, CompletedAt: 3010, DurationMs: 10}
	failure := &FailureNode{
		Code:    "DOWNSTREAM_CONSISTENCY_FAILURE",
		Message: "snapshot mismatch after fold",
	}
	rec := BuildReceipt(req, StatusPartial, "def456", "def456", failure, timing, 1)

	if rec.Status != StatusPartial {
		t.Fatalf("expected PARTIAL, got %s", rec.Status)
	}
	if rec.Failure == nil {
		t.Fatal("expected non-nil failure")
	}
	if rec.CCNFHash != "def456" {
		t.Fatalf("expected def456, got %s", rec.CCNFHash)
	}
	if rec.CERRootHash != "def456" {
		t.Fatalf("expected def456, got %s", rec.CERRootHash)
	}
}

func TestValidateReceipt_ValidSuccess(t *testing.T) {
	rec := &ExecutionReceipt{
		RequestID:   "r1",
		CCNFHash:    "aabb",
		CERRootHash: "aabb",
		Status:      StatusSuccess,
		Timing:      Timing{DurationMs: 10},
		CCNFVersion: 1,
	}
	if err := ValidateReceipt(rec); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateReceipt_ValidFailure(t *testing.T) {
	rec := &ExecutionReceipt{
		RequestID: "r2",
		Status:    StatusFailure,
		Failure:   &FailureNode{Code: "ERR", Message: "test"},
		Timing:    Timing{DurationMs: 0},
	}
	if err := ValidateReceipt(rec); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateReceipt_ValidPartial(t *testing.T) {
	rec := &ExecutionReceipt{
		RequestID: "r3",
		CCNFHash:  "ccdd",
		Status:    StatusPartial,
		Failure:   &FailureNode{Code: "DOWNSTREAM_CONSISTENCY_FAILURE", Message: "mismatch"},
		Timing:    Timing{DurationMs: 50},
	}
	if err := ValidateReceipt(rec); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateReceipt_EmptyRequestID(t *testing.T) {
	rec := &ExecutionReceipt{
		RequestID: "",
		Status:    StatusSuccess,
		CCNFHash:  "aabb",
		Timing:    Timing{DurationMs: 10},
	}
	if err := ValidateReceipt(rec); err != ErrEmptyRequestID {
		t.Fatalf("expected ErrEmptyRequestID, got %v", err)
	}
}

func TestValidateReceipt_SuccessWithFailureNode(t *testing.T) {
	rec := &ExecutionReceipt{
		RequestID: "r4",
		Status:    StatusSuccess,
		Failure:   &FailureNode{Code: "ERR", Message: "should not be here"},
		CCNFHash:  "aabb",
		Timing:    Timing{DurationMs: 10},
	}
	if err := ValidateReceipt(rec); err != ErrNonNilFailureNode {
		t.Fatalf("expected ErrNonNilFailureNode, got %v", err)
	}
}

func TestValidateReceipt_FailureWithNilFailure(t *testing.T) {
	rec := &ExecutionReceipt{
		RequestID: "r5",
		Status:    StatusFailure,
		Timing:    Timing{DurationMs: 10},
	}
	if err := ValidateReceipt(rec); err != ErrNilFailureNode {
		t.Fatalf("expected ErrNilFailureNode, got %v", err)
	}
}

func TestValidateReceipt_NegativeDuration(t *testing.T) {
	rec := &ExecutionReceipt{
		RequestID: "r6",
		Status:    StatusSuccess,
		CCNFHash:  "aabb",
		Timing:    Timing{DurationMs: -1},
	}
	if err := ValidateReceipt(rec); err != ErrNegativeDuration {
		t.Fatalf("expected ErrNegativeDuration, got %v", err)
	}
}

func TestValidateReceipt_SuccessEmptyHash(t *testing.T) {
	rec := &ExecutionReceipt{
		RequestID: "r7",
		Status:    StatusSuccess,
		CCNFHash:  "",
		Timing:    Timing{DurationMs: 10},
	}
	if err := ValidateReceipt(rec); err != ErrEmptyCCNFHash {
		t.Fatalf("expected ErrEmptyCCNFHash, got %v", err)
	}
}

func TestFailureNode_CauseChain(t *testing.T) {
	root := &FailureNode{
		Code:    "TOP_FAILURE",
		Message: "pipeline aborted",
		Cause: &FailureNode{
			Code:    "INTENT_NORMALIZATION_FAILURE",
			Message: "unknown intent",
			Cause: &FailureNode{
				Code:    "STRUCTURAL_PARSE_FAILURE",
				Message: "missing required field: intent.action",
			},
		},
	}

	if root.Cause.Cause.Code != "STRUCTURAL_PARSE_FAILURE" {
		t.Fatalf("expected STRUCTURAL_PARSE_FAILURE at depth 2, got %s", root.Cause.Cause.Code)
	}
}
