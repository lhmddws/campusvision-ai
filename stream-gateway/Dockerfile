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
WORKDIR /app
COPY --from=builder /build/stream-gateway .
COPY config.yaml .
EXPOSE 8080
ENTRYPOINT ["./stream-gateway", "--config", "config.yaml"]
