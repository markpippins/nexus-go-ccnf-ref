package decode

import "github.com/anomalyco/nexus-ccnf-ref/runtime/rehydrate/view"

type Decoder interface {
	Decode(key, value []byte) (view.View, error)
}
