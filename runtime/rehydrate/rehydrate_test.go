package rehydrate_test

import (
	"testing"

	"github.com/anomalyco/nexus-ccnf-ref/runtime/rehydrate/reader"
	"github.com/anomalyco/nexus-ccnf-ref/runtime/rehydrate/registry"
	"github.com/anomalyco/nexus-ccnf-ref/runtime/rehydrate/snapshot"
	"github.com/anomalyco/nexus-ccnf-ref/runtime/rehydrate/view"
	replay "github.com/anomalyco/nexus-ccnf-ref/runtime/replay"
)

type exampleDecoder struct{}

func (exampleDecoder) Decode(key, value []byte) (view.View, error) {
	return view.ExampleView{Key: string(key), Value: string(value)}, nil
}

func TestRehydrationRoundTrip(t *testing.T) {
	state := replay.RuntimeState{
		Data: map[replay.StateKey]replay.StateValue{
			"example/k1": []byte("v1"),
			"example/k2": []byte("v2"),
			"other/z":    []byte("ignored"),
		},
		Version: 1,
	}

	snap := snapshot.NewFromRuntimeState(state)
	reg := registry.New(registry.RouteSpec{
		Prefix:  []byte("example/"),
		Decoder: exampleDecoder{},
	})

	r := reader.New(snap, reg)
	views := r.Scan([]byte("example/"))

	if len(views) != 2 {
		t.Fatalf("expected 2 views, got %d", len(views))
	}

	found := map[string]bool{}
	for _, v := range views {
		ev, ok := v.(view.ExampleView)
		if !ok {
			t.Fatalf("expected ExampleView, got %T", v)
		}
		found[ev.Key] = true
	}
	if !found["example/k1"] || !found["example/k2"] {
		t.Fatal("expected both example keys in results")
	}
}

func TestRehydrationEmptyPrefix(t *testing.T) {
	state := replay.RuntimeState{
		Data:    map[replay.StateKey]replay.StateValue{},
		Version: 0,
	}

	snap := snapshot.NewFromRuntimeState(state)
	reg := registry.New()

	r := reader.New(snap, reg)
	views := r.Scan(nil)

	if len(views) != 0 {
		t.Fatalf("expected 0 views, got %d", len(views))
	}
}

func TestRehydrationNoMatchingPrefix(t *testing.T) {
	state := replay.RuntimeState{
		Data: map[replay.StateKey]replay.StateValue{
			"a/k1": []byte("v1"),
		},
		Version: 1,
	}

	snap := snapshot.NewFromRuntimeState(state)
	reg := registry.New(registry.RouteSpec{
		Prefix:  []byte("b/"),
		Decoder: exampleDecoder{},
	})

	r := reader.New(snap, reg)
	views := r.Scan([]byte("b/"))

	if len(views) != 0 {
		t.Fatalf("expected 0 views, got %d", len(views))
	}
}

func TestRehydrationSnapshotHeight(t *testing.T) {
	state := replay.RuntimeState{
		Data:    map[replay.StateKey]replay.StateValue{},
		Version: 42,
	}

	snap := snapshot.NewFromRuntimeState(state)
	if snap.Height() != 42 {
		t.Fatalf("expected height 42, got %d", snap.Height())
	}
}

func TestRehydrationIteratorClose(t *testing.T) {
	state := replay.RuntimeState{
		Data: map[replay.StateKey]replay.StateValue{
			"x/a": []byte("1"),
			"x/b": []byte("2"),
		},
		Version: 1,
	}

	snap := snapshot.NewFromRuntimeState(state)
	it := snap.Scan([]byte("x/"))
	it.Close()

	if it.Next() {
		t.Fatal("iterator should return false after close")
	}
}

func TestRehydrationGet(t *testing.T) {
	state := replay.RuntimeState{
		Data: map[replay.StateKey]replay.StateValue{
			"key1": []byte("val1"),
		},
		Version: 1,
	}

	snap := snapshot.NewFromRuntimeState(state)
	v, ok := snap.Get([]byte("key1"))
	if !ok || string(v) != "val1" {
		t.Fatalf("expected val1, got %s", string(v))
	}

	_, ok = snap.Get([]byte("nonexistent"))
	if ok {
		t.Fatal("expected false for nonexistent key")
	}
}

func TestRehydrationPrefixFiltering(t *testing.T) {
	state := replay.RuntimeState{
		Data: map[replay.StateKey]replay.StateValue{
			"alpha/1":   []byte("a1"),
			"alpha/2":   []byte("a2"),
			"beta/1":    []byte("b1"),
			"gamma/1":   []byte("g1"),
		},
		Version: 1,
	}

	snap := snapshot.NewFromRuntimeState(state)
	reg := registry.New(
		registry.RouteSpec{
			Prefix:  []byte("alpha/"),
			Decoder: exampleDecoder{},
		},
		registry.RouteSpec{
			Prefix:  []byte("gamma/"),
			Decoder: exampleDecoder{},
		},
	)

	r := reader.New(snap, reg)
	views := r.Scan([]byte("alpha/"))

	if len(views) != 2 {
		t.Fatalf("expected 2 alpha views, got %d", len(views))
	}

	r2 := reader.New(snap, reg)
	views2 := r2.Scan([]byte("gamma/"))

	if len(views2) != 1 {
		t.Fatalf("expected 1 gamma view, got %d", len(views2))
	}

	viewsBeta := r.Scan([]byte("beta/"))
	if len(viewsBeta) != 0 {
		t.Fatalf("expected 0 beta views (no decoder), got %d", len(viewsBeta))
	}
}
