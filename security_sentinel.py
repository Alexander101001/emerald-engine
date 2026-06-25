import asyncio
import json
import logging
import os
import time
from pathlib import Path

from aiohttp import web

from crypto_vault import EmeraldCryptoVault, SecureVaultFileManager
from opcode_base import ParadigmAgentBase

SENTINEL_PORT = int(os.getenv("SENTINEL_PORT", "8443"))
SENTINEL_AUTH_TOKEN = os.getenv("SENTINEL_AUTH_TOKEN", "")
ENCRYPTED_CONFIG_PATH = Path(os.getenv("ENCRYPTED_CONFIG_PATH", "/etc/sentinel/config.encrypted"))
VAULT_BACKUP_PATH = Path(os.getenv("VAULT_BACKUP_PATH", "/var/lib/sentinel/backup"))


class VaultSentinel(ParadigmAgentBase):
    def __init__(self, telemetry=None):
        super().__init__(telemetry=telemetry)
        self.vault = EmeraldCryptoVault()
        self.file_manager = SecureVaultFileManager(self.vault)
        self._start_time = time.time()
        self._decrypt_count = 0
        self._encrypt_count = 0
        self._health_status = "operational"
        self._decrypted_cache = {}
        self._lock = asyncio.Lock()
        VAULT_BACKUP_PATH.mkdir(parents=True, exist_ok=True)
        self._app = None
        self._runner = None

    def _check_auth(self, request) -> bool:
        if not SENTINEL_AUTH_TOKEN:
            return True
        return request.headers.get("X-Sentinel-Auth", "") == SENTINEL_AUTH_TOKEN

    def _verify_with_rules(self, entity: str) -> bool:
        return self._apply_reasoning_rules(entity, "b2b_saas")

    # ── Paradigm 2: 161-rule verified endpoints ──────────────────────────

    async def handle_health(self, request):
        if not self._check_auth(request):
            return web.json_response({"status": "unauthorized"}, status=403)
        return web.json_response({
            "status": self._health_status,
            "uptime": round(time.time() - self._start_time, 2),
            "decrypt_ops": self._decrypt_count,
            "encrypt_ops": self._encrypt_count,
            "opcode_paradigms": ["5_channel_parallel", "161_logic_rules",
                                  "niche_domain_map", "telemetry_ui", "hot_reload"],
        })

    async def handle_decrypt(self, request):
        if not self._check_auth(request):
            return web.json_response({"status": "unauthorized"}, status=403)
        try:
            body = await request.json()
            encrypted_hex = body.get("data")
            if not encrypted_hex:
                return web.json_response({"error": "missing data"}, status=400)
            entity_id = body.get("entity", encrypted_hex[:16])
            if not self._verify_with_rules(entity_id):
                return web.json_response({"error": "entity failed 161-rule verification"}, status=403)
            encrypted_bytes = bytes.fromhex(encrypted_hex)
            decrypted = self.vault.decrypt_data_payload(encrypted_bytes)
            self._decrypt_count += 1
            self._emit_telemetry("decrypt", entity=entity_id[:16])
            return web.json_response({"data": decrypted})
        except Exception as e:
            logging.error(f"Decrypt failed: {e}")
            return web.json_response({"error": str(e)}, status=500)

    async def handle_encrypt(self, request):
        if not self._check_auth(request):
            return web.json_response({"status": "unauthorized"}, status=403)
        try:
            body = await request.json()
            plaintext = body.get("data")
            if not plaintext:
                return web.json_response({"error": "missing data"}, status=400)
            entity_id = body.get("entity", plaintext[:16])
            if not self._verify_with_rules(entity_id):
                return web.json_response({"error": "entity failed 161-rule verification"}, status=403)
            encrypted = self.vault.encrypt_data_payload(plaintext)
            self._encrypt_count += 1
            self._emit_telemetry("encrypt", entity=entity_id[:16])
            return web.json_response({"data": encrypted.hex()})
        except Exception as e:
            logging.error(f"Encrypt failed: {e}")
            return web.json_response({"error": str(e)}, status=500)

    async def handle_decrypt_file(self, request):
        if not self._check_auth(request):
            return web.json_response({"status": "unauthorized"}, status=403)
        try:
            body = await request.json()
            file_path = body.get("path")
            if not file_path:
                return web.json_response({"error": "missing path"}, status=400)
            if not self._verify_with_rules(file_path):
                return web.json_response({"error": "path failed 161-rule verification"}, status=403)
            plaintext = self.file_manager.decrypt_file(file_path)
            self._decrypt_count += 1
            self._emit_telemetry("decrypt_file", path=file_path)
            return web.json_response({"data": plaintext})
        except Exception as e:
            logging.error(f"Decrypt file failed: {e}")
            return web.json_response({"error": str(e)}, status=500)

    async def handle_status(self, request):
        if not self._check_auth(request):
            return web.json_response({"status": "unauthorized"}, status=403)
        return web.json_response({
            "health": self._health_status,
            "uptime": round(time.time() - self._start_time, 2),
            "decrypt_ops": self._decrypt_count,
            "encrypt_ops": self._encrypt_count,
            "cache_entries": len(self._decrypted_cache),
            "version": "1.1.0-sentinel-opcode",
            "opcode_paradigms": ["5_channel_parallel", "161_logic_rules",
                                  "niche_domain_map", "telemetry_ui", "hot_reload"],
        })

    async def handle_telemetry(self, request):
        if not self._check_auth(request):
            return web.json_response({"status": "unauthorized"}, status=403)
        return web.json_response({
            "decrypt_count": self._decrypt_count,
            "encrypt_count": self._encrypt_count,
            "uptime": round(time.time() - self._start_time, 2),
            "status": self._health_status,
            "cache_size": len(self._decrypted_cache),
        })

    # ── Paradigm 5: Self-check with hot-reload ───────────────────────────

    async def _backup_encrypted_state(self):
        if ENCRYPTED_CONFIG_PATH.exists():
            backup_file = VAULT_BACKUP_PATH / f"config_backup_{int(time.time())}.encrypted"
            backup_file.write_bytes(ENCRYPTED_CONFIG_PATH.read_bytes())
            logging.info(f"Backed up encrypted config to {backup_file}")
            old_backups = sorted(VAULT_BACKUP_PATH.glob("config_backup_*.encrypted"))
            while len(old_backups) > 10:
                old_backups[0].unlink()
                old_backups.pop(0)

    async def _self_check_loop(self):
        while True:
            try:
                if ENCRYPTED_CONFIG_PATH.exists():
                    self.file_manager.decrypt_file(str(ENCRYPTED_CONFIG_PATH))
                    self._health_status = "operational"
                await self._backup_encrypted_state()
            except Exception as e:
                self._health_status = "degraded"
                self._emit_telemetry("self_check_failed", error=str(e))
                logging.error(f"Sentinel self-check failed: {e}")
            await asyncio.sleep(300)

    async def start(self):
        self._app = web.Application()
        self._app.router.add_get("/health", self.handle_health)
        self._app.router.add_post("/decrypt", self.handle_decrypt)
        self._app.router.add_post("/encrypt", self.handle_encrypt)
        self._app.router.add_post("/decrypt-file", self.handle_decrypt_file)
        self._app.router.add_get("/status", self.handle_status)
        self._app.router.add_get("/telemetry", self.handle_telemetry)
        self._runner = web.AppRunner(self._app)
        await self._runner.setup()
        site = web.TCPSite(self._runner, "0.0.0.0", SENTINEL_PORT)
        await site.start()
        logging.info(f"Vault Sentinel (OP_CODE) listening on :{SENTINEL_PORT}")
        await asyncio.gather(
            self._self_check_loop(),
        )


async def main():
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [SENTINEL] %(levelname)s %(message)s")
    logging.info("Initializing Emerald Vault Sentinel — OP_CODE 5-paradigm compliance")
    sentinel = VaultSentinel()
    await sentinel.start()

if __name__ == "__main__":
    asyncio.run(main())
