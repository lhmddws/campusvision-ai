# Contributing to CampusVision AI

Thank you for considering contributing to CampusVision AI! This project is an AI-powered dormitory surveillance system spanning Go (stream-gateway, dormitory-service-go), Python (face-recognition), and Vue 3 + TypeScript (frontend). We welcome contributions of all kinds — bug reports, feature suggestions, documentation improvements, and code changes.

## Code of Conduct

This project and everyone participating in it is governed by the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/2/1/code_of_conduct/). By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

## How to Contribute

### Reporting Bugs

Before submitting a bug report, please check the existing issues to see if the problem has already been reported. If it hasn't, open a new issue and include:

- A clear, descriptive title
- Steps to reproduce (include config snippets, Kafka topic dumps, or API request/response logs where relevant)
- Expected behavior and actual behavior
- Environment details: OS, Go version, Python version, Docker version, ffmpeg version
- If relevant, which service is affected (`stream-gateway`, `face-recognition`, `dormitory-service-go`, or `frontend`)
- Log output or error messages (anonymize any sensitive data like RTSP URLs or camera credentials)

**Suggested template:**

```markdown
### Description
[Clear description of the bug]

### Steps to Reproduce
1. Start service: `...`
2. Send request: `...`
3. Observe: `...`

### Expected Behavior
[What should happen]

### Actual Behavior
[What actually happens]

### Environment
- OS: [e.g. macOS 14.5, Ubuntu 22.04]
- Go: [e.g. 1.26.1]
- Python: [e.g. 3.11.9]
- Docker: [e.g. 24.0.7]
- ffmpeg: [e.g. 6.0]
- Service: [stream-gateway / face-recognition / dormitory-service-go / frontend]

### Logs
```
[paste relevant logs]
```
```

### Suggesting Enhancements

Feature requests are welcome. Please open an issue with:

- A clear title and detailed description of the proposed feature
- The motivation — what problem does it solve?
- If applicable, which module it affects and how it fits into the existing pipeline
- Any relevant context from `doc/prd/` or existing design documents

### Pull Request Process

