import asyncio
import json
import logging
import os
import re
import subprocess
import sys
import time
from pathlib import Path

import aiohttp

from crypto_vault import EmeraldCryptoVault
from opcode_base import ParadigmAgentBase

SCAN_INTERVAL = int(os.getenv("HACKER_BOT_INTERVAL", "3600"))
SENTINEL_URL = os.getenv("SENTINEL_URL", "http://localhost:8443")
SENTINEL_AUTH_TOKEN = os.getenv("SENTINEL_AUTH_TOKEN", "")
REPO_PATH = Path(os.getenv("REPO_PATH", "/tmp/opencode/emerald-engine"))
HF_SPACE_URL = os.getenv("HF_SPACE_URL", "https://alexandergreater90-emerald-engine.hf.space")
GITHUB_REMOTE = os.getenv("GITHUB_REMOTE", "origin")

SECRET_PATTERNS = [
    (r'sk-[a-zA-Z0-9]{20,}', "OpenAI/Stripe API key"),
    (r'ghp_[a-zA-Z0-9]{36}', "GitHub Personal Access Token"),
    (r'hf_[a-zA-Z0-9]{36}', "Hugging Face Token"),
    (r'gho_[a-zA-Z0-9]{36}', "GitHub OAuth Access Token"),
    (r'AKIA[0-9A-Z]{16}', "AWS Access Key"),
    (r'-----BEGIN RSA PRIVATE KEY-----', "RSA Private Key"),
    (r'-----BEGIN OPENSSH PRIVATE KEY-----', "OpenSSH Private Key"),
]

LEAK_EXCLUSIONS = [
    ".git/", "node_modules/", "__pycache__/", ".aegis/",
    ".secrets/", "*.encrypted", ".env", "*.pyc",
]


