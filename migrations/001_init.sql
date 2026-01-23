CREATE TABLE IF NOT EXISTS statements (
tenant_id   text        NOT NULL,
id          text        NOT NULL,
stored_at   timestamptz NOT NULL,
actor_mbox  text        NULL,
verb_id     text        NULL,
object_id   text        NULL,
statement   jsonb       NOT NULL,
PRIMARY KEY (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_statements_tenant_stored ON statements(tenant_id, stored_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_statements_actor ON statements(tenant_id, actor_mbox, stored_at DESC);
CREATE INDEX IF NOT EXISTS idx_statements_verb  ON statements(tenant_id, verb_id, stored_at DESC);
CREATE INDEX IF NOT EXISTS idx_statements_obj   ON statements(tenant_id, object_id, stored_at DESC);

-- Optional: JSONB GIN index if you plan to query deep JSON frequently
-- CREATE INDEX IF NOT EXISTS idx_statements_statement_gin ON statements USING gin(statement);