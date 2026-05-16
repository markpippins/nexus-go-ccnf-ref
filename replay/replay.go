package replay

func Replay(events []CEREvent) RuntimeState {
	return Fold(events)
}

func ReplayFromCursor(events []CEREvent, cursor Cursor) RuntimeState {
	if cursor.Index < 0 {
		return Fold(events)
	}
	if cursor.Index >= len(events) {
		return initialState()
	}
	return Fold(events[cursor.Index:])
}

func ReplayRange(events []CEREvent, start, end int) RuntimeState {
	if start < 0 {
		start = 0
	}
	if end > len(events) {
		end = len(events)
	}
	if start >= end {
		return initialState()
	}
	return Fold(events[start:end])
}
