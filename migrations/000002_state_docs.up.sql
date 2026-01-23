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

CREATE INDEX IF NOT EXISTS idx_state_docs_lookup
    ON state_docs (tenant_id, activity_id, agent_key, registration);

CREATE INDEX IF NOT EXISTS idx_state_docs_updated
    ON state_docs (tenant_id, updated_at DESC);