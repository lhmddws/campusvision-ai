FROM python:3.11-slim

ARG APT_MIRROR
ARG PIP_INDEX_URL
ARG PIP_TRUSTED_HOST
ARG HF_ENDPOINT=https://huggingface.co

ENV HF_ENDPOINT=${HF_ENDPOINT}

# Build-time mirrors: apt
# Overwrite all sources with a single mirror entry (reliable across all mirrors)
RUN if [ -n "$APT_MIRROR" ]; then \
        rm -f /etc/apt/sources.list /etc/apt/sources.list.d/*.sources 2>/dev/null; \
        echo "deb $APT_MIRROR trixie main" > /etc/apt/sources.list; \
    fi && \
    apt-get update && apt-get install -y --no-install-recommends \
    libgl1 libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/*

RUN addgroup --system appgroup && adduser --system --ingroup appgroup appuser

WORKDIR /app

COPY requirements.txt .
# Build-time mirrors: pip
RUN pip install --no-cache-dir \
    $( [ -n "$PIP_INDEX_URL" ] && echo "--index-url $PIP_INDEX_URL" ) \
    $( [ -n "$PIP_TRUSTED_HOST" ] && echo "--trusted-host $PIP_TRUSTED_HOST" ) \
    -r requirements.txt

COPY config.docker.yaml config.yaml
COPY app/ app/

# ── ONNX Models ──────────────────────────────────────────────────
# Three strategies (checked in order):
#   1. Pre-downloaded: place *.onnx in app/models/ before build
#      (they are auto-COPY'd by `COPY app/ app/` above)
#   2. Build-time download: docker compose build --build-arg BUILD_MODELS=1
#   3. Runtime fallback: downloaded on container startup if missing
ARG BUILD_MODELS=0
RUN if [ "$BUILD_MODELS" = "1" ]; then \
        echo "[INFO] Downloading ONNX models at build time..."; \
        python -m app.download_models; \
    else \
        echo "[INFO] ONNX models: using pre-downloaded files from app/models/ (if any), or runtime fallback."; \
        python -c "import os; f=[x for x in os.listdir('app/models') if x.endswith('.onnx')]; print(f'[INFO] Found {len(f)} pre-downloaded model(s)' if f else '[INFO] No pre-downloaded models. Will download at runtime if needed.')"; \
    fi

USER appuser
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
  CMD python -c "import app.main; exit(0)" 2>/dev/null || exit 1

CMD ["python", "-m", "app.main", "--config", "config.yaml"]
