package replay

import "fmt"

var (
	ErrEmptyEventList   = fmt.Errorf("REPLAY_INPUT: event list must not be empty")
	ErrEventCountMismatch = fmt.Errorf("REPLAY_INPUT: event_count mismatch")
	ErrEmptyCerRootHash   = fmt.Errorf("REPLAY_INPUT: cer_root_hash must not be empty")
	ErrEmptyTraceRootHash = fmt.Errorf("REPLAY_INPUT: trace_root_hash must not be empty")
	ErrInvalidEventCount  = fmt.Errorf("REPLAY_INPUT: event_count must be > 0")
	ErrEventIDEmpty       = fmt.Errorf("REPLAY_INPUT: event EventID must not be empty")
)

func ValidateReplayInput(input ReplayInput) error {
	if len(input.Events) == 0 {
		return ErrEmptyEventList
	}
	if input.EventCount == 0 {
		return ErrInvalidEventCount
	}
	if int(input.EventCount) != len(input.Events) {
		return ErrEventCountMismatch
	}
	if input.CerRootHash == "" {
		return ErrEmptyCerRootHash
	}
	if input.TraceRootHash == "" {
		return ErrEmptyTraceRootHash
	}
	for i, e := range input.Events {
		if e.EventID == "" {
			return fmt.Errorf("REPLAY_INPUT: events[%d].EventID must not be empty: %w", i, ErrEventIDEmpty)
		}
	}
	return nil
}
