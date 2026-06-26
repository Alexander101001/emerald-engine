#!/bin/bash
set -euo pipefail

ENGINE_DIR="${ENGINE_DIR:-/tmp/opencode/emerald-engine}"
SAAS_REPO="${SAAS_REPO:-https://github.com/Alexander101001/emerald-saas.git}"
SAAS_DIR="${SAAS_DIR:-/root/emerald-saas}"
OH_REPO="${OH_REPO:-https://github.com/All-Hands-AI/OpenHands.git}"
OH_DIR="${OH_DIR:-/opt/openhands}"
OLLAMA_ENDPOINT="${OLLAMA_ENDPOINT:-http://ollama_service:11434}"
NETWORK_NAME="${NETWORK_NAME:-emerald_network}"
LOG_FILE="${LOG_FILE:-/var/log/emerald-init-core.log}"
PYTHONPATH_EXTRA="/data/data/com.termux/files/usr/lib/python3.13/site-packages"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC}  $1"; echo "[$(date +%T)] [INFO] $1" >> "$LOG_FILE"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $1"; echo "[$(date +%T)] [WARN] $1" >> "$LOG_FILE"; }
fail()  { echo -e "${RED}[FAIL]${NC} $1"; echo "[$(date +%T)] [FAIL] $1" >> "$LOG_FILE"; exit 1; }
step()  { echo ""; echo -e "${CYAN}==>${NC} $1"; }

trap 'fail "Script interrupted at line $LINENO"' ERR
mkdir -p "$(dirname "$LOG_FILE")"
: > "$LOG_FILE"

check_prerequisites() {
    step "[1/8] Checking prerequisites"
    local missing=0
    for cmd in docker git curl python3 node npm; do
        if ! command -v "$cmd" &>/dev/null 2>&1; then
            warn "Missing: $cmd"
            missing=$((missing+1))
        fi
    done
    [ "$missing" -eq 0 ] && info "All prerequisites satisfied." || warn "$missing prerequisite(s) missing."
}

ensure_docker_network() {
    step "[2/8] Ensuring shared Docker network"
    if docker network inspect "$NETWORK_NAME" &>/dev/null 2>&1; then
        info "Network $NETWORK_NAME exists."
    else
        docker network create --driver bridge "$NETWORK_NAME" 2>&1
        info "Network $NETWORK_NAME created."
    fi
}

ensure_ollama() {
    step "[3/8] Inspecting Ollama container state"
    local ollama_up=false
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "ollama_service"; then
        ollama_up=true
        info "Ollama container already running."
    fi
    if [ "$ollama_up" = false ] && [ -f "$SAAS_DIR/docker-compose.yml" ]; then
        info "Starting Ollama via docker-compose..."
        cd "$SAAS_DIR"
        docker compose up -d ollama 2>&1 || warn "docker-compose ollama start failed."
        ollama_up=true
    fi
    if [ "$ollama_up" = false ] && command -v ollama &>/dev/null; then
        info "Starting Ollama binary..."
        nohup ollama serve > /tmp/ollama-core.log 2>&1 &
        sleep 3
        ollama_up=true
    fi
    if [ "$ollama_up" = false ]; then
        warn "Ollama not available. Will retry after pulling emerald-saas."
        return 1
    fi
    for i in $(seq 1 12); do
        if curl -sf --connect-timeout 3 "$OLLAMA_ENDPOINT/api/tags" >/dev/null 2>&1; then
            info "Ollama reachable at $OLLAMA_ENDPOINT"
            return 0
        fi
        sleep 5
    done
    warn "Ollama health check did not pass within 60s."
    return 1
}

ensure_openhands() {
    step "[4/8] Inspecting OpenHands container state"
    local oh_up=false
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "openhands"; then
        oh_up=true
        info "OpenHands container already running."
        return 0
    fi
    if [ -d "$OH_DIR" ] && [ -f "$OH_DIR/package.json" ]; then
        info "OpenHands source found at $OH_DIR"
        if [ -f "$OH_DIR/.env" ]; then
            info "OpenHands .env found."
            oh_up=true
        fi
    fi
    if [ "$oh_up" = false ]; then
        info "OpenHands not found. Cloning $OH_REPO ..."
        mkdir -p "$(dirname "$OH_DIR")"
        git clone --depth 1 "$OH_REPO" "$OH_DIR" 2>&1
        cd "$OH_DIR"
        npm install 2>&1 || warn "npm install encountered warnings."
    fi
    local env_file="$OH_DIR/.env"
    cat > "$env_file" <<ENVEOF
LLM_MODEL=qwen2.5-coder:1.5b
LLM_BASE_URL=$OLLAMA_ENDPOINT/v1
LLM_API_KEY=ollama
LLM_EMBEDDING_MODEL=nomic-embed-text
WORKSPACE_MOUNT_PATH=/workspace
LOG_LEVEL=info
ENVEOF
    chmod 600 "$env_file"
    info "OpenHands linked to Ollama at $OLLAMA_ENDPOINT"
    if command -v docker &>/dev/null && [ -f "$OH_DIR/Dockerfile" ]; then
        info "Building OpenHands Docker image..."
        docker build -t openhands:latest "$OH_DIR" 2>&1 || warn "Docker build failed."
        docker rm -f openhands 2>/dev/null || true
        docker run -d --name openhands --restart unless-stopped \
            -p 3001:3000 \
            -v "$OH_DIR:/app" \
            -v openhands_workspace:/workspace \
            --env-file "$env_file" \
            --network "$NETWORK_NAME" \
            openhands:latest 2>&1 || warn "Docker run failed."
        info "OpenHands container launched."
    else
        info "Starting OpenHands via npm..."
        cd "$OH_DIR"
        nohup npm start > /var/log/openhands-core.log 2>&1 &
        info "OpenHands npm started (PID: $!)."
    fi
}

