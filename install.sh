#!/bin/bash
set -e
#=============================================================================
# Edge Dispatch Framework - 一键安装脚本
# 类似 1Panel 安装风格，支持 Control Plane / Edge Agent / Origin 三种角色
# 版本: v0.6
#=============================================================================

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'
CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

EDF_VERSION="v0.6"
EDF_REPO="https://github.com/DarkInno/Edge-Dispatch-Framework.git"
EDF_BASE="/opt/edge-dispatch"
EDF_DATA="/data/edf"
DOCKER_MODE=false

log_info()  { echo -e "${GREEN}[INFO]${NC}  $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step()  { echo -e "\n${CYAN}${BOLD}>>> $1${NC}"; }

#=============================================================================
# 环境检测
#=============================================================================
check_os() {
    log_step "检测系统环境"
    if [ "$(uname)" != "Linux" ]; then
        log_error "仅支持 Linux 系统"
        exit 1
    fi
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        OS_VERSION=$VERSION_ID
    else
        OS="unknown"
    fi
    log_info "操作系统: $OS $OS_VERSION"
    log_info "架构: $(uname -m)"
    log_info "内存: $(free -m | awk '/Mem:/{print $2}')MB"
    log_info "磁盘: $(df -h / | awk 'NR==2{print $4}') 可用"
}

#=============================================================================
# 依赖安装
#=============================================================================
install_deps() {
    log_step "安装系统依赖"
    if command -v apt-get &>/dev/null; then
        apt-get update -qq
        apt-get install -y -qq curl wget git gzip tar 2>/dev/null
    elif command -v yum &>/dev/null; then
        yum install -y -q curl wget git gzip tar 2>/dev/null
    fi

    # Docker (可选)
    if ! command -v docker &>/dev/null; then
        log_warn "Docker 未安装，将使用 Go 源码编译"
        install_go
    else
        log_info "Docker 已安装"
        DOCKER_MODE=true
    fi
}

install_go() {
    if command -v go &>/dev/null; then
        GO_VER=$(go version | awk '{print $3}')
        log_info "Go 已安装: $GO_VER"
        return
    fi
    log_info "安装 Go 1.22+..."
    GO_URL="https://go.dev/dl/go1.22.10.linux-amd64.tar.gz"
    curl -sL "$GO_URL" -o /tmp/go.tar.gz
    tar -C /usr/local -xzf /tmp/go.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    export PATH=$PATH:/usr/local/go/bin
    rm -f /tmp/go.tar.gz
    log_info "Go 安装完成: $(go version)"
}

#=============================================================================
# 下载与编译
#=============================================================================
build_from_source() {
    log_step "编译 Edge Dispatch Framework"
    mkdir -p "$EDF_BASE"

    if [ -d "$EDF_BASE/src" ]; then
        cd "$EDF_BASE/src"
        git pull origin main 2>/dev/null || true
    else
        git clone "$EDF_REPO" "$EDF_BASE/src" 2>/dev/null || {
            log_warn "GitHub 不可达，请手动上传源码到 $EDF_BASE/src"
            return 1
        }
    fi

    cd "$EDF_BASE/src"
    export GOPROXY='https://goproxy.cn,direct'
    go build -ldflags="-s -w" -o "$EDF_BASE/control-plane" ./cmd/control-plane/
    go build -ldflags="-s -w" -o "$EDF_BASE/edge-agent" ./cmd/edge-agent/
    go build -ldflags="-s -w" -o "$EDF_BASE/origin" ./cmd/origin/
    log_info "编译完成"
}

#=============================================================================
# Control Plane 安装
#=============================================================================
install_control_plane() {
    log_step "安装 Control Plane"

    read -p "监听地址 (默认 :8080): " LISTEN_ADDR
    LISTEN_ADDR=${LISTEN_ADDR:-:8080}

    read -p "PostgreSQL 连接串 (默认 postgres://edf:edf@localhost:5432/edf?sslmode=disable): " PG_URL
    PG_URL=${PG_URL:-postgres://edf:edf@localhost:5432/edf?sslmode=disable}

    read -p "Redis 地址 (默认 localhost:6379): " REDIS_ADDR
    REDIS_ADDR=${REDIS_ADDR:-localhost:6379}

    read -sp "Token Secret (留空自动生成): " TOKEN_SECRET
    echo
    TOKEN_SECRET=${TOKEN_SECRET:-$(openssl rand -hex 32)}

    read -p "源站地址 (默认 http://localhost:7070): " ORIGIN_URL
    ORIGIN_URL=${ORIGIN_URL:-http://localhost:7070}

    read -p "启用小带宽优化? (y/n, 默认 y): " SB_ENABLED
    SB_ENABLED=${SB_ENABLED:-y}
    if [ "$SB_ENABLED" = "y" ]; then SB_ENABLED="true"; else SB_ENABLED="false"; fi

    cat > /etc/systemd/system/edf-cp.service << EOF
[Unit]
Description=Edge Dispatch Control Plane
After=network.target

[Service]
Type=simple
ExecStart=$EDF_BASE/control-plane
Environment="CP_LISTEN_ADDR=$LISTEN_ADDR"
Environment="CP_PG_URL=$PG_URL"
Environment="CP_REDIS_ADDR=$REDIS_ADDR"
Environment="CP_TOKEN_SECRET=$TOKEN_SECRET"
Environment="CP_ORIGIN_URL=$ORIGIN_URL"
Environment="CP_SB_OPT_ENABLED=$SB_ENABLED"
Environment="CP_SB_THRESHOLD=50"
Environment="CP_MAX_CANDIDATES=5"
Environment="CP_DEFAULT_TTL_MS=30000"
Environment="CP_DEGRADE_TO_ORIGIN=true"
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable edf-cp
    systemctl start edf-cp
    log_info "Control Plane 已安装并启动"
}

#=============================================================================
# Edge Agent 安装
#=============================================================================
install_edge_agent() {
    log_step "安装 Edge Agent"

    read -p "Control Plane 地址 (默认 http://127.0.0.1:8080): " CP_URL
    CP_URL=${CP_URL:-http://127.0.0.1:8080}

    read -p "源站地址 (默认 http://127.0.0.1:7070): " ORIGIN_URL
    ORIGIN_URL=${ORIGIN_URL:-http://127.0.0.1:7070}

    read -sp "Node Token: " NODE_TOKEN
    echo
    if [ -z "$NODE_TOKEN" ]; then
        log_error "Node Token 不能为空"
        exit 1
    fi

    read -p "公网IP (留空自动检测): " PUBLIC_HOST
    PUBLIC_HOST=${PUBLIC_HOST:-""}

    read -p "监听地址 (默认 :9090): " LISTEN_ADDR
    LISTEN_ADDR=${LISTEN_ADDR:-:9090}

    read -p "上行带宽 Mbps (默认 100): " MAX_UPLINK
    MAX_UPLINK=${MAX_UPLINK:-100}

    read -p "缓存目录 (默认 $EDF_DATA/cache): " CACHE_DIR
    CACHE_DIR=${CACHE_DIR:-$EDF_DATA/cache}

    read -p "最大缓存 GB (默认 50): " CACHE_MAX
    CACHE_MAX=${CACHE_MAX:-50}

    read -p "区域 (如 cn-hk, 默认 auto): " REGION
    REGION=${REGION:-auto}

    read -p "ISP (如 bgp, 默认 auto): " ISP
    ISP=${ISP:-auto}

    read -p "启用 P2P? (y/n, 默认 y): " P2P
    P2P=${P2P:-y}
    if [ "$P2P" = "y" ]; then P2P="true"; else P2P="false"; fi

    read -p "启用预拉取? (y/n, 默认 y): " PREFETCH
    PREFETCH=${PREFETCH:-y}
    if [ "$PREFETCH" = "y" ]; then PREFETCH="true"; else PREFETCH="false"; fi

    mkdir -p "$CACHE_DIR"

    ENV_LINES=""
    ENV_LINES="$ENV_LINES""Environment=\"EA_LISTEN_ADDR=$LISTEN_ADDR\"\n"
    ENV_LINES="$ENV_LINES""Environment=\"EA_CONTROL_PLANE_URL=$CP_URL\"\n"
    ENV_LINES="$ENV_LINES""Environment=\"EA_ORIGIN_URL=$ORIGIN_URL\"\n"
    ENV_LINES="$ENV_LINES""Environment=\"EA_NODE_TOKEN=$NODE_TOKEN\"\n"
    ENV_LINES="$ENV_LINES""Environment=\"EA_CACHE_DIR=$CACHE_DIR\"\n"
    ENV_LINES="$ENV_LINES""Environment=\"EA_CACHE_MAX_GB=$CACHE_MAX\"\n"
    ENV_LINES="$ENV_LINES""Environment=\"EA_MAX_UPLINK_MBPS=$MAX_UPLINK\"\n"
    ENV_LINES="$ENV_LINES""Environment=\"EA_P2P_ENABLED=$P2P\"\n"
    ENV_LINES="$ENV_LINES""Environment=\"EA_PREFETCH_ENABLED=$PREFETCH\"\n"
    ENV_LINES="$ENV_LINES""Environment=\"EA_PREFETCH_WORKERS=2\"\n"
    ENV_LINES="$ENV_LINES""Environment=\"EA_PREFETCH_BANDWIDTH_LIMIT=20\"\n"
    ENV_LINES="$ENV_LINES""Environment=\"EA_REGION=$REGION\"\n"
    ENV_LINES="$ENV_LINES""Environment=\"EA_ISP=$ISP\"\n"
    if [ -n "$PUBLIC_HOST" ]; then
        ENV_LINES="$ENV_LINES""Environment=\"EA_PUBLIC_HOST=$PUBLIC_HOST\"\n"
    fi

    cat > /etc/systemd/system/edf-ea.service << EOF
[Unit]
Description=Edge Dispatch Edge Agent
After=network.target

[Service]
Type=simple
ExecStart=$EDF_BASE/edge-agent
$(echo -e "$ENV_LINES")
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable edf-ea
    systemctl start edf-ea
    log_info "Edge Agent 已安装并启动"
}

#=============================================================================
# Origin 安装
#=============================================================================
install_origin() {
    log_step "安装 Origin 源站"

    read -p "监听地址 (默认 :7070): " LISTEN_ADDR
    LISTEN_ADDR=${LISTEN_ADDR:-:7070}

    read -p "数据目录 (默认 $EDF_DATA/origin): " DATA_DIR
    DATA_DIR=${DATA_DIR:-$EDF_DATA/origin}

    mkdir -p "$DATA_DIR"

    cat > /etc/systemd/system/edf-origin.service << EOF
[Unit]
Description=Edge Dispatch Origin Server
After=network.target

[Service]
Type=simple
ExecStart=$EDF_BASE/origin
Environment="ORIGIN_LISTEN_ADDR=$LISTEN_ADDR"
Environment="ORIGIN_DATA_DIR=$DATA_DIR"
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable edf-origin
    systemctl start edf-origin
    log_info "Origin 已安装并启动"
    log_info "将文件放入 $DATA_DIR 即可通过边缘节点分发"
}

#=============================================================================
# 全栈安装 (使用 Docker)
#=============================================================================
install_docker_full() {
    log_step "Docker 全栈部署"
    log_info "注意: 需要先安装 Docker 和 Docker Compose"

    cd "$EDF_BASE/src"
    docker compose up -d
    log_info "全栈已启动"
    docker compose ps
}

#=============================================================================
# 状态检查
#=============================================================================
check_status() {
    log_step "服务状态"
    for svc in edf-cp edf-ea edf-origin; do
        if systemctl is-active --quiet $svc 2>/dev/null; then
            echo -e "  ${GREEN}●${NC} $svc"
        else
            echo -e "  ${RED}○${NC} $svc (未运行)"
        fi
    done
}

#=============================================================================
# 主菜单
#=============================================================================
main_menu() {
    clear
    echo -e "${CYAN}${BOLD}"
    echo "  ╔══════════════════════════════════════════╗"
    echo "  ║     Edge Dispatch Framework v0.6        ║"
    echo "  ║     边缘分发加速框架 - 一键安装          ║"
    echo "  ╚══════════════════════════════════════════╝"
    echo -e "${NC}"

    check_os

    echo ""
    echo "  请选择安装角色:"
    echo "  ┌─────────────────────────────────────────┐"
    echo "  │  1) Control Plane  - 调度中心            │"
    echo "  │  2) Edge Agent     - 边缘缓存节点        │"
    echo "  │  3) Origin         - 源站服务            │"
    echo "  │  4) 全栈 (Docker)  - 一键部署全部        │"
    echo "  │  5) 仅编译         - 只构建二进制        │"
    echo "  │  6) 查看状态                             │"
    echo "  │  0) 退出                                 │"
    echo "  └─────────────────────────────────────────┘"
    echo ""

    read -p "请输入选项 [1-6]: " CHOICE

    case $CHOICE in
        1)
            install_deps
            build_from_source
            install_control_plane
            check_status
            ;;
        2)
            install_deps
            build_from_source
            install_edge_agent
            check_status
            ;;
        3)
            install_deps
            build_from_source
            install_origin
            check_status
            ;;
        4)
            install_deps
            build_from_source
            install_docker_full
            ;;
        5)
            install_deps
            build_from_source
            log_info "二进制文件位于 $EDF_BASE/"
            ls -lh "$EDF_BASE"/{control-plane,edge-agent,origin} 2>/dev/null
            ;;
        6)
            check_status
            ;;
        0)
            exit 0
            ;;
        *)
            log_error "无效选项"
            ;;
    esac
}

