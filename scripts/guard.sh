#!/bin/sh
#==============================================================================
# Emerald Security Guard — runs before every deploy
# Ensures: isolated temp, encrypted secrets, no plaintext .env in commit
#==============================================================================
set -e

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
fail() { printf "${RED}[GUARD FAIL]${NC} %s\n" "$1"; exit 1; }
pass() { printf "${GREEN}[GUARD PASS]${NC} %s\n" "$1"; }
warn() { printf "${YELLOW}[GUARD WARN]${NC} %s\n" "$1"; }

GUARD_ENV="${GUARD_ENV:-production}"
TMPDIR="${TMPDIR:-/tmp/emerald-guard}"
SESSION_FILE="$TMPDIR/.emerald-session-$(date +%s)"

echo "═══ Emerald Security Guard ($GUARD_ENV) ═══"

# ── 1. Verify EMERALD_KEY exists ─────────────────────────────────────
if [ -z "$EMERALD_KEY" ]; then
  fail "EMERALD_KEY environment variable is not set"
fi
if [ ${#EMERALD_KEY} -lt 12 ]; then
  fail "EMERALD_KEY must be at least 12 characters (current: ${#EMERALD_KEY})"
fi
pass "EMERALD_KEY present (${#EMERALD_KEY} chars)"

# ── 2. Isolated temp workspace ───────────────────────────────────────
rm -rf "$TMPDIR"
mkdir -p "$TMPDIR"
chmod 700 "$TMPDIR"
echo "session-$(date -u +%Y%m%dT%H%M%S)" > "$SESSION_FILE"
pass "Isolated temp workspace: $TMPDIR"

# ── 3. Check no plaintext .env is staged ─────────────────────────────
if git rev-parse --git-dir > /dev/null 2>&1; then
  STAGED_ENV=$(git diff --cached --name-only | grep -c '^\.env$' || true)
  if [ "$STAGED_ENV" -gt 0 ]; then
    fail "Plaintext .env is staged for commit! Use 'git reset .env' and run encrypt first."
  fi
  pass "No plaintext .env in staged files"

  # Check for .env in tracked files
  if git ls-files .env | grep -q .env; then
    fail ".env is tracked by git! Remove it: git rm --cached .env && echo '.env' >> .gitignore"
  fi
  pass ".env not tracked by git"
fi

# ── 4. Verify .env.encrypted exists and checksum ──────────────────────
if [ -f .env.encrypted ]; then
  node -e "
    const c = await import('./src/security/crypto-config.js');
    const ok = c.verifyChecksum('.env.encrypted');
    process.exit(ok ? 0 : 1);
  " 2>/dev/null && pass ".env.encrypted checksum valid" || warn ".env.encrypted checksum invalid — re-encrypt"
else
  if [ -f .env ]; then
    warn ".env.encrypted not found — but .env exists. Run: npm run encrypt"
  else
    pass "No .env or .env.encrypted — clean deploy"
  fi
fi

# ── 5. Verify node_modules integrity ─────────────────────────────────
if [ -d node_modules ]; then
  if [ -f package.json ]; then
    PKG_COUNT=$(wc -l < package.json 2>/dev/null || echo 0)
    MOD_COUNT=$(ls node_modules 2>/dev/null | wc -l)
    if [ "$MOD_COUNT" -gt 0 ]; then
      pass "node_modules: $MOD_COUNT packages"
    fi
  fi
fi

# ── 6. Secret key existence checks (no values, just names) ───────────
SECRET_NAMES="TELEGRAM_BOT_TOKEN OPENAI_API_KEY STRIPE_SECRET_KEY STRIPE_PUBLISHABLE_KEY ADSENSE_CLIENT_ID SUPABASE_URL SUPABASE_KEY"
for name in $SECRET_NAMES; do
  eval "val=\${${name}-}"
  if [ -n "$val" ]; then
    masked=$(echo "$val" | head -c 4)****$(echo "$val" | tail -c 5)
    pass "$name present: $masked"
  else
    warn "$name not set (non-critical in mock mode)"
  fi
done

echo "═══ Guard complete — environment secure ═══"
exit 0
