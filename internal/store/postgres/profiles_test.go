package postgres

import (
	"context"
	"testing"

	"github.com/SGoerzen/LRSolid/internal/store"
)

func TestProfiles_AgentAndActivity(t *testing.T) {
	withTestDB(t, func(ctx context.Context, pool *pgxpool.Pool) {
		aps := NewAgentProfileStore(pool)
		acps := NewActivityProfileStore(pool)

		ak := store.AgentProfileKey{TenantID: "t1", AgentKey: "agent1", ProfileID: "p1"}
		etag1, err := aps.PutAgentProfile(ctx, ak, []byte("a"), "text/plain", nil)
		if err != nil {
			t.Fatalf("put agent profile: %v", err)
		}
		doc, found, err := aps.GetAgentProfile(ctx, ak)
		if err != nil || !found || doc.ETag != etag1 {
			t.Fatalf("get agent profile failed")
		}

		x := store.ActivityProfileKey{TenantID: "t1", ActivityID: "act1", ProfileID: "p1"}
		etag2, err := acps.PutActivityProfile(ctx, x, []byte("b"), "text/plain", nil)
		if err != nil {
			t.Fatalf("put activity profile: %v", err)
		}
		doc2, found2, err := acps.GetActivityProfile(ctx, x)
		if err != nil || !found2 || doc2.ETag != etag2 {
			t.Fatalf("get activity profile failed")
		}
	})
}