#=============================================================================
# 命令行模式
#=============================================================================
if [ "$1" = "--quick" ]; then
    # 快速全栈部署 (非交互)
    check_os
    install_deps
    build_from_source

    # 启动 Origin
    mkdir -p "$EDF_DATA/origin"
    nohup "$EDF_BASE/origin" > /var/log/edf-origin.log 2>&1 &
    log_info "Origin 启动 :7070"

    # 启动 Control Plane (需要 PG + Redis)
    if command -v docker &>/dev/null; then
        docker run -d --name edf-pg -e POSTGRES_USER=edf -e POSTGRES_PASSWORD=edf -e POSTGRES_DB=edf -p 5432:5432 postgres:16-alpine 2>/dev/null || true
        docker run -d --name edf-redis -p 6379:6379 redis:7-alpine 2>/dev/null || true
        sleep 3
    fi

    TOKEN=$(openssl rand -hex 32 2>/dev/null || echo "auto-$(date +%s)")
    nohup env CP_LISTEN_ADDR=:8080 CP_TOKEN_SECRET="$TOKEN" \
        CP_PG_URL="postgres://edf:edf@localhost:5432/edf?sslmode=disable" \
        CP_REDIS_ADDR=localhost:6379 CP_DEGRADE_TO_ORIGIN=true \
        CP_ORIGIN_URL=http://localhost:7070 CP_SB_OPT_ENABLED=true \
        "$EDF_BASE/control-plane" > /var/log/edf-cp.log 2>&1 &
    log_info "Control Plane 启动 :8080"
    log_info "Token: $TOKEN"

    echo ""
    echo "============================================"
    echo "  快速部署完成!"
    echo "  Control Plane: http://localhost:8080"
    echo "  Origin:        http://localhost:7070"
    echo "  Token:         $TOKEN"
    echo "============================================"
    echo ""
    echo "  下一步: 在边缘节点运行:"
    echo "  bash $0 --edge cp_url=$CP_ADDR token=$TOKEN"
    echo ""

elif [ "$1" = "--edge" ]; then
    # 快速部署 Edge Agent
    CP_URL="http://localhost:8080"
    TOKEN=""
    for arg in "$@"; do
        case $arg in
            cp_url=*) CP_URL="${arg#*=}" ;;
            token=*) TOKEN="${arg#*=}" ;;
        esac
    done
    if [ -z "$TOKEN" ]; then
        log_error "需要 --token= 参数"
        exit 1
    fi
    check_os
    install_deps
    build_from_source
    mkdir -p "$EDF_DATA/cache"
    nohup env EA_LISTEN_ADDR=:9090 EA_NODE_TOKEN="$TOKEN" \
        EA_CONTROL_PLANE_URL="$CP_URL" EA_CACHE_DIR="$EDF_DATA/cache" \
        EA_CACHE_MAX_GB=50 EA_MAX_UPLINK_MBPS=100 EA_P2P_ENABLED=true \
        EA_PREFETCH_ENABLED=true \
        "$EDF_BASE/edge-agent" > /var/log/edf-ea.log 2>&1 &
    log_info "Edge Agent 启动 :9090 → $CP_URL"

elif [ "$1" = "--status" ]; then
    check_status
else
    main_menu
fi
