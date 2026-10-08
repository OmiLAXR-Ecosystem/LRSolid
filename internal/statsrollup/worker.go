package statsrollup

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Worker struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Worker {
	return &Worker{db: db}
}

type batchRow struct {
	StoredAt  time.Time
	TenantID  string
	ID        string
	Day       time.Time
	OrgID     string
	ProjectID string
	ClientID  *string
	Bytes     int64
}

type dayScopeKey struct {
	Day   time.Time
	Scope string
	ID    string
}

type hourScopeKey struct {
	Hour  time.Time
	Scope string
	ID    string
}

func (w *Worker) RunBatch(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 5000
	}

	tx, err := w.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO statement_rollup_state (id)
		VALUES (1)
		ON CONFLICT (id) DO NOTHING
	`); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM statement_stats_hourly WHERE hour < now() - interval '48 hours'`); err != nil {
		return 0, err
	}

	var lastStoredAt time.Time
	var lastTenantID, lastStatementID string
	if err := tx.QueryRow(ctx, `
		SELECT last_stored_at, last_tenant_id, last_statement_id
		FROM statement_rollup_state
		WHERE id = 1
		FOR UPDATE
	`).Scan(&lastStoredAt, &lastTenantID, &lastStatementID); err != nil {
		return 0, err
	}

	rows, err := tx.Query(ctx, `
		SELECT
			st.stored_at,
			st.tenant_id,
			st.id,
			date_trunc('day', st.stored_at AT TIME ZONE 'UTC')::date AS day,
			p.org_id::text,
			l.project_id::text,
			st.client_id::text,
			octet_length(st.statement::text)
		FROM statements st
		JOIN lrs l ON l.id = st.tenant_id
		JOIN projects p ON p.id = l.project_id
		WHERE (st.stored_at, st.tenant_id, st.id) > ($1, $2, $3)
		ORDER BY st.stored_at ASC, st.tenant_id ASC, st.id ASC
		LIMIT $4
	`, lastStoredAt, lastTenantID, lastStatementID, limit)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	batch := make([]batchRow, 0, limit)
	for rows.Next() {
		var r batchRow
		if err := rows.Scan(&r.StoredAt, &r.TenantID, &r.ID, &r.Day, &r.OrgID, &r.ProjectID, &r.ClientID, &r.Bytes); err != nil {
			return 0, err
		}
		batch = append(batch, r)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	if len(batch) == 0 {
		if _, err := tx.Exec(ctx, `UPDATE statement_rollup_state SET updated_at=now() WHERE id=1`); err != nil {
			return 0, err
		}
		if err := tx.Commit(ctx); err != nil {
			return 0, err
		}
		return 0, nil
	}

	orgCounts := map[string]int64{}
	projectCounts := map[string]int64{}
	lrsCounts := map[string]int64{}
	clientCounts := map[string]int64{}
	orgBytes := map[string]int64{}
	projectBytes := map[string]int64{}
	lrsBytes := map[string]int64{}
	clientBytes := map[string]int64{}
	dayCounts := map[dayScopeKey]int64{}
	hourCounts := map[hourScopeKey]int64{}

	for _, r := range batch {
		hour := r.StoredAt.UTC().Truncate(time.Hour)
		orgCounts[r.OrgID]++
		projectCounts[r.ProjectID]++
		lrsCounts[r.TenantID]++
		orgBytes[r.OrgID] += r.Bytes
		projectBytes[r.ProjectID] += r.Bytes
		lrsBytes[r.TenantID] += r.Bytes
		dayCounts[dayScopeKey{Day: r.Day, Scope: "global", ID: "*"}]++
		dayCounts[dayScopeKey{Day: r.Day, Scope: "organisation", ID: r.OrgID}]++
		dayCounts[dayScopeKey{Day: r.Day, Scope: "project", ID: r.ProjectID}]++
		dayCounts[dayScopeKey{Day: r.Day, Scope: "lrs", ID: r.TenantID}]++
		hourCounts[hourScopeKey{Hour: hour, Scope: "global", ID: "*"}]++
		hourCounts[hourScopeKey{Hour: hour, Scope: "organisation", ID: r.OrgID}]++
		hourCounts[hourScopeKey{Hour: hour, Scope: "project", ID: r.ProjectID}]++
		hourCounts[hourScopeKey{Hour: hour, Scope: "lrs", ID: r.TenantID}]++
		if r.ClientID != nil && *r.ClientID != "" {
			clientCounts[*r.ClientID]++
			clientBytes[*r.ClientID] += r.Bytes
			dayCounts[dayScopeKey{Day: r.Day, Scope: "client", ID: *r.ClientID}]++
			hourCounts[hourScopeKey{Hour: hour, Scope: "client", ID: *r.ClientID}]++
		}
	}

	pgBatch := &pgx.Batch{}
	for id, c := range orgCounts {
		pgBatch.Queue(`UPDATE organisations SET statement_count = statement_count + $1 WHERE id = $2::uuid`, c, id)
	}
	for id, c := range projectCounts {
		pgBatch.Queue(`UPDATE projects SET statement_count = statement_count + $1 WHERE id = $2::uuid`, c, id)
	}
	for id, c := range lrsCounts {
		pgBatch.Queue(`UPDATE lrs SET statement_count = statement_count + $1 WHERE id = $2`, c, id)
	}
	for id, c := range clientCounts {
		pgBatch.Queue(`UPDATE clients SET statement_count = statement_count + $1 WHERE id = $2::uuid`, c, id)
	}
	for id, c := range orgBytes {
		pgBatch.Queue(`UPDATE organisations SET storage_bytes = storage_bytes + $1 WHERE id = $2::uuid`, c, id)
	}
	for id, c := range projectBytes {
		pgBatch.Queue(`UPDATE projects SET storage_bytes = storage_bytes + $1 WHERE id = $2::uuid`, c, id)
	}
	for id, c := range lrsBytes {
		pgBatch.Queue(`UPDATE lrs SET storage_bytes = storage_bytes + $1 WHERE id = $2`, c, id)
	}
	for id, c := range clientBytes {
		pgBatch.Queue(`UPDATE clients SET storage_bytes = storage_bytes + $1 WHERE id = $2::uuid`, c, id)
	}
	for k, c := range dayCounts {
		pgBatch.Queue(`
			INSERT INTO statement_stats_daily (day, scope_type, scope_id, statement_count)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (day, scope_type, scope_id)
			DO UPDATE SET statement_count = statement_stats_daily.statement_count + EXCLUDED.statement_count
		`, k.Day, k.Scope, k.ID, c)
	}
	for k, c := range hourCounts {
		pgBatch.Queue(`
			INSERT INTO statement_stats_hourly (hour, scope_type, scope_id, statement_count)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (hour, scope_type, scope_id)
			DO UPDATE SET statement_count = statement_stats_hourly.statement_count + EXCLUDED.statement_count
		`, k.Hour, k.Scope, k.ID, c)
	}

	last := batch[len(batch)-1]
	pgBatch.Queue(`
		UPDATE statement_rollup_state
		SET last_stored_at = $1, last_tenant_id = $2, last_statement_id = $3, updated_at = now()
		WHERE id = 1
	`, last.StoredAt, last.TenantID, last.ID)

	br := tx.SendBatch(ctx, pgBatch)
	for i := 0; i < pgBatch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			_ = br.Close()
			return 0, err
		}
	}
	if err := br.Close(); err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(batch), nil
}
