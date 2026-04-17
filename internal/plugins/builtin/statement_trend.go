package builtin

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/SGoerzen/LRSolid/internal/plugins"
)

type StatementTrend struct{}

func (StatementTrend) ID() string { return "statement_trend" }

func (StatementTrend) Schema() []plugins.ConfigField {
	min, max := 3, 90
	return []plugins.ConfigField{
		{Key: "windowDays", Label: "Window (days)", Type: "int", Default: 7, Min: &min, Max: &max},
	}
}

type trendPoint struct {
	Day   string `json:"day"`
	Count int64  `json:"count"`
}

func (StatementTrend) FetchData(ctx context.Context, db *pgxpool.Pool, orgID string, cfg map[string]any) (any, error) {
	windowDays := cfgInt(cfg, "windowDays", 7)
	rows, err := db.Query(ctx, fmt.Sprintf(`
		SELECT day::text, statement_count
		FROM statement_stats_daily
		WHERE scope_type = 'organisation'
		  AND scope_id = $1
		  AND day >= (current_date - INTERVAL '%d days')::date
		ORDER BY day ASC
	`, windowDays-1), orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]trendPoint, 0, windowDays)
	for rows.Next() {
		var p trendPoint
		if err := rows.Scan(&p.Day, &p.Count); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}
