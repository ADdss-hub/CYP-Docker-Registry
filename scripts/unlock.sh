#!/bin/bash
# CYP-Docker-Registry 瑙ｉ攣鑴氭湰
# Version: v1.2.1
# 鐢ㄤ簬鍦ㄧ郴缁熻閿佸畾鏃舵墜鍔ㄨВ閿?

set -e

echo "鈺斺晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晽"
echo "鈺?    CYP-Docker-Registry 绯荤粺瑙ｉ攣宸ュ叿          鈺?
echo "鈺氣晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨暆"
echo ""

# 妫€鏌ユ槸鍚﹀湪瀹瑰櫒鍐呰繍琛?
if [ -f /.dockerenv ]; then
    HOST="localhost:8080"
else
    HOST="${CYP_HOST:-localhost:8080}"
fi

# 妫€鏌ョ郴缁熼攣瀹氱姸鎬?
echo "姝ｅ湪妫€鏌ョ郴缁熺姸鎬?.."
STATUS=$(curl -s "http://${HOST}/api/v1/system/lock/status" 2>/dev/null || echo '{"is_locked":false}')

IS_LOCKED=$(echo "$STATUS" | grep -o '"is_locked":[^,}]*' | cut -d':' -f2)

if [ "$IS_LOCKED" != "true" ]; then
    echo "鉁?绯荤粺褰撳墠鏈攣瀹氾紝鏃犻渶瑙ｉ攣銆?
    exit 0
fi

LOCK_REASON=$(echo "$STATUS" | grep -o '"lock_reason":"[^"]*"' | cut -d'"' -f4)
LOCKED_AT=$(echo "$STATUS" | grep -o '"locked_at":"[^"]*"' | cut -d'"' -f4)
LOCKED_BY_IP=$(echo "$STATUS" | grep -o '"locked_by_ip":"[^"]*"' | cut -d'"' -f4)

echo ""
echo "鈿?绯荤粺宸查攣瀹?
echo "  閿佸畾鍘熷洜: $LOCK_REASON"
echo "  閿佸畾鏃堕棿: $LOCKED_AT"
echo "  瑙﹀彂 IP:  $LOCKED_BY_IP"
echo ""

# 璇锋眰绠＄悊鍛樺瘑鐮?
read -p "璇疯緭鍏ョ鐞嗗憳瀵嗙爜: " -s PASSWORD
echo ""

if [ -z "$PASSWORD" ]; then
    echo "鉁?瀵嗙爜涓嶈兘涓虹┖"
    exit 1
fi

# 鍙戦€佽В閿佽姹?
echo "姝ｅ湪瑙ｉ攣绯荤粺..."
RESPONSE=$(curl -s -X POST "http://${HOST}/api/v1/system/lock/unlock" \
    -H "Content-Type: application/json" \
    -d "{\"password\": \"$PASSWORD\"}" 2>/dev/null)

if echo "$RESPONSE" | grep -q '"message"'; then
    echo ""
    echo "鈺斺晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晽"
    echo "鈺?             鉁?绯荤粺瑙ｉ攣鎴愬姛锛?                 鈺?
    echo "鈺氣晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨晲鈺愨暆"
    echo ""
    echo "寤鸿鎿嶄綔锛?
    echo "  1. 妫€鏌ュ璁℃棩蹇楋紝浜嗚В閿佸畾鍘熷洜"
    echo "  2. 濡傛湁蹇呰锛屼慨鏀瑰畨鍏ㄧ瓥鐣?
    echo "  3. 鑰冭檻鏇存敼绠＄悊鍛樺瘑鐮?
else
    echo ""
    echo "鉁?瑙ｉ攣澶辫触: $RESPONSE"
    exit 1
fi
# CYP-Docker-Registry Unlock Script
# Author: CYP | Contact: nasDSSCYP@outlook.com

set -e

echo "=========================================="
echo "  CYP-Docker-Registry 绯荤粺瑙ｉ攣宸ュ叿"
echo "=========================================="
echo ""

# Check if running in Docker
if [ -f /.dockerenv ]; then
    API_URL="http://localhost:8080"
else
    API_URL="${CYP_REGISTRY_URL:-http://localhost:8080}"
fi

# Check lock status
echo "妫€鏌ョ郴缁熼攣瀹氱姸鎬?.."
LOCK_STATUS=$(curl -s "${API_URL}/api/v1/system/lock/status" 2>/dev/null || echo '{"is_locked": false}')

IS_LOCKED=$(echo "$LOCK_STATUS" | grep -o '"is_locked":[^,}]*' | cut -d':' -f2 | tr -d ' ')

if [ "$IS_LOCKED" != "true" ]; then
    echo "绯荤粺鏈閿佸畾锛屾棤闇€瑙ｉ攣銆?
    exit 0
fi

echo "绯荤粺褰撳墠澶勪簬閿佸畾鐘舵€併€?
echo ""

# Get lock reason
LOCK_REASON=$(echo "$LOCK_STATUS" | grep -o '"lock_reason":"[^"]*"' | cut -d'"' -f4)
echo "閿佸畾鍘熷洜: $LOCK_REASON"
echo ""

# Prompt for password
read -sp "璇疯緭鍏ョ鐞嗗憳瀵嗙爜: " PASSWORD
echo ""

if [ -z "$PASSWORD" ]; then
    echo "閿欒: 瀵嗙爜涓嶈兘涓虹┖"
    exit 1
fi

# Attempt unlock
echo "姝ｅ湪灏濊瘯瑙ｉ攣..."
RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/system/lock/unlock" \
    -H "Content-Type: application/json" \
    -d "{\"password\": \"$PASSWORD\"}" 2>/dev/null)

if echo "$RESPONSE" | grep -q '"message"'; then
    echo ""
    echo "=========================================="
    echo "  绯荤粺瑙ｉ攣鎴愬姛锛?
    echo "=========================================="
    echo ""
    echo "璇烽噸鏂扮櫥褰曠郴缁熴€?
else
    echo ""
    echo "瑙ｉ攣澶辫触: $RESPONSE"
    exit 1
fi
