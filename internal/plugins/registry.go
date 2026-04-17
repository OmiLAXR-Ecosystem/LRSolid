package plugins

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Registry struct {
	widgets []WidgetPlugin
	byID    map[string]WidgetPlugin
}

var Global = &Registry{byID: make(map[string]WidgetPlugin)}

func (r *Registry) Register(p WidgetPlugin) {
	r.widgets = append(r.widgets, p)
	if r.byID == nil {
		r.byID = make(map[string]WidgetPlugin)
	}
	r.byID[p.ID()] = p
}

// All returns all registered plugins in registration order.
func (r *Registry) All() []WidgetPlugin {
	return r.widgets
}

// FetchAll calls each plugin with its specific config from the entries list.
// Only enabled entries are fetched; order from entries is preserved.
// If entries is nil, all plugins are fetched with empty config.
func (r *Registry) FetchAll(ctx context.Context, db *pgxpool.Pool, orgID string, entries []WidgetEntry) (map[string]any, error) {
	result := make(map[string]any, len(r.widgets))

	if len(entries) == 0 {
		for _, p := range r.widgets {
			data, err := p.FetchData(ctx, db, orgID, nil)
			if err != nil {
				return nil, fmt.Errorf("plugin %s: %w", p.ID(), err)
			}
			result[p.ID()] = data
		}
		return result, nil
	}

	for _, entry := range entries {
		if !entry.Enabled {
			continue
		}
		p, ok := r.byID[entry.ID]
		if !ok {
			continue
		}
		data, err := p.FetchData(ctx, db, orgID, entry.Config)
		if err != nil {
			return nil, fmt.Errorf("plugin %s: %w", p.ID(), err)
		}
		result[entry.ID] = data
	}
	return result, nil
}

// DefaultEntries builds the default WidgetEntry list (all enabled, default order).
func (r *Registry) DefaultEntries() []WidgetEntry {
	entries := make([]WidgetEntry, len(r.widgets))
	for i, p := range r.widgets {
		cfg := make(map[string]any)
		for _, f := range p.Schema() {
			cfg[f.Key] = f.Default
		}
		entries[i] = WidgetEntry{ID: p.ID(), Enabled: true, Order: i, Config: cfg}
	}
	return entries
}

// SchemaMap returns the config schema for every registered plugin, keyed by ID.
func (r *Registry) SchemaMap() map[string][]ConfigField {
	m := make(map[string][]ConfigField, len(r.widgets))
	for _, p := range r.widgets {
		m[p.ID()] = p.Schema()
	}
	return m
}
