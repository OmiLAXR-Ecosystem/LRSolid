package store

import (
	"context"
	"fmt"
	"sync"
)

// DataStoreConfig holds the runtime configuration for a single data store entry.
type DataStoreConfig struct {
	ID               string
	Name             string
	DatabaseType     string // "postgres" | "mongodb"
	ConnectionString string
}

// StoreBundle groups all xAPI stores for a single data store instance.
type StoreBundle struct {
	Statements      StatementStore
	States          StateStore
	AgentProfiles   AgentProfileStore
	ActivityProfiles ActivityProfileStore
	Attachments     AttachmentStore
}

// StoreFactory creates a StoreBundle from a DataStoreConfig.
// Register adapters at startup (e.g. postgres.NewStoreBundle, mongodb.NewStoreBundle).
type StoreFactory func(ctx context.Context, cfg DataStoreConfig) (StoreBundle, error)

// Registry maps data_store_id → StoreBundle and handles lazy initialization.
type Registry struct {
	mu        sync.RWMutex
	bundles   map[string]StoreBundle
	factories map[string]StoreFactory // keyed by database_type
	defaultID string
}

func NewRegistry() *Registry {
	return &Registry{
		bundles:   make(map[string]StoreBundle),
		factories: make(map[string]StoreFactory),
	}
}

// RegisterFactory registers a StoreFactory for a given database type.
func (r *Registry) RegisterFactory(databaseType string, f StoreFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[databaseType] = f
}

// Register initializes and registers a store bundle for a data store config.
func (r *Registry) Register(ctx context.Context, cfg DataStoreConfig) error {
	r.mu.RLock()
	factory, ok := r.factories[cfg.DatabaseType]
	r.mu.RUnlock()
	if !ok {
		return fmt.Errorf("no factory registered for database type %q", cfg.DatabaseType)
	}

	bundle, err := factory(ctx, cfg)
	if err != nil {
		return fmt.Errorf("init data store %s (%s): %w", cfg.ID, cfg.Name, err)
	}

	r.mu.Lock()
	r.bundles[cfg.ID] = bundle
	r.mu.Unlock()
	return nil
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

	bundle, ok := r.bundles[id]
	if !ok {
		return StoreBundle{}, fmt.Errorf("data store %q not found in registry", id)
	}
	return bundle, nil
}

// Unregister removes a data store from the registry (e.g. on delete).
func (r *Registry) Unregister(dataStoreID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.bundles, dataStoreID)
}
