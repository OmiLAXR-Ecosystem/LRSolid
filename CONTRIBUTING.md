# Contributing

Thank you for considering a contribution to LRSolid.

## Development Setup

Prerequisites:

- Go 1.22+
- Node.js 20+
- pnpm 9+
- Docker, for local PostgreSQL and integration tests

Common commands:

```bash
make deps
make dev
make test
make lint
make fmt
```

Run focused tests when changing one area:

```bash
go test ./internal/httpapi/... -run TestName -v
go test ./internal/store/postgres/... -run TestName -v
cd apps/web && pnpm lint
```

## Contribution Guidelines

- Keep changes focused and aligned with the existing architecture.
- Preserve xAPI 1.0.3 semantics and cursor-based statement pagination.
- Add migrations instead of editing already-applied migrations.
- Add or update tests for behavioral changes.
- Update documentation when behavior, configuration, public APIs, or research metadata changes.
- Do not commit secrets, local database dumps, generated build artifacts, or private deployment configuration.

## Pull Requests

Before opening a pull request:

- Run `make fmt`.
- Run the most relevant tests for your change.
- Explain the motivation, user-visible impact, and verification steps.
- Link related issues or research context where applicable.

## Research Software Metadata

When preparing a release, update:

- `CHANGELOG.md`
- `CITATION.cff`
- `codemeta.json`
- `apps/web/package.json`
- `apps/web/src/version.ts`

If a DOI, ORCID, grant, related publication, or archived release becomes available, add it to `CITATION.cff` and `codemeta.json`.
