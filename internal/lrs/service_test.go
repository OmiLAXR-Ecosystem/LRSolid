package lrs

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/SGoerzen/LRSolid/internal/store"
)

type fakeStmtStore struct {
	last []store.Statement
}

func (f *fakeStmtStore) UpsertStatements(ctx context.Context, stmts []store.Statement) (int, error) {
	f.last = append([]store.Statement(nil), stmts...)
	return len(stmts), nil
}

func (f *fakeStmtStore) GetStatements(ctx context.Context, flt store.StatementFilter) (store.StatementPage, error) {
	return store.StatementPage{}, nil
}

func (f *fakeStmtStore) GetStatementByID(ctx context.Context, tenant store.TenantID, id store.StatementID) (store.Statement, error) {
	return store.Statement{}, store.ErrNotFound
}

func TestIngestStatements_AssignsIDAndExtractsFields(t *testing.T) {
	fs := &fakeStmtStore{}
	svc := NewService(nil, fs, nil, nil)

	stmt := map[string]any{
		"actor":  map[string]any{"mbox": "mailto:alice@example.org"},
		"verb":   map[string]any{"id": "http://adlnet.gov/expapi/verbs/experienced"},
		"object": map[string]any{"id": "https://example.org/activity/1"},
	}
	raw, _ := json.Marshal(stmt)

	ids, err := svc.IngestStatements(context.Background(), store.TenantID("t1"), nil, []json.RawMessage{raw})
	if err != nil {
		t.Fatalf("ingest error: %v", err)
	}
	if len(ids) != 1 || ids[0] == "" {
		t.Fatalf("expected one returned id")
	}

	if len(fs.last) != 1 {
		t.Fatalf("expected 1 stored statement, got %d", len(fs.last))
	}
	if fs.last[0].ID == "" {
		t.Fatalf("expected generated id")
	}
	if fs.last[0].ActorMBox == nil || *fs.last[0].ActorMBox != "mailto:alice@example.org" {
		t.Fatalf("actor mbox not extracted")
	}
	if fs.last[0].VerbID == nil || *fs.last[0].VerbID != "http://adlnet.gov/expapi/verbs/experienced" {
		t.Fatalf("verb id not extracted")
	}
	if fs.last[0].ObjectID == nil || *fs.last[0].ObjectID != "https://example.org/activity/1" {
		t.Fatalf("object id not extracted")
	}
}

type denyLimits struct{}

func (denyLimits) Check(context.Context, store.TenantID, *string, int, int64) error {
	return errors.New("limited")
}

func TestIngestStatements_LimitBlocksStore(t *testing.T) {
	fs := &fakeStmtStore{}
	svc := NewService(nil, fs, nil, denyLimits{})
	raw := json.RawMessage(`{"actor":{"mbox":"mailto:a@b.c"},"verb":{"id":"http://example.com/v"},"object":{"id":"http://example.com/a"}}`)
	_, err := svc.IngestStatements(context.Background(), store.TenantID("t1"), nil, []json.RawMessage{raw})
	if err == nil {
		t.Fatal("expected limit error")
	}
	if len(fs.last) != 0 {
		t.Fatal("statement stored despite limit")
	}
}

func TestIngestStatements_InvalidJSON(t *testing.T) {
	fs := &fakeStmtStore{}
	svc := NewService(nil, fs, nil, nil)

	bad := json.RawMessage([]byte("{not-json"))
	_, err := svc.IngestStatements(context.Background(), store.TenantID("t1"), nil, []json.RawMessage{bad})
	if err == nil {
		t.Fatalf("expected error")
	}
}
