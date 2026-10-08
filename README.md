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

### 2. Set up the local environment

```bash
make install
```

This checks for Go, pnpm, and Docker, creates `.env` from `.env.example` when that file is missing, installs Go and web dependencies, starts the development Postgres, and applies migrations. An existing `.env` is left unchanged.

### 3. Start the development stack

```bash
make dev
```

This starts PostgreSQL on port `5433`, the Go API with live reload at `http://localhost:8081`, and the Vite development server at `http://localhost:5174`. When the stack is up, `make dev` prints those URLs and returns. Postgres from `make install` is reused. Follow logs with `make dev logs`, and stop the stack with `make down`.

If any port is already occupied, override it when starting the stack:

```bash
DEV_POSTGRES_PORT=55432 DEV_API_PORT=18080 DEV_WEB_PORT=15173 make dev
```

To run the API and web app directly on the host instead, start each server in a separate terminal after `make install`:

```bash
make dev api   # Go API on :8080
make dev web   # Vite dev server on :5173
```

### 4. Create the first user

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
| `APP_URL` | No | `http://localhost:5173` | Public web origin used in invite links and SSO redirect URIs |
| `LOCAL_LOGIN_ENABLED` | No | `true` | When set, updates password login for the default organisation on startup |
| `OIDC_ENABLED` | No | `false` | Seed an OpenID Connect provider when `OIDC_SLUG` does not exist yet |
| `OIDC_ISSUER` | If OIDC | | Issuer URL, for example `https://idp.example.edu/realms/main` |
| `OIDC_CLIENT_ID` | If OIDC | | OAuth client id |
| `OIDC_CLIENT_SECRET` | No | | Stored encrypted; requires `DATA_STORE_SECRET_KEY` |
| `SAML_ENABLED` | No | `false` | Seed a SAML / Shibboleth provider when `SAML_SLUG` does not exist yet |
| `SAML_IDP_METADATA_URL` | If SAML | | IdP metadata URL, for example `https://idp.example.edu/idp/shibboleth` |
| `SAML_SP_ENTITY_ID` | No | metadata URL | Stable SP entityID |
| `SAML_SP_CERT_FILE` / `SAML_SP_KEY_FILE` | No | generated | PEM files for the SP signing key |

---

## API Endpoints

### Auth

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/auth/login` | Login with email + password; sets `lrsolid_session` cookie |
| `POST` | `/api/auth/logout` | Clears session cookie |
| `GET` | `/api/auth/me` | Returns current user (id, email, role, auth provider) |
| `GET` | `/api/auth/providers` | Enabled SSO providers and whether password login is on |
| `GET` | `/api/auth/sso/{slug}/start` | Redirect to an OpenID Connect or SAML identity provider |
| `GET` | `/api/auth/oidc/{slug}/callback` | OpenID Connect redirect URI |
| `POST` | `/api/auth/saml/{slug}/acs` | SAML assertion consumer service |
| `GET` | `/api/auth/saml/{slug}/metadata` | SAML service-provider metadata |
| `GET` | `/api/auth/invite` | Look up an invitation token |
| `POST` | `/api/auth/invite/accept` | Accept an invitation with a password |

Password login can be turned off per organisation. OpenID Connect (Keycloak, Entra ID, Google, or a Shibboleth IdP with the OIDC plugin) and SAML 2.0 / Shibboleth are configured in Settings → Authentication, or seeded from the environment for the default organisation. See [docs/adr-0005.md](docs/adr-0005.md).

Redirect URLs are built from `APP_URL` and must be registered at the identity provider:

- OpenID Connect: `{APP_URL}/api/auth/oidc/{slug}/callback`
- SAML ACS: `{APP_URL}/api/auth/saml/{slug}/acs`
- SAML metadata: `{APP_URL}/api/auth/saml/{slug}/metadata`

For Shibboleth, download the metadata, register the SP with the IdP or federation (DFN-AAI, eduGAIN), and release `subject-id` or `eduPersonPrincipalName`, `mail`, and optionally `displayName`, `eduPersonScopedAffiliation`, `eduPersonEntitlement`. SSO users are created when auto-provision is on or when an unused invite matches their email. Otherwise sign-in stops with "no account".

### Test the connection

**Test** on a provider checks that the API can reach the identity provider. It does not sign a user in and it does not check the client secret. OpenID Connect must publish discovery and signing keys. SAML must return parseable IdP metadata; the message shows the IdP `entityID`.

The URL is fetched by the API process. With `make dev` that is the `api` container, so use a hostname that container can resolve (`dex`, `saml-idp`), not `localhost`.

1. Start the stack. `APP_URL` and `DATA_STORE_SECRET_KEY` come from `docker-compose.dev.yml`. Recreate the API container if it was started before those variables existed:

```bash
make dev
docker compose -f docker-compose.dev.yml up -d api
docker compose -f docker-compose.dev.yml --profile sso up -d dex saml-idp
```

2. Sign in with email and password and open **Settings → Authentication**.

3. OpenID Connect (Dex). Add a provider and click **Test**:

| Field | Value |
|-------|-------|
| Type | OpenID Connect |
| Display name | Dex |
| Slug | `dex` |
| Issuer | `http://dex:5556/dex` |
| Client ID | `lrsolid` |
| Client secret | `dev-dex-secret` |

A successful test reports the authorization endpoint, for example `issuer http://dex:5556/dex/auth`. Dex's user is `admin@example.com` / `password`.

4. SAML (test IdP, same shape as Shibboleth). Add a provider and click **Test**:

| Field | Value |
|-------|-------|
| Type | SAML / Shibboleth |
| Display name | Shibboleth |
| Slug | `shibboleth` |
| IdP metadata URL | `http://saml-idp:8080/simplesaml/saml2/idp/metadata.php` |

Leave the SP entity ID empty so it becomes `{APP_URL}/api/auth/saml/shibboleth/metadata`. Saving generates the SP key. **Test** reports `idp` plus the entityID. **Metadata** downloads the SP metadata; the test IdP is already aimed at that entityID and ACS URL.

5. Optional full sign-in, after **Test** succeeded. Enable **Auto-provision** (or invite the IdP email), sign out, and use **Continue with …**. For Dex, the browser must resolve the issuer host: add `127.0.0.1 dex` to `/etc/hosts` (port `5556` is published on the host). The test SAML IdP UI is at `http://localhost:8085`. Its login redirect only works in the browser when the SSO URL inside the metadata is a host the browser can open; the **Test** button remains the check that LRSolid itself can read the IdP.

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
make install               First-time setup (.env, dependencies, Postgres, migrations)
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

LRSolid is licensed under the [Apache License 2.0](LICENSE).

Copyright 2026 Sergej Görzen.

If you fork, extend, redistribute, or publish derived work based on LRSolid, retain the attribution notices in [NOTICE](NOTICE) and mention the original project and author.
