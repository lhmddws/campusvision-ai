#!/usr/bin/env python3
"""消费 Kafka t_dorm_frame topic 并验证帧格式与人脸检测。

用法:
  python scripts/test/kafka_consume_verify.py \\
      --count 5 \\
      --brokers localhost:29092 \\
      --topic t_dorm_frame

依赖: kafka-python, opencv-python-headless, numpy
"""
import argparse
import base64
import json
import sys

import cv2
import numpy as np
from kafka import KafkaConsumer, TopicPartition


def main():
    parser = argparse.ArgumentParser(description="消费并验证 Kafka 帧")
    parser.add_argument("--count", type=int, default=3, help="验证帧数")
    parser.add_argument("--brokers", default="localhost:29092", help="Kafka brokers")
    parser.add_argument("--topic", default="t_dorm_frame", help="Kafka topic")
    parser.add_argument("--partition", type=int, default=0, help="分区编号")
    parser.add_argument("--seek-latest", type=int, default=5,
                       help="从末尾前移 N 帧")
    parser.add_argument("--display", action="store_true",
                       help="显示检测到的人脸框（需要桌面环境）")
    args = parser.parse_args()

    consumer = KafkaConsumer(
        bootstrap_servers=args.brokers,
        value_deserializer=lambda m: json.loads(m.decode()),
        consumer_timeout_ms=10000,
        group_id=None,  # 无 consumer group，只读不提交
    )

    tp = TopicPartition(args.topic, args.partition)
    consumer.assign([tp])

    end_offset = consumer.end_offsets([tp])[tp]
    seek_to = max(0, end_offset - args.seek_latest)
    consumer.seek(tp, seek_to)
    print(f"Topic: {args.topic}[{args.partition}], "
          f"end_offset={end_offset}, seek_to={seek_to}")

    # Haar Cascade 检测器
    cascade = cv2.CascadeClassifier(
        cv2.data.haarcascades + "haarcascade_frontalface_default.xml"
    )

    count = 0
    for msg in consumer:
        val = msg.value
        is_valid = True

        # 验证必要字段
        required = [
            "camera_id", "building", "frame_sequence",
            "frame_data", "frame_width", "frame_height", "timestamp",
        ]
        missing = [k for k in required if k not in val]
        if missing:
            print(f"  [FAIL] seq={val.get('frame_sequence', '?')} "
                  f"缺少字段: {missing}")
            continue

        # 解码帧
        try:
            frame_bytes = base64.b64decode(val["frame_data"])
            np_arr = np.frombuffer(frame_bytes, dtype=np.uint8)
            frame = cv2.imdecode(np_arr, cv2.IMREAD_COLOR)
            if frame is None:
                print(f"  [FAIL] seq={val['frame_sequence']} 帧解码失败")
                continue
        except Exception as e:
            print(f"  [FAIL] seq={val['frame_sequence']} 解码异常: {e}")
            continue

        # 人脸检测
        gray = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)
        rects = cascade.detectMultiScale(gray, 1.1, 5, minSize=(40, 40))

        actual_h, actual_w = frame.shape[:2]
        declared_w, declared_h = val["frame_width"], val["frame_height"]
        match = "✓" if actual_w == declared_w and actual_h == declared_h else "✗"

        print(f"  seq={val['frame_sequence']:3d} "
              f"size=({actual_w}x{actual_h}) "
              f"declared=({declared_w}x{declared_h}) [{match}] "
              f"faces={len(rects)} camera={val['camera_id']}")

        for x, y, w, h in rects:
            print(f"    face at ({x},{y}) {w}x{h}")

        count += 1
        if count >= args.count:
            break

    consumer.close()
    print(f"\n验证完成: 检查了 {count} 帧")


if __name__ == "__main__":
    main()
