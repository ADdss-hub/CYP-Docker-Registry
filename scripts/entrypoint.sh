#!/bin/sh
# CYP-Docker-Registry 鐎圭懓娅掗崗銉ュ經閼存碍婀?# Version: v1.2.1
# Author: CYP | Contact: nasDSSCYP@outlook.com

set -e

# 妫版粏澹婃潏鎾冲毉
log_info() {
    echo "[INFO] $(date '+%Y-%m-%d %H:%M:%S') $1"
}

log_warn() {
    echo "[WARN] $(date '+%Y-%m-%d %H:%M:%S') $1"
}

log_error() {
    echo "[ERROR] $(date '+%Y-%m-%d %H:%M:%S') $1"
}

# 閹垫挸宓冮崥顖氬З娣団剝浼?print_banner() {
    echo ""
    echo "閳烘柡鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅?
    echo "閳?       CYP-Docker-Registry v1.2.1              閳?
    echo "閳?    闂嗘湹淇婃禒璇差啇閸ｃ劑鏆呴崓蹇曨潌閺堝绮ㄦ惔鎾额吀閻炲棛閮寸紒?            閳?
    echo "閳烘埃鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏅查埡鎰ㄦ櫜閳烘劏鏆?
    echo ""
}

# 濡偓閺屻儳娲拌ぐ鏇熸綀闂?
check_directories() {
    log_info "濡偓閺屻儳娲拌ぐ鏇熸綀闂?.."
    
    for dir in /app/data/blobs /app/data/meta /app/data/cache /app/data/signatures /app/data/sboms; do
        if [ ! -d "$dir" ]; then
            log_info "閸掓稑缂撻惄顔肩秿: $dir"
            mkdir -p "$dir"
        fi
        
        if [ ! -w "$dir" ]; then
            log_error "閻╊喖缍嶆稉宥呭讲閸? $dir"
            exit 1
        fi
    done
    
    log_info "閻╊喖缍嶅Λ鈧弻銉ョ暚閹?
}

# 濡偓閺屻儵鍘ょ純顔芥瀮娴?
check_config() {
    log_info "濡偓閺屻儵鍘ょ純顔芥瀮娴?.."
    
    CONFIG_FILE="${CONFIG_FILE:-/app/configs/config.yaml}"
    
    if [ ! -f "$CONFIG_FILE" ]; then
        log_warn "闁板秶鐤嗛弬鍥︽娑撳秴鐡ㄩ崷顭掔礉娴ｈ法鏁ゆ妯款吇闁板秶鐤?
        if [ -f "/app/configs/config.yaml.example" ]; then
            cp /app/configs/config.yaml.example "$CONFIG_FILE"
            log_info "瀹歌弓绮犵粈杞扮伐閸掓稑缂撻柊宥囩枂閺傚洣娆?
        fi
    fi
    
    log_info "闁板秶鐤嗛弬鍥︽: $CONFIG_FILE"
}

# 閸掓繂顫愰崠鏍ㄦ殶閹诡喖绨?init_database() {
    log_info "閸掓繂顫愰崠鏍ㄦ殶閹诡喖绨?.."
    
    DB_FILE="/app/data/meta/registry.db"
    
    if [ ! -f "$DB_FILE" ]; then
        log_info "閸掓稑缂撻弬鐗堟殶閹诡喖绨?.."
    else
        log_info "閺佺増宓佹惔鎾冲嚒鐎涙ê婀?
    fi
}

# 鐠佸墽鐤嗛悳顖氼暔閸欐﹢鍣烘妯款吇閸?
set_defaults() {
    export PORT="${PORT:-8080}"
    export HOST="${HOST:-0.0.0.0}"
    export LOG_LEVEL="${LOG_LEVEL:-info}"
    export TZ="${TZ:-Asia/Shanghai}"
}

# 閸嬨儱鎮嶅Λ鈧弻?
health_check() {
    curl -sf http://localhost:${PORT}/health > /dev/null 2>&1
    return $?
}

# 娴兼﹢娉ら崗鎶芥４
graceful_shutdown() {
    log_info "閺€璺哄煂閸忔娊妫存穱鈥冲娇閿涘本顒滈崷銊ょ喘闂嗗懎鍙ч梻?.."
    
    # 閸欐垿鈧?SIGTERM 缂佹瑤瀵屾潻娑氣柤
    if [ -n "$SERVER_PID" ]; then
        kill -TERM "$SERVER_PID" 2>/dev/null
        
        # 缁涘绶熸潻娑氣柤闁偓閸?
        wait "$SERVER_PID"
    fi
    
    log_info "閺堝秴濮熷鎻掑彠闂?
    exit 0
}

# 閹规洝骞忔穱鈥冲娇
trap graceful_shutdown SIGTERM SIGINT SIGQUIT

# 娑撹鍤遍弫?
main() {
    print_banner
    set_defaults
    check_directories
    check_config
    init_database
    
    log_info "閸氼垰濮?CYP-Docker-Registry 閺堝秴濮?.."
    log_info "閻╂垵鎯夐崷鏉挎絻: ${HOST}:${PORT}"
    
    # 閸氼垰濮╅張宥呭閸?
    exec /app/server "$@"
}

# 鏉╂劘顢戞稉璇插毐閺?
main "$@"
