package registry

import (
	"bytes"

	"github.com/anomalyco/nexus-ccnf-ref/runtime/rehydrate/decode"
	"github.com/anomalyco/nexus-ccnf-ref/runtime/rehydrate/view"
)

type route struct {
	prefix  []byte
	decoder decode.Decoder
}

type ViewRegistry struct {
	routes []route
}

type RouteSpec struct {
	Prefix  []byte
	Decoder decode.Decoder
}

func New(specs ...RouteSpec) *ViewRegistry {
	r := &ViewRegistry{}
	for _, s := range specs {
		r.routes = append(r.routes, route{prefix: s.Prefix, decoder: s.Decoder})
	}
	return r
}

func (r *ViewRegistry) Decode(k, v []byte) (view.View, bool) {
	for _, rt := range r.routes {
		if bytes.HasPrefix(k, rt.prefix) {
			view, err := rt.decoder.Decode(k, v)
			if err != nil {
				return nil, false
			}
			return view, true
		}
	}
	return nil, false
}
