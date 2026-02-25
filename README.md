# Fantasy League Backend (Indonesia-first, multi-league ready)

Backend service in Go for a fantasy football platform inspired by Premier League Fantasy, starting with Indonesian league teams and designed to support other leagues.

## Architecture

This project uses a pragmatic Clean Architecture split:

- `internal/domain`: core entities and business rules (DDD style)
- `internal/usecase`: application services and orchestration
- `internal/interfaces/httpapi`: HTTP transport layer
- `internal/infrastructure`: repository/account-service adapters
- `cmd/api`: composition root and runtime bootstrap

## Current Features

- Multi-league foundation (`/v1/leagues`)
- Dashboard endpoint for fantasy frontend
- Team, fixture, and player listing per league
- PAT lineup endpoints (11 starters + 4 substitutes)
- Authenticated squad creation/upsert
- Swagger/OpenAPI docs endpoint (`/docs`, `/openapi.yaml`)
- Uptrace/OpenTelemetry integration (configurable via env)
- pprof and Pyroscope profiling integration (configurable via env)
- PostgreSQL repositories implemented with `sqlx`
- API runtime served by `fasthttp` (adapter for existing handlers)
- Squad rule validation:
  - exact 11 players
  - budget cap
  - max players from same real club
  - minimum formation constraints
- Login/auth verification via Anubis account service (`../../rust/anubis`) through token introspection
- Resilience for auth dependency: circuit breaker + singleflight dedup on concurrent token introspection

## Assumptions for Anubis Integration

`ANUBIS_BASE_URL` and `ANUBIS_INTROSPECT_PATH` are configurable.

Default introspection request/response expected:

- Request: `POST {"token":"<bearer-token>"}`
- Header: `x-admin-key: <ANUBIS_ADMIN_KEY>`
- Success response (HTTP 200):

```json
{
  "active": true,
  "user_id": "8e4d9d2f-89f2-4e2f-8f69-08d8d0a31f9f",
  "app_id": "0be4d46b-7ab6-4f67-8dbf-f4ae5afbf5a1",
  "roles": ["app_admin"],
  "permissions": ["users.read"],
  "exp": 1730000000,
  "iat": 1729990000,
  "jti": "7fe8385a-2938-4c68-a9d0-c8658edcc0af"
}
```

If your Anubis response schema differs, adjust:

- `internal/infrastructure/account/anubis/client.go`

## Run

Using Makefile:

```bash
make run
```

`make` targets auto-load local `.env` if the file exists.
You can also run by profile:

```bash
make run-dev
make run-stage
make run-prod
```

Direct Go command:

```bash
go run ./cmd/api
```

Default server address: `:8080`

## Environment Profiles

Predefined profile files:

- `.env.dev`
- `.env.stage`
- `.env.prod`

Select profile via `APP_ENV`:

```bash
make run APP_ENV=stage
```

Local `.env` values (if present) override profile values.

## Profiling

Enable pprof and/or Pyroscope via env toggle:

- `PPROF_ENABLED=true` and `PPROF_ADDR=:6060`
- `PYROSCOPE_ENABLED=true` and `PYROSCOPE_SERVER_ADDRESS=http://localhost:4040`

Use Make targets for pprof:

```bash
make pprof-cpu
make pprof-heap
make pprof-goroutine
make pprof-allocs
```

Customize source endpoint:

```bash
PPROF_URL=http://localhost:6060/debug/pprof make pprof-heap
```

## Database Migrations

Migration files are in:

- `db/migrations/1771776000_create_touch_updated_at_function.up.sql`
- `db/migrations/1771776001_create_leagues.up.sql`

Run migrations (requires `golang-migrate` CLI and a PostgreSQL `DB_URL`):

```bash
make migrate-up
```

Or use the built-in Go migration runner (no external migrate CLI):

```bash
make migrate-app-up
make migrate-app-version
```

Rollback one step:

```bash
make migrate-down
```

Example:

```bash
DB_URL='postgres://postgres:postgres@localhost:5432/fantasy_league?sslmode=disable' make migrate-up
```

Default custom league data (Global + Indonesia only) is seeded as part of migration `1771776027_seed_default_custom_leagues`, so `make migrate-up` is enough.

## Fly.io Deployment

This repo now includes:

- `fly.toml`
- `Dockerfile`
- `.dockerignore`
- `make fly-secrets`
- `make fly-deploy`

Deploy steps (pattern aligned with `../../rust/anubis`):

1. Set required env vars:

```bash
export FLY_APP='fantasy-league-rw84mq'
export DB_URL='postgres://...'
export ANUBIS_BASE_URL='https://anubis.example.com'
export ANUBIS_ADMIN_KEY='...'
```

2. Push secrets to Fly:

```bash
make fly-secrets
```

3. Deploy:

```bash
make fly-deploy
```

4. Run schema migrations on Fly:

```bash
export FLY_APP='fantasy-league'
make fly-migrate-up
make fly-migrate-version
```

5. Bootstrap QStash-driven cron chain (one-time or when needed):

```bash
export INTERNAL_JOB_TOKEN='...'
APP_BASE_URL='https://fantasy-league.fly.dev' make jobs-bootstrap
```

Optional single-league bootstrap:

```bash
export INTERNAL_JOB_TOKEN='...'
APP_BASE_URL='https://fantasy-league.fly.dev' make jobs-bootstrap league_id=idn-liga-1-2025
```

The Fly image includes:

- `/app/fantasy-league` (API)
- `/app/fantasy-league-migrate` (migration runner)
- `/app/db/migrations` (SQL files)

