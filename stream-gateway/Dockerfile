# ---- Build stage ----
FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o stream-gateway ./cmd/main.go

# ---- Runtime stage ----
FROM alpine:3.21
RUN apk add --no-cache ca-certificates ffmpeg
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY --from=builder /build/stream-gateway .
COPY config.docker.yaml config.yaml
EXPOSE 8080
USER appuser
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1
ENTRYPOINT ["./stream-gateway", "--config", "config.yaml"]
