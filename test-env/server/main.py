"""
CampusVision AI — 测试环境服务器

提供 Web 测试面板，支持：
- 模拟 4 路摄像头画面（Pillow/SVG 生成）
- 本机摄像头实时画面（OpenCV）
- 所有可调参数（灵敏度、阈值、分辨率等）在线配置
- 模拟事件注入 Kafka
启动: uvicorn server.main:app --host 0.0.0.0 --port 8082
"""

import csv
import io
import json
import os
import shutil
import subprocess
import time
import uuid
import base64
import logging
import threading
from collections import deque
from datetime import datetime
from pathlib import Path
from typing import Optional

import uvicorn
from fastapi import FastAPI, File, HTTPException, UploadFile
from fastapi.responses import Response
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel

logger = logging.getLogger("test-env")

# ── Config ──────────────────────────────────────────────────────────────────

KAFKA_BROKERS = os.getenv("KAFKA_BROKERS", "localhost:9092")
FRAME_TOPIC = os.getenv("FRAME_TOPIC", "t_dorm_frame")
EVENT_TOPIC = os.getenv("EVENT_TOPIC", "t_dorm_event")
SERVER_PORT = int(os.getenv("TEST_SERVER_PORT", "8082"))
STREAM_GW_HEALTH = os.getenv("STREAM_GW_HEALTH", "http://localhost:8080/health")

# ── Default camera definitions ──────────────────────────────────────────────

CAMERAS = {
    "cam-a": {"building": "A", "label": "A栋入口", "color": "#2980b9"},
    "cam-b": {"building": "B", "label": "B栋入口", "color": "#27ae60"},
    "cam-c": {"building": "C", "label": "C栋入口", "color": "#8e44ad"},
    "cam-d": {"building": "D", "label": "D栋入口", "color": "#e67e22"},
}

# ── Global config (all adjustable parameters) ────────────────────────────────

SERVER_CONFIG = {
    # Frame settings
    "jpeg_quality": 80,
    "frame_width": 640,
    "frame_height": 360,
    "fps": 5,

    # Detection settings (face-recognition)
    "confidence_threshold": 0.6,
    "min_face_size": 80,

    # Match settings
    "match_threshold": 0.65,
    "cache_ttl": 3600,

    # Direction settings
    "roi_line_x": 0.5,
    "min_track_points": 3,

    # Dedup settings
    "dedup_window_seconds": 10,

    # Stranger settings
    "stranger_alert_enabled": True,
    "stranger_alert_threshold": 0.45,

    # Night mode
    "night_mode_enabled": True,
    "night_mode_start_hour": 22,
    "night_mode_end_hour": 6,

    # Motion / dynamic extraction
    "motion_threshold": 0.05,
    "dynamic_extraction": True,

    # Camera source: "simulated" | "webcam"
    "camera_source": "simulated",
    "webcam_device": 0,

    # Test simulation
    "test_people": [
        "张三 (2024001)", "李四 (2024002)", "王五 (2024003)",
        "赵六 (2024004)", "孙七 (2024005)", "周八 (2024006)",
    ],
}

# Per-camera webcam device index override (default: use global webcam_device)
CAMERA_WEBCAM_INDEX: dict[str, Optional[int]] = {
    cid: None for cid in CAMERAS
}

# ── Kafka ────────────────────────────────────────────────────────────────────

producer = None
try:
    from kafka import KafkaProducer as _KP
    producer = _KP(
        bootstrap_servers=KAFKA_BROKERS,
        value_serializer=lambda v: json.dumps(v).encode(),
    )
    logger.info("Kafka producer ready %s", KAFKA_BROKERS)
except Exception as e:
    logger.warning("Kafka unavailable: %s", e)

# ── OpenCV availability ─────────────────────────────────────────────────────

try:
    import cv2
    CV2_OK = True
except ImportError:
    CV2_OK = False
    logger.warning("OpenCV not available, webcam mode disabled")

