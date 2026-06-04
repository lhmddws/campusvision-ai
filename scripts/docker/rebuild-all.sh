#!/bin/bash
# CampusVision AI — 一键重建所有服务
# 用法: bash scripts/docker/rebuild-all.sh [--with-models] [--no-cache]

set -euo pipefail

WITH_MODELS=false
NO_CACHE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --with-models) WITH_MODELS=true ;;
        --no-cache) NO_CACHE="--no-cache" ;;
        *) echo "未知参数: $1"; exit 1 ;;
    esac
    shift
done

BUILD_ARGS=""
if [ -n "${APT_MIRROR:-}" ]; then
    BUILD_ARGS="$BUILD_ARGS --build-arg APT_MIRROR=$APT_MIRROR"
fi
if [ -n "${PIP_INDEX_URL:-}" ]; then
    BUILD_ARGS="$BUILD_ARGS --build-arg PIP_INDEX_URL=$PIP_INDEX_URL"
fi
if [ -n "${HF_ENDPOINT:-}" ]; then
    BUILD_ARGS="$BUILD_ARGS --build-arg HF_ENDPOINT=$HF_ENDPOINT"
fi

if [ "$WITH_MODELS" = true ]; then
    BUILD_ARGS="$BUILD_ARGS --build-arg BUILD_MODELS=1"
fi

echo "=== 构建 stream-gateway ==="
docker compose build $NO_CACHE $BUILD_ARGS stream-gateway

echo "=== 构建 face-recognition ==="
docker compose build $NO_CACHE $BUILD_ARGS face-recognition

echo "=== 构建 dormitory-service-go ==="
docker compose build $NO_CACHE $BUILD_ARGS dormitory-service-go

echo "=== 构建 frontend ==="
docker compose build $NO_CACHE $BUILD_ARGS frontend

echo "=== 完成 ==="
echo "全部服务构建完成。使用 'docker compose up -d' 启动。"
