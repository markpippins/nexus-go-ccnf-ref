package replay

func Apply(prev RuntimeState, delta StateDelta) RuntimeState {
	data := make(map[StateKey]StateValue, len(prev.Data)+len(delta.Writes))
	for k, v := range prev.Data {
		data[k] = v
	}
	for k, v := range delta.Writes {
		data[k] = v
	}
	return RuntimeState{
		Data:    data,
		Version: prev.Version + 1,
	}
}
