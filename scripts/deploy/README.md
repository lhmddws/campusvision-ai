# CampusVision AI — Remote Deployment Guide

SSH-based deployment script for pushing CampusVision AI to production servers.

## Prerequisites

### Local Machine

- **Docker & Docker Compose** — for building images before deployment
- **rsync** — for file transfer over SSH
- **SSH key** — added to the remote server's `authorized_keys`

### Remote Server

- **Linux** (Ubuntu 22.04+ / Debian 12+ recommended)
- **Docker Engine** 24+ and **Docker Compose** v2+
- **SSH server** running with key-based authentication
- **Ports open**: 80 (frontend), 8080 (health), 8083 (API), 9092 (Kafka), 6379 (Redis), 3306 (MariaDB), 8554 (RTSP)

### SSH Key Setup

```bash
# Generate SSH key (if you don't have one)
ssh-keygen -t ed25519 -C "campusvision-deploy"

# Copy to remote server
ssh-copy-id user@your-server-ip

# Test connection
ssh user@your-server-ip "docker --version"
```

## Quick Start

### 1. Preview the Deployment Plan

```bash
# See what the script will do without making changes
bash scripts/deploy/deploy-remote.sh --dry-run --server 192.168.1.100
```

### 2. Configure Production Environment

Edit `scripts/deploy/.env.production` with your actual secrets:

```bash
# Copy and edit
cp scripts/deploy/.env.production scripts/deploy/.env.production.local
vim scripts/deploy/.env.production.local
```

> **Never commit real secrets to version control.** Use `.env.production.local` and add it to `.gitignore`.

### 3. Build Images Locally

```bash
# Build all Docker images before deploying
docker compose build
```

### 4. Deploy

```bash
# Basic deployment
bash scripts/deploy/deploy-remote.sh --server 192.168.1.100

# With custom user and path
bash scripts/deploy/deploy-remote.sh \
  --server 192.168.1.100 \
  --user deploy \
  --path /opt/campusvision \
  --env-file scripts/deploy/.env.production.local
```

## Configuration

### Script Options

| Option         | Description                                    | Default                          |
| -------------- | ---------------------------------------------- | -------------------------------- |
| `--server`     | SSH server hostname or IP                      | *(required for non-dry-run)*     |
| `--user`       | SSH username                                   | `root`                           |
| `--path`       | Remote deployment directory                    | `/opt/campusvision`              |
| `--env-file`   | Path to environment variables file             | `scripts/deploy/.env.production` |
| `--dry-run`    | Print deployment plan without executing        | —                                |
| `--help`       | Show usage information                         | —                                |

### Environment Variables

See `.env.production` for all configurable variables. Key ones:

| Variable                | Description                              | Example                          |
| ----------------------- | ---------------------------------------- | -------------------------------- |
| `MARIADB_ROOT_PASSWORD` | MariaDB root password                    | `your-secure-password`           |
| `MARIADB_PASSWORD`      | MariaDB application user password        | `your-secure-password`           |
| `JWT_SECRET`            | JWT signing secret (32+ chars)           | `random-256-bit-string`          |
| `CAMERA_ENCRYPTION_KEY` | AES-256-GCM key (exactly 32 bytes)       | `32-byte-exact-key-here!!!!`     |
| `MANAGEMENT_KEY`        | Stream gateway management API key        | `random-management-key`          |
| `FACE_AUTH_TOKEN`       | Face recognition matcher auth token      | `random-auth-token`              |
| `CORS_ALLOWED_ORIGINS`  | Comma-separated allowed origins          | `http://192.168.1.100`           |
| `KAFKA_BROKERS`         | Kafka broker addresses                   | `kafka:9092`                     |

## Deployed Services

After deployment, the following services will be running:

| Service                | Port    | URL                              |
| ---------------------- | ------- | -------------------------------- |
| Frontend (Vue 3)       | 80      | `http://<server>/`               |
| Stream Gateway (health)| 8080    | `http://<server>:8080/health`    |
| Stream Gateway (mgmt)  | 8081    | `http://<server>:8081/`          |
| Dormitory Service (API)| 8083    | `http://<server>:8083/api/`      |
| Kafka                  | 9092    | `<server>:9092`                  |
| Redis                  | 6379    | `<server>:6379`                  |
| MariaDB                | 3306    | `<server>:3306`                  |
| Mediamtx (RTSP)        | 8554    | `<server>:8554`                  |

## Verification

### Check Container Status

```bash
ssh user@server "cd /opt/campusvision && docker compose ps"
```

### Health Checks

```bash
# Stream Gateway
curl http://<server>:8080/health

# Dormitory Service
curl http://<server>:8083/api/health

# Frontend
curl -I http://<server>
```

### View Logs

```bash
# All services
ssh user@server "cd /opt/campusvision && docker compose logs -f"

# Specific service
ssh user@server "cd /opt/campusvision && docker compose logs -f stream-gateway"
```

## Troubleshooting

### SSH Connection Failed

```bash
# Test SSH connectivity
ssh -v user@server "echo connected"

# Check SSH key permissions
chmod 600 ~/.ssh/id_ed25519
chmod 644 ~/.ssh/id_ed25519.pub
```

### Docker Not Found on Remote

```bash
# Install Docker on Ubuntu/Debian
ssh user@server "curl -fsSL https://get.docker.com | sh"
ssh user@server "sudo usermod -aG docker $USER"
```

### Port Already in Use

```bash
# Check what's using a port
ssh user@server "sudo lsof -i :8080"

# Stop conflicting service
ssh user@server "sudo systemctl stop nginx"
```

### Service Not Starting

```bash
# Check logs
ssh user@server "cd /opt/campusvision && docker compose logs <service-name>"

# Restart specific service
ssh user@server "cd /opt/campusvision && docker compose up -d <service-name>"
```

## Manual Deployment (Without Script)

If you prefer manual deployment:

```bash
# 1. SSH to server
ssh user@server

# 2. Create deployment directory
mkdir -p /opt/campusvision
cd /opt/campusvision

# 3. Copy files from local
# (from your local machine)
scp docker-compose.yml user@server:/opt/campusvision/
scp .env user@server:/opt/campusvision/
scp -r stream-gateway/config.docker.yaml user@server:/opt/campusvision/stream-gateway/
scp -r face-recognition/config.docker.yaml user@server:/opt/campusvision/face-recognition/
scp -r dormitory-service-go/config.docker.yaml user@server:/opt/campusvision/dormitory-service-go/

# 4. Start services
docker compose pull
docker compose up -d

# 5. Verify
docker compose ps
```
