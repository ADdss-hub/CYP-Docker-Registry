#!/bin/sh
# CYP-Docker-Registry 瀹瑰櫒鍏ュ彛鑴氭湰
# Version: v1.2.1
# Author: CYP | Contact: nasDSSCYP@outlook.com

set -e

# 棰滆壊杈撳嚭
log_info() {
    echo "[INFO] $(date '+%Y-%m-%d %H:%M:%S') $1"
}

log_warn() {
    echo "[WARN] $(date '+%Y-%m-%d %H:%M:%S') $1"
}

log_error() {
    echo "[ERROR] $(date '+%Y-%m-%d %H:%M:%S') $1"
}

# 鎵撳嵃鍚姩淇℃伅
print_banner() {
    echo ""
    echo "鈺斺晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晽"
    echo "鈺?       CYP-Docker-Registry v1.2.1              鈺?
    echo "鈺?    闆朵俊浠诲鍣ㄩ暅鍍忕鏈変粨搴撶鐞嗙郴缁?            鈺?
    echo "鈺氣晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨暆"
    echo ""
}

# 妫€鏌ョ洰褰曟潈闄?
check_directories() {
    log_info "妫€鏌ョ洰褰曟潈闄?.."
    
    for dir in /app/data/blobs /app/data/meta /app/data/cache /app/data/signatures /app/data/sboms; do
        if [ ! -d "$dir" ]; then
            log_info "鍒涘缓鐩綍: $dir"
            mkdir -p "$dir"
        fi
        
        if [ ! -w "$dir" ]; then
            log_error "鐩綍涓嶅彲鍐? $dir"
            exit 1
        fi
    done
    
    log_info "鐩綍妫€鏌ュ畬鎴?
}

# 妫€鏌ラ厤缃枃浠?
check_config() {
    log_info "妫€鏌ラ厤缃枃浠?.."
    
    CONFIG_FILE="${CONFIG_FILE:-/app/configs/config.yaml}"
    
    if [ ! -f "$CONFIG_FILE" ]; then
        log_warn "閰嶇疆鏂囦欢涓嶅瓨鍦紝浣跨敤榛樿閰嶇疆"
        if [ -f "/app/configs/config.yaml.example" ]; then
            cp /app/configs/config.yaml.example "$CONFIG_FILE"
            log_info "宸蹭粠绀轰緥鍒涘缓閰嶇疆鏂囦欢"
        fi
    fi
    
    log_info "閰嶇疆鏂囦欢: $CONFIG_FILE"
}

# 鍒濆鍖栨暟鎹簱
init_database() {
    log_info "鍒濆鍖栨暟鎹簱..."
    
    DB_FILE="/app/data/meta/registry.db"
    
    if [ ! -f "$DB_FILE" ]; then
        log_info "鍒涘缓鏂版暟鎹簱..."
    else
        log_info "鏁版嵁搴撳凡瀛樺湪"
    fi
}

# 璁剧疆鐜鍙橀噺榛樿鍊?
set_defaults() {
    export PORT="${PORT:-8080}"
    export HOST="${HOST:-0.0.0.0}"
    export LOG_LEVEL="${LOG_LEVEL:-info}"
    export TZ="${TZ:-Asia/Shanghai}"
}

# 鍋ュ悍妫€鏌?
health_check() {
    curl -sf http://localhost:${PORT}/health > /dev/null 2>&1
    return $?
}

# 浼橀泤鍏抽棴
graceful_shutdown() {
    log_info "鏀跺埌鍏抽棴淇″彿锛屾鍦ㄤ紭闆呭叧闂?.."
    
    # 鍙戦€?SIGTERM 缁欎富杩涚▼
    if [ -n "$SERVER_PID" ]; then
        kill -TERM "$SERVER_PID" 2>/dev/null
        
        # 绛夊緟杩涚▼閫€鍑?
        wait "$SERVER_PID"
    fi
    
    log_info "鏈嶅姟宸插叧闂?
    exit 0
}

# 鎹曡幏淇″彿
trap graceful_shutdown SIGTERM SIGINT SIGQUIT

# 涓诲嚱鏁?
main() {
    print_banner
    set_defaults
    check_directories
    check_config
    init_database
    
    log_info "鍚姩 CYP-Docker-Registry 鏈嶅姟..."
    log_info "鐩戝惉鍦板潃: ${HOST}:${PORT}"
    
    # 鍚姩鏈嶅姟鍣?
    exec /app/server "$@"
}

# 杩愯涓诲嚱鏁?
main "$@"
