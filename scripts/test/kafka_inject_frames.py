#!/usr/bin/env python3
"""向 Kafka t_dorm_frame topic 注入测试帧。

用法:
  python scripts/test/kafka_inject_frames.py \\
      --image path/to/test.jpg \\
      --count 20 \\
      --topic t_dorm_frame \\
      --brokers localhost:29092 \\
      --building A \\
      --camera test_cam_001

依赖: kafka-python, opencv-python-headless
"""
import argparse
import base64
import json
import sys
import time

import cv2
import numpy as np
from kafka import KafkaProducer


def encode_frame(image_path: str) -> tuple[str, int, int]:
    """读取图片，编码为 base64 JPEG。返回 (base64_str, width, height)。"""
    img = cv2.imread(image_path, cv2.IMREAD_COLOR)
    if img is None:
        # 尝试以 raw bytes 加载（PNG/其他格式）
        with open(image_path, "rb") as f:
            raw = f.read()
        np_arr = np.frombuffer(raw, dtype=np.uint8)
        img = cv2.imdecode(np_arr, cv2.IMREAD_COLOR)
        if img is None:
            raise ValueError(f"无法解码图片: {image_path}")

    height, width = img.shape[:2]
    _, buffer = cv2.imencode(".jpg", img, [cv2.IMWRITE_JPEG_QUALITY, 95])
    b64 = base64.b64encode(buffer).decode("utf-8")
    return b64, width, height


def main():
    parser = argparse.ArgumentParser(description="向 Kafka 注入测试帧")
    parser.add_argument("--image", required=True, help="测试图片路径")
    parser.add_argument("--count", type=int, default=10, help="发送帧数")
    parser.add_argument("--topic", default="t_dorm_frame", help="Kafka topic")
    parser.add_argument("--brokers", default="localhost:29092", help="Kafka brokers")
    parser.add_argument("--building", default="A", help="楼栋标识")
    parser.add_argument("--camera", default="test_cam_001", help="摄像头 ID")
    parser.add_argument("--interval", type=float, default=0.1, help="帧间隔(秒)")
    parser.add_argument("--key", default=None, help="分区键(默认=building)")
    args = parser.parse_args()

    # 编码图片
    sys.stderr.write(f"加载图片: {args.image}\n")
    frame_data, width, height = encode_frame(args.image)
    sys.stderr.write(
        f"图片尺寸: {width}x{height}, base64: {len(frame_data)} bytes\n"
    )

    # 创建 producer
    producer = KafkaProducer(
        bootstrap_servers=args.brokers,
        value_serializer=lambda v: json.dumps(v).encode("utf-8"),
        key_serializer=lambda k: k.encode("utf-8"),
        acks=1,
    )

    partition_key = args.key or args.building
    for seq in range(args.count):
        msg = {
            "camera_id": args.camera,
            "building": args.building,
            "frame_sequence": seq,
            "frame_data": frame_data,
            "frame_width": width,
            "frame_height": height,
            "timestamp": int(time.time() * 1000),
        }
        future = producer.send(args.topic, value=msg, key=partition_key)
        result = future.get(timeout=10)
        sys.stderr.write(
            f"  seq={seq:3d} offset={result.offset}\n"
        )
        time.sleep(args.interval)

    producer.flush()
    producer.close()
    sys.stderr.write(f"完成: 发送 {args.count} 帧到 {args.topic}\n")


if __name__ == "__main__":
    main()