class HackerBotAgent(ParadigmAgentBase):
    def __init__(self, telemetry=None):
        super().__init__(telemetry=telemetry)
        self.vault = EmeraldCryptoVault()

    # ── Paradigm 1: 5 search channels mapping for nicherisk analysis ──────

    def _classify_finding_niche(self, finding: dict) -> str:
        text = f"{finding.get('type', '')} {finding.get('file', '')} {finding.get('endpoint', '')}"
        tl = text.lower()
        if any(k in tl for k in ("fintech", "bank", "payment", "stripe", "finance")):
            return "fintech"
        if any(k in tl for k in ("saas", "b2b", "api", "endpoint", "host")):
            return "b2b_saas"
        if any(k in tl for k in ("game", "gaming", "udp", "latency")):
            return "gaming"
        if any(k in tl for k in ("health", "dental", "hipaa", "medical")):
            return "health_dental"
        if any(k in tl for k in ("web3", "nft", "blockchain")):
            return "web3_nft"
        return "b2b_saas"

    # ── Audit channels (Paradigm 1) ───────────────────────────────────────

    async def _channel_credential_leaks(self) -> list:
        findings = []
        for ext in ("**/*.py", "**/*.js", "**/*.go", "**/*.sh", "**/*.yml",
                     "**/*.yaml", "**/*.json", "**/*.toml", "**/*.conf",
                     "**/*.cfg", "**/*.ini", "**/*.env.example", "**/Dockerfile*"):
            for f in REPO_PATH.glob(ext):
                try:
                    rel = str(f.relative_to(REPO_PATH))
                    if any(exc.replace("*", "") in rel for exc in LEAK_EXCLUSIONS):
                        continue
                    content = f.read_text(errors="replace")
                    for pattern, desc in SECRET_PATTERNS:
                        matches = re.findall(pattern, content)
                        for m in matches:
                            findings.append({
                                "file": rel, "type": desc,
                                "match": m[:8] + "..." + m[-4:], "line": None,
                            })
                except Exception:
                    continue
        return findings

    async def _channel_api_fuzz(self) -> list:
        endpoints = ["/health", "/api/telemetry", "/api/stream", "/decrypt", "/encrypt", "/status"]
        findings = []
        payloads = [
            {"data": "' OR '1'='1"}, {"data": "../../etc/passwd"},
            {"data": "<script>alert(1)</script>"}, {"data": "${7*7}"},
            {"data": "\x00\x01\x02"},
        ]
        async with aiohttp.ClientSession() as session:
            for ep in endpoints:
                for payload in payloads:
                    try:
                        async with session.post(f"{HF_SPACE_URL}{ep}", json=payload,
                                                 timeout=aiohttp.ClientTimeout(total=5)) as resp:
                            if resp.status not in (400, 404, 405, 500):
                                findings.append({
                                    "endpoint": ep, "payload": str(payload)[:50],
                                    "status": resp.status,
                                    "finding": f"Unexpected response {resp.status} for {ep}",
                                })
                    except (asyncio.TimeoutError, aiohttp.ClientError):
                        pass
                    await asyncio.sleep(0.1)
        return findings

    async def _channel_dependency_audit(self) -> list:
        findings = []
        req_files = list(REPO_PATH.glob("requirements.txt"))
        req_files += list(REPO_PATH.glob("Pipfile"))
        req_files += list(REPO_PATH.glob("pyproject.toml"))
        for req_file in req_files:
            try:
                content = req_file.read_text()
                if "datasets" in content and "2.12.0" in content:
                    findings.append({
                        "file": str(req_file.relative_to(REPO_PATH)),
                        "type": "outdated",
                        "detail": "datasets==2.12.0 may have known vulns, check latest",
                    })
            except Exception:
                continue
        return findings

    async def _channel_hardcoded_keys(self) -> list:
        findings = []
        patterns = [
            ('MASTER_KEY\\s*=\\s*["\']?[A-Fa-f0-9]{64}', "Hardcoded MASTER_KEY"),
            ('MASTER_KEY\\s*=\\s*os\\.getenv\\([^)]*\\)\\s*,\\s*["\'][A-Fa-f0-9]{64}', "Hardcoded MASTER_KEY fallback"),
        ]
        for ext in ("**/*.py", "**/*.js", "**/*.go", "**/Dockerfile*"):
            for f in REPO_PATH.glob(ext):
                try:
                    rel = str(f.relative_to(REPO_PATH))
                    content = f.read_text(errors="replace")
                    for pattern, desc in patterns:
                        if re.search(pattern, content):
                            findings.append({"file": rel, "type": desc, "severity": "critical"})
                except Exception:
                    continue
        return findings

    async def _channel_self_heal(self, findings: list) -> int:
        patched = 0
        for finding in findings:
            if self._apply_reasoning_rules(
                finding.get("file", ""),
                self._classify_finding_niche(finding),
            ) and (finding.get("severity") == "critical" or finding.get("type", "").startswith("Hardcoded")):
                if await self._auto_patch(finding):
                    patched += 1
        return patched

    # ── Auto-patch & push ─────────────────────────────────────────────────

    async def _auto_patch(self, finding: dict) -> bool:
        file_path = REPO_PATH / finding["file"]
        if not file_path.exists():
            return False
        try:
            content = file_path.read_text()
            if finding["type"] == "Hardcoded MASTER_KEY":
                content = content.replace(
                    'MASTER_KEY = os.getenv("MASTER_KEY", "5916b6fd7d8e6a4b52c99497b23868dee12869ef7e6b4b549941d57fce77cbaa")',
                    'MASTER_KEY = os.getenv("EMERALD_MASTER_SECURE_KEY")',
                )
                for line in content.split("\n"):
                    if 'MASTER_KEY = os.getenv("MASTER_KEY"' in line:
                        patched = line.replace(
                            'os.getenv("MASTER_KEY", "5916b6fd7d8e6a4b52c99497b23868dee12869ef7e6b4b549941d57fce77cbaa")',
                            'os.getenv("EMERALD_MASTER_SECURE_KEY")',
                        )
                        content = content.replace(line, patched)
                file_path.write_text(content)
                logging.info(f"Auto-patched hardcoded key in {finding['file']}")
                return True
            if finding["type"] == "Hardcoded MASTER_KEY fallback":
                content = content.replace(
                    '"5916b6fd7d8e6a4b52c99497b23868dee12869ef7e6b4b549941d57fce77cbaa"',
                    '""',
                )
                file_path.write_text(content)
                logging.info(f"Auto-patched hardcoded key fallback in {finding['file']}")
                return True
            if finding["type"] in ("OpenAI/Stripe API key", "GitHub Personal Access Token",
                                    "Hugging Face Token"):
                match_str = finding["match"]
                if "..." in match_str:
                    prefix = match_str[:8]
                    suffix = match_str[-4:]
                    pattern = re.escape(prefix) + "[A-Za-z0-9_-]+" + re.escape(suffix)
                    content = re.sub(pattern, "[REDACTED]", content)
                file_path.write_text(content)
                logging.info(f"Auto-redacted {finding['type']} in {finding['file']}")
                return True
        except Exception as e:
            logging.error(f"Auto-patch failed for {finding['file']}: {e}")
        return False

    async def _commit_and_push(self):
        branch = f"security-patch/{int(time.time())}"
        try:
            subprocess.run(["git", "checkout", "-b", branch],
                           cwd=REPO_PATH, capture_output=True, timeout=10)
        except Exception:
            branch = f"security-patch/main-{int(time.time())}"
            subprocess.run(["git", "checkout", "-b", branch],
                           cwd=REPO_PATH, capture_output=True, timeout=10)
        subprocess.run(["git", "add", "."], cwd=REPO_PATH, capture_output=True, timeout=10)
        subprocess.run(["git", "commit", "-m", f"auto:security-patch {branch}"],
                       cwd=REPO_PATH,
                       env={**os.environ, "GIT_AUTHOR_NAME": "Emerald HackerBot",
                            "GIT_AUTHOR_EMAIL": "hacker-bot@emerald.security",
                            "GIT_COMMITTER_NAME": "Emerald HackerBot",
                            "GIT_COMMITTER_EMAIL": "hacker-bot@emerald.security"},
                       capture_output=True, timeout=10)
        result = subprocess.run(["git", "push", GITHUB_REMOTE, branch],
                                cwd=REPO_PATH, capture_output=True, timeout=30)
        if result.returncode == 0:
            logging.info(f"Patch pushed as branch: {branch}")
            return branch
        return None

    # ── Paradigm 5: Audit cycle ───────────────────────────────────────────

    async def run_security_audit(self) -> dict:
        logging.info("=== Hacker Bot Security Audit ===")
        start = time.time()
        channel_results = await asyncio.gather(
            self._channel_credential_leaks(),
            self._channel_api_fuzz(),
            self._channel_dependency_audit(),
            self._channel_hardcoded_keys(),
        )
        credential_leaks, fuzz_findings, dep_findings, hc_findings = channel_results
        all_findings = credential_leaks + fuzz_findings + dep_findings + hc_findings
        ranked = self._bm25_rank(all_findings, "critical security vulnerability leak hardcoded credential",
                                 key=lambda f: f"{f.get('type', '')} {f.get('file', '')} {f.get('endpoint', '')}")
        self._emit_telemetry("audit_complete", total=len(ranked),
                             credential_leaks=len(credential_leaks),
                             api_fuzz=len(fuzz_findings),
                             deps=len(dep_findings),
                             hardcoded=len(hc_findings))
        logging.info(f"Credential leaks: {len(credential_leaks)}, API fuzz: {len(fuzz_findings)}, "
                     f"Deps: {len(dep_findings)}, Hardcoded: {len(hc_findings)}")
        report = {
            "timestamp": time.time(),
            "duration": round(time.time() - start, 2),
            "total_findings": len(ranked),
            "credential_leaks": len(credential_leaks),
            "api_vulnerabilities": len(fuzz_findings),
            "dependency_issues": len(dep_findings),
            "hardcoded_keys": len(hc_findings),
            "findings": ranked[:50],
        }
        Path("/tmp/hacker_bot_report.json").write_text(json.dumps(report, indent=2))
        if hc_findings:
            patched = await self._channel_self_heal(hc_findings)
            if patched > 0:
                self._emit_telemetry("patches_applied", count=patched)
                branch = await self._commit_and_push()
                if branch:
                    logging.info(f"Patch pushed: {branch}")
        return report

    async def hacker_bot_loop(self):
        logging.info("Hacker Bot activated — 5-paradigm OP_CODE Zero-Trust security monitoring")
        await self.run_security_audit()
        await self._hot_daemon_loop(self.run_security_audit, SCAN_INTERVAL)


async def hacker_bot_loop(telemetry=None):
    agent = HackerBotAgent(telemetry=telemetry)
    await agent.hacker_bot_loop()


async def main():
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [HACKERBOT] %(levelname)s %(message)s")
    await hacker_bot_loop()

if __name__ == "__main__":
    asyncio.run(main())
