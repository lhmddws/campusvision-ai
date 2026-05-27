#!/usr/bin/env bash
set -euo pipefail

# ────────────────────────────────────────────────────────
# CampusVision - Model Downloader (Linux/macOS/Docker)
# ────────────────────────────────────────────────────────
# Downloads ONNX models with mirror fallback and retry.
#
# Usage:
#   ./scripts/download-models.sh
#   ./scripts/download-models.sh --mirror https://hf-mirror.com
#   ./scripts/download-models.sh --proxy http://127.0.0.1:7890
#   HF_ENDPOINT=https://hf-mirror.com ./scripts/download-models.sh
# ────────────────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

MIRROR="${MIRROR:-${HF_ENDPOINT:-https://hf-mirror.com}}"
RETRIES="${RETRIES:-3}"
TIMEOUT="${TIMEOUT:-300}"
PROXY="${PROXY:-}"

echo "═══════════════════════════════════════════════"
echo "  CampusVision - Model Downloader"
echo "═══════════════════════════════════════════════"
echo "  Mirror:   $MIRROR"
echo "  Retries:  $RETRIES"
echo "  Timeout:  ${TIMEOUT}s"
[ -n "$PROXY" ] && echo "  Proxy:    $PROXY"
echo ""

ARGS=()
ARGS+=("--mirror" "$MIRROR")
ARGS+=("--retries" "$RETRIES")
ARGS+=("--timeout" "$TIMEOUT")
[ -n "$PROXY" ] && ARGS+=("--proxy" "$PROXY")

python -m app.download_models "${ARGS[@]}"

EXIT_CODE=$?
if [ $EXIT_CODE -eq 0 ]; then
    echo ""
    echo "✅ Models downloaded successfully!"
    echo "   Location: $PROJECT_ROOT/app/models/"
    ls -lh "$PROJECT_ROOT/app/models/"*.onnx 2>/dev/null
else
    echo ""
    echo "❌ Download failed. Options:"
    echo "  1. With proxy:   PROXY=http://127.0.0.1:7890 ./scripts/download-models.sh"
    echo "  2. Direct:       MIRROR=https://huggingface.co ./scripts/download-models.sh"
    echo "  3. Manual:       Download *.onnx files and place in app/models/"
fi

exit $EXIT_CODE
