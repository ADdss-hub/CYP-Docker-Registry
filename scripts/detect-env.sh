#!/bin/bash
# CYP-Docker-Registry 环境检测脚本
# Version: v1.0.8
# Author: CYP | Contact: nasDSSCYP@outlook.com

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}"
echo "╔════════════════════════════════════════════════╗"
echo "║     CYP-Docker-Registry 环境检测工具 v1.0.8  ║"
echo "╚════════════════════════════════════════════════╝"
echo -e "${NC}"

# 检测结果
ENV_TYPE="physical"
ENV_DETAILS=""

# 检测 Docker 容器
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

# 检测 Kubernetes
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

# 检测 AWS
detect_aws() {
    if curl -s --connect-timeout 2 http://169.254.169.254/latest/meta-data/ > /dev/null 2>&1; then
        INSTANCE_TYPE=$(curl -s --connect-timeout 2 http://169.254.169.254/latest/meta-data/instance-type 2>/dev/null || echo "unknown")
        ENV_TYPE="cloud-aws"
        ENV_DETAILS="AWS EC2 ($INSTANCE_TYPE)"
        return 0
    fi
    
    return 1
}

# 检测阿里云
detect_aliyun() {
    if curl -s --connect-timeout 2 http://100.100.100.200/latest/meta-data/ > /dev/null 2>&1; then
        INSTANCE_TYPE=$(curl -s --connect-timeout 2 http://100.100.100.200/latest/meta-data/instance/instance-type 2>/dev/null || echo "unknown")
        ENV_TYPE="cloud-aliyun"
        ENV_DETAILS="Alibaba Cloud ECS ($INSTANCE_TYPE)"
        return 0
    fi
    
    return 1
}

# 检测群晖 NAS
detect_synology() {
    if [ -f /etc/synoinfo.conf ]; then
        MODEL=$(grep "upnpmodelname" /etc/synoinfo.conf 2>/dev/null | cut -d'"' -f2 || echo "unknown")
        ENV_TYPE="nas-synology"
        ENV_DETAILS="Synology NAS ($MODEL)"
        return 0
    fi
    
    return 1
}

# 检测 QNAP NAS
detect_qnap() {
    if [ -f /etc/config/qpkg.conf ]; then
        MODEL=$(cat /etc/default_config/QNAP.conf 2>/dev/null | grep "Model" | cut -d'=' -f2 || echo "unknown")
        ENV_TYPE="nas-qnap"
        ENV_DETAILS="QNAP NAS ($MODEL)"
        return 0
    fi
    
    return 1
}

# 检测树莓派
detect_raspberry() {
    if grep -q "Raspberry Pi" /proc/cpuinfo 2>/dev/null; then
        MODEL=$(grep "Model" /proc/cpuinfo | cut -d':' -f2 | xargs || echo "unknown")
        ENV_TYPE="raspberry"
        ENV_DETAILS="Raspberry Pi ($MODEL)"
        return 0
    fi
    
    return 1
}

# 检测虚拟机
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

# 获取系统信息
get_system_info() {
    echo ""
    echo -e "${YELLOW}系统信息:${NC}"
    echo "----------------------------------------"
    
    # 操作系统
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        echo "操作系统: $NAME $VERSION_ID"
    else
        echo "操作系统: $(uname -s) $(uname -r)"
    fi
    
    # 内核
    echo "内核版本: $(uname -r)"
    
    # 架构
    echo "系统架构: $(uname -m)"
    
    # CPU
    CPU_CORES=$(nproc 2>/dev/null || echo "unknown")
    echo "CPU 核心: $CPU_CORES"
    
    # 内存
    if [ -f /proc/meminfo ]; then
        MEM_TOTAL=$(grep MemTotal /proc/meminfo | awk '{printf "%.1f GB", $2/1024/1024}')
        MEM_FREE=$(grep MemAvailable /proc/meminfo | awk '{printf "%.1f GB", $2/1024/1024}')
        echo "总内存: $MEM_TOTAL"
        echo "可用内存: $MEM_FREE"
    fi
    
    # 磁盘
    DISK_INFO=$(df -h / | awk 'NR==2 {print "总计: "$2", 已用: "$3", 可用: "$4}')
    echo "磁盘空间: $DISK_INFO"
    
    # Docker
    if command -v docker &> /dev/null; then
        DOCKER_VERSION=$(docker --version 2>/dev/null | awk '{print $3}' | tr -d ',')
        echo "Docker: $DOCKER_VERSION"
    else
        echo "Docker: 未安装"
    fi
}

# 主检测流程
main() {
    echo -e "${YELLOW}正在检测运行环境...${NC}"
    echo ""
    
    # 按优先级检测
    detect_docker || \
    detect_kubernetes || \
    detect_aws || \
    detect_aliyun || \
    detect_synology || \
    detect_qnap || \
    detect_raspberry || \
    detect_vm || \
    ENV_DETAILS="Physical Machine"
    
    echo -e "${GREEN}检测结果:${NC}"
    echo "----------------------------------------"
    echo "环境类型: $ENV_TYPE"
    echo "环境详情: $ENV_DETAILS"
    
    get_system_info
    
    echo ""
    echo -e "${YELLOW}推荐配置模板: ${NC}$ENV_TYPE.yaml"
    echo ""
    
    # 输出 JSON 格式（供程序使用）
    if [ "$1" = "--json" ]; then
        echo ""
        echo '{"env_type":"'$ENV_TYPE'","env_details":"'$ENV_DETAILS'"}'
    fi
}

main "$@"
