SHELL := /bin/bash

# Runtime profile (dev|stage|prod)
APP_ENV ?= dev
ENV_PROFILE_FILE := .env.$(APP_ENV)
USE_LOCAL_ENV ?= true

# Load profile env first, then local .env overrides for developer machine.
ifneq (,$(wildcard $(ENV_PROFILE_FILE)))
include $(ENV_PROFILE_FILE)
export
endif
ifeq ($(USE_LOCAL_ENV),true)
ifneq (,$(wildcard .env))
include .env
export
endif
endif

GO ?= go
APP ?= ./cmd/api
BINARY ?= ./bin/fantasy-league-api
MIGRATION_APP ?= ./cmd/migration
MIGRATION_BINARY ?= ./bin/fantasy-league-migrate
MIGRATIONS_DIR ?= ./db/migrations
DB_URL ?= postgres://postgres:postgres@localhost:5432/fantasy_league?sslmode=disable
PPROF_URL ?= http://localhost:6060/debug/pprof
PPROF_SECONDS ?= 30
name ?= new_migration

.PHONY: help run run-dev run-stage run-prod build build-migration test tidy fmt pprof-cpu pprof-heap pprof-goroutine pprof-allocs migrate-up migrate-down migrate-version migrate-force migrate-create migrate-app-up migrate-app-down migrate-app-version migrate-app-force migrate-app-goto check-migrate fly-secrets fly-deploy fly-migrate-up fly-migrate-down fly-migrate-version fly-migrate-force

help:
	@echo "Available targets:"
	@echo "  make run APP_ENV=dev - run API using profile env"
	@echo "  make run-dev         - run API with dev profile"
	@echo "  make run-stage       - run API with stage profile"
	@echo "  make run-prod        - run API with prod profile"
	@echo "  make run             - run API server"
	@echo "  make build           - build API binary"
	@echo "  make build-migration - build migration binary"
	@echo "  make test            - run all tests"
	@echo "  make tidy            - tidy go modules"
	@echo "  make fmt             - format all Go files"
	@echo "  make pprof-cpu       - open CPU profile from pprof endpoint"
	@echo "  make pprof-heap      - open heap profile from pprof endpoint"
	@echo "  make pprof-goroutine - open goroutine profile from pprof endpoint"
	@echo "  make pprof-allocs    - open allocs profile from pprof endpoint"
	@echo "  make migrate-up      - apply all migrations (requires DB_URL and migrate CLI)"
	@echo "  make migrate-down    - rollback 1 migration"
	@echo "  make migrate-version - show current migration version"
	@echo "  make migrate-force version=1 - force migration version"
	@echo "  make migrate-create name=add_table - create new migration files"
	@echo "  make migrate-app-up  - run migration binary (up)"
	@echo "  make migrate-app-down steps=1 - run migration binary (down)"
	@echo "  make migrate-app-version - run migration binary (version)"
	@echo "  make migrate-app-force version=1 - run migration binary (force)"
	@echo "  make migrate-app-goto version=1 - run migration binary (goto)"
	@echo "  make fly-secrets     - set Fly secrets from env (FLY_APP required)"
	@echo "  make fly-deploy      - deploy to Fly (FLY_APP required)"
	@echo "  make fly-migrate-up  - run migrations in Fly machine"
	@echo "  make fly-migrate-down steps=1 - rollback migrations in Fly machine"
	@echo "  make fly-migrate-version - show migration version in Fly machine"
	@echo "  make fly-migrate-force version=1 - force migration version in Fly machine"

run:
	@echo "Running with APP_ENV=$(APP_ENV)"
	$(GO) run $(APP)

run-dev:
	@APP_ENV=dev $(MAKE) run

run-stage:
	@APP_ENV=stage $(MAKE) run

run-prod:
	@APP_ENV=prod $(MAKE) run

build:
	mkdir -p ./bin
	$(GO) build -o $(BINARY) $(APP)

build-migration:
	mkdir -p ./bin
	$(GO) build -o $(MIGRATION_BINARY) $(MIGRATION_APP)

test:
	$(GO) test ./...

tidy:
	$(GO) mod tidy

fmt:
	@files="$$(if command -v rg >/dev/null 2>&1; then rg --files -g '*.go'; else find . -type f -name '*.go' -not -path './vendor/*'; fi)"; \
	if [ -z "$$files" ]; then \
		echo "no Go files found"; \
	else \
		gofmt -w $$files; \
	fi

pprof-cpu:
	@echo "Fetching CPU profile from $(PPROF_URL)/profile?seconds=$(PPROF_SECONDS)"
	$(GO) tool pprof -http=:0 "$(PPROF_URL)/profile?seconds=$(PPROF_SECONDS)"

pprof-heap:
	$(GO) tool pprof -http=:0 "$(PPROF_URL)/heap"

pprof-goroutine:
	$(GO) tool pprof -http=:0 "$(PPROF_URL)/goroutine"

pprof-allocs:
	$(GO) tool pprof -http=:0 "$(PPROF_URL)/allocs"

check-migrate:
	@command -v migrate >/dev/null 2>&1 || (echo "migrate CLI not found. Install: https://github.com/golang-migrate/migrate/tree/master/cmd/migrate" && exit 1)

