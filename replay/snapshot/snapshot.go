package snapshot

import (
	"fmt"

	"github.com/anomalyco/nexus-ccnf-ref/replay"
)

type ValidationError string

const (
	ErrVersionLockViolation ValidationError = "VERSION_LOCK_VIOLATION"
	ErrStateDivergence      ValidationError = "STATE_DIVERGENCE"
)

func (e ValidationError) Error() string { return string(e) }

func SnapshotFromEvents(events []replay.CEREvent) Snapshot {
	return BuildFromReplay(events)
}

func Validate(snapshot Snapshot, events []replay.CEREvent) error {
	if err := ValidateVersionLock(snapshot); err != nil {
		return err
	}
	if err := ValidateStateEquivalence(snapshot, events); err != nil {
		return err
	}
	return nil
}

func ValidateVersionLock(s Snapshot) error {
	return ValidateTriVersionLock(s)
}

func ValidateStateEquivalence(snapshot Snapshot, events []replay.CEREvent) error {
	replayState := replay.Fold(events)
	if !EqualStates(replayState, snapshot.State) {
		return fmt.Errorf("%w: Fold(events) != snapshot.State", ErrStateDivergence)
	}
	return nil
}

func Verify(snapshot Snapshot, events []replay.CEREvent) error {
	return Validate(snapshot, events)
}

func RoundTrip(events []replay.CEREvent) error {
	snapshot := BuildFromReplay(events)
	return Validate(snapshot, events)
}
