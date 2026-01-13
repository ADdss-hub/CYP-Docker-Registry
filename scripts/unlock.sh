#!/bin/bash
# CYP-Registry 解锁脚本
# 用于在系统被锁定时手动解锁

set -e

echo "╔════════════════════════════════════════════════╗"
echo "║        CYP-Registry 系统解锁工具               ║"
echo "╚════════════════════════════════════════════════╝"
echo ""

# 检查是否在容器内运行
if [ -f /.dockerenv ]; then
    HOST="localhost:8080"
else
    HOST="${CYP_HOST:-localhost:8080}"
fi

# 检查系统锁定状态
echo "正在检查系统状态..."
STATUS=$(curl -s "http://${HOST}/api/v1/system/lock/status" 2>/dev/null || echo '{"is_locked":false}')

IS_LOCKED=$(echo "$STATUS" | grep -o '"is_locked":[^,}]*' | cut -d':' -f2)

if [ "$IS_LOCKED" != "true" ]; then
    echo "✓ 系统当前未锁定，无需解锁。"
    exit 0
fi

LOCK_REASON=$(echo "$STATUS" | grep -o '"lock_reason":"[^"]*"' | cut -d'"' -f4)
LOCKED_AT=$(echo "$STATUS" | grep -o '"locked_at":"[^"]*"' | cut -d'"' -f4)
LOCKED_BY_IP=$(echo "$STATUS" | grep -o '"locked_by_ip":"[^"]*"' | cut -d'"' -f4)

echo ""
echo "⚠ 系统已锁定"
echo "  锁定原因: $LOCK_REASON"
echo "  锁定时间: $LOCKED_AT"
echo "  触发 IP:  $LOCKED_BY_IP"
echo ""

# 请求管理员密码
read -p "请输入管理员密码: " -s PASSWORD
echo ""

if [ -z "$PASSWORD" ]; then
    echo "✗ 密码不能为空"
    exit 1
fi

# 发送解锁请求
echo "正在解锁系统..."
RESPONSE=$(curl -s -X POST "http://${HOST}/api/v1/system/lock/unlock" \
    -H "Content-Type: application/json" \
    -d "{\"password\": \"$PASSWORD\"}" 2>/dev/null)

if echo "$RESPONSE" | grep -q '"message"'; then
    echo ""
    echo "╔════════════════════════════════════════════════╗"
    echo "║              ✓ 系统解锁成功！                  ║"
    echo "╚════════════════════════════════════════════════╝"
    echo ""
    echo "建议操作："
    echo "  1. 检查审计日志，了解锁定原因"
    echo "  2. 如有必要，修改安全策略"
    echo "  3. 考虑更改管理员密码"
else
    echo ""
    echo "✗ 解锁失败: $RESPONSE"
    exit 1
fi
# CYP-Registry Unlock Script
# Author: CYP | Contact: nasDSSCYP@outlook.com

set -e

echo "=========================================="
echo "  CYP-Registry 系统解锁工具"
echo "=========================================="
echo ""

# Check if running in Docker
if [ -f /.dockerenv ]; then
    API_URL="http://localhost:8080"
else
    API_URL="${CYP_REGISTRY_URL:-http://localhost:8080}"
fi

# Check lock status
echo "检查系统锁定状态..."
LOCK_STATUS=$(curl -s "${API_URL}/api/v1/system/lock/status" 2>/dev/null || echo '{"is_locked": false}')

IS_LOCKED=$(echo "$LOCK_STATUS" | grep -o '"is_locked":[^,}]*' | cut -d':' -f2 | tr -d ' ')

if [ "$IS_LOCKED" != "true" ]; then
    echo "系统未被锁定，无需解锁。"
    exit 0
fi

echo "系统当前处于锁定状态。"
echo ""

# Get lock reason
LOCK_REASON=$(echo "$LOCK_STATUS" | grep -o '"lock_reason":"[^"]*"' | cut -d'"' -f4)
echo "锁定原因: $LOCK_REASON"
echo ""

# Prompt for password
read -sp "请输入管理员密码: " PASSWORD
echo ""

if [ -z "$PASSWORD" ]; then
    echo "错误: 密码不能为空"
    exit 1
fi

# Attempt unlock
echo "正在尝试解锁..."
RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/system/lock/unlock" \
    -H "Content-Type: application/json" \
    -d "{\"password\": \"$PASSWORD\"}" 2>/dev/null)

if echo "$RESPONSE" | grep -q '"message"'; then
    echo ""
    echo "=========================================="
    echo "  系统解锁成功！"
    echo "=========================================="
    echo ""
    echo "请重新登录系统。"
else
    echo ""
    echo "解锁失败: $RESPONSE"
    exit 1
fi
