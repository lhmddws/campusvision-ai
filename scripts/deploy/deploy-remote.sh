#!/usr/bin/env bash
# =============================================================================
# CampusVision AI — SSH Remote Deploy Script
# =============================================================================
# Transfers Docker configuration to a remote server and starts services.
# Uses SSH key authentication only — no password-based auth.
#
# Usage:
#   ./deploy-remote.sh --server <host> [options]
#   ./deploy-remote.sh --server 192.168.1.100 --user admin --dry-run
#
# Options:
#   --server <host>       SSH server hostname or IP (required unless --dry-run)
#   --user <username>     SSH username (default: root)
#   --path <path>         Remote deploy path (default: /opt/campusvision)
#   --env-file <path>     Local env file path (default: scripts/deploy/.env.production)
#   --dry-run             Print deployment plan without executing
#   --help                Show this help message
# =============================================================================

set -euo pipefail

# --- Defaults ---
DRY_RUN=false
SERVER=""
SSH_USER="root"
REMOTE_PATH="/opt/campusvision"
ENV_FILE="scripts/deploy/.env.production"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# --- Colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# --- Logging ---
log_info()  { echo -e "${BLUE}[INFO]${NC}  $*"; }
log_ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }
log_step()  { echo -e "\n${BLUE}━━━ $* ━━━${NC}"; }

# --- Usage ---
usage() {
    sed -n '2,/^# ===/p' "$0" | sed 's/^# \?//'
    exit 0
}

# --- Parse arguments ---
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --server)
                SERVER="$2"
                shift 2
                ;;
            --user)
                SSH_USER="$2"
                shift 2
                ;;
            --path)
                REMOTE_PATH="$2"
                shift 2
                ;;
            --env-file)
                ENV_FILE="$2"
                shift 2
                ;;
            --help|-h)
                usage
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                ;;
        esac
    done
}

# --- Validation ---
validate() {
    if [[ -z "$SERVER" && "$DRY_RUN" == false ]]; then
        log_error "--server is required (or use --dry-run)"
        exit 1
    fi

    if [[ ! -f "$PROJECT_ROOT/docker-compose.yml" ]]; then
        log_error "docker-compose.yml not found at project root: $PROJECT_ROOT"
        exit 1
    fi

    if [[ ! -f "$PROJECT_ROOT/$ENV_FILE" ]]; then
        log_error "Environment file not found: $PROJECT_ROOT/$ENV_FILE"
        exit 1
    fi

    # Check required config files exist
    local config_files=(
        "stream-gateway/config.docker.yaml"
        "face-recognition/config.docker.yaml"
        "dormitory-service-go/config.docker.yaml"
    )
    for cf in "${config_files[@]}"; do
        if [[ ! -f "$PROJECT_ROOT/$cf" ]]; then
            log_error "Config file not found: $cf"
            exit 1
        fi
    done
}

# --- Dry Run ---
run_dry_run() {
    log_step "DRY RUN — Deployment Plan"
    echo ""
    echo "  Server:       ${SERVER:-<not specified>}"
    echo "  SSH User:     $SSH_USER"
    echo "  Remote Path:  $REMOTE_PATH"
    echo "  Env File:     $ENV_FILE"
    echo ""

    log_step "Step 1: Pre-deployment Checks"
    echo "  ✓ Verify docker-compose.yml exists"
    echo "  ✓ Verify all config.docker.yaml files exist"
    echo "  ✓ Verify environment file exists"
    echo "  ✓ Check Docker is running locally"
    echo "  ✓ Check docker compose images are built"

    log_step "Step 2: File Transfer (rsync over SSH)"
    echo "  Files to transfer to ${SSH_USER}@${SERVER:-<host>}:$REMOTE_PATH/:"
    echo "    → docker-compose.yml"
    echo "    → stream-gateway/config.docker.yaml"
    echo "    → face-recognition/config.docker.yaml"
    echo "    → dormitory-service-go/config.docker.yaml"
    echo "    → .env.production → .env"

    log_step "Step 3: Remote Service Startup"
    echo "  SSH commands on ${SSH_USER}@${SERVER:-<host>}:"
    echo "    1. mkdir -p $REMOTE_PATH"
    echo "    2. cd $REMOTE_PATH"
    echo "    3. docker compose pull          # Pull latest images"
    echo "    4. docker compose up -d         # Start all services"
    echo "    5. docker compose ps            # Verify running containers"

    log_step "Step 4: Health Checks"
    echo "  Endpoints to verify:"
    echo "    → http://${SERVER:-<host>}:8080/health          (stream-gateway)"
    echo "    → http://${SERVER:-<host>}:8083/api/health      (dormitory-service-go)"
    echo "    → http://${SERVER:-<host>}                      (frontend)"

    log_step "Step 5: Deployment Report"
    echo "  Print service status and any warnings"
    echo ""
    log_info "Dry run complete. No changes were made."
    log_info "Run without --dry-run to execute this deployment plan."
}

# --- Pre-deployment Checks ---
pre_deploy_checks() {
    log_step "Step 1: Pre-deployment Checks"

    log_info "Checking Docker is running..."
    if ! docker info >/dev/null 2>&1; then
        log_error "Docker is not running. Please start Docker and try again."
        exit 1
    fi
    log_ok "Docker is running"

    log_info "Checking docker compose images..."
    local images
    images=$(docker compose -f "$PROJECT_ROOT/docker-compose.yml" config --services 2>/dev/null || true)
    if [[ -z "$images" ]]; then
        log_warn "docker compose config failed — will rely on remote build"
    else
        log_ok "Docker Compose services found: $(echo "$images" | tr '\n' ', ')"
    fi
}

