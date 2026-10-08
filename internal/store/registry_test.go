package store

import (
	"context"
	"errors"
	"testing"
)

type nopLife struct{ closed int }

func (n *nopLife) Close() error                 { n.closed++; return nil }
func (n *nopLife) Health(context.Context) error { return nil }

func TestRegistrySkipsUnchangedConnection(t *testing.T) {
	r := NewRegistry()
	opens := 0
	r.RegisterFactory("mongodb", func(ctx context.Context, cfg DataStoreConfig) (StoreBundle, error) {
		opens++
		return StoreBundle{Lifecycle: &nopLife{}, DatabaseType: "mongodb", Capabilities: ExternalCapabilities()}, nil
	})
	cfg := DataStoreConfig{ID: "ds1", Name: "mongo", DatabaseType: "mongodb", ConnectionString: "mongodb://localhost/lrsolid"}
	if err := r.Register(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if opens != 1 {
		t.Fatalf("opens = %d, want 1", opens)
	}
	if !r.Has("ds1") {
		t.Fatal("expected connected store")
	}
}

func TestRegistryProbeDoesNotKeepConnection(t *testing.T) {
	r := NewRegistry()
	life := &nopLife{}
	r.RegisterFactory("postgres", func(ctx context.Context, cfg DataStoreConfig) (StoreBundle, error) {
		return StoreBundle{Lifecycle: life, DatabaseType: "postgres"}, nil
	})
	err := r.Probe(context.Background(), DataStoreConfig{DatabaseType: "postgres", ConnectionString: "postgres://localhost/db"})
	if err != nil {
		t.Fatal(err)
	}
	if life.closed != 1 {
		t.Fatalf("closed = %d, want 1", life.closed)
	}
	if r.Has("") || len(r.IDs()) != 0 {
		t.Fatal("probe must not register the store")
	}
}

func TestReconcileDropsRemovedStores(t *testing.T) {
	r := NewRegistry()
	r.RegisterFactory("mongodb", func(ctx context.Context, cfg DataStoreConfig) (StoreBundle, error) {
		if cfg.ConnectionString == "mongodb://bad" {
			return StoreBundle{}, errors.New("refused")
		}
		return StoreBundle{Lifecycle: &nopLife{}, DatabaseType: "mongodb"}, nil
	})
	ctx := context.Background()
	events := r.Reconcile(ctx, []DataStoreConfig{
		{ID: "keep", Name: "keep", DatabaseType: "mongodb", ConnectionString: "mongodb://localhost/a"},
		{ID: "seed", Name: "seed", DatabaseType: "postgres", ConnectionString: "127.0.0.1"},
	})
	if len(events) != 0 {
		t.Fatalf("unexpected events: %+v", events)
	}
	if !r.Has("keep") {
		t.Fatal("expected keep to be connected")
	}
	if r.Has("seed") {
		t.Fatal("host-only seed must not be registered")
	}

	events = r.Reconcile(ctx, nil)
	if r.Has("keep") {
		t.Fatal("removed store should be disconnected")
	}
	_ = events
}
