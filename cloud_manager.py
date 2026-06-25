import asyncio
import json
import logging
import os
import time
from copy import deepcopy
from pathlib import Path

import aiohttp
import requests

from crypto_vault import EmeraldCryptoVault
from opcode_base import ParadigmAgentBase

SENTINEL_URL = os.getenv("SENTINEL_URL", "http://localhost:8443")
SENTINEL_AUTH_TOKEN = os.getenv("SENTINEL_AUTH_TOKEN", "")
CLOUD_INTERVAL = int(os.getenv("CLOUD_MANAGER_INTERVAL", "600"))


class CloudResourceManager(ParadigmAgentBase):
    def __init__(self, telemetry=None):
        super().__init__(telemetry=telemetry)
        self.vault = EmeraldCryptoVault()
        self.registry = {}
        self.quota_limit_percentage = 80.0
        self._session = None

    async def _get_session(self):
        if self._session is None or self._session.closed:
            self._session = aiohttp.ClientSession()
        return self._session

    def add_target_platform(self, platform_name: str, endpoint_url: str, api_key: str):
        encrypted_key = self.vault.encrypt_data_payload(api_key)
        niche = self._niche_for_type(platform_name)
        if not self._apply_reasoning_rules(platform_name, niche):
            logging.warning(f"[CLOUD] Platform {platform_name} failed 161-rule verification")
            return False
        self.registry[platform_name] = {
            "endpoint": endpoint_url,
            "encrypted_key": encrypted_key,
            "status": "active",
            "allocated_resources": 0,
            "niche": niche,
        }
        self._emit_telemetry("platform_added", platform=platform_name, niche=niche)
        return True

    def _resolve_api_key(self, platform_name: str) -> str:
        platform = self.registry[platform_name]
        encrypted = platform["encrypted_key"]
        if SENTINEL_URL and SENTINEL_AUTH_TOKEN:
            try:
                resp = requests.post(
                    f"{SENTINEL_URL}/decrypt",
                    json={"data": encrypted.hex()},
                    headers={"X-Sentinel-Auth": SENTINEL_AUTH_TOKEN},
                    timeout=10,
                )
                if resp.status_code == 200:
                    return resp.json()["data"]
            except Exception as e:
                logging.warning(f"Sentinel decrypt failed, falling back to local: {e}")
        return self.vault.decrypt_data_payload(encrypted)

    async def verify_free_tier_status(self, platform_name: str) -> bool:
        if platform_name not in self.registry:
            return False
        platform = self.registry[platform_name]
        api_key = None
        try:
            api_key = self._resolve_api_key(platform_name)
            session = await self._get_session()
            async with session.get(
                f"{platform['endpoint']}/user/quota",
                headers={"Authorization": f"Bearer {api_key}"},
                timeout=aiohttp.ClientTimeout(total=10),
            ) as resp:
                if resp.status == 200:
                    usage_data = await resp.json()
                    current_usage = usage_data.get("usage_percent", 0.0)
                    return current_usage < self.quota_limit_percentage
        except Exception:
            return True
        finally:
            if api_key:
                api_key = None
        return True

    async def allocate_new_cluster(self, platform_name: str, project_payload: dict) -> bool:
        if not self._apply_reasoning_rules(platform_name,
                                            self.registry.get(platform_name, {}).get("niche", "b2b_saas")):
            return False
        if not await self.verify_free_tier_status(platform_name):
            return False
        platform = self.registry[platform_name]
        api_key = None
        try:
            api_key = self._resolve_api_key(platform_name)
            session = await self._get_session()
            async with session.post(
                f"{platform['endpoint']}/deployments",
                json=project_payload,
                headers={"Authorization": f"Bearer {api_key}"},
                timeout=aiohttp.ClientTimeout(total=15),
            ) as resp:
                if resp.status in (200, 201):
                    platform["allocated_resources"] += 1
                    self._emit_telemetry("cluster_allocated", platform=platform_name,
                                          resources=platform["allocated_resources"])
                    return True
        except Exception:
            return False
        finally:
            if api_key:
                api_key = None
        return False

    # ── Paradigm 1: 5-channel parallel allocation ─────────────────────────

    async def _channel_quota_check(self, platforms: list) -> list:
        results = []
        tasks = []
        for p in platforms:
            tasks.append(self.verify_free_tier_status(p))
        statuses = await asyncio.gather(*tasks)
        for i, p in enumerate(platforms):
            if statuses[i]:
                results.append(p)
        return results

    async def _channel_deploy(self, platform: str, payload: dict) -> bool:
        return await self.allocate_new_cluster(platform, payload)

    async def _channel_bulk_allocate(self, platform_pairs: list) -> dict:
        results = {}
        tasks = []
        for platform, payload in platform_pairs:
            tasks.append(self._channel_deploy(platform, payload))
        statuses = await asyncio.gather(*tasks)
        for i, (platform, _) in enumerate(platform_pairs):
            results[platform] = statuses[i]
        return results

    async def _channel_health_check(self) -> dict:
        statuses = {}
        for pname in list(self.registry.keys()):
            statuses[pname] = await self.verify_free_tier_status(pname)
        return statuses

    async def _channel_telemetry_report(self) -> dict:
        report = {
            "total_platforms": len(self.registry),
            "total_allocated": sum(p.get("allocated_resources", 0) for p in self.registry.values()),
            "active": sum(1 for p in self.registry.values() if p.get("status") == "active"),
            "timestamp": time.time(),
        }
        self._emit_telemetry("cloud_manager_report", **report)
        return report

    # ── Paradigm 5: Hot-reload daemon ─────────────────────────────────────

    async def bulk_allocate_parallel(self, deploy_queue: list) -> dict:
        ranked = self._bm25_rank(
            deploy_queue,
            "cloud deployment infrastructure compute scalable",
            key=lambda x: x[0],
        )
        return await self._channel_bulk_allocate(ranked)

    async def run_management_cycle(self):
        logging.info("=== Cloud Manager Cycle ===")
        start = time.time()
        await self._channel_telemetry_report()
        health = await self._channel_health_check()
        active = sum(1 for v in health.values() if v)
        logging.info(f"Cloud status: {active}/{len(health)} platforms within quota")
        self._emit_telemetry("cycle_complete", elapsed=time.time() - start,
                              active_platforms=active)

    async def execution_loop(self):
        logging.info("Cloud Manager activated — 5-paradigm OP_CODE resource orchestration")
        await self._hot_daemon_loop(self.run_management_cycle, CLOUD_INTERVAL)


async def run_cloud_loop(telemetry=None):
    agent = CloudResourceManager(telemetry=telemetry)
    await agent.execution_loop()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [CLOUD] %(levelname)s %(message)s")
    asyncio.run(run_cloud_loop())