# ── Webcam capture via ffmpeg subprocess ────────────────────────────────────
# Why ffmpeg? macOS AVFoundation + OpenCV VideoCapture crashes after
# uvicorn's fork()-based worker spawning because AVFoundation initialisation
# cannot safely cross a fork boundary.  ffmpeg runs as a separate process and
# pipes MJPEG frames back to us — no fork- or thread-safety issues.

_webcam_captures: dict = {}
_FFMPEG_Q = 5  # default, overridden from config


def _webcam_capture_worker(camera_id: str, device_idx: int):
    """Background thread: captures webcam via ffmpeg → MJPEG pipe."""
    proc = None
    try:
        w = SERVER_CONFIG["frame_width"]
        h = SERVER_CONFIG["frame_height"]
        fps = max(1, SERVER_CONFIG["fps"])
        # ffmpeg mjpeg q:v 2–31 (lower = better), map jpeg_quality 10–100 → 2–25
        q_raw = SERVER_CONFIG["jpeg_quality"]
        q = max(2, min(25, int(27 - q_raw * 0.25)))

        # macOS AVFoundation input must use a framerate the device supports
        # (usually 15-30 fps).  We capture at 30 fps then decimate via fps filter.
        input_fps = 30
        cmd = [
            "ffmpeg", "-loglevel", "error",
            "-f", "avfoundation",
            "-video_device_index", str(device_idx),
            "-video_size", "640x480",  # native mode supported by FaceTime HD
            "-r", str(input_fps),
            "-i", "",
            "-vf", f"scale={w}:{h},fps={fps}",
            "-f", "mjpeg",
            "-q:v", str(q),
            "-",
        ]

        _webcam_captures[camera_id] = {"running": True, "frame": None}
        proc = subprocess.Popen(
            cmd, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL,
            bufsize=0,
        )
        logger.info("Webcam %s: started ffmpeg PID %d", camera_id, proc.pid)

        buf = bytearray()
        SOI = b"\xff\xd8"
        EOI = b"\xff\xd9"

        running = True
        while running and proc.poll() is None:
            chunk = proc.stdout.read(65536)
            if not chunk:
                time.sleep(0.01)
                continue
            buf.extend(chunk)

            while True:
                start = buf.find(SOI)
                if start == -1:
                    buf.clear()
                    break
                end = buf.find(EOI, start + 2)
                if end == -1:  # incomplete; keep buffer
                    # Trim leading bytes before a partial SOI
                    if start > 0 and buf[:start].count(SOI) == 0:
                        buf = bytearray(buf[start:])
                    break
                jpeg = bytes(buf[start: end + 2])
                del buf[: end + 2]
                latest_frames[camera_id] = jpeg

            running = _webcam_captures.get(camera_id, {}).get("running", False)

    except Exception as e:
        logger.error("Webcam %s error: %s", camera_id, e)
    finally:
        if proc is not None:
            proc.kill()
            try:
                proc.wait(timeout=3)
            except subprocess.TimeoutExpired:
                pass
        _webcam_captures.pop(camera_id, None)
        logger.info("Webcam %s: stopped", camera_id)


# ── Pillow frame generation (simulated) ─────────────────────────────────────

latest_frames: dict[str, bytes] = {}

try:
    from PIL import Image, ImageDraw
    PIL_OK = True
except ImportError:
    PIL_OK = False


def _make_frame(camera_id: str, building: str, action: str, person: Optional[str]) -> bytes:
    """Generate a test frame as JPEG bytes."""
    if PIL_OK:
        return _pillow_frame(camera_id, building, action, person)
    return _svg_frame(camera_id, building, action, person)