migrate-up: check-migrate
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up

migrate-down: check-migrate
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down 1

migrate-version: check-migrate
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" version

migrate-force: check-migrate
	@test -n "$(version)" || (echo "version is required. Usage: make migrate-force version=1" && exit 1)
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" force $(version)

migrate-create:
	@version=$$(date +%s); \
	up="$(MIGRATIONS_DIR)/$${version}_$(name).up.sql"; \
	down="$(MIGRATIONS_DIR)/$${version}_$(name).down.sql"; \
	while [ -e "$$up" ] || [ -e "$$down" ]; do \
		version=$$((version+1)); \
		up="$(MIGRATIONS_DIR)/$${version}_$(name).up.sql"; \
		down="$(MIGRATIONS_DIR)/$${version}_$(name).down.sql"; \
	done; \
	touch "$$up" "$$down"; \
	echo "created $$up and $$down"

migrate-app-up:
	DB_URL="$(DB_URL)" MIGRATIONS_DIR="$(MIGRATIONS_DIR)" $(GO) run $(MIGRATION_APP) up

migrate-app-down:
	@steps="$${steps:-1}"; \
	DB_URL="$(DB_URL)" MIGRATIONS_DIR="$(MIGRATIONS_DIR)" $(GO) run $(MIGRATION_APP) down "$$steps"

migrate-app-version:
	DB_URL="$(DB_URL)" MIGRATIONS_DIR="$(MIGRATIONS_DIR)" $(GO) run $(MIGRATION_APP) version

migrate-app-force:
	@test -n "$(version)" || (echo "version is required. Usage: make migrate-app-force version=1" && exit 1)
	DB_URL="$(DB_URL)" MIGRATIONS_DIR="$(MIGRATIONS_DIR)" $(GO) run $(MIGRATION_APP) force "$(version)"

migrate-app-goto:
	@test -n "$(version)" || (echo "version is required. Usage: make migrate-app-goto version=1" && exit 1)
	DB_URL="$(DB_URL)" MIGRATIONS_DIR="$(MIGRATIONS_DIR)" $(GO) run $(MIGRATION_APP) goto "$(version)"

fly-secrets:
	@test -n "$$FLY_APP" || (echo "FLY_APP is required (e.g. export FLY_APP=fantasy-league-rw84mq)"; exit 1)
	@test -n "$$DB_URL" || (echo "DB_URL is required"; exit 1)
	@test -n "$$ANUBIS_BASE_URL" || (echo "ANUBIS_BASE_URL is required"; exit 1)
	@test -n "$$ANUBIS_ADMIN_KEY" || (echo "ANUBIS_ADMIN_KEY is required"; exit 1)
	@cmd=(fly secrets set -a "$$FLY_APP" \
		"DB_URL=$$DB_URL" \
		"ANUBIS_BASE_URL=$$ANUBIS_BASE_URL" \
		"ANUBIS_ADMIN_KEY=$$ANUBIS_ADMIN_KEY"); \
	if [ -n "$$UPTRACE_DSN" ]; then cmd+=("UPTRACE_DSN=$$UPTRACE_DSN"); fi; \
	if [ -n "$$PYROSCOPE_SERVER_ADDRESS" ]; then cmd+=("PYROSCOPE_SERVER_ADDRESS=$$PYROSCOPE_SERVER_ADDRESS"); fi; \
	if [ -n "$$PYROSCOPE_AUTH_TOKEN" ]; then cmd+=("PYROSCOPE_AUTH_TOKEN=$$PYROSCOPE_AUTH_TOKEN"); fi; \
	"$${cmd[@]}"

fly-deploy:
	@test -n "$$FLY_APP" || (echo "FLY_APP is required (e.g. export FLY_APP=fantasy-league-rw84mq)"; exit 1)
	fly deploy -a "$$FLY_APP"

fly-migrate-up:
	@test -n "$$FLY_APP" || (echo "FLY_APP is required (e.g. export FLY_APP=fantasy-league)"; exit 1)
	fly ssh console -a "$$FLY_APP" -C "/app/fantasy-league-migrate up"

fly-migrate-down:
	@test -n "$$FLY_APP" || (echo "FLY_APP is required (e.g. export FLY_APP=fantasy-league)"; exit 1)
	@steps="$${steps:-1}"; \
	fly ssh console -a "$$FLY_APP" -C "/app/fantasy-league-migrate down $$steps"

fly-migrate-version:
	@test -n "$$FLY_APP" || (echo "FLY_APP is required (e.g. export FLY_APP=fantasy-league)"; exit 1)
	fly ssh console -a "$$FLY_APP" -C "/app/fantasy-league-migrate version"

fly-migrate-force:
	@test -n "$$FLY_APP" || (echo "FLY_APP is required (e.g. export FLY_APP=fantasy-league)"; exit 1)
	@test -n "$(version)" || (echo "version is required. Usage: make fly-migrate-force version=1" && exit 1)
	fly ssh console -a "$$FLY_APP" -C "/app/fantasy-league-migrate force $(version)"
