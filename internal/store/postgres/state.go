package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/apps/api/internal/store"
)

type StateStore struct {
	db *pgxpool.Pool
}

func NewStateStore(db *pgxpool.Pool) *StateStore {
	return &StateStore{db: db}
}

func (s *StateStore) PutState(ctx context.Context, key store.StateKey, content []byte, contentType string, ifMatch *store.ETag) (store.ETag, error) {
	now := time.Now().UTC()
	newETag := computeETag(content)

	// If-Match: update only if existing etag matches
	if ifMatch != nil {
		ct := nullableString(contentType)
		var updated int64
		err := s.db.QueryRow(ctx, `
			UPDATE state_docs
			SET content=$1, content_type=$2, etag=$3, updated_at=$4
			WHERE tenant_id=$5 AND activity_id=$6 AND agent_key=$7
			  AND (registration IS NOT DISTINCT FROM $8)
			  AND state_id=$9
			  AND etag=$10
		`, content, ct, newETag, now,
			key.TenantID, key.ActivityID, key.AgentKey, key.Registration, key.StateID, *ifMatch,
		).Scan(&updated)
		if err == pgx.ErrNoRows {
			// mismatch or missing
			return "", store.ErrPreconditionFailed
		}
		if err != nil {
			return "", err
		}
		return store.ETag(newETag), nil
	}

	// No If-Match: upsert
	ct := nullableString(contentType)
	_, err := s.db.Exec(ctx, `
		INSERT INTO state_docs(
			tenant_id, activity_id, agent_key, registration, state_id,
			content, content_type, etag, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (tenant_id, activity_id, agent_key, registration, state_id)
		DO UPDATE SET
			content=EXCLUDED.content,
			content_type=EXCLUDED.content_type,
			etag=EXCLUDED.etag,
			updated_at=EXCLUDED.updated_at
	`, key.TenantID, key.ActivityID, key.AgentKey, key.Registration, key.StateID,
		content, ct, newETag, now,
	)
	if err != nil {
		return "", err
	}
	return store.ETag(newETag), nil
}

func (s *StateStore) GetState(ctx context.Context, key store.StateKey) (store.StateDoc, bool, error) {
	var doc store.StateDoc
	doc.Key = key

	var ct *string
	var etag string
	var updated time.Time
	err := s.db.QueryRow(ctx, `
		SELECT content, content_type, etag, updated_at
		FROM state_docs
		WHERE tenant_id=$1 AND activity_id=$2 AND agent_key=$3
		  AND (registration IS NOT DISTINCT FROM $4)
		  AND state_id=$5
	`, key.TenantID, key.ActivityID, key.AgentKey, key.Registration, key.StateID).Scan(
		&doc.Content, &ct, &etag, &updated,
	)
	if err == pgx.ErrNoRows {
		return store.StateDoc{}, false, nil
	}
	if err != nil {
		return store.StateDoc{}, false, err
	}

	if ct != nil {
		doc.ContentType = *ct
	}
	doc.ETag = store.ETag(etag)
	doc.UpdatedAt = updated
	return doc, true, nil
}

func (s *StateStore) DeleteState(ctx context.Context, key store.StateKey) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM state_docs
		WHERE tenant_id=$1 AND activity_id=$2 AND agent_key=$3
		  AND (registration IS NOT DISTINCT FROM $4)
		  AND state_id=$5
	`, key.TenantID, key.ActivityID, key.AgentKey, key.Registration, key.StateID)
	return err
}

func (s *StateStore) ListStateIDs(ctx context.Context, tenant store.TenantID, activityID, agentKey string, registration *string, since *time.Time) ([]string, error) {
	args := []any{tenant, activityID, agentKey, registration}
	q := `
		SELECT state_id
		FROM state_docs
		WHERE tenant_id=$1 AND activity_id=$2 AND agent_key=$3
		  AND (registration IS NOT DISTINCT FROM $4)
	`
	if since != nil {
		q += " AND updated_at > $5"
		args = append(args, *since)
	}
	q += " ORDER BY state_id ASC"

	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func computeETag(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func nullableString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
