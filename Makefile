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

GO ?= go
PNPM ?= pnpm
DOCKER ?= docker
COMPOSE ?= docker compose

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
dev-api: ## Run API locally (go run)
	@cd $(API_DIR) && $(GO) run ./cmd/api

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