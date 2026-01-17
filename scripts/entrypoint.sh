#!/bin/sh
# CYP-Docker-Registry Container Entrypoint Script
# Version: v1.2.1
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
    echo "        CYP-Docker-Registry v1.2.1"
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
