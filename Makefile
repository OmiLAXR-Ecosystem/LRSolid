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
PG_DSN ?= postgres://postgres:postgres@localhost:5432/lrsolid?sslmode=disable

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
.PHONY: test
test: ## Run unit tests (Go + Web if configured)
	@echo "==> Go test"
	@cd $(API_DIR) && $(GO) test ./... -count=1
	@echo "==> Web test (if configured)"
	@cd $(WEB_DIR) && $(PNPM) run -s test || true

.PHONY: test-api
test-api: ## Run Go tests only
	@cd $(API_DIR) && $(GO) test ./... -count=1

.PHONY: test-web
test-web: ## Run Web tests only (if configured)
	@cd $(WEB_DIR) && $(PNPM) run -s test

# -----------------------
# Build
# -----------------------
.PHONY: build
build: build-api build-web ## Build API binary + Web bundle

.PHONY: build-api
build-api: ## Build API binary (apps/api/bin/api)
	@mkdir -p $(API_DIR)/bin
	@cd $(API_DIR) && $(GO) build -o bin/api ./cmd/api

.PHONY: build-web
build-web: ## Build Web bundle (Vite build)
	@cd $(WEB_DIR) && $(PNPM) run build

# -----------------------
# Dev / Run
# -----------------------
.PHONY: dev
dev: ## Run API + Web in dev (two terminals recommended)
	@echo "Run these in two terminals:"
	@echo "  make dev-api"
	@echo "  make dev-web"

.PHONY: dev-api
dev-api: ## Run API locally (requires PG_DSN)
	@cd $(ROOT) && PG_DSN="$(PG_DSN)" $(GO) run ./apps/api

.PHONY: dev-web
dev-web: ## Run Web locally (pnpm dev)
	@cd $(WEB_DIR) && $(PNPM) dev

# -----------------------
# Docker / Compose
# -----------------------
.PHONY: up
up: ## Docker compose up (local infra/app if compose.yml present)
	@$(COMPOSE) up -d

.PHONY: down
down: ## Docker compose down
	@$(COMPOSE) down

.PHONY: logs
logs: ## Tail docker compose logs
	@$(COMPOSE) logs -f --tail=200

.PHONY: docker-build
docker-build: ## Build docker images (compose build)
	@$(COMPOSE) build

# -----------------------
# Clean
# -----------------------
.PHONY: clean
clean: ## Clean build artifacts
	@rm -rf $(API_DIR)/bin
	@rm -rf $(WEB_DIR)/dist
	@rm -rf $(WEB_DIR)/node_modules/.vite

pg-up: ## Start local Postgres (docker)
	@docker run --name lrsolid-pg -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=lrsolid -p 5432:5432 -d postgres:16 || true

pg-down: ## Stop/remove local Postgres container
	@docker rm -f lrsolid-pg || true

migrate-up: ## Run DB migrations up
	@migrate -database "$(PG_DSN)" -path "$(MIGRATIONS_DIR)" up

migrate-down: ## Run DB migrations up
	@migrate -database "$(PG_DSN)" -path "$(MIGRATIONS_DIR)" down


# -----------------------
# xAPI smoke tests
# -----------------------
BASE_URL ?= http://localhost:8080/xapi
AUTH ?= clientKey:clientSecret

# Use uuidgen if available, otherwise fallback to a timestamp-based id
STATEMENT_ID ?= $(shell (uuidgen 2>/dev/null | tr '[:upper:]' '[:lower:]') || echo "st-$$(date +%s)")

.PHONY: xapi-post
xapi-post: ## POST a test statement batch, prints returned IDs
	@curl -sS -u "$(AUTH)" \
		-H "Content-Type: application/json" \
		-H "X-Experience-API-Version: 1.0.3" \
		-X POST "$(BASE_URL)/statements" \
		-d '{"statements":[{"id":"$(STATEMENT_ID)","actor":{"mbox":"mailto:sergej@example.org"},"verb":{"id":"http://adlnet.gov/expapi/verbs/experienced","display":{"en-US":"experienced"}},"object":{"id":"https://example.org/activity/intro"}}]}'
	@echo ""

.PHONY: xapi-get
xapi-get: ## GET a statement by STATEMENT_ID
	@curl -sS -u "$(AUTH)" \
		-H "X-Experience-API-Version: 1.0.3" \
		"$(BASE_URL)/statements?statementId=$(STATEMENT_ID)"
	@echo ""

.PHONY: xapi-smoke
xapi-smoke: ## POST then GET same statement id
	@$(MAKE) xapi-post STATEMENT_ID=$(STATEMENT_ID)
	@$(MAKE) xapi-get  STATEMENT_ID=$(STATEMENT_ID)

.PHONY: admin-create
admin-create: ## Create/update an admin user (EMAIL=... PASSWORD=... ORG=... ROLE=owner)
	@test -n "$(EMAIL)" || (echo "Missing EMAIL=..."; exit 1)
	@test -n "$(PASSWORD)" || (echo "Missing PASSWORD=..."; exit 1)
	@PG_DSN="$(PG_DSN)" $(GO) run ./scripts/create_admin_user.go \
		--email "$(EMAIL)" \
		--password "$(PASSWORD)" \
		--org "$(or $(ORG),default)" \
		--role "$(or $(ROLE),owner)"