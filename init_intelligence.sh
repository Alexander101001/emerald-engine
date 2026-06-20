#!/usr/bin/env bash
set -e

# ============================================================
# init_intelligence.sh
# Lightweight Ollama + Qwen2.5 auto-installer
# Downloads the smallest Qwen model on every container boot.
# Designed to be called from supervisord or entrypoint.
# ============================================================

OLLAMA_VERSION="${OLLAMA_VERSION:-0.5.13}"
QWEEN_MODEL="${QWEEN_MODEL:-qwen2.5:0.5b}"
OLLAMA_BIN="/usr/local/bin/ollama"
OLLAMA_SERVICE="ollama"
OLLAMA_PORT="${OLLAMA_PORT:-11434}"
OLLAMA_HOST="http://127.0.0.1:${OLLAMA_PORT}"

log() { echo "[intelligence] $(date '+%Y-%m-%d %H:%M:%S') $*"; }

# ---- 1. Install Ollama binary if missing ----
if ! command -v ollama &>/dev/null; then
	log "Ollama not found — downloading v${OLLAMA_VERSION}..."
	curl -fsSL "https://github.com/ollama/ollama/releases/download/v${OLLAMA_VERSION}/ollama-linux-amd64" -o /tmp/ollama
	chmod +x /tmp/ollama
	mv /tmp/ollama "${OLLAMA_BIN}"
	log "Ollama installed at ${OLLAMA_BIN}"
else
	log "Ollama already installed at $(which ollama)"
fi

# ---- 2. Start Ollama server in background ----
if ! pgrep -x ollama &>/dev/null; then
	log "Starting Ollama server on port ${OLLAMA_PORT}..."
	OLLAMA_HOST="127.0.0.1:${OLLAMA_PORT}" nohup ollama serve >/tmp/ollama_server.log 2>&1 &
	sleep 3
	log "Ollama server PID: $(pgrep -x ollama)"
else
	log "Ollama server already running"
fi

# ---- 3. Wait for Ollama to be ready ----
for i in $(seq 1 30); do
	if curl -sf "${OLLAMA_HOST}/api/tags" >/dev/null 2>&1; then
		log "Ollama API ready after ${i}s"
		break
	fi
	sleep 1
done

# ---- 4. Pull the lightweight Qwen model ----
log "Pulling model ${QWEEN_MODEL}..."
ollama pull "${QWEEN_MODEL}" 2>&1 | while IFS= read -r line; do log "${line}"; done

# ---- 5. Verify ----
if ollama list 2>/dev/null | grep -q "qwen"; then
	log "Model ${QWEEN_MODEL} ready"
else
	log "WARNING: model pull may have failed — check /tmp/ollama_server.log"
fi

log "Intelligence layer initialized successfully"
