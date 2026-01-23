// internal/store/store.go
//
// Storage ports (interfaces) for LRSolid.
//
// Goals:
// - Keep xAPI semantics (sorting, paging, since/until) consistent in the core.
// - Allow multiple storage providers (Postgres now; MongoDB later) via adapters.
// - Avoid offset pagination; use cursor-based pagination for stable, scalable results.
package store

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

var (
	// ErrNotFound indicates a missing record.
	ErrNotFound = errors.New("not found")
	// ErrPreconditionFailed indicates an ETag mismatch (optimistic concurrency).
	ErrPreconditionFailed = errors.New("precondition failed")
	// ErrAlreadyExists indicates a uniqueness conflict (e.g., statement id already stored).
	ErrAlreadyExists = errors.New("already exists")
)

// TenantID identifies an isolated tenant/workspace.
type TenantID string

// StatementID is the xAPI statement id (UUID per spec; we store as string).
type StatementID string

// ETag is used for xAPI State/Profile optimistic concurrency.
type ETag string

// Statement is the storage representation for xAPI statements.
// Keep provider-agnostic fields that are useful for indexing/querying.
// RawJSON must contain the full original xAPI statement.
type Statement struct {
	TenantID  TenantID
	ID        StatementID
	StoredAt  time.Time // when the LRS stored it (used for since/until + stable paging)
	Timestamp *time.Time

	ActorMBox   *string // e.g., "mailto:alice@example.org"
	ActorSHA256 *string // optional: derived canonical hash for privacy-friendly queries

	VerbID   *string // e.g., "http://adlnet.gov/expapi/verbs/experienced"
	ObjectID *string // activity id (IRI)

	Registration *string // UUID string (optional)

	RawJSON []byte
}

// StatementFilter models the canonical query semantics used by the core.
// Providers MUST implement identical ordering and cursor behavior.
type StatementFilter struct {
	TenantID TenantID

	ActorMBox    *string
	ActorSHA256  *string // preferred for privacy-safe filters
	VerbID       *string
	ObjectID     *string
	Registration *string

	// StoredSince/StoredUntil are applied to StoredAt.
	StoredSince *time.Time
	StoredUntil *time.Time

	// Limit is required; the API layer should clamp to a safe max (e.g., 100).
	Limit int

	// Ascending controls sort order by (stored_at, id).
	// If false, results are returned descending (newest first).
	Ascending bool

	// Cursor is an opaque token that encodes the last seen (stored_at, id).
	// It must be treated as an implementation detail of the core semantics,
	// not a provider-specific token.
	Cursor *string
}

// StatementPage represents one page of statements and an optional cursor for the next page.
type StatementPage struct {
	Statements []Statement
	NextCursor *string
}

// StatementStore is the port for xAPI Statements API persistence and queries.
type StatementStore interface {
	// UpsertStatements stores statements idempotently.
	// - If a statement with the same (tenant,id) exists with identical RawJSON, it must be treated as success.
	// - If it exists but differs, provider may return ErrAlreadyExists (spec-wise this is a conflict).
	UpsertStatements(ctx context.Context, stmts []Statement) error

	// GetStatements returns a stable-ordered page according to StatementFilter.
	// Ordering MUST be by (StoredAt, ID) with Ascending controlling direction.
	GetStatements(ctx context.Context, f StatementFilter) (StatementPage, error)

	// GetStatementByID returns a single statement by id.
	GetStatementByID(ctx context.Context, tenant TenantID, id StatementID) (Statement, error)
}

// ---- xAPI State API ----

// StateKey identifies a State document uniquely according to xAPI.
type StateKey struct {
	TenantID TenantID

	ActivityID   string
	AgentKey     string // canonicalized agent identity (e.g., hash of actor representation)
	Registration *string

	StateID string
}

// StateDoc holds the stored state payload and metadata.
type StateDoc struct {
	Key         StateKey
	Content     []byte
	ContentType string // optional (e.g., "application/json")
	ETag        ETag
	UpdatedAt   time.Time
}

