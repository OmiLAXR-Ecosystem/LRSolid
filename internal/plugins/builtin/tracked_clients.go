package builtin

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/SGoerzen/LRSolid/internal/plugins"
)

type TrackedClients struct{}

func (TrackedClients) ID() string { return "tracked_clients" }

func (TrackedClients) Schema() []plugins.ConfigField { return nil }

func (TrackedClients) FetchData(ctx context.Context, db *pgxpool.Pool, orgID string, _ map[string]any) (any, error) {
	var total, active int64
	err := db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE c.statement_count > 0)
		FROM clients c
		JOIN projects p ON p.id = c.project_id
		WHERE p.org_id = $1
	`, orgID).Scan(&total, &active)
	return map[string]any{"value": active, "total": total}, err
}
