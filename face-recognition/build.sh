#!/usr/bin/env bash
set -euo pipefail

# ====================================================================
# CampusVision AI — face-recognition: 安装依赖 + 下载模型
# ====================================================================
# 用法:
#   ./face-recognition/build.sh         # 完整构建 (依赖 + 模型)
#   SKIP_MODELS=1 ./build.sh            # 仅安装依赖，跳过模型
# ====================================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

VENV_DIR=".venv"

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
# 获取 venv Python (优先 .venv，回退系统 python3)
# ====================================================================
get_python() {
    if [ -f "$VENV_DIR/bin/python" ]; then
        echo "$VENV_DIR/bin/python"
    elif command -v python3 >/dev/null 2>&1; then
        echo "$(command -v python3)"
    else
        err "python3 未安装"
        exit 1
    fi
}

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  face-recognition — 构建中"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

PYTHON="$(get_python)"

# Step 1: 安装依赖
info "安装 Python 依赖 ($PYTHON)..."
if command -v uv >/dev/null 2>&1; then
    uv sync
    log "uv sync 完成"
else
    pip install -r requirements.txt
    log "pip install 完成"
fi

# Step 2: 下载 ONNX 模型
if [ "${SKIP_MODELS:-0}" != "1" ]; then
    info "下载 ONNX 模型..."
    $PYTHON -m app.download_models
    log "模型下载完成"
else
    warn "SKIP_MODELS=1，跳过模型下载"
fi

echo ""
log "face-recognition 构建完成"
