package snapshot

import "github.com/anomalyco/nexus-ccnf-ref/replay"

type Snapshot struct {
	State              replay.RuntimeState
	CCNFVersion        int64
	CollapseVersion    int64
	RehydrationVersion int64
	Timestamp          int64
}

type SnapshotContext struct {
	Snapshot     Snapshot
	SourceEvents []replay.CEREvent
}
