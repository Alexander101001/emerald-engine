"""Auto-synthesized adapter from n8n-io/n8n.
Stars: 194039 | Topics: ai, apis, automation, cli, data-flow, development, integration-framework, integrations, ipaas, low-code, low-code-platform, mcp, mcp-client, mcp-server, n8n, no-code, self-hosted, typescript, workflow, workflow-automation
"""

import asyncio
import logging
from opcode_base import ParadigmAgentBase


class N8NAdapter(ParadigmAgentBase):
    """Adapter synthesised from https://github.com/n8n-io/n8n."""

    def __init__(self, telemetry=None):
        super().__init__(telemetry=telemetry)
        self._name = "n8n"

    async def execute(self):
        logging.info(f"[{self._name}] Executing synthesised capability")
        self._emit_telemetry("adapter_execute", name="{self._name}")
        return {"status": "ok", "source": "n8n-io/n8n"}

    async def execution_loop(self):
        await self.execute()
        await self._hot_daemon_loop(self.execute, 7200)