// StateStore is the port for xAPI State API persistence.
type StateStore interface {
	// PutState creates or updates state.
	// If ifMatch is provided, it must match existing ETag; otherwise return ErrPreconditionFailed.
	// Returns new ETag.
	PutState(ctx context.Context, key StateKey, content []byte, contentType string, ifMatch *ETag) (ETag, error)

	// GetState returns the state document; if missing returns (StateDoc{}, false, nil).
	GetState(ctx context.Context, key StateKey) (StateDoc, bool, error)

	// DeleteState deletes a state document (idempotent).
	DeleteState(ctx context.Context, key StateKey) error

	// ListStateIDs lists existing StateIDs for the given (activity, agent, registration).
	ListStateIDs(ctx context.Context, tenant TenantID, activityID, agentKey string, registration *string, since *time.Time) ([]string, error)
}

// ---- xAPI Agent/Activity Profile APIs ----

type AgentProfileKey struct {
	TenantID  TenantID
	AgentKey  string
	ProfileID string
}

type ActivityProfileKey struct {
	TenantID   TenantID
	ActivityID string
	ProfileID  string
}

type ProfileDoc struct {
	Content     []byte
	ContentType string
	ETag        ETag
	UpdatedAt   time.Time
}

type AgentProfileStore interface {
	PutAgentProfile(ctx context.Context, key AgentProfileKey, content []byte, contentType string, ifMatch *ETag) (ETag, error)
	GetAgentProfile(ctx context.Context, key AgentProfileKey) (ProfileDoc, bool, error)
	DeleteAgentProfile(ctx context.Context, key AgentProfileKey) error
	ListAgentProfileIDs(ctx context.Context, tenant TenantID, agentKey string, since *time.Time) ([]string, error)
}

type ActivityProfileStore interface {
	PutActivityProfile(ctx context.Context, key ActivityProfileKey, content []byte, contentType string, ifMatch *ETag) (ETag, error)
	GetActivityProfile(ctx context.Context, key ActivityProfileKey) (ProfileDoc, bool, error)
	DeleteActivityProfile(ctx context.Context, key ActivityProfileKey) error
	ListActivityProfileIDs(ctx context.Context, tenant TenantID, activityID string, since *time.Time) ([]string, error)
}

// ---- Attachments (metadata only; payload lives in S3/MinIO or similar) ----

// AttachmentRef identifies an attachment payload in object storage.
type AttachmentRef struct {
	Bucket string
	Key    string
	Size   int64
	SHA256 string
}

type AttachmentMeta struct {
	TenantID TenantID

	// xAPI attachment fields
	SHA256      string
	ContentType string
	Length      int64
	FileURL     *string // optional external URL

	// Internal storage reference (if stored by us)
	Stored *AttachmentRef

	CreatedAt time.Time
}

type AttachmentStore interface {
	// UpsertAttachmentMeta stores metadata idempotently by (tenant, sha256).
	UpsertAttachmentMeta(ctx context.Context, meta AttachmentMeta) error
	GetAttachmentMeta(ctx context.Context, tenant TenantID, sha256 string) (AttachmentMeta, bool, error)
}

// ---- Cursor helpers (core semantic) ----

// Cursor encodes a paging position (stored_at, id).
type Cursor struct {
	StoredAt time.Time
	ID       StatementID
}

// EncodeCursor creates an opaque token from storedAt + id.
// This token format belongs to the core semantics and must remain stable across providers.
func EncodeCursor(c Cursor) string {
	plain := fmt.Sprintf("%d|%s", c.StoredAt.UnixNano(), string(c.ID))
	return base64.RawURLEncoding.EncodeToString([]byte(plain))
}

// DecodeCursor parses an opaque token into Cursor.
func DecodeCursor(token string) (Cursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return Cursor{}, fmt.Errorf("invalid cursor encoding: %w", err)
	}
	var ts int64
	var id string
	_, err = fmt.Sscanf(string(b), "%d|%s", &ts, &id)
	if err != nil {
		return Cursor{}, fmt.Errorf("invalid cursor format: %w", err)
	}
	return Cursor{StoredAt: time.Unix(0, ts).UTC(), ID: StatementID(id)}, nil
}

// HashAgentKey generates a stable identifier for an Agent (Actor) representation.
// Use this to avoid storing raw PII (like mbox) when desired.
// NOTE: This is a helper; canonicalization should happen at the xAPI layer.
func HashAgentKey(canonicalAgent string) string {
	sum := sha256.Sum256([]byte(canonicalAgent))
	return hex.EncodeToString(sum[:])
}

// HashMBox returns sha256(mbox) as hex, useful for privacy-safe lookup.
func HashMBox(mbox string) string {
	sum := sha256.Sum256([]byte(mbox))
	return hex.EncodeToString(sum[:])
}
