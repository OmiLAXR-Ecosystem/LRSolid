# LRSolid Monorepo Makefile
# Usage: make <target>
# Notes:
# - Go API lives in apps/api
# - Web UI lives in apps/web
# - Docker Compose (optional) for local infra

SHELL := /bin/bash

ROOT := $(CURDIR)
API_DIR := $(ROOT)/apps/api
WEB_DIR := $(ROOT)/apps/web
MIGRATIONS_DIR := $(ROOT)/migrations

GO ?= go
PNPM ?= pnpm
DOCKER ?= docker
COMPOSE ?= docker compose

# --- Env loading (optional) ---
# Loads variables from .env if present (exported to recipes)
ifneq (,$(wildcard .env))
include .env
export
endif

# Local development defaults. They match docker-compose.dev.yml and can be
# overridden in the shell or .env file.
DEV_POSTGRES_PORT ?= 5433
DEV_API_PORT ?= 8081
DEV_WEB_PORT ?= 5174
DEV_S3_PORT ?= 9000
DEV_S3_CONSOLE_PORT ?= 9001
DEV_S3_ACCESS_KEY ?= rustfsadmin
DEV_S3_SECRET_KEY ?= rustfsadmin
DEV_S3_BUCKET ?= lrsolid
DEV_PG_DSN ?= postgres://lrsolid:lrsolid@localhost:$(DEV_POSTGRES_PORT)/lrsolid?sslmode=disable
PG_DSN ?= $(DEV_PG_DSN)

.DEFAULT_GOAL := help

# Command reference. Commands are parsed from MAKECMDGOALS so that grouped
# forms such as `make dev api` can be used with GNU Make.
# HELP: admin create | Create/update an admin user (EMAIL=... PASSWORD=... ORG=... ROLE=owner)
# HELP: build | Build API binary + Web bundle
# HELP: build api | Build API binary (apps/api/bin/api)
# HELP: build web | Build Web bundle (Vite build)
# HELP: clean | Clean build artifacts
# HELP: user create | Create/update a user (EMAIL=... PASSWORD=... [ORG=default] [ROLE=owner])
# HELP: deps | Install dependencies (Go + Web)
# HELP: dev | Start the complete dev stack (Postgres, API, and Web)
# HELP: dev api | Run API locally with hot reload via air (requires PG_DSN)
# HELP: dev build | Rebuild dev images (e.g. after Dockerfile.dev changes)
# HELP: dev logs | Tail all dev service logs
# HELP: dev logs api | Tail API dev logs
# HELP: dev logs web | Tail Web dev logs (Vite)
# HELP: dev psql | Open psql shell in dev postgres container
# HELP: dev restart api | Restart only the API container (triggers air rebuild)
# HELP: dev up | Start dev stack (API live reload via air, Vite HMR)
# HELP: dev up detached | Start dev stack detached
# HELP: dev web | Run Web locally (pnpm dev)
# HELP: docker build | Build all docker images (production)
# HELP: docker push | Push images to registry (set REGISTRY=... first)
# HELP: down | Stop and remove development and production containers
# HELP: down volumes | Stop both stacks and remove their volumes
# HELP: fmt | Format Go + Web
# HELP: help | Show available commands
# HELP: lint | Lint Go + Web (if configured)
# HELP: logs | Tail all service logs (production)
# HELP: logs api | Tail API logs (production)
# HELP: migrate down | Roll back one DB migration
# HELP: migrate up | Run DB migrations up
# HELP: pg down | Stop/remove local Postgres container
# HELP: pg up | Start local Postgres (docker)
# HELP: s3 down | Stop/remove local S3-compatible RustFS container
# HELP: s3 logs | Tail local S3-compatible RustFS logs
# HELP: s3 up | Start local S3-compatible RustFS
# HELP: test | Run unit tests (Go + Web if configured)
# HELP: test api | Run Go tests only
# HELP: test web | Run Web tests only (if configured)
# HELP: tidy | Go: go mod tidy
# HELP: up | Start all services (production, detached)
# HELP: up build | Build images and start all services (production)
# HELP: xapi get | GET a statement by STATEMENT_ID
# HELP: xapi post | POST a test statement batch, prints returned IDs
# HELP: xapi smoke | POST then GET same statement id

