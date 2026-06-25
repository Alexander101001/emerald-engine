import asyncio
import json
import logging
import os
import subprocess
import time
from pathlib import Path

import aiohttp

from opcode_base import ParadigmAgentBase

HARVEST_INTERVAL = int(os.getenv("HARVEST_INTERVAL", "600"))
OLLAMA_URL = os.getenv("OLLAMA_URL", "http://127.0.0.1:11434/api/generate")
MODEL_NAME = os.getenv("HARVESTER_MODEL", "qwen2.5-coder:1.5b")

TARGET_REPOS = [
    "https://github.com/crewAI/crewAI",
    "https://github.com/langchain-ai/langgraph",
    "https://github.com/microsoft/autogen",
    "https://github.com/unclecode/crawl4ai",
    "https://github.com/browser-use/browser-use",
]

HARVEST_DIR = Path("harvested_repos")
INTEGRATIONS_DIR = Path("integrations")


class RepoHarvesterAgent(ParadigmAgentBase):
    """Harvests open-source repos and generates integration schemas via Ollama."""

    def __init__(self, telemetry=None):
        super().__init__(telemetry=telemetry)
        self._cycle = 0

    async def harvest_cycle(self):
        self._cycle += 1
        logging.info(f"=== Harvester Cycle #{self._cycle} ===")

        HARVEST_DIR.mkdir(parents=True, exist_ok=True)
        INTEGRATIONS_DIR.mkdir(parents=True, exist_ok=True)

        for repo_url in TARGET_REPOS:
            repo_name = repo_url.rstrip("/").split("/")[-1]
            target_path = HARVEST_DIR / repo_name

            if not target_path.exists():
                logging.info(f"  Cloning {repo_name}...")
                proc = await asyncio.create_subprocess_exec(
                    "git", "clone", "--depth", "1", repo_url, str(target_path),
                    stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                )
                await proc.wait()
            else:
                logging.info(f"  Pulling {repo_name}...")
                proc = await asyncio.create_subprocess_exec(
                    "git", "-C", str(target_path), "pull",
                    stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                )
                await proc.wait()

            readme_file = target_path / "README.md"
            context = ""
            if readme_file.exists():
                context = readme_file.read_text(errors="ignore")[:1500]

            prompt = (
                f"Analyze {repo_name}. Context: {context}. "
                f"Output a JSON object with keys: name, description, key_features, integration_steps. "
                f"Strictly valid JSON only."
            )

            try:
                async with aiohttp.ClientSession() as session:
                    payload = {"model": MODEL_NAME, "prompt": prompt, "stream": False, "format": "json"}
                    async with session.post(OLLAMA_URL, json=payload, timeout=aiohttp.ClientTimeout(total=90)) as resp:
                        if resp.status == 200:
                            data = await resp.json()
                            ai_response = data.get("response", "{}")
                            out_path = INTEGRATIONS_DIR / f"{repo_name}_schema.json"
                            out_path.write_text(ai_response)
                            logging.info(f"  Integrated: {repo_name}")
                        else:
                            logging.warning(f"  Ollama error for {repo_name}: {resp.status}")
            except Exception as e:
                logging.warning(f"  Harvester error for {repo_name}: {e}")

            if self.telemetry:
                self.telemetry.record_stream_item()

        logging.info(f"Harvest cycle #{self._cycle} complete — {len(TARGET_REPOS)} repos processed")


async def run_harvester_loop(telemetry=None):
    agent = RepoHarvesterAgent(telemetry=telemetry)
    logging.info("Harvester Agent activated — autonomous repo harvesting + Ollama analysis")
    while True:
        try:
            await agent.harvest_cycle()
        except Exception as e:
            logging.error(f"Harvester cycle error: {e}")
        logging.info(f"Harvester sleeping {HARVEST_INTERVAL}s...")
        await asyncio.sleep(HARVEST_INTERVAL)
