package postgres

import (
	"context"
	"fmt"

	"github.com/SGoerzen/LRSolid/internal/store"
)

// NewStoreBundle is a store.StoreFactory for PostgreSQL.
func NewStoreBundle(ctx context.Context, cfg store.DataStoreConfig) (store.StoreBundle, error) {
	db, err := Open(ctx, cfg.ConnectionString)
	if err != nil {
		return store.StoreBundle{}, fmt.Errorf("postgres connect: %w", err)
	}

	return store.StoreBundle{
		Statements:       NewStatementStore(db),
		States:           NewStateStore(db),
		AgentProfiles:    NewAgentProfileStore(db),
		ActivityProfiles: NewActivityProfileStore(db),
		Attachments:      NewAttachmentStore(db),
	}, nil
}