.PHONY: help
help: ## Show available targets
	@{ \
		printf "\nCommands:\n"; \
		awk -F ' \\| ' '/^# HELP:/ {sub(/^# HELP: /, ""); printf "%s\t%s\n", $$1, $$2}' $(MAKEFILE_LIST) | \
			sort -k1,1 | \
			awk -F '\t' '{printf "  \033[36m%-32s\033[0m %s\n", "make " $$1, $$2}'; \
		printf "\nDevelopment endpoints:\n"; \
		printf "  Web         http://localhost:%s  (DEV_WEB_PORT)\n" "$(DEV_WEB_PORT)"; \
		printf "  API         http://localhost:%s  (DEV_API_PORT)\n" "$(DEV_API_PORT)"; \
		printf "  PostgreSQL  localhost:%s         (DEV_POSTGRES_PORT)\n" "$(DEV_POSTGRES_PORT)"; \
		printf "  S3 API      http://localhost:%s  (DEV_S3_PORT)\n" "$(DEV_S3_PORT)"; \
		printf "  S3 Console  http://localhost:%s  (DEV_S3_CONSOLE_PORT)\n" "$(DEV_S3_CONSOLE_PORT)"; \
		printf "\nOverride example:\n"; \
		printf "  DEV_POSTGRES_PORT=55432 DEV_API_PORT=18080 DEV_WEB_PORT=15173 make dev\n\n"; \
	}

COMMAND_GOALS := $(strip $(MAKECMDGOALS))
COMMAND_WORDS := admin api build clean create deps dev detached docker down fmt get help lint logs migrate pg post psql push restart s3 smoke test tidy up user volumes web xapi

.PHONY: $(COMMAND_WORDS) __dispatch
$(filter-out help,$(COMMAND_WORDS)): __dispatch
$(filter-out help,$(COMMAND_WORDS)):
	@:

__dispatch:
	@case "$(COMMAND_GOALS)" in \
		"admin create") $(MAKE) --no-print-directory __admin_create ;; \
		"build") $(MAKE) --no-print-directory __build ;; \
		"build api") $(MAKE) --no-print-directory __build_api ;; \
		"build web") $(MAKE) --no-print-directory __build_web ;; \
		"clean") $(MAKE) --no-print-directory __clean ;; \
		"deps") $(MAKE) --no-print-directory __deps ;; \
		"dev") $(MAKE) --no-print-directory __dev ;; \
		"dev api") $(MAKE) --no-print-directory __dev_api ;; \
		"dev build") $(MAKE) --no-print-directory __dev_build ;; \
		"dev logs") $(MAKE) --no-print-directory __dev_logs ;; \
		"dev logs api") $(MAKE) --no-print-directory __dev_logs_api ;; \
		"dev logs web") $(MAKE) --no-print-directory __dev_logs_web ;; \
		"dev psql") $(MAKE) --no-print-directory __dev_psql ;; \
		"dev restart api") $(MAKE) --no-print-directory __dev_restart_api ;; \
		"dev up") $(MAKE) --no-print-directory __dev_up ;; \
		"dev up detached") $(MAKE) --no-print-directory __dev_up_detached ;; \
		"dev web") $(MAKE) --no-print-directory __dev_web ;; \
		"docker build") $(MAKE) --no-print-directory __docker_build ;; \
		"docker push") $(MAKE) --no-print-directory __docker_push ;; \
		"down") $(MAKE) --no-print-directory __down ;; \
		"down volumes") $(MAKE) --no-print-directory __down_volumes ;; \
		"fmt") $(MAKE) --no-print-directory __fmt ;; \
		"lint") $(MAKE) --no-print-directory __lint ;; \
		"logs") $(MAKE) --no-print-directory __logs ;; \
		"logs api") $(MAKE) --no-print-directory __logs_api ;; \
		"migrate down") $(MAKE) --no-print-directory __migrate_down ;; \
		"migrate up") $(MAKE) --no-print-directory __migrate_up ;; \
		"pg down") $(MAKE) --no-print-directory __pg_down ;; \
		"pg up") $(MAKE) --no-print-directory __pg_up ;; \
		"s3 down") $(MAKE) --no-print-directory __s3_down ;; \
		"s3 logs") $(MAKE) --no-print-directory __s3_logs ;; \
		"s3 up") $(MAKE) --no-print-directory __s3_up ;; \
		"test") $(MAKE) --no-print-directory __test ;; \
		"test api") $(MAKE) --no-print-directory __test_api ;; \
		"test web") $(MAKE) --no-print-directory __test_web ;; \
		"tidy") $(MAKE) --no-print-directory __tidy ;; \
		"up") $(MAKE) --no-print-directory __up ;; \
		"up build") $(MAKE) --no-print-directory __up_build ;; \
		"user create") $(MAKE) --no-print-directory __user_create ;; \
		"xapi get") $(MAKE) --no-print-directory __xapi_get ;; \
		"xapi post") $(MAKE) --no-print-directory __xapi_post ;; \
		"xapi smoke") $(MAKE) --no-print-directory __xapi_smoke ;; \
		*) echo "Unknown command: make $(COMMAND_GOALS)" >&2; exit 2 ;; \
	esac

