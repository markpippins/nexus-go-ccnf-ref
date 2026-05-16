package replay

func initialState() RuntimeState {
	return RuntimeState{
		Entities: make(map[string]EntityState),
		Version:  0,
	}
}

func ApplyEvent(state RuntimeState, event CEREvent) RuntimeState {
	entity, _ := getEntity(state, event.EntityKey)
	updated := applyDelta(entity, event.ArtifactID, event.StateDelta, event.Sequence)
	return updateEntity(state, event.EntityKey, updated)
}

func Fold(events []CEREvent) RuntimeState {
	state := initialState()
	for _, e := range events {
		state = ApplyEvent(state, e)
	}
	return state
}
