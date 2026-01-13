#!/bin/bash
# CYP-Registry 智能安装脚本
# 自动检测环境并配置最优参数

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 版本信息
VERSION="1.0.0"
AUTHOR="CYP"
EMAIL="nasDSSCYP@outlook.com"

# 默认配置
INSTALL_DIR="/opt/cyp-registry"
DATA_DIR="/opt/cyp-registry/data"
CONFIG_FILE="/opt/cyp-registry/config.yaml"
PORT=8080

echo -e "${BLUE}"
echo "╔════════════════════════════════════════════════╗"
echo "║        CYP-Registry 智能安装脚本               ║"
echo "║        版本: v${VERSION}                           ║"
echo "╚════════════════════════════════════════════════╝"
echo -e "${NC}"

# 检测操作系统
detect_os() {
    echo -e "${YELLOW}正在检测操作系统...${NC}"
    
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$NAME
        VER=$VERSION_ID
    elif [ -f /etc/debian_version ]; then
        OS="Debian"
        VER=$(cat /etc/debian_version)
    elif [ -f /etc/redhat-release ]; then
        OS="Red Hat"
        VER=$(cat /etc/redhat-release)
    else
        OS=$(uname -s)
        VER=$(uname -r)
    fi
    
    echo -e "${GREEN}✓ 操作系统: $OS $VER${NC}"
}

# 检测运行环境
detect_environment() {
    echo -e "${YELLOW}正在检测运行环境...${NC}"
    
    ENV_TYPE="physical"
    
    # 检测 Docker
    if [ -f /.dockerenv ]; then
        ENV_TYPE="docker"
        echo -e "${GREEN}✓ 检测到 Docker 容器环境${NC}"
        return
    fi
    
    # 检测 Kubernetes
    if [ -n "$KUBERNETES_SERVICE_HOST" ]; then
        ENV_TYPE="kubernetes"
        echo -e "${GREEN}✓ 检测到 Kubernetes 环境${NC}"
        return
    fi
    
    # 检测云环境
    if curl -s --connect-timeout 2 http://169.254.169.254/latest/meta-data/ > /dev/null 2>&1; then
        ENV_TYPE="cloud-aws"
        echo -e "${GREEN}✓ 检测到 AWS 云环境${NC}"
        return
    fi
    
    if curl -s --connect-timeout 2 http://100.100.100.200/latest/meta-data/ > /dev/null 2>&1; then
        ENV_TYPE="cloud-aliyun"
        echo -e "${GREEN}✓ 检测到阿里云环境${NC}"
        return
    fi
    
    # 检测 NAS
    if [ -f /etc/synoinfo.conf ]; then
        ENV_TYPE="nas-synology"
        echo -e "${GREEN}✓ 检测到群晖 NAS 环境${NC}"
        return
    fi
    
    if [ -f /etc/config/qpkg.conf ]; then
        ENV_TYPE="nas-qnap"
        echo -e "${GREEN}✓ 检测到 QNAP NAS 环境${NC}"
        return
    fi
    
    # 检测树莓派
    if grep -q "Raspberry Pi" /proc/cpuinfo 2>/dev/null; then
        ENV_TYPE="raspberry"
        echo -e "${GREEN}✓ 检测到树莓派环境${NC}"
        return
    fi
    
    echo -e "${GREEN}✓ 检测到物理机/虚拟机环境${NC}"
}

# 检测硬件资源
detect_hardware() {
    echo -e "${YELLOW}正在检测硬件资源...${NC}"
    
    # CPU
    CPU_CORES=$(nproc 2>/dev/null || echo "1")
    echo -e "${GREEN}✓ CPU 核心数: $CPU_CORES${NC}"
    
    # 内存
    if [ -f /proc/meminfo ]; then
        MEM_TOTAL=$(grep MemTotal /proc/meminfo | awk '{print int($2/1024)}')
        echo -e "${GREEN}✓ 总内存: ${MEM_TOTAL}MB${NC}"
    fi
    
    # 磁盘
    DISK_FREE=$(df -h / | awk 'NR==2 {print $4}')
    echo -e "${GREEN}✓ 可用磁盘: $DISK_FREE${NC}"
    
    # 架构
    ARCH=$(uname -m)
    echo -e "${GREEN}✓ 系统架构: $ARCH${NC}"
}

# 检测 Docker
check_docker() {
    echo -e "${YELLOW}正在检测 Docker...${NC}"
    
    if command -v docker &> /dev/null; then
        DOCKER_VERSION=$(docker --version | awk '{print $3}' | tr -d ',')
        echo -e "${GREEN}✓ Docker 已安装: $DOCKER_VERSION${NC}"
        return 0
    else
        echo -e "${RED}✗ Docker 未安装${NC}"
        return 1
    fi
}

# 安装 Docker
install_docker() {
    echo -e "${YELLOW}正在安装 Docker...${NC}"
    
    if [ "$OS" = "Ubuntu" ] || [ "$OS" = "Debian GNU/Linux" ]; then
        apt-get update
        apt-get install -y docker.io docker-compose
    elif [ "$OS" = "CentOS Linux" ] || [ "$OS" = "Red Hat" ]; then
        yum install -y docker docker-compose
    else
        echo -e "${RED}不支持的操作系统，请手动安装 Docker${NC}"
        exit 1
    fi
    
    systemctl enable docker
    systemctl start docker
    
    echo -e "${GREEN}✓ Docker 安装完成${NC}"
}

