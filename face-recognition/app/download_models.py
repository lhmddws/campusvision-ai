#!/usr/bin/env python3
"""
Model download script for face-recognition.
Downloads ONNX models defined in model_urls.yaml, verifies SHA256 checksums.

Supports multiple download strategies for restricted networks (China etc.):

  Strategy A — Pre-download (recommended for Docker):
    1. Download models locally:  python -m app.download_models
    2. Build Docker image with:  docker compose build face-recognition
       (Dockerfile automatically COPYs local *.onnx files)

  Strategy B — Build-time download (needs network access):
    1. docker compose build --build-arg "BUILD_MODELS=1" \
       --build-arg "HF_ENDPOINT=https://hf-mirror.com" face-recognition

  Strategy C — Runtime fallback (no models in image):
    Container downloads models on first startup if missing (via app.main init).

Mirror priority (highest first):
  1. --mirror CLI flag
  2. HF_ENDPOINT environment variable
  3. Multiple built-in mirrors tried automatically on failure

Usage:
    python -m app.download_models
    python -m app.download_models --retries 5
    python -m app.download_models --mirror https://hf-mirror.com
    python -m app.download_models --proxy http://127.0.0.1:7890

Exit codes:
    0 — all models OK (downloaded or already present)
    1 — one or more models failed
"""

import argparse
import hashlib
import logging
import os
import sys
import tempfile
import time
from pathlib import Path

import requests
import yaml

logger = logging.getLogger(__name__)

SCRIPT_DIR = Path(__file__).resolve().parent
MODELS_DIR = SCRIPT_DIR / "models"
MANIFEST_PATH = MODELS_DIR / "model_urls.yaml"

PLACEHOLDER_SHA256 = "PLACEHOLDER_UPDATE_ME"
DEFAULT_TIMEOUT = 300
DEFAULT_RETRIES = 3
CHUNK_SIZE = 8192

# Built-in mirrors tried in order (duplicates are skipped)
DEFAULT_MIRRORS = [
    "https://huggingface.co",       # official
    "https://hf-mirror.com",        # Chinese mirror (may redirect)
]


def parse_args() -> argparse.Namespace:
    """Parse CLI arguments."""
    parser = argparse.ArgumentParser(
        description="Download ONNX models for face-recognition",
    )
    parser.add_argument(
        "--mirror", default=None,
        help="Custom HuggingFace mirror URL (overrides HF_ENDPOINT env var)",
    )
    parser.add_argument(
        "--retries", type=int, default=DEFAULT_RETRIES,
        help=f"Max retries per mirror (default: {DEFAULT_RETRIES})",
    )
    parser.add_argument(
        "--timeout", type=int, default=DEFAULT_TIMEOUT,
        help=f"Download timeout in seconds (default: {DEFAULT_TIMEOUT})",
    )
    parser.add_argument(
        "--proxy", default=None,
        help="HTTP/HTTPS proxy URL, e.g. http://127.0.0.1:7890",
    )
    parser.add_argument(
        "--list-mirrors", action="store_true",
        help="List configured mirrors and exit",
    )
    return parser.parse_args()


def get_mirrors(args_mirror: str | None) -> list[str]:
    """Build the ordered list of mirrors to try (deduplicated)."""
    mirrors: list[str] = []

    # 1. CLI --mirror flag (highest priority)
    if args_mirror:
        mirrors.append(args_mirror.rstrip("/"))

    # 2. HF_ENDPOINT env var
    env_mirror = os.environ.get("HF_ENDPOINT", "").strip().rstrip("/")
    if env_mirror and env_mirror not in mirrors:
        mirrors.append(env_mirror)

    # 3. Built-in defaults
    for m in DEFAULT_MIRRORS:
        if m not in mirrors:
            mirrors.append(m)

    return mirrors


def apply_mirror(url: str, mirror: str) -> str:
    """Replace the huggingface.co host in *url* with *mirror*."""
    return url.replace("https://huggingface.co", mirror)


def sha256_file(path: Path) -> str:
    """Compute SHA256 hex digest of a file."""
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(CHUNK_SIZE), b""):
            h.update(chunk)
    return h.hexdigest()


