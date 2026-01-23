package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SGoerzen/LRSolid/internal/store"
)

type AttachmentStore struct {
	db *pgxpool.Pool
}

func NewAttachmentStore(db *pgxpool.Pool) *AttachmentStore {
	return &AttachmentStore{db: db}
}

func (s *AttachmentStore) UpsertAttachmentMeta(ctx context.Context, meta store.AttachmentMeta) error {
	now := time.Now().UTC()
	meta.CreatedAt = now

	var storedBucket, storedKey *string
	var storedSize *int64
	var storedSHA *string
	if meta.Stored != nil {
		storedBucket = &meta.Stored.Bucket
		storedKey = &meta.Stored.Key
		storedSize = &meta.Stored.Size
		storedSHA = &meta.Stored.SHA256
	}

	_, err := s.db.Exec(ctx, `
		INSERT INTO attachment_meta(
			tenant_id, sha256, content_type, length, file_url,
			stored_bucket, stored_key, stored_size, stored_sha256, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (tenant_id, sha256) DO UPDATE SET
			content_type=EXCLUDED.content_type,
			length=EXCLUDED.length,
			file_url=EXCLUDED.file_url,
			stored_bucket=EXCLUDED.stored_bucket,
			stored_key=EXCLUDED.stored_key,
			stored_size=EXCLUDED.stored_size,
			stored_sha256=EXCLUDED.stored_sha256
	`, meta.TenantID, meta.SHA256, meta.ContentType, meta.Length, meta.FileURL,
		storedBucket, storedKey, storedSize, storedSHA, meta.CreatedAt,
	)
	return err
}

func (s *AttachmentStore) GetAttachmentMeta(ctx context.Context, tenant store.TenantID, sha256 string) (store.AttachmentMeta, bool, error) {
	var meta store.AttachmentMeta
	meta.TenantID = tenant
	meta.SHA256 = sha256

	var fileURL *string
	var storedBucket, storedKey, storedSHA *string
	var storedSize *int64

	err := s.db.QueryRow(ctx, `
		SELECT content_type, length, file_url,
		       stored_bucket, stored_key, stored_size, stored_sha256,
		       created_at
		FROM attachment_meta
		WHERE tenant_id=$1 AND sha256=$2
	`, tenant, sha256).Scan(
		&meta.ContentType, &meta.Length, &fileURL,
		&storedBucket, &storedKey, &storedSize, &storedSHA,
		&meta.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return store.AttachmentMeta{}, false, nil
	}
	if err != nil {
		return store.AttachmentMeta{}, false, err
	}

	meta.FileURL = fileURL
	if storedBucket != nil && storedKey != nil && storedSize != nil && storedSHA != nil {
		meta.Stored = &store.AttachmentRef{
			Bucket: *storedBucket,
			Key:    *storedKey,
			Size:   *storedSize,
			SHA256: *storedSHA,
		}
	}

	return meta, true, nil
}
