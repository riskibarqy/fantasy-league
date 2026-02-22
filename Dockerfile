# syntax=docker/dockerfile:1

FROM golang:1.26-bookworm AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/fantasy-league ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/fantasy-league-migrate ./cmd/migration

FROM gcr.io/distroless/base-debian12
WORKDIR /app

COPY --from=builder /out/fantasy-league /app/fantasy-league
COPY --from=builder /out/fantasy-league-migrate /app/fantasy-league-migrate
COPY --from=builder /app/db/migrations /app/db/migrations

ENV APP_HTTP_ADDR=:8080
EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/app/fantasy-league"]
