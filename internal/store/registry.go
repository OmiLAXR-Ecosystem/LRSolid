package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
)

// DataStoreConfig holds the runtime configuration for a single data store entry.
type DataStoreConfig struct {
	ID               string
	Name             string
	DatabaseType     string // "postgres" | "mongodb"
	ConnectionString string
}

// StoreFactory creates a StoreBundle from a DataStoreConfig.
// Register adapters at startup (e.g. postgres.NewStoreBundle, mongodb.NewStoreBundle).
type StoreFactory func(ctx context.Context, cfg DataStoreConfig) (StoreBundle, error)

type slot struct {
	bundle StoreBundle
	fp     string
	err    string
	ready  bool
}

// Registry maps data_store_id → StoreBundle and handles lazy initialization.
type Registry struct {
	mu        sync.RWMutex
	slots     map[string]slot
	factories map[string]StoreFactory // keyed by database_type
	defaultID string
}

func NewRegistry() *Registry {
	return &Registry{
		slots:     make(map[string]slot),
		factories: make(map[string]StoreFactory),
	}
}

// RegisterFactory registers a StoreFactory for a given database type.
func (r *Registry) RegisterFactory(databaseType string, f StoreFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[databaseType] = f
}

func fingerprint(cfg DataStoreConfig) string {
	sum := sha256.Sum256([]byte(cfg.DatabaseType + "\n" + cfg.ConnectionString))
	return hex.EncodeToString(sum[:])
}

// Register connects a data store and keeps it for later requests.
// An unchanged, healthy registration is left in place.
func (r *Registry) Register(ctx context.Context, cfg DataStoreConfig) error {
	fp := fingerprint(cfg)

	r.mu.RLock()
	current := r.slots[cfg.ID]
	factory, ok := r.factories[cfg.DatabaseType]
	r.mu.RUnlock()

	if current.ready && current.err == "" && current.fp == fp {
		return nil
	}
	if !ok {
		err := fmt.Errorf("no factory registered for database type %q", cfg.DatabaseType)
		r.fail(cfg.ID, fp, current, err)
		return err
	}

	bundle, err := factory(ctx, cfg)
	if err != nil {
		err = fmt.Errorf("init data store %s (%s): %w", cfg.ID, cfg.Name, err)
		r.fail(cfg.ID, fp, current, err)
		return err
	}
	if bundle.Lifecycle != nil {
		if err := bundle.Lifecycle.Health(ctx); err != nil {
			_ = bundle.Lifecycle.Close()
			err = fmt.Errorf("init data store %s (%s): %w", cfg.ID, cfg.Name, err)
			r.fail(cfg.ID, fp, current, err)
			return err
		}
	}

	var previous Lifecycle
	r.mu.Lock()
	if old, exists := r.slots[cfg.ID]; exists && old.ready && old.bundle.Lifecycle != nil && old.fp != fp {
		previous = old.bundle.Lifecycle
	}
	r.slots[cfg.ID] = slot{bundle: bundle, fp: fp, ready: true}
	r.mu.Unlock()
	if previous != nil {
		_ = previous.Close()
	}
	return nil
}

// fail records a connection error. A healthy bundle is dropped when the
// credentials themselves changed, so requests stop using the old database.
func (r *Registry) fail(id, fp string, current slot, err error) {
	if current.ready && current.fp != fp {
		r.Unregister(id)
		current.ready = false
		current.bundle = StoreBundle{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current.fp = fp
	current.err = err.Error()
	r.slots[id] = current
}

// Probe opens a data store, checks it, and closes it without keeping the connection.
func (r *Registry) Probe(ctx context.Context, cfg DataStoreConfig) error {
	r.mu.RLock()
	factory, ok := r.factories[cfg.DatabaseType]
	r.mu.RUnlock()
	if !ok {
		return fmt.Errorf("no factory registered for database type %q", cfg.DatabaseType)
	}
	bundle, err := factory(ctx, cfg)
	if err != nil {
		return err
	}
	if bundle.Lifecycle == nil {
		return nil
	}
	defer bundle.Lifecycle.Close()
	return bundle.Lifecycle.Health(ctx)
}

// SetDefault marks a data store id as the default fallback.
func (r *Registry) SetDefault(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultID = id
}

// Get returns the StoreBundle for a given data_store_id.
// Falls back to the default if id is empty.
func (r *Registry) Get(dataStoreID string) (StoreBundle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id := dataStoreID
	if id == "" {
		id = r.defaultID
	}

	s, ok := r.slots[id]
	if !ok || !s.ready {
		return StoreBundle{}, fmt.Errorf("data store %q not found in registry", id)
	}
	return s.bundle, nil
}

// Has reports whether a data store is connected.
func (r *Registry) Has(dataStoreID string) bool {
	if r == nil || dataStoreID == "" {
		return false
	}
	_, err := r.Get(dataStoreID)
	return err == nil
}

// Status returns whether the store is connected and the last connection error.
func (r *Registry) Status(dataStoreID string) (connected bool, errMsg string) {
	if r == nil {
		return false, ""
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.slots[dataStoreID]
	if !ok {
		return false, ""
	}
	return s.ready && s.err == "", s.err
}

// IDs lists connected data store ids.
func (r *Registry) IDs() []string {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.slots))
	for id, s := range r.slots {
		if s.ready {
			out = append(out, id)
		}
	}
	return out
}

// Unregister removes a data store from the registry and closes it.
func (r *Registry) Unregister(dataStoreID string) {
	r.mu.Lock()
	s, ok := r.slots[dataStoreID]
	delete(r.slots, dataStoreID)
	r.mu.Unlock()
	if ok && s.ready && s.bundle.Lifecycle != nil {
		_ = s.bundle.Lifecycle.Close()
	}
}

// Close disconnects every registered data store.
func (r *Registry) Close() {
	if r == nil {
		return
	}
	r.mu.Lock()
	slots := r.slots
	r.slots = make(map[string]slot)
	r.mu.Unlock()
	for _, s := range slots {
		if s.ready && s.bundle.Lifecycle != nil {
			_ = s.bundle.Lifecycle.Close()
		}
	}
}

// ReconcileEvent is a connection failure that was not already reported.
type ReconcileEvent struct {
	ID   string
	Name string
	Err  error
}

// Reconcile connects the desired stores and drops registrations that are no longer active.
// Connection strings without a URI scheme are ignored (the seeded control-plane row stores a host, not a DSN).
func (r *Registry) Reconcile(ctx context.Context, cfgs []DataStoreConfig) []ReconcileEvent {
	want := make(map[string]struct{}, len(cfgs))
	var events []ReconcileEvent
	for _, cfg := range cfgs {
		if !strings.Contains(cfg.ConnectionString, "://") {
			continue
		}
		want[cfg.ID] = struct{}{}
		r.mu.RLock()
		prev := r.slots[cfg.ID].err
		r.mu.RUnlock()
		if err := r.Register(ctx, cfg); err != nil && err.Error() != prev {
			events = append(events, ReconcileEvent{ID: cfg.ID, Name: cfg.Name, Err: err})
		}
	}

	r.mu.RLock()
	var stale []string
	for id, s := range r.slots {
		if _, ok := want[id]; !ok && s.ready {
			stale = append(stale, id)
		}
	}
	r.mu.RUnlock()
	for _, id := range stale {
		r.Unregister(id)
	}
	return events
}
