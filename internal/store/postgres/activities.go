package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/store"
)

type ActivityStore struct {
	db *pgxpool.Pool
}

func NewActivityStore(db *pgxpool.Pool) *ActivityStore {
	return &ActivityStore{db: db}
}

func (s *ActivityStore) PutActivity(ctx context.Context, tenant store.TenantID, id string, definition []byte) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO activities (tenant_id, id, definition, updated_at)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (tenant_id, id) DO UPDATE
		SET definition = EXCLUDED.definition, updated_at = EXCLUDED.updated_at
	`, tenant, id, definition, time.Now().UTC())
	return err
}

func (s *ActivityStore) GetActivity(ctx context.Context, tenant store.TenantID, id string) ([]byte, bool, error) {
	var def []byte
	err := s.db.QueryRow(ctx, `
		SELECT definition FROM activities WHERE tenant_id=$1 AND id=$2
	`, tenant, id).Scan(&def)
	if err == pgx.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return def, true, nil
}
