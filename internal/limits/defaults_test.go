package limits

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/pgtest"
)

func TestApplyOrgDefaults(t *testing.T) {
	pgtest.WithDB(t, func(ctx context.Context, pool *pgxpool.Pool) {
		orgID := insertOrg(t, ctx, pool, "demo", "Demo")

		if err := inTx(ctx, pool, func(tx pgx.Tx) error {
			return ApplyOrgDefaults(ctx, tx, orgID, 0, 0)
		}); err != nil {
			t.Fatal(err)
		}
		if n := countLimits(t, ctx, pool, orgID); n != 0 {
			t.Fatalf("zero defaults wrote %d rules", n)
		}

		if err := inTx(ctx, pool, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `
				INSERT INTO usage_limits (subject_type, subject_id, metric, max_value, enabled)
				VALUES ('organisation', $1, $2, 999, true)
			`, orgID, MetricStorageBytes)
			return err
		}); err != nil {
			t.Fatal(err)
		}

		if err := inTx(ctx, pool, func(tx pgx.Tx) error {
			return ApplyOrgDefaults(ctx, tx, orgID, 5*1024*1024, 1000)
		}); err != nil {
			t.Fatal(err)
		}

		if got, ok := limitMax(t, ctx, pool, orgID, MetricStorageBytes); !ok || got != 999 {
			t.Fatalf("existing storage rule = %d ok=%v, want 999", got, ok)
		}
		if got, ok := limitMax(t, ctx, pool, orgID, MetricStatementsPerDay); !ok || got != 1000 {
			t.Fatalf("statements per day = %d ok=%v, want 1000", got, ok)
		}
		if n := countLimits(t, ctx, pool, orgID); n != 2 {
			t.Fatalf("rule count = %d, want 2", n)
		}
	})
}

func insertOrg(t *testing.T, ctx context.Context, pool *pgxpool.Pool, slug, name string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(ctx, `INSERT INTO organisations (slug, name) VALUES ($1, $2) RETURNING id::text`, slug, name).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func inTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func countLimits(t *testing.T, ctx context.Context, pool *pgxpool.Pool, orgID string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM usage_limits WHERE subject_type = 'organisation' AND subject_id = $1`, orgID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func limitMax(t *testing.T, ctx context.Context, pool *pgxpool.Pool, orgID, metric string) (int64, bool) {
	t.Helper()
	var max int64
	err := pool.QueryRow(ctx, `
		SELECT max_value FROM usage_limits
		WHERE subject_type = 'organisation' AND subject_id = $1 AND metric = $2
	`, orgID, metric).Scan(&max)
	if err == pgx.ErrNoRows {
		return 0, false
	}
	if err != nil {
		t.Fatal(err)
	}
	return max, true
}
