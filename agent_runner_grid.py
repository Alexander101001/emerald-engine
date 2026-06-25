import asyncio
import json
import logging
import os
import time
from pathlib import Path

from opcode_base import ParadigmAgentBase

GRID_INTERVAL = int(os.getenv("GRID_INTERVAL", "600"))
GRID_STATE_PATH = Path("/tmp/emerald_runner_grid.json")
MAX_RUNNERS = 20
ACTIVE_LIMIT = 2


class RunnerGridAgent(ParadigmAgentBase):
    """SECTION 5: 20-runner rotational grid, 2 active at a time, dual pipeline."""

    def __init__(self, telemetry=None):
        super().__init__(telemetry=telemetry)
        self._cycle = 0

    def _load_grid(self) -> dict:
        if GRID_STATE_PATH.exists():
            try:
                return json.loads(GRID_STATE_PATH.read_text())
            except Exception:
                pass
        return {
            "runners": [],
            "pipeline_a_active": False,
            "pipeline_b_active": False,
            "current_shift": 0,
        }

    def _save_grid(self, grid: dict):
        GRID_STATE_PATH.write_text(json.dumps(grid, indent=2))

    def _initialize_runners(self) -> list:
        runners = []
        for i in range(1, MAX_RUNNERS + 1):
            runners.append({
                "id": f"runner-{i:03d}",
                "pipeline": "A" if i % 2 == 1 else "B",
                "status": "idle",
                "shifts_completed": 0,
                "last_active": 0,
                "assigned_task": "",
            })
        return runners

    async def _rotate_shift(self, grid: dict) -> dict:
        runners = grid.get("runners", [])
        if not runners:
            runners = self._initialize_runners()
        current_shift = grid.get("current_shift", 0)
        for runner in runners:
            runner["status"] = "idle"
        shift_size = ACTIVE_LIMIT
        a_runners = [r for r in runners if r["pipeline"] == "A"]
        b_runners = [r for r in runners if r["pipeline"] == "B"]
        a_idx = (current_shift // 2) % max(len(a_runners), 1)
        b_idx = (current_shift // 2) % max(len(b_runners), 1)
        if a_runners:
            a_runners[a_idx]["status"] = "active"
            a_runners[a_idx]["last_active"] = time.time()
            a_runners[a_idx]["shifts_completed"] += 1
            a_runners[a_idx]["assigned_task"] = "pipeline_a_scout_synth_evolve"
        if b_runners:
            b_runners[b_idx]["status"] = "active"
            b_runners[b_idx]["last_active"] = time.time()
            b_runners[b_idx]["shifts_completed"] += 1
            b_runners[b_idx]["assigned_task"] = "pipeline_b_dashboard_cmd_exec"
        grid["runners"] = runners
        grid["pipeline_a_active"] = True
        grid["pipeline_b_active"] = True
        grid["current_shift"] = current_shift + 1
        self._save_grid(grid)
        return grid

    async def grid_cycle(self):
        self._cycle += 1
        logging.info(f"=== Runner Grid Cycle #{self._cycle} ===")
        start = time.time()
        grid = self._load_grid()
        grid = await self._rotate_shift(grid)
        active = len([r for r in grid.get("runners", []) if r["status"] == "active"])
        idle = len([r for r in grid.get("runners", []) if r["status"] == "idle"])
        total_shifts = sum(r.get("shifts_completed", 0) for r in grid.get("runners", []))
        logging.info(f"  Runners: {active} active / {idle} idle (limit {ACTIVE_LIMIT}), "
                     f"Total shifts: {total_shifts}, "
                     f"Pipeline A: {grid.get('pipeline_a_active')}, "
                     f"Pipeline B: {grid.get('pipeline_b_active')}")
        self._emit_telemetry("runner_grid", active=active, idle=idle,
                              shifts=total_shifts, cycle=self._cycle)

    async def execution_loop(self):
        logging.info("Runner Grid activated — SECTION 5: 20-runner rotational pool, 2 active at a time")
        await self.grid_cycle()
        await self._hot_daemon_loop(self.grid_cycle, GRID_INTERVAL)


async def run_runner_grid_loop(telemetry=None):
    agent = RunnerGridAgent(telemetry=telemetry)
    await agent.execution_loop()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [GRID] %(levelname)s %(message)s")
    asyncio.run(run_runner_grid_loop())
