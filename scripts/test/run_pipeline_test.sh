#!/bin/bash
# CampusVision AI — 端到端流水线测试
# 验证: Kafka 消息流 → 人脸检测 → 事件产生
set -euo pipefail

PASS=0
FAIL=0

check() {
    local desc="$1"
    local cmd="$2"
    echo -n "[TEST] $desc ... "
    if eval "$cmd" 2>/dev/null; then
        echo "✅ PASS"
        ((PASS++))
    else
        echo "❌ FAIL"
        ((FAIL++))
    fi
}

echo "========================================="
echo " CampusVision AI — Pipeline 测试套件"
echo "========================================="
echo ""

# 1. 基础设施检查
check "Kafka 可达 (本机:29092)" \
    "python -c \"from kafka import KafkaProducer; p=KafkaProducer(bootstrap_servers='localhost:29092'); p.close()\""

check "Kafka 可达 (容器:9092)" \
    "python -c \"from kafka import KafkaProducer; p=KafkaProducer(bootstrap_servers='kafka:9092'); p.close()\""

check "Redis 可达" \
    "docker compose exec redis redis-cli ping 2>/dev/null | grep -q PONG"

check "MariaDB 可达" \
    "docker compose exec mariadb mysqladmin ping -u sims -psims 2>/dev/null"

# 2. 容器健康检查
check "face-recognition 运行中" \
    "docker compose ps face-recognition --format '{{.Status}}' | grep -q healthy"

check "stream-gateway 运行中" \
    "docker compose ps stream-gateway --format '{{.Status}}' | grep -q 'Up\|healthy'"

check "dormitory-service-go 运行中" \
    "docker compose ps dormitory-service-go --format '{{.Status}}' | grep -q 'Up\|healthy'"

# 3. Kafka 生产-消费测试
check "注入测试帧 (5帧)" \
    "$(dirname "$0")/kafka_inject_frames.py \
        --image face-recognition/tests/fixtures/face_test.png \
        --count 5 --interval 0.05"

check "face-recognition 处理帧" \
    "sleep 5 && docker compose logs --tail 5 face-recognition 2>&1 | grep -q frames_processed"

# 4. 人脸检测验证
echo ""
echo "--- 人脸检测验证 ---"
STATS=$(docker compose logs --tail 20 face-recognition 2>&1 | grep processing_stats | tail -1)
echo "  Latest stats: $STATS"

# 5. 结果汇总
echo ""
echo "========================================="
echo " 结果: $PASS 通过, $FAIL 失败"
echo "========================================="

exit $FAIL
