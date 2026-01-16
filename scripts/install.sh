#!/bin/bash
# CYP-Docker-Registry 鏅鸿兘瀹夎鑴氭湰
# Version: v1.2.1
# 鑷姩妫€娴嬬幆澧冨苟閰嶇疆鏈€浼樺弬鏁?

set -e

# 棰滆壊瀹氫箟
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 鐗堟湰淇℃伅
VERSION="1.2.1"
AUTHOR="CYP"
EMAIL="nasDSSCYP@outlook.com"

# 榛樿閰嶇疆
INSTALL_DIR="/opt/cyp-docker-registry"
DATA_DIR="/opt/cyp-docker-registry/data"
CONFIG_FILE="/opt/cyp-docker-registry/config.yaml"
PORT=8080

echo -e "${BLUE}"
echo "鈺斺晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晽"
echo "鈺?  CYP-Docker-Registry 鏅鸿兘瀹夎鑴氭湰 v1.2.1    鈺?
echo "鈺?       鐗堟湰: v${VERSION}                           鈺?
echo "鈺氣晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨暆"
echo -e "${NC}"

# 妫€娴嬫搷浣滅郴缁?
detect_os() {
    echo -e "${YELLOW}姝ｅ湪妫€娴嬫搷浣滅郴缁?..${NC}"
    
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
    
    echo -e "${GREEN}鉁?鎿嶄綔绯荤粺: $OS $VER${NC}"
}

# 妫€娴嬭繍琛岀幆澧?
detect_environment() {
    echo -e "${YELLOW}姝ｅ湪妫€娴嬭繍琛岀幆澧?..${NC}"
    
    ENV_TYPE="physical"
    
    # 妫€娴?Docker
    if [ -f /.dockerenv ]; then
        ENV_TYPE="docker"
        echo -e "${GREEN}鉁?妫€娴嬪埌 Docker 瀹瑰櫒鐜${NC}"
        return
    fi
    
    # 妫€娴?Kubernetes
    if [ -n "$KUBERNETES_SERVICE_HOST" ]; then
        ENV_TYPE="kubernetes"
        echo -e "${GREEN}鉁?妫€娴嬪埌 Kubernetes 鐜${NC}"
        return
    fi
    
    # 妫€娴嬩簯鐜
    if curl -s --connect-timeout 2 http://169.254.169.254/latest/meta-data/ > /dev/null 2>&1; then
        ENV_TYPE="cloud-aws"
        echo -e "${GREEN}鉁?妫€娴嬪埌 AWS 浜戠幆澧?{NC}"
        return
    fi
    
    if curl -s --connect-timeout 2 http://100.100.100.200/latest/meta-data/ > /dev/null 2>&1; then
        ENV_TYPE="cloud-aliyun"
        echo -e "${GREEN}鉁?妫€娴嬪埌闃块噷浜戠幆澧?{NC}"
        return
    fi
    
    # 妫€娴?NAS
    if [ -f /etc/synoinfo.conf ]; then
        ENV_TYPE="nas-synology"
        echo -e "${GREEN}鉁?妫€娴嬪埌缇ゆ櫀 NAS 鐜${NC}"
        return
    fi
    
    if [ -f /etc/config/qpkg.conf ]; then
        ENV_TYPE="nas-qnap"
        echo -e "${GREEN}鉁?妫€娴嬪埌 QNAP NAS 鐜${NC}"
        return
    fi
    
    # 妫€娴嬫爲鑾撴淳
    if grep -q "Raspberry Pi" /proc/cpuinfo 2>/dev/null; then
        ENV_TYPE="raspberry"
        echo -e "${GREEN}鉁?妫€娴嬪埌鏍戣帗娲剧幆澧?{NC}"
        return
    fi
    
    echo -e "${GREEN}鉁?妫€娴嬪埌鐗╃悊鏈?铏氭嫙鏈虹幆澧?{NC}"
}

