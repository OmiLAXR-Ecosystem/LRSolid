package storetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/SGoerzen/LRSolid/internal/store"
)

// RunStatementTests checks the canonical statement semantics against a provider.
func RunStatementTests(t *testing.T, ss store.StatementStore) {
	t.Helper()
	ctx := context.Background()
	tenant := store.TenantID("tenant-contract")
	t0 := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	t1 := t0.Add(time.Minute)
	rawA := []byte(`{"id":"a","actor":{"mbox":"mailto:a@x"},"verb":{"id":"http://adlnet.gov/expapi/verbs/experienced"},"object":{"id":"http://example.com/activities/1"}}`)
	rawB := []byte(`{"id":"b","actor":{"mbox":"mailto:b@x"},"verb":{"id":"http://adlnet.gov/expapi/verbs/completed"},"object":{"id":"http://example.com/activities/2"}}`)

	n, err := ss.UpsertStatements(ctx, []store.Statement{
		{TenantID: tenant, ID: "a", StoredAt: t0, ActorMBox: strp("mailto:a@x"), VerbID: strp("http://adlnet.gov/expapi/verbs/experienced"), ObjectID: strp("http://example.com/activities/1"), RawJSON: rawA},
		{TenantID: tenant, ID: "b", StoredAt: t1, ActorMBox: strp("mailto:b@x"), VerbID: strp("http://adlnet.gov/expapi/verbs/completed"), ObjectID: strp("http://example.com/activities/2"), RawJSON: rawB},
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if n != 2 {
		t.Fatalf("inserted = %d, want 2", n)
	}

	n, err = ss.UpsertStatements(ctx, []store.Statement{
		{TenantID: tenant, ID: "a", StoredAt: t0, RawJSON: rawA},
	})
	if err != nil {
		t.Fatalf("identical replay: %v", err)
	}
	if n != 0 {
		t.Fatalf("replay inserted = %d, want 0", n)
	}

	_, err = ss.UpsertStatements(ctx, []store.Statement{
		{TenantID: tenant, ID: "a", StoredAt: t0, RawJSON: []byte(`{"id":"a","changed":true}`)},
	})
	if !errors.Is(err, store.ErrAlreadyExists) {
		t.Fatalf("conflict err = %v, want ErrAlreadyExists", err)
	}

	got, err := ss.GetStatementByID(ctx, tenant, "a")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !store.JSONEqual(got.RawJSON, rawA) {
		t.Fatalf("stored json changed: %s", got.RawJSON)
	}

	mbox := "mailto:b@x"
	page, err := ss.GetStatements(ctx, store.StatementFilter{TenantID: tenant, ActorMBox: &mbox, Limit: 10})
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	if len(page.Statements) != 1 || page.Statements[0].ID != "b" {
		t.Fatalf("filter page: %+v", page.Statements)
	}

	page, err = ss.GetStatements(ctx, store.StatementFilter{TenantID: tenant, Limit: 1, Ascending: false})
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page.Statements) != 1 || page.Statements[0].ID != "b" || page.NextCursor == nil {
		t.Fatalf("newest page: %+v cursor=%v", page.Statements, page.NextCursor)
	}
	page2, err := ss.GetStatements(ctx, store.StatementFilter{TenantID: tenant, Limit: 10, Ascending: false, Cursor: page.NextCursor})
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2.Statements) != 1 || page2.Statements[0].ID != "a" {
		t.Fatalf("older page: %+v", page2.Statements)
	}

	verb := "http://adlnet.gov/expapi/verbs/voided"
	objType := "StatementRef"
	target := store.StatementID("a")
	rawVoid := []byte(`{"id":"v","verb":{"id":"http://adlnet.gov/expapi/verbs/voided"},"object":{"objectType":"StatementRef","id":"a"}}`)
	if _, err := ss.UpsertStatements(ctx, []store.Statement{{
		TenantID: tenant, ID: "v", StoredAt: t1.Add(time.Minute), VerbID: &verb, ObjectID: strp("a"), ObjectType: &objType, VoidsID: &target, RawJSON: rawVoid,
	}}); err != nil {
		t.Fatalf("void insert: %v", err)
	}
	if err := ss.MarkVoided(ctx, tenant, "a"); err != nil {
		t.Fatalf("mark voided: %v", err)
	}
	ok, err := ss.HasVoidingStatement(ctx, tenant, "a")
	if err != nil || !ok {
		t.Fatalf("has voiding = %v %v", ok, err)
	}
	page, err = ss.GetStatements(ctx, store.StatementFilter{TenantID: tenant, StatementID: strp("a"), Limit: 1})
	if err != nil {
		t.Fatalf("voided get: %v", err)
	}
	if len(page.Statements) != 0 {
		t.Fatalf("voided statement still visible: %+v", page.Statements)
	}
	page, err = ss.GetStatements(ctx, store.StatementFilter{TenantID: tenant, VoidedStatementID: strp("a"), Limit: 1})
	if err != nil {
		t.Fatalf("voidedStatementId: %v", err)
	}
	if len(page.Statements) != 1 || page.Statements[0].ID != "a" {
		t.Fatalf("voided lookup: %+v", page.Statements)
	}
}

func strp(s string) *string { return &s }
