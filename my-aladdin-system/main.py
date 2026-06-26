import asyncio
import os
import sys
import json
import time
import hashlib
import logging
import subprocess
import signal
from pathlib import Path
from datetime import datetime, timezone
from typing import Optional

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s")
logger = logging.getLogger("AladdinCore")

BASE_DIR = Path(__file__).parent
MEMORY_PATH = BASE_DIR / "evolutionary_memory.json"
NOTIF_PATH = BASE_DIR / "notifications.json"
ACCUMULATION_THRESHOLD = 20.0
SCALING_FACTOR = 1.01
PULSE_INTERVAL = 300
EVOLUTION_INTERVAL = 1800
OLLAMA_HOST = os.getenv("OLLAMA_HOST", "http://localhost:11434")
OLLAMA_MODEL = os.getenv("OLLAMA_MODEL", "qwen2.5-coder:1.5b")

PROCESS_REGISTRY = {}


def _load_json(path: Path) -> dict:
    if path.exists() and path.stat().st_size > 0:
        try:
            return json.loads(path.read_text())
        except (json.JSONDecodeError, ValueError):
            return {}
    return {}


def _save_json(path: Path, data):
    path.write_text(json.dumps(data, indent=2, default=str))


def _timestamp() -> str:
    return datetime.now(timezone.utc).isoformat()


def _fingerprint(text: str) -> str:
    return hashlib.sha256(text.encode()).hexdigest()[:12]


class PulseSystem:
    def __init__(self):
        self.last_pulse = 0
        self.checks = {"ollama": False, "binance": False, "balance": 0.0}

    async def check_ollama(self) -> bool:
        try:
            import aiohttp
            async with aiohttp.ClientSession() as session:
                async with session.get(f"{OLLAMA_HOST}/api/tags", timeout=5) as resp:
                    ok = resp.status == 200
                    self.checks["ollama"] = ok
                    return ok
        except Exception as e:
            logger.warning(f"Ollama check failed: {e}")
            self.checks["ollama"] = False
            return False

    async def check_binance(self) -> bool:
        try:
            import aiohttp
            async with aiohttp.ClientSession() as session:
                async with session.get(
                    "https://api.binance.com/api/v3/ping", timeout=5
                ) as resp:
                    ok = resp.status == 200
                    self.checks["binance"] = ok
                    return ok
        except Exception as e:
            logger.warning(f"Binance check failed: {e}")
            self.checks["binance"] = False
            return False

    async def pulse(self):
        self.last_pulse = time.time()
        ollama_ok = await self.check_ollama()
        binance_ok = await self.check_binance()
        balance = await self._get_balance()
        self.checks["balance"] = balance
        logger.info(
            f"Pulse | ollama={'OK' if ollama_ok else 'DOWN'} "
            f"binance={'OK' if binance_ok else 'DOWN'} "
            f"balance=${balance:.4f} "
            f"mode={'TRADING' if balance >= ACCUMULATION_THRESHOLD else 'ACCUMULATION'}"
        )
        return self.checks

    async def _get_balance(self) -> float:
        try:
            api_key = os.getenv("BINANCE_API_KEY", "")
            api_secret = os.getenv("BINANCE_API_SECRET", "")
            if not api_key or not api_secret:
                return 0.0
            import aiohttp
            import hmac
            import hashlib as hl
            timestamp = int(time.time() * 1000)
            query = f"timestamp={timestamp}"
            signature = hmac.new(
                api_secret.encode(), query.encode(), hl.sha256
            ).hexdigest()
            headers = {"X-MBX-APIKEY": api_key}
            url = f"https://api.binance.com/api/v3/account?{query}&signature={signature}"
            async with aiohttp.ClientSession() as session:
                async with session.get(url, headers=headers, timeout=10) as resp:
                    if resp.status == 200:
                        data = await resp.json()
                        free = sum(float(b["free"]) for b in data.get("balances", []) if float(b["free"]) > 0)
                        locked = sum(float(b["locked"]) for b in data.get("balances", []) if float(b["locked"]) > 0)
                        return free + locked
                    return 0.0
        except Exception as e:
            logger.warning(f"Balance check error: {e}")
            return 0.0


class EvolutionaryMemory:
    def __init__(self):
        self.data = _load_json(MEMORY_PATH)
        if "strategies" not in self.data:
            self.data["strategies"] = []
        if "forbidden_patterns" not in self.data:
            self.data["forbidden_patterns"] = []

    def record_strategy(self, strategy: dict):
        strategy["timestamp"] = _timestamp()
        strategy["fingerprint"] = _fingerprint(json.dumps(strategy, default=str))
        self.data["strategies"].append(strategy)
        _save_json(MEMORY_PATH, self.data)

    def record_failure(self, strategy: dict, reason: str, loss_amount: float):
        failure = {
            "timestamp": _timestamp(),
            "strategy": strategy,
            "reason": reason,
            "loss_amount": loss_amount,
            "fingerprint": _fingerprint(strategy.get("code", "")),
        }
        self.data["forbidden_patterns"].append(failure)
        self.data["strategies"].append({**strategy, "status": "failed", "failure_reason": reason})
        _save_json(MEMORY_PATH, self.data)
        logger.warning(f"Recorded forbidden pattern: {reason} (loss=${loss_amount:.6f})")

    def get_recent_strategies(self, n: int = 10) -> list:
        return self.data["strategies"][-n:]

    def get_failure_rate(self) -> float:
        total = len(self.data["strategies"])
        if total == 0:
            return 0.0
        failed = sum(1 for s in self.data["strategies"] if s.get("status") == "failed")
        return failed / total

    def get_last_failure_reason(self) -> Optional[str]:
        if self.data["forbidden_patterns"]:
            return self.data["forbidden_patterns"][-1].get("reason")
        return None


