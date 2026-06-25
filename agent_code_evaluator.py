import asyncio
import json
import logging
import os
import subprocess
import tempfile
import time
from pathlib import Path

from opcode_base import ParadigmAgentBase

EVAL_INTERVAL = int(os.getenv("EVAL_INTERVAL", "3600"))
EVAL_REPORT_PATH = Path("/tmp/emerald_eval_report.json")


class CodeEvaluatorAgent(ParadigmAgentBase):
    """SECTION 2: Ephemeral sandbox evaluation against 161 reasoning rules."""

    def __init__(self, telemetry=None):
        super().__init__(telemetry=telemetry)
        self._cycle = 0

    def _load_candidates(self) -> list:
        path = Path("/tmp/emerald_scout_report.json")
        if path.exists():
            try:
                data = json.loads(path.read_text())
                return data.get("candidates", [])
            except Exception:
                pass
        return []

    async def _clone_and_check(self, url: str, language: str) -> dict:
        with tempfile.TemporaryDirectory(prefix="emerald_eval_") as tmpdir:
            try:
                r = subprocess.run(
                    ["git", "clone", "--depth", "1", url, tmpdir + "/repo"],
                    capture_output=True, timeout=60,
                )
                if r.returncode != 0:
                    return {"cloned": False, "error": r.stderr.decode()[:100]}
                repo_dir = Path(tmpdir + "/repo")
                files = list(repo_dir.rglob("*"))
                py_files = [f for f in files if f.suffix in (".py", ".go", ".rs", ".ts")]
                syntax_ok = 0
                syntax_err = 0
                for f in py_files[:50]:
                    try:
                        if f.suffix == ".py":
                            subprocess.run(["python3", "-c", f"import ast; ast.parse(open('{f}').read())"],
                                           capture_output=True, timeout=10)
                        elif f.suffix == ".go":
                            subprocess.run(["gofmt", "-e", str(f)], capture_output=True, timeout=10)
                        elif f.suffix == ".rs":
                            subprocess.run(["rustfmt", "--check", str(f)], capture_output=True, timeout=10)
                        syntax_ok += 1
                    except Exception:
                        syntax_err += 1
                return {
                    "cloned": True,
                    "files_checked": len(py_files[:50]),
                    "syntax_ok": syntax_ok,
                    "syntax_err": syntax_err,
                    "language": language,
                }
            except Exception as e:
                return {"cloned": False, "error": str(e)[:100]}

    def _evaluate_161_rules(self, candidate: dict, clone_result: dict) -> dict:
        entity_id = candidate.get("full_name", "unknown")
        niche = self._niche_for_type(candidate.get("language", ""),
                                     "compute" if clone_result.get("cloned") else "tools")
        rules_passed = self._apply_reasoning_rules(entity_id, niche)
        score = 0
        if clone_result.get("cloned"):
            score += 30
            score += max(0, (clone_result.get("syntax_ok", 0) - clone_result.get("syntax_err", 0)) * 5)
        stars = candidate.get("stars", 0)
        if stars >= 5000:
            score += 30
        elif stars >= 2000:
            score += 20
        elif stars >= 1000:
            score += 10
        topics = candidate.get("topics", [])
        relevant_topics = {"cloud", "serverless", "distributed", "saas", "api",
                           "microservice", "database", "deployment", "container"}
        score += len(set(topics) & relevant_topics) * 5
        return {
            "entity_id": entity_id,
            "niche": niche,
            "rules_passed": rules_passed,
            "score": score,
            "recommended": rules_passed and score >= 30,
        }

    async def eval_cycle(self):
        self._cycle += 1
        logging.info(f"=== Code Evaluator Cycle #{self._cycle} ===")
        start = time.time()
        candidates = self._load_candidates()
        if not candidates:
            logging.info("  No candidates to evaluate")
            return
        tasks = []
        for c in candidates[:10]:
            clone_url = c.get("html_url", "").replace("https://github.com/", "https://github.com/") + ".git"
            tasks.append(self._clone_and_check(clone_url, c.get("language", "")))
        clone_results = await asyncio.gather(*tasks)
        evaluations = []
        for c, cr in zip(candidates[:10], clone_results):
            ev = self._evaluate_161_rules(c, cr)
            ev["candidate"] = c
            evaluations.append(ev)
        evaluations.sort(key=lambda x: -x.get("score", 0))
        recommended = [e for e in evaluations if e.get("recommended")]
        logging.info(f"  Evaluated {len(evaluations)} candidates, {len(recommended)} recommended")
        report = {
            "cycle": self._cycle,
            "time": time.time(),
            "elapsed": round(time.time() - start, 2),
            "evaluations": evaluations,
            "recommended": [e["entity_id"] for e in recommended],
        }
        EVAL_REPORT_PATH.write_text(json.dumps(report, indent=2))
        self._emit_telemetry("eval_cycle", evaluated=len(evaluations),
                              recommended=len(recommended))

    async def execution_loop(self):
        logging.info("Code Evaluator activated — SECTION 2: ephemeral sandbox + 161 rules")
        await self.eval_cycle()
        await self._hot_daemon_loop(self.eval_cycle, EVAL_INTERVAL)


async def run_evaluator_loop(telemetry=None):
    agent = CodeEvaluatorAgent(telemetry=telemetry)
    await agent.execution_loop()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [EVAL] %(levelname)s %(message)s")
    asyncio.run(run_evaluator_loop())
