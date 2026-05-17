#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== CampusVision AI — 测试环境 ==="
echo ""

# 1. Ensure infra is running (Kafka + Redis)
echo "[1/3] Checking infrastructure..."
if docker compose -f "$SCRIPT_DIR/../docker-compose.yml" ps --status running 2>/dev/null | grep -q "cv-kafka"; then
    echo "  ✅ Kafka is running"
else
    echo "  ⚠️  Kafka not running. Starting infrastructure..."
    docker compose -f "$SCRIPT_DIR/../docker-compose.yml" up -d kafka redis
    echo "  ⏳ Waiting for Kafka to be healthy..."
    sleep 8
    echo "  ✅ Kafka started"
fi

# 2. Install deps with uv
echo ""
echo "[2/3] Installing Python dependencies..."
if command -v uv &>/dev/null; then
    uv pip install -q -r requirements.txt
else
    pip install -q -r requirements.txt
fi
echo "  ✅ Dependencies ready"

# 3. Start test server
echo ""
echo "[3/3] Starting test server..."
echo ""
echo "  🌐 Web dashboard : http://localhost:8082/"
echo "  📡 API base      : http://localhost:8082/api/"
echo "  📋 Events        : http://localhost:8082/api/events"
echo "  ❤️  Health        : http://localhost:8082/api/health"
echo ""
echo "  Press Ctrl+C to stop"
echo ""

if command -v uv &>/dev/null; then
    uv run uvicorn server.main:app --host 0.0.0.0 --port 8082 --reload
else
    uvicorn server.main:app --host 0.0.0.0 --port 8082 --reload
fi
