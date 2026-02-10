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
- Authorization uses HTTP Basic Auth (key/secret), similar to LL’s APIs.

## Proposed API Families (LL-inspired)

### 1) Connection API (preferred for large datasets)
LL’s Connection API uses cursor-based pagination inspired by GraphQL connections and supports filtering + sort. This pattern is a good fit for large, append-heavy statement collections.

**Endpoints (proposed)**
- `GET /api/connection/statement`
- `GET /api/connection/user`
- `GET /api/connection/client`
- `GET /api/connection/lrs`

**Common Query Params**
- `sort` (JSON encoded, required; default to `{"stored_at":-1,"id":1}`)
- `filter` (JSON encoded, provider-agnostic filter language)
- `first` (page size; default 25, max 1000)
- `after` (cursor)

**Response Shape (example)**
```
{
  "edges": [{"cursor": "...", "node": { ... }}],
  "pageInfo": {"hasNextPage": true, "endCursor": "..."}
}
```

### 2) REST API (CRUD for management models)
LL’s REST API provides CRUD for models and counts.

**Endpoints (proposed)**
- `GET /api/v2/{model}`
- `GET /api/v2/{model}/count`
- `POST /api/v1/{model}`
- `GET /api/v1/{model}/{id}`
- `PATCH /api/v1/{model}/{id}`
- `DELETE /api/v1/{model}/{id}`

**Models (initial)**
- `user`, `client`, `organisation`, `project`, `lrs`

### 3) Aggregation API (statements analytics)
LL’s Aggregation API exposes Mongo’s aggregation pipeline for statements only. We can provide a Postgres-backed aggregation interface that supports a safe, limited pipeline DSL.

**Endpoint (proposed)**
- `GET /api/aggregation/statement` (query params)
- Optional `POST /api/aggregation/statement` for large pipelines

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

## OpenAPI/Swagger
- Provide a single OpenAPI 3.1 spec (YAML) under `docs/openapi.yaml`.
- Generate Swagger UI at `/docs` and spec at `/docs/openapi.yaml`.

## Next Steps
1. Decide on initial API families to ship (recommend Connection + REST first).
2. Define the filter DSL and cursor format as provider-agnostic contracts.
3. Add OpenAPI skeleton and wire a basic Swagger UI.
4. Incrementally implement endpoints, starting with `GET /api/connection/statements` and `GET /api/v1/lrs`.