1. **Discuss first** — for significant changes, open an issue to discuss the approach before writing code. This avoids wasted effort on changes that may not align with the project direction.
2. **Fork and branch** — create a feature branch from `main` (see [Branch Strategy](#branch-strategy)).
3. **Write code** — follow the [Coding Standards](#coding-standards) and [Testing Guidelines](#testing-guidelines) below.
4. **Run tests** — ensure the full test suite passes for all affected modules.
5. **Keep commits clean** — use conventional commit messages (see [Commit Message Format](#commit-message-format)).
6. **Open a pull request** — target `main`. Provide a clear description of what the PR does and why.
7. **Code review** — at least one maintainer will review. Address feedback promptly.
8. **Merge** — once approved, a maintainer will merge. Squash commits are preferred for feature branches.

## Development Setup

### Prerequisites

| Tool          | Minimum Version | Required For              | Notes                                   |
| ------------- | --------------- | ------------------------- | --------------------------------------- |
| Go            | 1.26            | stream-gateway, dormitory-service-go | —                                |
| Python        | 3.11            | face-recognition          | 3.14 also supported (`.venv/`)          |
| Docker        | 24+             | Infrastructure services   | Docker Compose v2                       |
| ffmpeg        | 6.0+            | stream-gateway            | Must be on `$PATH`                      |
| pnpm          | 8+              | frontend                  | Node.js 18+ required                    |
| Node.js       | 18+             | frontend                  | —                                       |

### Getting Started

```bash
# 1. Clone the repository
git clone https://github.com/sims/campusvision-ai.git
cd campusvision-ai

# 2. Start infrastructure (Kafka + Redis for minimal dev)
docker compose up -d kafka redis

# Or start everything (requires more resources)
docker compose up -d

# 3. Download ONNX models for face-recognition
cd face-recognition
python -m app.download_models
cd ..
```

### Running Each Service

**Stream Gateway** (port 8080 health, 8081 management):

```bash
cd stream-gateway
go run cmd/main.go --config config.yaml
```

**Face Recognition** (Kafka consumer/producer, no HTTP port):

```bash
cd face-recognition
python -m app.main --config config.yaml
```

**Dormitory Service** (HTTP API on port 8083):

```bash
cd dormitory-service-go
CONFIG_PATH=config.yaml go run ./cmd/dormitory-service/
```

**Frontend** (SPA dev server on port 80):

```bash
cd frontend
pnpm install
pnpm dev
```

### Docker Build

```bash
# Build with China mirror support (optional)
docker compose build \
  --build-arg "HF_ENDPOINT=https://hf-mirror.com" \
  --build-arg "APT_MIRROR=https://mirrors.tuna.tsinghua.edu.cn/debian" \
  --build-arg "PIP_INDEX_URL=https://pypi.tuna.tsinghua.edu.cn/simple" \
  face-recognition

# Build all services
docker compose build
```

## Coding Standards

### Go (stream-gateway, dormitory-service-go)

- **Formatting**: All Go code must be formatted with `gofmt`. Run `gofmt -s -w .` before committing.
- **Linting**: Use `golangci-lint` with the project's `.golangci.yml` configuration — enables `errcheck`, `govet` (all checks except fieldalignment), `staticcheck`, `gosimple`, `ineffassign`, `unused`.
- **Project layout**: Follow the [Standard Go Project Layout](https://github.com/golang-standards/project-layout) conventions. `cmd/` for entry points, `internal/` for private packages, `pkg/` for shared library code.
- **Error handling**: Always check returned errors. Never discard errors with `_ =`. Use `fmt.Errorf("context: %w", err)` for error wrapping.
- **Naming**: Use camelCase for variables, PascalCase for exported identifiers. Single-letter variable names are acceptable only in very short scopes.

### Python (face-recognition)

- **Formatter**: Use `ruff format` with the project's `ruff.toml` (`line-length = 120`).
- **Linting**: Run `ruff check` — enables F (pyflakes), E/W (pycodestyle), I (isort), N (pep8-naming).
- **Type hints**: All function signatures must include type annotations. Use `from __future__ import annotations` for forward references.
- **Imports**: Group in order: standard library → third-party → local. Use absolute imports over relative.
- **Target version**: Python 3.11+ (`target-version = "py311"` in ruff.toml).
- **Docstrings**: Use Google-style docstrings for public modules, classes, and functions.

### Frontend (Vue 3 + TypeScript)

- **ESLint**: Run `npx eslint .` using the project's `.eslintrc.js` configuration.
- **Prettier**: Run `npx prettier --check .` using the project's `.prettierrc` (printWidth 100, single quotes, no trailing commas, arrow parens avoided).
- **TypeScript**: Enable strict mode in `tsconfig.json`. Avoid `any` — use proper types or `unknown` with type guards.
- **Vue**: Use Composition API with `<script setup lang="ts">`. Component files use PascalCase naming.
- **EditorConfig**: The root `.editorconfig` enforces UTF-8, trailing whitespace trimming, and final newlines across all file types.

### General

- **Line endings**: LF (Unix), not CRLF.
- **File encoding**: UTF-8.
- **Trailing whitespace**: Trimmed.
- **Final newline**: Every file ends with one.
- **Commit messages**: Follow [Conventional Commits](#commit-message-format).

## Testing Guidelines

- All new code should include tests. Bug fixes should include a regression test.
- Run the full test suite for affected modules before opening a pull request:

```bash
# Go services
cd stream-gateway && go test ./...
cd dormitory-service-go && go test ./...

# Face recognition
cd face-recognition && pytest tests/

# Frontend
cd frontend && npx vitest run
```

- **Go tests**: Use `testing` package with `go-sqlmock` for database mocking. Test files should follow `*_test.go` naming and reside alongside the code they test.
- **Python tests**: Use `pytest` with `unittest.mock.patch` for Kafka and HTTP mocking. Tests use Haar Cascade fallback (no ONNX models required). See `tests/conftest.py` for shared fixtures.
- **Frontend tests**: Use Vitest with `@vue/test-utils`.
- **Integration tests**: Where possible, mock external dependencies (Kafka, Redis, HTTP APIs) rather than requiring a running infrastructure.
- **Test coverage** (aspirational): Aim for 70%+ coverage on new code.

## Branch Strategy

- **`main`** — Production-ready code. All commits to `main` must come from reviewed pull requests.
- **`feature/*`** — New features. Branch from `main`, merge back via squash-merge PR. Example: `feature/behavior-alert-persistence`.
- **`fix/*`** — Bug fixes. Branch from `main`, merge back via squash-merge PR. Example: `fix/haar-cascade-windows-path`.
- **`chore/*`** — Maintenance, dependency updates, documentation. Same workflow.
- Branches should be short-lived. If a branch diverges significantly from `main`, rebase before opening a PR.

## Commit Message Format

This project uses [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/). Each commit message must follow this structure:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**

| Type       | Usage                                              |
| ---------- | -------------------------------------------------- |
| `feat`     | A new feature                                     |
| `fix`      | A bug fix                                         |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `test`     | Adding or updating tests                          |
| `docs`     | Documentation changes                             |
| `chore`    | Maintenance, dependencies, tooling                 |
| `perf`     | Performance improvement                           |

**Scopes:**

| Scope                | Module             |
| -------------------- | ------------------ |
| `stream-gateway`     | Go stream ingestion service    |
| `face-recognition`   | Python face processing service |
| `dormitory-service`  | Go business API service        |
| `frontend`           | Vue 3 SPA                      |
| `infra`              | Docker Compose, DB migrations  |
| `docs`               | Documentation, API specs       |

**Examples:**

```
feat(stream-gateway): add motion-based frame extraction gating

Introduce motion_threshold config parameter (default 0.05) to skip
frame production when inter-frame pixel difference is below threshold.
Reduces Kafka load during static scenes.

Closes #42
```

```
fix(face-recognition): use portable Haar Cascade path

Replace hardcoded macOS Homebrew path with cv2.data.haarcascades
to fix face detection fallback on Linux and Windows.

Fixes #38
```

```
docs(docs): add CHANGELOG and CONTRIBUTING guides
```

## Review Process

1. All pull requests require at least one approval from a maintainer.
2. Automated checks (linting, tests) must pass before merge.
3. Reviewers will assess:
   - Correctness — does the code do what it claims?
   - Test coverage — are there adequate tests?
   - Code quality — does it follow the project's coding standards?
   - Performance — are there obvious bottlenecks (e.g., N+1 queries, unnecessary Kafka messages)?
   - Security — are credentials, tokens, or camera URLs handled safely?
4. Keep PRs focused — one logical change per PR. Large features should be broken into smaller, reviewable PRs.
5. Address review comments promptly. If you disagree, provide reasoning — respectful debate is encouraged.
6. Once approved, a maintainer will merge. If the PR has multiple commits, prefer squash merge to keep `main` history clean.
