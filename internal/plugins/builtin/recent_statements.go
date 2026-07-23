package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/SGoerzen/LRSolid/internal/plugins"
)

type RecentStatements struct{}

func (RecentStatements) ID() string { return "recent_statements" }

func (RecentStatements) Schema() []plugins.ConfigField {
	min, max := 1, 25
	return []plugins.ConfigField{
		{Key: "limit", Label: "Number of rows", Type: "int", Default: 5, Min: &min, Max: &max},
	}
}

type recentStatement struct {
	ID          string     `json:"id"`
	ActorName   *string    `json:"actorName,omitempty"`
	ActorMBox   *string    `json:"actorMbox,omitempty"`
	VerbID      *string    `json:"verbId,omitempty"`
	ObjectID    *string    `json:"objectId,omitempty"`
	TenantID    string     `json:"tenantId"`
	LRSName     *string    `json:"lrsName,omitempty"`
	ProjectName string     `json:"projectName"`
	ClientID    *string    `json:"clientId,omitempty"`
	ClientName  *string    `json:"clientName,omitempty"`
	StoredAt    time.Time  `json:"storedAt"`
	Timestamp   *time.Time `json:"timestamp,omitempty"`
}

func (RecentStatements) FetchData(ctx context.Context, db *pgxpool.Pool, orgID string, cfg map[string]any) (any, error) {
	limit := cfgInt(cfg, "limit", 5)
	rows, err := db.Query(ctx, fmt.Sprintf(`
		SELECT
			st.id,
			st.statement->'actor'->>'name',
			st.actor_mbox,
			st.verb_id,
			st.object_id,
			st.tenant_id,
			l.name,
			p.name,
			st.client_id::text,
			c.name,
			st.stored_at,
			(st.statement->>'timestamp')::timestamptz
		FROM statements st
		JOIN lrs l ON l.id = st.tenant_id
		JOIN projects p ON p.id = l.project_id
		LEFT JOIN clients c ON c.id = st.client_id
		WHERE p.org_id = $1
		ORDER BY st.stored_at DESC, st.id DESC
		LIMIT %d
	`, limit), orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]recentStatement, 0, limit)
	for rows.Next() {
		var s recentStatement
		var ts *time.Time
		if err := rows.Scan(
			&s.ID, &s.ActorName, &s.ActorMBox, &s.VerbID, &s.ObjectID,
			&s.TenantID, &s.LRSName, &s.ProjectName, &s.ClientID, &s.ClientName,
			&s.StoredAt, &ts,
		); err != nil {
			return nil, err
		}
		s.Timestamp = ts
		items = append(items, s)
	}
	return items, rows.Err()
}