# --- Transfer Files ---
transfer_files() {
    log_step "Step 2: Transferring Files to Remote Server"

    local ssh_target="${SSH_USER}@${SERVER}"

    log_info "Creating remote directory structure..."
    if [[ "$DRY_RUN" == false ]]; then
        ssh "$ssh_target" "mkdir -p ${REMOTE_PATH}/stream-gateway ${REMOTE_PATH}/face-recognition ${REMOTE_PATH}/dormitory-service-go"
    fi
    log_ok "Remote directories ready"

    log_info "Syncing files via rsync..."
    if [[ "$DRY_RUN" == false ]]; then
        rsync -avz --delete \
            --include='docker-compose.yml' \
            --include='stream-gateway/config.docker.yaml' \
            --include='face-recognition/config.docker.yaml' \
            --include='dormitory-service-go/config.docker.yaml' \
            --include='.env' \
            --exclude='*' \
            "$PROJECT_ROOT/" \
            "${ssh_target}:${REMOTE_PATH}/"

        # Transfer env file as .env
        rsync -avz "$PROJECT_ROOT/$ENV_FILE" "${ssh_target}:${REMOTE_PATH}/.env"
    else
        log_info "[DRY-RUN] Would rsync config files to ${ssh_target}:${REMOTE_PATH}/"
    fi
    log_ok "Files transferred"
}

# --- Remote Startup ---
remote_startup() {
    log_step "Step 3: Starting Remote Services"

    local ssh_target="${SSH_USER}@${SERVER}"

    log_info "Pulling latest images..."
    if [[ "$DRY_RUN" == false ]]; then
        ssh "$ssh_target" "cd ${REMOTE_PATH} && docker compose pull" || {
            log_warn "docker compose pull failed — continuing with local images"
        }
    fi
    log_ok "Images pulled (or skipped)"

    log_info "Starting services..."
    if [[ "$DRY_RUN" == false ]]; then
        ssh "$ssh_target" "cd ${REMOTE_PATH} && docker compose up -d"
    fi
    log_ok "Services started"

    log_info "Waiting for services to initialize (30s)..."
    if [[ "$DRY_RUN" == false ]]; then
        sleep 30
    fi
}

# --- Health Checks ---
health_checks() {
    log_step "Step 4: Health Checks"

    local checks=(
        "http://${SERVER}:8080/health|stream-gateway"
        "http://${SERVER}:8083/api/health|dormitory-service-go"
        "http://${SERVER}/|frontend"
    )

    local all_passed=true
    for check in "${checks[@]}"; do
        IFS='|' read -r url name <<< "$check"
        log_info "Checking $name ($url)..."
        if [[ "$DRY_RUN" == false ]]; then
            local http_code
            http_code=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 --max-time 10 "$url" 2>/dev/null || echo "000")
            if [[ "$http_code" =~ ^[23] ]]; then
                log_ok "$name is healthy (HTTP $http_code)"
            else
                log_warn "$name returned HTTP $http_code — may still be starting"
                all_passed=false
            fi
        else
            log_info "[DRY-RUN] Would check $url"
        fi
    done

    if [[ "$DRY_RUN" == false ]]; then
        log_info "Remote container status:"
        ssh "${SSH_USER}@${SERVER}" "cd ${REMOTE_PATH} && docker compose ps" 2>/dev/null || true
    fi

    if [[ "$all_passed" == false ]]; then
        log_warn "Some health checks did not pass — services may need more time to start"
    else
        log_ok "All health checks passed"
    fi
}

# --- Deployment Report ---
deployment_report() {
    log_step "Step 5: Deployment Report"

    echo ""
    echo "  ╔══════════════════════════════════════════════════════════╗"
    echo "  ║           CampusVision AI — Deployment Summary          ║"
    echo "  ╠══════════════════════════════════════════════════════════╣"
    printf "  ║  Server:       %-44s║\n" "$SERVER"
    printf "  ║  User:         %-44s║\n" "$SSH_USER"
    printf "  ║  Path:         %-44s║\n" "$REMOTE_PATH"
    printf "  ║  Env File:     %-44s║\n" "$ENV_FILE"
    echo "  ╠══════════════════════════════════════════════════════════╣"
    echo "  ║  Services:                                               ║"
    echo "  ║    • stream-gateway      → http://${SERVER}:8080        ║"
    echo "  ║    • dormitory-service   → http://${SERVER}:8083        ║"
    echo "  ║    • frontend            → http://${SERVER}             ║"
    echo "  ║    • face-recognition    → (Kafka consumer, no port)    ║"
    echo "  ║    • kafka               → ${SERVER}:9092               ║"
    echo "  ║    • redis               → ${SERVER}:6379               ║"
    echo "  ║    • mariadb             → ${SERVER}:3306               ║"
    echo "  ║    • mediamtx            → ${SERVER}:8554 (RTSP)        ║"
    echo "  ╚══════════════════════════════════════════════════════════╝"
    echo ""
}

# --- Main ---
main() {
    parse_args "$@"
    validate

    if [[ "$DRY_RUN" == true ]]; then
        run_dry_run
        exit 0
    fi

    log_info "Deploying CampusVision AI to ${SSH_USER}@${SERVER}:${REMOTE_PATH}"

    pre_deploy_checks
    transfer_files
    remote_startup
    health_checks
    deployment_report

    log_ok "Deployment complete!"
}

main "$@"