# -----------------------
# Tooling / Setup
# -----------------------
.PHONY: __deps
__deps:
	@echo "==> Go deps"
	@cd $(API_DIR) && $(GO) mod download
	@echo "==> Web deps"
	@cd $(WEB_DIR) && $(PNPM) install

.PHONY: __tidy
__tidy:
	@cd $(API_DIR) && $(GO) mod tidy

# -----------------------
# Formatting / Linting
# -----------------------
.PHONY: __fmt
__fmt:
	@echo "==> Go fmt"
	@cd $(API_DIR) && $(GO) fmt ./...
	@echo "==> Web format (if configured)"
	@cd $(WEB_DIR) && $(PNPM) run -s format || true

.PHONY: __lint
__lint:
	@echo "==> Go vet"
	@cd $(API_DIR) && $(GO) vet ./...
	@echo "==> Web lint (if configured)"
	@cd $(WEB_DIR) && $(PNPM) run -s lint || true

# -----------------------
# Testing
# -----------------------
.PHONY: __test
__test:
	@echo "==> Go test"
	@cd $(API_DIR) && $(GO) test ./... -count=1
	@echo "==> Web test (if configured)"
	@cd $(WEB_DIR) && $(PNPM) run -s test || true

.PHONY: __test_api
__test_api:
	@cd $(API_DIR) && $(GO) test ./... -count=1

.PHONY: __test_web
__test_web:
	@cd $(WEB_DIR) && $(PNPM) run -s test

# -----------------------
# Build
# -----------------------
.PHONY: __build
__build: __build_api __build_web

.PHONY: __build_api
__build_api:
	@mkdir -p $(API_DIR)/bin
	@cd $(API_DIR) && $(GO) build -o bin/api ./cmd/api

.PHONY: __build_web
__build_web:
	@cd $(WEB_DIR) && $(PNPM) run build

# -----------------------
# Dev / Run
# -----------------------
.PHONY: __dev
__dev: __dev_up

AIR ?= $(shell go env GOPATH)/bin/air

.PHONY: __dev_api
__dev_api:
	@test -x "$(AIR)" || (echo "air not found — install with: go install github.com/air-verse/air@latest" && exit 1)
	@cd $(ROOT) && PG_DSN="$(PG_DSN)" $(AIR)

.PHONY: __dev_web
__dev_web:
	@cd $(WEB_DIR) && $(PNPM) dev

# -----------------------
# Docker / Compose — Production
# -----------------------
.PHONY: __up
__up:
	@$(COMPOSE) up -d

.PHONY: __up_build
__up_build:
	@$(COMPOSE) up -d --build

.PHONY: __down
__down:
	@$(COMPOSE) down
	@$(DEV_COMPOSE) down

.PHONY: __down_volumes
__down_volumes:
	@$(COMPOSE) down -v
	@$(DEV_COMPOSE) down -v

.PHONY: __logs
__logs:
	@$(COMPOSE) logs -f --tail=200

.PHONY: __logs_api
__logs_api:
	@$(COMPOSE) logs -f --tail=200 api

.PHONY: __docker_build
__docker_build:
	@$(COMPOSE) build

.PHONY: __docker_push
__docker_push:
	@$(COMPOSE) push

# -----------------------
# Docker / Compose — Development (live reload)
# -----------------------
DEV_COMPOSE := $(COMPOSE) -f docker-compose.dev.yml

.PHONY: __dev_up
__dev_up:
	@$(DEV_COMPOSE) up

.PHONY: __dev_up_detached
__dev_up_detached:
	@$(DEV_COMPOSE) up -d

.PHONY: __dev_logs
__dev_logs:
	@$(DEV_COMPOSE) logs -f --tail=200

.PHONY: __dev_logs_api
__dev_logs_api:
	@$(DEV_COMPOSE) logs -f --tail=200 api

.PHONY: __dev_logs_web
__dev_logs_web:
	@$(DEV_COMPOSE) logs -f --tail=200 web

.PHONY: __dev_build
__dev_build:
	@$(DEV_COMPOSE) build

.PHONY: __dev_psql
__dev_psql:
	@$(DEV_COMPOSE) exec postgres psql -U lrsolid lrsolid

