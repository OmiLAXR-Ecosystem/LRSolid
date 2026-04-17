package builtin

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/SGoerzen/LRSolid/internal/plugins"
)

type StatementsToday struct{}

func (StatementsToday) ID() string { return "statements_today" }

func (StatementsToday) Schema() []plugins.ConfigField { return nil }

func (StatementsToday) FetchData(ctx context.Context, db *pgxpool.Pool, orgID string, _ map[string]any) (any, error) {
	day := time.Now().UTC().Format("2006-01-02")
	var value int64
	err := db.QueryRow(ctx, `
		SELECT COALESCE(statement_count, 0)
		FROM statement_stats_daily
		WHERE day = $1::date
		  AND scope_type = 'organisation'
		  AND scope_id = $2
	`, day, orgID).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}
	return map[string]any{"value": value}, err
}
