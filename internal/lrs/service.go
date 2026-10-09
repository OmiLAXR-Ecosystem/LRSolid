package lrs

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/SGoerzen/LRSolid/internal/store"
	"github.com/SGoerzen/LRSolid/internal/xapi"
)

type contextKey string

const dataStoreIDKey contextKey = "lrs_data_store_id"

// WithDataStoreID injects a data_store_id into the context so the Service
// can route to the correct StatementStore. Called by the HTTP layer.
func WithDataStoreID(ctx context.Context, dataStoreID string) context.Context {
	return context.WithValue(ctx, dataStoreIDKey, dataStoreID)
}

func dataStoreIDFromCtx(ctx context.Context) string {
	if id, ok := ctx.Value(dataStoreIDKey).(string); ok {
		return id
	}
	return ""
}

// StatementCounters updates control-plane totals for statements that are not
// stored in the primary database (the rollup worker never sees those rows).
type StatementCounters interface {
	Add(ctx context.Context, tenant store.TenantID, clientID *string, n int, bytes int64, at time.Time) error
}

// LimitChecker rejects a batch that would cross a quota on the tenant chain.
type LimitChecker interface {
	Check(ctx context.Context, tenant store.TenantID, clientID *string, count int, bytes int64) error
}

// ErrCannotVoidVoiding is returned when a statement targets another voiding statement.
var ErrCannotVoidVoiding = errors.New("cannot void a voiding statement")

type Service struct {
	registry      *store.Registry
	fallback      store.StatementStore // control-plane Postgres, used when no external store is selected
	control       store.StoreBundle
	counters      StatementCounters
	limits        LimitChecker
	attachmentDir string
}

// IngestOptions carries the xAPI version negotiation and the authority the LRS stamps on.
type IngestOptions struct {
	Version          string
	AuthorityName    string
	AuthorityHome    string
	DefineActivities bool
	StatementID      string
}

func NewService(registry *store.Registry, fallback store.StatementStore, counters StatementCounters, limits LimitChecker) *Service {
	return &Service{registry: registry, fallback: fallback, counters: counters, limits: limits}
}

// HasExternalStore reports whether statements for this data store live outside the control plane.
func (s *Service) HasExternalStore(dataStoreID string) bool {
	return s != nil && s.registry != nil && s.registry.Has(dataStoreID)
}

// ExternalStoreIDs lists connected external data stores.
func (s *Service) ExternalStoreIDs() []string {
	if s == nil || s.registry == nil {
		return nil
	}
	return s.registry.IDs()
}

// UseControlPlane registers the primary database's full xAPI bundle.
func (s *Service) UseControlPlane(b store.StoreBundle) {
	s.control = b
	if b.Statements != nil {
		s.fallback = b.Statements
	}
}

// SetAttachmentDir is the local directory used for attachment bytes.
func (s *Service) SetAttachmentDir(dir string) {
	s.attachmentDir = dir
}

// Bundle resolves the xAPI stores for this request.
func (s *Service) Bundle(ctx context.Context) store.StoreBundle {
	if s.registry != nil {
		if id := dataStoreIDFromCtx(ctx); id != "" {
			if bundle, err := s.registry.Get(id); err == nil && bundle.Statements != nil {
				return bundle
			}
		}
	}
	if s.control.Statements != nil {
		return s.control
	}
	return store.StoreBundle{Statements: s.fallback}
}

func (s *Service) statementsFor(ctx context.Context) store.StatementStore {
	if s.registry != nil {
		if id := dataStoreIDFromCtx(ctx); id != "" {
			if bundle, err := s.registry.Get(id); err == nil && bundle.Statements != nil {
				return bundle.Statements
			}
		}
	}
	return s.fallback
}

func (s *Service) IngestStatements(ctx context.Context, tenant store.TenantID, clientID *string, raw []json.RawMessage) ([]string, error) {
	return s.Ingest(ctx, tenant, clientID, raw, IngestOptions{})
}

