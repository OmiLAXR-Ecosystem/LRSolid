# LRSolid

**LRSolid** is an open-source Learning Record Store (LRS) compliant with the [xAPI 1.0.3](https://github.com/adlnet/xAPI-Spec) specification. It is designed for self-hosting in university and enterprise environments, with a focus on performance, analytics, and extensibility.

---

## Features

- xAPI 1.0.3 statement ingestion and querying
- State, Agent Profile, and Activity Profile APIs
- Multi-tenant architecture (Organisations → Projects → LRS)
- JWT-based authentication via httpOnly session cookies
- Role-based access control (RBAC) at Organisation, Project, and LRS level
- Admin UI built with React 19, Tailwind CSS, and Vite
- PostgreSQL (JSONB + extracted columns) as primary store
- Storage Provider abstraction (MongoDB planned, see ADR-0001)
- Analytics pipeline planned: Kafka + ClickHouse/TimescaleDB (see ADR-0003)

---

## Architecture Overview

```
Browser (React SPA)
       |
       v
  Go HTTP API  (Chi router, port 8080)
       |
       +---> PostgreSQL  (statements, users, orgs, projects, clients)
       |
       +---> [planned] Kafka → ClickHouse / TimescaleDB
```

The monorepo contains two apps:

| Path | Description |
|------|-------------|
| `apps/api/` | Go REST + xAPI backend |
| `apps/web/` | React 19 admin frontend (Vite, Tailwind) |

---

## Role System

LRSolid uses a three-level RBAC model. A user can hold a role at each level independently.

### Organisation Roles

Granted via the `org_memberships` table.

| Role | Description |
|------|-------------|
| `owner` | Full control over the organisation — manage members, projects, billing, deletion |
| `admin` | Manage members and projects within the org, but cannot delete the org |
| `member` | Read access to org resources; can be added to projects |
| `viewer` | Read-only visibility of org-level information |

### Project Roles

Granted via the `project_memberships` table. A project belongs to one organisation.

| Role | Description |
|------|-------------|
| `owner` | Full control over the project — manage members, LRS instances, clients |
| `admin` | Manage LRS instances and clients within the project |
| `member` | Access project resources according to their LRS-level role |
| `viewer` | Read-only visibility of project-level information |

### LRS Roles

Granted via the `lrs_memberships` table. Fine-grained access override per LRS instance.

| Role | Description |
|------|-------------|
| `admin` | Full access to statements, state, profiles; manage clients for this LRS |
| `writer` | Can POST statements, write state and profile documents |
| `reader` | Can GET statements, state, and profile documents (read-only) |

> **Note:** LRS-level membership is optional. If not set, access falls back to the project-level role.

### API Clients

In addition to user roles, **Clients** are machine auth entities (xAPI Basic Auth via `key_id` / `secret`). Clients are scoped to a project and optionally to a single LRS. Their permissions are controlled via a `scopes` JSON object, e.g.:

```json
{
  "statements:write": true,
  "statements:read": true,
  "state:write": true
}
```

---

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 20+ / pnpm 9+
- PostgreSQL 15+ (or Docker)

### 1. Clone the repository

```bash
git clone https://github.com/SGoerzen/LRSolid.git
cd LRSolid
```

### 2. Start the development stack

```bash
make dev
```

This starts PostgreSQL on port `5433`, the Go API with live reload at `http://localhost:8081`, and the Vite development server at `http://localhost:5174`. Press Ctrl+C to stop the stack.

If any port is already occupied, override it when starting the stack:

```bash
DEV_POSTGRES_PORT=55432 DEV_API_PORT=18080 DEV_WEB_PORT=15173 make dev
```

To run the API and web app directly on the host instead, install the dependencies and start each server in a separate terminal:

```bash
cp .env.example .env
# Edit .env and set PG_DSN and JWT_SECRET.
make deps
make pg up
make migrate up
make dev api   # Go API on :8080
make dev web   # Vite dev server on :5173
```

### 3. Create the first user

With the development stack running, create a user in another terminal:

```bash
make user create EMAIL=you@example.com PASSWORD=secret ROLE=owner
```

Available roles: `owner`, `admin`, `member`, `viewer`

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PG_DSN` | Yes | `postgres://lrsolid:lrsolid@localhost:5433/lrsolid?sslmode=disable` | PostgreSQL connection string |
| `JWT_SECRET` | Yes (prod) | dev fallback | HMAC-SHA256 signing secret for session tokens |
| `DEFAULT_ORG` | No | `default` | Slug of the default organisation |

---

## API Endpoints

### Auth

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/auth/login` | Login with email + password; sets `lrsolid_session` cookie |
| `POST` | `/api/auth/logout` | Clears session cookie |
| `GET` | `/api/auth/me` | Returns current user (id, email, role) |

### xAPI

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/xapi/statements` | Ingest a statement or batch of statements (requires client scope `statements:write`) |
| `GET` | `/xapi/statements` | Query statements (requires client scope `statements:read`) |

### REST

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/user` | List users (filter: name, email, role) |
| `GET` | `/api/v1/user/count` | Count users |
| `GET` | `/api/v1/client` | List API clients |
| `POST` | `/api/v1/client` | Create API client and return `keyId` + `secret` (secret shown once) |
| `GET` | `/api/statement/{id}` | Fetch single statement by ID |
| `GET` | `/api/connection/statement` | Cursor-paginated statement connection (GraphQL-style edges) |

`/api/*` routes require a valid session cookie. `/xapi/*` routes accept HTTP Basic auth with client `keyId:secret` (and also accept session cookie for compatibility in dev/UI flows).

### xAPI Basic Auth Quick Example

1. Create a client (while logged in with session cookie):

```bash
curl -sS -X POST http://localhost:8081/api/v1/client \
  -H 'Content-Type: application/json' \
  -H 'Cookie: lrsolid_session=YOUR_SESSION_COOKIE' \
  -d '{"name":"local-cli","scopes":{"statements:read":true,"statements:write":true}}'
```

2. Use returned `keyId` + `secret` for xAPI:

```bash
curl -sS -u "YOUR_KEY_ID:YOUR_SECRET" \
  -H 'Content-Type: application/json' \
  -H 'X-Experience-API-Version: 1.0.3' \
  -X POST http://localhost:8081/xapi/statements \
  -d '{"statement":{"actor":{"mbox":"mailto:sergej@example.org"},"verb":{"id":"http://adlnet.gov/expapi/verbs/experienced","display":{"en-US":"experienced"}},"object":{"id":"https://example.org/activity/intro"}}}'
```

---

## Database Migrations

Migrations live in `migrations/` and are managed with `golang-migrate`.

| Migration | Description |
|-----------|-------------|
| `000001` | xAPI statements table (full schema with actor, verb, object, context, result, voiding) |
| `000002` | State documents |
| `000003` | Agent profiles |
| `000004` | Activity profiles |
| `000005` | Organisations |
| `000006` | Users |
| `000007` | Projects |
| `000008` | Org and project memberships (with role CHECK constraints) |
| `000009` | LRS instances and LRS memberships |
| `000010` | Clients (xAPI Basic Auth with scopes) |

```bash
make migrate up    # Apply all pending migrations
make migrate down  # Roll back one migration
```

---

## Makefile Targets

```
make deps                  Install Go + pnpm dependencies
make dev api               Run Go API locally
make dev web               Run Vite dev server
make build                 Build API binary + Web bundle
make test                  Run all tests
make lint                  Lint Go + Web
make fmt                   Format Go + Web
make migrate up            Apply DB migrations
make migrate down          Roll back DB migrations
make user create           Create a user (EMAIL= PASSWORD= [ROLE=owner] [ORG=default])
make pg up                 Start local Postgres via Docker
make pg down               Stop local Postgres container
make xapi smoke            POST + GET a test xAPI statement
```

---

## Documentation

| Document | Description |
|----------|-------------|
| [docs/adr-0001.md](docs/adr-0001.md) | Storage strategy: PostgreSQL-first, provider abstraction |
| [docs/adr-0002.md](docs/adr-0002.md) | xAPI statement schema: column extraction, denormalisation |
| [docs/adr-0003.md](docs/adr-0003.md) | Analytics architecture: Kafka + ClickHouse/TimescaleDB |
| [docs/frontend-components.md](docs/frontend-components.md) | React component reference |

---

## Research Software Metadata

| File | Purpose |
|------|---------|
| [CITATION.cff](CITATION.cff) | Citation metadata for GitHub, Zotero, and related tooling |
| [codemeta.json](codemeta.json) | FAIR research software metadata in CodeMeta JSON-LD format |
| [CHANGELOG.md](CHANGELOG.md) | Release history and notable changes |
| [AUTHORS.md](AUTHORS.md) | Maintainers, authors, and institutional cooperation |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Development and contribution guidelines |
| [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) | Community conduct expectations |
| [SECURITY.md](SECURITY.md) | Vulnerability reporting and supported versions |

If you use LRSolid in research, teaching, or publications, please cite it using the metadata in [CITATION.cff](CITATION.cff).

---

## License

© 2026 Sergej Görzen. All rights reserved.
