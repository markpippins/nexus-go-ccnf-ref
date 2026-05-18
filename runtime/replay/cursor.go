package replay

type Cursor struct {
	index int
}

func NewCursor() Cursor {
	return Cursor{index: 0}
}

func (c Cursor) Step() Cursor {
	return Cursor{index: c.index + 1}
}

func (c Cursor) Jump(i int) Cursor {
	if i < 0 {
		return Cursor{index: 0}
	}
	return Cursor{index: i}
}

func (c Cursor) Index() int {
	return c.index
}

func (c Cursor) Valid(length int) bool {
	return c.index >= 0 && c.index < length
}

func (c Cursor) Event(events []ReplayEvent) (ReplayEvent, bool) {
	if !c.Valid(len(events)) {
		return ReplayEvent{}, false
	}
	return events[c.index], true
}
