#!/usr/bin/env python3
import os
import sys
import time
import json
import hashlib
import logging
import subprocess
import asyncio
from pathlib import Path
from typing import Optional

import aiohttp
import aiofiles

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger("SelfEvolution")

OLLAMA_HOST = os.getenv("OLLAMA_HOST", "http://localhost:11434")
OLLAMA_URL = f"{OLLAMA_HOST}/api/generate"
OLLAMA_TAGS = f"{OLLAMA_HOST}/api/tags"
MODEL = os.getenv("EVOLUTION_MODEL", "qwen2.5-coder:1.5b")
SCAN_INTERVAL = int(os.getenv("EVOLUTION_INTERVAL", "300"))
MAX_FILE_SIZE = int(os.getenv("EVOLUTION_MAX_FILE_SIZE", "8192"))
EVOLUTION_LOG = os.getenv("EVOLUTION_LOG", "/tmp/evolution_log.json")
AUTO_PUSH = os.getenv("EVOLUTION_AUTO_PUSH", "true").lower() == "true"

REPO_DIR = Path(os.getenv("REPO_DIR", "/tmp/opencode/emerald-engine"))
EXCLUDE_DIRS = {
    ".git", "__pycache__", "harvested_repos", "external_tools",
    "node_modules", ".aegis", "compose_fragments", "workspace_core",
    ".secrets", ".vault", ".github",
}
EXCLUDE_FILES = {
    "self_evolution.py", "omni_engine_core.py", "integrations/cc_switch_adapter.rs",
}

AUDIT_PROMPT = (
    "You are an expert Python code auditor and optimizer. "
    "Analyze the following code snippet for performance bottlenecks, "
    "architectural improvements, security vulnerabilities, and resource leaks. "
    "Return ONLY the complete optimized version of the code as a single code block. "
    "Do NOT add explanatory text outside the code block. "
    "Preserve all existing functionality and imports. "
    "Fix any bugs you find. Apply idiomatic Python patterns. "
    "Here is the code:\n\n"
)


def _fingerprint(text: str) -> str:
    return hashlib.sha256(text.encode()).hexdigest()[:16]


class EvolutionSnapshot:
    def __init__(self):
        self.start_time = time.time()
        self.files_scanned = 0
        self.files_eligible = 0
        self.files_optimized = 0
        self.total_bytes_before = 0
        self.total_bytes_after = 0
        self.errors = []

    def record(self, path: str, before: str, after: str, optimized: bool):
        self.files_scanned += 1
        if len(before) > 50:
            self.files_eligible += 1
            self.total_bytes_before += len(before)
        if optimized:
            self.files_optimized += 1
            self.total_bytes_after += len(after)

    def summary(self) -> dict:
        elapsed = time.time() - self.start_time
        return {
            "timestamp": self.start_time,
            "elapsed_seconds": round(elapsed, 2),
            "files_scanned": self.files_scanned,
            "files_eligible": self.files_eligible,
            "files_optimized": self.files_optimized,
            "bytes_before": self.total_bytes_before,
            "bytes_after": self.total_bytes_after,
            "bytes_saved": max(0, self.total_bytes_before - self.total_bytes_after),
            "errors": len(self.errors),
        }


async def _wait_for_ollama(session: aiohttp.ClientSession):
    while True:
        try:
            async with session.get(OLLAMA_TAGS, timeout=aiohttp.ClientTimeout(total=5)) as resp:
                if resp.status == 200:
                    logger.info("Ollama service is online")
                    return
        except (aiohttp.ClientError, asyncio.TimeoutError):
            logger.warning("Ollama not ready, retrying in 10s...")
            await asyncio.sleep(10)


async def _ollama_audit(session: aiohttp.ClientSession, code: str) -> Optional[str]:
    payload = {
        "model": MODEL,
        "prompt": AUDIT_PROMPT + code,
        "stream": False,
        "options": {
            "temperature": 0.3,
            "top_p": 0.9,
            "num_predict": 4096,
        },
    }
    try:
        async with session.post(
            OLLAMA_URL, json=payload, timeout=aiohttp.ClientTimeout(total=120)
        ) as resp:
            if resp.status != 200:
                logger.warning(f"Ollama returned {resp.status}")
                return None
            data = await resp.json()
            response_text = data.get("response", "")
            start = response_text.find("```python")
            end = response_text.rfind("```")
            if start != -1 and end != -1:
                code_block = response_text[start + 9:end].strip()
                if code_block:
                    return code_block
            if len(response_text) > 50:
                return response_text
            return None
    except (aiohttp.ClientError, asyncio.TimeoutError, json.JSONDecodeError) as e:
        logger.warning(f"Ollama audit error: {e}")
        return None


def _apply_optimization(path: Path, original: str, optimized: str) -> bool:
    if original.strip() == optimized.strip():
        return False
    try:
        compile(optimized, path.name, "exec")
    except SyntaxError as e:
        logger.warning(f"Optimized code has syntax error in {path.name}: {e}")
        return False
    path.write_text(optimized)
    return True


