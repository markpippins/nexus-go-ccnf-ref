package snapshot

import (
	"reflect"

	"github.com/anomalyco/nexus-ccnf-ref/replay"
)

func Compare(a, b Snapshot) bool {
	if a.CCNFVersion != b.CCNFVersion {
		return false
	}
	if a.CollapseVersion != b.CollapseVersion {
		return false
	}
	if a.RehydrationVersion != b.RehydrationVersion {
		return false
	}
	if a.Timestamp != b.Timestamp {
		return false
	}
	return equalStates(a.State, b.State)
}

func equalStates(a, b replay.RuntimeState) bool {
	if a.Version != b.Version {
		return false
	}
	if len(a.Entities) != len(b.Entities) {
		return false
	}
	for key, ea := range a.Entities {
		eb, ok := b.Entities[key]
		if !ok {
			return false
		}
		if !equalEntityState(ea, eb) {
			return false
		}
	}
	return true
}

func equalEntityState(a, b replay.EntityState) bool {
	if a.LastEventSeq != b.LastEventSeq {
		return false
	}
	if len(a.ArtifactStates) != len(b.ArtifactStates) {
		return false
	}
	for k, va := range a.ArtifactStates {
		vb, ok := b.ArtifactStates[k]
		if !ok {
			return false
		}
		if !reflect.DeepEqual(va, vb) {
			return false
		}
	}
	return true
}
