# syntax=docker/dockerfile:1

FROM golang:1.26-bookworm AS builder
WORKDIR /app

ARG TARGETOS=linux
ARG TARGETARCH=amd64

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/fantasy-league ./cmd/api
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/fantasy-league-migrate ./cmd/migration

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=builder /out/fantasy-league /app/fantasy-league
COPY --from=builder /out/fantasy-league-migrate /app/fantasy-league-migrate
COPY --from=builder /app/db/migrations /app/db/migrations

ENV APP_HTTP_ADDR=:8080
EXPOSE 8080

ENTRYPOINT ["/app/fantasy-league"]
