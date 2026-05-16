package replay

type Cursor struct {
	Index int
}

func NewCursor() Cursor {
	return Cursor{Index: 0}
}

func (c Cursor) Step() Cursor {
	return Cursor{Index: c.Index + 1}
}

func (c Cursor) Jump(i int) Cursor {
	if i < 0 {
		return Cursor{Index: 0}
	}
	return Cursor{Index: i}
}

func (c Cursor) GetIndex() int {
	return c.Index
}

func (c Cursor) Event(events []CEREvent) (CEREvent, bool) {
	if c.Index < 0 || c.Index >= len(events) {
		return CEREvent{}, false
	}
	return events[c.Index], true
}
