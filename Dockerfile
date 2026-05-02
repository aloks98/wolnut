# Frontend build stage - use slim (Debian) for better native module compatibility
FROM node:22-slim AS frontend

WORKDIR /app/web

# Install pnpm
RUN corepack enable && corepack prepare pnpm@latest --activate

# Install dependencies
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# Build frontend
COPY web/ .
RUN pnpm build

# Backend build stage
FROM golang:1.22-alpine3.19 AS builder

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
FROM alpine:3.19

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