def _pillow_frame(camera_id: str, building: str, action: str, person: Optional[str]) -> bytes:
    W = SERVER_CONFIG["frame_width"]
    H = SERVER_CONFIG["frame_height"]
    color_hex = CAMERAS[camera_id]["color"]
    rgb = tuple(int(color_hex.lstrip("#")[i: i + 2], 16) for i in (0, 2, 4))

    img = Image.new("RGB", (W, H), rgb)
    drw = ImageDraw.Draw(img)

    # Bottom gradient bar
    for i in range(60):
        drw.rectangle([0, H - 60 + i, W, H - 60 + i + 1], fill=(0, 0, 0))

    # Timestamp
    ts = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    drw.text((12, 8), ts, fill=(255, 255, 255))

    # Camera label
    label = CAMERAS[camera_id]["label"]
    drw.text((12, 28), f"{label} [{camera_id}]", fill=(200, 230, 255))

    # Door frame
    door_x, door_y, door_w, door_h = W // 2 - 50, H // 4, 100, H // 2
    drw.rectangle([door_x, door_y, door_x + door_w, door_y + door_h],
                  outline=(255, 255, 255, 120), width=2)

    # Action indicator
    if action == "entry":
        drw.ellipse([door_x + 30, door_y + 20, door_x + 70, door_y + 60],
                    fill="#2ecc71", outline="#27ae60")
        drw.rectangle([door_x + 40, door_y + 60, door_x + 60, door_y + 120],
                      fill="#2ecc71", outline="#27ae60")
        drw.text((door_x - 20, door_y + door_h + 20), "→  进入", fill="#2ecc71")
    elif action == "exit":
        drw.ellipse([door_x - 10, door_y + 20, door_x + 30, door_y + 60],
                    fill="#e74c3c", outline="#c0392b")
        drw.rectangle([door_x, door_y + 60, door_x + 20, door_y + 120],
                      fill="#e74c3c", outline="#c0392b")
        drw.text((door_x - 20, door_y + door_h + 20), "←  离开", fill="#e74c3c")
    else:
        drw.text((door_x, door_y + door_h + 20), "● 无人", fill="#95a5a6")

    if person:
        drw.text((12, H - 48), f"Person: {person}", fill="#ecf0f1")
        if action != "idle":
            drw.text((12, H - 28), f"Action: {action.upper()}", fill="#f1c40f")

    quality = SERVER_CONFIG["jpeg_quality"]
    buf = io.BytesIO()
    img.save(buf, format="JPEG", quality=quality)
    return buf.getvalue()


def _svg_frame(camera_id: str, building: str, action: str, person: Optional[str]) -> bytes:
    c = CAMERAS[camera_id]
    W = SERVER_CONFIG["frame_width"]
    H = SERVER_CONFIG["frame_height"]
    ts = datetime.now().strftime("%H:%M:%S")
    svg = f"""<svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{H}">
<rect width="{W}" height="{H}" fill="{c["color"]}"/>
<text x="12" y="20" fill="white" font-family="monospace" font-size="14">{ts}</text>
<text x="12" y="40" fill="#c8e6ff" font-family="monospace" font-size="14">{c["label"]} [{camera_id}]</text>
<rect x="{W//2-50}" y="{H//4}" width="100" height="{H//2}" fill="none" stroke="white" stroke-width="2"/>
<text x="{W//2-60}" y="{H//4+H//2+30}" fill="{'#2ecc71' if action=='entry' else '#e74c3c' if action=='exit' else '#95a5a6'}" font-family="monospace">
{'→  进入' if action=='entry' else '←  离开' if action=='exit' else '● 无人'}
</text>
</svg>"""
    return svg.encode()


# ── In-memory event log ─────────────────────────────────────────────────────

event_log: deque = deque(maxlen=300)
start_time = time.time()


def _log(cam_id: str, ev_type: str, detail: str) -> dict:
    building = CAMERAS[cam_id]["building"] if cam_id in CAMERAS else cam_id
    entry = dict(
        time=datetime.now().strftime("%H:%M:%S"),
        camera_id=cam_id,
        building=building,
        event_type=ev_type,
        detail=detail,
    )
    event_log.appendleft(entry)
    return entry


# ── FastAPI app ──────────────────────────────────────────────────────────────

app = FastAPI(title="CampusVision Test Env", version="2.0")


class SimulateBody(BaseModel):
    action: str = "entry"  # entry | exit | idle
    person: Optional[str] = None


