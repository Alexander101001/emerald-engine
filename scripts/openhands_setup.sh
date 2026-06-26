#!/bin/bash
set -euo pipefail

OH_DIR="${OH_DIR:-/opt/openhands}"
OH_REPO="${OH_REPO:-https://github.com/All-Hands-AI/OpenHands.git}"
OLLAMA_ENDPOINT="${OLLAMA_ENDPOINT:-http://ollama_service:11434}"
OH_BRANCH="${OH_BRANCH:-main}"
CONFIG_DIR="${CONFIG_DIR:-/etc/openhands}"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
log_info()  { echo -e "${GREEN}[INFO]${NC}  $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

verify_prerequisites() {
    log_info "Verifying prerequisites..."
    local missing=0
    for cmd in docker git curl node npm python3; do
        if ! command -v "$cmd" &>/dev/null 2>&1; then
            log_warn "Missing: $cmd"
            missing=$((missing+1))
        fi
    done
    if [ "$missing" -gt 0 ]; then
        log_error "$missing prerequisite(s) missing. Install them first."
        exit 1
    fi
    log_info "All prerequisites satisfied."
}

check_openhands_installed() {
    log_info "Checking if OpenHands is deployed..."
    if [ -d "$OH_DIR" ] && [ -f "$OH_DIR/package.json" ]; then
        log_info "OpenHands found at $OH_DIR"
        return 0
    fi
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -qi "openhands"; then
        log_info "OpenHands container is running."
        return 0
    fi
    return 1
}

fetch_and_build_openhands() {
    log_info "OpenHands not found. Fetching from $OH_REPO ..."
    mkdir -p "$OH_DIR"
    git clone --depth 1 --branch "$OH_BRANCH" "$OH_REPO" "$OH_DIR" 2>&1
    log_info "Clone complete. Installing dependencies..."
    cd "$OH_DIR"
    npm install 2>&1
    log_info "Dependencies installed."
    mkdir -p "$CONFIG_DIR"
}

configure_openhands_ollama() {
    log_info "Linking OpenHands to local Ollama at $OLLAMA_ENDPOINT ..."
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
    local config_file="$CONFIG_DIR/config.toml"
    cat > "$config_file" <<TOMLEOF
[llm]
model = "qwen2.5-coder:1.5b"
base_url = "$OLLAMA_ENDPOINT/v1"
api_key = "ollama"
temperature = 0.1
max_tokens = 4096

[sandbox]
container_image = "openhands-sandbox:latest"
timeout = 120

[workspace]
mount_path = "/workspace"
TOMLEOF
    chmod 600 "$config_file"
    log_info "OpenHands configuration written."
}

build_openhands_docker() {
    log_info "Building OpenHands Docker image..."
    cd "$OH_DIR"
    if [ -f "Dockerfile" ]; then
        docker build -t openhands:latest . 2>&1
        log_info "OpenHands image built."
    else
        log_warn "No Dockerfile found at $OH_DIR. Attempting npm build..."
        npm run build 2>&1 || log_warn "npm build encountered issues."
    fi
}

start_openhands_container() {
    log_info "Starting OpenHands container..."
    docker rm -f openhands 2>/dev/null || true
    docker run -d \
        --name openhands \
        --restart unless-stopped \
        -p 3001:3000 \
        -v "$OH_DIR:/app" \
        -v openhands_workspace:/workspace \
        --env-file "$OH_DIR/.env" \
        --network saas_network \
        openhands:latest 2>&1 || {
        log_warn "Docker start failed. Attempting direct npm start..."
        cd "$OH_DIR"
        nohup npm start > /var/log/openhands.log 2>&1 &
        log_info "OpenHands started via npm (PID: $!)."
    }
}

verify_openhands_health() {
    log_info "Verifying OpenHands health..."
    for i in $(seq 1 12); do
        if curl -sf http://localhost:3001/api/health >/dev/null 2>&1; then
            log_info "OpenHands is healthy (port 3001)."
            return 0
        fi
        sleep 5
    done
    log_warn "OpenHands health check did not pass within 60s."
    return 1
}

main() {
    echo "============================================================"
    echo "  OpenHands Verification & Lifecycle Setup"
    echo "============================================================"
    verify_prerequisites
    if check_openhands_installed; then
        log_info "OpenHands lifecycle: UPDATE"
        configure_openhands_ollama
        log_info "Configuration refreshed. Skipping rebuild."
    else
        log_info "OpenHands lifecycle: INSTALL"
        fetch_and_build_openhands
        configure_openhands_ollama
        build_openhands_docker
        start_openhands_container
        verify_openhands_health
    fi
    log_info "OpenHands setup complete."
}

main
