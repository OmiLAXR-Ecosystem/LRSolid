ALTER TABLE statements ADD COLUMN IF NOT EXISTS activity_ids text[] NULL;

CREATE INDEX IF NOT EXISTS idx_statements_activity_ids
    ON statements USING gin (activity_ids);

CREATE TABLE IF NOT EXISTS activities (
    tenant_id   text        NOT NULL,
    id          text        NOT NULL,
    definition  jsonb       NOT NULL,
    updated_at  timestamptz NOT NULL,
    PRIMARY KEY (tenant_id, id)
);
