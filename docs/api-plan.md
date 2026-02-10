# API Plan (LL-inspired)

Date: 2026-02-10
Status: Draft

## Goals
- Provide LRS-scoped, client-authenticated APIs without `/admin` in the URI.
- Support high-volume statements with cursor pagination and server-side aggregation.
- Keep interfaces consistent and extensible, inspired by Learning Locker’s REST, Connection, and Aggregation APIs.

## Scope & Auth Model
- All API access is scoped to an LRS, enforced by client credentials and scopes.
- Each client is bound to a project and optionally an LRS; if bound, access is limited to that LRS.
- Authorization is planned (likely HTTP Basic Auth, key/secret, similar to LL’s APIs).

## Proposed API Families (LL-inspired)

### 1) Connection API (preferred for large datasets)
LL’s Connection API uses cursor-based pagination inspired by GraphQL connections and supports filtering + sort. This pattern is a good fit for large, append-heavy statement collections.

**Endpoints**
- `GET /api/connection/statement` (implemented)
- `GET /api/connection/user` (planned)
- `GET /api/connection/client` (planned)
- `GET /api/connection/lrs` (planned)

**Common Query Params**
- `first` (page size; default 25, max 1000)
- `after` (cursor)

**Statement Query Params (current)**
- `actor_mbox`
- `verb` (verb id)
- `activity` (object id)
- `since` (stored_at lower bound)
- `until` (stored_at upper bound; date-only expands to end-of-day)

**Response Shape (example)**
```
{
  "edges": [{"cursor": "...", "node": { ... }}],
  "pageInfo": {"hasNextPage": true, "endCursor": "..."}
}
```

### 2) REST API (CRUD for management models)
LL’s REST API provides CRUD for models and counts.

**Endpoints (current + planned)**
- `GET /api/v1/{model}` (implemented for `user`)
- `GET /api/v1/{model}/count` (implemented for `user`)
- `POST /api/v1/{model}` (planned)
- `GET /api/v1/{model}/{id}` (planned)
- `PATCH /api/v1/{model}/{id}` (planned)
- `DELETE /api/v1/{model}/{id}` (planned)

**Models (initial)**
- `user` (implemented), `client`, `organisation`, `project`, `lrs` (planned)

### 3) Aggregation API (statements analytics)
LL’s Aggregation API exposes Mongo’s aggregation pipeline for statements only. We can provide a Postgres-backed aggregation interface that supports a safe, limited pipeline DSL.

**Endpoint (proposed)**
- `GET /api/aggregation/statement` (stubbed)
- Optional `POST /api/aggregation/statement` for large pipelines (stubbed)

**Query Params (proposed)**
- `pipeline` (JSON encoded array of stages)
- `maxTimeMS`, `maxScan`, `cache` (optional execution guards)

**Supported Stages (initial)**
- `limit`, `skip`, `sort`, `match`, `project`, `group`

## Why these APIs
- **Connection API** scales best for large, continuously growing datasets because of cursor pagination. LL explicitly describes this as cursor-based and inspired by GraphQL connections.
- **Aggregation API** is valuable for analytics/rollups at scale; LL notes it is more advanced than xAPI and enables complex server-side filtering/grouping.
- **REST API** remains useful for model management and simple CRUD.

## Routing (no /admin)
- Replace `/api/admin/*` with the public model APIs above.
- Admin views use the same public APIs with elevated scopes; no separate admin namespace.

## Detail Endpoints
- `GET /api/statement/{id}` for statement detail (raw xAPI payload + key fields).

## OpenAPI/Swagger
- Provide a single OpenAPI 3.1 spec (YAML) under `docs/openapi.yaml`.
- Generate Swagger UI at `/docs` and spec at `/docs/openapi.yaml`.

## Next Steps
1. Implement `GET /api/connection/user|client|lrs` and remaining REST models.
2. Define filter DSL and connection sort contract (provider-agnostic).
3. Implement aggregation stages (`match`, `group`, `sort`, `limit`) for Postgres.
4. Add auth/scopes to all public endpoints; enforce LRS scoping per client.