def download_model(
    name: str,
    url: str,
    expected_sha256: str,
    target_path: Path,
    *,
    mirrors: list[str],
    max_retries: int,
    timeout: int,
    proxy: str | None = None,
) -> bool:
    """Download a model with mirror fallback and retry logic.

    Tries each mirror in order. For each mirror, retries up to *max_retries*
    times with exponential backoff. Verifies SHA256 after download.
    Returns True on success, False if all mirrors & retries exhausted.
    """
    # Already exists and valid → skip
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

    # Prepare session (proxy support)
    session = requests.Session()
    if proxy:
        session.proxies = {"http": proxy, "https": proxy}
        logger.info("🔌 using proxy: %s", proxy)

    for mirror in mirrors:
        effective_url = apply_mirror(url, mirror)

        for attempt in range(1, max_retries + 1):
            logger.info(
                "↓ %s: attempt %d/%d — %s",
                name, attempt, max_retries, effective_url,
            )

            try:
                resp = session.get(effective_url, stream=True, timeout=timeout)
                resp.raise_for_status()

                # Detect redirect to blocked host
                if resp.history:
                    final_url = resp.url
                    for hist in resp.history:
                        logger.debug(
                            "  ↪ %s → %s", hist.url, hist.headers.get("Location", ""),
                        )
                    if "huggingface.co" in final_url and final_url != effective_url:
                        logger.warning(
                            "⚠ %s: mirror %s redirected to huggingface.co (may be blocked). "
                            "Trying next strategy...",
                            name, mirror,
                        )
                        break  # try next mirror

                # Stream write to temp file
                try:
                    tmp = tempfile.NamedTemporaryFile(
                        delete=False, dir=target_path.parent, suffix=".onnx",
                    )
                    with tmp:
                        for chunk in resp.iter_content(chunk_size=CHUNK_SIZE):
                            if chunk:
                                tmp.write(chunk)
                        tmp_path = Path(tmp.name)
                except OSError as e:
                    logger.error("✗ %s: write failed: %s", name, e)
                    return False

                # SHA256 verification
                actual = sha256_file(tmp_path)
                size_mb = tmp_path.stat().st_size / (1024 * 1024)

                if expected_sha256 == PLACEHOLDER_SHA256:
                    logger.info(
                        "✓ %s: downloaded (%.1f MB, SHA256 placeholder — skipping verification)",
                        name, size_mb,
                    )
                    is_valid = True
                elif actual == expected_sha256:
                    logger.info("✓ %s: downloaded (%.1f MB, SHA256 match)", name, size_mb)
                    is_valid = True
                else:
                    logger.error(
                        "✗ %s: SHA256 mismatch (expected=%s, actual=%s)",
                        name, expected_sha256, actual,
                    )
                    tmp_path.unlink(missing_ok=True)
                    is_valid = False

                if not is_valid:
                    if attempt < max_retries:
                        wait = 2 ** attempt
                        logger.info("  retrying in %ds...", wait)
                        time.sleep(wait)
                    continue

                # Atomically rename
                try:
                    tmp_path.rename(target_path)
                except OSError as e:
                    logger.error("✗ %s: rename failed: %s", name, e)
                    tmp_path.unlink(missing_ok=True)
                    return False

                logger.info("✓ %s: saved to %s", name, target_path)
                return True

            except requests.RequestException as e:
                logger.warning("⚠ %s: attempt %d/%d failed: %s", name, attempt, max_retries, e)
                if attempt < max_retries:
                    wait = 2 ** attempt
                    logger.info("  retrying in %ds...", wait)
                    time.sleep(wait)

        logger.warning("⚠ %s: all retries exhausted for mirror %s", name, mirror)

    logger.error("✗ %s: all mirrors exhausted — download failed", name)
    return False


def list_mirrors(mirrors: list[str]) -> None:
    """Print configured mirrors and exit."""
    print("Configured HuggingFace mirrors (in order):")
    for i, m in enumerate(mirrors, 1):
        print(f"  {i}. {m}")
    print()
    print("Set HF_ENDPOINT env var or use --mirror to override.")


def main() -> None:
    args = parse_args()

    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s [%(levelname)s] %(message)s",
        datefmt="%H:%M:%S",
    )

    mirrors = get_mirrors(args.mirror)

    if args.list_mirrors:
        list_mirrors(mirrors)
        sys.exit(0)

    if args.proxy:
        logger.info("🔌 HTTP proxy: %s", args.proxy)

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

    logger.info("📋 Mirrors: %s", mirrors)

    success = True
    for name, info in models.items():
        url = info.get("url")
        expected_sha256 = info.get("sha256", PLACEHOLDER_SHA256)
        target_path = MODELS_DIR / f"{name}.onnx"

        if not url:
            logger.error("✗ %s: no URL defined in manifest", name)
            success = False
            continue

        ok = download_model(
            name,
            url,
            expected_sha256,
            target_path,
            mirrors=mirrors,
            max_retries=args.retries,
            timeout=args.timeout,
            proxy=args.proxy,
        )
        if not ok:
            success = False

    if success:
        logger.info("✅ All models ready")
    else:
        logger.warning(
            "⚠ Some models failed. Options:\n"
            "  1. Use a proxy:  python -m app.download_models --proxy http://YOUR_PROXY:PORT\n"
            "  2. Set HF_ENDPOINT:  HF_ENDPOINT=https://hf-mirror.com python -m app.download_models\n"
            "  3. Download manually and place *.onnx files in app/models/\n"
            "  4. Run Docker build with pre-downloaded models (they are auto-COPY'd)"
        )

    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
