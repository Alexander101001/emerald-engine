import asyncio
import json
import logging
import os
import time
from pathlib import Path

from opcode_base import ParadigmAgentBase

DASHBOARD_INTERVAL = int(os.getenv("DASHBOARD_INTERVAL", "3600"))
DASHBOARD_DIR = Path("/tmp/emerald_dashboard")
DASHBOARD_JSON = DASHBOARD_DIR / "dashboard.json"
DASHBOARD_HTML = DASHBOARD_DIR / "index.html"


class DashboardCompilerAgent(ParadigmAgentBase):
    """SECTION 8: Static live web dashboard every 60 min, deploy to GitHub Pages."""

    def __init__(self, telemetry=None):
        super().__init__(telemetry=telemetry)
        self._cycle = 0

    def _collect_metrics(self) -> dict:
        metrics = {
            "timestamp": time.time(),
            "cycle": self._cycle,
            "agents": {},
        }
        report_paths = {
            "scouter": "/tmp/emerald_scout_report.json",
            "evaluator": "/tmp/emerald_eval_report.json",
            "synthesizer": "/tmp/emerald_synth_report.json",
            "state_relay": "/tmp/emerald_engine_state.json",
            "runner_grid": "/tmp/emerald_runner_grid.json",
            "hf_sync": "/tmp/emerald_hf_sync.json",
            "hunter": "/tmp/hunter_report.json",
            "expansion": "/tmp/expansion_report.json",
            "hacker_bot": "/tmp/hacker_bot_report.json",
            "qa": "/tmp/emerald_qa_report.json",
            "orchestrator": "/tmp/emerald_orchestrator_report.json",
            "cloud_manager": "/tmp/opencode_telemetry.json",
            "compliance": "/tmp/emerald_compliance.json",
        }
        for name, path in report_paths.items():
            p = Path(path)
            if p.exists():
                try:
                    metrics["agents"][name] = json.loads(p.read_text())
                except Exception:
                    metrics["agents"][name] = {"error": "parse_failed"}
            else:
                metrics["agents"][name] = {"status": "no_report"}
        try:
            with open("/proc/meminfo") as f:
                lines = f.readlines()
            mem_total = int([l for l in lines if "MemTotal" in l][0].split()[1]) // 1024
            mem_avail = int([l for l in lines if "MemAvailable" in l][0].split()[1]) // 1024
            metrics["system"] = {
                "ram_total_mb": mem_total,
                "ram_used_mb": mem_total - mem_avail,
                "ram_percent": round(100.0 * (mem_total - mem_avail) / mem_total, 1),
            }
        except Exception:
            metrics["system"] = {}
        return metrics

    def _generate_html(self, metrics: dict) -> str:
        agents = metrics.get("agents", {})
        system = metrics.get("system", {})
        ram_pct = system.get("ram_percent", 0)
        ram_color = "green" if ram_pct < 60 else ("orange" if ram_pct < 80 else "red")
        agent_rows = ""
        for name, data in sorted(agents.items()):
            status = "ok" if isinstance(data, dict) and "error" not in data else "missing"
            agent_rows += (
                f"<tr><td>{name}</td>"
                f"<td style='color:{'lime' if status=='ok' else 'red'}'>{status}</td>"
                f"<td>{json.dumps(data)[:80]}</td></tr>\n"
            )
        return (
            "<!DOCTYPE html>\n"
            "<html lang='en'><head><meta charset='UTF-8'>\n"
            "<meta name='viewport' content='width=device-width, initial-scale=1.0'>\n"
            "<title>Emerald Engine — Live Dashboard</title>\n"
            "<style>\n"
            "body{font-family:monospace;background:#0a0e14;color:#c0caf5;margin:20px}\n"
            "h1{color:#7dcfff;border-bottom:1px solid #364a5e}\n"
            "h2{color:#bb9af7}.metric{display:inline-block;margin:10px;padding:15px;"
            "background:#1a1b26;border-radius:6px;min-width:150px}\n"
            ".metric .label{color:#565f89}.metric .value{font-size:1.5em;font-weight:bold}\n"
            "table{width:100%;border-collapse:collapse;margin-top:10px}\n"
            "th,td{text-align:left;padding:6px 12px;border-bottom:1px solid #364a5e}\n"
            "th{color:#7dcfff}.green{color:#9ece6a}.orange{color:#e0af68}.red{color:#f7768e}\n"
            ".footer{margin-top:30px;color:#565f89;font-size:0.8em}\n"
            "</style></head><body>\n"
            "<h1>Emerald Engine — Live System Dashboard</h1>\n"
            f"<div class='metric'><div class='label'>RAM Usage</div>"
            f"<div class='value {ram_color}'>{ram_pct}%</div></div>\n"
            f"<div class='metric'><div class='label'>Cycle</div>"
            f"<div class='value'>{self._cycle}</div></div>\n"
            f"<div class='metric'><div class='label'>Agents Reporting</div>"
            f"<div class='value'>{len(agents)}</div></div>\n"
            f"<div class='metric'><div class='label'>Generated</div>"
            f"<div class='value' style='font-size:0.8em'>{time.strftime('%H:%M UTC')}</div></div>\n"
            "<h2>Agent Reports</h2>\n"
            "<table><tr><th>Agent</th><th>Status</th><th>Preview</th></tr>\n"
            f"{agent_rows}</table>\n"
            "<div class='footer'>Emerald Engine — Self-Mutating SaaS Orchestrator</div>\n"
            "</body></html>\n"
        )

    async def dashboard_cycle(self):
        self._cycle += 1
        logging.info(f"=== Dashboard Compiler Cycle #{self._cycle} ===")
        start = time.time()
        DASHBOARD_DIR.mkdir(parents=True, exist_ok=True)
        metrics = self._collect_metrics()
        DASHBOARD_JSON.write_text(json.dumps(metrics, indent=2))
        html = self._generate_html(metrics)
        DASHBOARD_HTML.write_text(html)
        logging.info(f"  Dashboard compiled: {DASHBOARD_JSON} ({len(json.dumps(metrics))} bytes), "
                     f"{DASHBOARD_HTML} ({len(html)} bytes)")
        self._emit_telemetry("dashboard_cycle", agents_reporting=len(metrics.get("agents", {})))

    async def execution_loop(self):
        logging.info("Dashboard Compiler activated — SECTION 8: static dashboard every 60 min")
        await self.dashboard_cycle()
        await self._hot_daemon_loop(self.dashboard_cycle, DASHBOARD_INTERVAL)


async def run_dashboard_loop(telemetry=None):
    agent = DashboardCompilerAgent(telemetry=telemetry)
    await agent.execution_loop()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [DASH] %(levelname)s %(message)s")
    asyncio.run(run_dashboard_loop())
