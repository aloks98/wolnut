# Frontend build stage. Track the active Node 24 LTS (Jod) line so security
# patches flow through; the major is pinned so we control any cross-version
# bumps. Node 25 unbundled corepack, so staying on 24 LTS lets us keep the
# clean `corepack enable` install path below — actual pnpm version comes from
# the `packageManager` field in package.json (with integrity hash).
FROM node:24-slim AS frontend

WORKDIR /app/web

# `corepack enable` is enough — when `pnpm install` runs below, corepack reads
# the `packageManager` field from package.json and pulls that exact pnpm
# version (with integrity hash). Avoids `pnpm@latest` floating to a version
# that doesn't match local dev.
RUN corepack enable

# Install dependencies. pnpm-workspace.yaml carries the approved-builds list
# (allowBuilds: esbuild) — without it here, pnpm 10+ aborts the install with
# ERR_PNPM_IGNORED_BUILDS for esbuild's native-binary build script. .npmrc
# carries engine-strict. Both must be in the build context, not just locally.
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml web/.npmrc ./
RUN pnpm install --frozen-lockfile

# Build frontend
COPY web/ .
RUN pnpm build

# Backend build stage. Go version tracks go.mod's directive; keep CI
# (build.yml / release.yml) on the same minor so dev, CI, and release binaries
# are built with one toolchain.
FROM golang:1.25-alpine3.21 AS builder

WORKDIR /app

# Dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy frontend build
COPY --from=frontend /app/web/build ./web/build

# Source
COPY . .

# Build
ARG VERSION=dev
ARG COMMIT=none
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" -o wol-nut

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/wol-nut .

# Create data directory
RUN mkdir -p /data

# Default environment
ENV WOLNUT_SERVER_HOST=0.0.0.0
ENV WOLNUT_SERVER_PORT=8080
ENV WOLNUT_DATA_PATH=/data

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

ENTRYPOINT ["/app/wol-nut"]
