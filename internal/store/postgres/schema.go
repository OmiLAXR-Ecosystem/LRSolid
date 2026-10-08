package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EnsureDataPlane creates the xAPI tables an external PostgreSQL database needs.
// It is safe to run against a database that already has the control-plane migrations:
// existing tables are left in place.
func EnsureDataPlane(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, dataPlaneSQL)
	return err
}

const dataPlaneSQL = `
CREATE TABLE IF NOT EXISTS statements (
    tenant_id   text        NOT NULL,
    id          text        NOT NULL,
    stored_at   timestamptz NOT NULL,
    actor_mbox  text        NULL,
    verb_id     text        NULL,
    object_id   text        NULL,
    client_id   uuid        NULL,
    statement   jsonb       NOT NULL,
    PRIMARY KEY (tenant_id, id)
);

ALTER TABLE statements ADD COLUMN IF NOT EXISTS client_id uuid NULL;
ALTER TABLE statements ADD COLUMN IF NOT EXISTS actor_mbox text NULL;
ALTER TABLE statements ADD COLUMN IF NOT EXISTS verb_id text NULL;
ALTER TABLE statements ADD COLUMN IF NOT EXISTS object_id text NULL;

CREATE INDEX IF NOT EXISTS idx_statements_tenant_stored
    ON statements (tenant_id, stored_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_statements_actor_mbox
    ON statements (tenant_id, actor_mbox, stored_at DESC);
CREATE INDEX IF NOT EXISTS idx_statements_verb
    ON statements (tenant_id, verb_id, stored_at DESC);
CREATE INDEX IF NOT EXISTS idx_statements_object
    ON statements (tenant_id, object_id, stored_at DESC);

CREATE TABLE IF NOT EXISTS state_docs (
    tenant_id    text        NOT NULL,
    activity_id  text        NOT NULL,
    agent_key    text        NOT NULL,
    registration text        NOT NULL DEFAULT '',
    state_id     text        NOT NULL,
    content      bytea       NOT NULL,
    content_type text        NULL,
    etag         text        NOT NULL,
    updated_at   timestamptz NOT NULL,
    PRIMARY KEY (tenant_id, activity_id, agent_key, registration, state_id)
);

CREATE TABLE IF NOT EXISTS agent_profiles (
    tenant_id    text        NOT NULL,
    agent_key    text        NOT NULL,
    profile_id   text        NOT NULL,
    content      bytea       NOT NULL,
    content_type text        NULL,
    etag         text        NOT NULL,
    updated_at   timestamptz NOT NULL,
    PRIMARY KEY (tenant_id, agent_key, profile_id)
);

CREATE TABLE IF NOT EXISTS activity_profiles (
    tenant_id    text        NOT NULL,
    activity_id  text        NOT NULL,
    profile_id   text        NOT NULL,
    content      bytea       NOT NULL,
    content_type text        NULL,
    etag         text        NOT NULL,
    updated_at   timestamptz NOT NULL,
    PRIMARY KEY (tenant_id, activity_id, profile_id)
);

CREATE TABLE IF NOT EXISTS attachment_meta (
    tenant_id     text        NOT NULL,
    sha256        text        NOT NULL,
    content_type  text        NOT NULL,
    length        bigint      NOT NULL,
    file_url      text        NULL,
    stored_bucket text        NULL,
    stored_key    text        NULL,
    stored_size   bigint      NULL,
    stored_sha256 text        NULL,
    created_at    timestamptz NOT NULL,
    PRIMARY KEY (tenant_id, sha256)
);
`
