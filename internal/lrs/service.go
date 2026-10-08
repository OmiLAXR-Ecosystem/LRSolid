package lrs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/SGoerzen/LRSolid/internal/store"
)

type contextKey string

const dataStoreIDKey contextKey = "lrs_data_store_id"

// WithDataStoreID injects a data_store_id into the context so the Service
// can route to the correct StatementStore. Called by the HTTP layer.
func WithDataStoreID(ctx context.Context, dataStoreID string) context.Context {
	return context.WithValue(ctx, dataStoreIDKey, dataStoreID)
}

func dataStoreIDFromCtx(ctx context.Context) string {
	if id, ok := ctx.Value(dataStoreIDKey).(string); ok {
		return id
	}
	return ""
}

// StatementCounters updates control-plane totals for statements that are not
// stored in the primary database (the rollup worker never sees those rows).
type StatementCounters interface {
	Add(ctx context.Context, tenant store.TenantID, clientID *string, n int, bytes int64, at time.Time) error
}

// LimitChecker rejects a batch that would cross a quota on the tenant chain.
type LimitChecker interface {
	Check(ctx context.Context, tenant store.TenantID, clientID *string, count int, bytes int64) error
}

type Service struct {
	registry *store.Registry
	fallback store.StatementStore // control-plane Postgres, used when no external store is selected
	counters StatementCounters
	limits   LimitChecker
}

func NewService(registry *store.Registry, fallback store.StatementStore, counters StatementCounters, limits LimitChecker) *Service {
	return &Service{registry: registry, fallback: fallback, counters: counters, limits: limits}
}

// HasExternalStore reports whether statements for this data store live outside the control plane.
func (s *Service) HasExternalStore(dataStoreID string) bool {
	return s != nil && s.registry != nil && s.registry.Has(dataStoreID)
}

// ExternalStoreIDs lists connected external data stores.
func (s *Service) ExternalStoreIDs() []string {
	if s == nil || s.registry == nil {
		return nil
	}
	return s.registry.IDs()
}

func (s *Service) statementsFor(ctx context.Context) store.StatementStore {
	if s.registry != nil {
		if id := dataStoreIDFromCtx(ctx); id != "" {
			if bundle, err := s.registry.Get(id); err == nil && bundle.Statements != nil {
				return bundle.Statements
			}
		}
	}
	return s.fallback
}

func (s *Service) IngestStatements(ctx context.Context, tenant store.TenantID, clientID *string, raw []json.RawMessage) ([]string, error) {
	now := time.Now().UTC()
	stmts := make([]store.Statement, 0, len(raw))
	ids := make([]string, 0, len(raw))

	for _, r := range raw {
		// Minimal parse: id, actor.mbox, verb.id, object.id, timestamp
		var m map[string]any
		if err := json.Unmarshal(r, &m); err != nil {
			return nil, fmt.Errorf("invalid statement json: %w", err)
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
			ClientID: clientID,
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
		ids = append(ids, id)
	}

	var totalBytes int64
	for _, st := range stmts {
		totalBytes += int64(len(st.RawJSON))
	}
	if s.limits != nil {
		if err := s.limits.Check(ctx, tenant, clientID, len(stmts), totalBytes); err != nil {
			return nil, err
		}
	}

	target := s.statementsFor(ctx)
	n, err := target.UpsertStatements(ctx, stmts)
	if err != nil {
		return nil, err
	}
	if s.counters != nil && target != s.fallback && n > 0 {
		bytes := totalBytes
		if n < len(stmts) && len(stmts) > 0 {
			bytes = totalBytes * int64(n) / int64(len(stmts))
		}
		_ = s.counters.Add(ctx, tenant, clientID, n, bytes, now)
	}
	return ids, nil
}

func (s *Service) QueryStatements(ctx context.Context, f store.StatementFilter) (store.StatementPage, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 1000 {
		f.Limit = 1000
	}
	return s.statementsFor(ctx).GetStatements(ctx, f)
}

func (s *Service) GetStatementByID(ctx context.Context, tenant store.TenantID, id store.StatementID) (store.Statement, error) {
	return s.statementsFor(ctx).GetStatementByID(ctx, tenant, id)
}
