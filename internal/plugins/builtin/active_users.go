package builtin

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/SGoerzen/LRSolid/internal/plugins"
)

type ActiveUsers struct{}

func (ActiveUsers) ID() string { return "active_users" }

func (ActiveUsers) Schema() []plugins.ConfigField {
	min, max := 1, 365
	return []plugins.ConfigField{
		{Key: "windowDays", Label: "Lookback (days)", Type: "int", Default: 30, Min: &min, Max: &max},
	}
}

func (ActiveUsers) FetchData(ctx context.Context, db *pgxpool.Pool, orgID string, cfg map[string]any) (any, error) {
	windowDays := cfgInt(cfg, "windowDays", 30)
	lookback := time.Now().UTC().Add(-time.Duration(windowDays) * 24 * time.Hour)
	var value int64
	err := db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT st.actor_mbox)
		FROM statements st
		JOIN lrs l ON l.id = st.tenant_id
		JOIN projects p ON p.id = l.project_id
		WHERE p.org_id = $1
		  AND st.actor_mbox IS NOT NULL
		  AND st.stored_at >= $2
	`, orgID, lookback).Scan(&value)
	return map[string]any{"value": value, "windowDays": windowDays}, err
}