.PHONY: __dev_restart_api
__dev_restart_api:
	@$(DEV_COMPOSE) restart api

.PHONY: __s3_up
__s3_up:
	@$(DEV_COMPOSE) up -d --remove-orphans rustfs
	@$(DEV_COMPOSE) run --rm rustfs-init
	@echo "Local S3 API:     http://localhost:$(DEV_S3_PORT)"
	@echo "Local S3 Console: http://localhost:$(DEV_S3_CONSOLE_PORT)"
	@echo "Bucket:           $(DEV_S3_BUCKET)"
	@echo "Access key:       $(DEV_S3_ACCESS_KEY)"
	@echo "Secret key:       $(DEV_S3_SECRET_KEY)"
	@echo "Export upload:    http://rustfs:9000/$(DEV_S3_BUCKET) (private, API uses signed requests)"

.PHONY: __s3_down
__s3_down:
	@$(DEV_COMPOSE) stop rustfs || true
	@$(DEV_COMPOSE) rm -f -s rustfs || true

.PHONY: __s3_logs
__s3_logs:
	@$(DEV_COMPOSE) logs -f --tail=200 rustfs

# -----------------------
# Clean
# -----------------------
.PHONY: __clean
__clean:
	@rm -rf $(API_DIR)/bin
	@rm -rf $(WEB_DIR)/dist
	@rm -rf $(WEB_DIR)/node_modules/.vite

.PHONY: __pg_up
__pg_up:
	@docker run --name lrsolid-pg \
		-e POSTGRES_USER=lrsolid \
		-e POSTGRES_PASSWORD=lrsolid \
		-e POSTGRES_DB=lrsolid \
		-p $(DEV_POSTGRES_PORT):5432 \
		-d postgres:16 || true

.PHONY: __pg_down
__pg_down:
	@docker rm -f lrsolid-pg || true

.PHONY: __migrate_up
__migrate_up:
	@$(GO) run ./scripts/migrate -database "$(PG_DSN)" -path "$(MIGRATIONS_DIR)" up

.PHONY: __migrate_down
__migrate_down:
	@$(GO) run ./scripts/migrate -database "$(PG_DSN)" -path "$(MIGRATIONS_DIR)" down


# -----------------------
# xAPI smoke tests
# -----------------------
BASE_URL ?= http://localhost:$(DEV_API_PORT)/xapi
AUTH ?= clientKey:clientSecret

# Use uuidgen if available, otherwise fallback to a timestamp-based id
STATEMENT_ID ?= $(shell (uuidgen 2>/dev/null | tr '[:upper:]' '[:lower:]') || echo "st-$$(date +%s)")

.PHONY: __xapi_post
__xapi_post:
	@curl -sS -u "$(AUTH)" \
		-H "Content-Type: application/json" \
		-H "X-Experience-API-Version: 1.0.3" \
		-X POST "$(BASE_URL)/statements" \
		-d '{"statements":[{"id":"$(STATEMENT_ID)","actor":{"mbox":"mailto:sergej@example.org"},"verb":{"id":"http://adlnet.gov/expapi/verbs/experienced","display":{"en-US":"experienced"}},"object":{"id":"https://example.org/activity/intro"}}]}'
	@echo ""

.PHONY: __xapi_get
__xapi_get:
	@curl -sS -u "$(AUTH)" \
		-H "X-Experience-API-Version: 1.0.3" \
		"$(BASE_URL)/statements?statementId=$(STATEMENT_ID)"
	@echo ""

.PHONY: __xapi_smoke
__xapi_smoke:
	@$(MAKE) --no-print-directory __xapi_post STATEMENT_ID=$(STATEMENT_ID)
	@$(MAKE) --no-print-directory __xapi_get  STATEMENT_ID=$(STATEMENT_ID)

.PHONY: __admin_create
__admin_create:
	@test -n "$(EMAIL)" || (echo "Missing EMAIL=..."; exit 1)
	@test -n "$(PASSWORD)" || (echo "Missing PASSWORD=..."; exit 1)
	@PG_DSN="$(PG_DSN)" $(GO) run ./scripts/create_admin_user.go \
		--email "$(EMAIL)" \
		--password "$(PASSWORD)" \
		--org "$(or $(ORG),default)" \
		--role "$(ROLE)"

.PHONY: __user_create
__user_create:
	@$(MAKE) --no-print-directory __admin_create EMAIL="$(EMAIL)" PASSWORD="$(PASSWORD)" ORG="$(ORG)" ROLE="$(ROLE)"
