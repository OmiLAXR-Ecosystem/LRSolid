package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/SGoerzen/LRSolid/apps/api/internal/store"
)

func TestStatements_InsertAndQueryWithCursor(t *testing.T) {
	withTestDB(t, func(ctx context.Context, pool any) {
		db := pool.(*pgxpool.Pool) // typed in helper via interface, keep local tidy
		_ = db
	})
}

// typed wrapper (so we can keep helper signature stable)
type pgxpool struct{ *pgxpool.Pool }

func TestStatements_EndToEnd(t *testing.T) {
	withTestDB(t, func(ctx context.Context, pool *pgxpool.Pool) {
		ss := NewStatementStore(pool)

		t0 := time.Now().UTC().Add(-10 * time.Minute)
		t1 := time.Now().UTC().Add(-5 * time.Minute)

		stmtA := map[string]any{"id": "a", "actor": map[string]any{"mbox": "mailto:a@x"}, "verb": map[string]any{"id": "v1"}, "object": map[string]any{"id": "o1"}}
		stmtB := map[string]any{"id": "b", "actor": map[string]any{"mbox": "mailto:b@x"}, "verb": map[string]any{"id": "v1"}, "object": map[string]any{"id": "o1"}}
		rawA, _ := json.Marshal(stmtA)
		rawB, _ := json.Marshal(stmtB)

		err := ss.UpsertStatements(ctx, []store.Statement{
			{TenantID: "t1", ID: "a", StoredAt: t0, ActorMBox: strp("mailto:a@x"), VerbID: strp("v1"), ObjectID: strp("o1"), RawJSON: rawA},
			{TenantID: "t1", ID: "b", StoredAt: t1, ActorMBox: strp("mailto:b@x"), VerbID: strp("v1"), ObjectID: strp("o1"), RawJSON: rawB},
		})
		if err != nil {
			t.Fatalf("upsert: %v", err)
		}

		// Query newest first, limit 1 => should return b and cursor for next
		page1, err := ss.GetStatements(ctx, store.StatementFilter{
			TenantID:  "t1",
			Limit:     1,
			Ascending: false,
		})
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if len(page1.Statements) != 1 || page1.Statements[0].ID != "b" {
			t.Fatalf("unexpected page1: %+v", page1.Statements)
		}
		if page1.NextCursor == nil {
			t.Fatalf("expected cursor")
		}

		page2, err := ss.GetStatements(ctx, store.StatementFilter{
			TenantID:  "t1",
			Limit:     10,
			Ascending: false,
			Cursor:    page1.NextCursor,
		})
		if err != nil {
			t.Fatalf("get2: %v", err)
		}
		if len(page2.Statements) != 1 || page2.Statements[0].ID != "a" {
			t.Fatalf("unexpected page2: %+v", page2.Statements)
		}
	})
}

func strp(s string) *string { return &s }
