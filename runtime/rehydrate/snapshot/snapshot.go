package snapshot

import (
	"sort"

	replay "github.com/anomalyco/nexus-ccnf-ref/runtime/replay"
)

type Iterator interface {
	Next() bool
	Key() []byte
	Value() []byte
	Close()
}

type ReplaySnapshot interface {
	Get(key []byte) ([]byte, bool)
	Scan(prefix []byte) Iterator
	Height() uint64
}

type snapshotAdapter struct {
	state replay.RuntimeState
}

func NewFromRuntimeState(state replay.RuntimeState) ReplaySnapshot {
	keys := make([]string, 0, len(state.Data))
	for k := range state.Data {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)
	return &snapshotAdapter{state: state}
}

func (s *snapshotAdapter) Get(key []byte) ([]byte, bool) {
	v, ok := s.state.Data[replay.StateKey(key)]
	return v, ok
}

func (s *snapshotAdapter) Height() uint64 {
	return s.state.Version
}

func (s *snapshotAdapter) Scan(prefix []byte) Iterator {
	keys := make([]string, 0, len(s.state.Data))
	for k := range s.state.Data {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)

	p := string(prefix)
	var filtered []string
	for _, k := range keys {
		if len(k) >= len(p) && k[:len(p)] == p {
			filtered = append(filtered, k)
		}
	}

	idx := 0
	return &sliceIterator{
		keys:   filtered,
		state:  s.state,
		idx:    &idx,
		closed: false,
	}
}

type sliceIterator struct {
	keys   []string
	state  replay.RuntimeState
	idx    *int
	closed bool
}

func (it *sliceIterator) Next() bool {
	if it.closed {
		return false
	}
	*it.idx++
	return *it.idx <= len(it.keys)
}

func (it *sliceIterator) Key() []byte {
	if *it.idx == 0 || *it.idx > len(it.keys) {
		return nil
	}
	return []byte(it.keys[*it.idx-1])
}

func (it *sliceIterator) Value() []byte {
	if *it.idx == 0 || *it.idx > len(it.keys) {
		return nil
	}
	v, _ := it.state.Data[replay.StateKey(it.keys[*it.idx-1])]
	return v
}

func (it *sliceIterator) Close() {
	it.closed = true
}
