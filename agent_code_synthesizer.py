import asyncio
import json
import logging
import os
import subprocess
import time
from pathlib import Path

from opcode_base import ParadigmAgentBase

SYNTH_INTERVAL = int(os.getenv("SYNTH_INTERVAL", "3600"))
SYNTH_REPORT_PATH = Path("/tmp/emerald_synth_report.json")
REPO_PATH = Path(os.getenv("REPO_PATH", "/tmp/opencode/emerald-engine"))


class CodeSynthesizerAgent(ParadigmAgentBase):
    """SECTION 3: Merge validated skills into source tree, resolve deps, refactor."""

    def __init__(self, telemetry=None):
        super().__init__(telemetry=telemetry)
        self._cycle = 0

    def _load_recommended(self) -> list:
        path = Path("/tmp/emerald_eval_report.json")
        if path.exists():
            try:
                data = json.loads(path.read_text())
                return [e["candidate"] for e in data.get("evaluations", [])
                        if e.get("recommended") and "candidate" in e]
            except Exception:
                pass
        return []

    async def _resolve_dependencies(self) -> dict:
        results = {}
        req_file = REPO_PATH / "requirements.txt"
        if req_file.exists():
            try:
                r = subprocess.run(["pip3", "list", "--format=json"],
                                   capture_output=True, timeout=30,
                                   cwd=REPO_PATH)
                installed = json.loads(r.stdout) if r.returncode == 0 else []
                installed_names = {p["name"].lower() for p in installed}
                required = []
                for line in req_file.read_text().split("\n"):
                    line = line.strip()
                    if line and not line.startswith("#"):
                        pkg = line.split("==")[0].split(">=")[0].split("<")[0].strip().lower()
                        if pkg and pkg not in ("", "-e", "."):
                            required.append(pkg)
                missing = [p for p in required if p not in installed_names]
                results["required"] = len(required)
                results["installed"] = len(installed_names & set(required))
                results["missing"] = missing
            except Exception as e:
                results["error"] = str(e)[:80]
        go_mod = REPO_PATH / "go.mod"
        if go_mod.exists():
            try:
                r = subprocess.run(["go", "list", "-m", "all"],
                                   capture_output=True, timeout=30,
                                   cwd=REPO_PATH)
                results["go_modules"] = len(r.stdout.decode().split("\n")) if r.returncode == 0 else 0
            except Exception:
                pass
        return results

    async def _synthesize_patches(self, recommended: list) -> list:
        patches = []
        for candidate in recommended[:5]:
            full_name = candidate.get("full_name", "")
            name_part = full_name.split("/")[-1] if "/" in full_name else full_name
            topics = candidate.get("topics", [])
            lang = candidate.get("language", "python").lower()
            rel_path = f"integrations/{name_part.lower().replace('-', '_')}_adapter.py"
            module_path = REPO_PATH / rel_path
            module_path.parent.mkdir(parents=True, exist_ok=True)
            if lang == "python":
                content = self._gen_python_adapter(name_part, candidate)
            elif lang == "go":
                content = self._gen_go_adapter(name_part, candidate)
            elif lang == "rust":
                content = self._gen_rust_adapter(name_part, candidate)
            else:
                content = self._gen_python_adapter(name_part, candidate)
            if content:
                module_path.write_text(content)
                patches.append({"file": rel_path, "language": lang, "from": full_name})
                logging.info(f"  Synthesized {rel_path} from {full_name}")
        return patches

    def _gen_python_adapter(self, name: str, candidate: dict) -> str:
        safe_name = name.lower().replace("-", "_").replace(".", "_")
        return (
            f'"""Auto-synthesized adapter from {candidate.get("full_name", name)}.\n'
            f"Stars: {candidate.get('stars', 0)} | "
            f"Topics: {', '.join(candidate.get('topics', []))}\n"
            f'"""\n'
            f'\n'
            f'import asyncio\n'
            f'import logging\n'
            f'from opcode_base import ParadigmAgentBase\n'
            f'\n'
            f'\n'
            f'class {safe_name.title().replace("_", "")}Adapter(ParadigmAgentBase):\n'
            f'    """Adapter synthesised from {candidate.get("html_url", "")}."""\n'
            f'\n'
            f'    def __init__(self, telemetry=None):\n'
            f'        super().__init__(telemetry=telemetry)\n'
            f'        self._name = "{safe_name}"\n'
            f'\n'
            f'    async def execute(self):\n'
            f'        logging.info(f"[{self._name}] Executing synthesised capability")\n'
            f'        self._emit_telemetry("adapter_execute", name=self._name)\n'
            f'        return {{"status": "ok", "source": "{candidate.get("full_name", name)}"}}\n'
            f'\n'
            f'    async def execution_loop(self):\n'
            f'        await self.execute()\n'
            f'        await self._hot_daemon_loop(self.execute, 7200)\n'
        )

    def _gen_go_adapter(self, name: str, candidate: dict) -> str:
        safe_name = name.lower().replace("-", "_")
        return (
            f'package main\n'
            f'\n'
            f'import (\n'
            f'\t"log"\n'
            f')\n'
            f'\n'
            f'// {safe_name}Adapter synthesised from {candidate.get("full_name", name)}\n'
            f'type {safe_name}Adapter struct {{\n'
            f'\tName string\n'
            f'}}\n'
            f'\n'
            f'func New{safe_name}Adapter() *{safe_name}Adapter {{\n'
            f'\treturn &{safe_name}Adapter{{Name: "{safe_name}"}}\n'
            f'}}\n'
            f'\n'
            f'func (a *{safe_name}Adapter) Execute() {{\n'
            f'\tlog.Printf("[%s] Executing synthesised capability", a.Name)\n'
            f'}}\n'
        )

    def _gen_rust_adapter(self, name: str, candidate: dict) -> str:
        safe_name = name.lower().replace("-", "_").replace(".", "_")
        return (
            f'/// Auto-synthesised adapter from {candidate.get("full_name", name)}\n'
            f'struct {safe_name}Adapter {{\n'
            f'    name: String,\n'
            f'}}\n'
            f'\n'
            f'impl {safe_name}Adapter {{\n'
            f'    fn new() -> Self {{\n'
            f'        Self {{ name: "{safe_name}".to_string() }}\n'
            f'    }}\n'
            f'\n'
            f'    fn execute(&self) {{\n'
            f'        println!("[{{}}] Executing synthesised capability", self.name);\n'
            f'    }}\n'
            f'}}\n'
        )

    async def synth_cycle(self):
        self._cycle += 1
        logging.info(f"=== Code Synthesizer Cycle #{self._cycle} ===")
        start = time.time()
        recommended = self._load_recommended()
        deps = await self._resolve_dependencies()
        patches = await self._synthesize_patches(recommended)
        logging.info(f"  Recommended: {len(recommended)}, Deps: {deps}, Patches: {len(patches)}")
        report = {
            "cycle": self._cycle,
            "time": time.time(),
            "elapsed": round(time.time() - start, 2),
            "dependencies": deps,
            "patches_generated": patches,
        }
        SYNTH_REPORT_PATH.write_text(json.dumps(report, indent=2))
        self._emit_telemetry("synth_cycle", patches=len(patches),
                              deps_missing=len(deps.get("missing", [])))

    async def execution_loop(self):
        logging.info("Code Synthesizer activated — SECTION 3: merge skills, resolve deps, refactor")
        await self.synth_cycle()
        await self._hot_daemon_loop(self.synth_cycle, SYNTH_INTERVAL)


async def run_synthesizer_loop(telemetry=None):
    agent = CodeSynthesizerAgent(telemetry=telemetry)
    await agent.execution_loop()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [SYNTH] %(levelname)s %(message)s")
    asyncio.run(run_synthesizer_loop())
