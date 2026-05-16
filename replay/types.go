package replay

type CEREvent struct {
	EventID       string
	CausalChainID string
	Sequence      int64
	Timestamp     int64
	EntityKey     string
	ArtifactID    string
	StateDelta    map[string]any
}

type RuntimeState struct {
	Entities map[string]EntityState
	Version  int64
}

type EntityState struct {
	ArtifactStates map[string]any
	LastEventSeq   int64
}
