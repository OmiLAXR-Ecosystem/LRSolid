package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/store"
)

type AgentProfileStore struct{ db *pgxpool.Pool }
type ActivityProfileStore struct{ db *pgxpool.Pool }

func NewAgentProfileStore(db *pgxpool.Pool) *AgentProfileStore { return &AgentProfileStore{db: db} }
func NewActivityProfileStore(db *pgxpool.Pool) *ActivityProfileStore {
	return &ActivityProfileStore{db: db}
}

func (s *AgentProfileStore) PutAgentProfile(ctx context.Context, key store.AgentProfileKey, content []byte, contentType string, ifMatch *store.ETag) (store.ETag, error) {
	return putProfile(ctx, s.db, "agent_profiles", key.TenantID, key.AgentKey, "", key.ProfileID, content, contentType, ifMatch)
}

func (s *AgentProfileStore) GetAgentProfile(ctx context.Context, key store.AgentProfileKey) (store.ProfileDoc, bool, error) {
	return getProfile(ctx, s.db, "agent_profiles", key.TenantID, key.AgentKey, "", key.ProfileID)
}

func (s *AgentProfileStore) DeleteAgentProfile(ctx context.Context, key store.AgentProfileKey) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM agent_profiles
		WHERE tenant_id=$1 AND agent_key=$2 AND profile_id=$3
	`, key.TenantID, key.AgentKey, key.ProfileID)
	return err
}

func (s *AgentProfileStore) ListAgentProfileIDs(ctx context.Context, tenant store.TenantID, agentKey string, since *time.Time) ([]string, error) {
	return listProfileIDs(ctx, s.db, "agent_profiles", tenant, agentKey, "", since)
}

func (s *ActivityProfileStore) PutActivityProfile(ctx context.Context, key store.ActivityProfileKey, content []byte, contentType string, ifMatch *store.ETag) (store.ETag, error) {
	return putProfile(ctx, s.db, "activity_profiles", key.TenantID, "", key.ActivityID, key.ProfileID, content, contentType, ifMatch)
}

func (s *ActivityProfileStore) GetActivityProfile(ctx context.Context, key store.ActivityProfileKey) (store.ProfileDoc, bool, error) {
	return getProfile(ctx, s.db, "activity_profiles", key.TenantID, "", key.ActivityID, key.ProfileID)
}

func (s *ActivityProfileStore) DeleteActivityProfile(ctx context.Context, key store.ActivityProfileKey) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM activity_profiles
		WHERE tenant_id=$1 AND activity_id=$2 AND profile_id=$3
	`, key.TenantID, key.ActivityID, key.ProfileID)
	return err
}

func (s *ActivityProfileStore) ListActivityProfileIDs(ctx context.Context, tenant store.TenantID, activityID string, since *time.Time) ([]string, error) {
	return listProfileIDs(ctx, s.db, "activity_profiles", tenant, "", activityID, since)
}

// shared helpers

