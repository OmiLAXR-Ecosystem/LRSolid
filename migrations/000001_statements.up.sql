-- ============================================================
-- xAPI Statements table
-- Design goals:
--   - Store the full raw statement as JSONB (source of truth)
--   - Extract indexed columns for common filter/analytics queries
--   - Support all xAPI 1.0.3 actor IFI types
--   - Support all xAPI object types incl. StatementRef (voiding)
--   - Denormalize Group members into statement_agents for lookup
-- ============================================================

CREATE TABLE IF NOT EXISTS statements (

    -- Core identity
    tenant_id   text        NOT NULL,
    id          text        NOT NULL,
    stored_at   timestamptz NOT NULL,       -- when the LRS stored it (server-assigned)
    activity_at timestamptz NULL,           -- statement.timestamp (when activity occurred)

    -- Actor: objectType
    actor_type          text NULL,          -- 'Agent' | 'Group'

    -- Actor: IFI (all 4 types per xAPI 1.0.3)
    actor_mbox          text NULL,          -- "mailto:alice@example.org"
    actor_mbox_sha1sum  text NULL,
    actor_openid        text NULL,
    actor_account_home  text NULL,          -- account.homePage
    actor_account_name  text NULL,          -- account.name

    -- Actor: privacy hash (server-derived, optional)
    actor_sha256        text NULL,

    -- Verb
    verb_id             text NULL,          -- IRI

    -- Object
    object_type         text NULL,          -- 'Activity' | 'StatementRef' | 'SubStatement' | 'Agent' | 'Group'
    object_id           text NULL,          -- IRI (Activity) or UUID (StatementRef)
    activity_type       text NULL,          -- object.definition.type IRI e.g. ".../course"

    -- Context
    registration        text NULL,          -- context.registration UUID
    context_platform    text NULL,
    context_parent_ids  text[] NULL,        -- contextActivities.parent[].id
    context_group_ids   text[] NULL,        -- contextActivities.grouping[].id
    context_category_ids text[] NULL,       -- contextActivities.category[].id

    -- Result
    result_completion   boolean NULL,
    result_success      boolean NULL,
    result_score_scaled numeric(7,4) NULL,  -- -1.0000 to 1.0000
    result_duration     text NULL,          -- ISO 8601 duration e.g. "PT10M30S"

    -- Voiding (xAPI spec: verb voided + object StatementRef)
    voided              boolean NOT NULL DEFAULT false,

    -- Full statement payload (source of truth)
    statement           jsonb NOT NULL,

    PRIMARY KEY (tenant_id, id)
);

-- ============================================================
-- Indexes on statements
-- ============================================================

-- Primary sort/cursor index (used by GetStatements pagination)
CREATE INDEX IF NOT EXISTS idx_statements_tenant_stored
    ON statements(tenant_id, stored_at DESC, id DESC);

-- Actor filters
CREATE INDEX IF NOT EXISTS idx_statements_actor_mbox
    ON statements(tenant_id, actor_mbox, stored_at DESC);

CREATE INDEX IF NOT EXISTS idx_statements_actor_account
    ON statements(tenant_id, actor_account_home, actor_account_name, stored_at DESC);

CREATE INDEX IF NOT EXISTS idx_statements_actor_sha256
    ON statements(tenant_id, actor_sha256, stored_at DESC);

-- Verb + Object filters
CREATE INDEX IF NOT EXISTS idx_statements_verb
    ON statements(tenant_id, verb_id, stored_at DESC);

CREATE INDEX IF NOT EXISTS idx_statements_object
    ON statements(tenant_id, object_id, stored_at DESC);

CREATE INDEX IF NOT EXISTS idx_statements_object_type
    ON statements(tenant_id, object_type, stored_at DESC);

CREATE INDEX IF NOT EXISTS idx_statements_activity_type
    ON statements(tenant_id, activity_type, stored_at DESC);

-- Context
CREATE INDEX IF NOT EXISTS idx_statements_registration
    ON statements(tenant_id, registration, stored_at DESC);

-- Array indexes for contextActivities (GIN)
CREATE INDEX IF NOT EXISTS idx_statements_context_parent
    ON statements USING gin(context_parent_ids);

CREATE INDEX IF NOT EXISTS idx_statements_context_group
    ON statements USING gin(context_group_ids);

CREATE INDEX IF NOT EXISTS idx_statements_context_category
    ON statements USING gin(context_category_ids);

-- Result analytics
CREATE INDEX IF NOT EXISTS idx_statements_result
    ON statements(tenant_id, result_completion, result_success, stored_at DESC);

CREATE INDEX IF NOT EXISTS idx_statements_score
    ON statements(tenant_id, result_score_scaled, stored_at DESC)
    WHERE result_score_scaled IS NOT NULL;

-- Full JSONB index for deep extension queries
CREATE INDEX IF NOT EXISTS idx_statements_gin
    ON statements USING gin(statement jsonb_path_ops);

-- ============================================================
-- statement_agents: denormalized actor members for Group lookup
--
-- Why: xAPI Group actors have member[] arrays. Queries like
-- "find all statements by alice@example.org" must also match
-- statements where alice is a Group member, not just the Group
-- itself. This mirrors Learning Locker's "agents" array.
-- ============================================================

CREATE TABLE IF NOT EXISTS statement_agents (
    id           bigserial   PRIMARY KEY,
    tenant_id    text        NOT NULL,
    statement_id text        NOT NULL,

    -- IFI of the member agent (at least one must be non-null)
    mbox         text NULL,
    mbox_sha1sum text NULL,
    openid       text NULL,
    account_home text NULL,
    account_name text NULL
);

CREATE INDEX IF NOT EXISTS idx_stmt_agents_stmt
    ON statement_agents(tenant_id, statement_id);

CREATE INDEX IF NOT EXISTS idx_stmt_agents_mbox
    ON statement_agents(tenant_id, mbox)
    WHERE mbox IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_stmt_agents_account
    ON statement_agents(tenant_id, account_home, account_name)
    WHERE account_home IS NOT NULL;
