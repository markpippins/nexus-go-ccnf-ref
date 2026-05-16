package snapshot

import (
	"time"

	"github.com/anomalyco/nexus-ccnf-ref/replay"
)

const (
	CurrentCCNFVersion        int64 = 1
	CurrentCollapseVersion    int64 = 1
	CurrentRehydrationVersion int64 = 1
)

func Build(state replay.RuntimeState, events []replay.CEREvent) Snapshot {
	var ts int64
	if len(events) > 0 {
		ts = events[len(events)-1].Timestamp
	}
	if ts == 0 {
		ts = time.Now().Unix()
	}

	return Snapshot{
		State:              copyState(state),
		CCNFVersion:        CurrentCCNFVersion,
		CollapseVersion:    CurrentCollapseVersion,
		RehydrationVersion: CurrentRehydrationVersion,
		Timestamp:          ts,
	}
}

func copyState(state replay.RuntimeState) replay.RuntimeState {
	entities := make(map[string]replay.EntityState, len(state.Entities))
	for ek, es := range state.Entities {
		artifacts := make(map[string]any, len(es.ArtifactStates))
		for k, v := range es.ArtifactStates {
			artifacts[k] = v
		}
		entities[ek] = replay.EntityState{
			ArtifactStates: artifacts,
			LastEventSeq:   es.LastEventSeq,
		}
	}
	return replay.RuntimeState{
		Entities: entities,
		Version:  state.Version,
	}
}

func BuildFromReplay(events []replay.CEREvent) Snapshot {
	state := replay.Fold(events)
	return Build(state, events)
}
