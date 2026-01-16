#!/bin/bash
# CYP-Docker-Registry 蹇€熷惎鍔ㄨ剼鏈?
# Version: v1.2.1
# Author: CYP | Contact: nasDSSCYP@outlook.com

set -e

# 棰滆壊瀹氫箟
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}"
echo "鈺斺晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晽"
echo "鈺?  CYP-Docker-Registry 蹇€熷惎鍔ㄨ剼鏈?v1.2.1    鈺?
echo "鈺氣晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨暆"
echo -e "${NC}"

# 妫€鏌?Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}閿欒: Docker 鏈畨瑁?{NC}"
    echo "璇峰厛瀹夎 Docker: https://docs.docker.com/get-docker/"
    exit 1
fi

# 妫€鏌?Docker Compose
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo -e "${RED}閿欒: Docker Compose 鏈畨瑁?{NC}"
    exit 1
fi

# 鑾峰彇鑴氭湰鎵€鍦ㄧ洰褰?
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

# 妫€鏌ラ厤缃枃浠?
if [ ! -f "configs/config.yaml" ]; then
    echo -e "${YELLOW}閰嶇疆鏂囦欢涓嶅瓨鍦紝浠庣ず渚嬪垱寤?..${NC}"
    cp configs/config.yaml.example configs/config.yaml
fi

# 鏋勫缓骞跺惎鍔?
echo -e "${YELLOW}姝ｅ湪鏋勫缓骞跺惎鍔ㄦ湇鍔?..${NC}"

if command -v docker-compose &> /dev/null; then
    docker-compose up -d --build
else
    docker compose up -d --build
fi

# 绛夊緟鏈嶅姟鍚姩
echo -e "${YELLOW}绛夊緟鏈嶅姟鍚姩...${NC}"
sleep 5

# 妫€鏌ュ仴搴风姸鎬?
MAX_RETRIES=30
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
        echo -e "${GREEN}鉁?鏈嶅姟鍚姩鎴愬姛锛?{NC}"
        break
    fi
    
    RETRY_COUNT=$((RETRY_COUNT + 1))
    echo "绛夊緟鏈嶅姟灏辩华... ($RETRY_COUNT/$MAX_RETRIES)"
    sleep 2
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    echo -e "${RED}鏈嶅姟鍚姩瓒呮椂锛岃妫€鏌ユ棩蹇?{NC}"
    docker-compose logs --tail=50
    exit 1
fi

# 鎵撳嵃璁块棶淇℃伅
echo ""
echo -e "${GREEN}鈺斺晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晽${NC}"
echo -e "${GREEN}鈺?             鏈嶅姟宸叉垚鍔熷惎鍔紒                  鈺?{NC}"
echo -e "${GREEN}鈺氣晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨暆${NC}"
echo ""
echo "璁块棶鍦板潃: http://localhost:8080"
echo ""
echo "榛樿璐︽埛:"
echo "  鐢ㄦ埛鍚? admin"
echo "  瀵嗙爜: admin123"
echo ""
echo -e "${YELLOW}鈿?瀹夊叏鎻愮ず:${NC}"
echo "  - 棣栨鐧诲綍鍚庤绔嬪嵆淇敼榛樿瀵嗙爜"
echo "  - 鐧诲綍澶辫触3娆″皢閿佸畾绯荤粺"
echo "  - 瑙ｉ攣鍛戒护: ./scripts/unlock.sh"
echo ""
echo "甯哥敤鍛戒护:"
echo "  鏌ョ湅鏃ュ織: docker-compose logs -f"
echo "  鍋滄鏈嶅姟: docker-compose down"
echo "  閲嶅惎鏈嶅姟: docker-compose restart"
echo ""