class ConfigUpdateBody(BaseModel):
    # Allow partial updates - only specified fields are updated
    jpeg_quality: Optional[int] = None
    frame_width: Optional[int] = None
    frame_height: Optional[int] = None
    fps: Optional[int] = None
    confidence_threshold: Optional[float] = None
    min_face_size: Optional[int] = None
    match_threshold: Optional[float] = None
    cache_ttl: Optional[int] = None
    roi_line_x: Optional[float] = None
    min_track_points: Optional[int] = None
    dedup_window_seconds: Optional[int] = None
    stranger_alert_enabled: Optional[bool] = None
    stranger_alert_threshold: Optional[float] = None
    night_mode_enabled: Optional[bool] = None
    night_mode_start_hour: Optional[int] = None
    night_mode_end_hour: Optional[int] = None
    motion_threshold: Optional[float] = None
    dynamic_extraction: Optional[bool] = None
    camera_source: Optional[str] = None
    webcam_device: Optional[int] = None
    test_people: Optional[list[str]] = None


class WebcamControlBody(BaseModel):
    camera_id: str
    action: str = "start"  # start | stop
    device_index: Optional[int] = None


class AddPersonBody(BaseModel):
    name: str


class CameraDefBody(BaseModel):
    building: str
    label: str
    color: Optional[str] = None


# ── Health ──────────────────────────────────────────────────────────────────


@app.get("/api/cameras")
async def api_cameras():
    """Return all camera definitions."""
    return CAMERAS


@app.put("/api/cameras/{camera_id}")
async def upsert_camera(camera_id: str, body: CameraDefBody):
    """Add or update a camera definition."""
    CAMERAS[camera_id] = {
        "building": body.building,
        "label": body.label,
        "color": body.color or "#555577",
    }
    if camera_id not in CAMERA_WEBCAM_INDEX:
        CAMERA_WEBCAM_INDEX[camera_id] = None
    logger.info("Camera upserted: %s -> %s", camera_id, CAMERAS[camera_id])
    return dict(success=True, camera_id=camera_id, config=CAMERAS[camera_id])


@app.delete("/api/cameras/{camera_id}")
async def remove_camera(camera_id: str):
    """Remove a camera definition."""
    if camera_id not in CAMERAS:
        raise HTTPException(404, f"Unknown camera: {camera_id}")
    if camera_id in _webcam_captures:
        _webcam_captures[camera_id]["running"] = False
    CAMERAS.pop(camera_id, None)
    CAMERA_WEBCAM_INDEX.pop(camera_id, None)
    latest_frames.pop(camera_id, None)
    logger.info("Camera deleted: %s", camera_id)
    return dict(success=True, camera_id=camera_id)


@app.get("/api/health")
async def api_health():
    return dict(
        status="ok",
        kafka=producer is not None,
        pillow=PIL_OK,
        opencv=CV2_OK,
        cameras=CAMERAS,
        uptime_sec=int(time.time() - start_time),
        config=SERVER_CONFIG,
        webcams={cid: cid in _webcam_captures for cid in CAMERAS},
    )

# ── Config endpoints ─────────────────────────────────────────────────────────


@app.get("/api/config")
async def get_config():
    """Return the full server configuration."""
    return dict(
        config=SERVER_CONFIG,
        cameras=CAMERAS,
        webcam_status={cid: cid in _webcam_captures for cid in CAMERAS},
        camera_webcam_indices=CAMERA_WEBCAM_INDEX,
    )


@app.put("/api/config")
async def update_config(body: ConfigUpdateBody):
    """Update server configuration (partial update)."""
    updates = {}
    for field, value in body.model_dump(exclude_none=True).items():
        if field in SERVER_CONFIG:
            SERVER_CONFIG[field] = value
            updates[field] = value

    logger.info("Config updated: %s", updates)
    return dict(success=True, updated=updates, config=SERVER_CONFIG)


