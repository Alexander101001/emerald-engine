#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENGINE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
SAAS_DIR="${SAAS_DIR:-/root/emerald-saas}"
TARGET_AGENT="${1:-agent_01}"
OLLAMA_MODEL="${2:-qwen2.5-coder:1.5b}"
LOG_FILE="${LOG_FILE:-/var/log/emerald-activation.log}"
ENV_FILE="${ENV_FILE:-$ENGINE_DIR/.env}"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
log()     { echo -e "${CYAN}[$(date +%H:%M:%S)]${NC} $1"; echo "[$(date +%T)] $1" >> "$LOG_FILE"; }
ok()      { log "${GREEN}OK${NC}  $1"; }
warn()    { log "${YELLOW}WARN${NC} $1"; }
fail()    { log "${RED}FAIL${NC} $1"; exit 1; }

echo "============================================================"
echo "  Emerald Engine - Full Autonomous Activation"
echo "  Target Agent: $TARGET_AGENT"
echo "  Ollama Model: $OLLAMA_MODEL"
echo "============================================================"

mkdir -p "$(dirname "$LOG_FILE")"
: > "$LOG_FILE"

step_scan_engine() {
    log "Scanning emerald-engine ..."
    local missing=0
    local required=(
        app.py orchestrator.py autonomous_hunter.py hacker_bot.py
        cloud_manager.py platform_integration.py crypto_vault.py
        telemetry.py opcode_base.py self_improve_agent.py
        agent_chat_interpreter.py agent_code_evaluator.py
        agent_code_synthesizer.py agent_dashboard_compiler.py
        agent_devsecops.py agent_git_lifecycle.py agent_harvester.py
        agent_hf_sync.py agent_identity_manager.py agent_qa.py
        agent_runner_grid.py agent_software_architect.py
        agent_state_relay.py agent_trend_scouter.py
    )
    for f in "${required[@]}"; do
        if [ ! -f "$ENGINE_DIR/$f" ]; then
            warn "Missing engine file: $f"
            missing=$((missing+1))
        fi
    done
    if [ "$missing" -eq 0 ]; then
        ok "All $required engine files present."
    else
        warn "$missing engine file(s) missing."
    fi
    log "Engine directory size: $(du -sh "$ENGINE_DIR" 2>/dev/null | cut -f1)"
}

step_scan_saas() {
    log "Scanning emerald-saas ..."
    if [ -d "$SAAS_DIR" ]; then
        ok "SaaS directory found."
        if [ -f "$SAAS_DIR/app/index.js" ]; then
            ok "SaaS app entry point found."
        else
            warn "SaaS app/index.js missing."
        fi
        if [ -f "$SAAS_DIR/docker-compose.yml" ]; then
            ok "SaaS docker-compose.yml found."
        else
            warn "SaaS docker-compose.yml missing."
        fi
    else
        warn "SaaS directory not found at $SAAS_DIR. Cloning..."
        git clone --depth 1 https://github.com/Alexander101001/emerald-saas.git "$SAAS_DIR" 2>&1
    fi
    log "SaaS directory size: $(du -sh "$SAAS_DIR" 2>/dev/null | cut -f1)"
}

step_detect_missing_code() {
    log "Detecting and patching missing code files..."
    local prompts_dir="$ENGINE_DIR/.aegis/prompts"
    if [ -f "$prompts_dir/$TARGET_AGENT.txt" ]; then
        local prompt_size
        prompt_size=$(wc -c < "$prompts_dir/$TARGET_AGENT.txt")
        ok "Prompt for $TARGET_AGENT found ($prompt_size bytes)."
    else
        warn "No prompt found for $TARGET_AGENT. Creating stub..."
        mkdir -p "$prompts_dir"
        echo "# Agent $TARGET_AGENT - Autonomous prompt stub" > "$prompts_dir/$TARGET_AGENT.txt"
    fi
    if [ -f "$ENGINE_DIR/requirements.txt" ]; then
        local unmet=0
        while IFS= read -r dep; do
            dep_name=$(echo "$dep" | cut -d'>' -f1 | cut -d'=' -f1 | cut -d'~' -f1 | xargs)
            if [ -n "$dep_name" ]; then
                if ! python3 -c "import $dep_name" 2>/dev/null; then
                    unmet=$((unmet+1))
                fi
            fi
        done < "$ENGINE_DIR/requirements.txt"
        if [ "$unmet" -gt 0 ]; then
            warn "$unmet unmet Python dependencies. Installing..."
            pip3 install -r "$ENGINE_DIR/requirements.txt" 2>&1
        else
            ok "All Python dependencies satisfied."
        fi
    fi
}

