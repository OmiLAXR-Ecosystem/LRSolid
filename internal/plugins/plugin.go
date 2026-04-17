package plugins

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConfigField describes a single configurable option exposed by a plugin.
type ConfigField struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Type    string   `json:"type"` // "int" | "select"
	Default any      `json:"default"`
	Options []any    `json:"options,omitempty"`
	Min     *int     `json:"min,omitempty"`
	Max     *int     `json:"max,omitempty"`
}

// WidgetPlugin provides server-side data for a dashboard widget.
type WidgetPlugin interface {
	ID() string
	// Schema declares plugin-specific config fields. Return nil if none.
	Schema() []ConfigField
	// FetchData receives the per-widget config map (may be nil/empty).
	FetchData(ctx context.Context, db *pgxpool.Pool, orgID string, cfg map[string]any) (any, error)
}

// WidgetEntry is a single entry in a saved dashboard config.
type WidgetEntry struct {
	ID      string         `json:"id"`
	Enabled bool           `json:"enabled"`
	Order   int            `json:"order"`
	Config  map[string]any `json:"config"`
}
