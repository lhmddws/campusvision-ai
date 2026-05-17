#!/usr/bin/env python3
"""
Model download script for face-recognition.
Downloads ONNX models defined in model_urls.yaml, verifies SHA256 checksums.

Usage:
    python -m app.download_models

Exit codes:
    0 — all models OK (downloaded or already present)
    1 — one or more models failed
"""

import hashlib
import logging
import sys
import tempfile
from pathlib import Path

import requests
import yaml

logger = logging.getLogger(__name__)

SCRIPT_DIR = Path(__file__).resolve().parent
MODELS_DIR = SCRIPT_DIR / "models"
MANIFEST_PATH = MODELS_DIR / "model_urls.yaml"

# Placeholder sentinel — actual values verified in a later task
PLACEHOLDER_SHA256 = "PLACEHOLDER_UPDATE_ME"


def sha256_file(path: Path) -> str:
    """Compute SHA256 hex digest of a file."""
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(8192), b""):
            h.update(chunk)
    return h.hexdigest()


def download_model(name: str, url: str, expected_sha256: str, target_path: Path) -> bool:
    """Download a model file, verify SHA256, and atomically rename to target_path.

    Returns True on success, False on failure.
    """
    if target_path.exists():
        if expected_sha256 == PLACEHOLDER_SHA256:
            logger.info("✓ %s: already exists (SHA256 placeholder, skipping verification)", name)
            return True
        actual = sha256_file(target_path)
        if actual == expected_sha256:
            logger.info("✓ %s: already exists, skipping", name)
            return True
        logger.warning(
            "⚠ %s: exists but SHA256 mismatch (expected=%s, actual=%s), re-downloading",
            name, expected_sha256, actual,
        )
        target_path.unlink()

    logger.info("↓ %s: downloading from %s", name, url)
    try:
        resp = requests.get(url, stream=True, timeout=300)
        resp.raise_for_status()
    except requests.RequestException as e:
        logger.error("✗ %s: download failed: %s", name, e)
        return False

    try:
        tmp = tempfile.NamedTemporaryFile(delete=False, dir=MODELS_DIR, suffix=".onnx")
        with tmp:
            for chunk in resp.iter_content(chunk_size=8192):
                if chunk:
                    tmp.write(chunk)
            tmp_path = Path(tmp.name)
    except OSError as e:
        logger.error("✗ %s: write failed: %s", name, e)
        return False

    actual = sha256_file(tmp_path)
    if expected_sha256 != PLACEHOLDER_SHA256 and actual != expected_sha256:
        logger.error(
            "✗ %s: SHA256 mismatch after download (expected=%s, actual=%s)",
            name, expected_sha256, actual,
        )
        tmp_path.unlink(missing_ok=True)
        return False

    try:
        tmp_path.rename(target_path)
    except OSError as e:
        logger.error("✗ %s: rename failed: %s", name, e)
        tmp_path.unlink(missing_ok=True)
        return False

    logger.info("✓ %s: saved to %s", name, target_path)
    return True


def main() -> None:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s [%(levelname)s] %(message)s",
        datefmt="%H:%M:%S",
    )

    if not MANIFEST_PATH.exists():
        logger.error("Manifest not found: %s", MANIFEST_PATH)
        sys.exit(1)

    with open(MANIFEST_PATH, "r") as f:
        manifest = yaml.safe_load(f)

    models = manifest.get("models", {})
    if not models:
        logger.warning("No models defined in manifest — nothing to download")
        sys.exit(0)

    MODELS_DIR.mkdir(parents=True, exist_ok=True)

    success = True
    for name, info in models.items():
        url = info.get("url")
        expected_sha256 = info.get("sha256", PLACEHOLDER_SHA256)
        target_path = MODELS_DIR / f"{name}.onnx"

        if not url:
            logger.error("✗ %s: no URL defined in manifest", name)
            success = False
            continue

        if not download_model(name, url, expected_sha256, target_path):
            success = False

    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