step_verify_network() {
    log "Verifying network stability..."
    local targets=(
        "localhost:7860"
        "localhost:11434"
        "github.com"
    )
    local failures=0
    for target in "${targets[@]}"; do
        if curl -sf --connect-timeout 5 "http://$target" >/dev/null 2>&1; then
            ok "Reachable: $target"
        else
            warn "Unreachable: $target (may be expected)"
            failures=$((failures+1))
        fi
    done
    if curl -sf --connect-timeout 5 http://localhost:11434/api/tags >/dev/null 2>&1; then
        ok "Ollama API responding."
    else
        warn "Ollama API not responding."
    fi
    log "Network check complete ($failures failures)."
}

step_verify_env() {
    log "Verifying environment configuration..."
    if [ -f "$ENV_FILE" ]; then
        ok "Engine .env file found."
    else
        warn "No .env file found. Creating from .env.example..."
        if [ -f "$ENGINE_DIR/.env.example" ]; then
            cp "$ENGINE_DIR/.env.example" "$ENV_FILE"
            ok ".env created from example."
        else
            warn "No .env.example found either. Creating minimal .env..."
            cat > "$ENV_FILE" <<ENVSIMPLE
EMERALD_MASTER_SECURE_KEY=dev-mode-insecure-key
OLLAMA_HOST=http://localhost:11434
HARVESTER_MODEL=$OLLAMA_MODEL
LOG_LEVEL=INFO
ENVSIMPLE
        fi
    fi
    set -a; source "$ENV_FILE"; set +a
    ok "Environment loaded from $ENV_FILE"
}

step_start_services() {
    log "Starting core services..."
    local port_7860_free
    port_7860_free=true
    if lsof -i :7860 >/dev/null 2>&1; then
        port_7860_free=false
        ok "Emerald Engine already running on :7860"
    fi
    if [ "$port_7860_free" = true ]; then
        if [ -f "$ENGINE_DIR/app.py" ]; then
            PYTHONPATH="/data/data/com.termux/files/usr/lib/python3.13/site-packages:$PYTHONPATH" \
            EMERALD_MASTER_SECURE_KEY=$(grep EMERALD_MASTER_SECURE_KEY "$ENV_FILE" 2>/dev/null | cut -d= -f2) \
            nohup python3 "$ENGINE_DIR/app.py" > /tmp/emerald-activation.log 2>&1 &
            local pid=$!
            ok "Emerald Engine started (PID: $pid)"
            sleep 3
        else
            fail "app.py not found!"
        fi
    fi
    local ollama_running=false
    if curl -sf http://localhost:11434/api/tags >/dev/null 2>&1; then
        ollama_running=true
        ok "Ollama already running."
    fi
    if [ "$ollama_running" = false ] && command -v ollama &>/dev/null; then
        ollama serve > /tmp/ollama-activation.log 2>&1 &
        ok "Ollama started."
        sleep 3
    fi
}

step_verify_lifecycle() {
    log "Verifying overall system lifecycle..."
    sleep 5
    local engine_ok=false
    if curl -sf --connect-timeout 5 http://localhost:7860/health >/dev/null 2>&1; then
        engine_ok=true
        ok "Emerald Engine health check passed."
    fi
    local ollama_ok=false
    if curl -sf --connect-timeout 5 http://localhost:11434/api/tags >/dev/null 2>&1; then
        ollama_ok=true
        ok "Ollama health check passed."
    fi
    log "Engine: $engine_ok | Ollama: $ollama_ok"
}

run_post_activation_hooks() {
    log "Running post-activation hooks..."
    local hooks_dir="$ENGINE_DIR/scripts/hooks"
    if [ -d "$hooks_dir" ]; then
        for hook in "$hooks_dir"/*.sh; do
            if [ -x "$hook" ]; then
                log "Executing hook: $(basename "$hook")"
                bash "$hook" || warn "Hook $(basename "$hook") exited with $?"
            fi
        done
    else
        ok "No post-activation hooks to run."
    fi
}

main() {
    step_scan_engine
    step_scan_saas
    step_detect_missing_code
    step_verify_network
    step_verify_env
    step_start_services
    step_verify_lifecycle
    run_post_activation_hooks
    echo ""
    echo "============================================================"
    echo "  Activation complete."
    echo "  Emerald Engine  :7860/health"
    echo "  Ollama          :11434/api/tags"
    echo "  Agent           $TARGET_AGENT"
    echo "  Model           $OLLAMA_MODEL"
    echo "============================================================"
}

main
