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
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI for migrations

### 1. Clone and install dependencies

```bash
git clone https://github.com/SGoerzen/LRSolid.git
cd LRSolid
make deps
```

### 2. Configure environment

```bash
cp .env.example .env
# Edit .env — set PG_DSN, JWT_SECRET
```

Generate a strong JWT secret:

```bash
openssl rand -hex 32
```

### 3. Start a local PostgreSQL instance (optional)

```bash
make pg-up
```

### 4. Run database migrations

```bash
make migrate-up
```

### 5. Create the first user

```bash
make create-user EMAIL=you@example.com PASSWORD=secret ROLE=owner
```

Available roles: `owner`, `admin`, `member`, `viewer`

### 6. Run the development servers

In two separate terminals:

```bash
make dev-api   # Go API on :8080
make dev-web   # Vite dev server on :5173
```

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PG_DSN` | Yes | `postgres://...localhost/lrsolid` | PostgreSQL connection string |
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
curl -sS -X POST http://localhost:8080/api/v1/client \
  -H 'Content-Type: application/json' \
  -H 'Cookie: lrsolid_session=YOUR_SESSION_COOKIE' \
  -d '{"name":"local-cli","scopes":{"statements:read":true,"statements:write":true}}'
```

2. Use returned `keyId` + `secret` for xAPI:

```bash
curl -sS -u "YOUR_KEY_ID:YOUR_SECRET" \
  -H 'Content-Type: application/json' \
  -H 'X-Experience-API-Version: 1.0.3' \
  -X POST http://localhost:8080/xapi/statements \
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
make migrate-up    # Apply all pending migrations
make migrate-down  # Roll back one migration
```

---

## Makefile Targets

```
make deps          Install Go + pnpm dependencies
make dev-api       Run Go API locally
make dev-web       Run Vite dev server
make build         Build API binary + Web bundle
make test          Run all tests
make lint          Lint Go + Web
make fmt           Format Go + Web
make migrate-up    Apply DB migrations
make migrate-down  Roll back DB migrations
make create-user   Create a user (EMAIL= PASSWORD= [ROLE=owner] [ORG=default])
make pg-up         Start local Postgres via Docker
make pg-down       Stop local Postgres container
make xapi-smoke    POST + GET a test xAPI statement
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

## License

© 2026 Sergej Görzen. All rights reserved.
