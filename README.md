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
- Indonesian league seeded by default (`Liga 1 Indonesia`)
- Team and player listing per league
- Authenticated squad creation/upsert
- Squad rule validation:
  - exact 11 players
  - budget cap
  - max players from same real club
  - minimum formation constraints
- Login/auth verification via Anubis account service (`../../rust/anubis`) through token introspection

## Assumptions for Anubis Integration

`ANUBIS_BASE_URL` and `ANUBIS_INTROSPECT_PATH` are configurable.

Default introspection request/response expected:

- Request: `POST {"token":"<bearer-token>"}`
- Success response (HTTP 200):

```json
{
  "active": true,
  "user_id": "usr_123",
  "email": "user@example.com"
}
```

If your Anubis response schema differs, adjust:

- `internal/infrastructure/account/anubis/client.go`

## Run

```bash
go run ./cmd/api
```

Default server address: `:8080`

## Environment Variables

- `APP_HTTP_ADDR` (default `:8080`)
- `APP_READ_TIMEOUT` (default `10s`)
- `APP_WRITE_TIMEOUT` (default `15s`)
- `ANUBIS_BASE_URL` (default `http://localhost:8081`)
- `ANUBIS_INTROSPECT_PATH` (default `/v1/auth/introspect`)
- `ANUBIS_TIMEOUT` (default `3s`)
- `APP_LOG_LEVEL` (default `info`)

## API Endpoints

- `GET /healthz`
- `GET /v1/leagues`
- `GET /v1/leagues/{leagueID}/teams`
- `GET /v1/leagues/{leagueID}/players`
- `POST /v1/fantasy/squads` (Bearer token required)
- `GET /v1/fantasy/squads/me?league_id=<id>` (Bearer token required)

