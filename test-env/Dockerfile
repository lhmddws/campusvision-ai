# ---- Build stage (Go + frontend) ----
FROM golang:1.26-alpine AS builder
WORKDIR /build
RUN apk add --no-cache nodejs npm
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN cd frontend && npm ci && npm run build
RUN CGO_ENABLED=0 go build -o test-env ./cmd/test-env/

# ---- Runtime stage ----
FROM alpine:3.21
RUN apk add --no-cache ffmpeg ca-certificates
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY --from=builder /build/test-env .
COPY --from=builder /build/frontend/dist/ frontend/dist/
COPY config.docker.yaml config.yaml

EXPOSE 8082
ENV TEST_SERVER_PORT=8082
USER appuser
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8082/health || exit 1
CMD ["./test-env"]
