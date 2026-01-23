package postgres

import (
	"context"
	"testing"

	"github.com/SGoerzen/LRSolid/internal/store"
)

func TestAttachments_UpsertGet(t *testing.T) {
	withTestDB(t, func(ctx context.Context, pool *pgxpool.Pool) {
		as := NewAttachmentStore(pool)

		meta := store.AttachmentMeta{
			TenantID:    "t1",
			SHA256:      "abc",
			ContentType: "image/png",
			Length:      123,
			Stored: &store.AttachmentRef{
				Bucket: "lrsold",
				Key:    "abc.png",
				Size:   123,
				SHA256: "abc",
			},
		}
		if err := as.UpsertAttachmentMeta(ctx, meta); err != nil {
			t.Fatalf("upsert: %v", err)
		}

		got, found, err := as.GetAttachmentMeta(ctx, "t1", "abc")
		if err != nil || !found {
			t.Fatalf("get: found=%v err=%v", found, err)
		}
		if got.Stored == nil || got.Stored.Key != "abc.png" {
			t.Fatalf("stored ref missing")
		}
	})
}