@app.put("/api/config/reset")
async def reset_config():
    """Reset config to defaults."""
    # Re-apply defaults (except test_people)
    default_people = SERVER_CONFIG["test_people"]
    SERVER_CONFIG.clear()
    SERVER_CONFIG.update({
        "jpeg_quality": 80,
        "frame_width": 640,
        "frame_height": 360,
        "fps": 5,
        "confidence_threshold": 0.6,
        "min_face_size": 80,
        "match_threshold": 0.65,
        "cache_ttl": 3600,
        "roi_line_x": 0.5,
        "min_track_points": 3,
        "dedup_window_seconds": 10,
        "stranger_alert_enabled": True,
        "stranger_alert_threshold": 0.45,
        "night_mode_enabled": True,
        "night_mode_start_hour": 22,
        "night_mode_end_hour": 6,
        "motion_threshold": 0.05,
        "dynamic_extraction": True,
        "camera_source": "simulated",
        "webcam_device": 0,
        "test_people": default_people,
    })
    return dict(success=True, config=SERVER_CONFIG)


# ── People management ───────────────────────────────────────────────────────


@app.get("/api/people")
async def get_people():
    return dict(people=SERVER_CONFIG["test_people"])


@app.post("/api/people")
async def add_person(body: AddPersonBody):
    name = body.name.strip()
    if name and name not in SERVER_CONFIG["test_people"]:
        SERVER_CONFIG["test_people"].append(name)
    return dict(success=True, people=SERVER_CONFIG["test_people"])


@app.delete("/api/people")
async def remove_person(name: str):
    if name in SERVER_CONFIG["test_people"]:
        SERVER_CONFIG["test_people"].remove(name)
    return dict(success=True, people=SERVER_CONFIG["test_people"])


@app.post("/api/people/import-csv")
async def import_people_csv(file: UploadFile = File(...)):
    """导入 CSV 格式的人员名单 (name, student_id)"""
    if not file.filename or not file.filename.endswith(".csv"):
        raise HTTPException(400, "请上传 .csv 文件")

    content = await file.read()
    try:
        text = content.decode("utf-8-sig")  # handle BOM
    except UnicodeDecodeError:
        text = content.decode("gbk", errors="replace")

    reader = csv.DictReader(io.StringIO(text))
    if not reader.fieldnames:
        raise HTTPException(400, "CSV 文件为空或格式错误")

    # Normalize column names
    name_key = None
    for k in reader.fieldnames:
        k_lower = k.strip().lower()
        if k_lower in ("name", "姓名", "名字", "学生姓名"):
            name_key = k
            break
    if not name_key:
        raise HTTPException(400, "CSV 缺少姓名列 (name/姓名)")

    imported = 0
    for row in reader:
        raw = row.get(name_key, "").strip()
        if not raw:
            continue
        # Try to build a nice display name: "姓名 (学号)"
        student_id = None
        for k in reader.fieldnames:
            k_lower = k.strip().lower()
            if k_lower in ("student_id", "学号", "id", "编号", "studentid"):
                student_id = row.get(k, "").strip()
                break
        display = f"{raw} ({student_id})" if student_id else raw
        if display not in SERVER_CONFIG["test_people"]:
            SERVER_CONFIG["test_people"].append(display)
            imported += 1

    _log("system", "csv_import", f"CSV导入 {imported} 人 ({file.filename})")
    return dict(success=True, imported=imported, people=SERVER_CONFIG["test_people"])


# ── Webcam control ──────────────────────────────────────────────────────────


@app.post("/api/webcam/start")
async def webcam_start(body: WebcamControlBody):
    """Start webcam capture for a specific camera (via ffmpeg subprocess)."""
    if not shutil.which("ffmpeg"):
        raise HTTPException(400, "ffmpeg not found — webcam capture requires ffmpeg installed")

    cid = body.camera_id
    if cid not in CAMERAS:
        raise HTTPException(404, f"Unknown camera: {cid}")

    # Stop existing capture if any
    if cid in _webcam_captures:
        _webcam_captures[cid]["running"] = False
        time.sleep(0.5)

    # Update global camera source to webcam
    SERVER_CONFIG["camera_source"] = "webcam"

    device_idx = body.device_index if body.device_index is not None else SERVER_CONFIG["webcam_device"]
    CAMERA_WEBCAM_INDEX[cid] = device_idx

    # Start capture thread
    t = threading.Thread(target=_webcam_capture_worker, args=(cid, device_idx), daemon=True)
    t.start()
    time.sleep(0.3)

    return dict(success=True, camera_id=cid, device_index=device_idx, running=cid in _webcam_captures)


