#!/usr/bin/env bash
set -euo pipefail

CLOUDFLARE_TOKEN="${CLOUDFLARE_TUNNEL_TOKEN:-}"
CLOUDFLARE_VERSION="${CLOUDFLARE_VERSION:-2025.2.0}"
ARCH="$(uname -m)"
case "$ARCH" in
    aarch64|arm64)  ARCH_SUFFIX="arm64" ;;
    x86_64|amd64)   ARCH_SUFFIX="amd64" ;;
    *)              echo "[cloudflared] Unsupported architecture: $ARCH"; exit 1 ;;
esac

if [ -z "$CLOUDFLARE_TOKEN" ]; then
    echo "[cloudflared] CLOUDFLARE_TUNNEL_TOKEN not set. Tunnel disabled."
    exit 0
fi

if ! command -v cloudflared &>/dev/null; then
    echo "[cloudflared] Installing cloudflared ${CLOUDFLARE_VERSION} (${ARCH_SUFFIX})..."
    curl -sL "https://github.com/cloudflare/cloudflared/releases/download/${CLOUDFLARE_VERSION}/cloudflared-linux-${ARCH_SUFFIX}" \
        -o /usr/local/bin/cloudflared
    chmod +x /usr/local/bin/cloudflared
fi

echo "[cloudflared] Starting tunnel to Cloudflare..."
exec cloudflared tunnel --no-autoupdate run --token "$CLOUDFLARE_TOKEN"
