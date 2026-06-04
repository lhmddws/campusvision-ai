#!/bin/bash
# CampusVision AI — 健康检查
# 检查所有 Docker 容器、Kafka Topics、Redis、MariaDB 状态

set -euo pipefail

echo "=== 容器状态 ==="
docker compose ps --format "table {{.Name}}\t{{.Status}}"

echo ""
echo "=== Kafka Topics ==="
docker compose exec kafka kafka-topics \
    --bootstrap-server localhost:9092 --list 2>/dev/null \
    || echo "Kafka 不可用"

echo ""
echo "=== Redis ==="
docker compose exec redis redis-cli ping 2>/dev/null \
    || echo "Redis 不可用"

echo ""
echo "=== MariaDB ==="
docker compose exec mariadb mysqladmin ping \
    -u sims -psims 2>/dev/null \
    || echo "MariaDB 不可用"

echo ""
echo "=== 端口监听 ==="
for port in 8080 8081 8083 9092 6379 3306 8554; do
    case $port in
        8080) desc="stream-gateway health" ;;
        8081) desc="stream-gateway mgmt" ;;
        8083) desc="dormitory-service" ;;
        9092) desc="Kafka" ;;
        6379) desc="Redis" ;;
        3306) desc="MariaDB" ;;
        8554) desc="Mediamtx RTSP" ;;
    esac
    if netstat -an 2>/dev/null | grep -q ":$port "; then
        echo "  ✅ $port ($desc) — listening"
    else
        echo "  ❌ $port ($desc) — not listening"
    fi
done
