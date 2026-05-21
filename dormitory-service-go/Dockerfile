# ---- Build stage ----
FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o dormitory-service ./cmd/dormitory-service/

# ---- Runtime stage ----
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /app
COPY --from=builder /build/dormitory-service .
COPY config.docker.yaml config.yaml
EXPOSE 8083
USER appuser
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8083/health || exit 1
ENTRYPOINT ["/app/dormitory-service"]
