# CYP-Docker-Registry - Multi-stage Dockerfile
# Version: v1.2.4
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

# 设置 Go 代理加速（国内镜像）
ENV GOPROXY=https://goproxy.cn,https://goproxy.io,direct
ENV GO111MODULE=on

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
LABEL version="1.2.4"

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    coreutils \
    procps \
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

# Copy unlock script
COPY scripts/unlock.sh /app/scripts/unlock.sh

# Create entrypoint script directly in container to avoid line ending issues
RUN cat > /app/entrypoint.sh << 'ENTRYPOINT_EOF'
#!/bin/sh
# CYP-Docker-Registry Container Entrypoint Script
# Version: v1.2.4
# Author: CYP | Contact: nasDSSCYP@outlook.com

set -e

log_info() {
    echo "[INFO] $(date '+%Y-%m-%d %H:%M:%S') $1"
}

log_warn() {
    echo "[WARN] $(date '+%Y-%m-%d %H:%M:%S') $1"
}

log_error() {
    echo "[ERROR] $(date '+%Y-%m-%d %H:%M:%S') $1"
}

print_banner() {
    echo ""
    echo "================================================"
    echo "        CYP-Docker-Registry v1.2.3"
    echo "     Private Docker Image Registry System"
    echo "================================================"
    echo ""
}

check_directories() {
    log_info "Checking directories..."
    
    for dir in /app/data/blobs /app/data/meta /app/data/cache /app/data/signatures /app/data/sboms; do
        if [ ! -d "$dir" ]; then
            log_info "Creating directory: $dir"
            mkdir -p "$dir"
        fi
        
        if [ ! -w "$dir" ]; then
            log_error "Directory not writable: $dir"
            exit 1
        fi
    done
    
    log_info "Directory check completed"
}

check_config() {
    log_info "Checking configuration..."
    
    CONFIG_FILE="${CONFIG_FILE:-/app/configs/config.yaml}"
    
    if [ ! -f "$CONFIG_FILE" ]; then
        log_warn "Config file not found, using defaults"
        if [ -f "/app/configs/config.yaml.example" ]; then
            cp /app/configs/config.yaml.example "$CONFIG_FILE"
            log_info "Created config from example"
        fi
    fi
    
    log_info "Config file: $CONFIG_FILE"
}

init_database() {
    log_info "Initializing database..."
    
    DB_FILE="/app/data/meta/registry.db"
    
    if [ ! -f "$DB_FILE" ]; then
        log_info "Creating new database..."
    else
        log_info "Database exists"
    fi
}

set_defaults() {
    export PORT="${PORT:-8080}"
    export HOST="${HOST:-0.0.0.0}"
    export LOG_LEVEL="${LOG_LEVEL:-info}"
    export TZ="${TZ:-Asia/Shanghai}"
}

health_check() {
    curl -sf http://localhost:${PORT}/health > /dev/null 2>&1
    return $?
}

graceful_shutdown() {
    log_info "Received shutdown signal, gracefully shutting down..."
    
    if [ -n "$SERVER_PID" ]; then
        kill -TERM "$SERVER_PID" 2>/dev/null
        wait "$SERVER_PID"
    fi
    
    log_info "Service stopped"
    exit 0
}

trap graceful_shutdown SIGTERM SIGINT SIGQUIT

main() {
    print_banner
    set_defaults
    check_directories
    check_config
    init_database
    
    log_info "Starting CYP-Docker-Registry service..."
    log_info "Listen address: ${HOST}:${PORT}"
    
    exec /app/server "$@"
}

main "$@"
ENTRYPOINT_EOF

# Convert line endings for unlock.sh and set permissions
RUN sed -i 's/\r$//' /app/scripts/unlock.sh && \
    chmod +x /app/entrypoint.sh /app/scripts/unlock.sh && \
    chown -R registry:registry /app

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