func (s *Service) Ingest(ctx context.Context, tenant store.TenantID, clientID *string, raw []json.RawMessage, opt IngestOptions) ([]string, error) {
	now := time.Now().UTC()
	version := opt.Version
	if version == "" {
		version = xapi.Version103
	}
	if opt.StatementID != "" && len(raw) != 1 {
		return nil, &xapi.ValidationError{Reason: "PUT accepts a single statement"}
	}

	stmts := make([]store.Statement, 0, len(raw))
	ids := make([]string, 0, len(raw))
	var activities []xapi.ActivityDef
	seen := map[string]struct{}{}

	for _, r := range raw {
		prepared, err := xapi.Prepare(r, version, now, opt.AuthorityName, opt.AuthorityHome, opt.StatementID)
		if err != nil {
			return nil, err
		}
		st := prepared.Statement
		st.TenantID = tenant
		st.ClientID = clientID
		if _, ok := seen[string(st.ID)]; ok {
			return nil, &xapi.ValidationError{Reason: "duplicate statement id in batch"}
		}
		seen[string(st.ID)] = struct{}{}
		stmts = append(stmts, st)
		ids = append(ids, string(st.ID))
		activities = append(activities, prepared.Activities...)
	}

	target := s.statementsFor(ctx)
	voiding := map[string]struct{}{}
	for _, st := range stmts {
		if st.VoidsID != nil {
			voiding[string(st.ID)] = struct{}{}
		}
	}
	for _, st := range stmts {
		if st.VoidsID == nil {
			continue
		}
		if _, ok := voiding[string(*st.VoidsID)]; ok {
			return nil, ErrCannotVoidVoiding
		}
		existing, err := target.GetStatementByID(ctx, tenant, *st.VoidsID)
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			return nil, err
		}
		if err == nil && existing.VerbID != nil && *existing.VerbID == xapi.VerbVoided {
			return nil, ErrCannotVoidVoiding
		}
	}
	for i := range stmts {
		if stmts[i].VoidsID != nil {
			continue
		}
		voided, err := target.HasVoidingStatement(ctx, tenant, stmts[i].ID)
		if err != nil {
			return nil, err
		}
		stmts[i].Voided = voided
	}

	var totalBytes int64
	for _, st := range stmts {
		totalBytes += int64(len(st.RawJSON))
	}
	if s.limits != nil {
		if err := s.limits.Check(ctx, tenant, clientID, len(stmts), totalBytes); err != nil {
			return nil, err
		}
	}

	n, err := target.UpsertStatements(ctx, stmts)
	if err != nil {
		return nil, err
	}
	for _, st := range stmts {
		if st.VoidsID != nil {
			if err := target.MarkVoided(ctx, tenant, *st.VoidsID); err != nil {
				return nil, err
			}
		}
	}
	if opt.DefineActivities {
		if activitiesStore := s.Bundle(ctx).Activities; activitiesStore != nil {
			for _, act := range activities {
				_ = activitiesStore.PutActivity(ctx, tenant, act.ID, act.Definition)
			}
		}
	}
	if s.counters != nil && target != s.fallback && n > 0 {
		bytes := totalBytes
		if n < len(stmts) && len(stmts) > 0 {
			bytes = totalBytes * int64(n) / int64(len(stmts))
		}
		_ = s.counters.Add(ctx, tenant, clientID, n, bytes, now)
	}
	return ids, nil
}

// SaveAttachment writes attachment bytes and metadata for a tenant.
func (s *Service) SaveAttachment(ctx context.Context, tenant store.TenantID, sha, contentType string, body []byte) error {
	if s.attachmentDir == "" {
		return errors.New("attachment storage is not configured")
	}
	dir := filepath.Join(s.attachmentDir, string(tenant))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	path := filepath.Join(dir, sha)
	if err := os.WriteFile(path, body, 0o640); err != nil {
		return err
	}
	if at := s.Bundle(ctx).Attachments; at != nil {
		return at.UpsertAttachmentMeta(ctx, store.AttachmentMeta{
			TenantID:    tenant,
			SHA256:      sha,
			ContentType: contentType,
			Length:      int64(len(body)),
			Stored:      &store.AttachmentRef{Bucket: "local", Key: sha, Size: int64(len(body)), SHA256: sha},
			CreatedAt:   time.Now().UTC(),
		})
	}
	return nil
}

// ReadAttachment returns previously stored attachment bytes.
func (s *Service) ReadAttachment(tenant store.TenantID, sha string) ([]byte, error) {
	if s.attachmentDir == "" {
		return nil, os.ErrNotExist
	}
	return os.ReadFile(filepath.Join(s.attachmentDir, string(tenant), sha))
}

func (s *Service) QueryStatements(ctx context.Context, f store.StatementFilter) (store.StatementPage, error) {
	if f.Limit < 0 {
		f.Limit = 50
	}
	if f.Limit > 1000 {
		f.Limit = 1000
	}
	return s.statementsFor(ctx).GetStatements(ctx, f)
}

func (s *Service) GetStatementByID(ctx context.Context, tenant store.TenantID, id store.StatementID) (store.Statement, error) {
	return s.statementsFor(ctx).GetStatementByID(ctx, tenant, id)
}