@app.post("/api/webcam/stop")
async def webcam_stop(body: WebcamControlBody):
    """Stop webcam capture for a specific camera."""
    cid = body.camera_id
    if cid not in CAMERAS:
        raise HTTPException(404, f"Unknown camera: {cid}")

    if cid in _webcam_captures:
        _webcam_captures[cid]["running"] = False
        # Wait for thread to exit
        time.sleep(0.3)

    CAMERA_WEBCAM_INDEX[cid] = None

    # If no cameras are using webcam, revert to simulated
    if not any(v is not None for v in CAMERA_WEBCAM_INDEX.values()):
        SERVER_CONFIG["camera_source"] = "simulated"

    return dict(success=True, camera_id=cid, running=cid in _webcam_captures)


@app.post("/api/webcam/start-all")
async def webcam_start_all():
    """Start webcam capture for all cameras (sequential device indices)."""
    if not shutil.which("ffmpeg"):
        raise HTTPException(400, "ffmpeg not found — webcam capture requires ffmpeg installed")

    results = {}
    for i, cid in enumerate(CAMERAS):
        if cid in _webcam_captures:
            _webcam_captures[cid]["running"] = False
            time.sleep(0.2)
        SERVER_CONFIG["camera_source"] = "webcam"
        device_idx = i
        CAMERA_WEBCAM_INDEX[cid] = device_idx
        t = threading.Thread(target=_webcam_capture_worker, args=(cid, device_idx), daemon=True)
        t.start()
        results[cid] = dict(device_index=device_idx, running=True)
        time.sleep(0.2)

    return dict(success=True, results=results)


@app.post("/api/webcam/status")
async def webcam_bulk_status():
    """Return webcam status for all cameras."""
    return dict(
        active_captures={cid: {
            "running": cid in _webcam_captures,
            "device_index": CAMERA_WEBCAM_INDEX.get(cid),
            "has_frame": cid in latest_frames,
        } for cid in CAMERAS},
        camera_source=SERVER_CONFIG["camera_source"],
    )


@app.post("/api/webcam/stop-all")
async def webcam_stop_all():
    """Stop all webcam captures."""
    for cid in list(_webcam_captures.keys()):
        _webcam_captures[cid]["running"] = False
    time.sleep(0.3)
    for cid in CAMERAS:
        CAMERA_WEBCAM_INDEX[cid] = None
    SERVER_CONFIG["camera_source"] = "simulated"
    return dict(success=True, stopped=list(_webcam_captures.keys()))


@app.get("/api/webcam/status")
async def webcam_status():
    return dict(
        opencv_available=CV2_OK,
        active_captures={cid: {
            "device_index": CAMERA_WEBCAM_INDEX.get(cid),
            "has_frame": cid in latest_frames,
        } for cid in CAMERAS},
        camera_source=SERVER_CONFIG["camera_source"],
    )

# ── Simulate event ──────────────────────────────────────────────────────────


