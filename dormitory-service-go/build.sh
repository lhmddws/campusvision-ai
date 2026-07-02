#!/usr/bin/env bash
set -euo pipefail

# ====================================================================
# CampusVision AI — dormitory-service-go: 生产构建
# ====================================================================
# 用法:
#   ./dormitory-service-go/build.sh            # 构建到 bin/dormitory-service
#   OUTPUT=./my-custom-path ./build.sh         # 指定输出路径
# ====================================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

BIN_DIR="bin"
OUTPUT="${OUTPUT:-$BIN_DIR/dormitory-service}"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

log()  { echo -e "${GREEN}[✓]${NC} $1"; }
err()  { echo -e "${RED}[✗]${NC} $1"; }

# ====================================================================
# Build
# ====================================================================
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  dormitory-service-go — 构建中"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

mkdir -p "$(dirname "$OUTPUT")"

go build -o "$OUTPUT" ./cmd/dormitory-service/

log "构建成功: $OUTPUT"
