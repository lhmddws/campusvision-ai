FROM python:3.11-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    libgl1 libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/*

RUN addgroup --system appgroup && adduser --system --ingroup appgroup appuser

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY config.docker.yaml config.yaml
COPY app/ app/

# Download ONNX models (build-time)
RUN python -m app.download_models

USER appuser
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
  CMD python -c "import app.main; exit(0)" 2>/dev/null || exit 1

CMD ["python", "-m", "app.main"]
