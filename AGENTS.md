# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

LRSolid is a self-hosted Learning Record Store (LRS) that serves xAPI 1.0.3 and 2.0. Monorepo with a Go backend (`apps/api`) and a React 19 admin frontend (`apps/web`), backed by PostgreSQL.

```
Browser (React SPA) → Go HTTP API (Chi, :8080) → PostgreSQL
                                                 → [planned] Kafka → ClickHouse/TimescaleDB
```

## Commands

All common tasks go through the root `Makefile` (`make help` lists everything).

```bash
make deps          # go mod download + pnpm install
make install         # first-time setup: .env, deps, Postgres, migrations
make dev             # start development stack with live reload
make up              # start production stack
make down            # stop development and production stacks
make build           # build API binary + web bundle
make test             # go test ./... + web tests
make test api         # Go tests only
make test web         # web tests only
make lint              # go vet + web lint
make fmt                # go fmt + web format
make migrate up        # apply pending migrations (golang-migrate, needs PG_DSN)
make migrate down      # roll back one migration
make pg up / pg down   # local Postgres via Docker
make admin EMAIL=you@example.com PASSWORD=secret [ORG=default]
make owner EMAIL=you@example.com PASSWORD=secret [ORG=default]
make user EMAIL=you@example.com PASSWORD=secret [ORG=default]
make xapi smoke        # POST + GET a test xAPI statement (needs AUTH=keyId:secret)
make xapi conformance  # ADL conformance suite (AUTH=key:secret VERSION=1.0.3|2.0.0)
```

Single-package/single-test Go runs (from repo root, module is `github.com/SGoerzen/LRSolid`):

```bash
go test ./internal/httpapi/... -run TestXAPIStatementsRouteWithoutAuth_MatchesBothSlashVariants -v
go test ./internal/store/postgres/... -run TestStatements_EndToEnd -v
```

Web app lives in `apps/web`; run `pnpm <script>` from there (or `cd apps/web && pnpm dev/build/lint`).

**Testcontainers**: `internal/store/postgres/*_test.go` spin up a real `postgres:16` container via testcontainers-go (`withTestDB` in `db_test.go`). Docker must be running locally for these tests to pass.

## Architecture

### Multi-tenancy hierarchy

`Organisation → Project → LRS`. Each LRS instance is its own xAPI tenant (`store.TenantID` == LRS ID). Roles are granted independently at each level via `org_memberships`, `project_memberships`, and `lrs_memberships` tables (`owner/admin/member/viewer` at org/project, `admin/writer/reader` at LRS level). LRS-level membership is optional and falls back to the project-level role. See `README.md` "Role System" for the full matrix.

In addition to user roles, **Clients** (`internal/models.Client`) are machine credentials scoped to a project and optionally a single LRS, authenticated via HTTP Basic (`key_id:secret`) and gated by a JSON `scopes` map (e.g. `statements:write`).

### xAPI resources

Spec logic that does not touch HTTP or a database lives in `internal/xapi` (version negotiation, validation, agent keys, statement preparation, formats, JWS). HTTP handlers are `internal/httpapi/statements.go`, `xapi_documents.go`, `xapi_activities.go`, `xapi_version.go`, and `xapi_alternate.go`. `X-Experience-API-Version` is required unless `XAPI_LEGACY_COMPAT=true`, which also restores form bodies, the `{statements:[...]}` wrapper, and a `cursor` field beside `more`. Document writes use `409` when an existing document has no `If-Match` / `If-None-Match`, and `412` when the precondition fails. Attachment bytes go to `XAPI_ATTACHMENT_DIR`.

### Two auth mechanisms, two route groups

- `/api/*` — human users, session-cookie auth (JWT in `lrsolid_session` httpOnly cookie), enforced by `auth.Middleware` (`internal/auth/middleware.go`). Password login is `POST /api/auth/login`. OpenID Connect and SAML 2.0 (Shibboleth) logins live in `internal/sso` and issue the same cookie; see `docs/adr-0005.md`.
- `/xapi/*` — xAPI clients, HTTP Basic auth (`key_id:secret`) enforced by `XAPIClientAuth` (`internal/httpapi/xapi_auth.go`); also accepts session cookies for dev/UI convenience.

