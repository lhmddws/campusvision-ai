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
WORKDIR /app
COPY --from=builder /build/dormitory-service .
COPY config.docker.yaml config.yaml
EXPOSE 8083
ENTRYPOINT ["/app/dormitory-service"]
