"""Generate test Kafka message and produce via docker kafka-console-producer."""

import base64
import json
import os
import subprocess
import sys
import time

import cv2


def main():
    test_img = os.path.join(os.path.dirname(os.path.abspath(__file__)), "test_face.jpg")
    if not os.path.exists(test_img):
        print(f"ERROR: {test_img} not found")
        return 1

    img = cv2.imread(test_img)
    _, jpeg = cv2.imencode(".jpg", img, [cv2.IMWRITE_JPEG_QUALITY, 95])
    frame_data = base64.b64encode(jpeg.tobytes()).decode()

    msg = {
        "frame_data": frame_data,
        "camera_id": "test-cam-01",
        "building": "A",
        "frame_width": img.shape[1],
        "frame_height": img.shape[0],
        "frame_sequence": 1,
        "timestamp": int(time.time() * 1000),
    }
    msg_json = json.dumps(msg)

    print(f"=== Publishing test frame ===")
    print(f"  Image: 480x640, JPEG: {len(jpeg)} bytes")
    print(f"  Kafka: localhost:9092 -> t_dorm_frame")
    print(f"  camera_id: test-cam-01, building: A, seq: 1")
    print(f"  Message size: {len(msg_json)} bytes")
    print()

    # Produce via docker kafka-console-producer
    cmd = [
        "docker", "exec", "-i", "cv-kafka",
        "kafka-console-producer",
        "--bootstrap-server", "localhost:9092",
        "--topic", "t_dorm_frame",
    ]
    print(f"Running: {' '.join(cmd)}")
    proc = subprocess.run(
        cmd,
        input=msg_json,
        capture_output=True,
        text=True,
        timeout=15,
    )
    if proc.returncode != 0:
        print(f"ERROR (rc={proc.returncode}): {proc.stderr.strip()}")
        return 1
    if proc.stderr:
        print(f"stderr: {proc.stderr.strip()}")
    print("Done!")
    return 0


if __name__ == "__main__":
    sys.exit(main())
