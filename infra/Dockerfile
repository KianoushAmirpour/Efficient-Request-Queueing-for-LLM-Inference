
# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.25.7-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY infra/config/ ./infra/config/
COPY infra/pg_db/ ./infra/pg_db/
COPY infra/redis/ ./infra/redis/
COPY configs/ ./configs/

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" \
    -o /out/inference-platform ./cmd


FROM alpine:3.22 AS runtime

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S -g 10001 app && adduser -S -D -H -u 10001 -G app app

ENV CFG_PATH=/app/configs \
    GIN_MODE=release

COPY --from=builder --chown=app:app /out/inference-platform ./inference-platform
COPY --from=builder --chown=app:app /app/configs ./configs

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:8080/api/health/live || exit 1

USER app

ENTRYPOINT ["./inference-platform"]