func putProfile(ctx context.Context, db *pgxpool.Pool, table string, tenant store.TenantID, agentKey, activityID, profileID string,
	content []byte, contentType string, ifMatch *store.ETag) (store.ETag, error) {

	now := time.Now().UTC()
	newETag := etagOf(content)
	ct := nullableString(contentType)

	// If-Match update
	if ifMatch != nil {
		var updated int64
		var q string
		if table == "agent_profiles" {
			q = `
				UPDATE agent_profiles
				SET content=$1, content_type=$2, etag=$3, updated_at=$4
				WHERE tenant_id=$5 AND agent_key=$6 AND profile_id=$7 AND etag=$8
			`
			err := db.QueryRow(ctx, q, content, ct, newETag, now, tenant, agentKey, profileID, *ifMatch).Scan(&updated)
			if err == pgx.ErrNoRows {
				return "", store.ErrPreconditionFailed
			}
			if err != nil {
				return "", err
			}
			return store.ETag(newETag), nil
		}

		q = `
			UPDATE activity_profiles
			SET content=$1, content_type=$2, etag=$3, updated_at=$4
			WHERE tenant_id=$5 AND activity_id=$6 AND profile_id=$7 AND etag=$8
		`
		err := db.QueryRow(ctx, q, content, ct, newETag, now, tenant, activityID, profileID, *ifMatch).Scan(&updated)
		if err == pgx.ErrNoRows {
			return "", store.ErrPreconditionFailed
		}
		if err != nil {
			return "", err
		}
		return store.ETag(newETag), nil
	}

	// Upsert
	if table == "agent_profiles" {
		_, err := db.Exec(ctx, `
			INSERT INTO agent_profiles(tenant_id, agent_key, profile_id, content, content_type, etag, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
			ON CONFLICT (tenant_id, agent_key, profile_id)
			DO UPDATE SET content=EXCLUDED.content, content_type=EXCLUDED.content_type, etag=EXCLUDED.etag, updated_at=EXCLUDED.updated_at
		`, tenant, agentKey, profileID, content, ct, newETag, now)
		if err != nil {
			return "", err
		}
		return store.ETag(newETag), nil
	}

	_, err := db.Exec(ctx, `
		INSERT INTO activity_profiles(tenant_id, activity_id, profile_id, content, content_type, etag, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (tenant_id, activity_id, profile_id)
		DO UPDATE SET content=EXCLUDED.content, content_type=EXCLUDED.content_type, etag=EXCLUDED.etag, updated_at=EXCLUDED.updated_at
	`, tenant, activityID, profileID, content, ct, newETag, now)
	if err != nil {
		return "", err
	}
	return store.ETag(newETag), nil
}

func getProfile(ctx context.Context, db *pgxpool.Pool, table string, tenant store.TenantID, agentKey, activityID, profileID string) (store.ProfileDoc, bool, error) {
	var doc store.ProfileDoc
	var ct *string
	var etag string
	var updated time.Time

	if table == "agent_profiles" {
		err := db.QueryRow(ctx, `
			SELECT content, content_type, etag, updated_at
			FROM agent_profiles
			WHERE tenant_id=$1 AND agent_key=$2 AND profile_id=$3
		`, tenant, agentKey, profileID).Scan(&doc.Content, &ct, &etag, &updated)
		if err == pgx.ErrNoRows {
			return store.ProfileDoc{}, false, nil
		}
		if err != nil {
			return store.ProfileDoc{}, false, err
		}
	} else {
		err := db.QueryRow(ctx, `
			SELECT content, content_type, etag, updated_at
			FROM activity_profiles
			WHERE tenant_id=$1 AND activity_id=$2 AND profile_id=$3
		`, tenant, activityID, profileID).Scan(&doc.Content, &ct, &etag, &updated)
		if err == pgx.ErrNoRows {
			return store.ProfileDoc{}, false, nil
		}
		if err != nil {
			return store.ProfileDoc{}, false, err
		}
	}

	if ct != nil {
		doc.ContentType = *ct
	}
	doc.ETag = store.ETag(etag)
	doc.UpdatedAt = updated
	return doc, true, nil
}

func listProfileIDs(ctx context.Context, db *pgxpool.Pool, table string, tenant store.TenantID, agentKey, activityID string, since *time.Time) ([]string, error) {
	args := []any{tenant}
	q := "SELECT profile_id FROM " + table + " WHERE tenant_id=$1"

	if table == "agent_profiles" {
		q += " AND agent_key=$2"
		args = append(args, agentKey)
	} else {
		q += " AND activity_id=$2"
		args = append(args, activityID)
	}

	if since != nil {
		q += " AND updated_at > $3"
		args = append(args, *since)
	}

	q += " ORDER BY profile_id ASC"

	rows, err := db.Query(ctx, q, args...)
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

func etagOf(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
