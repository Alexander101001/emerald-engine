import asyncio
import json
import logging
import os
import subprocess
import time
from pathlib import Path

from opcode_base import ParadigmAgentBase
from crypto_vault import EmeraldCryptoVault

STATE_INTERVAL = int(os.getenv("STATE_RELAY_INTERVAL", "600"))
STATE_BRANCH = os.getenv("STATE_BRANCH", "emerald-state")
STATE_PATH = Path("/tmp/emerald_engine_state.json")
REPO_PATH = Path(os.getenv("REPO_PATH", "/tmp/opencode/emerald-engine"))

RAM_THRESHOLD = int(os.getenv("RAM_THRESHOLD_PCT", "80"))
DISK_THRESHOLD = int(os.getenv("DISK_THRESHOLD_PCT", "80"))
JOB_TIME_LIMIT = int(os.getenv("JOB_TIME_LIMIT_MINUTES", "330"))


class StateRelayAgent(ParadigmAgentBase):
    """SECTION 4: Monitor resources, serialize state, encrypt, commit to state branch, dispatch."""

    def __init__(self, telemetry=None):
        super().__init__(telemetry=telemetry)
        self.vault = EmeraldCryptoVault()
        self._cycle = 0
        self._start_time = time.time()

    def _read_resource_usage(self) -> dict:
        usage = {"time_elapsed_minutes": round((time.time() - self._start_time) / 60, 1)}
        try:
            with open("/proc/meminfo") as f:
                lines = f.readlines()
            mem_total = int([l for l in lines if "MemTotal" in l][0].split()[1])
            mem_avail = int([l for l in lines if "MemAvailable" in l][0].split()[1])
            mem_used = mem_total - mem_avail
            usage["ram_used_mb"] = round(mem_used / 1024, 1)
            usage["ram_total_mb"] = round(mem_total / 1024, 1)
            usage["ram_percent"] = round(100.0 * mem_used / mem_total, 1)
        except Exception:
            usage["ram_percent"] = 0.0
        try:
            s = os.statvfs("/")
            total = s.f_frsize * s.f_blocks
            free = s.f_frsize * s.f_bfree
            used = total - free
            usage["disk_used_gb"] = round(used / (1024 ** 3), 1)
            usage["disk_total_gb"] = round(total / (1024 ** 3), 1)
            usage["disk_percent"] = round(100.0 * used / total, 1)
        except Exception:
            usage["disk_percent"] = 0.0
        try:
            with open("/proc/stat") as f:
                line = f.readline()
            parts = line.strip().split()
            usage["cpu_user"] = int(parts[1]) if len(parts) > 1 else 0
        except Exception:
            pass
        return usage

    def _check_thresholds(self, usage: dict) -> dict:
        alerts = []
        if usage.get("ram_percent", 0) >= RAM_THRESHOLD:
            alerts.append(f"RAM at {usage['ram_percent']}% (threshold {RAM_THRESHOLD}%)")
        if usage.get("disk_percent", 0) >= DISK_THRESHOLD:
            alerts.append(f"Disk at {usage['disk_percent']}% (threshold {DISK_THRESHOLD}%)")
        if usage.get("time_elapsed_minutes", 0) >= JOB_TIME_LIMIT:
            alerts.append(f"Job time {usage['time_elapsed_minutes']}m (limit {JOB_TIME_LIMIT}m)")
        return {"alerts": alerts, "triggered": len(alerts) > 0}

    def _collect_state(self) -> dict:
        state = {
            "timestamp": time.time(),
            "cycle": self._cycle,
            "resources": self._read_resource_usage(),
        }
        for report_name in ["scout", "eval", "synth", "hunter", "expansion",
                            "hacker_bot", "qa", "orchestrator"]:
            path = Path(f"/tmp/emerald_{report_name}_report.json")
            if path.exists():
                try:
                    state[f"{report_name}_report"] = json.loads(path.read_text())
                except Exception:
                    pass
        state["files"] = [str(f.relative_to(REPO_PATH)) for f in REPO_PATH.glob("*.py")
                          if ".aegis" not in str(f)]
        state["go_binaries"] = [str(f) for f in REPO_PATH.glob("*.go")]
        return state

    async def _commit_state(self, state: dict, alerts: dict):
        encrypted = self.vault.encrypt_data_payload(json.dumps(state))
        STATE_PATH.write_bytes(encrypted)
        try:
            subprocess.run(["git", "checkout", "-b", STATE_BRANCH],
                           capture_output=True, timeout=10, cwd=REPO_PATH)
        except Exception:
            subprocess.run(["git", "checkout", STATE_BRANCH],
                           capture_output=True, timeout=10, cwd=REPO_PATH)
        state_rel_path = Path("tmp/emerald_engine_state.json.enc")
        state_path_abs = REPO_PATH / state_rel_path
        state_path_abs.parent.mkdir(parents=True, exist_ok=True)
        state_path_abs.write_bytes(encrypted)
        subprocess.run(["git", "add", str(state_rel_path)],
                       capture_output=True, timeout=10, cwd=REPO_PATH)
        ts = int(time.time())
        msg = f"emerald:state-relay-cycle-{self._cycle}-ts-{ts}"
        subprocess.run(["git", "commit", "-m", msg],
                       capture_output=True, timeout=10, cwd=REPO_PATH,
                       env={**os.environ,
                            "GIT_AUTHOR_NAME": "Emerald StateRelay",
                            "GIT_AUTHOR_EMAIL": "state-relay@emerald.engine"})
        subprocess.run(["git", "push", "origin", STATE_BRANCH],
                       capture_output=True, timeout=30, cwd=REPO_PATH)
        logging.info(f"  State committed to {STATE_BRANCH}: {msg}")
        if alerts.get("triggered"):
            logging.warning(f"  Threshold alerts triggered: {alerts['alerts']}")

    async def state_cycle(self):
        self._cycle += 1
        logging.info(f"=== State Relay Cycle #{self._cycle} ===")
        start = time.time()
        usage = self._read_resource_usage()
        alerts = self._check_thresholds(usage)
        logging.info(f"  Resources: RAM {usage.get('ram_percent', '?')}%, "
                     f"Disk {usage.get('disk_percent', '?')}%, "
                     f"Time {usage.get('time_elapsed_minutes', '?')}m")
        state = self._collect_state()
        await self._commit_state(state, alerts)
        self._emit_telemetry("state_relay", **usage, alerts=alerts["alerts"])

    async def execution_loop(self):
        logging.info("State Relay activated — SECTION 4: resource monitor + state encryption + dispatch")
        await self.state_cycle()
        await self._hot_daemon_loop(self.state_cycle, STATE_INTERVAL)


async def run_state_relay_loop(telemetry=None):
    agent = StateRelayAgent(telemetry=telemetry)
    await agent.execution_loop()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [STATE] %(levelname)s %(message)s")
    asyncio.run(run_state_relay_loop())