@app.post("/api/cameras/{camera_id}/simulate")
async def simulate(camera_id: str, body: SimulateBody):
    if camera_id not in CAMERAS:
        raise HTTPException(404, f"Unknown camera: {camera_id}")

    building = CAMERAS[camera_id]["building"]
    label = CAMERAS[camera_id]["label"]
    action = body.action if body.action in ("entry", "exit", "idle") else "idle"

    # Default person for entry/exit
    people = SERVER_CONFIG["test_people"]
    person = body.person
    if not person and action != "idle" and people:
        person = people[int(time.time()) % len(people)]

    # Generate frame (only if webcam not active for this camera)
    if camera_id not in _webcam_captures:
        jpeg = _make_frame(camera_id, building, action, person)
        latest_frames[camera_id] = jpeg
    else:
        jpeg = latest_frames.get(camera_id, b"")

    # Push to Kafka t_dorm_frame
    seq = int(time.time_ns() // 1_000_000)
    quality = SERVER_CONFIG["jpeg_quality"]
    kmsg = dict(
        camera_id=camera_id,
        building=building,
        timestamp=seq,
        frame_sequence=seq,
        frame_data=base64.b64encode(jpeg).decode() if jpeg else "",
        frame_width=SERVER_CONFIG["frame_width"],
        frame_height=SERVER_CONFIG["frame_height"],
        jpeg_quality=quality,
        is_dynamic=action != "idle",
    )
    kafka_ok = False
    if producer:
        try:
            producer.send(FRAME_TOPIC, value=kmsg)
            producer.flush()
            kafka_ok = True
        except Exception as e:
            logger.error("Kafka send: %s", e)

    # Push event to t_dorm_event too (for downstream testing)
    if action != "idle" and kafka_ok:
        ev = dict(
            camera_id=camera_id,
            building=building,
            event_type=action,
            student_id=None,
            name=person.split("(")[0].strip() if person else None,
            confidence=SERVER_CONFIG["confidence_threshold"],
            timestamp=seq,
            frame_sequence=seq,
            is_stranger=False,
            snapshot_path="",
            direction_method="roi_line",
        )
        try:
            producer.send(EVENT_TOPIC, value=ev)
            producer.flush()
        except Exception:
            pass

    detail = f"{label} {action}"
    if person:
        detail += f" [{person}]"
    entry = _log(camera_id, action, detail)

    return dict(
        success=True, camera_id=camera_id, action=action,
        kafka=kafka_ok, event=entry, frame_bytes=len(jpeg),
    )


# ── Frame serving ───────────────────────────────────────────────────────────


@app.get("/api/cameras/{camera_id}/frame.jpg")
async def frame_jpg(camera_id: str):
    if camera_id not in CAMERAS:
        raise HTTPException(404)

    # If webcam is active, it continuously updates latest_frames
    if camera_id not in latest_frames:
        if camera_id in _webcam_captures:
            # Wait briefly for a frame
            for _ in range(10):
                if camera_id in latest_frames:
                    break
                time.sleep(0.05)
        else:
            latest_frames[camera_id] = _make_frame(
                camera_id, CAMERAS[camera_id]["building"], "idle", None,
            )

    frame_data = latest_frames.get(camera_id, b"")
    if not frame_data:
        frame_data = _make_frame(camera_id, CAMERAS[camera_id]["building"], "idle", None)

    return Response(content=frame_data, media_type="image/jpeg")


@app.get("/api/cameras/{camera_id}/status")
async def cam_status(camera_id: str):
    if camera_id not in CAMERAS:
        raise HTTPException(404)
    return dict(
        camera_id=camera_id,
        building=CAMERAS[camera_id]["building"],
        label=CAMERAS[camera_id]["label"],
        has_frame=camera_id in latest_frames,
        frame_size=len(latest_frames.get(camera_id, b"")),
        is_webcam=camera_id in _webcam_captures,
        webcam_device=CAMERA_WEBCAM_INDEX.get(camera_id),
    )


@app.get("/api/events")
async def events(limit: int = 50):
    return list(event_log)[:limit]


@app.get("/api/stats")
async def stats():
    """Return aggregated statistics for the dashboard."""
    total_events = len(event_log)
    event_type_counts = {"entry": 0, "exit": 0, "idle": 0}
    building_stats = {}
    camera_event_counts = {}
    now = time.time()

    for ev in event_log:
        etype = ev.get("event_type", ev.get("action", "idle"))
        if etype in event_type_counts:
            event_type_counts[etype] += 1
        cam = ev.get("camera_id", "unknown")
        camera_event_counts[cam] = camera_event_counts.get(cam, 0) + 1
        bld = ev.get("building", "")
        if bld:
            building_stats[bld] = building_stats.get(bld, 0) + 1

    # Events per minute (based on last 60s)
    recent_count = sum(
        1 for ev in event_log
        if ev.get("timestamp") and (now - ev["timestamp"] / 1000) < 60
    )
    events_per_min = recent_count

    # Frames generated: count of unique frame entries in latest_frames
    frames_generated = len(latest_frames)
    kafka_connected = producer is not None

    return dict(
        frames_generated=frames_generated,
        events_total=total_events,
        event_type_counts=event_type_counts,
        building_stats=building_stats,
        camera_event_counts=camera_event_counts,
        events_per_min=events_per_min,
        peak_events_per_min=events_per_min,
        active_cameras=len(CAMERAS),
        kafka_connected=kafka_connected,
        kafka_frames_sent=frames_generated,
        kafka_events_sent=total_events,
        uptime_sec=int(time.time() - start_time),
    )


@app.get("/api/scenarios/random")
async def scenario_random(count: int = 5):
    """Generate random traffic for testing."""
    import random as rnd

    results = []
    for _ in range(count):
        cid = rnd.choice(list(CAMERAS))
        act = rnd.choice(["entry", "exit"])
        people = SERVER_CONFIG["test_people"]
        person = rnd.choice(people) if people else None
        res = await simulate(cid, SimulateBody(action=act, person=person))
        results.append(res)
        time.sleep(0.05)
    return dict(generated=count, results=results)


@app.post("/api/scenarios/preset")
async def scenario_preset(body: dict):
    """Execute a preset demo scenario.

    Available presets:
      - rush_hour:   密集 entry/exit 流量 across all cameras
      - night_mode:   启用夜间模式 + idle 事件
      - stranger:     模拟低置信度 (stranger) 事件
      - all_entry:    所有摄像头依次 entry
      - all_exit:     所有摄像头依次 exit
    """
    import random as rnd

    preset = body.get("preset", "rush_hour")
    results = []

    if preset == "rush_hour":
        for _ in range(20):
            cid = rnd.choice(list(CAMERAS))
            act = rnd.choice(["entry", "exit"])
            people = SERVER_CONFIG["test_people"]
            person = rnd.choice(people) if people else None
            res = await simulate(cid, SimulateBody(action=act, person=person))
            results.append(res)
            time.sleep(0.05)
    elif preset == "night_mode":
        SERVER_CONFIG["night_mode_enabled"] = True
        for _ in range(8):
            cid = rnd.choice(list(CAMERAS))
            res = await simulate(cid, SimulateBody(action="idle"))
            results.append(res)
            time.sleep(0.05)
    elif preset == "stranger":
        old_threshold = SERVER_CONFIG["confidence_threshold"]
        SERVER_CONFIG["confidence_threshold"] = 0.9
        for _ in range(6):
            cid = rnd.choice(list(CAMERAS))
            act = rnd.choice(["entry", "exit"])
            people = SERVER_CONFIG["test_people"]
            person = rnd.choice(people) if people else None
            res = await simulate(cid, SimulateBody(action=act, person=person))
            results.append(res)
            time.sleep(0.05)
        SERVER_CONFIG["confidence_threshold"] = old_threshold
    elif preset == "all_entry":
        for cid in CAMERAS:
            people = SERVER_CONFIG["test_people"]
            person = rnd.choice(people) if people else None
            res = await simulate(cid, SimulateBody(action="entry", person=person))
            results.append(res)
            time.sleep(0.1)
    elif preset == "all_exit":
        for cid in CAMERAS:
            people = SERVER_CONFIG["test_people"]
            person = rnd.choice(people) if people else None
            res = await simulate(cid, SimulateBody(action="exit", person=person))
            results.append(res)
            time.sleep(0.1)
    elif preset == "clear_log":
        event_log.clear()
        return dict(success=True, preset=preset, events_cleared=True)
    else:
        raise HTTPException(400, f"Unknown preset: {preset}")

    return dict(success=True, preset=preset, generated=len(results), results=results)


# ── Static files (web dashboard) ────────────────────────────────────────────

static = Path(__file__).parent / "static"
static.mkdir(exist_ok=True)
app.mount("/", StaticFiles(directory=str(static), html=True), name="static")

# ── Entrypoint ──────────────────────────────────────────────────────────────

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(name)s [%(levelname)s] %(message)s")
    uvicorn.run("server.main:app", host="0.0.0.0", port=SERVER_PORT, reload=True)
