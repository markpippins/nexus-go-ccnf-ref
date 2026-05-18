package reader

import (
	"github.com/anomalyco/nexus-ccnf-ref/runtime/rehydrate/registry"
	"github.com/anomalyco/nexus-ccnf-ref/runtime/rehydrate/snapshot"
	"github.com/anomalyco/nexus-ccnf-ref/runtime/rehydrate/view"

	"github.com/anomalyco/nexus-ccnf-ref/runtime/rehydrate/scan"
)

type Reader struct {
	snap snapshot.ReplaySnapshot
	reg  *registry.ViewRegistry
}

func New(snap snapshot.ReplaySnapshot, reg *registry.ViewRegistry) *Reader {
	return &Reader{snap: snap, reg: reg}
}

func (r *Reader) Scan(prefix []byte) []view.View {
	it := scan.Scan(r.snap, prefix, r.reg)
	defer it.Close()

	var result []view.View
	for it.Next() {
		if v := it.View(); v != nil {
			result = append(result, v)
		}
	}
	return result
}
