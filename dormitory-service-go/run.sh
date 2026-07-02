#!/usr/bin/env bash
set -euo pipefail

# ====================================================================
# CampusVision AI — dormitory-service-go: 构建 + 运行 (dev)
# ====================================================================
# 依赖: make infra-up (Kafka, MariaDB, Redis)
# 监听: 8083 (API)
# 用法:
#   ./dormitory-service-go/run.sh            # 默认配置
#   CONFIG_PATH=myconfig.yaml ./run.sh       # 指定配置
# ====================================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

CONFIG_PATH="${CONFIG_PATH:-config.yaml}"
BIN_DIR="bin"

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
    command -v go >/dev/null 2>&1 || { err "go 未安装"; exit 1; }
    [ -f "$CONFIG_PATH" ] || { err "配置文件 $CONFIG_PATH 不存在"; exit 1; }
    log "环境检查通过"
}

# ====================================================================
# Main
# ====================================================================
check_prereqs

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  dormitory-service-go — 启动中"
echo "  配置: $CONFIG_PATH"
echo "  端口: 8083 (API)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

CONFIG_PATH="$CONFIG_PATH" go run ./cmd/dormitory-service/
