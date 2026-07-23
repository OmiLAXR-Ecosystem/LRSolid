# Changelog

All notable changes to LRSolid are documented in this file.

The format is based on Keep a Changelog, and this project uses semantic versioning for public releases where practical.

## [Unreleased]

### Added

- Research software metadata files: `CITATION.cff`, `codemeta.json`, `AUTHORS.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, and `SECURITY.md`.
- Footer version display in the web admin UI.

### Changed

- Web package version set to `0.1.0`.

## [0.1.0] - 2026-07-23

### Added

- Initial xAPI 1.0.3 Learning Record Store implementation.
- PostgreSQL-backed statement, state, agent profile, activity profile, and attachment storage.
- Multi-tenant Organisation -> Project -> LRS model.
- Session-based admin API and Basic Auth xAPI client flow.
- React admin UI for statements, users, clients, projects, LRS instances, plugins, settings, and profile management.
- Migration, Docker, Bruno, and Makefile-based development workflows.
