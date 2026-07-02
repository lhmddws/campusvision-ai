#!/usr/bin/env bash
set -euo pipefail

# ====================================================================
# CampusVision AI — frontend: 安装依赖 + 运行 (dev)
# ====================================================================
# 监听: 80 (开发服务器)
# 用法:
#   ./frontend/run.sh                    # 默认 dev 模式
#   PORT=3000 ./run.sh                   # 指定端口
# ====================================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

PORT="${PORT:-80}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log()  { echo -e "${GREEN}[✓]${NC} $1"; }
warn() { echo -e "${YELLOW}[!]${NC} $1"; }
err()  { echo -e "${RED}[✗]${NC} $1"; }
info() { echo -e "${CYAN}[i]${NC} $1"; }

# ====================================================================
# Prerequisites
# ====================================================================
check_prereqs() {
    command -v pnpm >/dev/null 2>&1 || { err "pnpm 未安装 (请先 npm install -g pnpm)"; exit 1; }

    if [ ! -d "node_modules" ]; then
        info "node_modules 不存在，执行 pnpm install..."
        pnpm install
    fi

    log "环境检查通过"
}

# ====================================================================
# Main
# ====================================================================
check_prereqs

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  frontend — 启动开发服务器"
echo "  端口: $PORT"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

pnpm dev --port "$PORT"
