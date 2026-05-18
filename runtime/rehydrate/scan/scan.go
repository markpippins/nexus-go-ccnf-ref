package scan

import (
	"github.com/anomalyco/nexus-ccnf-ref/runtime/rehydrate/registry"
	"github.com/anomalyco/nexus-ccnf-ref/runtime/rehydrate/snapshot"
	"github.com/anomalyco/nexus-ccnf-ref/runtime/rehydrate/view"
)

type Iterator interface {
	Next() bool
	Key() []byte
	View() view.View
	Close()
}

type scanIterator struct {
	inner snapshot.Iterator
	reg   *registry.ViewRegistry
}

func (it *scanIterator) Next() bool {
	return it.inner.Next()
}

func (it *scanIterator) Key() []byte {
	return it.inner.Key()
}

func (it *scanIterator) View() view.View {
	k := it.inner.Key()
	v := it.inner.Value()
	decoded, ok := it.reg.Decode(k, v)
	if !ok {
		return nil
	}
	return decoded
}

func (it *scanIterator) Close() {
	it.inner.Close()
}

func Scan(snap snapshot.ReplaySnapshot, prefix []byte, reg *registry.ViewRegistry) Iterator {
	inner := snap.Scan(prefix)
	return &scanIterator{inner: inner, reg: reg}
}