Routing is centralized in `internal/httpapi/router.go` — start there to see the full endpoint surface before adding routes.

### Storage is provider-abstracted (`internal/store`)

`internal/store/store.go` defines the storage **ports** (`StatementStore`, `StateStore`, `AgentProfileStore`, `ActivityProfileStore`, `AttachmentStore`) independent of any database. `internal/store/postgres/` is the only implementation today (MongoDB is planned — see `docs/adr-0001.md`). When touching statement/state/profile persistence, change the interface in `store.go` first, then the Postgres implementation — don't leak Postgres-specific types into `internal/lrs` or `internal/httpapi`.

Key semantics the core enforces on every provider (must not be broken by a new provider):
- **Cursor-based pagination only** — ordering is always `(stored_at, id)`, never offset-based. `store.EncodeCursor`/`DecodeCursor` define the opaque token format and live in core, not per-provider.
- Statement upserts are idempotent by `(tenant, id)`; identical `RawJSON` on conflict is a no-op success, differing content is `ErrAlreadyExists`.

### xAPI statement schema (Postgres)

Statements are stored as full JSONB (source of truth) *plus* extracted columns for anything used in filtering/analytics (actor identifiers, verb id, object id/type, context registration, result completion/success/score, voided flag). Fields kept JSONB-only (display strings, extensions, `authority`) are intentionally not promoted to columns. Rationale and the exact column list are in `docs/adr-0002.md` — read it before adding a new filterable statement field, since there's a deliberate line between "columnize" and "leave in JSONB with GIN index".

Group actors (`objectType: "Group"` with `member[]`) are denormalized into a separate `statement_agents` table so "all statements involving agent X" queries also match group membership (mirrors Learning Locker's pattern).

Voiding: a statement with `verb.id = .../verbs/voided` and `object.objectType = "StatementRef"` sets `voided = true` on the referenced statement; voided statements are excluded from normal queries per spec.

### Async stats rollup

`internal/statsrollup/worker.go` is a background job (enabled by default, `STATS_ROLLUP_ENABLED`) that batches updates to statement counters/daily stats rather than maintaining them synchronously on every write. Tune via `STATS_ROLLUP_INTERVAL_S` / `STATS_ROLLUP_BATCH_SIZE`.

### Analytics pipeline (planned, not yet implemented)

`docs/adr-0003.md` describes the target design: Postgres stays the xAPI source of truth; an outbox/CDC path feeds Kafka; a separate analytics service consumes and writes aggregates to ClickHouse/TimescaleDB for dashboarding, kept off the ingest path. Don't build analytics aggregation directly against the hot Postgres statements table — that's the problem this ADR exists to avoid.

### Frontend (`apps/web`)

React 19 + TypeScript + Tailwind + Vite. Strict layering, enforced by convention (see `docs/frontend-components.md` for the full component/hook reference):

```
api/        typed HTTP calls only — no fetch() in components
types/      shared TS types
hooks/      data-fetching + state (e.g. useStatements, useUsers)
components/ ui/ (primitives) , statements/, users/ (feature), layout/ (AdminShell)
views/      thin containers — just wire a hook to feature components
```

`Pagination` in `components/ui/` supports both cursor-based (statements) and offset-based (users) paging through the same component. Feature flags/env toggles (`VITE_IS_ADMIN`, `VITE_CAN_EXPORT`, `VITE_CAN_VOID`) gate UI affordances client-side.

### Migrations

`migrations/` uses `golang-migrate`, numbered sequentially (currently up to `000028`). Each has a paired `.up.sql`/`.down.sql`. Always add new migrations at the next number — never edit an already-applied migration in place.

### `packages/`

`packages/contracts`, `packages/schemas`, `packages/sdk` are currently placeholders (`.keep` only) — reserved for future shared-contract/SDK extraction, not yet wired into either app.
