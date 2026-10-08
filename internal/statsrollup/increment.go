package statsrollup

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/store"
)

// Counters applies statement totals for external data stores directly.
// Rows in the control-plane statements table are still counted by Worker.
type Counters struct {
	DB *pgxpool.Pool
}

func (c Counters) Add(ctx context.Context, tenant store.TenantID, clientID *string, n int, bytes int64, at time.Time) error {
	if c.DB == nil || n <= 0 || tenant == "" {
		return nil
	}
	if bytes < 0 {
		bytes = 0
	}
	tx, err := c.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var projectID, orgID string
	err = tx.QueryRow(ctx, `
		SELECT l.project_id::text, p.org_id::text
		FROM lrs l
		JOIN projects p ON p.id = l.project_id
		WHERE l.id = $1
	`, string(tenant)).Scan(&projectID, &orgID)
	if err == pgx.ErrNoRows {
		return tx.Commit(ctx)
	}
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `UPDATE lrs SET statement_count = statement_count + $1, storage_bytes = storage_bytes + $2 WHERE id = $3`, n, bytes, string(tenant)); err != nil {
		return err
	}
	if projectID != "" {
		if _, err := tx.Exec(ctx, `UPDATE projects SET statement_count = statement_count + $1, storage_bytes = storage_bytes + $2 WHERE id = $3::uuid`, n, bytes, projectID); err != nil {
			return err
		}
	}
	if orgID != "" {
		if _, err := tx.Exec(ctx, `UPDATE organisations SET statement_count = statement_count + $1, storage_bytes = storage_bytes + $2 WHERE id = $3::uuid`, n, bytes, orgID); err != nil {
			return err
		}
	}
	if clientID != nil && *clientID != "" {
		if _, err := tx.Exec(ctx, `UPDATE clients SET statement_count = statement_count + $1, storage_bytes = storage_bytes + $2 WHERE id = $3::uuid`, n, bytes, *clientID); err != nil {
			return err
		}
	}

	day := time.Date(at.UTC().Year(), at.UTC().Month(), at.UTC().Day(), 0, 0, 0, 0, time.UTC)
	hour := at.UTC().Truncate(time.Hour)
	scopes := []struct{ typ, id string }{
		{"global", "*"},
		{"lrs", string(tenant)},
	}
	if orgID != "" {
		scopes = append(scopes, struct{ typ, id string }{"organisation", orgID})
	}
	if projectID != "" {
		scopes = append(scopes, struct{ typ, id string }{"project", projectID})
	}
	if clientID != nil && *clientID != "" {
		scopes = append(scopes, struct{ typ, id string }{"client", *clientID})
	}
	for _, scope := range scopes {
		if _, err := tx.Exec(ctx, `
			INSERT INTO statement_stats_daily (day, scope_type, scope_id, statement_count)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (day, scope_type, scope_id)
			DO UPDATE SET statement_count = statement_stats_daily.statement_count + EXCLUDED.statement_count
		`, day, scope.typ, scope.id, n); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO statement_stats_hourly (hour, scope_type, scope_id, statement_count)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (hour, scope_type, scope_id)
			DO UPDATE SET statement_count = statement_stats_hourly.statement_count + EXCLUDED.statement_count
		`, hour, scope.typ, scope.id, n); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
