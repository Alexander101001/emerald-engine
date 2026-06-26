#!/bin/bash
set -euo pipefail

ENGINE_DIR="${ENGINE_DIR:-/tmp/opencode/emerald-engine}"
SAAS_DIR="${SAAS_DIR:-/root/emerald-saas}"
STATE_FILE="${STATE_FILE:-tmp/emerald_engine_state.json}"
STATE_FILE_ENC="${STATE_FILE_ENC:-tmp/emerald_engine_state.json.enc}"
DISK_THRESHOLD="${DISK_THRESHOLD:-80}"
MEM_THRESHOLD="${MEM_THRESHOLD:-85}"
SCALE_SERVICE="${SCALE_SERVICE:-saas_app}"
SCALE_REPLICAS="${SCALE_REPLICAS:-2}"
POLL_INTERVAL="${POLL_INTERVAL:-30}"
LOG_FILE="${LOG_FILE:-/var/log/emerald-disk-monitor.log}"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
info()  { echo -e "${GREEN}[$(date +%H:%M:%S)]${NC} $1"; echo "[$(date +%T)] [INFO] $1" >> "$LOG_FILE"; }
warn()  { echo -e "${YELLOW}[$(date +%H:%M:%S)]${NC} $1"; echo "[$(date +%T)] [WARN] $1" >> "$LOG_FILE"; }
alert() { echo -e "${RED}[$(date +%H:%M:%S)]${NC} $1"; echo "[$(date +%T)] [ALERT] $1" >> "$LOG_FILE"; }

mkdir -p "$(dirname "$LOG_FILE")"
: > "$LOG_FILE"

init_vars() {
    STATE_DIR="$(cd "$ENGINE_DIR" 2>/dev/null && pwd)"
    STATE_FULL="$STATE_DIR/$STATE_FILE"
    STATE_ENC_FULL="$STATE_DIR/$STATE_FILE_ENC"
    COMPOSE_FILE="$SAAS_DIR/docker-compose.yml"
}

get_disk_usage() {
    local target="${1:-/}"
    df -h "$target" 2>/dev/null | awk 'NR==2 {print $5}' | sed 's/%//'
}

get_mem_usage() {
    if command -v free &>/dev/null; then
        free | awk '/Mem:/ {printf "%.0f", $3/$2 * 100}'
    elif [ -f /proc/meminfo ]; then
        local total used
        total=$(awk '/MemTotal/ {print $2}' /proc/meminfo)
        used=$(awk '/MemAvailable/ {print $2}' /proc/meminfo)
        if [ -n "$total" ] && [ -n "$used" ] && [ "$total" -gt 0 ]; then
            echo $(( 100 - (used * 100 / total) ))
        else
            echo 0
        fi
    else
        echo 0
    fi
}

get_container_mem() {
    local container="$1"
    docker stats --no-stream --format '{{.MemPerc}}' "$container" 2>/dev/null | sed 's/%//' || echo 0
}

replicate_state() {
    local source="$1"
    local replica_label="$2"
    if [ ! -f "$source" ]; then
        warn "State file $source not found for replication."
        return 1
    fi
    local replica_dir="$ENGINE_DIR/tmp/replicas/$replica_label"
    mkdir -p "$replica_dir"
    cp "$source" "$replica_dir/"
    info "State replicated to $replica_dir/ ($(wc -c < "$source") bytes)"
}

scale_service() {
    local service="$1"
    local replicas="$2"
    if [ ! -f "$COMPOSE_FILE" ]; then
        warn "Compose file not found at $COMPOSE_FILE. Cannot scale."
        return 1
    fi
    cd "$SAAS_DIR"
    info "Scaling $service to $replicas replicas..."
    docker compose up -d --scale "$service=$replicas" --no-recreate 2>&1 || {
        warn "Scale command failed. Attempting direct docker-compose up..."
        docker compose up -d 2>&1 || true
    }
    local new_container
    new_container=$(docker ps --filter "name=${service}" --format '{{.Names}}' | tail -1)
    if [ -n "$new_container" ]; then
        docker network connect emerald_network "$new_container" 2>/dev/null || true
        info "New replica $new_container bridged to emerald_network"
        replicate_state "$STATE_FULL" "$new_container" || true
        replicate_state "$STATE_ENC_FULL" "${new_container}.enc" || true
    fi
}