# 创建目录结构
create_directories() {
    echo -e "${YELLOW}正在创建目录结构...${NC}"
    
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$DATA_DIR/blobs"
    mkdir -p "$DATA_DIR/meta"
    mkdir -p "$DATA_DIR/cache"
    mkdir -p "$INSTALL_DIR/scripts"
    
    echo -e "${GREEN}✓ 目录创建完成${NC}"
}

# 生成配置文件
generate_config() {
    echo -e "${YELLOW}正在生成配置文件...${NC}"
    
    # 根据环境选择模板
    TEMPLATE="docker"
    case $ENV_TYPE in
        "nas-synology") TEMPLATE="nas-synology" ;;
        "nas-qnap") TEMPLATE="nas-qnap" ;;
        "cloud-aws") TEMPLATE="cloud-aws" ;;
        "cloud-aliyun") TEMPLATE="cloud-aliyun" ;;
        "raspberry") TEMPLATE="raspberry" ;;
        "kubernetes") TEMPLATE="kubernetes" ;;
    esac
    
    # 生成基础配置
    cat > "$CONFIG_FILE" <<EOF
# CYP-Registry 配置文件
# 环境: $ENV_TYPE
# 生成时间: $(date)

app:
  name: "CYP-Registry"
  version: "v${VERSION}"
  port: ${PORT}
  host: "0.0.0.0"
  log_level: "info"

storage:
  blob_path: "${DATA_DIR}/blobs"
  meta_path: "${DATA_DIR}/meta"
  cache_path: "${DATA_DIR}/cache"
  max_cache_size: "10GB"

accelerator:
  enabled: true
  upstreams:
    - name: "Docker Hub"
      url: "https://registry-1.docker.io"
      priority: 1
    - name: "阿里云"
      url: "https://registry.cn-hangzhou.aliyuncs.com"
      priority: 2

security:
  force_login:
    enabled: true
    mode: "strict"
  
  failed_attempts:
    max_login_attempts: 3
    max_token_attempts: 5
    lock_duration: "1h"
  
  auto_lock:
    enabled: true
    lock_on_bypass_attempt: true
  
  intrusion_detection:
    enabled: true
    rules:
      - name: "direct_url_access"
        action: "lock"
        threshold: 1
      - name: "forged_jwt"
        action: "lock"
        threshold: 1
      - name: "login_failure"
        action: "lock"
        threshold: 3
  
  audit:
    log_all_requests: true
    blockchain_hash: true
    retention: "1y"
EOF
    
    echo -e "${GREEN}✓ 配置文件生成完成: $CONFIG_FILE${NC}"
}

# 生成 Docker Compose 文件
generate_docker_compose() {
    echo -e "${YELLOW}正在生成 Docker Compose 文件...${NC}"
    
    cat > "$INSTALL_DIR/docker-compose.yaml" <<EOF
version: '3.8'

services:
  cyp-registry:
    image: cyp/registry:latest
    container_name: cyp-registry
    restart: unless-stopped
    ports:
      - "${PORT}:8080"
    volumes:
      - ${DATA_DIR}:/app/data
      - ${CONFIG_FILE}:/app/config.yaml:ro
    environment:
      - TZ=Asia/Shanghai
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
EOF
    
    echo -e "${GREEN}✓ Docker Compose 文件生成完成${NC}"
}

# 生成解锁脚本
generate_unlock_script() {
    cat > "$INSTALL_DIR/scripts/unlock.sh" <<'SCRIPT'
#!/bin/bash
# CYP-Registry 解锁脚本

echo "CYP-Registry 解锁工具"
echo "====================="

read -p "请输入管理员密码: " -s PASSWORD
echo ""

curl -X POST http://localhost:8080/api/v1/system/lock/unlock \
  -H "Content-Type: application/json" \
  -d "{\"password\": \"$PASSWORD\"}"

echo ""
SCRIPT
    
    chmod +x "$INSTALL_DIR/scripts/unlock.sh"
}

# 打印安装完成信息
print_completion() {
    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║          安装完成！安全机制已激活              ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "安装信息:"
    echo "  安装目录: $INSTALL_DIR"
    echo "  数据目录: $DATA_DIR"
    echo "  配置文件: $CONFIG_FILE"
    echo "  访问地址: http://localhost:${PORT}"
    echo ""
    echo "默认账户:"
    echo "  用户名: admin"
    echo "  密码: admin123 (请立即修改)"
    echo ""
    echo "安全提示:"
    echo "  - 所有页面必须登录后访问"
    echo "  - 登录失败3次将锁定系统"
    echo "  - 审计日志已启用（区块链哈希防篡改）"
    echo "  - 解锁命令: $INSTALL_DIR/scripts/unlock.sh"
    echo ""
    echo "启动命令:"
    echo "  cd $INSTALL_DIR && docker-compose up -d"
    echo ""
    echo -e "${BLUE}感谢使用 CYP-Registry！${NC}"
    echo -e "${BLUE}作者: $AUTHOR | 邮箱: $EMAIL${NC}"
}

# 主流程
main() {
    # 检查 root 权限
    if [ "$EUID" -ne 0 ]; then
        echo -e "${RED}请使用 root 权限运行此脚本${NC}"
        exit 1
    fi
    
    detect_os
    detect_environment
    detect_hardware
    
    if ! check_docker; then
        read -p "是否安装 Docker? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            install_docker
        else
            echo -e "${RED}Docker 是必需的，安装已取消${NC}"
            exit 1
        fi
    fi
    
    create_directories
    generate_config
    generate_docker_compose
    generate_unlock_script
    
    print_completion
}

# 运行主流程
main "$@"
