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

# Default if not set anywhere (shell or .env)
PG_DSN ?= postgres://lrsolid:lrsolid@localhost:5432/lrsolid?sslmode=disable

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show available targets
	@awk 'BEGIN {FS = ":.*##"; printf "\nTargets:\n"} /^[a-zA-Z0-9_\-]+:.*##/ {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2} END {printf "\n"}' $(MAKEFILE_LIST)

# -----------------------
# Tooling / Setup
# -----------------------
.PHONY: deps
deps: ## Install dependencies (Go + Web)
	@echo "==> Go deps"
	@cd $(API_DIR) && $(GO) mod download
	@echo "==> Web deps"
	@cd $(WEB_DIR) && $(PNPM) install

.PHONY: tidy
tidy: ## Go: go mod tidy
	@cd $(API_DIR) && $(GO) mod tidy

# -----------------------
# Formatting / Linting
# -----------------------
.PHONY: fmt
fmt: ## Format Go + Web
	@echo "==> Go fmt"
	@cd $(API_DIR) && $(GO) fmt ./...
	@echo "==> Web format (if configured)"
	@cd $(WEB_DIR) && $(PNPM) run -s format || true

.PHONY: lint
lint: ## Lint Go + Web (if configured)
	@echo "==> Go vet"
	@cd $(API_DIR) && $(GO) vet ./...
	@echo "==> Web lint (if configured)"
	@cd $(WEB_DIR) && $(PNPM) run -s lint || true

# -----------------------
# Testing
# -----------------------
.PHONY: test _test_api _test_web
test: ## Run tests: make test [api|web]
	@case "$(word 2,$(MAKECMDGOALS))" in \
		api) $(MAKE) --no-print-directory _test_api ;; \
		web) $(MAKE) --no-print-directory _test_web ;; \
		"") $(MAKE) --no-print-directory _test_api _test_web ;; \
		*) echo "Usage: make test [api|web]"; exit 2 ;; \
	esac

_test_api:
	@cd $(API_DIR) && $(GO) test ./... -count=1

_test_web:
	@cd $(WEB_DIR) && $(PNPM) run -s test

# -----------------------
# Build
# -----------------------
.PHONY: build _build_api _build_web
build: ## Build applications: make build [api|web]
	@if [[ " $(MAKECMDGOALS) " == *" dev "* || " $(MAKECMDGOALS) " == *" docker "* || " $(MAKECMDGOALS) " == *" up "* ]]; then \
		exit 0; \
	fi; \
	case "$(word 2,$(MAKECMDGOALS))" in \
		api) $(MAKE) --no-print-directory _build_api ;; \
		web) $(MAKE) --no-print-directory _build_web ;; \
		"") $(MAKE) --no-print-directory _build_api _build_web ;; \
		*) echo "Usage: make build [api|web]"; exit 2 ;; \
	esac

_build_api:
	@mkdir -p $(API_DIR)/bin
	@cd $(API_DIR) && $(GO) build -o bin/api ./cmd/api

_build_web:
	@cd $(WEB_DIR) && $(PNPM) run build

# -----------------------
# Dev / Run
# -----------------------
.PHONY: dev
dev: ## Development stack: make dev [detached|down|logs [api|web]|build|psql|restart api]
	@action="$(word 2,$(MAKECMDGOALS))"; service="$(word 3,$(MAKECMDGOALS))"; \
	case "$$action:$$service" in \
		:) $(DEV_COMPOSE) up ;; \
		detached:) $(DEV_COMPOSE) up -d ;; \
		down:) $(DEV_COMPOSE) down ;; \
		logs:) $(DEV_COMPOSE) logs -f --tail=200 ;; \
		logs:api|logs:web) $(DEV_COMPOSE) logs -f --tail=200 "$$service" ;; \
		build:) $(DEV_COMPOSE) build ;; \
		psql:) $(DEV_COMPOSE) exec postgres psql -U lrsolid lrsolid ;; \
		restart:api) $(DEV_COMPOSE) restart api ;; \
		*) echo "Usage: make dev [detached|down|logs [api|web]|build|psql|restart api]"; exit 2 ;; \
	esac

AIR ?= $(shell go env GOPATH)/bin/air

# -----------------------
# Docker / Compose — Production
# -----------------------
.PHONY: up
up: ## Start all services (production, detached)
	@if [[ " $(MAKECMDGOALS) " != *" migrate "* && " $(MAKECMDGOALS) " != *" pg "* ]]; then \
		if [[ " $(MAKECMDGOALS) " == *" build "* ]]; then $(COMPOSE) up -d --build; else $(COMPOSE) up -d; fi; \
	fi

.PHONY: down
down: ## Stop and remove both development and production containers
	@if [[ " $(MAKECMDGOALS) " != *" migrate "* && " $(MAKECMDGOALS) " != *" pg "* && " $(MAKECMDGOALS) " != *" dev "* ]]; then \
		$(DEV_COMPOSE) down; \
		$(COMPOSE) down; \
	fi

.PHONY: logs
logs: ## Tail all service logs (production)
	@if [[ " $(MAKECMDGOALS) " != *" dev "* ]]; then \
		service="$(word 2,$(MAKECMDGOALS))"; \
		case "$$service" in "") $(COMPOSE) logs -f --tail=200 ;; api|web) $(COMPOSE) logs -f --tail=200 "$$service" ;; *) echo "Usage: make logs [api|web]"; exit 2 ;; esac; \
	fi