check_disk_limits() {
    local disk_pct
    disk_pct=$(get_disk_usage /)
    info "Disk usage: ${disk_pct}% (threshold: ${DISK_THRESHOLD}%)"
    if [ "$disk_pct" -gt "$DISK_THRESHOLD" ]; then
        alert "DISK LIMIT REACHED: ${disk_pct}% > ${DISK_THRESHOLD}%"
        scale_service "$SCALE_SERVICE" "$SCALE_REPLICAS"
        return 0
    fi
    return 1
}

check_memory_limits() {
    local mem_pct
    mem_pct=$(get_mem_usage)
    info "Host memory: ${mem_pct}% (threshold: ${MEM_THRESHOLD}%)"
    if [ "$mem_pct" -gt "$MEM_THRESHOLD" ]; then
        alert "MEMORY LIMIT REACHED: ${mem_pct}% > ${MEM_THRESHOLD}%"
        scale_service "$SCALE_SERVICE" "$SCALE_REPLICAS"
        return 0
    fi
    for container in $(docker ps --format '{{.Names}}' 2>/dev/null); do
        local c_mem
        c_mem=$(get_container_mem "$container")
        if [ -n "$c_mem" ] && [ "$(echo "$c_mem > $MEM_THRESHOLD" | bc 2>/dev/null)" = "1" ]; then
            alert "Container $container memory at ${c_mem}%"
        fi
    done
    return 1
}

check_engine_health() {
    if curl -sf --connect-timeout 5 http://localhost:7860/health >/dev/null 2>&1; then
        return 0
    else
        warn "Engine health check failed. Attempting restart..."
        if [ -f "$ENGINE_DIR/app.py" ]; then
            cd "$ENGINE_DIR"
            local key
            key=$(grep EMERALD_MASTER_SECURE_KEY .env 2>/dev/null | cut -d= -f2)
            PYTHONPATH="/data/data/com.termux/files/usr/lib/python3.13/site-packages:$PYTHONPATH" \
            EMERALD_MASTER_SECURE_KEY="$key" \
            nohup python3 app.py > /var/log/emerald-engine-core.log 2>&1 &
            info "Engine restarted (PID: $!)."
        fi
        return 1
    fi
}

housekeeping() {
    local cutoff=$(( $(date +%s) - 86400 ))
    local rotated=0
    for f in "$STATE_DIR"/tmp/replicas/*/emerald_engine_state.json*; do
        if [ -f "$f" ] && [ "$(stat -c %Y "$f")" -lt "$cutoff" ]; then
            rm -f "$f"
            rotated=$((rotated+1))
        fi
    done
    if [ "$rotated" -gt 0 ]; then
        info "Rotated $rotated stale replica state file(s)."
    fi
}

main_loop() {
    info "Disk monitor started. Polling every ${POLL_INTERVAL}s."
    info "Disk threshold: ${DISK_THRESHOLD}% | Mem threshold: ${MEM_THRESHOLD}%"
    init_vars
    while true; do
        local triggered=false
        check_disk_limits && triggered=true
        check_memory_limits && triggered=true
        check_engine_health
        if [ "$triggered" = true ]; then
            alert "Safeguard triggered. Verifying state replication..."
            if [ -f "$STATE_FULL" ]; then
                replicate_state "$STATE_FULL" "safeguard_$(date +%s)" || true
            fi
            if [ -f "$STATE_ENC_FULL" ]; then
                replicate_state "$STATE_ENC_FULL" "safeguard_enc_$(date +%s)" || true
            fi
        fi
        housekeeping
        sleep "$POLL_INTERVAL"
    done
}

main_loop
