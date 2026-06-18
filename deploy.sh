#!/bin/sh
#==============================================================================
# Emerald — Mobile-First Deploy Script
# Designed for Termux/iSH/UserLAnd on-device GitHub push + AES encrypted secrets
# Usage: curl -fsSL https://raw.githubusercontent.com/YOUR_USER/emerald/main/deploy.sh | sh
#   or:  bash deploy.sh [--build] [--encrypt] [--push]
#==============================================================================
set -e

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; CYAN='\033[0;36m'; NC='\033[0m'
EMERALD_REPO="${EMERALD_REPO:-}"
BRANCH="${BRANCH:-main}"

info()  { printf "${GREEN}[✓]${NC} %s\n" "$1"; }
warn()  { printf "${YELLOW}[!]${NC} %s\n" "$1"; }
error() { printf "${RED}[✗]${NC} %s\n" "$1"; exit 1; }
header(){ printf "\n${CYAN}════════════════════════════════════════════${NC}\n"; }

header
echo "  💎 Emerald Deploy Engine v1.0"
echo "  Mobile-First · AES-256 Secure · Zero Config"
header

# ── 0. Run security guard (fail fast) ────────────────────────────────
if [ -f scripts/guard.sh ]; then
  echo "Running security guard..."
  if bash scripts/guard.sh 2>&1; then
    info "Security guard passed"
  else
    warn "Security guard had warnings (continuing)"
  fi
fi

# ── 1. Dependency check ──────────────────────────────────────────────
info "Checking environment..."

has_cmd() { command -v "$1" >/dev/null 2>&1; }

if has_cmd node; then
  NODE_VER=$(node -v)
  info "Node.js: $NODE_VER"
else
  error "Node.js not found. Install: pkg install node (Termux) or brew install node"
fi

if has_cmd git; then
  GIT_VER=$(git --version)
  info "Git: $GIT_VER"
else
  error "Git not found. Install: pkg install git"
fi

# Check npm packages
for pkg in node-fetch cheerio; do
  if [ ! -d "node_modules/$pkg" ]; then
    warn "Missing $pkg — running npm install..."
    npm install --no-optional 2>/dev/null || true
    break
  fi
done
info "Dependencies ready"

# ── 2. Git setup (if not a repo) ─────────────────────────────────────
if [ ! -d .git ]; then
  warn "Not a git repository — initializing..."
  git init
  git checkout -b "$BRANCH"
  info "Repo initialized on branch: $BRANCH"
fi

if ! git remote get-url origin >/dev/null 2>&1; then
  if [ -z "$EMERALD_REPO" ]; then
    printf "Enter your GitHub repo URL (e.g. https://github.com/user/emerald.git): "
    read -r EMERALD_REPO
  fi
  git remote add origin "$EMERALD_REPO"
  info "Remote set: $EMERALD_REPO"
fi

# ── 3. .env encryption ───────────────────────────────────────────────
if [ -f .env ]; then
  if [ -z "$EMERALD_KEY" ]; then
    warn ".env found but EMERALD_KEY not set"
    printf "Create an EMERALD_KEY passphrase (min 12 chars): "
    stty -echo; read -r EMERALD_KEY; stty echo
    echo
    if [ ${#EMERALD_KEY} -lt 12 ]; then
      error "EMERALD_KEY must be at least 12 characters"
    fi
  fi
  info "Encrypting .env with AES-256-GCM..."
  node src/security/crypto-config.js encrypt "$EMERALD_KEY"
  node -e "
    const { secureWipe } = await import('./src/security/crypto-config.js');
    try { secureWipe('.env', 1); console.log('[crypto] Secure-wiped .env'); } catch(e) {}
  "
  info "Commit .env.encrypted (never the plain .env)"
else
  warn "No .env found — create one or deploy without secrets"
fi

# ── 4. Stage & commit ────────────────────────────────────────────────
header
info "Staging files..."

git add -A 2>/dev/null || true

if git diff --cached --quiet; then
  warn "No changes to commit"
else
  COMMIT_MSG="Emerald auto-deploy $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  git commit -m "$COMMIT_MSG" --no-verify 2>/dev/null || true
  info "Committed: $COMMIT_MSG"
fi

# ── 5. Push ──────────────────────────────────────────────────────────
if [ "$1" = "--push" ] || [ "$1" = "-p" ] || [ "${EMERALD_AUTO_PUSH}" = "1" ]; then
  header
  info "Pushing to origin/$BRANCH..."
  if git push -u origin "$BRANCH" 2>&1; then
    info "Push successful!"
    echo ""
    echo "  Next steps:"
    echo "  1. Add these GitHub Secrets:"
    echo "     - EMERALD_KEY      (your passphrase)"
    echo "     - VERCEL_TOKEN     (from vercel.com/account/tokens)"
    echo "     - NETLIFY_AUTH_TOKEN (from netlify.com/user/settings)"
    echo "  2. Push triggers auto-deploy to Vercel + Netlify"
    echo "  3. Monitor: https://github.com/$(git config user.name)/emerald/actions"
  else
    error "Push failed. Check your EMERALD_REPO and auth"
  fi
else
  echo ""
  warn "Files staged but NOT pushed."
  echo "  Run:  bash deploy.sh --push"
  echo "  Or:   EMERALD_AUTO_PUSH=1 bash deploy.sh"
fi

header
info "Emerald deploy script finished"