Cost-focused defaults in `fly.toml`:

- `auto_stop_machines = "stop"`
- `auto_start_machines = true`
- `min_machines_running = 0`
- smallest shared VM (`shared`, 1 CPU, 256 MB)
- no Fly volume attached by default

Important for near-$0:

- Use an external free-tier Postgres provider (no Fly volume/managed DB from this app).
- Keep `min_machines_running = 0` so idle app can scale to zero.
- Do not allocate dedicated IPv4 unless needed.

## VS Code Run/Debug

Configured files:

- `.vscode/launch.json` (Debug API / Run API without debug)
- `.vscode/tasks.json` (Run API, Build API, Test All, Migrate Up)
- `.vscode/settings.json` (Go formatter + format-on-save)

## Environment Variables

- `APP_ENV` (`dev|stage|prod`)
- `APP_SERVICE_NAME` (default `fantasy-league-api`)
- `APP_SERVICE_VERSION` (default `dev`)
- `APP_HTTP_ADDR` (default `:8080`)
- `APP_READ_TIMEOUT` (default `10s`)
- `APP_WRITE_TIMEOUT` (default `15s`)
- `SWAGGER_ENABLED` (default `true`, but default `false` in prod)
- `CORS_ALLOWED_ORIGINS` (comma-separated, default `*`)
- `ANUBIS_BASE_URL` (default `http://localhost:8081`)
- `ANUBIS_INTROSPECT_PATH` (default `/v1/auth/introspect`)
- `ANUBIS_ADMIN_KEY` (default empty; required when introspect endpoint is protected by admin guard)
- `ANUBIS_TIMEOUT` (default `3s`)
- `ANUBIS_CIRCUIT_ENABLED` (default `true`)
- `ANUBIS_CIRCUIT_FAILURE_COUNT` (default `5`)
- `ANUBIS_CIRCUIT_OPEN_TIMEOUT` (default `15s`)
- `ANUBIS_CIRCUIT_HALF_OPEN_MAX_REQ` (default `2`)
- `UPTRACE_ENABLED` (default `false`)
- `UPTRACE_DSN` (required when `UPTRACE_ENABLED=true`)
- `PPROF_ENABLED` (default `false`)
- `PPROF_ADDR` (default `:6060`)
- `PYROSCOPE_ENABLED` (default `false`)
- `PYROSCOPE_SERVER_ADDRESS` (required when `PYROSCOPE_ENABLED=true`)
- `PYROSCOPE_APP_NAME` (default `APP_SERVICE_NAME`)
- `PYROSCOPE_UPLOAD_RATE` (default `15s`)
- `PYROSCOPE_AUTH_TOKEN` (optional)
- `PYROSCOPE_BASIC_AUTH_USER` (optional)
- `PYROSCOPE_BASIC_AUTH_PASSWORD` (optional)
- `APP_LOG_LEVEL` (default `info`)
- `CACHE_ENABLED` (default `true`)
- `CACHE_TTL` (default `60s`)
- `SPORTMONKS_ENABLED` (default `false`; when true, internal jobs fetch fixtures/standings from SportMonks)
- `SPORTMONKS_BASE_URL` (default `https://api.sportmonks.com/v3/football`)
- `SPORTMONKS_TOKEN` (required when `SPORTMONKS_ENABLED=true`)
- `SPORTMONKS_TIMEOUT` (default `20s`)
- `SPORTMONKS_MAX_RETRIES` (default `1`)
- `SPORTMONKS_SEASON_ID_MAP` (`league_public_id:season_id`, comma-separated, required when enabled)
- `SPORTMONKS_LEAGUE_ID_MAP` (`league_public_id:league_id`, optional fallback for live standings)
- `INTERNAL_JOB_TOKEN` (required for internal job endpoints; required when `QSTASH_ENABLED=true`)
- `JOB_SCHEDULE_INTERVAL` (default `15m`)
- `JOB_LIVE_INTERVAL` (default `5m`)
- `JOB_PRE_KICKOFF_LEAD` (default `15m`)
- `QSTASH_ENABLED` (default `false`)
- `QSTASH_BASE_URL` (default `https://qstash.upstash.io`)
- `QSTASH_TOKEN` (required when `QSTASH_ENABLED=true`)
- `QSTASH_TARGET_BASE_URL` (required when `QSTASH_ENABLED=true`, e.g. `https://fantasy-league.fly.dev`)
- `QSTASH_RETRIES` (default `3`)

## API Endpoints

- `GET /healthz`
- `GET /docs` (Swagger UI, when enabled)
- `GET /openapi.yaml` (OpenAPI spec, when enabled)
- `GET /v1/dashboard` (Bearer token required)
- `GET /v1/leagues`
- `GET /v1/leagues/{leagueID}/teams`
- `GET /v1/leagues/{leagueID}/fixtures`
- `GET /v1/leagues/{leagueID}/players`
- `GET /v1/leagues/{leagueID}/players/{playerID}`
- `GET /v1/leagues/{leagueID}/players/{playerID}/history`
- `GET /v1/leagues/{leagueID}/lineup`
- `PUT /v1/leagues/{leagueID}/lineup`
- `POST /v1/fantasy/squads` (Bearer token required)
- `POST /v1/fantasy/squads/picks` (Bearer token required)
- `GET /v1/fantasy/squads/me?league_id=<id>` (Bearer token required)
- `GET /v1/fantasy/squads/me/players?league_id=<id>` (Bearer token required)
- `POST /v1/fantasy/squads/me/players` (Bearer token required)

Note:
- Responses use a Google-style envelope with `apiVersion` and `data` / `error`.
