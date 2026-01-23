package lrs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/SGoerzen/LRSolid/apps/api/internal/store"
)

type Service struct {
	statements store.StatementStore
}

func NewService(statements store.StatementStore) *Service {
	return &Service{statements: statements}
}

func (s *Service) IngestStatements(ctx context.Context, tenant store.TenantID, raw []json.RawMessage) error {
	now := time.Now().UTC()
	stmts := make([]store.Statement, 0, len(raw))

	for _, r := range raw {
		// Minimal parse: id, actor.mbox, verb.id, object.id, timestamp
		var m map[string]any
		if err := json.Unmarshal(r, &m); err != nil {
			return fmt.Errorf("invalid statement json: %w", err)
		}

		id := ""
		if v, ok := m["id"].(string); ok && v != "" {
			id = v
		} else {
			// allow server generated id (MVP)
			id = uuid.NewString()
			m["id"] = id
			rebuilt, _ := json.Marshal(m)
			r = rebuilt
		}

		stmt := store.Statement{
			TenantID: tenant,
			ID:       store.StatementID(id),
			StoredAt: now,
			RawJSON:  []byte(r),
		}

		// actor.mbox
		if actor, ok := m["actor"].(map[string]any); ok {
			if mb, ok := actor["mbox"].(string); ok && mb != "" {
				stmt.ActorMBox = &mb
			}
		}

		// verb.id
		if verb, ok := m["verb"].(map[string]any); ok {
			if vid, ok := verb["id"].(string); ok && vid != "" {
				stmt.VerbID = &vid
			}
		}

		// object.id
		if obj, ok := m["object"].(map[string]any); ok {
			if oid, ok := obj["id"].(string); ok && oid != "" {
				stmt.ObjectID = &oid
			}
		}

		// timestamp (optional)
		if ts, ok := m["timestamp"].(string); ok && ts != "" {
			if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
				stmt.Timestamp = &t
			}
		}

		stmts = append(stmts, stmt)
	}

	return s.statements.UpsertStatements(ctx, stmts)
}

func (s *Service) QueryStatements(ctx context.Context, f store.StatementFilter) (store.StatementPage, error) {
	// Clamp limit at API layer ideally; still safe-guard here:
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 1000 {
		f.Limit = 1000
	}
	return s.statements.GetStatements(ctx, f)
}
