package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/SGoerzen/LRSolid/internal/store"
)

type AttachmentStore struct {
	col *mongo.Collection
}

func NewAttachmentStore(db *mongo.Database) *AttachmentStore {
	return &AttachmentStore{col: db.Collection(colAttachmentMeta)}
}

type attachmentDoc struct {
	DocID       string  `bson:"_id"`
	TenantID    string  `bson:"tenant_id"`
	SHA256      string  `bson:"sha256"`
	ContentType string  `bson:"content_type"`
	Length      int64   `bson:"length"`
	FileURL     *string `bson:"file_url,omitempty"`

	StoredBucket *string `bson:"stored_bucket,omitempty"`
	StoredKey    *string `bson:"stored_key,omitempty"`
	StoredSize   *int64  `bson:"stored_size,omitempty"`
	StoredSHA256 *string `bson:"stored_sha256,omitempty"`

	CreatedAt time.Time `bson:"created_at"`
}

func (s *AttachmentStore) UpsertAttachmentMeta(ctx context.Context, meta store.AttachmentMeta) error {
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now().UTC()
	}

	doc := attachmentDoc{
		DocID:       string(meta.TenantID) + "|" + meta.SHA256,
		TenantID:    string(meta.TenantID),
		SHA256:      meta.SHA256,
		ContentType: meta.ContentType,
		Length:      meta.Length,
		FileURL:     meta.FileURL,
		CreatedAt:   meta.CreatedAt,
	}
	if meta.Stored != nil {
		doc.StoredBucket = &meta.Stored.Bucket
		doc.StoredKey = &meta.Stored.Key
		doc.StoredSize = &meta.Stored.Size
		doc.StoredSHA256 = &meta.Stored.SHA256
	}

	_, err := s.col.ReplaceOne(ctx,
		bson.D{{Key: "_id", Value: doc.DocID}},
		doc,
		options.Replace().SetUpsert(true),
	)
	return err
}

func (s *AttachmentStore) GetAttachmentMeta(ctx context.Context, tenant store.TenantID, sha256 string) (store.AttachmentMeta, bool, error) {
	docID := string(tenant) + "|" + sha256
	var doc attachmentDoc
	err := s.col.FindOne(ctx, bson.D{{Key: "_id", Value: docID}}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return store.AttachmentMeta{}, false, nil
	}
	if err != nil {
		return store.AttachmentMeta{}, false, err
	}

	meta := store.AttachmentMeta{
		TenantID:    tenant,
		SHA256:      doc.SHA256,
		ContentType: doc.ContentType,
		Length:      doc.Length,
		FileURL:     doc.FileURL,
		CreatedAt:   doc.CreatedAt,
	}
	if doc.StoredBucket != nil && doc.StoredKey != nil && doc.StoredSize != nil && doc.StoredSHA256 != nil {
		meta.Stored = &store.AttachmentRef{
			Bucket: *doc.StoredBucket,
			Key:    *doc.StoredKey,
			Size:   *doc.StoredSize,
			SHA256: *doc.StoredSHA256,
		}
	}
	return meta, true, nil
}
