#!/bin/bash
# CYP-Docker-Registry 鐟欙綁鏀ｉ懘姘拱
# Version: v1.2.1
# 閻劋绨崷銊ч兇缂佺喕顫﹂柨浣哥暰閺冭埖澧滈崝銊ㄐ掗柨?

set -e

echo "閳烘柡鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅?
echo "閳?    CYP-Docker-Registry 缁崵绮虹憴锝夋敚瀹搞儱鍙?         閳?
echo "閳烘埃鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏆?
echo ""

# 濡偓閺屻儲妲搁崥锕€婀€圭懓娅掗崘鍛扮箥鐞?
if [ -f /.dockerenv ]; then
    HOST="localhost:8080"
else
    HOST="${CYP_HOST:-localhost:8080}"
fi

# 濡偓閺屻儳閮寸紒鐔兼敚鐎规氨濮搁幀?
echo "濮濓絽婀Λ鈧弻銉ч兇缂佺喓濮搁幀?.."
STATUS=$(curl -s "http://${HOST}/api/v1/system/lock/status" 2>/dev/null || echo '{"is_locked":false}')

IS_LOCKED=$(echo "$STATUS" | grep -o '"is_locked":[^,}]*' | cut -d':' -f2)

if [ "$IS_LOCKED" != "true" ]; then
    echo "閴?缁崵绮鸿ぐ鎾冲閺堫亪鏀ｇ€规熬绱濋弮鐘绘付鐟欙綁鏀ｉ妴?
    exit 0
fi

LOCK_REASON=$(echo "$STATUS" | grep -o '"lock_reason":"[^"]*"' | cut -d'"' -f4)
LOCKED_AT=$(echo "$STATUS" | grep -o '"locked_at":"[^"]*"' | cut -d'"' -f4)
LOCKED_BY_IP=$(echo "$STATUS" | grep -o '"locked_by_ip":"[^"]*"' | cut -d'"' -f4)

echo ""
echo "閳?缁崵绮哄鏌ユ敚鐎?
echo "  闁夸礁鐣鹃崢鐔锋礈: $LOCK_REASON"
echo "  闁夸礁鐣鹃弮鍫曟？: $LOCKED_AT"
echo "  鐟欙箑褰?IP:  $LOCKED_BY_IP"
echo ""

# 鐠囬攱鐪扮粻锛勬倞閸涙ê鐦戦惍?
read -p "鐠囩柉绶崗銉ь吀閻炲棗鎲崇€靛棛鐖? " -s PASSWORD
echo ""

if [ -z "$PASSWORD" ]; then
    echo "閴?鐎靛棛鐖滄稉宥堝厴娑撹櫣鈹?
    exit 1
fi

# 閸欐垿鈧浇袙闁夸浇顕Ч?
echo "濮濓絽婀憴锝夋敚缁崵绮?.."
RESPONSE=$(curl -s -X POST "http://${HOST}/api/v1/system/lock/unlock" \
    -H "Content-Type: application/json" \
    -d "{\"password\": \"$PASSWORD\"}" 2>/dev/null)

if echo "$RESPONSE" | grep -q '"message"'; then
    echo ""
    echo "閳烘柡鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅?
    echo "閳?             閴?缁崵绮虹憴锝夋敚閹存劕濮涢敍?                 閳?
    echo "閳烘埃鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏆?
    echo ""
    echo "瀵ら缚顔呴幙宥勭稊閿?
    echo "  1. 濡偓閺屻儱顓哥拋鈩冩）韫囨绱濇禍鍡毿掗柨浣哥暰閸樼喎娲?
    echo "  2. 婵″倹婀佽箛鍛邦洣閿涘奔鎱ㄩ弨鐟扮暔閸忋劎鐡ラ悾?
    echo "  3. 閼板啳妾婚弴瀛樻暭缁狅紕鎮婇崨妯虹槕閻?
else
    echo ""
    echo "閴?鐟欙綁鏀ｆ径杈Е: $RESPONSE"
    exit 1
fi
# CYP-Docker-Registry Unlock Script
# Author: CYP | Contact: nasDSSCYP@outlook.com

set -e

echo "=========================================="
echo "  CYP-Docker-Registry 缁崵绮虹憴锝夋敚瀹搞儱鍙?
echo "=========================================="
echo ""

# Check if running in Docker
if [ -f /.dockerenv ]; then
    API_URL="http://localhost:8080"
else
    API_URL="${CYP_REGISTRY_URL:-http://localhost:8080}"
fi

# Check lock status
echo "濡偓閺屻儳閮寸紒鐔兼敚鐎规氨濮搁幀?.."
LOCK_STATUS=$(curl -s "${API_URL}/api/v1/system/lock/status" 2>/dev/null || echo '{"is_locked": false}')

IS_LOCKED=$(echo "$LOCK_STATUS" | grep -o '"is_locked":[^,}]*' | cut -d':' -f2 | tr -d ' ')

if [ "$IS_LOCKED" != "true" ]; then
    echo "缁崵绮洪張顏囶潶闁夸礁鐣鹃敍灞炬￥闂団偓鐟欙綁鏀ｉ妴?
    exit 0
fi

echo "缁崵绮鸿ぐ鎾冲婢跺嫪绨柨浣哥暰閻樿埖鈧降鈧?
echo ""

# Get lock reason
LOCK_REASON=$(echo "$LOCK_STATUS" | grep -o '"lock_reason":"[^"]*"' | cut -d'"' -f4)
echo "闁夸礁鐣鹃崢鐔锋礈: $LOCK_REASON"
echo ""

# Prompt for password
read -sp "鐠囩柉绶崗銉ь吀閻炲棗鎲崇€靛棛鐖? " PASSWORD
echo ""

if [ -z "$PASSWORD" ]; then
    echo "闁挎瑨顕? 鐎靛棛鐖滄稉宥堝厴娑撹櫣鈹?
    exit 1
fi

# Attempt unlock
echo "濮濓絽婀亸婵婄槸鐟欙綁鏀?.."
RESPONSE=$(curl -s -X POST "${API_URL}/api/v1/system/lock/unlock" \
    -H "Content-Type: application/json" \
    -d "{\"password\": \"$PASSWORD\"}" 2>/dev/null)

if echo "$RESPONSE" | grep -q '"message"'; then
    echo ""
    echo "=========================================="
    echo "  缁崵绮虹憴锝夋敚閹存劕濮涢敍?
    echo "=========================================="
    echo ""
    echo "鐠囩兘鍣搁弬鎵瑜版洜閮寸紒鐔粹偓?
else
    echo ""
    echo "鐟欙綁鏀ｆ径杈Е: $RESPONSE"
    exit 1
fi