async def _git_commit_push(files: list[str]):
    if not files:
        return
    try:
        subprocess.run(
            ["git", "config", "--global", "user.name", "emerald-evolution"],
            capture_output=True, cwd=REPO_DIR,
        )
        subprocess.run(
            ["git", "config", "--global", "user.email", "evolution@emerald.engine"],
            capture_output=True, cwd=REPO_DIR,
        )
        subprocess.run(
            ["git", "add"] + files,
            capture_output=True, cwd=REPO_DIR,
        )
        result = subprocess.run(
            ["git", "diff-index", "--quiet", "HEAD"],
            capture_output=True, cwd=REPO_DIR,
        )
        if result.returncode != 0:
            subprocess.run(
                ["git", "commit", "-m", "evolution: auto-optimize via Ollama self-evolution loop"],
                capture_output=True, cwd=REPO_DIR,
            )
            if AUTO_PUSH:
                subprocess.run(
                    ["git", "push"],
                    capture_output=True, cwd=REPO_DIR,
                )
                logger.info("Pushed evolution commit to remote")
            logger.info(f"Committed {len(files)} optimized files")
    except Exception as e:
        logger.error(f"Git operation failed: {e}")


def _collect_python_files() -> list[Path]:
    files = []
    for path in REPO_DIR.rglob("*.py"):
        rel = path.relative_to(REPO_DIR)
        parts = rel.parts
        if any(p in EXCLUDE_DIRS for p in parts):
            continue
        if str(rel) in EXCLUDE_FILES:
            continue
        if path.stat().st_size > MAX_FILE_SIZE * 2:
            continue
        files.append(path)
    return sorted(files)


def _extract_code_block(text: str) -> Optional[str]:
    start = text.find("```python")
    if start == -1:
        start = text.find("```")
    if start != -1:
        end = text.find("```", start + 3)
        if end != -1:
            content = text[start + (9 if text[start:start+9] == "```python" else 3):end].strip()
            if content:
                return content
    if text.count("\n") >= 3 and len(text) > 100:
        return text.strip()
    return None


def _log_evolution(snapshot: EvolutionSnapshot):
    log_data = {"history": []}
    if os.path.exists(EVOLUTION_LOG):
        try:
            with open(EVOLUTION_LOG) as f:
                log_data = json.load(f)
        except (json.JSONDecodeError, FileNotFoundError):
            log_data = {"history": []}
    log_data["history"].append(snapshot.summary())
    if len(log_data["history"]) > 100:
        log_data["history"] = log_data["history"][-100:]
    with open(EVOLUTION_LOG, "w") as f:
        json.dump(log_data, f, indent=2)


async def evolution_cycle():
    snapshot = EvolutionSnapshot()
    files = _collect_python_files()
    logger.info(f"Evolution cycle scanning {len(files)} Python files")

    connector = aiohttp.TCPConnector(limit=4, force_close=True)
    async with aiohttp.ClientSession(connector=connector) as session:
        await _wait_for_ollama(session)
        optimized_files = []
        for path in files:
            try:
                original = path.read_text(encoding="utf-8", errors="ignore")
            except Exception as e:
                snapshot.errors.append(str(e))
                continue
            if len(original) < 100 or len(original) > MAX_FILE_SIZE * 2:
                snapshot.record(str(path.relative_to(REPO_DIR)), original, "", False)
                continue
            snapshot.record(str(path.relative_to(REPO_DIR)), original, "", False)
            logger.info(f"Auditing {path.relative_to(REPO_DIR)} ({len(original)} chars)")
            result = await _ollama_audit(session, original[:MAX_FILE_SIZE])
            if not result:
                await asyncio.sleep(1)
                continue
            optimized = _extract_code_block(result)
            if not optimized or optimized == original.strip():
                await asyncio.sleep(1)
                continue
            if _apply_optimization(path, original, optimized):
                snapshot.record(str(path.relative_to(REPO_DIR)), original, optimized, True)
                optimized_files.append(str(path))
                logger.info(f"Optimized: {path.relative_to(REPO_DIR)}")
            await asyncio.sleep(2)

    if optimized_files:
        await _git_commit_push(optimized_files)
    _log_evolution(snapshot)
    logger.info(f"Cycle complete: {snapshot.summary()}")
    return snapshot


async def main():
    logger.info(f"Self-Evolution Loop starting (model={MODEL}, interval={SCAN_INTERVAL}s)")
    while True:
        try:
            await evolution_cycle()
        except Exception as e:
            logger.error(f"Evolution cycle crashed: {e}", exc_info=True)
        logger.info(f"Sleeping {SCAN_INTERVAL}s before next cycle")
        await asyncio.sleep(SCAN_INTERVAL)


if __name__ == "__main__":
    asyncio.run(main())
