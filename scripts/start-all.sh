#!/usr/bin/env bash
set -euo pipefail

# ====================================================================
# CampusVision AI — 一键启动所有服务
# ====================================================================
# 用法:
#   ./scripts/start-all.sh          # 默认 tmux 模式
#   ./scripts/start-all.sh --bg     # 后台进程模式 (输出到 logs/)
#   ./scripts/start-all.sh --help   # 帮助
# ====================================================================

SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$SCRIPT_DIR"

MODE="${1:-tmux}"
LOGDIR="$SCRIPT_DIR/logs"
SESSION="campusvision"

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

log()  { echo -e "${GREEN}[✓]${NC} $1"; }
warn() { echo -e "${YELLOW}[!]${NC} $1"; }
err()  { echo -e "${RED}[✗]${NC} $1"; }
info() { echo -e "${CYAN}[i]${NC} $1"; }

# ====================================================================
# Prerequisites check
# ====================================================================
check_prereqs() {
    info "检查环境依赖..."

    command -v docker >/dev/null 2>&1 || { err "docker 未安装"; exit 1; }
    command -v go >/dev/null 2>&1 || { err "go 未安装"; exit 1; }
    command -v pnpm >/dev/null 2>&1 || { err "pnpm 未安装"; exit 1; }
    command -v ffmpeg >/dev/null 2>&1 || { warn "ffmpeg 未安装 (stream-gateway 需要)"; }

    [ -f "face-recognition/.venv/bin/python" ] || { err "face-recognition/.venv 不存在，请先执行 uv sync"; exit 1; }
    [ -f "face-recognition/app/models/retinaface-R50.onnx" ] || warn "ONNX 模型未下载，face-recognition 会降级为 Haar Cascade"
    [ -f "face-recognition/app/models/arcface-resnet100.onnx" ] || warn "ONNX 模型未下载，face-recognition 会降级为零向量"

    log "环境检查通过"
}

# ====================================================================
# Start Docker infrastructure
# ====================================================================
start_infra() {
    info "启动 Docker 基础设施 (kafka, redis, mariadb)..."

    docker compose up -d kafka redis mariadb 2>&1

    info "等待服务健康检查..."
    for svc in kafka redis mariadb; do
        printf "  %-20s" "$svc"
        while true; do
            status=$(docker compose ps --format json "$svc" 2>/dev/null | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('State',''), d.get('Health',''))" 2>/dev/null || echo "starting")
            if echo "$status" | grep -q "running healthy"; then
                echo -e "${GREEN}✓ healthy${NC}"
                break
            fi
            printf "."
            sleep 2
        done
    done
    log "Docker 基础设施就绪"
}

# ====================================================================
# Services (每个服务定义一个启动命令)
# ====================================================================
SERVICES=(
    "stream-gateway:stream-gateway:go run cmd/main.go --config config.yaml"
    "face-recognition:face-recognition:.venv/bin/python -m app.main --config config.yaml"
    "dormitory-service:dormitory-service-go:CONFIG_PATH=config.yaml go run ./cmd/dormitory-service/"
    "frontend:frontend:pnpm dev"
)

# ====================================================================
# Mode: tmux
# ====================================================================
start_tmux() {
    info "创建 tmux 会话: $SESSION"

    tmux new-session -d -s "$SESSION" -n "infra" -c "$SCRIPT_DIR"
    tmux send-keys -t "$SESSION:infra" "docker compose logs -f" Enter

    for svc in "${SERVICES[@]}"; do
        IFS=':' read -r name dir cmd <<< "$svc"
        tmux new-window -t "$SESSION" -n "$name" -c "$SCRIPT_DIR/$dir"
        tmux send-keys -t "$SESSION:$name" "$cmd" Enter
    done

    tmux select-window -t "$SESSION:infra"

    echo ""
    log "所有服务已启动!"
    echo ""
    echo "  tmux 会话: $SESSION"
    echo ""
    echo "  窗口列表:"
    echo "    0: infra    — Docker 日志"
    echo "    1: stream-gateway  — RTSP 帧采集 (8080/8081)"
    echo "    2: face-recognition — 人脸识别 (Kafka)"
    echo "    3: dormitory-service — 业务 API (8083)"
    echo "    4: frontend — Vue 前端 (80)"
    echo ""
    echo "  连接: tmux attach -t $SESSION"
    echo "  分离: Ctrl+B d"
    echo "  停止: tmux kill-session -t $SESSION"
    echo ""
    echo "  端口: 8080(health) 8081(mgmt) 8083(API) 80(前端)"
    echo ""
}

# ====================================================================
# Mode: background (logs to files)
# ====================================================================
start_background() {
    mkdir -p "$LOGDIR"
    info "后台模式 — 日志输出到 $LOGDIR/"

    declare -A PIDS

    # Start infrastructure (already running from start_infra)

    for svc in "${SERVICES[@]}"; do
        IFS=':' read -r name dir cmd <<< "$svc"
        logfile="$LOGDIR/$name.log"
        info " 启动 $name ..."

        cd "$SCRIPT_DIR/$dir"
        # Use setsid to detach from shell
        nohup bash -c "$cmd" > "$logfile" 2>&1 &
        PIDS[$name]=$!
        cd "$SCRIPT_DIR"
    done

    echo ""
    log "所有服务已启动!" 
    echo ""
    echo "  PID 列表:"
    for name in "${!PIDS[@]}"; do
        printf "    %-20s PID %d\n" "$name" "${PIDS[$name]}"
    done
    echo ""
    echo "  日志文件: $LOGDIR/*.log"
    echo "  停止:     ./scripts/stop-all.sh"
    echo ""
    echo "  Ports:  8080(health) 8081(mgmt) 8083(API) 80(前端)"
    echo ""
}

# ====================================================================
# Main
# ====================================================================
case "${MODE:-}" in
    --help|-h)
        echo "用法: $0 [--bg]"
        echo "  默认:  tmux 模式 (创建 tmux 会话)"
        echo "  --bg:  后台进程模式 (日志输出到 logs/)"
        echo "  停止:  ./scripts/stop-all.sh"
        exit 0
        ;;
    --bg|-b)
        check_prereqs
        start_infra
        start_background
        log "停止: ./scripts/stop-all.sh"
        ;;
    *)
        check_prereqs
        start_infra
        start_tmux
        ;;
esac
