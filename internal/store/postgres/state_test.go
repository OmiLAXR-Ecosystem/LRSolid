package postgres

import (
	"context"
	"testing"

	"github.com/SGoerzen/LRSolid/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestState_PutGetETag(t *testing.T) {
	withTestDB(t, func(ctx context.Context, pool *pgxpool.Pool) {
		ss := NewStateStore(pool)

		key := store.StateKey{
			TenantID:   "t1",
			ActivityID: "act1",
			AgentKey:   "agent1",
			StateID:    "s1",
		}

		etag1, err := ss.PutState(ctx, key, []byte("hello"), "text/plain", nil)
		if err != nil {
			t.Fatalf("put: %v", err)
		}

		doc, found, err := ss.GetState(ctx, key)
		if err != nil || !found {
			t.Fatalf("get: found=%v err=%v", found, err)
		}
		if doc.ETag != etag1 {
			t.Fatalf("etag mismatch")
		}

		// If-Match mismatch
		bad := store.ETag("deadbeef")
		_, err = ss.PutState(ctx, key, []byte("world"), "text/plain", &bad)
		if err != store.ErrPreconditionFailed {
			t.Fatalf("expected precondition failed, got %v", err)
		}

		// If-Match ok
		etag2, err := ss.PutState(ctx, key, []byte("world"), "text/plain", &etag1)
		if err != nil {
			t.Fatalf("put2: %v", err)
		}
		if etag2 == etag1 {
			t.Fatalf("expected new etag")
		}
	})
}
