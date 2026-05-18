package replay

func initialState() RuntimeState {
	return RuntimeState{
		Data:    make(map[StateKey]StateValue),
		Version: 0,
	}
}

func Fold(events []ReplayEvent) RuntimeState {
	state := initialState()
	for _, e := range events {
		state = Apply(state, e.Delta)
	}
	return state
}
