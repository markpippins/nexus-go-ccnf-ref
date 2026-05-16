package replay

func applyDelta(entity EntityState, artifactID string, delta map[string]any, seq int64) EntityState {
	artifacts := entity.ArtifactStates
	if artifacts == nil {
		artifacts = make(map[string]any)
	}

	merged := make(map[string]any, len(artifacts))
	for k, v := range artifacts {
		merged[k] = v
	}
	for k, v := range delta {
		merged[k] = v
	}

	return EntityState{
		ArtifactStates: merged,
		LastEventSeq:   seq,
	}
}

func updateEntity(state RuntimeState, entityKey string, updated EntityState) RuntimeState {
	entities := make(map[string]EntityState, len(state.Entities)+1)
	for k, v := range state.Entities {
		entities[k] = v
	}
	entities[entityKey] = updated

	return RuntimeState{
		Entities: entities,
		Version:  state.Version + 1,
	}
}

func getEntity(state RuntimeState, key string) (EntityState, bool) {
	e, ok := state.Entities[key]
	return e, ok
}
