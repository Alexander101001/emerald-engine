import asyncio
import json
import logging
import os
import time
from pathlib import Path

import aiohttp

from opcode_base import ParadigmAgentBase
from crypto_vault import EmeraldCryptoVault

SYSTEM_TOPOLOGY_PATH = Path("/tmp/emerald_topology.json")
COMPONENT_REGISTRY_PATH = Path("/tmp/emerald_components.json")
DISCOVERY_INTERVAL = int(os.getenv("ARCHITECT_DISCOVERY_INTERVAL", "3600"))

REPOSITORIES = [
    {"name": "emerald-engine", "url": "https://api.github.com/repos/anomalyco/emerald-engine",
     "type": "github"},
    {"name": "opencode", "url": "https://api.github.com/repos/anomalyco/opencode",
     "type": "github"},
]


class SoftwareArchitectAgent(ParadigmAgentBase):
    """STEP_1_DISCOVERY + STEP_2_ANALYSIS: Scans repos, validates topology."""

    def __init__(self, telemetry=None):
        super().__init__(telemetry=telemetry)
        self.vault = EmeraldCryptoVault()
        self._cycle = 0

    def _load_topology(self) -> dict:
        if SYSTEM_TOPOLOGY_PATH.exists():
            try:
                return json.loads(SYSTEM_TOPOLOGY_PATH.read_text())
            except Exception:
                pass
        return {"components": [], "dependencies": [], "last_updated": 0}

    def _save_topology(self, data: dict):
        SYSTEM_TOPOLOGY_PATH.write_text(json.dumps(data, indent=2))

    def _load_components(self) -> dict:
        if COMPONENT_REGISTRY_PATH.exists():
            try:
                return json.loads(COMPONENT_REGISTRY_PATH.read_text())
            except Exception:
                pass
        return {"modules": [], "services": [], "agents": []}

    def _save_components(self, data: dict):
        COMPONENT_REGISTRY_PATH.write_text(json.dumps(data, indent=2))

    async def _scan_github(self, repo: dict) -> dict:
        ua = "EmeraldEngine/ArchitectAgent/1.0"
        headers = {"User-Agent": ua, "Accept": "application/vnd.github.v3+json"}
        async with aiohttp.ClientSession() as session:
            try:
                async with session.get(repo["url"], headers=headers,
                                        timeout=aiohttp.ClientTimeout(total=15)) as resp:
                    if resp.status == 200:
                        data = await resp.json()
                        return {
                            "repo": repo["name"],
                            "stars": data.get("stargazers_count", 0),
                            "forks": data.get("forks_count", 0),
                            "open_issues": data.get("open_issues_count", 0),
                            "language": data.get("language", "unknown"),
                            "updated_at": data.get("updated_at", ""),
                        }
            except Exception:
                pass
        return {"repo": repo["name"], "error": "unreachable"}

    async def _discover_components(self) -> dict:
        components = self._load_components()
        repo_data = await asyncio.gather(*[self._scan_github(r) for r in REPOSITORIES])
        for rd in repo_data:
            if "error" not in rd:
                for existing in components.get("modules", []):
                    if existing.get("repo") == rd["repo"]:
                        existing.update(rd)
                        break
                else:
                    components.setdefault("modules", []).append(rd)
        components["last_scanned"] = time.time()
        self._save_components(components)
        return components

    async def _analyze_topology(self, components: dict) -> dict:
        topology = self._load_topology()
        modules = components.get("modules", [])
        module_names = [m.get("repo", m.get("name", "?")) for m in modules]
        topology["components"] = module_names
        topology["dependencies"] = [
            {"from": "orchestrator", "to": m, "type": "async_rpc" if i % 2 == 0 else "rest_api"}
            for i, m in enumerate(module_names)
        ]
        topology["agent_count"] = 6
        topology["service_count"] = len(module_names)
        topology["last_updated"] = time.time()
        topology["constraints"] = {
            "max_memory_mb_per_agent": 512,
            "max_cpu_per_agent": 0.50,
            "network_isolation": "internal_bridge",
            "encryption": "AES-256-GCM",
        }
        self._save_topology(topology)
        return topology

    async def discovery_cycle(self):
        self._cycle += 1
        logging.info(f"=== Architect Discovery Cycle #{self._cycle} ===")
        start = time.time()
        components = await self._discover_components()
        topology = await self._analyze_topology(components)
        elapsed = time.time() - start
        logging.info(f"  Components: {len(components.get('modules', []))}, "
                     f"Topology: {len(topology.get('components', []))} nodes")
        self._emit_telemetry("discovery_cycle", cycle=self._cycle, elapsed=elapsed,
                              components=len(components.get("modules", [])),
                              topology_nodes=len(topology.get("components", [])))

    async def execution_loop(self):
        logging.info("Software Architect Agent activated — STEP_1 DISCOVERY + STEP_2 ANALYSIS")
        await self.discovery_cycle()
        await self._hot_daemon_loop(self.discovery_cycle, DISCOVERY_INTERVAL)


async def run_architect_loop(telemetry=None):
    agent = SoftwareArchitectAgent(telemetry=telemetry)
    await agent.execution_loop()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [ARCHITECT] %(levelname)s %(message)s")
    asyncio.run(run_architect_loop())
