# CYP-Docker-Registry - Multi-stage Dockerfile
# Version: v1.0.8
# Author: CYP | Contact: nasDSSCYP@outlook.com
# 
# 重要说明：
# - 使用纯 Go 实现的 SQLite 驱动 (modernc.org/sqlite)，无需 CGO
# - 支持 CGO_ENABLED=0 编译，适用于 Alpine 等精简镜像
# - 解决了 go-sqlite3 在 Docker 容器中的兼容性问题

# =============================================================================
# Stage 1: Build Go Backend
# =============================================================================
FROM golang:1.21-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY VERSION ./

# Build the binary
# 使用 CGO_ENABLED=0 编译，配合 modernc.org/sqlite 纯 Go 驱动
# 无需 C 编译器，生成静态链接的二进制文件
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X cyp-docker-registry/internal/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X cyp-docker-registry/internal/version.GitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')" \
    -o server ./cmd/server

# =============================================================================
# Stage 2: Build Vue Frontend
# =============================================================================
FROM node:20-alpine AS frontend-builder

WORKDIR /build

# Copy package files first for better caching
COPY web/package*.json ./

# Install dependencies
RUN npm ci --silent

# Copy source code
COPY web/ ./

# Build the frontend
RUN npm run build

# =============================================================================
# Stage 3: Final Runtime Image
# =============================================================================
FROM alpine:3.19

# Labels
LABEL maintainer="CYP <nasDSSCYP@outlook.com>"
LABEL description="CYP-Docker-Registry - Private Docker Image Registry"
LABEL version="1.0.8"

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && rm -rf /var/cache/apk/*

# Create non-root user for security
RUN addgroup -g 1000 registry && \
    adduser -u 1000 -G registry -s /bin/sh -D registry

# Create necessary directories
RUN mkdir -p /app/data/blobs /app/data/meta /app/data/cache /app/configs /app/web && \
    chown -R registry:registry /app

WORKDIR /app

# Copy binary from backend builder
COPY --from=backend-builder /build/server /app/server
COPY --from=backend-builder /build/VERSION /app/VERSION

# Copy frontend build from frontend builder
COPY --from=frontend-builder /build/dist /app/web/dist

# Copy default configuration
COPY configs/config.yaml.example /app/configs/config.yaml.example

# Copy scripts
COPY scripts/entrypoint.sh /app/entrypoint.sh
COPY scripts/unlock.sh /app/scripts/unlock.sh

# Set ownership and permissions
RUN chown -R registry:registry /app && \
    chmod +x /app/entrypoint.sh /app/scripts/unlock.sh

# Switch to non-root user
USER registry

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set environment variables
ENV TZ=Asia/Shanghai

# Volume for persistent data
VOLUME ["/app/data", "/app/configs"]

# Entry point
ENTRYPOINT ["/app/entrypoint.sh"]
CMD []
