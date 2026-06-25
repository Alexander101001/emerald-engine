"""Auto-synthesized adapter from donnemartin/system-design-primer.
Stars: 354819 | Topics: design, design-patterns, design-system, development, interview, interview-practice, interview-questions, programming, python, system, web, web-application, webapp
"""

import asyncio
import logging
from opcode_base import ParadigmAgentBase


class SystemDesignPrimerAdapter(ParadigmAgentBase):
    """Adapter synthesised from https://github.com/donnemartin/system-design-primer."""

    def __init__(self, telemetry=None):
        super().__init__(telemetry=telemetry)
        self._name = "system_design_primer"

    async def execute(self):
        logging.info(f"[{self._name}] Executing synthesised capability")
        self._emit_telemetry("adapter_execute", name="{self._name}")
        return {"status": "ok", "source": "donnemartin/system-design-primer"}

    async def execution_loop(self):
        await self.execute()
        await self._hot_daemon_loop(self.execute, 7200)
