import asyncio
import json
import logging
import os
import time
from pathlib import Path

from opcode_base import ParadigmAgentBase

CHAT_INTERVAL = int(os.getenv("CHAT_INTERVAL", "30"))
CHAT_STREAM_FILE = Path("/tmp/emerald_chat_stream.json")
CHAT_HISTORY = Path("/tmp/emerald_chat_history.json")


class ChatInterpreterAgent(ParadigmAgentBase):
    """SECTION 9: Async polling loop, parse directives, route to core agents."""

    def __init__(self, telemetry=None):
        super().__init__(telemetry=telemetry)
        self._cycle = 0
        self._processed_ids = set()

    COMMAND_ROUTES = {
        "scan": "trend_scouter",
        "evaluate": "code_evaluator",
        "synthesize": "code_synthesizer",
        "deploy": "cloud_manager",
        "audit": "devsecops",
        "test": "qa_automation",
        "status": "orchestrator",
        "help": "chat_interpreter",
        "dashboard": "dashboard_compiler",
        "identities": "identity_manager",
    }

    def _load_stream(self) -> list:
        if CHAT_STREAM_FILE.exists():
            try:
                data = json.loads(CHAT_STREAM_FILE.read_text())
                return data if isinstance(data, list) else [data]
            except Exception:
                pass
        return []

    def _load_history(self) -> list:
        if CHAT_HISTORY.exists():
            try:
                return json.loads(CHAT_HISTORY.read_text())
            except Exception:
                pass
        return []

    def _save_history(self, history: list):
        CHAT_HISTORY.write_text(json.dumps(history[-200:], indent=2))

    def _parse_directive(self, text: str) -> dict:
        text_lower = text.lower().strip()
        for cmd, route in self.COMMAND_ROUTES.items():
            if text_lower.startswith(cmd) or text_lower.startswith(f"/{cmd}"):
                args = text[len(cmd):].strip().lstrip("/")
                return {
                    "command": cmd,
                    "route": route,
                    "args": args,
                    "raw": text,
                    "parsed": True,
                }
        return {
            "command": "unknown",
            "route": None,
            "args": text,
            "raw": text,
            "parsed": False,
        }

    async def _execute_command(self, directive: dict) -> dict:
        cmd = directive.get("command", "unknown")
        if cmd == "help":
            available = ", ".join(f"/{c}" for c in self.COMMAND_ROUTES)
            return {"response": f"Available commands: {available}"}
        elif cmd == "status":
            reports = []
            for p in ["/tmp/emerald_orchestrator_report.json", "/tmp/emerald_scout_report.json",
                      "/tmp/emerald_qa_report.json", "/tmp/emerald_compliance.json"]:
                rp = Path(p)
                if rp.exists():
                    try:
                        reports.append({p: json.loads(rp.read_text()).get("all_passed", "?")})
                    except Exception:
                        pass
            return {"response": json.dumps(reports)}
        elif cmd == "scan":
            return {"response": "Triggering scout cycle (next scheduled interval)"}
        elif cmd == "audit":
            return {"response": "Triggering security audit (next scheduled interval)"}
        else:
            route = directive.get("route", "unknown")
            return {"response": f"Command '{cmd}' routed to {route}"}

    async def chat_cycle(self):
        self._cycle += 1
        stream = self._load_stream()
        if not stream:
            return
        history = self._load_history()
        new_commands = 0
        for msg in stream:
            msg_id = msg.get("id", "") or str(hash(str(msg)))
            if msg_id in self._processed_ids:
                continue
            text = msg.get("text", msg.get("message", msg.get("command", "")))
            if not text:
                continue
            directive = self._parse_directive(text)
            if directive["parsed"]:
                result = await self._execute_command(directive)
                entry = {
                    "id": msg_id,
                    "timestamp": time.time(),
                    "directive": directive,
                    "result": result,
                }
                history.append(entry)
                new_commands += 1
                logging.info(f"  Chat command: {directive['command']} -> {result['response'][:60]}")
            self._processed_ids.add(msg_id)
        self._save_history(history)
        if len(self._processed_ids) > 1000:
            self._processed_ids = set(list(self._processed_ids)[-500:])
        self._emit_telemetry("chat_cycle", processed=new_commands,
                              total_history=len(history))

    async def execution_loop(self):
        logging.info("Chat Interpreter activated — SECTION 9: async polling + command routing")
        while True:
            try:
                await self.chat_cycle()
            except Exception as e:
                logging.error(f"Chat cycle error: {e}")
            await asyncio.sleep(CHAT_INTERVAL)


async def run_chat_loop(telemetry=None):
    agent = ChatInterpreterAgent(telemetry=telemetry)
    await agent.execution_loop()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [CHAT] %(levelname)s %(message)s")
    asyncio.run(run_chat_loop())
