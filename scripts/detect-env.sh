#!/bin/bash
# CYP-Docker-Registry 鐜妫€娴嬭剼鏈?# Version: v1.2.1
# Author: CYP | Contact: nasDSSCYP@outlook.com

set -e

# 棰滆壊瀹氫箟
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}"
echo "鈺斺晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晽"
echo "鈺?    CYP-Docker-Registry 鐜妫€娴嬪伐鍏?v1.2.1  鈺?
echo "鈺氣晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨暆"
echo -e "${NC}"

# 妫€娴嬬粨鏋?ENV_TYPE="physical"
ENV_DETAILS=""

# 妫€娴?Docker 瀹瑰櫒
detect_docker() {
    if [ -f /.dockerenv ]; then
        ENV_TYPE="docker"
        ENV_DETAILS="Docker Container"
        return 0
    fi
    
    if grep -q docker /proc/1/cgroup 2>/dev/null; then
        ENV_TYPE="docker"
        ENV_DETAILS="Docker Container"
        return 0
    fi
    
    return 1
}

# 妫€娴?Kubernetes
detect_kubernetes() {
    if [ -n "$KUBERNETES_SERVICE_HOST" ]; then
        ENV_TYPE="kubernetes"
        ENV_DETAILS="Kubernetes Pod"
        return 0
    fi
    
    if [ -f /var/run/secrets/kubernetes.io/serviceaccount/token ]; then
        ENV_TYPE="kubernetes"
        ENV_DETAILS="Kubernetes Pod"
        return 0
    fi
    
    return 1
}

# 妫€娴?AWS
detect_aws() {
    if curl -s --connect-timeout 2 http://169.254.169.254/latest/meta-data/ > /dev/null 2>&1; then
        INSTANCE_TYPE=$(curl -s --connect-timeout 2 http://169.254.169.254/latest/meta-data/instance-type 2>/dev/null || echo "unknown")
        ENV_TYPE="cloud-aws"
        ENV_DETAILS="AWS EC2 ($INSTANCE_TYPE)"
        return 0
    fi
    
    return 1
}

# 妫€娴嬮樋閲屼簯
detect_aliyun() {
    if curl -s --connect-timeout 2 http://100.100.100.200/latest/meta-data/ > /dev/null 2>&1; then
        INSTANCE_TYPE=$(curl -s --connect-timeout 2 http://100.100.100.200/latest/meta-data/instance/instance-type 2>/dev/null || echo "unknown")
        ENV_TYPE="cloud-aliyun"
        ENV_DETAILS="Alibaba Cloud ECS ($INSTANCE_TYPE)"
        return 0
    fi
    
    return 1
}

# 妫€娴嬬兢鏅?NAS
detect_synology() {
    if [ -f /etc/synoinfo.conf ]; then
        MODEL=$(grep "upnpmodelname" /etc/synoinfo.conf 2>/dev/null | cut -d'"' -f2 || echo "unknown")
        ENV_TYPE="nas-synology"
        ENV_DETAILS="Synology NAS ($MODEL)"
        return 0
    fi
    
    return 1
}

# 妫€娴?QNAP NAS
detect_qnap() {
    if [ -f /etc/config/qpkg.conf ]; then
        MODEL=$(cat /etc/default_config/QNAP.conf 2>/dev/null | grep "Model" | cut -d'=' -f2 || echo "unknown")
        ENV_TYPE="nas-qnap"
        ENV_DETAILS="QNAP NAS ($MODEL)"
        return 0
    fi
    
    return 1
}

# 妫€娴嬫爲鑾撴淳
detect_raspberry() {
    if grep -q "Raspberry Pi" /proc/cpuinfo 2>/dev/null; then
        MODEL=$(grep "Model" /proc/cpuinfo | cut -d':' -f2 | xargs || echo "unknown")
        ENV_TYPE="raspberry"
        ENV_DETAILS="Raspberry Pi ($MODEL)"
        return 0
    fi
    
    return 1
}

# 妫€娴嬭櫄鎷熸満
detect_vm() {
    if command -v systemd-detect-virt &> /dev/null; then
        VIRT=$(systemd-detect-virt 2>/dev/null || echo "none")
        if [ "$VIRT" != "none" ]; then
            ENV_DETAILS="Virtual Machine ($VIRT)"
            return 0
        fi
    fi
    
    if grep -q "hypervisor" /proc/cpuinfo 2>/dev/null; then
        ENV_DETAILS="Virtual Machine"
        return 0
    fi
    
    return 1
}

# 鑾峰彇绯荤粺淇℃伅
get_system_info() {
    echo ""
    echo -e "${YELLOW}绯荤粺淇℃伅:${NC}"
    echo "----------------------------------------"
    
    # 鎿嶄綔绯荤粺
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        echo "鎿嶄綔绯荤粺: $NAME $VERSION_ID"
    else
        echo "鎿嶄綔绯荤粺: $(uname -s) $(uname -r)"
    fi
    
    # 鍐呮牳
    echo "鍐呮牳鐗堟湰: $(uname -r)"
    
    # 鏋舵瀯
    echo "绯荤粺鏋舵瀯: $(uname -m)"
    
    # CPU
    CPU_CORES=$(nproc 2>/dev/null || echo "unknown")
    echo "CPU 鏍稿績: $CPU_CORES"
    
    # 鍐呭瓨
    if [ -f /proc/meminfo ]; then
        MEM_TOTAL=$(grep MemTotal /proc/meminfo | awk '{printf "%.1f GB", $2/1024/1024}')
        MEM_FREE=$(grep MemAvailable /proc/meminfo | awk '{printf "%.1f GB", $2/1024/1024}')
        echo "鎬诲唴瀛? $MEM_TOTAL"
        echo "鍙敤鍐呭瓨: $MEM_FREE"
    fi
    
    # 纾佺洏
    DISK_INFO=$(df -h / | awk 'NR==2 {print "鎬昏: "$2", 宸茬敤: "$3", 鍙敤: "$4}')
    echo "纾佺洏绌洪棿: $DISK_INFO"
    
    # Docker
    if command -v docker &> /dev/null; then
        DOCKER_VERSION=$(docker --version 2>/dev/null | awk '{print $3}' | tr -d ',')
        echo "Docker: $DOCKER_VERSION"
    else
        echo "Docker: 鏈畨瑁?
    fi
}

# 涓绘娴嬫祦绋?main() {
    echo -e "${YELLOW}姝ｅ湪妫€娴嬭繍琛岀幆澧?..${NC}"
    echo ""
    
    # 鎸変紭鍏堢骇妫€娴?    detect_docker || \
    detect_kubernetes || \
    detect_aws || \
    detect_aliyun || \
    detect_synology || \
    detect_qnap || \
    detect_raspberry || \
    detect_vm || \
    ENV_DETAILS="Physical Machine"
    
    echo -e "${GREEN}妫€娴嬬粨鏋?${NC}"
    echo "----------------------------------------"
    echo "鐜绫诲瀷: $ENV_TYPE"
    echo "鐜璇︽儏: $ENV_DETAILS"
    
    get_system_info
    
    echo ""
    echo -e "${YELLOW}鎺ㄨ崘閰嶇疆妯℃澘: ${NC}$ENV_TYPE.yaml"
    echo ""
    
    # 杈撳嚭 JSON 鏍煎紡锛堜緵绋嬪簭浣跨敤锛?    if [ "$1" = "--json" ]; then
        echo ""
        echo '{"env_type":"'$ENV_TYPE'","env_details":"'$ENV_DETAILS'"}'
    fi
}

main "$@"
