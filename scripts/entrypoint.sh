#!/bin/sh
# CYP-Registry 容器入口脚本
# Author: CYP | Contact: nasDSSCYP@outlook.com

set -e

# 颜色输出
log_info() {
    echo "[INFO] $(date '+%Y-%m-%d %H:%M:%S') $1"
}

log_warn() {
    echo "[WARN] $(date '+%Y-%m-%d %H:%M:%S') $1"
}

log_error() {
    echo "[ERROR] $(date '+%Y-%m-%d %H:%M:%S') $1"
}

# 打印启动信息
print_banner() {
    echo ""
    echo "╔════════════════════════════════════════════════╗"
    echo "║           CYP-Registry v1.0.0                  ║"
    echo "║     零信任容器镜像私有仓库管理系统             ║"
    echo "╚════════════════════════════════════════════════╝"
    echo ""
}

# 检查目录权限
check_directories() {
    log_info "检查目录权限..."
    
    for dir in /app/data/blobs /app/data/meta /app/data/cache /app/data/signatures /app/data/sboms; do
        if [ ! -d "$dir" ]; then
            log_info "创建目录: $dir"
            mkdir -p "$dir"
        fi
        
        if [ ! -w "$dir" ]; then
            log_error "目录不可写: $dir"
            exit 1
        fi
    done
    
    log_info "目录检查完成"
}

# 检查配置文件
check_config() {
    log_info "检查配置文件..."
    
    CONFIG_FILE="${CONFIG_FILE:-/app/configs/config.yaml}"
    
    if [ ! -f "$CONFIG_FILE" ]; then
        log_warn "配置文件不存在，使用默认配置"
        if [ -f "/app/configs/config.yaml.example" ]; then
            cp /app/configs/config.yaml.example "$CONFIG_FILE"
            log_info "已从示例创建配置文件"
        fi
    fi
    
    log_info "配置文件: $CONFIG_FILE"
}

# 初始化数据库
init_database() {
    log_info "初始化数据库..."
    
    DB_FILE="/app/data/meta/registry.db"
    
    if [ ! -f "$DB_FILE" ]; then
        log_info "创建新数据库..."
    else
        log_info "数据库已存在"
    fi
}

# 设置环境变量默认值
set_defaults() {
    export PORT="${PORT:-8080}"
    export HOST="${HOST:-0.0.0.0}"
    export LOG_LEVEL="${LOG_LEVEL:-info}"
    export TZ="${TZ:-Asia/Shanghai}"
}

# 健康检查
health_check() {
    curl -sf http://localhost:${PORT}/health > /dev/null 2>&1
    return $?
}

# 优雅关闭
graceful_shutdown() {
    log_info "收到关闭信号，正在优雅关闭..."
    
    # 发送 SIGTERM 给主进程
    if [ -n "$SERVER_PID" ]; then
        kill -TERM "$SERVER_PID" 2>/dev/null
        
        # 等待进程退出
        wait "$SERVER_PID"
    fi
    
    log_info "服务已关闭"
    exit 0
}

# 捕获信号
trap graceful_shutdown SIGTERM SIGINT SIGQUIT

# 主函数
main() {
    print_banner
    set_defaults
    check_directories
    check_config
    init_database
    
    log_info "启动 CYP-Registry 服务..."
    log_info "监听地址: ${HOST}:${PORT}"
    
    # 启动服务器
    exec /app/server "$@"
}

# 运行主函数
main "$@"
