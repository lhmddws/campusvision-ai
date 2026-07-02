#!/usr/bin/env bash
set -euo pipefail

# ====================================================================
# CampusVision AI — face-recognition: 构建依赖 + 运行 (dev)
# ====================================================================
# 依赖: make infra-up (Kafka, Redis), stream-gateway (生产者)
# 用法:
#   ./face-recognition/run.sh              # 默认配置
#   CONFIG=myconfig.yaml ./run.sh          # 指定配置
#   SKIP_MODELS=1 ./run.sh                 # 跳过模型下载
# ====================================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

CONFIG="${CONFIG:-config.yaml}"
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

# ====================================================================
# Prerequisites
# ====================================================================
check_prereqs() {
    local py
    py="$(get_python)"
    [ -f "$CONFIG" ] || { err "配置文件 $CONFIG 不存在"; exit 1; }

    if [ ! -d "$VENV_DIR" ]; then
        warn ".venv 不存在，尝试安装依赖..."
        if command -v uv >/dev/null 2>&1; then
            uv sync
        else
            pip install -r requirements.txt
        fi
    fi

    if [ ! -f "app/models/retinaface-R50.onnx" ] || [ ! -f "app/models/arcface-resnet100.onnx" ]; then
        if [ "${SKIP_MODELS:-0}" != "1" ]; then
            info "ONNX 模型未下载，自动下载..."
            $py -m app.download_models || warn "模型下载失败，将降级为 Haar Cascade + 零向量回退"
        else
            warn "SKIP_MODELS=1，跳过模型下载 (将降级为 Haar Cascade)"
        fi
    fi

    log "环境检查通过"
}

# ====================================================================
# Main
# ====================================================================
check_prereqs

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  face-recognition — 启动中"
echo "  配置: $CONFIG"
echo "  模式: Kafka 消费者"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

PYTHON="$(get_python)"

$PYTHON -m app.main --config "$CONFIG"
