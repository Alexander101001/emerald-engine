import asyncio
import json
import logging
import os
import time
from pathlib import Path

import aiohttp

from opcode_base import ParadigmAgentBase

HF_SYNC_INTERVAL = int(os.getenv("HF_SYNC_INTERVAL", "3600"))
HF_SYNC_REPORT = Path("/tmp/emerald_hf_sync.json")
HF_TOKEN = os.getenv("HF_TOKEN", "")


class HFCognitiveSyncAgent(ParadigmAgentBase):
    """SECTION 6: Persistent HF WebSocket + API, scan for SLMs (Gemma, Llama)."""

    def __init__(self, telemetry=None):
        super().__init__(telemetry=telemetry)
        self._cycle = 0
        self._session = None
        self._ws = None

    async def _get_session(self):
        if self._session is None or self._session.closed:
            self._session = aiohttp.ClientSession()
        return self._session

    async def _scan_slms(self, session: aiohttp.ClientSession) -> list:
        models = []
        queries = [
            "google/gemma", "meta/llama", "microsoft/phi", "mistralai/mistral",
            "tiiuae/falcon", "upstage/solar",
        ]
        for q in queries:
            url = f"https://huggingface.co/api/models?search={q}&sort=downloads&direction=-1&limit=5"
            headers = {"User-Agent": "EmeraldEngine/HFSync/1.0"}
            if HF_TOKEN:
                headers["Authorization"] = f"Bearer {HF_TOKEN}"
            try:
                async with session.get(url, headers=headers,
                                        timeout=aiohttp.ClientTimeout(total=15)) as resp:
                    if resp.status == 200:
                        data = await resp.json()
                        for item in data[:5] if isinstance(data, list) else []:
                            models.append({
                                "model_id": item.get("modelId", item.get("id", "")),
                                "pipeline_tag": item.get("pipeline_tag", ""),
                                "downloads": item.get("downloads", 0),
                                "likes": item.get("likes", 0),
                                "library": item.get("library_name", ""),
                                "tags": item.get("tags", [])[:5],
                            })
            except Exception:
                continue
            await asyncio.sleep(0.5)
        return models

    async def _hf_health_check(self, session: aiohttp.ClientSession) -> dict:
        try:
            async with session.get("https://huggingface.co/api/models?limit=1",
                                    timeout=aiohttp.ClientTimeout(total=10)) as resp:
                return {"reachable": resp.status == 200}
        except Exception:
            return {"reachable": False}

    async def _connect_websocket(self):
        try:
            ws_url = "wss://huggingface.co/api/ws"
            headers = {"User-Agent": "EmeraldEngine/HFSync/1.0"}
            if HF_TOKEN:
                headers["Authorization"] = f"Bearer {HF_TOKEN}"
            session = await self._get_session()
            self._ws = await session.ws_connect(ws_url, headers=headers,
                                                 heartbeat=30, receive_timeout=60)
            logging.info("  HF WebSocket connected")
        except Exception as e:
            self._ws = None
            logging.debug(f"  HF WebSocket not available: {e}")

    async def _get_model_details(self, session: aiohttp.ClientSession, model_id: str) -> dict:
        url = f"https://huggingface.co/api/models/{model_id}"
        try:
            async with session.get(url, timeout=aiohttp.ClientTimeout(total=10)) as resp:
                if resp.status == 200:
                    data = await resp.json()
                    return {
                        "model_id": model_id,
                        "private": data.get("private", False),
                        "downloads": data.get("downloads", 0),
                        "library": data.get("library_name", ""),
                        "siblings": len(data.get("siblings", [])),
                        "created_at": data.get("created_at", ""),
                    }
        except Exception:
            pass
        return {"model_id": model_id, "error": "unreachable"}

    async def hf_sync_cycle(self):
        self._cycle += 1
        logging.info(f"=== HF Cognitive Sync Cycle #{self._cycle} ===")
        start = time.time()
        session = await self._get_session()
        health = await self._hf_health_check(session)
        models = await self._scan_slms(session)
        small_models = [m for m in models if m.get("pipeline_tag") in
                         ("text-generation", "text2text-generation", "feature-extraction")]
        combined_query = "small language model slm gemma llama phi inference efficient"
        ranked = self._bm25_rank(small_models, combined_query,
                                  key=lambda m: f"{m.get('model_id', '')} {' '.join(m.get('tags', []))}")
        logging.info(f"  HF reachable: {health.get('reachable')}, "
                     f"Models found: {len(models)}, SLM candidates: {len(ranked)}")
        if self._ws is None or (self._ws and self._ws.closed):
            await self._connect_websocket()
        report = {
            "cycle": self._cycle,
            "time": time.time(),
            "elapsed": round(time.time() - start, 2),
            "hf_reachable": health.get("reachable", False),
            "models_total": len(models),
            "slm_candidates": ranked[:10],
            "websocket_connected": self._ws is not None and not self._ws.closed,
        }
        HF_SYNC_REPORT.write_text(json.dumps(report, indent=2))
        self._emit_telemetry("hf_sync", models_found=len(models),
                              slm_candidates=len(ranked),
                              ws_connected=report["websocket_connected"])

    async def execution_loop(self):
        logging.info("HF Sync activated — SECTION 6: HF WebSocket + API + SLM scanning")
        await self.hf_sync_cycle()
        await self._hot_daemon_loop(self.hf_sync_cycle, HF_SYNC_INTERVAL)


async def run_hf_sync_loop(telemetry=None):
    agent = HFCognitiveSyncAgent(telemetry=telemetry)
    await agent.execution_loop()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [HF] %(levelname)s %(message)s")
    asyncio.run(run_hf_sync_loop())