# 妫€娴嬬‖浠惰祫婧?
detect_hardware() {
    echo -e "${YELLOW}姝ｅ湪妫€娴嬬‖浠惰祫婧?..${NC}"
    
    # CPU
    CPU_CORES=$(nproc 2>/dev/null || echo "1")
    echo -e "${GREEN}鉁?CPU 鏍稿績鏁? $CPU_CORES${NC}"
    
    # 鍐呭瓨
    if [ -f /proc/meminfo ]; then
        MEM_TOTAL=$(grep MemTotal /proc/meminfo | awk '{print int($2/1024)}')
        echo -e "${GREEN}鉁?鎬诲唴瀛? ${MEM_TOTAL}MB${NC}"
    fi
    
    # 纾佺洏
    DISK_FREE=$(df -h / | awk 'NR==2 {print $4}')
    echo -e "${GREEN}鉁?鍙敤纾佺洏: $DISK_FREE${NC}"
    
    # 鏋舵瀯
    ARCH=$(uname -m)
    echo -e "${GREEN}鉁?绯荤粺鏋舵瀯: $ARCH${NC}"
}

# 妫€娴?Docker
check_docker() {
    echo -e "${YELLOW}姝ｅ湪妫€娴?Docker...${NC}"
    
    if command -v docker &> /dev/null; then
        DOCKER_VERSION=$(docker --version | awk '{print $3}' | tr -d ',')
        echo -e "${GREEN}鉁?Docker 宸插畨瑁? $DOCKER_VERSION${NC}"
        return 0
    else
        echo -e "${RED}鉁?Docker 鏈畨瑁?{NC}"
        return 1
    fi
}

# 瀹夎 Docker
install_docker() {
    echo -e "${YELLOW}姝ｅ湪瀹夎 Docker...${NC}"
    
    if [ "$OS" = "Ubuntu" ] || [ "$OS" = "Debian GNU/Linux" ]; then
        apt-get update
        apt-get install -y docker.io docker-compose
    elif [ "$OS" = "CentOS Linux" ] || [ "$OS" = "Red Hat" ]; then
        yum install -y docker docker-compose
    else
        echo -e "${RED}涓嶆敮鎸佺殑鎿嶄綔绯荤粺锛岃鎵嬪姩瀹夎 Docker${NC}"
        exit 1
    fi
    
    systemctl enable docker
    systemctl start docker
    
    echo -e "${GREEN}鉁?Docker 瀹夎瀹屾垚${NC}"
}

# 鍒涘缓鐩綍缁撴瀯
create_directories() {
    echo -e "${YELLOW}姝ｅ湪鍒涘缓鐩綍缁撴瀯...${NC}"
    
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$DATA_DIR/blobs"
    mkdir -p "$DATA_DIR/meta"
    mkdir -p "$DATA_DIR/cache"
    mkdir -p "$INSTALL_DIR/scripts"
    
    echo -e "${GREEN}鉁?鐩綍鍒涘缓瀹屾垚${NC}"
}

# 鐢熸垚閰嶇疆鏂囦欢
generate_config() {
    echo -e "${YELLOW}姝ｅ湪鐢熸垚閰嶇疆鏂囦欢...${NC}"
    
    # 鏍规嵁鐜閫夋嫨妯℃澘
    TEMPLATE="docker"
    case $ENV_TYPE in
        "nas-synology") TEMPLATE="nas-synology" ;;
        "nas-qnap") TEMPLATE="nas-qnap" ;;
        "cloud-aws") TEMPLATE="cloud-aws" ;;
        "cloud-aliyun") TEMPLATE="cloud-aliyun" ;;
        "raspberry") TEMPLATE="raspberry" ;;
        "kubernetes") TEMPLATE="kubernetes" ;;
    esac
    
    # 鐢熸垚鍩虹閰嶇疆
    cat > "$CONFIG_FILE" <<EOF
# CYP-Docker-Registry 閰嶇疆鏂囦欢
# 鐜: $ENV_TYPE
# 鐢熸垚鏃堕棿: $(date)

app:
  name: "CYP-Docker-Registry"
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
    - name: "闃块噷浜?
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
    
    echo -e "${GREEN}鉁?閰嶇疆鏂囦欢鐢熸垚瀹屾垚: $CONFIG_FILE${NC}"
}

