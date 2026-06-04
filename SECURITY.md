# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in CampusVision AI, please report it privately — **do not open a public issue**.

**Where to report:**

1. **GitHub Private Vulnerability Reporting** (preferred): Navigate to the repository's "Security" tab and use the "Report a vulnerability" feature. This creates a private advisory that only maintainers can see.
2. **Email**: Send details to the project maintainers. If no dedicated security email is listed, contact the repository owner through GitHub.

**What to include in your report:**

- A clear description of the vulnerability and the affected component (stream-gateway, face-recognition, dormitory-service-go, frontend, or infrastructure)
- Steps to reproduce — include configuration snippets, API requests, or Docker Compose state needed to trigger the issue
- Potential impact — what an attacker could achieve (e.g., unauthorized camera access, credential leakage, denial of service)
- A suggested fix, if you have one
- Your contact information for follow-up questions

**What we ask of you:**

- Give us a reasonable timeframe to address the issue before disclosing it publicly
- Do not exploit the vulnerability beyond what is necessary to demonstrate it
- Do not access, modify, or exfiltrate data you are not authorized to access

## Response Time

We aim to acknowledge receipt of your report within **48 hours**. We will then work on a verified reproduction and a fix. The timeline depends on severity:

| Severity | Expected Fix Timeline |
| -------- | ---------------------- |
| Critical | 7 days                |
| High     | 14 days               |
| Medium   | 30 days               |
| Low      | 90 days               |

We will keep you informed of progress throughout the process. Once a fix is released, we will credit you in the release notes (unless you prefer to remain anonymous).

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | ✅ Current release |
| < 0.1   | ❌ Not supported   |

Only the latest release receives security patches. We do not maintain long-term support (LTS) branches at this time.

## Security-Relevant Configuration

The following configuration values are critical for production security. Ensure they are set correctly and kept secret.

### JWT_SECRET (dormitory-service-go)

```
JWT_SECRET=<random 256-bit hex string>
```

Used to sign and verify JWT tokens for frontend API authentication. **Must be set in production.** Generate a strong random value:

```bash
openssl rand -hex 32
```

If left at its default or a known value, an attacker can forge authentication tokens and gain full administrative access.

### CAMERA_ENCRYPTION_KEY (stream-gateway, dormitory-service-go)

```
CAMERA_ENCRYPTION_KEY=<32-byte key>
```

AES-256-GCM encryption key used to encrypt and decrypt camera RTSP passwords stored in the database. **Must be exactly 32 bytes.** Both stream-gateway and dormitory-service-go must use the same key to read each other's encrypted data.

```bash
# Generate a 32-byte key
openssl rand -base64 32
```

A weak or mismatched key will cause decryption failures and camera connectivity loss.

### MANAGEMENT_KEY (stream-gateway)

```
MANAGEMENT_KEY=<strong random token>
```

API key for the stream-gateway management API on port 8081. Sent via the `X-Management-Key` header. Controls camera configuration reload, service stop, and other administrative operations. Without this key, the management endpoints return 401 Unauthorized.

### HTTPS

All HTTP APIs exposed beyond localhost **must** use HTTPS. The system does not include built-in TLS termination — use a reverse proxy (nginx, Caddy, Traefik) or a cloud load balancer for:

- dormitory-service-go API (port 8083)
- Frontend SPA (port 80)
- Stream-gateway management API (port 8081)

### ONNX Model Integrity

Face detection and recognition rely on ONNX model files (`retinaface-R50.onnx`, `arcface-resnet100.onnx`). Models from untrusted sources can execute arbitrary code through deserialization vulnerabilities in ONNX Runtime.

**Mitigations:**

- Download models only from the URLs listed in `face-recognition/app/models/model_urls.yaml` — these point to the official ONNX Model Zoo and the project's verified Hugging Face repository
- Every model URL has a **SHA256 hash** that is verified after download (`model_urls.yaml` lines 4 and 10). Do **not** use models whose hashes do not match
- If you mirror models internally, generate and pin your own SHA256 hashes in a fork of `model_urls.yaml`
- Model files are gitignored (`*.onnx`) — do not commit models from untrusted sources to the repository

### Redis

Both face-recognition and dormitory-service-go connect to Redis on `127.0.0.1:6379` (db=0). Redis is used for:

- Event deduplication (face-recognition)
- Identity cache (face-recognition → dormitory-service-go)
- Session and cache data (dormitory-service-go)

**Production recommendations:**

- Enable Redis AUTH (`requirepass`) and configure `redis.host` and `redis.password` in each service's config
- Do not expose Redis to the public internet
- Consider Redis TLS for connections traversing untrusted networks

### MariaDB

- The database is initialized with `infra/mariadb/init.sql`
- Default credentials in `config.yaml` files are for local development only — **change them in production**
- Use a dedicated database user per service with minimal required privileges
- Enable TLS for database connections across networks

### Kafka

Kafka topics (`t_dorm_frame`, `t_dorm_event`, `t_dorm_alert`) carry potentially sensitive frame and event data.

**Production recommendations:**

- Enable Kafka SASL/SCRAM authentication
- Use TLS encryption for inter-broker and client-broker communication
- Set topic-level ACLs to restrict producer/consumer access

### General Security Practices

- **Environment variables over config files** for secrets. The codebase already supports `JWT_SECRET`, `CAMERA_ENCRYPTION_KEY`, and `KAFKA_BROKERS` via environment variables. Use this pattern for any new secrets.
- **Principle of least privilege** — services should only have access to the Kafka topics, Redis keys, and database tables they need.
- **Regular dependency updates** — run `go mod tidy`, `pip audit`, and `pnpm audit` periodically. Known vulnerabilities in dependencies are a common attack vector.
- **No hardcoded credentials** — RTSP URLs with inline passwords are placeholders only. In production, store camera credentials encrypted in the `dorm_camera` table (encrypted with `CAMERA_ENCRYPTION_KEY`).
- **Network segmentation** — services should be on an internal Docker network (`campusvision_default`). Only expose required ports (e.g., 8083 for the API, 80 for the frontend, 8081 for management) to the host or load balancer.
