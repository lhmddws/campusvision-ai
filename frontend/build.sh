#!/usr/bin/env bash
set -euo pipefail

# ====================================================================
# CampusVision AI — frontend: 生产构建
# ====================================================================
# 输出: dist/
# 用法:
#   ./frontend/build.sh          # 构建到 dist/
#   OUTPUT=./my-deploy ./build.sh # 指定输出目录
# ====================================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

OUTPUT="${OUTPUT:-dist}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log()  { echo -e "${GREEN}[✓]${NC} $1"; }
err()  { echo -e "${RED}[✗]${NC} $1"; }
info() { echo -e "${CYAN}[i]${NC} $1"; }

# ====================================================================
# Prerequisites
# ====================================================================
check_prereqs() {
    command -v pnpm >/dev/null 2>&1 || { err "pnpm 未安装"; exit 1; }

    if [ ! -d "node_modules" ]; then
        info "node_modules 不存在，执行 pnpm install..."
        pnpm install
    fi

    log "环境检查通过"
}

# ====================================================================
# Build
# ====================================================================
check_prereqs

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  frontend — 生产构建中"
echo "  输出: $OUTPUT/"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

pnpm build:prod

log "构建成功: $OUTPUT/"