ensure_saas() {
    step "[5/8] Ensuring emerald-saas deployment"
    if [ -d "$SAAS_DIR/.git" ]; then
        info "SaaS repository found. Pulling latest..."
        cd "$SAAS_DIR"
        git pull 2>&1 || warn "git pull failed."
    else
        info "Cloning emerald-saas from $SAAS_REPO ..."
        git clone --depth 1 "$SAAS_REPO" "$SAAS_DIR" 2>&1
    fi
    if [ -f "$SAAS_DIR/docker-compose.yml" ]; then
        info "docker-compose.yml found. Connecting to network..."
        cd "$SAAS_DIR"
        docker compose up -d 2>&1 || warn "docker-compose up failed."
        docker network connect "$NETWORK_NAME" saas_main_app 2>/dev/null || true
        docker network connect "$NETWORK_NAME" ollama_service 2>/dev/null || true
        info "SaaS containers attached to $NETWORK_NAME"
    fi
    if [ -f "$SAAS_DIR/app/index.js" ]; then
        info "SaaS Node app entry point verified."
    fi
}

ensure_engine() {
    step "[6/8] Verifying emerald-engine readiness"
    if [ ! -f "$ENGINE_DIR/app.py" ]; then
        fail "Engine directory missing app.py at $ENGINE_DIR"
    fi
    info "Engine core present."
    for f in orchestrator.py crypto_vault.py telemetry.py opcode_base.py; do
        [ -f "$ENGINE_DIR/$f" ] || warn "Missing engine file: $f"
    done
    if [ -f "$ENGINE_DIR/requirements.txt" ]; then
        PYTHONPATH="$PYTHONPATH_EXTRA:$PYTHONPATH" pip3 install -r "$ENGINE_DIR/requirements.txt" --quiet 2>&1 || warn "pip install had issues."
        info "Python dependencies installed."
    fi
    if [ ! -f "$ENGINE_DIR/.env" ]; then
        cat > "$ENGINE_DIR/.env" <<ENVSIMPLE
EMERALD_MASTER_SECURE_KEY=dev-mode-insecure-key
OLLAMA_HOST=$OLLAMA_ENDPOINT
HARVESTER_MODEL=qwen2.5-coder:1.5b
PYTHONPATH=$PYTHONPATH_EXTRA
LOG_LEVEL=INFO
ENVSIMPLE
        info "Minimal .env created."
    fi
}

start_engine() {
    step "[7/8] Starting Emerald Engine orchestrator"
    local pid_file="/tmp/emerald-engine.pid"
    if [ -f "$pid_file" ] && kill -0 "$(cat "$pid_file")" 2>/dev/null; then
        info "Engine already running (PID: $(cat "$pid_file"))."
        return 0
    fi
    cd "$ENGINE_DIR"
    local key
    key=$(grep EMERALD_MASTER_SECURE_KEY .env 2>/dev/null | cut -d= -f2)
    PYTHONPATH="$PYTHONPATH_EXTRA:$PYTHONPATH" \
    EMERALD_MASTER_SECURE_KEY="$key" \
    nohup python3 app.py > /var/log/emerald-engine-core.log 2>&1 &
    local pid=$!
    echo "$pid" > "$pid_file"
    info "Engine started (PID: $pid)"
    sleep 5
    if curl -sf --connect-timeout 5 http://localhost:7860/health >/dev/null 2>&1; then
        info "Engine health check passed on :7860"
    else
        warn "Engine health check did not pass immediately."
    fi
}

feed_workspace_webhook() {
    step "[8/8] Feeding workspace params to initialization webhook"
    local webhook_url="${WEBHOOK_URL:-http://localhost:7860/api/telemetry}"
    local payload
    payload=$(cat <<PAYLOADEOF
{
    "event": "core_init_complete",
    "timestamp": $(date +%s),
    "components": {
        "engine": "$ENGINE_DIR",
        "saas": "$SAAS_DIR",
        "openhands": "$OH_DIR",
        "ollama_endpoint": "$OLLAMA_ENDPOINT",
        "network": "$NETWORK_NAME"
    },
    "status": "provisioned",
    "agent_count": 66
}
PAYLOADEOF
    )
    if curl -sf -X POST -H "Content-Type: application/json" -d "$payload" "$webhook_url" >/dev/null 2>&1; then
        info "Workspace parameters fed to $webhook_url"
    else
        warn "Webhook POST to $webhook_url failed (non-critical)."
    fi
}

main() {
    echo "============================================================"
    echo "  Emerald Engine - Autonomous Core Initialization"
    echo "  Target: Termux / Mobile Docker Environment"
    echo "============================================================"
    check_prerequisites
    ensure_docker_network
    ensure_ollama
    ensure_openhands
    ensure_saas
    ensure_engine
    start_engine
    feed_workspace_webhook
    echo ""
    echo "============================================================"
    echo "  Core initialization complete."
    echo "  Engine    : http://localhost:7860/health"
    echo "  Ollama    : $OLLAMA_ENDPOINT/api/tags"
    echo "  OpenHands : http://localhost:3001"
    echo "  SaaS      : $SAAS_DIR"
    echo "  Network   : $NETWORK_NAME"
    echo "============================================================"
}

main
