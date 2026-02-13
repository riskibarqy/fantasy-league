SHELL := /bin/bash

# Runtime profile (dev|stage|prod)
APP_ENV ?= dev
ENV_PROFILE_FILE := .env.$(APP_ENV)

# Load profile env first, then local .env overrides for developer machine.
ifneq (,$(wildcard $(ENV_PROFILE_FILE)))
include $(ENV_PROFILE_FILE)
export
endif
ifneq (,$(wildcard .env))
include .env
export
endif

GO ?= go
APP ?= ./cmd/api
BINARY ?= ./bin/fantasy-league-api
MIGRATIONS_DIR ?= ./db/migrations
DB_URL ?= postgres://postgres:postgres@localhost:5432/fantasy_league?sslmode=disable
PPROF_URL ?= http://localhost:6060/debug/pprof
PPROF_SECONDS ?= 30
name ?= new_migration

.PHONY: help run run-dev run-stage run-prod build test tidy fmt pprof-cpu pprof-heap pprof-goroutine pprof-allocs migrate-up migrate-down migrate-version migrate-force migrate-create check-migrate

help:
	@echo "Available targets:"
	@echo "  make run APP_ENV=dev - run API using profile env"
	@echo "  make run-dev         - run API with dev profile"
	@echo "  make run-stage       - run API with stage profile"
	@echo "  make run-prod        - run API with prod profile"
	@echo "  make run             - run API server"
	@echo "  make build           - build API binary"
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

test:
	$(GO) test ./...

tidy:
	$(GO) mod tidy

fmt:
	gofmt -w $$(rg --files -g '*.go')

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

migrate-create: check-migrate
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)