# 鐢熸垚 Docker Compose 鏂囦欢
generate_docker_compose() {
    echo -e "${YELLOW}姝ｅ湪鐢熸垚 Docker Compose 鏂囦欢...${NC}"
    
    cat > "$INSTALL_DIR/docker-compose.yaml" <<EOF
version: '3.8'

services:
  cyp-docker-registry:
    image: cyp/docker-registry:latest
    container_name: cyp-docker-registry
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
    
    echo -e "${GREEN}鉁?Docker Compose 鏂囦欢鐢熸垚瀹屾垚${NC}"
}

# 鐢熸垚瑙ｉ攣鑴氭湰
generate_unlock_script() {
    cat > "$INSTALL_DIR/scripts/unlock.sh" <<'SCRIPT'
#!/bin/bash
# CYP-Docker-Registry 瑙ｉ攣鑴氭湰

echo "CYP-Docker-Registry 瑙ｉ攣宸ュ叿"
echo "====================="

read -p "璇疯緭鍏ョ鐞嗗憳瀵嗙爜: " -s PASSWORD
echo ""

curl -X POST http://localhost:8080/api/v1/system/lock/unlock \
  -H "Content-Type: application/json" \
  -d "{\"password\": \"$PASSWORD\"}"

echo ""
SCRIPT
    
    chmod +x "$INSTALL_DIR/scripts/unlock.sh"
}

# 鎵撳嵃瀹夎瀹屾垚淇℃伅
print_completion() {
    echo ""
    echo -e "${GREEN}鈺斺晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晽${NC}"
    echo -e "${GREEN}鈺?         瀹夎瀹屾垚锛佸畨鍏ㄦ満鍒跺凡婵€娲?             鈺?{NC}"
    echo -e "${GREEN}鈺氣晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨暆${NC}"
    echo ""
    echo "瀹夎淇℃伅:"
    echo "  瀹夎鐩綍: $INSTALL_DIR"
    echo "  鏁版嵁鐩綍: $DATA_DIR"
    echo "  閰嶇疆鏂囦欢: $CONFIG_FILE"
    echo "  璁块棶鍦板潃: http://localhost:${PORT}"
    echo ""
    echo "榛樿璐︽埛:"
    echo "  鐢ㄦ埛鍚? admin"
    echo "  瀵嗙爜: admin123 (璇风珛鍗充慨鏀?"
    echo ""
    echo "瀹夊叏鎻愮ず:"
    echo "  - 鎵€鏈夐〉闈㈠繀椤荤櫥褰曞悗璁块棶"
    echo "  - 鐧诲綍澶辫触3娆″皢閿佸畾绯荤粺"
    echo "  - 瀹¤鏃ュ織宸插惎鐢紙鍖哄潡閾惧搱甯岄槻绡℃敼锛?
    echo "  - 瑙ｉ攣鍛戒护: $INSTALL_DIR/scripts/unlock.sh"
    echo ""
    echo "鍚姩鍛戒护:"
    echo "  cd $INSTALL_DIR && docker-compose up -d"
    echo ""
    echo -e "${BLUE}鎰熻阿浣跨敤 CYP-Docker-Registry锛?{NC}"
    echo -e "${BLUE}浣滆€? $AUTHOR | 閭: $EMAIL${NC}"
}

# 涓绘祦绋?
main() {
    # 妫€鏌?root 鏉冮檺
    if [ "$EUID" -ne 0 ]; then
        echo -e "${RED}璇蜂娇鐢?root 鏉冮檺杩愯姝よ剼鏈?{NC}"
        exit 1
    fi
    
    detect_os
    detect_environment
    detect_hardware
    
    if ! check_docker; then
        read -p "鏄惁瀹夎 Docker? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            install_docker
        else
            echo -e "${RED}Docker 鏄繀闇€鐨勶紝瀹夎宸插彇娑?{NC}"
            exit 1
        fi
    fi
    
    create_directories
    generate_config
    generate_docker_compose
    generate_unlock_script
    
    print_completion
}

# 杩愯涓绘祦绋?
main "$@"
