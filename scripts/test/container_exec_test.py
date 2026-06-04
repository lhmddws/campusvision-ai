#!/usr/bin/env python3
"""在 Docker face-recognition 容器内执行检测诊断。

在容器内运行 FaceDetector + NightModeEnhancer，输出检测结果。
用于排查 faces_detected=0 的问题。

用法:
  python scripts/test/container_exec_test.py \\
      --container cv-face-recognition \\
      --config config.yaml
"""
import argparse
import subprocess
import sys
import tempfile

DIAG_SCRIPT = r'''"""Diagnostic: verify face detection inside container."""
import sys, json, base64, os

os.chdir("/app")
sys.path.insert(0, "/app")

import numpy as np
import cv2
from kafka import KafkaConsumer, TopicPartition
from app.detector import FaceDetector
from app.night_mode import NightModeEnhancer
from app.config import load_config

cfg = load_config("CONFIG_PLACEHOLDER")
detector = FaceDetector(
    model_path=cfg.detection.model_path,
    conf_threshold=cfg.detection.confidence_threshold,
    input_size=tuple(cfg.detection.input_size),
    min_face_size=cfg.detection.min_face_size,
    blur_threshold=cfg.detection.blur_threshold,
    nms_iou_threshold=cfg.detection.nms_iou_threshold,
)
enhancer = NightModeEnhancer(cfg.night_mode)

results = []
results.append(f"model_path={cfg.detection.model_path!r}")
results.append(f"min_face_size={detector.min_face_size}")
results.append(f"conf_threshold={detector.conf_threshold}")
results.append(f"session={'ONNX' if detector.session else 'Haar (fallback)'}")

# Read latest frames from Kafka
consumer = KafkaConsumer(
    bootstrap_servers="kafka:9092",
    value_deserializer=lambda m: json.loads(m.decode()),
    consumer_timeout_ms=15000,
    group_id=None,
)
tp = TopicPartition("t_dorm_frame", 0)
consumer.assign([tp])
end = consumer.end_offsets([tp])[tp]
consumer.seek(tp, max(0, end - 5))

count = 0
for msg in consumer:
    v = msg.value
    d = base64.b64decode(v["frame_data"])
    f = cv2.imdecode(np.frombuffer(d, np.uint8), 1)
    if f is None:
        results.append(f"seq={v['frame_sequence']} DECODE FAILED")
    else:
        f2 = enhancer.enhance(f)
        faces = detector.detect(f2)
        results.append(
            f"seq={v['frame_sequence']} shape={f.shape} faces={len(faces)}"
        )
        for face in faces:
            results.append(
                f"  face: ({face.x1:.0f},{face.y1:.0f})-"
                f"({face.x2:.0f},{face.y2:.0f}) "
                f"conf={face.confidence:.3f}"
            )
    count += 1
    if count >= 3:
        break

results.append(f"done (read {count})")
with open("/tmp/diag_result.txt", "w") as out:
    out.write("\n".join(results))
print("OK: results written to /tmp/diag_result.txt")
'''


def main():
    parser = argparse.ArgumentParser(description="容器内检测诊断")
    parser.add_argument(
        "--container", default="cv-face-recognition",
        help="Docker 容器名",
    )
    parser.add_argument(
        "--config", default="config.yaml",
        help="容器内配置路径",
    )
    args = parser.parse_args()

    # 写入诊断脚本（替换配置路径）
    with tempfile.NamedTemporaryFile(
        mode="w", suffix=".py", delete=False, prefix="diag_"
    ) as f:
        f.write(DIAG_SCRIPT.replace("CONFIG_PLACEHOLDER", args.config))
        script_path = f.name

    # 复制到容器
    subprocess.run(
        ["docker", "cp", script_path,
         f"{args.container}:/tmp/container_diag.py"],
        check=True,
    )

    # 执行
    result = subprocess.run(
        ["docker", "compose", "exec", "-T", args.container,
         "python", "/tmp/container_diag.py"],
        capture_output=True, text=True,
    )
    print(result.stdout)
    if result.stderr:
        print(f"STDERR: {result.stderr[:500]}", file=sys.stderr)

    # 读取结果
    result2 = subprocess.run(
        ["docker", "compose", "exec", args.container,
         "cat", "/tmp/diag_result.txt"],
        capture_output=True, text=True,
    )
    if result2.stdout:
        print("\n--- Diagnostic Results ---")
        print(result2.stdout)


if __name__ == "__main__":
    main()
