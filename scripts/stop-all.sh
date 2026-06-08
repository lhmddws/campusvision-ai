#!/usr/bin/env bash
set -euo pipefail

# ====================================================================
# CampusVision AI — 一键关闭所有服务
# ====================================================================
# 用法:
#   ./scripts/stop-all.sh                  # 关闭服务，保留 Docker 基础设施
#   ./scripts/stop-all.sh --docker-down    # 关闭服务 + 停 Docker 容器
#   ./scripts/stop-all.sh --docker-down -v # 关闭服务 + 停 Docker 容器 + 删数据卷
#   ./scripts/stop-all.sh --help           # 帮助
# ====================================================================

SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$SCRIPT_DIR"

SESSION="campusvision"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log()  { echo -e "${GREEN}[✓]${NC} $1"; }
warn() { echo -e "${YELLOW}[!]${NC} $1"; }
info() { echo -e "${CYAN}[i]${NC} $1"; }

# ====================================================================
# Step 1: Kill tmux session
# ====================================================================
stop_tmux() {
    if tmux has-session -t "$SESSION" 2>/dev/null; then
        tmux kill-session -t "$SESSION"
        log "tmux 会话 '$SESSION' 已关闭"
    else
        info "tmux 会话 '$SESSION' 不存在，跳过"
    fi
}

# ====================================================================
# Step 2: Kill background processes (--bg 模式)
# ====================================================================
stop_background() {
    # 查找本项目启动的后台进程：Go stream-gateway / Python face-recognition / pnpm frontend
    local killed=0

    # stream-gateway (Go)
    if pkill -f "stream-gateway/cmd/main.go" 2>/dev/null; then
        log "已终止 stream-gateway"
        killed=1
    fi

    # face-recognition (Python)
    if pkill -f "face-recognition.*app.main" 2>/dev/null; then
        log "已终止 face-recognition"
        killed=1
    fi

    # dormitory-service (Go)
    if pkill -f "dormitory-service-go.*cmd/dormitory-service" 2>/dev/null; then
        log "已终止 dormitory-service"
        killed=1
    fi

    # pnpm dev (frontend)
    if pkill -f "pnpm.*dev" 2>/dev/null; then
        log "已终止 frontend"
        killed=1
    fi

    [ "$killed" -eq 0 ] && info "未找到后台进程"
}

# ====================================================================
# Step 3: Docker infra
# ====================================================================
stop_docker() {
    local action="${1:-}"
    local volumes="${2:-}"
    case "$action" in
        down)
            if [ "$volumes" = "true" ]; then
                info "停止 Docker 容器并删除数据卷..."
                docker compose down -v
                log "Docker 容器已停止，数据卷已删除"
            else
                info "停止 Docker 容器..."
                docker compose down
                log "Docker 容器已停止"
            fi
            ;;
        *)
            info "保留 Docker 容器（kafka/redis/mariadb 继续运行）"
            info "如需关闭 Docker，执行: docker compose down"
            ;;
    esac
}

# ====================================================================
# Step 4: Cleanup
# ====================================================================
cleanup_logs() {
    if ls "$SCRIPT_DIR/logs/"*.log 2>/dev/null | head -1 >/dev/null; then
        warn "日志文件保留在 logs/ 目录下"
        info "删除日志: rm -rf logs/"
    fi
    if ls "$SCRIPT_DIR/logs/"*.pid 2>/dev/null | head -1 >/dev/null; then
        rm -f "$SCRIPT_DIR/logs/"*.pid
        log "已清理 PID 文件"
    fi
}

# ====================================================================
# Argument parsing
# ====================================================================
DOCKER_DOWN="keep"
DELETE_VOLUMES="false"

for arg in "$@"; do
    case "$arg" in
        --docker-down)
            DOCKER_DOWN="down"
            ;;
        -v|--volumes)
            DELETE_VOLUMES="true"
            ;;
        --help|-h)
            echo "用法: $0 [--docker-down] [-v] [--volumes]"
            echo "  默认:          关闭服务进程 + tmux，保留 Docker（推荐）"
            echo "  --docker-down  同时停止 Docker 容器"
            echo "  -v, --volumes  与 --docker-down 配合，删除数据卷"
            exit 0
            ;;
        *)
            echo "未知参数: $arg"
            echo "用法: $0 [--docker-down] [-v] [--volumes]"
            exit 1
            ;;
    esac
done

# ====================================================================
# Main
# ====================================================================
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  CampusVision AI — 停止所有服务"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

stop_tmux
stop_background
stop_docker "$DOCKER_DOWN" "$DELETE_VOLUMES"
cleanup_logs

echo ""
log "所有服务已关闭"
echo ""
