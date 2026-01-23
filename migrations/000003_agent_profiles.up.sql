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

CREATE INDEX IF NOT EXISTS idx_agent_profiles_updated
    ON agent_profiles (tenant_id, updated_at DESC);