package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/store"
)

type poolLifecycle struct {
	pool *pgxpool.Pool
}

func (p *poolLifecycle) Close() error {
	p.pool.Close()
	return nil
}

func (p *poolLifecycle) Health(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// NewStoreBundle is a store.StoreFactory for PostgreSQL.
func NewStoreBundle(ctx context.Context, cfg store.DataStoreConfig) (store.StoreBundle, error) {
	db, err := Open(ctx, cfg.ConnectionString)
	if err != nil {
		return store.StoreBundle{}, fmt.Errorf("postgres connect: %w", err)
	}
	if err := EnsureDataPlane(ctx, db); err != nil {
		db.Close()
		return store.StoreBundle{}, fmt.Errorf("postgres schema: %w", err)
	}

	return store.StoreBundle{
		Statements:       NewStatementStore(db),
		States:           NewStateStore(db),
		AgentProfiles:    NewAgentProfileStore(db),
		ActivityProfiles: NewActivityProfileStore(db),
		Attachments:      NewAttachmentStore(db),
		Activities:       NewActivityStore(db),
		Lifecycle:        &poolLifecycle{pool: db},
		DatabaseType:     "postgres",
		Capabilities:     store.ExternalCapabilities(),
	}, nil
}
