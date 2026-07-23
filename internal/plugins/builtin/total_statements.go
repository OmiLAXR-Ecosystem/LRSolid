package builtin

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/SGoerzen/LRSolid/internal/plugins"
)

type TotalStatements struct{}

func (TotalStatements) ID() string { return "total_statements" }

func (TotalStatements) Schema() []plugins.ConfigField { return nil }

func (TotalStatements) FetchData(ctx context.Context, db *pgxpool.Pool, orgID string, _ map[string]any) (any, error) {
	var value int64
	err := db.QueryRow(ctx, `SELECT statement_count FROM organisations WHERE id=$1`, orgID).Scan(&value)
	return map[string]any{"value": value}, err
}