class EvolutionaryLoop:
    def __init__(self, memory: EvolutionaryMemory):
        self.memory = memory
        self.cycle = 0
        self.accumulation_mode = True
        self.trade_size = 0.0001

    async def analyze_failure(self, strategy_code: str, failure_reason: str) -> Optional[str]:
        prompt = (
            f"Analyze this trading strategy that failed. Reason: {failure_reason}\n\n"
            f"Strategy code:\n{strategy_code}\n\n"
            f"Propose a safer variation that avoids the same mistake. "
            f"Return ONLY the corrected Python code as a single code block."
        )
        try:
            import aiohttp
            payload = {
                "model": OLLAMA_MODEL,
                "prompt": prompt,
                "stream": False,
                "options": {"temperature": 0.3, "num_predict": 2048},
            }
            async with aiohttp.ClientSession() as session:
                async with session.post(
                    f"{OLLAMA_HOST}/api/generate", json=payload, timeout=120
                ) as resp:
                    if resp.status == 200:
                        data = await resp.json()
                        text = data.get("response", "")
                        start = text.find("```python")
                        end = text.rfind("```")
                        if start != -1 and end != -1:
                            return text[start + 9:end].strip()
                        return text.strip() if len(text) > 50 else None
        except Exception as e:
            logger.error(f"Ollama analysis failed: {e}")
        return None

    async def evolve(self, balance: float):
        self.cycle += 1
        logger.info(f"=== Evolution Cycle #{self.cycle} ===")
        if balance >= ACCUMULATION_THRESHOLD:
            self.accumulation_mode = False
            logger.info("Threshold reached! Switching to TRADING MODE")
        failure_rate = self.memory.get_failure_rate()
        last_failure = self.memory.get_last_failure_reason()
        logger.info(f"Failure rate: {failure_rate:.1%} | Recent memory entries: {len(self.memory.data['strategies'])}")

        if last_failure:
            logger.info(f"Analyzing last failure: {last_failure}")
            last_failed = self.memory.data["forbidden_patterns"][-1]
            improved = await self.analyze_failure(
                json.dumps(last_failed.get("strategy", {}), indent=2),
                last_failure,
            )
            if improved:
                logger.info(f"Ollama proposed improved strategy ({len(improved)} chars)")
                self.memory.record_strategy({
                    "code": improved,
                    "source": "ollama_evolution",
                    "cycle": self.cycle,
                    "status": "proposed",
                    "balance_at_cycle": balance,
                })
        if not self.accumulation_mode and failure_rate < 0.3:
            self.trade_size *= SCALING_FACTOR
            logger.info(f"Scaling trade size to {self.trade_size:.6f} (factor={SCALING_FACTOR})")


class SelfHealingProcess:
    def __init__(self):
        self.processes = {}

    def register(self, name: str, target: callable):
        self.processes[name] = {"target": target, "task": None}

    async def heal(self):
        for name, proc in self.processes.items():
            if proc["task"] is None or proc["task"].done():
                logger.warning(f"Process '{name}' died. Restarting...")
                proc["task"] = asyncio.create_task(self._run_wrapper(name, proc["target"]))

    async def _run_wrapper(self, name: str, target: callable):
        try:
            await target()
        except Exception as e:
            logger.error(f"Process '{name}' crashed: {e}")
            self.processes[name]["task"] = None

    async def monitor_loop(self):
        while True:
            await self.heal()
            await asyncio.sleep(30)


async def pulse_loop(pulse: PulseSystem):
    while True:
        try:
            await pulse.pulse()
        except Exception as e:
            logger.error(f"Pulse failed: {e}")
        await asyncio.sleep(PULSE_INTERVAL)


async def evolution_loop(evolution: EvolutionaryLoop, pulse: PulseSystem):
    while True:
        try:
            balance = pulse.checks.get("balance", 0.0)
            await evolution.evolve(balance)
        except Exception as e:
            logger.error(f"Evolution cycle failed: {e}")
        await asyncio.sleep(EVOLUTION_INTERVAL)


async def main():
    logger.info("=== Aladdin Autonomous System Starting ===")
    logger.info(f"Mode: ACCUMULATION (threshold=${ACCUMULATION_THRESHOLD})")
    logger.info(f"Scaling factor: {SCALING_FACTOR} per profitable cycle")
    logger.info(f"Pulse interval: {PULSE_INTERVAL}s | Evolution interval: {EVOLUTION_INTERVAL}s")

    pulse = PulseSystem()
    memory = EvolutionaryMemory()
    evolution = EvolutionaryLoop(memory)
    healer = SelfHealingProcess()

    healer.register("pulse", lambda: pulse_loop(pulse))
    healer.register("evolution", lambda: evolution_loop(evolution, pulse))

    await healer.monitor_loop()


if __name__ == "__main__":
    asyncio.run(main())
