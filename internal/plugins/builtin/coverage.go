package builtin

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/SGoerzen/LRSolid/internal/plugins"
)

type Coverage struct{}

func (Coverage) ID() string { return "coverage" }

func (Coverage) Schema() []plugins.ConfigField { return nil }

type coverageItem struct {
	Name  string `json:"name"`
	Have  int64  `json:"have"`
	Total int64  `json:"total"`
	Pct   int    `json:"pct"`
}

func pct(have, total int64) int {
	if total <= 0 {
		return 0
	}
	return int(have * 100 / total)
}

func (Coverage) FetchData(ctx context.Context, db *pgxpool.Pool, orgID string, _ map[string]any) (any, error) {
	var projectsTotal, projectsHave int64
	if err := db.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE statement_count > 0)
		FROM projects WHERE org_id = $1
	`, orgID).Scan(&projectsTotal, &projectsHave); err != nil {
		return nil, err
	}

	var lrsTotal, lrsHave int64
	if err := db.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE l.statement_count > 0)
		FROM lrs l JOIN projects p ON p.id = l.project_id WHERE p.org_id = $1
	`, orgID).Scan(&lrsTotal, &lrsHave); err != nil {
		return nil, err
	}

	var clientsTotal, clientsHave int64
	if err := db.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE c.statement_count > 0)
		FROM clients c JOIN projects p ON p.id = c.project_id WHERE p.org_id = $1
	`, orgID).Scan(&clientsTotal, &clientsHave); err != nil {
		return nil, err
	}

	return []coverageItem{
		{Name: "Projects", Have: projectsHave, Total: projectsTotal, Pct: pct(projectsHave, projectsTotal)},
		{Name: "LRS", Have: lrsHave, Total: lrsTotal, Pct: pct(lrsHave, lrsTotal)},
		{Name: "Clients", Have: clientsHave, Total: clientsTotal, Pct: pct(clientsHave, clientsTotal)},
	}, nil
}
