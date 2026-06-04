#!/usr/bin/env python3
"""使用 Mediamtx + FFmpeg 模拟 RTSP 摄像头推送流。

依赖:
  - mediamtx 容器正在运行（docker compose up -d mediamtx）
  - ffmpeg 在 $PATH 中
  - 测试图片（如 face-recognition/tests/fixtures/face_test.png）

用法:
  # 推送单张图片为循环视频流
  python scripts/rtsp/simulate_camera.py \\
      --image face-recognition/tests/fixtures/face_test.png \\
      --rtsp-url rtsp://localhost:8554/cam_a \\
      --fps 5

  # 推送视频文件
  python scripts/rtsp/simulate_camera.py \\
      --video /path/to/test_video.mp4 \\
      --rtsp-url rtsp://localhost:8554/cam_b

  # 推送多摄像头
  python scripts/rtsp/simulate_camera.py --multi 4 \\
      --image face-recognition/tests/fixtures/face_test.png
"""
import argparse
import subprocess
import sys
import shutil


def find_ffmpeg() -> str:
    ffmpeg = shutil.which("ffmpeg")
    if not ffmpeg:
        raise RuntimeError(
            "ffmpeg 未安装。请安装 ffmpeg 并确保在 $PATH 中。\n"
            "  Windows: choco install ffmpeg 或 winget install ffmpeg\n"
            "  macOS: brew install ffmpeg\n"
            "  Linux: apt install ffmpeg"
        )
    return ffmpeg


def publish_image(ffmpeg: str, image: str, rtsp_url: str, fps: int):
    """使用图片循环推送 RTSP 流。"""
    cmd = [
        ffmpeg,
        "-re",
        "-loop", "1",
        "-i", image,
        "-c:v", "libx264",
        "-preset", "ultrafast",
        "-tune", "stillimage",
        "-r", str(fps),
        "-pix_fmt", "yuv420p",
        "-b:v", "500k",
        "-maxrate", "500k",
        "-bufsize", "1000k",
        "-f", "rtsp",
        "-rtsp_transport", "tcp",
        rtsp_url,
    ]
    sys.stderr.write(
        f"推送图片流: {image} → {rtsp_url} @ {fps}fps\n"
    )
    subprocess.run(cmd)


def publish_video(ffmpeg: str, video: str, rtsp_url: str):
    """推送视频文件为 RTSP 流。"""
    cmd = [
        ffmpeg,
        "-re",
        "-i", video,
        "-c:v", "libx264",
        "-preset", "ultrafast",
        "-pix_fmt", "yuv420p",
        "-b:v", "1000k",
        "-maxrate", "1000k",
        "-bufsize", "2000k",
        "-f", "rtsp",
        "-rtsp_transport", "tcp",
        rtsp_url,
    ]
    sys.stderr.write(f"推送视频: {video} → {rtsp_url}\n")
    subprocess.run(cmd)


def main():
    parser = argparse.ArgumentParser(description="RTSP 摄像头模拟")
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--image", help="测试图片路径（循环推送）")
    group.add_argument("--video", help="测试视频路径")
    group.add_argument(
        "--multi", type=int, metavar="N",
        help="同时启动 N 个模拟摄像头（自动分配 cam_a, cam_b, ...）",
    )

    parser.add_argument(
        "--rtsp-url", default="rtsp://localhost:8554/cam_a",
        help="RTSP 推送地址",
    )
    parser.add_argument("--fps", type=int, default=5, help="帧率")
    parser.add_argument(
        "--base-port", type=int, default=8554,
        help="多摄像头基础端口",
    )

    args = parser.parse_args()
    ffmpeg = find_ffmpeg()

    if args.multi:
        if not args.image:
            print("ERROR: --multi 需要 --image 参数", file=sys.stderr)
            sys.exit(1)
        cameras = [chr(ord("a") + i) for i in range(args.multi)]
        for cam in cameras:
            process = subprocess.Popen([
                sys.executable, __file__,
                "--image", args.image,
                "--rtsp-url", f"rtsp://localhost:{args.base_port}/cam_{cam}",
                "--fps", str(args.fps),
            ])
            sys.stderr.write(
                f"摄像头 cam_{cam}: PID {process.pid}\n"
            )
    elif args.image:
        publish_image(ffmpeg, args.image, args.rtsp_url, args.fps)
    elif args.video:
        publish_video(ffmpeg, args.video, args.rtsp_url)


if __name__ == "__main__":
    main()
