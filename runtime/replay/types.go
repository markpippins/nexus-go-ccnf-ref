package replay

type ReplayInput struct {
	Events            []ReplayEvent
	CerRootHash       string
	TraceRootHash     string
	ReplayBindingHash string
	CCNFVersion       int
	SemanticsVersion  int
	EventCount        uint64
}

type ReplayEvent struct {
	EventID     string
	PrevEventID string
	Delta       StateDelta
	DeltaHash   string
}

type StateDelta struct {
	Writes map[StateKey]StateValue
}

type StateKey string

type StateValue []byte

type RuntimeState struct {
	Data    map[StateKey]StateValue
	Version uint64
}

type ReplayOutput struct {
	FinalState        RuntimeState
	EventCount        uint64
	CerRootHash       string
	TraceRootHash     string
	ReplayBindingHash string
}
