#!/usr/bin/env bash
set -euo pipefail

# ====================================================================
# CampusVision AI — stream-gateway: 构建 + 运行 (dev)
# ====================================================================
# 依赖: make infra-up (Kafka, Redis, MariaDB)
# 监听: 8080 (health), 8081 (mgmt)
# 用法:
#   ./stream-gateway/run.sh          # 直接运行
#   CONFIG=myconfig.yaml ./run.sh    # 指定配置文件
# ====================================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

CONFIG="${CONFIG:-config.yaml}"
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
    command -v ffmpeg >/dev/null 2>&1 || { warn "ffmpeg 未安装 (RTSP 解码需要)"; }
    [ -f "$CONFIG" ] || { err "配置文件 $CONFIG 不存在"; exit 1; }
    log "环境检查通过"
}

# ====================================================================
# Main
# ====================================================================
check_prereqs

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  stream-gateway — 启动中"
echo "  配置: $CONFIG"
echo "  端口: 8080 (health) / 8081 (mgmt)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

go run cmd/main.go --config "$CONFIG"
