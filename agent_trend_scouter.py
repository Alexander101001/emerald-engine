import asyncio
import json
import logging
import os
import time
from pathlib import Path

import aiohttp

from opcode_base import ParadigmAgentBase

SCOUT_INTERVAL = int(os.getenv("SCOUT_INTERVAL", "3600"))
SCOUT_REPORT_PATH = Path("/tmp/emerald_scout_report.json")


class TrendScouterAgent(ParadigmAgentBase):
    """SECTION 1: GitHub Search API every 60 min for trending repos >1000 stars."""

    def __init__(self, telemetry=None):
        super().__init__(telemetry=telemetry)
        self._cycle = 0
        self.languages = ["Python", "Go", "Rust", "TypeScript"]
        self.min_stars = 1000

    def _load_seen(self) -> set:
        path = Path("/tmp/emerald_scout_seen.json")
        if path.exists():
            try:
                return set(json.loads(path.read_text()))
            except Exception:
                pass
        return set()

    def _save_seen(self, seen: set):
        Path("/tmp/emerald_scout_seen.json").write_text(json.dumps(list(seen)))

    async def _search_github(self, session: aiohttp.ClientSession, lang: str) -> list:
        url = (
            f"https://api.github.com/search/repositories"
            f"?q=language:{lang}+stars:>={self.min_stars}&sort=stars&order=desc&per_page=10"
        )
        headers = {"User-Agent": "EmeraldEngine/Scouter/1.0", "Accept": "application/vnd.github.v3+json"}
        try:
            async with session.get(url, headers=headers,
                                    timeout=aiohttp.ClientTimeout(total=15)) as resp:
                if resp.status == 200:
                    data = await resp.json()
                    return data.get("items", [])
        except Exception:
            pass
        return []

    async def _fetch_readme(self, session: aiohttp.ClientSession, full_name: str) -> str:
        url = f"https://api.github.com/repos/{full_name}/readme"
        headers = {"User-Agent": "EmeraldEngine/Scouter/1.0", "Accept": "application/vnd.github.v3.raw"}
        try:
            async with session.get(url, headers=headers,
                                    timeout=aiohttp.ClientTimeout(total=10)) as resp:
                if resp.status == 200:
                    return await resp.text()[:2000]
        except Exception:
            pass
        return ""

    async def scout_cycle(self):
        self._cycle += 1
        logging.info(f"=== Trend Scouter Cycle #{self._cycle} ===")
        start = time.time()
        seen = self._load_seen()
        candidates = []
        async with aiohttp.ClientSession() as session:
            tasks = [self._search_github(session, lang) for lang in self.languages]
            results = await asyncio.gather(*tasks)
            for lang, items in zip(self.languages, results):
                for repo in items:
                    full_name = repo.get("full_name", "")
                    if full_name in seen:
                        continue
                    readme = await self._fetch_readme(session, full_name)
                    candidates.append({
                        "full_name": full_name,
                        "language": lang,
                        "stars": repo.get("stargazers_count", 0),
                        "description": repo.get("description", "") or "",
                        "topics": repo.get("topics", []),
                        "html_url": repo.get("html_url", ""),
                        "readme_preview": readme[:500],
                    })
                    seen.add(full_name)
        self._save_seen(seen)
        combined_texts = [f"{c['full_name']} {c['description']} {' '.join(c['topics'])}"
                          for c in candidates]
        combined_query = "cloud platform deployment infrastructure scalable distributed"
        ranked = self._bm25_rank(combined_texts, combined_query)
        ranked_candidates = []
        for t in ranked:
            idx = combined_texts.index(t)
            ranked_candidates.append(candidates[idx])
        logging.info(f"  Discovered {len(candidates)} new repos across {len(self.languages)} languages")
        report = {
            "cycle": self._cycle,
            "time": time.time(),
            "elapsed": round(time.time() - start, 2),
            "new_repos": len(candidates),
            "languages_scanned": self.languages,
            "candidates": ranked_candidates[:20],
            "total_seen": len(seen),
        }
        SCOUT_REPORT_PATH.write_text(json.dumps(report, indent=2))
        self._emit_telemetry("scout_cycle", new_repos=len(candidates),
                              languages=self.languages, total_seen=len(seen))

    async def execution_loop(self):
        logging.info("Trend Scouter activated — SECTION 1: GitHub trending + BM25 ranking")
        await self.scout_cycle()
        await self._hot_daemon_loop(self.scout_cycle, SCOUT_INTERVAL)


async def run_scouter_loop(telemetry=None):
    agent = TrendScouterAgent(telemetry=telemetry)
    await agent.execution_loop()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [SCOUTER] %(levelname)s %(message)s")
    asyncio.run(run_scouter_loop())
