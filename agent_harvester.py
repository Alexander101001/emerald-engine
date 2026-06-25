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
OLLAMA_HOST = os.getenv("OLLAMA_HOST", "http://127.0.0.1:11434")
OLLAMA_GENERATE = f"{OLLAMA_HOST}/api/generate"
OLLAMA_PULL = f"{OLLAMA_HOST}/api/pull"
OLLAMA_TAGS = f"{OLLAMA_HOST}/api/tags"
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


def init_workspace_guards():
    HARVEST_DIR.mkdir(parents=True, exist_ok=True)
    INTEGRATIONS_DIR.mkdir(parents=True, exist_ok=True)


class RepoHarvesterAgent(ParadigmAgentBase):
    """Harvests open-source repos and generates integration schemas via Ollama."""

    def __init__(self, telemetry=None):
        super().__init__(telemetry=telemetry)
        self._cycle = 0
        self._ollama_ready = False

    async def _wait_for_ollama(self):
        logging.info("  Checking Ollama service availability...")
        for attempt in range(12):
            try:
                async with aiohttp.ClientSession() as session:
                    async with session.get(OLLAMA_TAGS, timeout=aiohttp.ClientTimeout(total=5)) as resp:
                        if resp.status == 200:
                            self._ollama_ready = True
                            logging.info("  Ollama is responsive and online.")
                            return
            except (aiohttp.ClientError, asyncio.TimeoutError):
                pass
            logging.warning(f"  Ollama unreachable. Retrying in 10s... (attempt {attempt+1}/12)")
            await asyncio.sleep(10)
        logging.warning("  Ollama not available after 12 attempts — skipping AI analysis this cycle")

    async def _ensure_model(self):
        if not self._ollama_ready:
            return
        try:
            async with aiohttp.ClientSession() as session:
                async with session.get(OLLAMA_TAGS, timeout=aiohttp.ClientTimeout(total=10)) as resp:
                    if resp.status == 200:
                        data = await resp.json()
                        models = [m["name"] for m in data.get("models", [])]
                        if MODEL_NAME in models:
                            return
            logging.info(f"  Pulling model {MODEL_NAME}...")
            async with aiohttp.ClientSession() as session:
                async with session.post(OLLAMA_PULL, json={"name": MODEL_NAME}, timeout=aiohttp.ClientTimeout(total=600)) as resp:
                    if resp.status == 200:
                        logging.info(f"  Model {MODEL_NAME} ready")
        except Exception as e:
            logging.warning(f"  Model check/pull failed: {e}")

    async def harvest_cycle(self):
        self._cycle += 1
        logging.info(f"=== Harvester Cycle #{self._cycle} ===")

        init_workspace_guards()

        if not self._ollama_ready:
            await self._wait_for_ollama()
        if self._ollama_ready:
            await self._ensure_model()

        for repo_url in TARGET_REPOS:
            repo_name = repo_url.rstrip("/").split("/")[-1]
            target_path = HARVEST_DIR / repo_name

            try:
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
            except Exception as e:
                logging.warning(f"  Git error for {repo_name}: {e}")
                continue

            readme_file = target_path / "README.md"
            context = ""
            if readme_file.exists():
                try:
                    context = readme_file.read_text(errors="ignore")[:1000]
                except Exception as e:
                    logging.warning(f"  Read error for {readme_file}: {e}")

            if not context:
                logging.info(f"  Empty context for {repo_name} — skipping AI analysis")

            if self._ollama_ready and context:
                prompt = (
                    f"Analyze the structural layout for {repo_name}. Data snippet: {context}. "
                    f"Provide an automated integration mapping blueprint for this system. "
                    f"Output valid JSON only with keys: name, description, key_features, integration_steps."
                )

                try:
                    async with aiohttp.ClientSession() as session:
                        payload = {"model": MODEL_NAME, "prompt": prompt, "stream": False, "format": "json"}
                        async with session.post(OLLAMA_GENERATE, json=payload, timeout=aiohttp.ClientTimeout(total=90)) as resp:
                            if resp.status == 200:
                                data = await resp.json()
                                ai_response = data.get("response", "{}")
                                out_path = INTEGRATIONS_DIR / f"{repo_name}_schema.json"
                                out_path.write_text(ai_response)
                                logging.info(f"  Integrated: {repo_name}")
                            else:
                                logging.warning(f"  Ollama error for {repo_name}: {resp.status}")
                except Exception as e:
                    logging.warning(f"  AI analysis skipped for {repo_name}: {e}")

            if self._telemetry:
                self._telemetry.record_stream_item()

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
