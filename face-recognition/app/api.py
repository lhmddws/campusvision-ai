"""Simple HTTP API for face embedding extraction.

Provides ``POST /api/face/extract`` — accepts a raw JPEG/PNG image,
detects a face, and returns a 512-dim embedding vector.

Designed for the enrollment workflow:
  browser → Go (multipart) → face-recognition (raw bytes) → embedding

Runs as a background thread alongside the main Kafka consumer.
"""

from __future__ import annotations

import json
import os
from http.server import HTTPServer, BaseHTTPRequestHandler
from socketserver import ThreadingMixIn

import cv2
import numpy as np
import structlog


class _Handler(BaseHTTPRequestHandler):
    """HTTP handler — class-level ``detector``, ``extractor``, ``log`` injected by ``start_api_server``."""

    detector = None  # type: ignore
    extractor = None  # type: ignore
    log: structlog.BoundLoggerBase = None  # type: ignore

    # ── routing ──────────────────────────────────────────────

    def do_POST(self) -> None:
        if self.path == "/api/face/extract":
            return self._handle_extract()
        self._json(404, {"error": "not found"})

    def do_GET(self) -> None:
        if self.path == "/health":
            return self._json(200, {"status": "ok"})
        self._json(404, {"error": "not found"})

    # ── POST /api/face/extract ───────────────────────────────

    def _handle_extract(self) -> None:
        try:
            length = int(self.headers.get("Content-Length", 0))
            if length == 0:
                return self._json(400, {"success": False, "error": "empty body"})

            body = self.rfile.read(length)

            np_arr = np.frombuffer(body, dtype=np.uint8)
            frame = cv2.imdecode(np_arr, cv2.IMREAD_COLOR)
            if frame is None:
                return self._json(400, {"success": False, "error": "invalid image"})

            faces = self.detector.detect(frame)
            if not faces:
                return self._json(200, {
                    "success": True,
                    "embedding": None,
                    "face_detected": False,
                })

            # Pick the largest face (by area)
            face = max(faces, key=lambda f: (f.x2 - f.x1) * (f.y2 - f.y1))
            embedding = self.extractor.extract(frame, face)

            self._json(200, {
                "success": True,
                "embedding": embedding.tolist(),
                "face_detected": True,
            })
        except Exception:
            self.log.exception("face_extract_internal_error")
            self._json(500, {"success": False, "error": "internal error"})

    # ── helpers ──────────────────────────────────────────────

    def _json(self, status: int, data: dict) -> None:
        body = json.dumps(data, ensure_ascii=False).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, fmt: str, *args: str) -> None:
        pass  # structured logging via `self.log`


class _ThreadedServer(ThreadingMixIn, HTTPServer):
    allow_reuse_address = True
    daemon_threads = True


def start_api_server(
    detector,
    extractor,
    host: str = "0.0.0.0",
    port: int = 8084,
) -> _ThreadedServer:
    """Start the face-extraction API server in a **daemon background thread**.

    Returns the server instance (call ``.shutdown()`` on exit).
    """
    import threading

    log = structlog.get_logger()

    _Handler.detector = detector
    _Handler.extractor = extractor
    _Handler.log = log

    server = _ThreadedServer((host, port), _Handler)
    thread = threading.Thread(target=server.serve_forever, name="face-api", daemon=True)
    thread.start()

    log.info("face_api_server_started", host=host, port=port)
    return server
