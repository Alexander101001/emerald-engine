import asyncio
import json
import logging
import os
import subprocess
import time
from pathlib import Path

from opcode_base import ParadigmAgentBase
from crypto_vault import EmeraldCryptoVault

GIT_INTERVAL = int(os.getenv("GIT_LIFECYCLE_INTERVAL", "3600"))
REPO_PATH = Path(os.getenv("REPO_PATH", "/tmp/opencode/emerald-engine"))
GIT_REMOTE = os.getenv("GITHUB_REMOTE", "origin")


class GitLifecycleManagerAgent(ParadigmAgentBase):
    """SECTION 10: Retrieve encrypted token, auto-commit, push to main."""

    def __init__(self, telemetry=None):
        super().__init__(telemetry=telemetry)
        self.vault = EmeraldCryptoVault()
        self._cycle = 0

    def _get_token(self) -> str:
        env_token = os.getenv("GITHUB_TOKEN", "")
        if env_token:
            return env_token
        env_pat = os.getenv("GITHUB_PAT", "")
        if env_pat:
            return env_pat
        registry_path = Path("/tmp/opencode/emerald-engine/.secrets/platforms.enc")
        if registry_path.exists():
            try:
                registry = json.loads(self.vault.decrypt_data_payload(registry_path.read_bytes()))
                for pid, pdata in registry.items():
                    if isinstance(pdata, dict) and pdata.get("token_encrypted"):
                        try:
                            tok = self.vault.decrypt_data_payload(bytes.fromhex(pdata["token_encrypted"]))
                            if tok.startswith("ghp_") or tok.startswith("github_pat_"):
                                return tok
                        except Exception:
                            continue
            except Exception:
                pass
        return ""

    async def _check_uncommitted(self) -> list:
        r = subprocess.run(["git", "status", "--porcelain"],
                           capture_output=True, timeout=10, cwd=REPO_PATH)
        if r.returncode == 0:
            lines = r.stdout.decode().strip().split("\n") if r.stdout.strip() else []
            return [l.strip() for l in lines if l.strip()]
        return []

    async def _git_auto_commit(self, uncommitted: list) -> bool:
        try:
            subprocess.run(["git", "add", "-A"],
                           capture_output=True, timeout=10, cwd=REPO_PATH)
            ts = int(time.time())
            msg = f"emerald:evolve-cycle-{self._cycle}-ts-{ts}"
            r = subprocess.run(
                ["git", "commit", "-m", msg],
                capture_output=True, timeout=10, cwd=REPO_PATH,
                env={**os.environ,
                     "GIT_AUTHOR_NAME": "Emerald GitLifecycle",
                     "GIT_AUTHOR_EMAIL": "git-lifecycle@emerald.engine",
                     "GIT_COMMITTER_NAME": "Emerald GitLifecycle",
                     "GIT_COMMITTER_EMAIL": "git-lifecycle@emerald.engine"},
            )
            if r.returncode == 0:
                logging.info(f"  Committed: {msg}")
                return True
            else:
                logging.info(f"  Commit skipped (nothing to commit): {r.stderr.decode()[:100]}")
                return False
        except Exception as e:
            logging.error(f"  Commit failed: {e}")
            return False

    async def _git_push(self) -> bool:
        token = self._get_token()
        if not token:
            logging.warning("  No GitHub token available, skipping push")
            return False
        try:
            remote_url = f"https://x-access-token:{token}@github.com/anomalyco/emerald-engine.git"
            r = subprocess.run(
                ["git", "push", remote_url, "HEAD:main"],
                capture_output=True, timeout=60, cwd=REPO_PATH,
            )
            if r.returncode == 0:
                logging.info("  Pushed to main")
                return True
            else:
                logging.warning(f"  Push failed: {r.stderr.decode()[:200]}")
                return False
        except Exception as e:
            logging.error(f"  Push error: {e}")
            return False

    async def git_cycle(self):
        self._cycle += 1
        logging.info(f"=== Git Lifecycle Cycle #{self._cycle} ===")
        start = time.time()
        uncommitted = await self._check_uncommitted()
        logging.info(f"  Uncommitted changes: {len(uncommitted)}")
        if uncommitted:
            committed = await self._git_auto_commit(uncommitted)
            if committed:
                pushed = await self._git_push()
                logging.info(f"  Pushed: {pushed}")
            else:
                logging.info("  Nothing to push")
        self._emit_telemetry("git_cycle", uncommitted=len(uncommitted),
                              cycle=self._cycle)

    async def execution_loop(self):
        logging.info("Git Lifecycle activated — SECTION 10: auto-commit + push to main")
        await self.git_cycle()
        await self._hot_daemon_loop(self.git_cycle, GIT_INTERVAL)


async def run_git_lifecycle_loop(telemetry=None):
    agent = GitLifecycleManagerAgent(telemetry=telemetry)
    await agent.execution_loop()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [GIT] %(levelname)s %(message)s")
    asyncio.run(run_git_lifecycle_loop())