.PHONY: docker
docker: ## Docker images: make docker build|push
	@case "$(word 2,$(MAKECMDGOALS))" in \
		build) $(COMPOSE) build ;; \
		push) $(COMPOSE) push ;; \
		*) echo "Usage: make docker build|push"; exit 2 ;; \
	esac

DEV_COMPOSE := $(COMPOSE) -f docker-compose.dev.yml

# -----------------------
# Clean
# -----------------------
.PHONY: clean
clean: ## Clean build artifacts
	@rm -rf $(API_DIR)/bin
	@rm -rf $(WEB_DIR)/dist
	@rm -rf $(WEB_DIR)/node_modules/.vite

.PHONY: pg
pg: ## Standalone PostgreSQL: make pg up|down
	@case "$(word 2,$(MAKECMDGOALS))" in \
		up) $(DOCKER) run --name lrsolid-pg -e POSTGRES_USER=lrsolid -e POSTGRES_PASSWORD=lrsolid -e POSTGRES_DB=lrsolid -p 5432:5432 -d postgres:16 ;; \
		down) $(DOCKER) rm -f lrsolid-pg ;; \
		*) echo "Usage: make pg up|down"; exit 2 ;; \
	esac

.PHONY: migrate
migrate: ## Run migrations: make migrate up|down
	@action="$(word 2,$(MAKECMDGOALS))"; \
	case "$$action" in \
		up|down) PG_DSN="$(PG_DSN)" $(GO) run ./scripts/migrate --direction "$$action" --path "$(MIGRATIONS_DIR)" ;; \
		*) echo "Usage: make migrate up|down"; exit 2 ;; \
	esac

# `make migrate up/down` gives Make two goals. The up/down stack recipes detect
# the accompanying `migrate` goal and become no-ops after the migration runs.


# -----------------------
# xAPI smoke tests
# -----------------------
BASE_URL ?= http://localhost:8080/xapi
AUTH ?= clientKey:clientSecret

# Use uuidgen if available, otherwise fallback to a timestamp-based id
STATEMENT_ID ?= $(shell (uuidgen 2>/dev/null | tr '[:upper:]' '[:lower:]') || echo "st-$$(date +%s)")

.PHONY: xapi _xapi_post _xapi_get
xapi: ## xAPI smoke requests: make xapi post|get|smoke
	@case "$(word 2,$(MAKECMDGOALS))" in \
		post) $(MAKE) --no-print-directory _xapi_post ;; \
		get) $(MAKE) --no-print-directory _xapi_get ;; \
		smoke) $(MAKE) --no-print-directory _xapi_post _xapi_get ;; \
		*) echo "Usage: make xapi post|get|smoke"; exit 2 ;; \
	esac

_xapi_post:
	@curl -sS -u "$(AUTH)" \
		-H "Content-Type: application/json" \
		-H "X-Experience-API-Version: 1.0.3" \
		-X POST "$(BASE_URL)/statements" \
		-d '{"statements":[{"id":"$(STATEMENT_ID)","actor":{"mbox":"mailto:sergej@example.org"},"verb":{"id":"http://adlnet.gov/expapi/verbs/experienced","display":{"en-US":"experienced"}},"object":{"id":"https://example.org/activity/intro"}}]}'
	@echo ""

_xapi_get:
	@curl -sS -u "$(AUTH)" \
		-H "X-Experience-API-Version: 1.0.3" \
		"$(BASE_URL)/statements?statementId=$(STATEMENT_ID)"
	@echo ""

.PHONY: _create_account
_create_account:
	@test -n "$(EMAIL)" || (echo "Missing EMAIL=..."; exit 1)
	@test -n "$(PASSWORD)" || (echo "Missing PASSWORD=..."; exit 1)
	@PG_DSN="$(PG_DSN)" $(GO) run ./scripts/create_admin_user.go \
		--email "$(EMAIL)" \
		--password "$(PASSWORD)" \
		--org "$(or $(ORG),default)" \
		--role "$(ROLE)"

.PHONY: admin owner user
admin: ## Create/update an admin: make admin EMAIL=you@example.com PASSWORD=secret [ORG=default]
	@$(MAKE) --no-print-directory _create_account EMAIL="$(EMAIL)" PASSWORD="$(PASSWORD)" ORG="$(ORG)" ROLE="admin"

owner: ## Create/update an owner: make owner EMAIL=you@example.com PASSWORD=secret [ORG=default]
	@$(MAKE) --no-print-directory _create_account EMAIL="$(EMAIL)" PASSWORD="$(PASSWORD)" ORG="$(ORG)" ROLE="owner"

user: ## Create/update a normal user: make user EMAIL=you@example.com PASSWORD=secret [ORG=default]
	@$(MAKE) --no-print-directory _create_account EMAIL="$(EMAIL)" PASSWORD="$(PASSWORD)" ORG="$(ORG)" ROLE="member"

# Words consumed by grouped commands. Their work is performed by the dispatcher.
.PHONY: api web detached restart psql post get smoke push
api web detached restart psql post get smoke push:
	@:
